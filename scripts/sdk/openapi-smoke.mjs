#!/usr/bin/env node
import net from 'node:net';
// Conformance smoke for SignalingServer/api/openapi.yaml.
//
// The spec is the contract business systems generate clients from, so it has to
// be checked against a running server rather than trusted: this script calls
// the documented operations and asserts the fields the spec promises exist with
// the promised types. If the server renames a field or changes its shape, this
// fails instead of quietly invalidating the spec.
//
// Read-only by default. Give QD_SMOKE_DEVICE=<device_id> and it also covers the
// documented operations with side effects - the offline-alert and access-policy
// PUTs (restored to exactly the prior state afterwards), issuing then revoking a
// short-lived guest code, the wrong-access-code rejection (which writes an
// access-log row and may notify the owner), the whole recording lifecycle (which
// the smoke creates and deletes itself) and the approval flow (which needs the
// device to be online and drives both triggers, the pending-queue cap, the
// denial cooldown and an optional host-offline claim by switching policy and
// presence temporarily).
//
// Usage:
//   QD_IDENTIFIER=user QD_PASSWORD=secret scripts/sdk/openapi-smoke.mjs
//   QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=126735214 scripts/sdk/openapi-smoke.mjs
//
// Env:
//   QD_SERVER      default http://127.0.0.1:8000
//   QD_API_KEY     default qd-local-dev-key
//   QD_IDENTIFIER / QD_PASSWORD   required
//   QD_SMOKE_DEVICE               optional; enables the write-path checks
//   QD_SMOKE_APPROVAL_DENY        set to 1 to also drive the deny outcome; a
//                                 deny arms the per-(device, source) cooldown,
//                                 so a second run against the same device would
//                                 be refused while it lasts
//   QD_SMOKE_HOST_OFFLINE         set to 1 to delete the seeded device's Redis
//                                 heartbeat mid-approval and assert the claim
//                                 is refused with 409 HOST_OFFLINE
//   QD_REDIS_HOST / QD_REDIS_PORT used only by the host-offline check
//
// Exit codes: 0 = the documented shapes hold, 3 = a documented field is missing
// or mistyped, 4 = the server could not be reached or returned an unexpected
// status.

const server = (process.env.QD_SERVER || 'http://127.0.0.1:8000').replace(/\/$/, '');
const apiKey = process.env.QD_API_KEY || 'qd-local-dev-key';
const identifier = process.env.QD_IDENTIFIER;
const password = process.env.QD_PASSWORD;
const smokeDevice = process.env.QD_SMOKE_DEVICE || '';

if (!identifier || !password) {
  console.error('QD_IDENTIFIER and QD_PASSWORD are required');
  process.exit(2);
}

let checks = 0;
const failures = [];

function check(condition, message) {
  checks += 1;
  if (!condition) failures.push(message);
}

// field asserts that a documented property exists and, when a type is given,
// has that JSON type. `null` is accepted for the nullable fields the spec
// declares as ["string","null"].
function field(obj, name, type, where) {
  const present = obj !== null && typeof obj === 'object' && name in obj;
  check(present, `${where}: missing documented field "${name}"`);
  if (!present) return undefined;
  const value = obj[name];
  if (type && value !== null && typeof value !== type) {
    check(false, `${where}: field "${name}" is ${typeof value}, spec says ${type}`);
  }
  return value;
}

async function call(method, path, { token, body, rawBody, contentType, expect, extraHeaders = {} } = {}) {
  const headers = { 'X-API-Key': apiKey, ...extraHeaders };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  // rawBody is for the two shapes JSON cannot express: a FormData multipart
  // upload (fetch sets its own boundary) and a streamed octet-stream body.
  let payload;
  if (rawBody !== undefined) {
    payload = rawBody;
    if (contentType) headers['Content-Type'] = contentType;
  } else if (body !== undefined) {
    payload = JSON.stringify(body);
  }
  let res;
  try {
    res = await fetch(server + path, {
      method,
      headers,
      body: payload,
    });
  } catch (err) {
    console.error(`transport error calling ${method} ${path}: ${err.message}`);
    process.exit(4);
  }
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    // Non-JSON bodies only matter if the caller expected JSON.
  }
  if (expect !== undefined && res.status !== expect) {
    console.error(`${method} ${path}: expected ${expect}, got ${res.status}`);
    console.error(text.slice(0, 400));
    process.exit(4);
  }
  return { status: res.status, json, text, headers: res.headers };
}

function problemShape(json, where, expectedCode) {
  field(json, 'type', 'string', where);
  field(json, 'title', 'string', where);
  field(json, 'status', 'number', where);
  const code = field(json, 'code', 'string', where);
  if (expectedCode) {
    check(code === expectedCode, `${where}: code is "${code}", spec documents "${expectedCode}"`);
  }
}

// The resumable upload reports how much it has staged in a response header, so
// every step of it asserts both the header's presence and its value.
function headerNumber(res, name, where) {
  const raw = res.headers.get(name);
  check(raw !== null && raw !== '', `${where}: ${name} response header is missing`);
  const value = Number(raw);
  check(Number.isInteger(value) && value >= 0, `${where}: ${name} is "${raw}", want a byte count`);
  return Number.isFinite(value) ? value : -1;
}

// A tiny RESP client for the one destructive Redis operation the smoke needs:
// removing the seeded device's heartbeat so the server observes it as offline.
// Keeping this dependency-free avoids pulling a Redis package into the smoke.
function redisCommand(...args) {
  const host = process.env.QD_REDIS_HOST || 'localhost';
  const port = Number(process.env.QD_REDIS_PORT || 6379);
  return new Promise((resolve, reject) => {
    const socket = net.createConnection({ host, port });
    let buffer = '';
    let settled = false;
    const finish = (err, value) => {
      if (settled) return;
      settled = true;
      socket.destroy();
      if (err) reject(err);
      else resolve(value);
    };
    socket.setTimeout(2000);
    socket.on('connect', () => {
      const parts = [`*${args.length}\r\n`];
      for (const arg of args) {
        const value = String(arg);
        parts.push(`$${Buffer.byteLength(value)}\r\n${value}\r\n`);
      }
      socket.write(parts.join(''));
    });
    socket.on('data', (chunk) => {
      buffer += chunk.toString('utf8');
      const end = buffer.indexOf('\r\n');
      if (end === -1) return;
      const line = buffer.slice(0, end);
      if (line.startsWith(':')) finish(null, Number(line.slice(1)));
      else if (line.startsWith('+')) finish(null, line.slice(1));
      else if (line.startsWith('-')) finish(new Error(line.slice(1)));
      else finish(new Error(`unexpected Redis reply: ${line}`));
    });
    socket.on('error', (err) => finish(err));
    socket.on('timeout', () => finish(new Error('Redis command timed out')));
  });
}

