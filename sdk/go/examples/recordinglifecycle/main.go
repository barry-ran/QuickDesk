// Command recordinglifecycle drives one whole business workflow through the
// committed Go client, not just a login: create a recording row, upload the
// media, patch the row to completed, find it in the index, mint a playback
// ticket, read the bytes back through the ticket (the route a browser uses,
// which carries no Authorization header) and delete the row again.
//
// It exists because "the client compiles" and "the client can log in" are much
// weaker claims than "the client can run a workflow". Like the login example it
// is hand-written and excluded from the freshness check, and the CI smoke job
// runs it against a live server with a freshly seeded account.
//
// Usage:
//
//	QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=123456789 go run ./examples/recordinglifecycle
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
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

	deviceName := "sdk-recording-example"
	sessionID := fmt.Sprintf("sdk-example-%d", os.Getpid())
	created, _, err := client.RecordingsAPI.CreateSessionRecording(ctx).
		SessionRecordingInput(signaling.SessionRecordingInput{
			DeviceId:   deviceID,
			DeviceName: &deviceName,
			SessionId:  &sessionID,
			Status:     "recording",
		}).
		Execute()
	if err != nil {
		return fmt.Errorf("create recording failed: %w", err)
	}
	recordingID := int64(created.GetId())
	fmt.Printf("recording %d created (status=%s)\n", recordingID, created.GetStatus())

	// The workflow is a write path against a real account, so it cleans up after
	// itself whichever branch it leaves by. A failure here is reported but does
	// not mask the original error.
	defer func() {
		if _, err := client.RecordingsAPI.DeleteSessionRecording(ctx, recordingID).Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "cleanup: delete recording %d failed: %v\n", recordingID, err)
			return
		}
		fmt.Printf("recording %d deleted\n", recordingID)
	}()

	// A tiny but real MP4 header: the server allow-lists the extension rather
	// than sniffing the container, so this is enough to exercise the upload.
	media := []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'm', 'p', '4', '2'}
	file, err := os.CreateTemp("", "qd-example-*.mp4")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(media); err != nil {
		return fmt.Errorf("write media: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind media: %w", err)
	}

	uploaded, _, err := client.RecordingsAPI.UploadRecordingArtifact(ctx, recordingID).
		File(file).
		Execute()
	if err != nil {
		return fmt.Errorf("upload artifact failed: %w", err)
	}
	if uploaded.GetSizeBytes() != int64(len(media)) {
		return fmt.Errorf("uploaded size_bytes=%d, want %d", uploaded.GetSizeBytes(), len(media))
	}
	if uploaded.GetArtifactUrl() == "" {
		return fmt.Errorf("upload returned no artifact_url")
	}
	fmt.Printf("artifact uploaded (%d bytes, format=%s)\n", uploaded.GetSizeBytes(), uploaded.GetFormat())

	completed := "completed"
	duration := int32(3)
	patched, _, err := client.RecordingsAPI.UpdateSessionRecording(ctx, recordingID).
		SessionRecordingPatch(signaling.SessionRecordingPatch{
			Status:          &completed,
			DurationSeconds: &duration,
		}).
		Execute()
	if err != nil {
		return fmt.Errorf("patch recording failed: %w", err)
	}
	if patched.GetStatus() != completed || patched.GetDurationSeconds() != duration {
		return fmt.Errorf("patch did not apply: status=%s duration=%d", patched.GetStatus(), patched.GetDurationSeconds())
	}
	fmt.Printf("recording marked %s (%ds)\n", patched.GetStatus(), patched.GetDurationSeconds())

	limit := int32(50)
	list, _, err := client.RecordingsAPI.ListSessionRecordings(ctx).Limit(limit).Execute()
	if err != nil {
		return fmt.Errorf("list recordings failed: %w", err)
	}
	found := false
	for _, row := range list.GetItems() {
		if int64(row.GetId()) == recordingID {
			found = true
			if row.GetStatus() != completed {
				return fmt.Errorf("index shows status=%s, want %s", row.GetStatus(), completed)
			}
		}
	}
	if !found {
		return fmt.Errorf("recording %d is missing from the index", recordingID)
	}
	fmt.Printf("recording found in the index among %d rows\n", len(list.GetItems()))

	ticket, _, err := client.RecordingsAPI.CreateRecordingArtifactTicket(ctx, recordingID).Execute()
	if err != nil {
		return fmt.Errorf("create playback ticket failed: %w", err)
	}
	parsed, err := url.Parse(ticket.GetUrl())
	if err != nil {
		return fmt.Errorf("ticket url %q: %w", ticket.GetUrl(), err)
	}
	token := parsed.Query().Get("t")
	if token == "" {
		return fmt.Errorf("ticket url %q carries no t= parameter", ticket.GetUrl())
	}

	// The ticket route is the one a <video> tag can reach: no Authorization
	// header, the signed query string is the whole credential.
	downloaded, _, err := client.RecordingsAPI.DownloadRecordingArtifactByTicket(ctx, recordingID).T(token).Execute()
	if err != nil {
		return fmt.Errorf("download through the ticket failed: %w", err)
	}
	defer os.Remove(downloaded.Name())
	defer downloaded.Close()
	got, err := io.ReadAll(downloaded)
	if err != nil {
		return fmt.Errorf("read downloaded artifact: %w", err)
	}
	if !bytes.Equal(got, media) {
		return fmt.Errorf("downloaded %d bytes %v, want %d bytes %v", len(got), got, len(media), media)
	}
	fmt.Printf("artifact read back through the ticket (%d bytes, identical)\n", len(got))
	fmt.Println("PASS")
	return nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
