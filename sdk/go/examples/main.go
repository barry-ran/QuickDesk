// Command examples drives the committed Go client against a running signaling
// server: it signs in with the seeded account and reads back the device list.
//
// This is the one hand-written file in this module - the freshness check
// excludes this directory for exactly that reason - and it earns its place by
// being compiled (`go vet ./...`) and, in the CI smoke job, actually run against
// a live server. A client that compiles but cannot talk to the server would
// otherwise pass every gate.
//
// Usage:
//
//	QD_IDENTIFIER=user QD_PASSWORD=secret go run ./examples
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
	if identifier == "" || password == "" {
		return fmt.Errorf("QD_IDENTIFIER and QD_PASSWORD are required")
	}

	cfg := signaling.NewConfiguration()
	cfg.Servers = signaling.ServerConfigurations{{URL: server}}
	client := signaling.NewAPIClient(cfg)

	// A self-hosted deployment that sets ALLOWED_ORIGINS requires the API key on
	// every request; sending it unconditionally is harmless on one that does not.
	ctx := context.WithValue(context.Background(), signaling.ContextAPIKeys, map[string]signaling.APIKey{
		"ApiKeyAuth": {Key: apiKey},
	})

	session, _, err := client.AuthAPI.CreateSession(ctx).
		AuthSessionRequest(signaling.AuthSessionRequest{Identifier: identifier, Password: password}).
		Execute()
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	token := session.GetAccessToken()
	if token == "" {
		return fmt.Errorf("login returned no access token")
	}
	fmt.Printf("login ok (token length %d)\n", len(token))

	ctx = context.WithValue(ctx, signaling.ContextAccessToken, token)
	devices, _, err := client.DevicesAPI.ListMyDevices(ctx).Execute()
	if err != nil {
		return fmt.Errorf("listing devices failed: %w", err)
	}
	items := devices.GetItems()
	fmt.Printf("devices: %d\n", len(items))
	for _, device := range items {
		fmt.Printf("  %s %q online=%v\n", device.GetDeviceId(), device.GetDeviceName(), device.GetOnline())
	}
	return nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