// 1. POST /v1/auth/sessions
const login = await call('POST', '/v1/auth/sessions', {
  body: { identifier, password },
  expect: 200,
});
const token = field(login.json, 'access_token', 'string', 'POST /v1/auth/sessions');
field(login.json, 'access_expires_at', 'string', 'POST /v1/auth/sessions');
field(login.json, 'refresh_token', 'string', 'POST /v1/auth/sessions');
field(login.json, 'refresh_expires_at', 'string', 'POST /v1/auth/sessions');
const loginUser = field(login.json, 'user', 'object', 'POST /v1/auth/sessions');
for (const name of ['id', 'username', 'status']) {
  field(loginUser, name, name === 'status' ? 'boolean' : name === 'id' ? 'number' : 'string', 'AuthSessionResponse.user');
}

// 2. GET /v1/me
const me = await call('GET', '/v1/me', { token, expect: 200 });
for (const name of ['id', 'username', 'status', 'device_count', 'channel_type', 'created_at']) {
  field(me.json, name, undefined, 'GET /v1/me');
}

// 3. GET /v1/me/devices
const devices = await call('GET', '/v1/me/devices', { token, expect: 200 });
const deviceItems = field(devices.json, 'items', 'object', 'GET /v1/me/devices');
check(Array.isArray(deviceItems), 'GET /v1/me/devices: items is not an array');
if (Array.isArray(deviceItems) && deviceItems.length > 0) {
  const device = deviceItems[0];
  for (const name of ['device_id', 'device_name', 'remark', 'online', 'logged_in', 'access_code']) {
    field(device, name, undefined, 'GET /v1/me/devices items[]');
  }
  check(typeof device.online === 'boolean', 'Device.online is not a boolean');
  check(typeof device.logged_in === 'boolean', 'Device.logged_in is not a boolean');
}

const deviceId = smokeDevice || (Array.isArray(deviceItems) && deviceItems[0]?.device_id) || '';

// 4. GET /v1/me/devices/{device_id}/offline-alert
const alertBefore = await call('GET', `/v1/me/devices/${deviceId}/offline-alert`, { token, expect: 200 });
for (const name of ['device_id', 'enabled', 'after_minutes', 'repeat_minutes', 'maintenance_windows', 'in_maintenance_window']) {
  field(alertBefore.json, name, undefined, 'GET offline-alert');
}
check(Array.isArray(alertBefore.json.maintenance_windows), 'offline-alert: maintenance_windows is not an array');
check(typeof alertBefore.json.in_maintenance_window === 'boolean', 'offline-alert: in_maintenance_window is not a boolean');

// 5. GET /v1/me/notification-channels
const channels = await call('GET', '/v1/me/notification-channels', { token, expect: 200 });
const channelItems = field(channels.json, 'items', 'object', 'GET notification-channels');
check(Array.isArray(channelItems), 'GET notification-channels: items is not an array');
field(channels.json, 'supported_events', 'object', 'GET notification-channels');
field(channels.json, 'signature_header', 'string', 'GET notification-channels');
field(channels.json, 'signature_scheme', 'string', 'GET notification-channels');
if (Array.isArray(channelItems) && channelItems.length > 0) {
  const ch = channelItems[0];
  for (const name of ['id', 'user_id', 'name', 'url', 'events', 'enabled', 'last_status']) {
    field(ch, name, undefined, 'NotificationChannel');
  }
  // The spec documents `events` as a JSON string, not an array; if the server
  // ever starts sending an array this has to be an explicit spec change.
  check(typeof ch.events === 'string', 'NotificationChannel.events is not a string');
}

// 6. GET /v1/me/devices/{device_id}/access-policy
const policy = await call('GET', `/v1/me/devices/${deviceId}/access-policy`, { token, expect: 200 });
for (const name of ['device_id', 'remote_disabled', 'allow_ips', 'max_sessions', 'allowed_windows', 'timezone', 'exception_dates', 'approval_mode', 'approval_trusted_ips']) {
  field(policy.json, name, undefined, 'GET access-policy');
}
for (const list of ['allow_ips', 'allowed_windows', 'exception_dates', 'approval_trusted_ips']) {
  check(Array.isArray(policy.json[list]), `access-policy: ${list} is not an array`);
}

// 7. GET /v1/me/devices/{device_id}/access-grants
const grants = await call('GET', `/v1/me/devices/${deviceId}/access-grants`, { token, expect: 200 });
const grantItems = field(grants.json, 'items', 'object', 'GET access-grants');
check(Array.isArray(grantItems), 'GET access-grants: items is not an array');
if (Array.isArray(grantItems) && grantItems.length > 0) {
  for (const name of ['id', 'device_id', 'code', 'expires_at', 'max_uses', 'uses', 'active', 'bound_client_id', 'bound_ips', 'created_at']) {
    field(grantItems[0], name, undefined, 'AccessGrant');
  }
  check(Array.isArray(grantItems[0].bound_ips), 'AccessGrant.bound_ips is not an array');
}

// 8. GET /v1/me/devices/{device_id}/access-approvals
const approvals = await call('GET', `/v1/me/devices/${deviceId}/access-approvals`, { token, expect: 200 });
const approvalItems = field(approvals.json, 'items', 'object', 'GET access-approvals');
check(Array.isArray(approvalItems), 'GET access-approvals: items is not an array');
if (Array.isArray(approvalItems) && approvalItems.length > 0) {
  for (const name of ['id', 'device_id', 'client_id', 'client_ip', 'source', 'status', 'expires_at', 'created_at']) {
    field(approvalItems[0], name, undefined, 'AccessApproval');
  }
}
const badStatus = await call('GET', `/v1/me/devices/${deviceId}/access-approvals?status=nope`, { token, expect: 400 });
problemShape(badStatus.json, 'GET access-approvals (unknown status)');

// 9. GET /v1/me/devices/{device_id}/access-approvals:summary
const summary = await call('GET', `/v1/me/devices/${deviceId}/access-approvals:summary`, { token, expect: 200 });
field(summary.json, 'since', 'string', 'GET access-approvals:summary');
field(summary.json, 'total', 'number', 'GET access-approvals:summary');
field(summary.json, 'denial_cooldown_seconds', 'number', 'GET access-approvals:summary');
field(summary.json, 'denial_cooldown_multiplier', 'number', 'GET access-approvals:summary');
field(summary.json, 'denial_cooldown_max_seconds', 'number', 'GET access-approvals:summary');
const byStatus = field(summary.json, 'by_status', 'object', 'GET access-approvals:summary');
for (const state of ['pending', 'approved', 'denied', 'expired', 'consumed']) {
  field(byStatus, state, 'number', 'AccessApprovalSummary.by_status');
}
const deniedSources = field(summary.json, 'denied_sources', 'object', 'GET access-approvals:summary');
check(Array.isArray(deniedSources), 'GET access-approvals:summary: denied_sources is not an array');
if (Array.isArray(deniedSources) && deniedSources.length > 0) {
  field(deniedSources[0], 'client_ip', 'string', 'AccessApprovalSummary.denied_sources[]');
  field(deniedSources[0], 'denials', 'number', 'AccessApprovalSummary.denied_sources[]');
}
const badSince = await call('GET', `/v1/me/devices/${deviceId}/access-approvals:summary?since=nope`, { token, expect: 400 });
problemShape(badSince.json, 'GET access-approvals:summary (bad since)');

