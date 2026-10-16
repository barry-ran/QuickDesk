// Minimal TURN client harness for capacity + relay-quality checks against a
// coturn server running in `use-auth-secret` mode (the same mode the signaling
// server issues credentials for).
//
// Capacity run (N concurrent allocations):
//   QD_TURN_URL=turn:127.0.0.1:3478 QD_TURN_SECRET=qd-turn-secret-dev \
//   QD_TURN_CLIENTS=50 node turn-capacity.mjs
//
// Relay-quality run (two allocations actually exchange data through the relay):
//   QD_TURN_URL=... QD_TURN_QUALITY=1 QD_TURN_PACKETS=200 QD_TURN_PAYLOAD=1024 \
//   node turn-capacity.mjs
//
// Soak run (sustained relayed traffic + periodic allocation refresh):
//   QD_TURN_URL=... QD_TURN_SOAK=1 QD_TURN_SOAK_SECONDS=300 QD_TURN_PAYLOAD=1200 \
//   node turn-capacity.mjs
//   # add QD_TURN_SEND_KBPS=1000 and QD_TURN_EXPECT_KBPS_MAX=400 to verify a
//   # coturn `max-bps` cap, or QD_TURN_MAX_LOSS_PERCENT=5 to gate on loss.
//
// TURN over TLS (any mode; the URL scheme picks the transport):
//   QD_TURN_URL=turns:127.0.0.1:5349 QD_TURN_INSECURE=1 QD_TURN_QUALITY=1 \
//   node turn-capacity.mjs
//   # verification is ON by default; QD_TURN_INSECURE only suits a local
//   # self-signed server. QD_TURN_CA=<pem path> (with QD_TURN_URL=turns:localhost:
//   # ...) verifies against a specific certificate instead. QD_TURN_URL also
//   # selects the endpoint when credentials come from the signaling server.
//
// ChannelData path (what a real client switches to after the first packet):
//   QD_TURN_QUALITY=1 QD_TURN_CHANNEL=1 node turn-capacity.mjs
//   # Binds a channel per direction and relays with ChannelData framing
//   # instead of Send/Data indications. In soak mode the bindings are
//   # re-sent on the refresh timer, because a binding only lasts 600s.
//   # Note the receive path has to tell the two framings apart: ChannelData is
//   # not a STUN message, so a parser that assumes STUN drops it silently.
//
// Relay-pool exhaustion and recycle (a small relay pool, filled on purpose):
//   QD_TURN_URL=turn:127.0.0.1:3479 QD_TURN_CLIENTS=8 \
//   QD_TURN_EXPECT_FAILURES=4 QD_TURN_EXPECT_ERROR=508 node turn-capacity.mjs
//   # `QD_TURN_EXPECT_FAILURES` turns "some allocations must be rejected" into
//   # an assertion (with `QD_TURN_EXPECT_ERROR` pinning the TURN error code),
//   # instead of a run that can only ever report failure.
//
//   QD_TURN_URL=turn:127.0.0.1:3479 QD_TURN_POOL_RECYCLE=1 node turn-capacity.mjs
//   # Fills the pool, releases everything, then polls until one allocation
//   # fits again, reporting how long the pool took to recycle.
//
// Credentials come from QD_TURN_SECRET (minted locally) or, if unset, from the
// signaling server's GET /v1/ice-config (QD_SIGNALING_URL + QD_API_KEY).
//
// Exit codes: 0 = assertions held, 2 = assertion failed, 3 = no credentials,
// 4 = could not reach the TURN server.

import dgram from 'node:dgram';
import net from 'node:net';
import tls from 'node:tls';
import fs from 'node:fs';
import crypto from 'node:crypto';

const MAGIC_COOKIE = 0x2112a442;

// Channel numbers are one per peer binding on an allocation; any value in
// 0x4000..0x7FFF works and the two directions are independent.
const CHANNEL_AB = 0x4000;
const CHANNEL_BA = 0x4001;

// Counts allocations that had to be retried because coturn still had the
// previous 5-tuple in a closing state (see allocate()).
let transientRetries = 0;

// STUN/TURN message types we care about.
const T = {
  Allocate: 0x0003,
  Refresh: 0x0004,
  SendInd: 0x0016,
  DataInd: 0x0017,
  AllocateSuccess: 0x0103,
  AllocateError: 0x0113,
  CreatePermission: 0x0008,
  CreatePermissionSuccess: 0x0108,
  CreatePermissionError: 0x0118,
  ChannelBind: 0x0009,
  ChannelBindSuccess: 0x0109,
  RefreshSuccess: 0x0104,
};

const A = {
  Username: 0x0006,
  MessageIntegrity: 0x0008,
  ErrorCode: 0x0009,
  Realm: 0x0014,
  Nonce: 0x0015,
  XorRelayed: 0x0016,
  XorPeer: 0x0012,
  ChannelNumber: 0x000c,
  Data: 0x0013,
  RequestedTransport: 0x0019,
  XorMapped: 0x0020,
  Fingerprint: 0x8028,
  Lifetime: 0x000d,
};

// ---------------------------------------------------------------------------
// Credentials
// ---------------------------------------------------------------------------

function mintCredential(secret, suffix = 'quickdesk', ttlSeconds = 86400) {
  const username = `${Math.floor(Date.now() / 1000) + ttlSeconds}:${suffix}`;
  const credential = crypto
    .createHmac('sha1', secret)
    .update(username)
    .digest('base64');
  return { username, credential };
}

async function credentialFromSignaling() {
  const base = process.env.QD_SIGNALING_URL ?? 'http://127.0.0.1:8000';
  const apiKey = process.env.QD_API_KEY ?? 'qd-local-dev-key';
  const res = await fetch(`${base}/v1/ice-config`, { headers: { 'X-API-Key': apiKey } });
  if (!res.ok) {
    throw new Error(`ice-config failed: HTTP ${res.status}`);
  }
  const body = await res.json();
  const turn = (body.ice_servers ?? []).find((s) => (s.urls ?? []).some((u) => u.startsWith('turn:')));
  if (!turn) {
    throw new Error('ice-config returned no turn: server');
  }
  const url = turn.urls.find((u) => u.startsWith('turn:'));
  return { url, username: turn.username, credential: turn.credential, lifetime: body.lifetime_duration };
}

