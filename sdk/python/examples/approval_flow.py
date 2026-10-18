#!/usr/bin/env python3
"""Drive the second-factor approval flow through the committed Python client.

The Go client already had `sdk/go/examples/approvalflow`; this is the same
workflow through the generated Python client, and it is the stronger proof of
the contract change because Python maps each status to its own class:
`access-code:verify` answers `AccessCodeApprovalPending` on the 202, and the
claim endpoint answers `ApprovalClaimPending`. Both still carry `status`, so a
typed caller reads the pending approval instead of guessing at the raw body.

Usage:

    QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=123456789 \
        python examples/approval_flow.py

Precondition: the device must be online. An approval request is only created
for a host the server currently sees, so a deleted or expired heartbeat makes
`access-code:verify` answer 409 HOST_OFFLINE before the owner is ever asked.
"""

import os
import sys

import quickdesk_signaling
from quickdesk_signaling.models.access_code_verify_request import AccessCodeVerifyRequest
from quickdesk_signaling.models.approval_claim_request import ApprovalClaimRequest
from quickdesk_signaling.models.approve_access_approval_request import (
    ApproveAccessApprovalRequest,
)
from quickdesk_signaling.models.auth_session_request import AuthSessionRequest
from quickdesk_signaling.models.device_access_policy_input import (
    DeviceAccessPolicyInput,
)


def env_or(key: str, fallback: str) -> str:
    return os.environ.get(key) or fallback


def policy_input(before, approval_mode: str) -> DeviceAccessPolicyInput:
    """Build a full policy replace that only changes `approval_mode`."""
    return DeviceAccessPolicyInput(
        remote_disabled=before.remote_disabled,
        allow_ips=before.allow_ips,
        max_sessions=before.max_sessions,
        allowed_windows=before.allowed_windows,
        timezone=before.timezone,
        exception_dates=before.exception_dates,
        approval_required=before.approval_required,
        approval_mode=approval_mode,
        approval_trusted_ips=before.approval_trusted_ips,
    )


def main() -> int:
    server = env_or("QD_SERVER", "http://127.0.0.1:8000")
    api_key = env_or("QD_API_KEY", "qd-local-dev-key")
    identifier = os.environ.get("QD_IDENTIFIER")
    password = os.environ.get("QD_PASSWORD")
    device_id = os.environ.get("QD_SMOKE_DEVICE")
    if not identifier or not password or not device_id:
        print(
            "QD_IDENTIFIER, QD_PASSWORD and QD_SMOKE_DEVICE are required",
            file=sys.stderr,
        )
        return 1

    config = quickdesk_signaling.Configuration(
        host=server,
        api_key={"ApiKeyAuth": api_key},
    )

    with quickdesk_signaling.ApiClient(config) as client:
        session = quickdesk_signaling.AuthApi(client).create_session(
            AuthSessionRequest(identifier=identifier, password=password)
        )
        config.access_token = session.access_token
        print("login ok")

        devices = quickdesk_signaling.DevicesApi(client)

        # The device list carries the plaintext access code, which is what a
        # controller would have been told out of band.
        access_code = ""
        for device in devices.list_my_devices().items:
            if device.device_id == device_id:
                access_code = device.access_code or ""
        if not access_code:
            print(f"device {device_id} has no access code", file=sys.stderr)
            return 1

        # The policy is a full replace, so put back exactly what was found.
        before = devices.get_access_policy(device_id)
        try:
            devices.put_access_policy(device_id, policy_input(before, "always"))
            print(f"device {device_id} now requires owner approval")

            # A unique client id per run: a denied source arms a cooldown.
            client_id = f"sdk-approval-python-{os.getpid()}"
            pending = devices.verify_access_code(
                device_id,
                AccessCodeVerifyRequest(code=access_code, client_id=client_id),
            )
            if pending.status != "pending":
                print(f"verify status={pending.status!r}, want pending", file=sys.stderr)
                return 1
            approval_id = pending.approval_id
            approval_secret = pending.approval_secret
            print(
                f"controller got a pending approval {approval_id} "
                f"(poll after {pending.poll_after_ms}ms)"
            )

            early = devices.claim_access_approval(
                device_id,
                ApprovalClaimRequest(
                    approval_id=approval_id, secret=approval_secret
                ),
            )
            if early.status != "pending":
                print(f"early claim status={early.status!r}, want pending", file=sys.stderr)
                return 1
            print("early claim answered pending, as documented")

            pending_rows = devices.list_access_approvals(device_id, status="pending").items
            row = next((r for r in pending_rows if r.id == approval_id), None)
            if row is None:
                print(f"approval {approval_id} is missing from the owner list", file=sys.stderr)
                return 1
            if row.client_id != client_id:
                print(
                    f"approval {approval_id} shows client_id={row.client_id!r}, "
                    f"want {client_id!r}",
                    file=sys.stderr,
                )
                return 1
            print(f"owner sees approval {approval_id} among {len(pending_rows)} pending requests")

            approved = devices.approve_access_approval(
                device_id,
                approval_id,
                ApproveAccessApprovalRequest(reason="sdk approval example"),
            )
            if approved.status != "approved":
                print(f"approve status={approved.status!r}, want approved", file=sys.stderr)
                return 1
            print(f"owner approved {approval_id}")

            granted = devices.claim_access_approval(
                device_id,
                ApprovalClaimRequest(
                    approval_id=approval_id, secret=approval_secret
                ),
            )
            if granted.status != "granted":
                print(f"claim status={granted.status!r}, want granted", file=sys.stderr)
                return 1
            token = granted.signal_token or ""
            if not token:
                print("the granted claim carried no signal_token", file=sys.stderr)
                return 1
            print(f"claimed the approval for a signal token ({len(token)} chars)")
            print("PASS")
        finally:
            # A failure part-way through must not leave the shared account in
            # `approval_mode: always`.
            devices.put_access_policy(
                device_id, policy_input(before, before.approval_mode or "off")
            )
            print("access policy restored")
    return 0


if __name__ == "__main__":
    sys.exit(main())