// 10. GET /v1/me/recordings (cursor-paginated index)
//
// The field list is shared with the write path at the end of the file, so a
// column added to one place has to exist in the other.
const RECORDING_FIELDS = ['id', 'user_id', 'device_id', 'device_name', 'session_id', 'status', 'format',
  'artifact_path', 'artifact_url', 'thumbnail_path', 'thumbnail_url', 'duration_seconds',
  'size_bytes', 'started_at', 'created_at', 'updated_at'];
const recordings = await call('GET', '/v1/me/recordings', { token, expect: 200 });
const recordingItems = field(recordings.json, 'items', 'object', 'GET /v1/me/recordings');
check(Array.isArray(recordingItems), 'GET /v1/me/recordings: items is not an array');
// The spec deliberately documents no `total` here (the endpoint does not count
// the whole set per page), so a new `total` field is a contract change.
check(!('total' in recordings.json),
  'GET /v1/me/recordings: envelope carries total, but the spec documents no count on this endpoint');
if ('next_cursor' in recordings.json) {
  field(recordings.json, 'next_cursor', 'string', 'GET /v1/me/recordings');
}
if (Array.isArray(recordingItems) && recordingItems.length > 0) {
  const rec = recordingItems[0];
  for (const name of RECORDING_FIELDS) {
    field(rec, name, undefined, 'SessionRecording');
  }
  check(['recording', 'completed', 'failed'].includes(rec.status),
    `SessionRecording.status is "${rec.status}", spec enumerates recording/completed/failed`);
}
const recLimited = await call('GET', '/v1/me/recordings?limit=1', { token, expect: 200 });
const recLimitedItems = field(recLimited.json, 'items', 'object', 'GET /v1/me/recordings?limit=1');
check(Array.isArray(recLimitedItems) && recLimitedItems.length <= 1,
  'GET /v1/me/recordings?limit=1 returned more than one item');

// 11. Approval decisions (approve / deny / claim) - not-found and validation shapes.
// These POSTs use an id that cannot exist, so they pin the documented error
// contract without creating or deciding anything. The happy path needs a real
// pending approval (an online device plus approval_mode), so it is covered by
// the end-to-end run rather than this always-runnable smoke.
if (deviceId) {
  const ghostApprovalId = 2147483647;
  const approveGhost = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${ghostApprovalId}/approve`, { token, expect: 404 });
  problemShape(approveGhost.json, 'POST access-approvals/:id/approve (unknown id)');
  const denyGhost = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${ghostApprovalId}/deny`, { token, expect: 404 });
  problemShape(denyGhost.json, 'POST access-approvals/:id/deny (unknown id)');
  const claimGhost = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
    body: { approval_id: ghostApprovalId, secret: 'openapi-smoke' }, expect: 404,
  });
  problemShape(claimGhost.json, 'POST access-approvals:claim (unknown id)');
  const claimMissing = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, { body: {}, expect: 400 });
  problemShape(claimMissing.json, 'POST access-approvals:claim (missing secret)');
}

// 12. Recording playback tickets and the ticket-gated media route.
// Minting a ticket is stateless (a signed URL; nothing is written), so these
// checks stay in the read-only path. The media fetch is exercised with a
// one-byte range request rather than downloading the file.
const ticketedRecording = Array.isArray(recordingItems)
  ? recordingItems.find((r) => typeof r.artifact_path === 'string' && r.artifact_path !== '')
  : undefined;
if (ticketedRecording) {
  const recordingId = ticketedRecording.id;
  const artifactTicket = await call('POST', `/v1/me/recordings/${recordingId}/artifact-ticket`, { token, expect: 200 });
  const artifactUrl = field(artifactTicket.json, 'url', 'string', 'POST artifact-ticket');
  field(artifactTicket.json, 'expires_at', 'string', 'POST artifact-ticket');
  check(typeof artifactUrl === 'string' && artifactUrl.includes('/artifact?t='),
    `artifact ticket url is "${artifactUrl}", want a signed /artifact?t= path`);

  const thumbTicket = await call('POST', `/v1/me/recordings/${recordingId}/thumbnail-ticket`, { token, expect: 200 });
  const thumbUrl = field(thumbTicket.json, 'url', 'string', 'POST thumbnail-ticket');
  check(typeof thumbUrl === 'string' && thumbUrl.includes('/thumbnail?t='),
    `thumbnail ticket url is "${thumbUrl}", want a signed /thumbnail?t= path`);

  if (typeof artifactUrl === 'string' && artifactUrl.startsWith('/')) {
    // No Authorization and no API key: the signed ticket is the authorization.
    let ranged;
    try {
      ranged = await fetch(server + artifactUrl, { headers: { Range: 'bytes=0-0' } });
    } catch (err) {
      console.error(`transport error fetching a ticket-gated artifact: ${err.message}`);
      process.exit(4);
    }
    check(ranged.status === 200 || ranged.status === 206,
      `ticket-gated artifact fetch returned ${ranged.status}, want 200 or 206`);
  }

  const badTicket = await call('GET', `/v1/recordings/${recordingId}/artifact?t=not-a-ticket`, { expect: 401 });
  problemShape(badTicket.json, 'GET artifact by ticket (bad ticket)');
} else {
  console.log('note: no pre-existing recording carries a stored artifact, so the list-based playback checks were skipped (the write path still covers playback on a recording it creates itself)');
}

const ghostTicket = await call('POST', '/v1/me/recordings/2147483647/artifact-ticket', { token, expect: 404 });
problemShape(ghostTicket.json, 'POST artifact-ticket (unknown recording)');

// 13. GET /v1/update:check - the self-hosted update channel.
//
// The smoke cannot publish a release (that is a super-admin write, documented
// outside this contract), so this pins the shape and the one invariant that
// holds on any database: `latest` is always present, and when it is null
// nothing is available. A published release additionally has to carry the
// documented fields.
const updateCheck = await call('GET',
  '/v1/update:check?platform=windows&arch=x64&channel=stable&current_version=4.2.0.0',
  { expect: 200 });
field(updateCheck.json, 'update_available', 'boolean', 'GET update:check');
field(updateCheck.json, 'mandatory', 'boolean', 'GET update:check');
field(updateCheck.json, 'latest', undefined, 'GET update:check');
check(updateCheck.json.channel === 'stable',
  `update check echo channel=${updateCheck.json.channel}, want stable`);
check(updateCheck.json.platform === 'windows' && updateCheck.json.arch === 'x64',
  `update check echo = ${JSON.stringify(updateCheck.json)}, want windows/x64`);
check(updateCheck.json.current_version === '4.2.0.0',
  `update check current_version=${updateCheck.json.current_version}, want the echoed 4.2.0.0`);
if (updateCheck.json.latest === null) {
  check(updateCheck.json.update_available === false && updateCheck.json.mandatory === false,
    `an empty update channel reported ${JSON.stringify(updateCheck.json)}, want no update`);
} else {
  const latest = updateCheck.json.latest;
  for (const name of ['id', 'channel', 'platform', 'arch', 'version', 'download_url', 'sha256', 'size_bytes', 'mandatory', 'published_at']) {
    field(latest, name, undefined, 'GET update:check (latest)');
  }
  // hosted_here tells the client whether the download_url is this server's own
  // artifact route or somebody else's static host, so its type is contractual.
  field(latest, 'hosted_here', 'boolean', 'GET update:check (latest)');
  check(updateCheck.json.update_available === true,
    'a published release did not report update_available');
}