function turnEndpoint(url) {
  // turn:host:port?transport=udp|tcp  |  turns:host:port?transport=tcp
  const secure = /^turns:/i.test(url);
  const [authority, query = ''] = url.replace(/^turns?:/i, '').split('?');
  const [host, port] = authority.split(':');
  const params = new URLSearchParams(query);
  // `turns:` is TLS over TCP by definition; plain `turn:` defaults to UDP and
  // only uses TCP when the URL asks for it.
  const transport = secure ? 'tls' : (params.get('transport') ?? 'udp').toLowerCase();
  return { host, port: Number(port ?? (secure ? 5349 : 3478)), transport };
}

// ---------------------------------------------------------------------------
// STUN/TURN message encoding
// ---------------------------------------------------------------------------

function attr(type, value) {
  const padding = (4 - (value.length % 4)) % 4;
  const buf = Buffer.alloc(4 + value.length + padding);
  buf.writeUInt16BE(type, 0);
  buf.writeUInt16BE(value.length, 2);
  value.copy(buf, 4);
  return buf;
}

function u32(value) {
  const b = Buffer.alloc(4);
  b.writeUInt32BE(value >>> 0, 0);
  return b;
}

function u16(value) {
  const b = Buffer.alloc(2);
  b.writeUInt16BE(value & 0xffff, 0);
  return b;
}

// ChannelData frames are not STUN messages: RFC 5766 sets the first two bits of
// the first byte to 01, so the channel number lands in 0x4000..0x7FFF. That is
// the path real WebRTC clients use once a binding exists, and it is also why a
// harness that only exercises Send/Data indications can pass while a client
// still fails.
function isChannelDataFrame(buf) {
  return buf.length >= 4 && buf[0] >= 0x40 && buf[0] <= 0x7f;
}

// `padded` is true for TCP/TLS, where each frame is 4-byte aligned; over UDP
// the datagram boundary already delimits the frame.
function channelDataFrameLength(buf, padded) {
  const length = buf.readUInt16BE(2);
  return 4 + length + (padded ? (4 - (length % 4)) % 4 : 0);
}

function parseChannelDataFrame(buf, padded) {
  const channel = buf.readUInt16BE(0);
  const length = buf.readUInt16BE(2);
  return { channelData: true, channel, data: buf.subarray(4, 4 + length) };
}

/** Long-term credential key: MD5(username ":" realm ":" password). */
function longTermKey(username, realm, password) {
  return crypto.createHash('md5').update(`${username}:${realm}:${password}`).digest();
}

function header(type, length, txId) {
  const buf = Buffer.alloc(20);
  buf.writeUInt16BE(type, 0);
  buf.writeUInt16BE(length, 2);
  buf.writeUInt32BE(MAGIC_COOKIE, 4);
  txId.copy(buf, 8);
  return buf;
}

/** Build a message with a trailing MESSAGE-INTEGRITY when key is given. */
function buildMessage(type, txId, attrs, key = null) {
  const attrsBuf = Buffer.concat(attrs);
  if (!key) {
    return Buffer.concat([header(type, attrsBuf.length, txId), attrsBuf]);
  }
  const integrityLen = 24; // 4-byte attr header + 20-byte HMAC-SHA1
  const length = attrsBuf.length + integrityLen;
  const head = header(type, length, txId);
  const mac = crypto.createHmac('sha1', key).update(Buffer.concat([head, attrsBuf])).digest();
  return Buffer.concat([head, attrsBuf, attr(A.MessageIntegrity, mac)]);
}

function parseMessage(buf) {
  const type = buf.readUInt16BE(0);
  const length = buf.readUInt16BE(2);
  const txId = buf.subarray(8, 20);
  const attrs = [];
  let off = 20;
  const end = Math.min(20 + length, buf.length);
  while (off + 4 <= end) {
    const at = buf.readUInt16BE(off);
    const al = buf.readUInt16BE(off + 2);
    const value = buf.subarray(off + 4, off + 4 + al);
    attrs.push({ type: at, value, offset: off });
    off += 4 + al + ((4 - (al % 4)) % 4);
  }
  return { type, length, txId, attrs, raw: buf };
}

function findAttr(msg, type) {
  return msg.attrs.find((a) => a.type === type);
}

function verifyIntegrity(msg, key) {
  const mi = findAttr(msg, A.MessageIntegrity);
  if (!mi) {
    return false;
  }
  const withMi = Buffer.alloc(20);
  msg.raw.copy(withMi, 0, 0, 20);
  withMi.writeUInt16BE(mi.offset - 20 + 24, 2);
  const mac = crypto
    .createHmac('sha1', key)
    .update(Buffer.concat([withMi, msg.raw.subarray(20, mi.offset)]))
    .digest();
  return crypto.timingSafeEqual(mac, mi.value);
}

function parseXorAddress(value) {
  const family = value[1];
  const port = value.readUInt16BE(2) ^ (MAGIC_COOKIE >>> 16);
  if (family === 0x01) {
    const mask = Buffer.alloc(4);
    mask.writeUInt32BE(MAGIC_COOKIE, 0);
    const addr = Buffer.from(value.subarray(4, 8).map((b, i) => b ^ mask[i]));
    return { address: Array.from(addr).join('.'), port, family: 'ipv4' };
  }
  // IPv6: the mask is the magic cookie followed by the transaction id; we only
  // need this for completeness, the harness runs over IPv4.
  return { address: value.subarray(4, 20).toString('hex'), port, family: 'ipv6' };
}

function xorAddressAttr(type, { address, port }) {
  const parts = address.split('.').map(Number);
  const b = Buffer.alloc(8);
  b[0] = 0;
  b[1] = 0x01;
  b.writeUInt16BE(port ^ (MAGIC_COOKIE >>> 16), 2);
  const mask = Buffer.alloc(4);
  mask.writeUInt32BE(MAGIC_COOKIE, 0);
  for (let i = 0; i < 4; i += 1) {
    b[4 + i] = parts[i] ^ mask[i];
  }
  return attr(type, b);
}

// ---------------------------------------------------------------------------
// Transport: one UDP socket, promises matched by transaction id
// ---------------------------------------------------------------------------

