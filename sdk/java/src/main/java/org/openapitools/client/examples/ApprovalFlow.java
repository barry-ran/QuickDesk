package org.openapitools.client.examples;

// Drive the second-factor approval flow through the committed Java client.
// The Go client has sdk/go/examples/approvalflow; this proves the same workflow
// through the generated Java client, where the 202 answers still deserialize
// into AccessCodeVerifyResponse and carry status=pending.
//
// Usage (from sdk/java, after `mvn -q -DskipTests compile`):
//
//   QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=123456789 \
//     java -cp "target/classes:$(cat /tmp/qd-java-cp.txt)" \
//     org.openapitools.client.examples.ApprovalFlow
//
// Precondition: the device must be online. An approval request is only created
// for a host the server currently sees, so a deleted or expired heartbeat makes
// `access-code:verify` answer 409 HOST_OFFLINE before the owner is ever asked.

import org.openapitools.client.ApiClient;
import org.openapitools.client.api.AuthApi;
import org.openapitools.client.api.DevicesApi;
import org.openapitools.client.model.AccessApproval;
import org.openapitools.client.model.AccessCodeVerifyRequest;
import org.openapitools.client.model.AccessCodeVerifyResponse;
import org.openapitools.client.model.ApprovalClaimRequest;
import org.openapitools.client.model.ApproveAccessApprovalRequest;
import org.openapitools.client.model.AuthSessionRequest;
import org.openapitools.client.model.AuthSessionResponse;
import org.openapitools.client.model.Device;
import org.openapitools.client.model.DeviceAccessPolicy;
import org.openapitools.client.model.DeviceAccessPolicyInput;
import org.openapitools.client.model.ListAccessApprovals200Response;
import org.openapitools.client.model.ListMyDevices200Response;

import java.util.List;

public final class ApprovalFlow {

    private ApprovalFlow() {
    }