// The client-facing artifact route. A run with no admin token cannot publish a
// release to download, so this pins the documented failure: an id that cannot
// exist is a Problem-shaped 404 rather than an empty 200.
const updateArtifactMiss = await call('GET', '/v1/update-releases/2147483647/artifact', { expect: 404 });
problemShape(updateArtifactMiss.json, 'GET update-releases/{id}/artifact (unknown release)', 'NOT_FOUND');

const updateDefaultChannel = await call('GET',
  '/v1/update:check?platform=macos&arch=arm64&current_version=4.2.0.0',
  { expect: 200 });
check(updateDefaultChannel.json.channel === 'stable',
  `an omitted channel defaulted to "${updateDefaultChannel.json.channel}", want stable`);

const updateBadPlatform = await call('GET',
  '/v1/update:check?platform=solaris&arch=x64&current_version=4.2.0.0',
  { expect: 400 });
problemShape(updateBadPlatform.json, 'GET update:check (unknown platform)');

const updateBadVersion = await call('GET',
  '/v1/update:check?platform=windows&arch=x64&current_version=not-a-version',
  { expect: 400 });
problemShape(updateBadVersion.json, 'GET update:check (invalid version)');

// 14. GET /v1/me/remote-sessions - the control plane's own session records.
//
// A read-only run cannot open a controller signaling socket (that needs an
// access code and a live host), so this pins the documented envelope and the
// documented failure; a row, when the database has one, additionally has to
// carry the documented fields and agree with its own `active` flag.
const remoteSessions = await call('GET', '/v1/me/remote-sessions?limit=5', { token, expect: 200 });
check(Array.isArray(remoteSessions.json?.items),
  `GET remote-sessions: items is ${typeof remoteSessions.json?.items}, spec says an array`);
for (const row of remoteSessions.json?.items ?? []) {
  for (const name of ['id', 'session_id', 'user_id', 'device_id', 'client_id', 'client_ip', 'started_at', 'ended_at', 'end_reason', 'duration_seconds', 'created_at', 'updated_at']) {
    field(row, name, undefined, 'GET remote-sessions (row)');
  }
  field(row, 'active', 'boolean', 'GET remote-sessions (row)');
  check(row.active === (row.ended_at === null || row.ended_at === undefined),
    `GET remote-sessions: active=${row.active} but ended_at=${row.ended_at}`);
}

const remoteSessionsActive = await call('GET', '/v1/me/remote-sessions?status=active&limit=5', { token, expect: 200 });
check((remoteSessionsActive.json?.items ?? []).every((row) => row.active === true),
  'GET remote-sessions?status=active returned a session that is not active');

const remoteSessionsBadStatus = await call('GET', '/v1/me/remote-sessions?status=soon', { token, expect: 400 });
problemShape(remoteSessionsBadStatus.json, 'GET remote-sessions (unknown status)');

// The detail read is what puts a session back together with the records that
// share its id. A read-only run cannot open a controller socket, so the failure
// shape is pinned; when the database already holds a session, the joined shape is
// checked against the real row.
const ghostSession = await call('GET', '/v1/me/remote-sessions/sig_does_not_exist', { token, expect: 404 });
problemShape(ghostSession.json, 'GET remote-sessions/{id} (unknown session)', 'NOT_FOUND');

if ((remoteSessions.json?.items ?? []).length > 0) {
  const first = remoteSessions.json.items[0];
  const detail = await call('GET', `/v1/me/remote-sessions/${encodeURIComponent(first.session_id)}`,
    { token, expect: 200 });
  field(detail.json, 'session', 'object', 'GET remote-sessions/{id}');
  for (const name of ['recordings', 'access_log']) {
    check(Array.isArray(detail.json?.[name]),
      `GET remote-sessions/{id}: ${name} is not an array`);
  }
  const counts = field(detail.json, 'counts', 'object', 'GET remote-sessions/{id}');
  field(counts, 'recordings', 'number', 'GET remote-sessions/{id} (counts)');
  field(counts, 'access_log', 'number', 'GET remote-sessions/{id} (counts)');
  check(counts.recordings === (detail.json.recordings ?? []).length,
    'GET remote-sessions/{id}: the recording count does not match the list');
  check(counts.access_log === (detail.json.access_log ?? []).length,
    'GET remote-sessions/{id}: the access-log count does not match the list');
  for (const row of detail.json.access_log ?? []) {
    for (const name of ['id', 'device_id', 'owner_id', 'client_id', 'session_id', 'ip', 'result', 'reason', 'created_at']) {
      field(row, name, undefined, 'GET remote-sessions/{id} (access_log row)');
    }
  }
  check(detail.json.session?.session_id === first.session_id,
    'GET remote-sessions/{id}: the detail read returned a different session');
}

// The device-scoped read: the same history, but every entry arrives already
// joined with its records, so a device page does not need a session id (or a
// second request per row) to reach the join.
if (smokeDevice) {
  const deviceSessions = await call('GET', `/v1/me/devices/${smokeDevice}/sessions?limit=3`,
    { token, expect: 200 });
  check(Array.isArray(deviceSessions.json?.items),
    `GET devices/{id}/sessions: items is ${typeof deviceSessions.json?.items}, spec says an array`);
  for (const item of deviceSessions.json?.items ?? []) {
    field(item, 'session', 'object', 'GET devices/{id}/sessions (item)');
    for (const name of ['recordings', 'access_log']) {
      check(Array.isArray(item?.[name]), `GET devices/{id}/sessions: ${name} is not an array`);
    }
    const itemCounts = field(item, 'counts', 'object', 'GET devices/{id}/sessions (item)');
    field(itemCounts, 'recordings', 'number', 'GET devices/{id}/sessions (item counts)');
    field(itemCounts, 'access_log', 'number', 'GET devices/{id}/sessions (item counts)');
    check(itemCounts.recordings === (item.recordings ?? []).length,
      'GET devices/{id}/sessions: an item count disagrees with its list');
    check(item.session?.device_id === smokeDevice,
      'GET devices/{id}/sessions returned a session for a different device');
  }
}

const unknownDeviceSessions = await call('GET', '/v1/me/devices/2147483647/sessions',
  { token, expect: 404 });
problemShape(unknownDeviceSessions.json, 'GET devices/{id}/sessions (unknown device)', 'DEVICE_NOT_FOUND');