class TurnSocket {
  constructor(host, port) {
    this.socket = dgram.createSocket('udp4');
    this.host = host;
    this.port = port;
    this.waiters = new Map();
    this.indications = [];
    this.indicationHandlers = [];
    // UDP datagrams delimit themselves; TCP/TLS frames are 4-byte aligned.
    this.paddedFraming = false;
    this.socket.on('message', (buf) => this.#onMessage(buf));
  }

  #onMessage(buf) {
    // ChannelData arrives as a raw frame, not a STUN message, so it has to be
    // recognised before parseMessage() throws on it.
    if (isChannelDataFrame(buf)) {
      const frame = parseChannelDataFrame(buf, false);
      this.indications.push(frame);
      for (const h of this.indicationHandlers) {
        h(frame);
      }
      return;
    }
    let msg;
    try {
      msg = parseMessage(buf);
    } catch {
      return;
    }
    if (msg.type === T.DataInd) {
      this.indications.push(msg);
      for (const h of this.indicationHandlers) {
        h(msg);
      }
      return;
    }
    const key = msg.txId.toString('hex');
    const waiter = this.waiters.get(key);
    if (waiter) {
      this.waiters.delete(key);
      clearTimeout(waiter.timer);
      waiter.resolve(msg);
    }
  }

  send(buf) {
    return new Promise((resolve, reject) => {
      this.socket.send(buf, this.port, this.host, (err) => (err ? reject(err) : resolve()));
    });
  }

  /** Fire-and-forget send, for sustained traffic where awaiting each packet
   *  would throttle the sender more than the relay does. */
  sendNoWait(buf) {
    this.socket.send(buf, this.port, this.host);
  }

  request(type, attrs, key, timeoutMs) {
    const txId = crypto.randomBytes(12);
    const buf = buildMessage(type, txId, attrs, key);
    const idHex = txId.toString('hex');
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.waiters.delete(idHex);
        reject(new Error(`timeout waiting for reply to 0x${type.toString(16)}`));
      }, timeoutMs);
      this.waiters.set(idHex, { resolve, timer });
      this.send(buf).catch((err) => {
        clearTimeout(timer);
        this.waiters.delete(idHex);
        reject(err);
      });
    });
  }

  close() {
    try {
      this.socket.close();
    } catch {
      /* already closed */
    }
  }
}

/**
 * TURN over TCP or TLS.
 *
 * NOTE: RFC 5389 describes a 2-byte length prefix for STUN over TCP/TLS, but
 * coturn 4.18 does NOT accept that framing - it treats the stream as a sequence
 * of self-delimiting STUN messages, reading the 20-byte header (magic cookie +
 * length) to find each message boundary. Sending the RFC prefix made coturn
 * accept the TLS connection but never parse a message (log: "TLS/TCP socket
 * buffer operation error"); sending unframed messages works and matches what
 * coturn's own turnutils_uclient does.
 */
class TurnStreamSocket {
  constructor({ host, port, secure, tlsOptions }) {
    this.host = host;
    this.port = port;
    this.secure = secure;
    this.tlsOptions = tlsOptions ?? {};
    this.socket = null;
    this.rx = Buffer.alloc(0);
    this.waiters = new Map();
    this.indications = [];
    this.indicationHandlers = [];
    this.paddedFraming = true;
  }

  connect() {
    return new Promise((resolve, reject) => {
      let settled = false;
      const socket = this.secure
        ? tls.connect({
            host: this.host,
            port: this.port,
            servername: this.tlsOptions.servername ?? this.host,
            rejectUnauthorized: this.tlsOptions.rejectUnauthorized ?? true,
            ca: this.tlsOptions.ca,
          })
        : net.connect({ host: this.host, port: this.port });
      this.socket = socket;

      const fail = (err) => {
        for (const waiter of this.waiters.values()) {
          clearTimeout(waiter.timer);
          waiter.reject(err);
        }
        this.waiters.clear();
        if (!settled) {
          settled = true;
          reject(err);
        }
      };
      socket.on('error', fail);
      socket.on('close', () => fail(new Error('stream closed')));
      socket.on('data', (chunk) => this.#onData(chunk));
      socket.once(this.secure ? 'secureConnect' : 'connect', () => {
        if (!settled) {
          settled = true;
          resolve();
        }
      });
    });
  }

  #onData(chunk) {
    this.rx = Buffer.concat([this.rx, chunk]);
    while (this.rx.length >= 4) {
      // A stream carries both ChannelData (4-byte header, 4-byte aligned
      // payload) and STUN messages (20-byte header), so the framing has to be
      // decided per frame.
      if (isChannelDataFrame(this.rx)) {
        const total = channelDataFrameLength(this.rx, true);
        if (this.rx.length < total) {
          break;
        }
        const frame = this.rx.subarray(0, total);
        this.rx = this.rx.subarray(total);
        const parsed = parseChannelDataFrame(frame, true);
        this.indications.push(parsed);
        for (const h of this.indicationHandlers) {
          h(parsed);
        }
        continue;
      }
      if (this.rx.length < 20) {
        break;
      }
      const length = this.rx.readUInt16BE(2);
      const total = 20 + length;
      if (this.rx.length < total) {
        break;
      }
      const msg = this.rx.subarray(0, total);
      this.rx = this.rx.subarray(total);
      let parsed;
      try {
        parsed = parseMessage(msg);
      } catch {
        continue;
      }
      if (parsed.type === T.DataInd) {
        this.indications.push(parsed);
        for (const h of this.indicationHandlers) {
          h(parsed);
        }
        continue;
      }
      const key = parsed.txId.toString('hex');
      const waiter = this.waiters.get(key);
      if (waiter) {
        this.waiters.delete(key);
        clearTimeout(waiter.timer);
        waiter.resolve(parsed);
      }
    }
  }

  send(buf) {
    return new Promise((resolve, reject) => {
      this.socket.write(buf, (err) => (err ? reject(err) : resolve()));
    });
  }

  sendNoWait(buf) {
    this.socket.write(buf);
  }

  request(type, attrs, key, timeoutMs) {
    const txId = crypto.randomBytes(12);
    const buf = buildMessage(type, txId, attrs, key);
    const idHex = txId.toString('hex');
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.waiters.delete(idHex);
        reject(new Error(`timeout waiting for reply to 0x${type.toString(16)}`));
      }, timeoutMs);
      this.waiters.set(idHex, { resolve, timer });
      this.send(buf).catch((err) => {
        clearTimeout(timer);
        this.waiters.delete(idHex);
        reject(err);
      });
    });
  }

  close() {
    try {
      this.socket?.destroy();
    } catch {
      /* already closed */
    }
  }
}

