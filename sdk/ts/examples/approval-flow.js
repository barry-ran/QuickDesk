'use strict';

// Drive the second-factor approval flow through the committed TypeScript
// client. The Go client has `sdk/go/examples/approvalflow`; this proves the
// same workflow through the `typescript-fetch` client, where a 202 is a
// success response that still deserializes into `AccessCodeVerifyResponse`.
//
// Usage (after `npm install && npm run build`):
//
//   QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=123456789 \
//     node examples/approval-flow.js
//
// Precondition: the device must be online. An approval request is only created
// for a host the server currently sees, so a deleted or expired heartbeat makes
// `access-code:verify` answer 409 HOST_OFFLINE before the owner is ever asked.

const { AuthApi, Configuration, DevicesApi } = require('../dist/index.js');

function envOr(key, fallback) {
  return process.env[key] || fallback;
}

// The policy is a full replace, so only `approvalMode` is allowed to change.
function policyInput(before, approvalMode) {
  return {
    remoteDisabled: before.remoteDisabled,
    allowIps: before.allowIps,
    maxSessions: before.maxSessions,
    allowedWindows: before.allowedWindows,
    timezone: before.timezone,
    exceptionDates: before.exceptionDates,
    approvalRequired: before.approvalRequired,
    approvalMode,
    approvalTrustedIps: before.approvalTrustedIps,
  };
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

  const devices = new DevicesApi(config);
  const list = await devices.listMyDevices();
  const device = (list.items || []).find((item) => item.deviceId === deviceId);
  const accessCode = device ? device.accessCode : '';
  if (!accessCode) {
    throw new Error(`device ${deviceId} has no access code in the device list`);
  }

  const before = await devices.getAccessPolicy({ deviceId });
  try {
    await devices.putAccessPolicy({
      deviceId,
      deviceAccessPolicyInput: policyInput(before, 'always'),
    });
    console.log(`device ${deviceId} now requires owner approval`);

    // Unique per run: denying a request arms a per-source cooldown.
    const clientId = `sdk-approval-ts-${process.pid}`;
    const pending = await devices.verifyAccessCode({
      deviceId,
      accessCodeVerifyRequest: { code: accessCode, clientId },
    });
    if (pending.status !== 'pending') {
      throw new Error(`verify status=${JSON.stringify(pending.status)}, want pending`);
    }
    const approvalId = pending.approvalId;
    const approvalSecret = pending.approvalSecret;
    if (!approvalId || !approvalSecret) {
      throw new Error('the pending answer carried no approval_id/approval_secret');
    }
    console.log(
      `controller got a pending approval ${approvalId} (poll after ${pending.pollAfterMs}ms)`,
    );

    const early = await devices.claimAccessApproval({
      deviceId,
      approvalClaimRequest: { approvalId, secret: approvalSecret },
    });
    if (early.status !== 'pending') {
      throw new Error(`early claim status=${JSON.stringify(early.status)}, want pending`);
    }
    console.log('early claim answered pending, as documented');

    const pendingList = await devices.listAccessApprovals({ deviceId, status: 'pending' });
    const rows = pendingList.items || [];
    const row = rows.find((item) => item.id === approvalId);
    if (!row) {
      throw new Error(`approval ${approvalId} is missing from the owner's pending list`);
    }
    if (row.clientId !== clientId) {
      throw new Error(
        `approval ${approvalId} shows client_id=${JSON.stringify(row.clientId)}, want ${JSON.stringify(clientId)}`,
      );
    }
    console.log(`owner sees approval ${approvalId} among ${rows.length} pending requests`);

    const approved = await devices.approveAccessApproval({
      deviceId,
      approvalId,
      approveAccessApprovalRequest: { reason: 'sdk approval example' },
    });
    if (approved.status !== 'approved') {
      throw new Error(`approve status=${JSON.stringify(approved.status)}, want approved`);
    }
    console.log(`owner approved ${approvalId}`);

    const granted = await devices.claimAccessApproval({
      deviceId,
      approvalClaimRequest: { approvalId, secret: approvalSecret },
    });
    if (granted.status !== 'granted') {
      throw new Error(`claim status=${JSON.stringify(granted.status)}, want granted`);
    }
    if (!granted.signalToken) {
      throw new Error('the granted claim carried no signal_token');
    }
    console.log(`claimed the approval for a signal token (${granted.signalToken.length} chars)`);
    console.log('PASS');
  } finally {
    await devices.putAccessPolicy({
      deviceId,
      deviceAccessPolicyInput: policyInput(before, before.approvalMode || 'off'),
    });
    console.log('access policy restored');
  }
}

main().catch((error) => {
  console.error(error && error.message ? error.message : error);
  process.exit(1);
});
