// Command resumableupload drives the chunked upload half of the recording
// contract through the committed Go client: stage two chunks with PATCH, prove
// an empty body with a valid offset is a status probe, prove a wrong offset is
// refused with the authoritative offset in the response header, let the server
// finalize on the declared total length, then read the bytes back through a
// playback ticket and delete the row.
//
// It exists because "the recording lifecycle works" only exercised the
// single-shot POST. The resumable path has different failure modes - offset
// resynchronisation, overshoot rejection, `.part` staging - and the generated
// signature takes an *os.File, so a caller has to know to stage chunks on disk.
// Like the other examples it is hand-written, excluded from the freshness
// check, and run by the CI smoke job against a live server.
//
// Usage:
//
//	QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=123456789 go run ./examples/resumableupload
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	signaling "github.com/ilh-github/go-remote-next/sdk/go"
)

const totalSize = 1 << 20 // 1 MiB

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

	deviceName := "sdk-resumable-example"
	sessionID := fmt.Sprintf("sdk-resumable-%d", os.Getpid())
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
	fmt.Printf("recording %d created for a %d-byte upload\n", recordingID, totalSize)

	defer func() {
		if _, err := client.RecordingsAPI.DeleteSessionRecording(ctx, recordingID).Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "cleanup: delete recording %d failed: %v\n", recordingID, err)
			return
		}
		fmt.Printf("recording %d deleted\n", recordingID)
	}()

	// Deterministic bytes, so a mismatch is a real content mismatch rather than
	// a random coincidence.
	media := make([]byte, totalSize)
	for i := range media {
		media[i] = byte(i*31 + 7)
	}
	length := strconv.Itoa(totalSize)

	// Chunk 1: bytes 0..400000. The declared total lets the server finalize by
	// itself when the last byte lands, which this example relies on later.
	offset, err := appendChunk(ctx, client, recordingID, 0, media[:400000], length, false)
	if err != nil {
		return err
	}
	if offset != 400000 {
		return fmt.Errorf("first chunk left offset=%d, want 400000", offset)
	}
	fmt.Printf("chunk 1 staged, server reports offset %d\n", offset)

	// The probe: a valid offset plus `Upload-Probe: 1`, which tells the server
	// to ignore the body and stage nothing, so a crashed client learns where it
	// stopped without a second endpoint. The body is one throwaway byte because
	// the generated client refuses to send a zero-length body at all.
	offset, err = appendChunk(ctx, client, recordingID, offset, []byte{0}, length, true)
	if err != nil {
		return err
	}
	if offset != 400000 {
		return fmt.Errorf("probe moved the offset to %d, want 400000", offset)
	}
	fmt.Printf("probe (Upload-Probe: 1) answered offset %d without staging\n", offset)

	// A wrong offset must be a 409 that carries the authoritative offset, so a
	// client can resync instead of restarting.
	conflictChunk, err := stageChunk(media[400000:800000])
	if err != nil {
		return fmt.Errorf("stage conflict chunk: %w", err)
	}
	defer os.Remove(conflictChunk.Name())
	defer conflictChunk.Close()
	wrong, conflictResp, err := client.RecordingsAPI.AppendRecordingArtifact(ctx, recordingID).
		Ext(".mp4").
		UploadOffset(12345).
		UploadLength(length).
		Body(conflictChunk).
		Execute()
	if err == nil {
		return fmt.Errorf("a chunk with a wrong offset was accepted (status=%s)", wrong.GetStatus())
	}
	if conflictResp == nil || conflictResp.StatusCode != http.StatusConflict {
		return fmt.Errorf("wrong offset answered %v (status=%v), want 409", err, conflictResp)
	}
	conflictOffset, err := parseOffset(conflictResp.Header.Get("Upload-Offset"))
	if err != nil {
		return fmt.Errorf("409 did not carry a usable Upload-Offset header: %w", err)
	}
	if conflictOffset != 400000 {
		return fmt.Errorf("409 reported offset=%d, want 400000", conflictOffset)
	}
	fmt.Printf("wrong offset refused with 409 and the authoritative offset %d\n", conflictOffset)

	// Chunk 2: bytes 400000..800000, from the offset the server reported.
	offset, err = appendChunk(ctx, client, recordingID, offset, media[400000:800000], length, false)
	if err != nil {
		return err
	}
	if offset != 800000 {
		return fmt.Errorf("second chunk left offset=%d, want 800000", offset)
	}
	fmt.Printf("chunk 2 staged, server reports offset %d\n", offset)

	// Final chunk: the last byte lands at the declared total, so the server
	// finalizes by itself and answers 200 with the updated row.
	final, err := stageChunk(media[800000:])
	if err != nil {
		return fmt.Errorf("stage final chunk: %w", err)
	}
	defer os.Remove(final.Name())
	defer final.Close()
	row, finalResp, err := client.RecordingsAPI.AppendRecordingArtifact(ctx, recordingID).
		Ext(".mp4").
		UploadOffset(offset).
		UploadLength(length).
		Body(final).
		Execute()
	if err != nil {
		return fmt.Errorf("final chunk failed: %w", err)
	}
	if finalResp.StatusCode != http.StatusOK {
		return fmt.Errorf("final chunk answered %d, want 200", finalResp.StatusCode)
	}
	if row.GetSizeBytes() != totalSize {
		return fmt.Errorf("finalized size_bytes=%d, want %d", row.GetSizeBytes(), totalSize)
	}
	fmt.Printf("final chunk finalized the upload (status=%s, size=%d)\n", row.GetStatus(), row.GetSizeBytes())

	// The bytes must survive the staged-file rename intact.
	ticket, _, err := client.RecordingsAPI.CreateRecordingArtifactTicket(ctx, recordingID).Execute()
	if err != nil {
		return fmt.Errorf("create playback ticket failed: %w", err)
	}
	parsed, err := url.Parse(ticket.GetUrl())
	if err != nil {
		return fmt.Errorf("ticket url %q: %w", ticket.GetUrl(), err)
	}
	downloaded, _, err := client.RecordingsAPI.DownloadRecordingArtifactByTicket(ctx, recordingID).
		T(parsed.Query().Get("t")).
		Execute()
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
		return fmt.Errorf("downloaded %d bytes, want %d with identical content", len(got), len(media))
	}
	fmt.Printf("artifact read back through the ticket (%d bytes, identical)\n", len(got))
	fmt.Println("PASS")
	return nil
}