/** Open the transport named by the TURN URL (UDP unless TCP/TLS was asked for). */
async function openTurnSocket({ host, port, transport, tlsOptions }) {
  if (transport === 'tcp' || transport === 'tls') {
    const sock = new TurnStreamSocket({ host, port, secure: transport === 'tls', tlsOptions });
    await sock.connect();
    return sock;
  }
  return new TurnSocket(host, port);
}

function errorCode(msg) {
  const e = findAttr(msg, A.ErrorCode);
  if (!e) {
    return null;
  }
  return { code: e.value[2] * 100 + e.value[3], reason: e.value.subarray(4).toString('utf8') };
}

// ---------------------------------------------------------------------------
// Operations
// ---------------------------------------------------------------------------

async function allocate(params, timeoutMs = 4000, attempt = 1) {
  const { username, password } = params;
  const sock = await openTurnSocket(params);
  const started = process.hrtime.bigint();

  // 1. Unauthenticated Allocate -> expects 401 with REALM + NONCE.
  const challenge = await sock.request(
    T.Allocate,
    [attr(A.RequestedTransport, Buffer.from([17, 0, 0, 0]))],
    null,
    timeoutMs,
  );
  if (challenge.type !== T.AllocateError || errorCode(challenge)?.code !== 401) {
    const err = errorCode(challenge);
    sock.close();
    // coturn keeps a released allocation's 5-tuple in a closing state for about
    // a second. A later run that rebinds the same ephemeral source port hits it
    // and gets 437 "Mismatched allocation: wrong transaction ID" instead of a
    // 401. A real client just retries on a fresh socket, so mirror that here.
    if (err?.code === 437 && attempt < 4) {
      transientRetries += 1;
      await new Promise((r) => setTimeout(r, 40));
      return allocate(params, timeoutMs, attempt + 1);
    }
    throw new Error(
      `expected 401 challenge, got type 0x${challenge.type.toString(16)}${err ? ` (${err.code} ${err.reason})` : ''}`,
    );
  }
  const realm = findAttr(challenge, A.Realm).value.toString('utf8');
  const nonce = findAttr(challenge, A.Nonce).value.toString('utf8');
  const key = longTermKey(username, realm, password);

  // 2. Authenticated Allocate.
  const reply = await sock.request(
    T.Allocate,
    [
      attr(A.RequestedTransport, Buffer.from([17, 0, 0, 0])),
      attr(A.Username, Buffer.from(username, 'utf8')),
      attr(A.Realm, Buffer.from(realm, 'utf8')),
      attr(A.Nonce, Buffer.from(nonce, 'utf8')),
    ],
    key,
    timeoutMs,
  );
  const elapsedMs = Number(process.hrtime.bigint() - started) / 1e6;
  if (reply.type !== T.AllocateSuccess) {
    sock.close();
    const err = errorCode(reply);
    throw new Error(`allocate rejected: ${err ? `${err.code} ${err.reason}` : `type 0x${reply.type.toString(16)}`}`);
  }
  const relayed = parseXorAddress(findAttr(reply, A.XorRelayed).value);
  return {
    socket: sock,
    key,
    realm,
    nonce,
    username,
    relayed,
    mapped: parseXorAddress(findAttr(reply, A.XorMapped).value),
    integrityOK: verifyIntegrity(reply, key),
    elapsedMs,
  };
}

async function createPermission(alloc, peer) {
  const reply = await authedRequest(
    alloc,
    T.CreatePermission,
    [xorAddressAttr(A.XorPeer, peer)],
    T.CreatePermissionSuccess,
  );
  if (reply.type !== T.CreatePermissionSuccess) {
    const err = errorCode(reply);
    throw new Error(`create-permission failed: ${err ? `${err.code} ${err.reason}` : `type 0x${reply.type.toString(16)}`}`);
  }
  return reply;
}

/**
 * Send an authenticated request. coturn may rotate the nonce, so a 438 (Stale
 * Nonce) is answered once with the nonce echoed in the error. Without this a
 * follow-up request silently fails and the allocation is never released.
 */
async function authedRequest(alloc, type, extraAttrs, expectSuccess) {
  const attempt = (nonce) =>
    alloc.socket.request(
      type,
      [
        ...extraAttrs,
        attr(A.Username, Buffer.from(alloc.username, 'utf8')),
        attr(A.Realm, Buffer.from(alloc.realm, 'utf8')),
        attr(A.Nonce, Buffer.from(nonce, 'utf8')),
      ],
      alloc.key,
      4000,
    );

  let reply = await attempt(alloc.nonce);
  if (reply.type !== expectSuccess && errorCode(reply)?.code === 438) {
    const fresh = findAttr(reply, A.Nonce);
    if (fresh) {
      alloc.nonce = fresh.value.toString('utf8');
      reply = await attempt(alloc.nonce);
    }
  }
  return reply;
}

function sendData(alloc, peer, payload) {
  // Send indications are not authenticated.
  const txId = crypto.randomBytes(12);
  const buf = buildMessage(
    T.SendInd,
    txId,
    [xorAddressAttr(A.XorPeer, peer), attr(A.Data, payload)],
    null,
  );
  return alloc.socket.send(buf);
}

function sendDataNoWait(alloc, peer, payload) {
  const txId = crypto.randomBytes(12);
  const buf = buildMessage(
    T.SendInd,
    txId,
    [xorAddressAttr(A.XorPeer, peer), attr(A.Data, payload)],
    null,
  );
  alloc.socket.sendNoWait(buf);
}

/**
 * Bind a peer to a channel number so payload can be relayed with the compact
 * ChannelData framing instead of Send/Data indications. This is the path real
 * clients use after the first packet, so exercising only indications can miss
 * a broken channel path entirely. A binding lives 600s and is refreshed by
 * re-sending ChannelBind.
 */
async function channelBind(alloc, peer, channel) {
  const reply = await authedRequest(
    alloc,
    T.ChannelBind,
    [attr(A.ChannelNumber, u16(channel)), xorAddressAttr(A.XorPeer, peer)],
    T.ChannelBindSuccess,
  );
  if (reply.type !== T.ChannelBindSuccess) {
    const err = errorCode(reply);
    throw new Error(`channel-bind failed: ${err ? `${err.code} ${err.reason}` : `type 0x${reply.type.toString(16)}`}`);
  }
  return reply;
}

