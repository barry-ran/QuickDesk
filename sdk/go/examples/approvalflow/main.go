// Command approvalflow drives the whole second-factor approval workflow through
// the committed Go client: the owner turns approval on, a controller presents
// the access code and receives a pending approval, the owner sees it and
// approves, and the controller exchanges the approval for a signal token.
//
// It is the workflow a typed client could not run before the contract changed:
// `access-code:verify` answers 200 with a token or 202 with an approval, and a
// generated client declares a single success type, so the approval fields have
// to live on that type. This example is what proves they do.
//
// Usage:
//
//	QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=123456789 go run ./examples/approvalflow
//
// Precondition: the device must be online. An approval request is only created
// for a host the server currently sees, so a deleted or expired heartbeat makes
// `access-code:verify` answer 409 HOST_OFFLINE before the owner is ever asked.
package main

import (
	"context"
	"fmt"
	"os"

	signaling "github.com/ilh-github/go-remote-next/sdk/go"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	server := envOr("QD_SERVER", "http://127.0.0.1:8000")
	apiKey := envOr("QD_API_KEY", "qd-local-dev-key")
	identifier := os.Getenv("QD_IDENTIFIER")
	password := os.Getenv("QD_PASSWORD")
	deviceID := os.Getenv("QD_SMOKE_DEVICE")
	if identifier == "" || password == "" || deviceID == "" {
		return fmt.Errorf("QD_IDENTIFIER, QD_PASSWORD and QD_SMOKE_DEVICE are required")
	}

	cfg := signaling.NewConfiguration()
	cfg.Servers = signaling.ServerConfigurations{{URL: server}}
	client := signaling.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), signaling.ContextAPIKeys, map[string]signaling.APIKey{
		"ApiKeyAuth": {Key: apiKey},
	})

	session, _, err := client.AuthAPI.CreateSession(ctx).
		AuthSessionRequest(signaling.AuthSessionRequest{Identifier: identifier, Password: password}).
		Execute()
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	ctx = context.WithValue(ctx, signaling.ContextAccessToken, session.GetAccessToken())
	fmt.Println("login ok")

	// The device list carries the plaintext access code, which is what a
	// controller would have been told out of band.
	devices, _, err := client.DevicesAPI.ListMyDevices(ctx).Execute()
	if err != nil {
		return fmt.Errorf("list devices failed: %w", err)
	}
	accessCode := ""
	for _, device := range devices.GetItems() {
		if device.GetDeviceId() == deviceID {
			accessCode = device.GetAccessCode()
		}
	}
	if accessCode == "" {
		return fmt.Errorf("device %s has no access code in the device list", deviceID)
	}

	// The policy is a full replace, so the example has to put back exactly what
	// it found - including on the failure paths.
	before, _, err := client.DevicesAPI.GetAccessPolicy(ctx, deviceID).Execute()
	if err != nil {
		return fmt.Errorf("read policy failed: %w", err)
	}
	restore := func() {
		_, _, err := client.DevicesAPI.PutAccessPolicy(ctx, deviceID).
			DeviceAccessPolicyInput(signaling.DeviceAccessPolicyInput{
				RemoteDisabled:     before.RemoteDisabled,
				AllowIps:           before.AllowIps,
				MaxSessions:        before.MaxSessions,
				AllowedWindows:     before.AllowedWindows,
				Timezone:           before.Timezone,
				ExceptionDates:     before.ExceptionDates,
				ApprovalRequired:   before.ApprovalRequired,
				ApprovalMode:       before.ApprovalMode,
				ApprovalTrustedIps: before.ApprovalTrustedIps,
			}).
			Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cleanup: restore access policy failed: %v\n", err)
			return
		}
		fmt.Println("access policy restored")
	}
	defer restore()

	always := "always"
	required := true
	policy, _, err := client.DevicesAPI.PutAccessPolicy(ctx, deviceID).
		DeviceAccessPolicyInput(signaling.DeviceAccessPolicyInput{
			RemoteDisabled:     before.RemoteDisabled,
			AllowIps:           before.AllowIps,
			MaxSessions:        before.MaxSessions,
			AllowedWindows:     before.AllowedWindows,
			Timezone:           before.Timezone,
			ExceptionDates:     before.ExceptionDates,
			ApprovalRequired:   &required,
			ApprovalMode:       &always,
			ApprovalTrustedIps: before.ApprovalTrustedIps,
		}).
		Execute()
	if err != nil {
		return fmt.Errorf("require approval failed: %w", err)
	}
	if policy.GetApprovalMode() != always {
		return fmt.Errorf("policy says approval_mode=%q, want %q", policy.GetApprovalMode(), always)
	}
	fmt.Printf("device %s now requires owner approval\n", deviceID)

	// A unique client id per run: denying a request arms a per-source cooldown,
	// and a fixed id would make the next run look like the same denied source.
	clientID := fmt.Sprintf("sdk-approval-example-%d", os.Getpid())
	pending, _, err := client.DevicesAPI.VerifyAccessCode(ctx, deviceID).
		AccessCodeVerifyRequest(signaling.AccessCodeVerifyRequest{Code: accessCode, ClientId: &clientID}).
		Execute()
	if err != nil {
		return fmt.Errorf("verify access code failed: %w", err)
	}
	if pending.GetStatus() != "pending" {
		return fmt.Errorf("verify status=%q, want pending", pending.GetStatus())
	}
	approvalID := pending.GetApprovalId()
	approvalSecret := pending.GetApprovalSecret()
	if approvalID == 0 || approvalSecret == "" {
		return fmt.Errorf("the pending answer carried no approval_id/approval_secret")
	}
	fmt.Printf("controller got a pending approval %d (poll after %dms)\n", approvalID, pending.GetPollAfterMs())

	// Before the owner decides, the claim is "keep polling" rather than an
	// error - and that 202 body now decodes into the same type.
	early, _, err := client.DevicesAPI.ClaimAccessApproval(ctx, deviceID).
		ApprovalClaimRequest(signaling.ApprovalClaimRequest{ApprovalId: approvalID, Secret: approvalSecret}).
		Execute()
	if err != nil {
		return fmt.Errorf("early claim failed: %w", err)
	}
	if early.GetStatus() != "pending" {
		return fmt.Errorf("early claim status=%q, want pending", early.GetStatus())
	}
	fmt.Println("early claim answered pending, as documented")

	list, _, err := client.DevicesAPI.ListAccessApprovals(ctx, deviceID).Status("pending").Execute()
	if err != nil {
		return fmt.Errorf("list approvals failed: %w", err)
	}
	found := false
	for _, row := range list.GetItems() {
		if row.GetId() == approvalID {
			found = true
			if row.GetClientId() != clientID {
				return fmt.Errorf("approval %d shows client_id=%q, want %q", approvalID, row.GetClientId(), clientID)
			}
		}
	}
	if !found {
		return fmt.Errorf("approval %d is missing from the owner's pending list", approvalID)
	}
	fmt.Printf("owner sees approval %d among %d pending requests\n", approvalID, len(list.GetItems()))

	reason := "sdk approval example"
	approved, _, err := client.DevicesAPI.ApproveAccessApproval(ctx, deviceID, approvalID).
		ApproveAccessApprovalRequest(signaling.ApproveAccessApprovalRequest{Reason: &reason}).
		Execute()
	if err != nil {
		return fmt.Errorf("approve failed: %w", err)
	}
	if approved.GetStatus() != "approved" {
		return fmt.Errorf("approve returned status=%q, want approved", approved.GetStatus())
	}
	fmt.Printf("owner approved %d\n", approvalID)

	granted, _, err := client.DevicesAPI.ClaimAccessApproval(ctx, deviceID).
		ApprovalClaimRequest(signaling.ApprovalClaimRequest{ApprovalId: approvalID, Secret: approvalSecret}).
		Execute()
	if err != nil {
		return fmt.Errorf("claim after approval failed: %w", err)
	}
	if granted.GetStatus() != "granted" {
		return fmt.Errorf("claim status=%q, want granted", granted.GetStatus())
	}
	if granted.GetSignalToken() == "" {
		return fmt.Errorf("the granted claim carried no signal_token")
	}
	fmt.Printf("claimed the approval for a signal token (%d chars)\n", len(granted.GetSignalToken()))
	fmt.Println("PASS")
	return nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