// appendChunk PATCHes one chunk and returns the offset the server reports in
// the Upload-Offset response header. With probe set, data must still be
// non-empty because the generated client refuses an empty body; the server
// ignores those bytes and stages nothing.
func appendChunk(ctx context.Context, client *signaling.APIClient, recordingID, offset int64, data []byte, length string, probe bool) (int64, error) {
	req := client.RecordingsAPI.AppendRecordingArtifact(ctx, recordingID).
		Ext(".mp4").
		UploadOffset(offset).
		UploadLength(length)
	if probe {
		req = req.UploadProbe("1")
	}
	chunk, err := stageChunk(data)
	if err != nil {
		return 0, fmt.Errorf("stage chunk at offset %d: %w", offset, err)
	}
	defer os.Remove(chunk.Name())
	defer chunk.Close()
	req = req.Body(chunk)
	_, resp, err := req.Execute()
	if err != nil {
		// The problem body names the header at fault, which the generated
		// error string alone does not.
		detail := ""
		var apiErr *signaling.GenericOpenAPIError
		if errors.As(err, &apiErr) {
			detail = " " + string(apiErr.Body())
		}
		return 0, fmt.Errorf("append at offset %d failed: %w%s", offset, err, detail)
	}
	if resp.StatusCode != http.StatusAccepted {
		return 0, fmt.Errorf("append at offset %d answered %d, want 202", offset, resp.StatusCode)
	}
	staged, err := parseOffset(resp.Header.Get("Upload-Offset"))
	if err != nil {
		return 0, fmt.Errorf("append at offset %d returned no usable Upload-Offset: %w", offset, err)
	}
	return staged, nil
}

// stageChunk writes bytes to disk because the generated signature takes an
// *os.File, not an io.Reader or a []byte.
func stageChunk(data []byte) (*os.File, error) {
	file, err := os.CreateTemp("", "qd-resumable-*.part")
	if err != nil {
		return nil, err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		os.Remove(file.Name())
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		os.Remove(file.Name())
		return nil, err
	}
	return file, nil
}

func parseOffset(raw string) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("empty Upload-Offset header")
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Upload-Offset %q: %w", raw, err)
	}
	return value, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