function buildChannelData(channel, payload, padded) {
  const padding = padded ? (4 - (payload.length % 4)) % 4 : 0;
  const buf = Buffer.alloc(4 + payload.length + padding);
  buf.writeUInt16BE(channel, 0);
  buf.writeUInt16BE(payload.length, 2);
  payload.copy(buf, 4);
  return buf;
}

function sendChannelData(alloc, channel, payload) {
  return alloc.socket.send(buildChannelData(channel, payload, alloc.socket.paddedFraming));
}

function sendChannelDataNoWait(alloc, channel, payload) {
  alloc.socket.sendNoWait(buildChannelData(channel, payload, alloc.socket.paddedFraming));
}

/**
 * The relayed bytes, whichever framing delivered them. Data indications carry
 * them in a DATA attribute; ChannelData frames carry them bare after a 4-byte
 * header.
 */
function relayedPayload(msg) {
  if (msg.channelData) {
    return msg.data;
  }
  return findAttr(msg, A.Data)?.value ?? null;
}

/**
 * Refresh an allocation's lifetime. coturn grants a finite lifetime (600s by
 * default) and drops the allocation when it expires, so a long session must
 * refresh; a soak that never refreshes only proves the first minute works.
 */
async function refreshAllocation(alloc, lifetimeSeconds) {
  const reply = await authedRequest(
    alloc,
    T.Refresh,
    [attr(A.Lifetime, u32(lifetimeSeconds))],
    T.RefreshSuccess,
  );
  return reply.type === T.RefreshSuccess;
}

/**
 * Release an allocation (Refresh with LIFETIME=0). Without this coturn keeps
 * the allocation for its full lifetime, which both inflates the global count
 * and - more subtly - makes a later run that reuses the same ephemeral source
 * port collide with the stale allocation and get `437 Mismatched allocation`.
 */
async function deallocate(alloc) {
  try {
    const reply = await authedRequest(alloc, T.Refresh, [attr(A.Lifetime, u32(0))], T.RefreshSuccess);
    if (reply.type !== T.RefreshSuccess) {
      return false;
    }
    return true;
  } catch {
    return false;
  } finally {
    alloc.socket.close();
  }
}

function percentile(sorted, p) {
  if (sorted.length === 0) {
    return 0;
  }
  const idx = Math.min(sorted.length - 1, Math.max(0, Math.ceil((p / 100) * sorted.length) - 1));
  return sorted[idx];
}

// ---------------------------------------------------------------------------
// Runs
// ---------------------------------------------------------------------------

async function runCapacity(conn, credFor) {
  const clients = Number(process.env.QD_TURN_CLIENTS ?? 50);
  const expectAllocated = process.env.QD_TURN_EXPECT_ALLOCATED
    ? Number(process.env.QD_TURN_EXPECT_ALLOCATED)
    : null;
  // An exhaustion run asks for failures on purpose, so it needs to assert the
  // *shape* of the failure (how many, and with what error) instead of only
  // printing it. Without these two the run can only ever report FAIL, which is
  // useless as a regression gate.
  const expectFailures = process.env.QD_TURN_EXPECT_FAILURES
    ? Number(process.env.QD_TURN_EXPECT_FAILURES)
    : null;
  const expectError = process.env.QD_TURN_EXPECT_ERROR ?? null;

  process.stdout.write(`opening ${clients} concurrent TURN allocations at ${conn.host}:${conn.port}\n`);
  const started = Date.now();
  const results = await Promise.allSettled(
    // A real deployment mints one credential per device; issuing a distinct
    // username per client keeps coturn's per-user quota out of the measurement.
    Array.from({ length: clients }, (_, i) => allocate({ ...conn, ...credFor(i) })),
  );
  const wallMs = Date.now() - started;

  const ok = [];
  const failures = new Map();
  const latencies = [];
  let integrityFailures = 0;
  for (const r of results) {
    if (r.status === 'fulfilled') {
      ok.push(r.value);
      latencies.push(r.value.elapsedMs);
      if (!r.value.integrityOK) {
        integrityFailures += 1;
      }
    } else {
      const reason = String(r.reason?.message ?? r.reason);
      failures.set(reason, (failures.get(reason) ?? 0) + 1);
    }
  }
  latencies.sort((a, b) => a - b);

  const relayed = [...new Set(ok.map((a) => `${a.relayed.address}:${a.relayed.port}`))];
  const released = (await Promise.all(ok.map(deallocate))).filter(Boolean).length;
  console.log(`\n=== TURN capacity ===`);
  console.log(`allocated:   ${ok.length}/${clients}`);
  console.log(`failed:      ${clients - ok.length}`);
  console.log(`wall time:   ${wallMs} ms`);
  console.log(`alloc latency: p50 ${percentile(latencies, 50).toFixed(1)} ms | p95 ${percentile(latencies, 95).toFixed(1)} ms | max ${percentile(latencies, 100).toFixed(1)} ms`);
  console.log(`distinct relayed addresses: ${relayed.length}`);
  console.log(`MESSAGE-INTEGRITY verified on replies: ${ok.length - integrityFailures}/${ok.length}`);
  console.log(`allocations released: ${released}/${ok.length}`);
  console.log(`transient 437 retries: ${transientRetries}`);
  if (failures.size > 0) {
    console.log('failure reasons:');
    for (const [reason, count] of failures) {
      console.log(`  ${count}x ${reason}`);
    }
  }

  if (integrityFailures > 0) {
    console.error(`FAIL: ${integrityFailures} allocation replies failed MESSAGE-INTEGRITY`);
    return 2;
  }
  if (released !== ok.length) {
    console.error(`FAIL: ${ok.length - released} allocations were not released; later runs will collide`);
    return 2;
  }
  if (expectFailures !== null) {
    if (clients - ok.length !== expectFailures) {
      console.error(`FAIL: expected ${expectFailures} rejections, got ${clients - ok.length}`);
      return 2;
    }
    if (expectError) {
      const unmatched = [...failures.entries()].filter(([reason]) => !reason.includes(expectError));
      if (unmatched.length > 0) {
        console.error(`FAIL: expected every rejection to mention "${expectError}", but saw:`);
        for (const [reason, count] of unmatched) {
          console.error(`  ${count}x ${reason}`);
        }
        return 2;
      }
    }
    console.log(`PASS (${expectFailures} expected rejection(s))`);
    return 0;
  }
  if (expectAllocated !== null && ok.length !== expectAllocated) {
    console.error(`FAIL: expected ${expectAllocated} allocations, got ${ok.length}`);
    return 2;
  }
  if (ok.length !== clients) {
    console.error(`FAIL: ${clients - ok.length} allocations did not succeed`);
    return 2;
  }
  console.log('PASS');
  return 0;
}