    public static void main(String[] args) throws Exception {
        String server = envOr("QD_SERVER", "http://127.0.0.1:8000");
        String apiKey = envOr("QD_API_KEY", "qd-local-dev-key");
        String identifier = System.getenv("QD_IDENTIFIER");
        String password = System.getenv("QD_PASSWORD");
        String deviceId = System.getenv("QD_SMOKE_DEVICE");
        if (identifier == null || password == null || deviceId == null) {
            System.err.println("QD_IDENTIFIER, QD_PASSWORD and QD_SMOKE_DEVICE are required");
            System.exit(1);
        }

        ApiClient client = new ApiClient();
        client.setBasePath(server);
        client.setApiKey(apiKey);

        AuthSessionResponse session = new AuthApi(client).createSession(
                new AuthSessionRequest().identifier(identifier).password(password));
        // `BearerAuth` is an HTTP bearer scheme, not OAuth2, so the generated
        // `setAccessToken` helper throws ("No OAuth2 authentication
        // configured!"). `setBearerToken` is the matching helper for it.
        client.setBearerToken(session.getAccessToken());
        System.out.println("login ok");

        DevicesApi devices = new DevicesApi(client);

        // The device list carries the plaintext access code, which is what a
        // controller would have been told out of band.
        String accessCode = "";
        ListMyDevices200Response devicesList = devices.listMyDevices();
        for (Device device : devicesList.getItems()) {
            if (deviceId.equals(device.getDeviceId())) {
                accessCode = device.getAccessCode() == null ? "" : device.getAccessCode();
            }
        }
        if (accessCode.isEmpty()) {
            System.err.println("device " + deviceId + " has no access code in the device list");
            System.exit(1);
        }

        // The policy is a full replace, so put back exactly what was found.
        DeviceAccessPolicy before = devices.getAccessPolicy(deviceId);
        try {
            devices.putAccessPolicy(deviceId, policyInput(before, "always"));
            System.out.println("device " + deviceId + " now requires owner approval");

            // Unique per run: denying a request arms a per-source cooldown.
            String clientId = "sdk-approval-java-" + ProcessHandle.current().pid();
            AccessCodeVerifyResponse pending = devices.verifyAccessCode(
                    deviceId,
                    new AccessCodeVerifyRequest().code(accessCode).clientId(clientId));
            if (pending.getStatus() != AccessCodeVerifyResponse.StatusEnum.PENDING) {
                throw new IllegalStateException(
                        "verify status=" + pending.getStatus() + ", want pending");
            }
            Long approvalId = pending.getApprovalId();
            String approvalSecret = pending.getApprovalSecret();
            if (approvalId == null || approvalSecret == null || approvalSecret.isEmpty()) {
                throw new IllegalStateException(
                        "the pending answer carried no approval_id/approval_secret");
            }
            System.out.println("controller got a pending approval " + approvalId
                    + " (poll after " + pending.getPollAfterMs() + "ms)");

            AccessCodeVerifyResponse early = devices.claimAccessApproval(
                    deviceId,
                    new ApprovalClaimRequest().approvalId(approvalId).secret(approvalSecret));
            if (early.getStatus() != AccessCodeVerifyResponse.StatusEnum.PENDING) {
                throw new IllegalStateException(
                        "early claim status=" + early.getStatus() + ", want pending");
            }
            System.out.println("early claim answered pending, as documented");

            ListAccessApprovals200Response pendingList =
                    devices.listAccessApprovals(deviceId, "pending", null);
            List<AccessApproval> rows = pendingList.getItems();
            AccessApproval row = null;
            for (AccessApproval candidate : rows) {
                if (approvalId.equals(candidate.getId())) {
                    row = candidate;
                }
            }
            if (row == null) {
                throw new IllegalStateException(
                        "approval " + approvalId + " is missing from the owner's pending list");
            }
            if (!clientId.equals(row.getClientId())) {
                throw new IllegalStateException("approval " + approvalId
                        + " shows client_id=" + row.getClientId() + ", want " + clientId);
            }
            System.out.println("owner sees approval " + approvalId
                    + " among " + rows.size() + " pending requests");

            AccessApproval approved = devices.approveAccessApproval(
                    deviceId,
                    approvalId,
                    new ApproveAccessApprovalRequest().reason("sdk approval example"));
            if (approved.getStatus() != AccessApproval.StatusEnum.APPROVED) {
                throw new IllegalStateException(
                        "approve status=" + approved.getStatus() + ", want approved");
            }
            System.out.println("owner approved " + approvalId);

            AccessCodeVerifyResponse granted = devices.claimAccessApproval(
                    deviceId,
                    new ApprovalClaimRequest().approvalId(approvalId).secret(approvalSecret));
            if (granted.getStatus() != AccessCodeVerifyResponse.StatusEnum.GRANTED) {
                throw new IllegalStateException(
                        "claim status=" + granted.getStatus() + ", want granted");
            }
            String token = granted.getSignalToken();
            if (token == null || token.isEmpty()) {
                throw new IllegalStateException("the granted claim carried no signal_token");
            }
            System.out.println("claimed the approval for a signal token (" + token.length() + " chars)");
            System.out.println("PASS");
        } finally {
            devices.putAccessPolicy(deviceId, policyInput(before, "restore"));
            System.out.println("access policy restored");
        }
    }

    private static DeviceAccessPolicyInput policyInput(DeviceAccessPolicy before, String mode) {
        String approvalMode = "restore".equals(mode)
                ? (before.getApprovalMode() == null ? "off" : before.getApprovalMode().getValue())
                : mode;
        return new DeviceAccessPolicyInput()
                .remoteDisabled(before.getRemoteDisabled())
                .allowIps(before.getAllowIps())
                .maxSessions(before.getMaxSessions())
                .allowedWindows(before.getAllowedWindows())
                .timezone(before.getTimezone())
                .exceptionDates(before.getExceptionDates())
                .approvalRequired(before.getApprovalRequired())
                .approvalMode(DeviceAccessPolicyInput.ApprovalModeEnum.fromValue(approvalMode))
                .approvalTrustedIps(before.getApprovalTrustedIps());
    }

    private static String envOr(String key, String fallback) {
        String value = System.getenv(key);
        return value == null || value.isEmpty() ? fallback : value;
    }
}