if (smokeDevice) {
  // 6. PUT the rule back unchanged, then restore the prior state exactly.
  const putBody = {
    enabled: alertBefore.json.enabled,
    after_minutes: alertBefore.json.after_minutes,
    repeat_minutes: alertBefore.json.repeat_minutes,
    maintenance_windows: alertBefore.json.maintenance_windows ?? [],
    maintenance_timezone: alertBefore.json.maintenance_timezone ?? '',
  };
  const put = await call('PUT', `/v1/me/devices/${deviceId}/offline-alert`, { token, body: putBody, expect: 200 });
  for (const name of ['device_id', 'enabled', 'after_minutes', 'repeat_minutes', 'maintenance_windows', 'in_maintenance_window']) {
    field(put.json, name, undefined, 'PUT offline-alert');
  }
  check(put.json.after_minutes === putBody.after_minutes, 'PUT offline-alert did not echo after_minutes');

  // The GET view cannot distinguish "no rule" from "a disabled rule", so
  // whether a row existed before is inferred from updated_at: the defaults come
  // back without one. If there was no rule, delete the one the PUT created so
  // the smoke leaves the account exactly as it found it.
  if (alertBefore.json.updated_at === undefined || alertBefore.json.updated_at === null) {
    const del = await call('DELETE', `/v1/me/devices/${deviceId}/offline-alert`, { token, expect: 204 });
    check(del.status === 204, 'DELETE offline-alert did not return 204');
  }

  // 7. PUT the access policy back unchanged. A full replace with what was just
  // read is a no-op semantically, so nothing about the device is loosened or
  // tightened by running the smoke.
  const policyPut = {
    remote_disabled: policy.json.remote_disabled,
    allow_ips: policy.json.allow_ips ?? [],
    max_sessions: policy.json.max_sessions,
    allowed_windows: policy.json.allowed_windows ?? [],
    timezone: policy.json.timezone ?? '',
    exception_dates: policy.json.exception_dates ?? [],
    approval_required: policy.json.approval_required ?? false,
    approval_mode: policy.json.approval_mode ?? 'off',
    approval_trusted_ips: policy.json.approval_trusted_ips ?? [],
  };
  const policyRes = await call('PUT', `/v1/me/devices/${deviceId}/access-policy`, { token, body: policyPut, expect: 200 });
  for (const name of ['device_id', 'remote_disabled', 'allow_ips', 'max_sessions', 'allowed_windows', 'timezone', 'exception_dates', 'approval_mode', 'approval_trusted_ips']) {
    field(policyRes.json, name, undefined, 'PUT access-policy');
  }
  check(policyRes.json.approval_mode === policyPut.approval_mode,
    `PUT access-policy changed approval_mode: ${policyRes.json.approval_mode} != ${policyPut.approval_mode}`);
  check(policyRes.json.max_sessions === policyPut.max_sessions, 'PUT access-policy did not echo max_sessions');

  // 8. Issue a short-lived guest code and revoke it. Revocation (not deletion)
  // is the documented way to disable a grant, so this leaves one inert
  // `active: false` row behind rather than pretending to erase it.
  const created = await call('POST', `/v1/me/devices/${deviceId}/access-grants`, {
    token,
    body: { label: 'openapi-smoke', expires_in_minutes: 5, max_uses: 1 },
    expect: 201,
  });
  for (const name of ['id', 'device_id', 'code', 'expires_at', 'max_uses', 'uses', 'active', 'created_at']) {
    field(created.json, name, undefined, 'POST access-grants');
  }
  check(typeof created.json.code === 'string' && created.json.code.length > 0,
    'POST access-grants returned no code');
  const revoked = await call('DELETE', `/v1/me/devices/${deviceId}/access-grants/${created.json.id}`, { token, expect: 200 });
  field(revoked.json, 'success', 'boolean', 'DELETE access-grants');

  // 9. POST access-code:verify with a code that cannot be right, to pin the
  // documented rejection shape.
  const verify = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
    body: { code: '000000', client_id: 'openapi-smoke' },
    expect: 403,
  });
  problemShape(verify.json, 'POST access-code:verify (rejected)', verify.json?.code);

  // 10. The recording write path: index a capture, patch it, upload media and
  // a preview, play the media back through a ticket, then delete the row.
  //
  // This is the half of the recordings surface a read-only run cannot reach,
  // and doing it here means a fresh account covers the whole lifecycle instead
  // of depending on a recording that some earlier run happened to leave behind.
  const newRecording = await call('POST', '/v1/me/recordings', {
    token,
    body: {
      device_id: deviceId,
      device_name: 'openapi-smoke',
      session_id: `smoke-${Date.now()}`,
      status: 'recording',
    },
    expect: 201,
  });
  for (const name of RECORDING_FIELDS) {
    field(newRecording.json, name, undefined, 'POST /v1/me/recordings');
  }
  const recordingId = newRecording.json.id;
  check(newRecording.json.status === 'recording',
    'POST /v1/me/recordings did not keep the status it was given');

  const missingStatus = await call('POST', '/v1/me/recordings', {
    token,
    body: { device_id: deviceId },
    expect: 400,
  });
  problemShape(missingStatus.json, 'POST /v1/me/recordings (missing status)');

  const oneRecording = await call('GET', `/v1/me/recordings/${recordingId}`, { token, expect: 200 });
  check(oneRecording.json.id === recordingId, 'GET recording returned a different row');

  const patched = await call('PATCH', `/v1/me/recordings/${recordingId}`, {
    token,
    body: { status: 'completed', duration_seconds: 3 },
    expect: 200,
  });
  check(patched.json.status === 'completed' && patched.json.duration_seconds === 3,
    'PATCH recording did not apply status/duration_seconds');
  const badPatch = await call('PATCH', `/v1/me/recordings/${recordingId}`, {
    token,
    body: { status: 'finished' },
    expect: 400,
  });
  problemShape(badPatch.json, 'PATCH /v1/me/recordings/:id (invalid status)');

  // A tiny but real MP4 header: the server allow-lists the extension rather
  // than sniffing video containers, and this pins that rule.
  const media = new Uint8Array([0, 0, 0, 24, 102, 116, 121, 112, 109, 112, 52, 50]);
  const mediaForm = new FormData();
  mediaForm.append('file', new Blob([media], { type: 'video/mp4' }), 'smoke.mp4');
  const uploaded = await call('POST', `/v1/me/recordings/${recordingId}/artifact`, {
    token,
    rawBody: mediaForm,
    expect: 200,
  });
  check(uploaded.json.size_bytes === media.length,
    `artifact size_bytes=${uploaded.json.size_bytes}, want ${media.length}`);
  check(uploaded.json.format === 'mp4', `artifact format=${uploaded.json.format}, want mp4`);
  check(typeof uploaded.json.artifact_url === 'string' && uploaded.json.artifact_url !== '',
    'artifact_url is empty after the upload');

  // The resumable half of the upload contract, exercised on the same recording:
  // two chunks with a declared total, a status probe in between, and a wrong
  // offset, which is what a client reconnecting after a dropped response does.
  const chunk1 = media.slice(0, 4);
  const chunk2 = media.slice(4);
  const firstChunk = await call('PATCH', `/v1/me/recordings/${recordingId}/artifact?ext=.mp4`, {
    token,
    rawBody: chunk1,
    contentType: 'application/octet-stream',
    extraHeaders: { 'Upload-Offset': '0', 'Upload-Length': String(media.length) },
    expect: 202,
  });
  const afterFirst = headerNumber(firstChunk, 'Upload-Offset', 'PATCH artifact (first chunk)');
  check(afterFirst === chunk1.length,
    `first chunk left the offset at ${afterFirst}, want ${chunk1.length}`);
  // The 202 body is the recording row, not a staging-only shape: a generated
  // client declares one success type, so both statuses share SessionRecording
  // and the authoritative offset travels in the Upload-Offset header.
  check(firstChunk.json.id === recordingId,
    `PATCH artifact (first chunk) body id=${firstChunk.json.id}, want the row ${recordingId}`);

  // An empty body is a probe: it must not stage anything, and it is how a client
  // that lost its own bookkeeping re-learns the offset.
  const probe = await call('PATCH', `/v1/me/recordings/${recordingId}/artifact?ext=.mp4`, {
    token,
    rawBody: new Uint8Array([]),
    contentType: 'application/octet-stream',
    extraHeaders: { 'Upload-Offset': String(chunk1.length), 'Upload-Length': String(media.length) },
    expect: 202,
  });
  check(headerNumber(probe, 'Upload-Offset', 'PATCH artifact (probe)') === chunk1.length,
    'the probe changed the staged offset');

  // The generated clients cannot send a zero-length body, so the same probe is
  // reachable through an explicit header.
  const headerProbe = await call('PATCH', `/v1/me/recordings/${recordingId}/artifact?ext=.mp4`, {
    token,
    rawBody: new Uint8Array([]),
    contentType: 'application/octet-stream',
    extraHeaders: {
      'Upload-Offset': String(chunk1.length),
      'Upload-Length': String(media.length),
      'Upload-Probe': '1',
    },
    expect: 202,
  });
  check(headerNumber(headerProbe, 'Upload-Offset', 'PATCH artifact (header probe)') === chunk1.length,
    'the Upload-Probe header changed the staged offset');

  // A chunk that starts somewhere else is refused, and the answer carries the
  // offset the server actually holds so the client can resume from it.
  const wrongOffset = await call('PATCH', `/v1/me/recordings/${recordingId}/artifact?ext=.mp4`, {
    token,
    rawBody: new Uint8Array([9]),
    contentType: 'application/octet-stream',
    extraHeaders: { 'Upload-Offset': '5', 'Upload-Length': String(media.length) },
    expect: 409,
  });
  problemShape(wrongOffset.json, 'PATCH artifact (offset mismatch)', 'CONFLICT');
  check(headerNumber(wrongOffset, 'Upload-Offset', 'PATCH artifact (offset mismatch)') === chunk1.length,
    'the 409 did not report the offset the server holds');

  const lastChunk = await call('PATCH', `/v1/me/recordings/${recordingId}/artifact?ext=.mp4`, {
    token,
    rawBody: chunk2,
    contentType: 'application/octet-stream',
    extraHeaders: { 'Upload-Offset': String(chunk1.length), 'Upload-Length': String(media.length) },
    expect: 200,
  });
  check(headerNumber(lastChunk, 'Upload-Offset', 'PATCH artifact (final chunk)') === media.length,
    'the final chunk did not report the completed size');
  check(lastChunk.json.size_bytes === media.length,
    `resumed artifact size_bytes=${lastChunk.json.size_bytes}, want ${media.length}`);
  check(lastChunk.json.format === 'mp4',
    `resumed artifact format=${lastChunk.json.format}, want mp4`);

  // The two documented rejections: an extension that is not on the allow-list,
  // and an empty body.
  const deniedExt = await call('POST', `/v1/me/recordings/${recordingId}/artifact?ext=.exe`, {
    token,
    rawBody: new Uint8Array([1, 2, 3]),
    contentType: 'application/octet-stream',
    expect: 400,
  });
  problemShape(deniedExt.json, 'POST artifact (denied extension)');
  const emptyUpload = await call('POST', `/v1/me/recordings/${recordingId}/artifact?ext=.mp4`, {
    token,
    rawBody: new Uint8Array([]),
    contentType: 'application/octet-stream',
    expect: 400,
  });
  problemShape(emptyUpload.json, 'POST artifact (empty body)');

  // A real 1x1 PNG, and a file that only claims to be one: the preview route
  // sniffs the bytes, so the second upload must be refused.
  const png = Buffer.from(
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8AAAwAB/AMQ/AMZ4zCfAAAAAElFTkSuQmCC',
    'base64',
  );
  const thumbForm = new FormData();
  thumbForm.append('file', new Blob([png], { type: 'image/png' }), 'smoke.png');
  const thumbUpload = await call('POST', `/v1/me/recordings/${recordingId}/thumbnail`, {
    token,
    rawBody: thumbForm,
    expect: 200,
  });
  check(typeof thumbUpload.json.thumbnail_path === 'string' && thumbUpload.json.thumbnail_path !== '',
    'thumbnail_path is empty after the preview upload');
  const fakeThumbForm = new FormData();
  fakeThumbForm.append('file', new Blob([new Uint8Array([1, 2, 3, 4])], { type: 'image/png' }), 'not-really.png');
  const fakeThumb = await call('POST', `/v1/me/recordings/${recordingId}/thumbnail`, {
    token,
    rawBody: fakeThumbForm,
    expect: 400,
  });
  problemShape(fakeThumb.json, 'POST thumbnail (bytes are not an image)');

  // Playback without an Authorization header, on the artifact that was just
  // uploaded: a one-byte range request rather than the whole file.
  const freshTicket = await call('POST', `/v1/me/recordings/${recordingId}/artifact-ticket`, { token, expect: 200 });
  const freshTicketUrl = field(freshTicket.json, 'url', 'string', 'POST artifact-ticket (fresh upload)');
  check(freshTicketUrl.includes('/artifact?t='),
    `fresh ticket url is "${freshTicketUrl}", want a signed /artifact?t= path`);
  const mediaRes = await fetch(new URL(freshTicketUrl, server), { headers: { Range: 'bytes=0-0' } });
  check(mediaRes.status === 206, `ticket media fetch returned ${mediaRes.status}, want 206`);
  check((mediaRes.headers.get('content-type') || '').startsWith('video/mp4'),
    `ticket media content-type is ${mediaRes.headers.get('content-type')}, want video/mp4`);
  const oneByte = new Uint8Array(await mediaRes.arrayBuffer());
  check(oneByte.length === 1, `ticket media fetch returned ${oneByte.length} bytes, want 1`);

  // Delete it again: the smoke leaves the account's recording index as it found
  // it, which is also the only assertion that covers the delete route.
  const deletedRecording = await call('DELETE', `/v1/me/recordings/${recordingId}`, { token, expect: 204 });
  check(deletedRecording.status === 204, 'DELETE recording did not return 204');
  const goneRecording = await call('GET', `/v1/me/recordings/${recordingId}`, { token, expect: 404 });
  problemShape(goneRecording.json, 'GET deleted recording');

  // 11. Second-factor approval: device-code and grant-backed triggers, both
  // outcomes.
  //
  // The read-only checks can pin the 404/400 shapes of these routes but not
  // what a real decision does, because that needs a pending request - and a
  // pending request needs a device that is online and requires approval. The
  // smoke provides both: it turns approval on for its own device, drives the
  // exchange, and puts the policy back immediately afterwards.
  //
  // Residue, stated rather than hidden: the device keeps three decided approval
  // rows (two approved-and-consumed, one denied) plus one consumed grant row
  // and their entries on the access timeline. They are the owner's own history
  // and the reason the flow is auditable.
  const originalApprovalMode = policy.json.approval_mode ?? 'off';
  const approvalPolicy = { ...policyPut, approval_mode: 'always' };
  const approvalOn = await call('PUT', `/v1/me/devices/${deviceId}/access-policy`, {
    token,
    body: approvalPolicy,
    expect: 200,
  });
  check(approvalOn.json.approval_mode === 'always',
    `PUT access-policy left approval_mode "${approvalOn.json.approval_mode}", want always`);

  // The device list carries the plaintext access code, which is what a
  // controller would have been told out of band.
  const approvalDevice = (deviceItems ?? []).find((d) => d.device_id === deviceId);
  const deviceAccessCode = approvalDevice?.access_code ?? '';
  check(typeof deviceAccessCode === 'string' && deviceAccessCode.length > 0,
    'the device list did not return an access_code to verify with');

  // A unique client id per run: denying arms the per-source cooldown, and a
  // fixed id would make the *next* run look like the same denied source.
  const approvalClient = `openapi-smoke-${Date.now()}`;
  const pending = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
    body: { code: deviceAccessCode, client_id: approvalClient },
    expect: 202,
  });
  for (const name of ['approval_id', 'approval_secret', 'status', 'expires_at', 'poll_after_ms']) {
    field(pending.json, name, undefined, 'POST access-code:verify (approval pending)');
  }
  check(pending.json.status === 'pending',
    `approval status is "${pending.json.status}", want pending`);

  const pendingList = await call('GET', `/v1/me/devices/${deviceId}/access-approvals?status=pending`, { token, expect: 200 });
  const pendingRow = (pendingList.json?.items ?? []).find((a) => a.id === pending.json.approval_id);
  check(pendingRow !== undefined, 'the pending request is missing from the owner\'s approval list');
  if (pendingRow) {
    for (const name of ['id', 'device_id', 'client_id', 'client_ip', 'source', 'status', 'expires_at', 'created_at']) {
      field(pendingRow, name, undefined, 'AccessApproval');
    }
    check(pendingRow.source === 'device_code',
      `device-code approval source is "${pendingRow.source}", want device_code`);
  }

  // Before a decision the claim is "keep polling", not an error.
  const claimPending = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
    body: { approval_id: pending.json.approval_id, secret: pending.json.approval_secret },
    expect: 202,
  });
  for (const name of ['status', 'expires_at', 'poll_after_ms']) {
    field(claimPending.json, name, undefined, 'POST access-approvals:claim (pending)');
  }
  check(claimPending.json.status === 'pending',
    `pending claim status is "${claimPending.json.status}", want pending`);

  const approved = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${pending.json.approval_id}/approve`, {
    token,
    body: { reason: 'openapi-smoke' },
    expect: 200,
  });
  check(approved.json.status === 'approved', `approve returned status "${approved.json.status}"`);
  check(typeof approved.json.decided_at === 'string', 'approve did not record decided_at');

  const claimed = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
    body: { approval_id: pending.json.approval_id, secret: pending.json.approval_secret },
    expect: 200,
  });
  for (const name of ['signal_token', 'expires_at', 'status']) {
    field(claimed.json, name, undefined, 'POST access-approvals:claim (approved)');
  }
  check(typeof claimed.json.signal_token === 'string' && claimed.json.signal_token.length > 0,
    'the approved claim returned no signal_token');
  check(claimed.json.status === 'granted',
    `approved claim status is "${claimed.json.status}", want granted`);

  // The approval is single-use: the same secret cannot mint a second token.
  const reclaimed = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
    body: { approval_id: pending.json.approval_id, secret: pending.json.approval_secret },
    expect: 409,
  });
  check(reclaimed.json?.status === 'consumed',
    `re-claim status is "${reclaimed.json?.status}", want consumed`);

  // The grant-backed trigger is a separate branch: approval_mode "grant" must
  // send guest-code sessions to the owner's decision queue while the device's
  // own code still passes. A one-use grant stays unconsumed across the pending
  // stage and is spent only when the approved claim actually mints a token.
  const grantPolicy = { ...policyPut, approval_mode: 'grant' };
  const grantOn = await call('PUT', `/v1/me/devices/${deviceId}/access-policy`, {
    token,
    body: grantPolicy,
    expect: 200,
  });
  check(grantOn.json.approval_mode === 'grant',
    `PUT access-policy left approval_mode "${grantOn.json.approval_mode}", want grant`);

  const approvalGrant = await call('POST', `/v1/me/devices/${deviceId}/access-grants`, {
    token,
    body: { label: 'openapi-smoke-approval', expires_in_minutes: 5, max_uses: 1 },
    expect: 201,
  });
  for (const name of ['id', 'device_id', 'code', 'expires_at', 'max_uses', 'uses', 'active', 'created_at']) {
    field(approvalGrant.json, name, undefined, 'POST access-grants (approval)');
  }
  check(typeof approvalGrant.json.code === 'string' && approvalGrant.json.code.length > 0,
    'approval grant returned no code');

  const grantClient = `${approvalClient}-grant`;
  const grantPending = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
    body: { code: approvalGrant.json.code, client_id: grantClient },
    expect: 202,
  });
  for (const name of ['approval_id', 'approval_secret', 'status', 'expires_at', 'poll_after_ms']) {
    field(grantPending.json, name, undefined, 'POST access-code:verify (grant approval pending)');
  }
  check(grantPending.json.status === 'pending',
    `grant approval status is "${grantPending.json.status}", want pending`);

  const grantPendingList = await call('GET', `/v1/me/devices/${deviceId}/access-approvals?status=pending`, { token, expect: 200 });
  const grantApprovalRow = (grantPendingList.json?.items ?? []).find((a) => a.id === grantPending.json.approval_id);
  check(grantApprovalRow !== undefined, 'the grant-backed pending request is missing from the owner\'s list');
  if (grantApprovalRow) {
    check(grantApprovalRow.source === 'grant',
      `grant approval source is "${grantApprovalRow.source}", want grant`);
    check(grantApprovalRow.grant_id === approvalGrant.json.id,
      `grant approval grant_id is "${grantApprovalRow.grant_id}", want ${approvalGrant.json.id}`);
  }

  const grantApproved = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${grantPending.json.approval_id}/approve`, {
    token,
    body: { reason: 'openapi-smoke-grant' },
    expect: 200,
  });
  check(grantApproved.json.status === 'approved',
    `grant approve returned status "${grantApproved.json.status}"`);

  const grantClaimed = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
    body: { approval_id: grantPending.json.approval_id, secret: grantPending.json.approval_secret },
    expect: 200,
  });
  field(grantClaimed.json, 'signal_token', 'string', 'POST access-approvals:claim (grant approved)');
  field(grantClaimed.json, 'status', 'string', 'POST access-approvals:claim (grant approved)');
  check(grantClaimed.json.status === 'granted',
    `grant-approved claim status is "${grantClaimed.json.status}", want granted`);
  check(typeof grantClaimed.json.signal_token === 'string' && grantClaimed.json.signal_token.length > 0,
    'the grant-backed approved claim returned no signal_token');

  const grantList = await call('GET', `/v1/me/devices/${deviceId}/access-grants`, { token, expect: 200 });
  const spentGrant = (grantList.json?.items ?? []).find((g) => g.id === approvalGrant.json.id);
  check(spentGrant !== undefined, 'the approval grant disappeared from the owner\'s grant list');
  if (spentGrant) {
    check(spentGrant.uses === 1,
      `approval grant uses is ${spentGrant.uses}, want 1 after the approved claim`);
    check(spentGrant.active === false,
      'approval grant is still active after its only approved claim');
  }

  // The queue guard and denial cooldown are security controls, not just unit
  // test subjects. Fill the device's pending queue from five distinct sources,
  // show the sixth source is refused, then deny the five to clear the queue
  // before proving the first denied source is cooled down while a fresh source
  // is still admitted.
  const queueOn = await call('PUT', `/v1/me/devices/${deviceId}/access-policy`, {
    token,
    body: { ...policyPut, approval_mode: 'always' },
    expect: 200,
  });
  check(queueOn.json.approval_mode === 'always',
    `PUT access-policy left approval_mode "${queueOn.json.approval_mode}", want always for queue guard`);

  const queueBaseIP = '198.51.100.';
  const queuedApprovals = [];
  for (let i = 1; i <= 5; i++) {
    const sourceIP = `${queueBaseIP}${i}`;
    const queued = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
      body: { code: deviceAccessCode, client_id: `${approvalClient}-queue-${i}` },
      extraHeaders: { 'X-Forwarded-For': sourceIP },
      expect: 202,
    });
    queuedApprovals.push({ ...queued.json, sourceIP });
  }

  const queueOverflow = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
    body: { code: deviceAccessCode, client_id: `${approvalClient}-queue-overflow` },
    extraHeaders: { 'X-Forwarded-For': `${queueBaseIP}6` },
    expect: 429,
  });
  problemShape(queueOverflow.json, 'approval queue overflow', 'RATE_LIMITED');
  check(queueOverflow.headers.get('Retry-After') !== null,
    'approval queue overflow response has no Retry-After header');

  for (const queued of queuedApprovals) {
    const denied = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${queued.approval_id}/deny`, {
      token,
      body: { reason: 'openapi-smoke-queue' },
      expect: 200,
    });
    check(denied.json.status === 'denied',
      `queue cleanup deny returned status "${denied.json.status}", want denied`);
  }

  const cooled = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
    body: { code: deviceAccessCode, client_id: `${approvalClient}-cooldown` },
    extraHeaders: { 'X-Forwarded-For': queuedApprovals[0].sourceIP },
    expect: 429,
  });
  problemShape(cooled.json, 'denied-source cooldown', 'RATE_LIMITED');
  const retryAfter = Number(cooled.headers.get('Retry-After'));
  check(Number.isFinite(retryAfter) && retryAfter > 0,
    `denied-source cooldown Retry-After is "${cooled.headers.get('Retry-After')}"`);

  const freshSource = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
    body: { code: deviceAccessCode, client_id: `${approvalClient}-fresh-source` },
    extraHeaders: { 'X-Forwarded-For': `${queueBaseIP}77` },
    expect: 202,
  });
  const freshDenied = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${freshSource.json.approval_id}/deny`, {
    token,
    body: { reason: 'openapi-smoke-fresh-source' },
    expect: 200,
  });
  check(freshDenied.json.status === 'denied',
    `fresh-source deny returned status "${freshDenied.json.status}", want denied`);

  // The other outcome, on a fresh request: deny, then the claim is refused.
  //
  // Opt-in, because a deny arms the per-(device, source address) cooldown
  // (APPROVAL_DENIAL_COOLDOWN_SECONDS, five minutes by default): running it
  // unconditionally would make a second smoke run against the same device fail
  // before the approve leg even started. CI sets QD_SMOKE_APPROVAL_DENY=1
  // because its device is created fresh for every run and nothing follows it.
  if (process.env.QD_SMOKE_APPROVAL_DENY === '1') {
    const deniedRequest = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
      body: { code: deviceAccessCode, client_id: `${approvalClient}-denied` },
      expect: 202,
    });
    const denied = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${deniedRequest.json.approval_id}/deny`, {
      token,
      body: { reason: 'openapi-smoke' },
      expect: 200,
    });
    check(denied.json.status === 'denied', `deny returned status "${denied.json.status}"`);
    const claimDenied = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
      body: { approval_id: deniedRequest.json.approval_id, secret: deniedRequest.json.approval_secret },
      expect: 403,
    });
    problemShape(claimDenied.json, 'POST access-approvals:claim (denied)');
  }

  // Host disappears while the owner is deciding. The approval can still be
  // approved, but claim must refuse with HOST_OFFLINE and leave the row
  // approved rather than consuming a session secret against an offline host.
  // This is opt-in because it needs Redis access and deliberately leaves the
  // disposable seeded device offline at the end of the run.
  if (process.env.QD_SMOKE_HOST_OFFLINE === '1') {
    const offlinePending = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
      body: { code: deviceAccessCode, client_id: `${approvalClient}-offline` },
      extraHeaders: { 'X-Forwarded-For': `${queueBaseIP}88` },
      expect: 202,
    });
    const deleted = await redisCommand('DEL', `qd:presence:device:${deviceId}:hb`);
    check(deleted === 1,
      `host-offline setup deleted ${deleted} heartbeat keys, want 1`);

    const offlineApproved = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${offlinePending.json.approval_id}/approve`, {
      token,
      body: { reason: 'openapi-smoke-host-offline' },
      expect: 200,
    });
    check(offlineApproved.json.status === 'approved',
      `host-offline approve returned status "${offlineApproved.json.status}", want approved`);

    const hostOffline = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
      body: { approval_id: offlinePending.json.approval_id, secret: offlinePending.json.approval_secret },
      expect: 409,
    });
    problemShape(hostOffline.json, 'approval claim with host offline', 'HOST_OFFLINE');

    const approvedList = await call('GET', `/v1/me/devices/${deviceId}/access-approvals?status=approved`, { token, expect: 200 });
    const stillApproved = (approvedList.json?.items ?? []).find((a) => a.id === offlinePending.json.approval_id);
    check(stillApproved !== undefined, 'host-offline approval disappeared after the refused claim');
    if (stillApproved) {
      check(stillApproved.status === 'approved',
        `host-offline approval became "${stillApproved.status}", want approved`);
    }
  }

  // Put the policy back exactly as it was before the flow.
  const approvalOff = await call('PUT', `/v1/me/devices/${deviceId}/access-policy`, {
    token,
    body: policyPut,
    expect: 200,
  });
  check(approvalOff.json.approval_mode === originalApprovalMode,
    `access-policy was not restored: approval_mode "${approvalOff.json.approval_mode}", want "${originalApprovalMode}"`);
}

if (failures.length > 0) {
  console.error(`FAIL: ${failures.length} of ${checks} documented assertions failed`);
  for (const f of failures) console.error('  - ' + f);
  process.exit(3);
}

const scope = smokeDevice ? 'read + write operations' : 'read-only operations';
console.log(`PASS: ${checks} documented assertions hold against ${server} (${scope})`);
if (!smokeDevice) {
  console.log('note: set QD_SMOKE_DEVICE=<device_id> to also check the offline-alert PUT, the rejected access-code shape and the recording write path');
}