async function runQuality(conn, credFor) {
  const packets = Number(process.env.QD_TURN_PACKETS ?? 200);
  const payloadSize = Number(process.env.QD_TURN_PAYLOAD ?? 1024);
  // Indications are the default; QD_TURN_CHANNEL=1 exercises the ChannelData
  // path instead, which is what a real client switches to after the first
  // packet.
  const useChannel = process.env.QD_TURN_CHANNEL === '1';

  console.log(
    `relay-quality run: ${packets} packets x ${payloadSize} B between two allocations ` +
      `(path: ${useChannel ? 'ChannelData' : 'Send/Data indications'})`,
  );
  const a = await allocate({ ...conn, ...credFor(0) });
  const b = await allocate({ ...conn, ...credFor(1) });
  console.log(`A relayed ${a.relayed.address}:${a.relayed.port}  B relayed ${b.relayed.address}:${b.relayed.port}`);

  // Each side must permit the other's relayed address.
  await createPermission(a, b.relayed);
  await createPermission(b, a.relayed);
  if (useChannel) {
    await channelBind(a, b.relayed, CHANNEL_AB);
    await channelBind(b, a.relayed, CHANNEL_BA);
  }

  let received = 0;
  let bytes = 0;
  b.socket.indicationHandlers.push((msg) => {
    const data = relayedPayload(msg);
    if (data) {
      received += 1;
      bytes += data.length;
    }
  });

  // Prime: some stacks drop the first datagram before the permission settles.
  if (useChannel) {
    await sendChannelData(a, CHANNEL_AB, Buffer.from('prime'));
  } else {
    await sendData(a, b.relayed, Buffer.from('prime'));
  }
  await new Promise((r) => setTimeout(r, 50));

  b.socket.indications.length = 0;
  received = 0;
  bytes = 0;

  const started = Date.now();
  for (let i = 0; i < packets; i += 1) {
    const payload = Buffer.alloc(payloadSize);
    payload.writeUInt32BE(i, 0);
    if (useChannel) {
      await sendChannelData(a, CHANNEL_AB, payload);
    } else {
      await sendData(a, b.relayed, payload);
    }
    if (i % 20 === 0) {
      await new Promise((r) => setTimeout(r, 2));
    }
  }
  // Give the relay a moment to drain.
  await new Promise((r) => setTimeout(r, 300));
  const elapsedMs = Date.now() - started;

  const lossPercent = ((packets - received) / packets) * 100;
  const kbps = (bytes * 8) / elapsedMs;

  console.log(`\n=== TURN relay quality ===`);
  console.log(`sent:      ${packets} packets, ${packets * payloadSize} B`);
  console.log(`received:  ${received} packets, ${bytes} B`);
  console.log(`loss:      ${lossPercent.toFixed(1)} %`);
  console.log(`goodput:   ~${kbps.toFixed(0)} kbps over ${elapsedMs} ms`);
  console.log(`A integrity: ${a.integrityOK}  B integrity: ${b.integrityOK}`);

  await Promise.all([deallocate(a), deallocate(b)]);

  const minReceived = Number(process.env.QD_TURN_EXPECT_RECEIVED ?? Math.floor(packets * 0.9));
  if (received < minReceived) {
    console.error(`FAIL: relayed ${received}/${packets}, below the ${minReceived} floor`);
    return 2;
  }
  console.log('PASS');
  return 0;
}

/**
 * Sustained relay run. Holds two allocations open for a configured duration,
 * keeps relaying datagrams through them, and refreshes the allocations on a
 * timer. This is the mode that shows whether a long session stays up (the
 * allocation would be dropped at its lifetime otherwise) and what sustained
 * goodput the relay actually delivers - which is also how a coturn `max-bps`
 * cap is verified.
 */
// Relay-pool recycle run.
//
// A drained pool is not an instant-recovery situation: releasing an allocation
// returns its relay port only after coturn finishes tearing the socket down,
// so a burst that exhausts a small pool leaves the next burst rejected for a
// while even though every client did release cleanly. The number that matters
// for sizing is how long "a while" is, and that is what this mode measures:
// fill the pool, release everything, then poll until one allocation fits.
//
// The measurement only means something against an idle server, so the fill
// phase waits QD_TURN_SETTLE_MS (default 1500 ms) first. That default is not
// arbitrary: a probe against a just-drained pool is rejected instantly with
// 508, and a client that does not retry sees a hard failure rather than a
// short delay.
async function runPoolRecycle(conn, credFor) {
  const fill = Number(process.env.QD_TURN_CLIENTS ?? 8);
  const pollMs = Number(process.env.QD_TURN_POLL_MS ?? 500);
  const maxWaitMs = Number(process.env.QD_TURN_MAX_WAIT_MS ?? 120000);
  const settleMs = Number(process.env.QD_TURN_SETTLE_MS ?? 1500);

  if (settleMs > 0) {
    await new Promise((r) => setTimeout(r, settleMs));
  }

  process.stdout.write(`filling the relay pool with ${fill} concurrent allocations\n`);
  const opened = [];
  const rejections = [];
  const filled = await Promise.allSettled(
    Array.from({ length: fill }, (_, i) => allocate({ ...conn, ...credFor(i) })),
  );
  for (const r of filled) {
    if (r.status === 'fulfilled') {
      opened.push(r.value);
    } else {
      rejections.push(String(r.reason?.message ?? r.reason));
    }
  }
  console.log(`pool fill:   ${opened.length}/${fill} allocated, ${rejections.length} rejected`);
  for (const reason of [...new Set(rejections)]) {
    console.log(`  rejection: ${reason}`);
  }
  if (opened.length === 0) {
    console.error('FAIL: nothing allocated, so there was no pool to drain');
    return 2;
  }

  const released = (await Promise.all(opened.map(deallocate))).filter(Boolean).length;
  console.log(`released:    ${released}/${opened.length}`);
  if (released !== opened.length) {
    console.error(`FAIL: ${opened.length - released} allocations were not released`);
    return 2;
  }

  const started = Date.now();
  let attempts = 0;
  for (;;) {
    const elapsed = Date.now() - started;
    if (elapsed > maxWaitMs) {
      console.error(`FAIL: pool still drained ${elapsed} ms after every allocation was released`);
      return 2;
    }
    attempts += 1;
    try {
      const probe = await allocate({ ...conn, ...credFor(0) }, 2000);
      const probeReleased = await deallocate(probe);
      console.log(`recycled:    ~${Date.now() - started} ms after release (${attempts} probe(s))`);
      if (!probeReleased) {
        console.error('FAIL: the probe allocation could not be released');
        return 2;
      }
      console.log('PASS');
      return 0;
    } catch {
      await new Promise((r) => setTimeout(r, pollMs));
    }
  }
}

