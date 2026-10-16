'use strict';

// Drive the chunked recording upload through the committed TypeScript client.
// The Go client has sdk/go/examples/resumableupload; this is the same contract
// through the `typescript-fetch` client: two PATCH chunks with a declared
// total, an `Upload-Probe: 1` offset probe, a 409 resync from the response
// header, server-side finalisation, a byte-identical ticket read back, and a
// delete.
//
// Usage (after `npm install && npm run build`):
//
//   QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=123456789 \
//     node examples/resumable-upload.js
//
// No online host is required: a recording row is an index entry, not a session.

const { AuthApi, Configuration, RecordingsApi } = require('../dist/index.js');

const TOTAL = 1 << 20; // 1 MiB

function envOr(key, fallback) {
  return process.env[key] || fallback;
}

function offsetOf(response) {
  const raw = response.headers.get('Upload-Offset');
  if (!raw) throw new Error('response carried no Upload-Offset header');
  return Number(raw);
}

async function append(recordings, recordingId, offset, data, probe = false) {
  const response = await recordings.appendRecordingArtifactRaw({
    ext: '.mp4',
    uploadOffset: offset,
    recordingId,
    uploadLength: String(TOTAL),
    uploadProbe: probe ? '1' : undefined,
    body: new Blob([data]),
  });
  if (response.raw.status !== 202) {
    throw new Error(`append at offset ${offset} answered ${response.raw.status}, want 202`);
  }
  return offsetOf(response.raw);
}

async function main() {
  const server = envOr('QD_SERVER', 'http://127.0.0.1:8000');
  const apiKey = envOr('QD_API_KEY', 'qd-local-dev-key');
  const identifier = process.env.QD_IDENTIFIER;
  const password = process.env.QD_PASSWORD;
  const deviceId = process.env.QD_SMOKE_DEVICE;
  if (!identifier || !password || !deviceId) {
    throw new Error('QD_IDENTIFIER, QD_PASSWORD and QD_SMOKE_DEVICE are required');
  }

  // The token is only known after login, so the configuration reads it through
  // a closure instead of being rebuilt for every later call.
  let accessToken = '';
  const config = new Configuration({
    basePath: server,
    apiKey: () => apiKey,
    accessToken: () => accessToken,
  });

  const session = await new AuthApi(config).createSession({
    authSessionRequest: { identifier, password },
  });
  accessToken = session.accessToken;
  console.log('login ok');

  const recordings = new RecordingsApi(config);
  const created = await recordings.createSessionRecording({
    sessionRecordingInput: {
      deviceId,
      deviceName: 'sdk-resumable-ts',
      sessionId: `sdk-resumable-ts-${process.pid}`,
      status: 'recording',
    },
  });
  const recordingId = created.id;
  console.log(`recording ${recordingId} created for a ${TOTAL}-byte upload`);

  try {
    const media = Buffer.alloc(TOTAL);
    for (let i = 0; i < TOTAL; i += 1) media[i] = (i * 31 + 7) & 0xff;

    let offset = await append(recordings, recordingId, 0, media.subarray(0, 400000));
    if (offset !== 400000) throw new Error(`first chunk left offset=${offset}, want 400000`);
    console.log(`chunk 1 staged, server reports offset ${offset}`);

    // The probe tells the server to ignore the body and stage nothing.
    offset = await append(recordings, recordingId, offset, Buffer.from([0]), true);
    if (offset !== 400000) throw new Error(`probe moved the offset to ${offset}, want 400000`);
    console.log(`probe (Upload-Probe: 1) answered offset ${offset} without staging`);

    try {
      await recordings.appendRecordingArtifact({
        ext: '.mp4',
        uploadOffset: 12345,
        recordingId,
        uploadLength: String(TOTAL),
        body: new Blob([media.subarray(400000, 800000)]),
      });
      throw new Error('a chunk with a wrong offset was accepted');
    } catch (error) {
      if (error && error.response && error.response.status === 409) {
        const conflictOffset = offsetOf(error.response);
        if (conflictOffset !== 400000) {
          throw new Error(`409 reported offset=${conflictOffset}, want 400000`);
        }
        console.log(
          `wrong offset refused with 409 and the authoritative offset ${conflictOffset}`,
        );
      } else {
        throw error;
      }
    }

    offset = await append(recordings, recordingId, offset, media.subarray(400000, 800000));
    if (offset !== 800000) throw new Error(`second chunk left offset=${offset}, want 800000`);
    console.log(`chunk 2 staged, server reports offset ${offset}`);

    // The last byte lands at the declared total, so the server finalizes by
    // itself; the 200 answer is the completed row.
    const row = await recordings.appendRecordingArtifact({
      ext: '.mp4',
      uploadOffset: offset,
      recordingId,
      uploadLength: String(TOTAL),
      body: new Blob([media.subarray(800000)]),
    });
    if (row.sizeBytes !== TOTAL) {
      throw new Error(`finalized size_bytes=${row.sizeBytes}, want ${TOTAL}`);
    }
    console.log(`final chunk finalized the upload (status=${row.status}, size=${row.sizeBytes})`);

    const ticket = await recordings.createRecordingArtifactTicket({ recordingId });
    // The ticket url is a relative path plus `?t=`, so it is parsed as a query
    // string rather than handed to `new URL` without a base.
    const queryStart = ticket.url.indexOf('?');
    const token = new URLSearchParams(queryStart >= 0 ? ticket.url.slice(queryStart + 1) : '').get('t');
    if (!token) throw new Error(`ticket url ${JSON.stringify(ticket.url)} carries no t= parameter`);
    const blob = await recordings.downloadRecordingArtifactByTicket({ t: token, recordingId });
    const got = Buffer.from(await blob.arrayBuffer());
    if (got.length !== media.length || !got.equals(media)) {
      throw new Error(`downloaded ${got.length} bytes, want ${media.length} with identical content`);
    }
    console.log(`artifact read back through the ticket (${got.length} bytes, identical)`);
    console.log('PASS');
  } finally {
    await recordings.deleteSessionRecording({ recordingId });
    console.log(`recording ${recordingId} deleted`);
  }
}

main().catch((error) => {
  console.error(error && error.message ? error.message : error);
  process.exit(1);
});