async function runSoak(conn, credFor) {
  const seconds = Number(process.env.QD_TURN_SOAK_SECONDS ?? 120);
  const payloadSize = Number(process.env.QD_TURN_PAYLOAD ?? 1200);
  const refreshEverySec = Number(process.env.QD_TURN_SOAK_REFRESH_SECONDS ?? 30);
  const reportEverySec = Number(process.env.QD_TURN_SOAK_REPORT_SECONDS ?? 15);
  const useChannel = process.env.QD_TURN_CHANNEL === '1';
  // 0 = send as fast as the event loop allows. Set it (above the expected cap)
  // to pace the sender, which is how a bandwidth cap is measured cleanly.
  const sendKbps = Number(process.env.QD_TURN_SEND_KBPS ?? 0);

  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
  const sendPace = sendKbps > 0 ? (sendKbps * 1000) / (payloadSize * 8) : 0; // packets/s
  const tickMs = 10;
  const perTick = sendPace > 0 ? Math.max(1, Math.round((sendPace * tickMs) / 1000)) : 25;

  console.log(
    `soak: ${payloadSize} B datagrams for ${seconds}s, refresh every ${refreshEverySec}s, ` +
      `send rate ${sendKbps > 0 ? `${sendKbps} kbps` : 'unthrottled'}, ` +
      `path ${useChannel ? 'ChannelData' : 'Send/Data indications'}`,
  );
  const a = await allocate({ ...conn, ...credFor(0) });
  const b = await allocate({ ...conn, ...credFor(1) });
  console.log(`A relayed ${a.relayed.address}:${a.relayed.port}  B relayed ${b.relayed.address}:${b.relayed.port}`);

  await createPermission(a, b.relayed);
  await createPermission(b, a.relayed);
  if (useChannel) {
    await channelBind(a, b.relayed, CHANNEL_AB);
    await channelBind(b, a.relayed, CHANNEL_BA);
  }

  let sent = 0;
  let received = 0;
  let bytes = 0;
  b.socket.indicationHandlers.push((msg) => {
    const data = relayedPayload(msg);
    if (data) {
      received += 1;
      bytes += data.length;
    }
  });

  // Warm the permission/relay path before measuring.
  if (useChannel) {
    sendChannelDataNoWait(a, CHANNEL_AB, Buffer.from('prime'));
  } else {
    sendDataNoWait(a, b.relayed, Buffer.from('prime'));
  }
  await sleep(100);
  b.socket.indications.length = 0;
  sent = 0;
  received = 0;
  bytes = 0;

  let refreshesOK = 0;
  let refreshesFailed = 0;
  let permsOK = 0;
  let permsFailed = 0;
  let bindsOK = 0;
  let bindsFailed = 0;
  let lastError = '';
  const started = Date.now();
  let lastReport = started;
  let lastSent = 0;
  let lastBytes = 0;
  let lastReceived = 0;
  let nextRefresh = started + refreshEverySec * 1000;
  const payload = Buffer.alloc(payloadSize);

  while (Date.now() - started < seconds * 1000) {
    for (let i = 0; i < perTick; i += 1) {
      payload.writeUInt32BE(sent & 0xffffffff, 0);
      if (useChannel) {
        sendChannelDataNoWait(a, CHANNEL_AB, payload);
      } else {
        sendDataNoWait(a, b.relayed, payload);
      }
      sent += 1;
    }
    await sleep(tickMs);

    const now = Date.now();
    if (now >= nextRefresh) {
      nextRefresh = now + refreshEverySec * 1000;
      try {
        const [okA, okB] = await Promise.all([refreshAllocation(a, 600), refreshAllocation(b, 600)]);
        if (okA && okB) {
          refreshesOK += 1;
        } else {
          refreshesFailed += 1;
          lastError = 'refresh rejected';
        }
      } catch (err) {
        refreshesFailed += 1;
        lastError = String(err.message ?? err);
      }
      // A TURN permission lives for 300s by default and refreshing the
      // allocation does NOT extend it: coturn keeps both allocations alive and
      // silently drops every datagram once the permission lapses. The first
      // hour-scale run died at exactly t+300s with 100% loss for this reason,
      // which is why permissions are now re-issued on the same timer.
      try {
        await createPermission(a, b.relayed);
        await createPermission(b, a.relayed);
        permsOK += 1;
      } catch (err) {
        permsFailed += 1;
        lastError = `permission: ${String(err.message ?? err)}`;
      }
      // A channel binding lasts 600s and, like a permission, is only extended
      // by re-sending ChannelBind. Refresh it on the same timer when the
      // ChannelData path is in use.
      if (useChannel) {
        try {
          await channelBind(a, b.relayed, CHANNEL_AB);
          await channelBind(b, a.relayed, CHANNEL_BA);
          bindsOK += 1;
        } catch (err) {
          bindsFailed += 1;
          lastError = `channel-bind: ${String(err.message ?? err)}`;
        }
      }
    }

    if (now - lastReport >= reportEverySec * 1000) {
      const dtSec = (now - lastReport) / 1000;
      const instKbps = ((bytes - lastBytes) * 8) / dtSec / 1000;
      const windowSent = sent - lastSent;
      const windowLoss = windowSent > 0 ? ((windowSent - (received - lastReceived)) / windowSent) * 100 : 0;
      console.log(
        `  t+${Math.round((now - started) / 1000)}s  relayed ${instKbps.toFixed(0)} kbps  ` +
          `loss ${windowLoss.toFixed(1)}%  refreshes ok/fail ${refreshesOK}/${refreshesFailed}  ` +
          `permissions ok/fail ${permsOK}/${permsFailed}` +
          (useChannel ? `  channel-binds ok/fail ${bindsOK}/${bindsFailed}` : ''),
      );
      lastReport = now;
      lastSent = sent;
      lastBytes = bytes;
      lastReceived = received;
    }
  }

  const elapsedMs = Date.now() - started;
  // Let the relay drain whatever is still in flight before the final count.
  await sleep(500);
  // bits per millisecond is numerically kbps, so no extra /1000 here.
  const sustainedKbps = (bytes * 8) / elapsedMs;
  const lossPercent = sent > 0 ? ((sent - received) / sent) * 100 : 0;

  console.log(`\n=== TURN soak (${(elapsedMs / 1000).toFixed(1)}s) ===`);
  console.log(`sent:      ${sent} packets, ${sent * payloadSize} B`);
  console.log(`received:  ${received} packets, ${bytes} B`);
  console.log(`loss:      ${lossPercent.toFixed(1)} %`);
  console.log(`sustained goodput: ~${sustainedKbps.toFixed(0)} kbps`);
  console.log(`allocations refreshed: ok ${refreshesOK}, failed ${refreshesFailed}${lastError ? ` (${lastError})` : ''}`);
  console.log(`permissions re-issued: ok ${permsOK}, failed ${permsFailed}`);
  if (useChannel) {
    console.log(`channel bindings refreshed: ok ${bindsOK}, failed ${bindsFailed}`);
  }
  console.log(`A integrity: ${a.integrityOK}  B integrity: ${b.integrityOK}`);

  await Promise.all([deallocate(a), deallocate(b)]);

  if (received === 0) {
    console.error('FAIL: nothing came back through the relay');
    return 2;
  }
  if (refreshesFailed > 0) {
    console.error(`FAIL: ${refreshesFailed} refresh cycle(s) failed`);
    return 2;
  }
  if (permsFailed > 0) {
    console.error(`FAIL: ${permsFailed} permission refresh cycle(s) failed`);
    return 2;
  }
  if (bindsFailed > 0) {
    console.error(`FAIL: ${bindsFailed} channel-bind refresh cycle(s) failed`);
    return 2;
  }

  const maxLoss = process.env.QD_TURN_MAX_LOSS_PERCENT ? Number(process.env.QD_TURN_MAX_LOSS_PERCENT) : null;
  if (maxLoss !== null && lossPercent > maxLoss) {
    console.error(`FAIL: loss ${lossPercent.toFixed(1)}% exceeded ${maxLoss}%`);
    return 2;
  }
  if (process.env.QD_TURN_EXPECT_KBPS_MAX && sustainedKbps > Number(process.env.QD_TURN_EXPECT_KBPS_MAX)) {
    console.error(`FAIL: sustained ${sustainedKbps.toFixed(0)} kbps exceeded the ${process.env.QD_TURN_EXPECT_KBPS_MAX} kbps cap`);
    return 2;
  }
  if (process.env.QD_TURN_EXPECT_KBPS_MIN && sustainedKbps < Number(process.env.QD_TURN_EXPECT_KBPS_MIN)) {
    console.error(`FAIL: sustained ${sustainedKbps.toFixed(0)} kbps below the ${process.env.QD_TURN_EXPECT_KBPS_MIN} kbps floor`);
    return 2;
  }
  console.log('PASS');
  return 0;
}

async function main() {
  const secret = process.env.QD_TURN_SECRET;
  let creds;
  if (secret) {
    creds = {
      url: process.env.QD_TURN_URL ?? 'turn:127.0.0.1:3478',
      ...mintCredential(secret, 'quality'),
    };
  } else {
    try {
      creds = await credentialFromSignaling();
    } catch (err) {
      console.error(`no credentials: ${err.message}`);
      return 3;
    }
  }
  // QD_TURN_URL selects the endpoint even when the credentials came from the
  // signaling server, so a `turns:` deployment can be tested with real
  // server-issued credentials instead of a locally minted pair.
  const endpointUrl = process.env.QD_TURN_URL ?? creds.url;
  const { host, port, transport } = turnEndpoint(endpointUrl);
  const credFor = (i) => {
    if (!secret) {
      return { username: creds.username, password: creds.credential };
    }
    const c = mintCredential(secret, `client-${i}`);
    return { username: c.username, password: c.credential };
  };
  // A production `turns:` endpoint should present a certificate the client
  // trusts; QD_TURN_INSECURE only exists for the local self-signed test server
  // and QD_TURN_CA / QD_TURN_SERVERNAME cover the trusted-cert cases.
  let ca;
  if (process.env.QD_TURN_CA) {
    try {
      // tls.connect wants PEM content, not a path.
      ca = fs.readFileSync(process.env.QD_TURN_CA);
    } catch (err) {
      console.error(`cannot read QD_TURN_CA ${process.env.QD_TURN_CA}: ${err.message}`);
      return 3;
    }
  }
  const tlsOptions = {
    rejectUnauthorized: !process.env.QD_TURN_INSECURE,
    ca,
    servername: process.env.QD_TURN_SERVERNAME,
  };
  const opts = { host, port, transport, tlsOptions };
  console.log(
    `transport: ${transport} -> ${host}:${port}` +
      (transport === 'tls' && !tlsOptions.rejectUnauthorized ? ' (certificate verification disabled)' : ''),
  );
  console.log(`credentials: user=${creds.username} (secret ${secret ? 'minted locally' : 'from signaling'})`);

  try {
    if (process.env.QD_TURN_SOAK) {
      return await runSoak(opts, credFor);
    }
    if (process.env.QD_TURN_POOL_RECYCLE) {
      return await runPoolRecycle(opts, credFor);
    }
    return process.env.QD_TURN_QUALITY
      ? await runQuality(opts, credFor)
      : await runCapacity(opts, credFor);
  } catch (err) {
    console.error(`FAIL: ${err.message}`);
    return 4;
  }
}

process.exit(await main());
