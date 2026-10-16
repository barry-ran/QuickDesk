package handler

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// The resumable upload protocol for recording artifacts.
//
// A control client on a train does not finish a 200 MB capture in one request
// body, and when the connection drops the browser-side recorder has already
// lost the bytes it sent. The single-shot POST therefore keeps the happy path,
// and this PATCH path adds the part that survives a failure: the client sends
// one chunk at a time, tells the server where the chunk starts, and gets back
// the offset the server actually holds. A retry after a dropped response
// resumes from that number instead of from zero.
//
// The protocol is deliberately the smallest thing that supports that - three
// headers and a status code - rather than a full implementation of tus:
//
//   - `Upload-Offset` (required): the byte offset this chunk starts at. It must
//     equal what is already staged, otherwise the answer is `409` with the
//     authoritative offset in the same header, and the client resyncs.
//   - `Upload-Length` (optional): the total size, `*` when the client does not
//     know it yet. Declaring it lets the server finish the upload by itself
//     when the last byte lands, and lets it refuse a chunk that would overshoot.
//   - `Upload-Complete: 1` (optional): "this chunk ends the upload". Needed when
//     the client never declared a total.
//
// An empty body with a valid offset is a status probe: it writes nothing and
// answers `202` with the current offset, which is how a client that crashed
// re-learns where it stopped without a second endpoint.

var (
	errArtifactOffsetMismatch   = errors.New("Upload-Offset does not match the bytes already staged")
	errArtifactLengthExceeded   = errors.New("chunk would exceed the declared upload length")
	errArtifactUploadTooLarge   = errors.New("artifact exceeds the configured upload limit")
	errArtifactUploadIncomplete = errors.New("Upload-Complete arrived before the declared upload length")
)

// artifactUploadRequest is the resume state one chunk declares.
type artifactUploadRequest struct {
	// Offset is where the client believes the upload stands.
	Offset int64
	// Length is the total size the client declared, or 0 when it declared none.
	Length int64
	// Complete is the client's promise that this chunk ends the upload.
	Complete bool
	// Probe asks for the current offset without staging bytes. A generated
	// client cannot send a zero-length body, so this is the reachable form of
	// the status probe.
	Probe bool
}

// parseArtifactUploadHeaders reads the resume headers. The messages are
// what the client sees in the 400 problem body, so they name the header at
// fault instead of saying "invalid request".
func parseArtifactUploadHeaders(offsetRaw, lengthRaw, completeRaw, probeRaw string) (artifactUploadRequest, error) {
	var out artifactUploadRequest

	offsetRaw = strings.TrimSpace(offsetRaw)
	if offsetRaw == "" {
		return out, errors.New("Upload-Offset header is required")
	}
	offset, err := strconv.ParseInt(offsetRaw, 10, 64)
	if err != nil || offset < 0 {
		return out, errors.New("Upload-Offset must be a non-negative integer")
	}
	out.Offset = offset

	switch strings.TrimSpace(lengthRaw) {
	case "", "*":
		// No total declared: the client has to send Upload-Complete on the
		// chunk that ends the upload, because the server cannot infer it.
	default:
		length, err := strconv.ParseInt(strings.TrimSpace(lengthRaw), 10, 64)
		if err != nil || length <= 0 {
			return out, errors.New("Upload-Length must be a positive integer, or * when unknown")
		}
		if length < offset {
			return out, errors.New("Upload-Length cannot be smaller than Upload-Offset")
		}
		out.Length = length
	}

	switch strings.ToLower(strings.TrimSpace(completeRaw)) {
	case "", "0", "false", "no":
	case "1", "true", "yes":
		out.Complete = true
	default:
		return out, errors.New("Upload-Complete must be 1 or 0")
	}

	switch strings.ToLower(strings.TrimSpace(probeRaw)) {
	case "", "0", "false", "no":
	case "1", "true", "yes":
		out.Probe = true
	default:
		return out, errors.New("Upload-Probe must be 1 or 0")
	}
	return out, nil
}

// planArtifactChunk decides what one chunk does, given how many bytes are
// already staged: write it, finish the upload, or both. Pure arithmetic, so the
// boundaries are pinned by tests rather than by a running server.
func planArtifactChunk(req artifactUploadRequest, staged, chunk, maxBytes int64) (write bool, finish bool, err error) {
	if chunk < 0 {
		chunk = 0
	}
	if req.Offset != staged {
		return false, false, errArtifactOffsetMismatch
	}
	total := staged + chunk
	if req.Length > 0 && total > req.Length {
		return false, false, errArtifactLengthExceeded
	}
	if maxBytes > 0 && total > maxBytes {
		return false, false, errArtifactUploadTooLarge
	}
	finish = req.Complete || (req.Length > 0 && total == req.Length)
	if !finish {
		return chunk > 0, false, nil
	}
	if total <= 0 {
		// An empty artifact is refused on every path; "finish now" with nothing
		// written is a client that lost its body, not a zero-byte recording.
		return false, false, service.ErrArtifactEmpty
	}
	if req.Length > 0 && total != req.Length {
		return false, false, errArtifactUploadIncomplete
	}
	return chunk > 0, true, nil
}

// artifactStagedBytes reports how much of this upload is already on disk. A
// missing temporary means zero: the client abandoned the upload, or this is the
// first chunk.
func artifactStagedBytes(tmp string) (int64, error) {
	info, err := os.Stat(tmp)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if info.IsDir() {
		return 0, errors.New("upload staging path is a directory")
	}
	return info.Size(), nil
}

// otherArtifactTemp looks for a staging file for the same recording under a
// different extension. Changing the extension mid-upload would otherwise start
// a second temporary and leave the first one behind forever.
func otherArtifactTemp(dir string, ownName string, recordingID uint) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	prefix := "rec-" + strconv.FormatUint(uint64(recordingID), 10) + "."
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || name == ownName || !strings.HasPrefix(name, prefix) {
			continue
		}
		if strings.HasSuffix(name, ".part") {
			return name
		}
	}
	return ""
}

// AppendArtifact handles PATCH /v1/me/recordings/:id/artifact: one chunk of a
// resumable upload.
func (h *SessionRecordingHandler) AppendArtifact(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	// Ownership before bytes, exactly like the single-shot path. The row is kept
	// so the 202 answer can return it: the resumable route uses one success
	// schema (SessionRecording) for both statuses, and the staged offset travels
	// in the Upload-Offset response header.
	row, err := h.recordings.Get(c.Request.Context(), uid, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}

	ext, err := service.NormalizeArtifactExtension(c.Query("ext"))
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			"resumable uploads need ?ext=.mp4|.mkv|.webm|.h264|.grf")
		return
	}
	req, err := parseArtifactUploadHeaders(
		c.GetHeader("Upload-Offset"),
		c.GetHeader("Upload-Length"),
		c.GetHeader("Upload-Complete"),
		c.GetHeader("Upload-Probe"),
	)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}

	dest := service.ArtifactPath(h.store.Dir, uid, uint(id), ext)
	tmp := dest + ".part"
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	staged, err := artifactStagedBytes(tmp)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if staged == 0 && req.Offset == 0 {
		if other := otherArtifactTemp(filepath.Dir(dest), filepath.Base(tmp), uint(id)); other != "" {
			ProblemBadRequest(c, ProblemCodeInvalidRequest,
				"an upload for this recording is already staged as "+other+"; resume with that extension")
			return
		}
	}

	// Refuse a chunk that starts somewhere else before reading its body: the
	// client gets the authoritative offset in the header and can resync without
	// uploading a single byte it would have to throw away.
	if req.Offset != staged {
		c.Header("Upload-Offset", strconv.FormatInt(staged, 10))
		ProblemConflict(c, ProblemCodeConflict, errArtifactOffsetMismatch.Error())
		return
	}

	// Bound the write by whichever limit is closer, so a client cannot make the
	// server read an unbounded body just because the file cap is generous.
	allowance := int64(-1)
	if h.store.MaxBytes > 0 {
		allowance = h.store.MaxBytes - staged
	}
	if req.Length > 0 {
		remaining := req.Length - staged
		if allowance < 0 || remaining < allowance {
			allowance = remaining
		}
	}
	if allowance == 0 {
		// Nothing may be appended: any byte means the declared length or the
		// cap is already reached.
		c.Header("Upload-Offset", strconv.FormatInt(staged, 10))
		if req.Length > 0 && staged >= req.Length {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, errArtifactLengthExceeded.Error())
			return
		}
		ProblemBadRequest(c, ProblemCodeInvalidRequest, errArtifactUploadTooLarge.Error())
		return
	}
	body := c.Request.Body
	if allowance > 0 {
		body = http.MaxBytesReader(c.Writer, body, allowance)
	}

	written := int64(0)
	// A probe never reads the body: the whole point is to ask where the upload
	// stands without staging bytes.
	if !req.Probe && c.Request.ContentLength != 0 {
		out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
		if err != nil {
			ProblemInternal(c, err.Error())
			return
		}
		written, err = io.Copy(out, body)
		closeErr := out.Close()
		if err != nil || closeErr != nil {
			// Roll the staging file back to the last confirmed offset, so a
			// truncated body leaves a resumable upload rather than a corrupt
			// prefix with a length nobody knows.
			_ = os.Truncate(tmp, staged)
			if err != nil && isBodyTooLarge(err) {
				c.Header("Upload-Offset", strconv.FormatInt(staged, 10))
				if req.Length > 0 && staged+written >= req.Length {
					ProblemBadRequest(c, ProblemCodeInvalidRequest, errArtifactLengthExceeded.Error())
					return
				}
				ProblemBadRequest(c, ProblemCodeInvalidRequest, errArtifactUploadTooLarge.Error())
				return
			}
			if err == nil {
				err = closeErr
			}
			ProblemInternal(c, err.Error())
			return
		}
	}

	_, finish, err := planArtifactChunk(req, staged, written, h.store.MaxBytes)
	if err != nil {
		_ = os.Truncate(tmp, staged)
		if written == 0 {
			_ = os.Remove(tmp)
		}
		c.Header("Upload-Offset", strconv.FormatInt(staged, 10))
		switch {
		case errors.Is(err, errArtifactOffsetMismatch):
			ProblemConflict(c, ProblemCodeConflict, err.Error())
		default:
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		}
		return
	}

	newSize, err := artifactStagedBytes(tmp)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.Header("Upload-Offset", strconv.FormatInt(newSize, 10))

	if !finish {
		c.JSON(http.StatusAccepted, row)
		return
	}

	finalized, err := h.finalizeArtifact(c, uid, uint(id), tmp, dest, ext, newSize)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, finalized)
}

// isBodyTooLarge spots the MaxBytesReader error without importing net/http's
// private type.
func isBodyTooLarge(err error) bool {
	var maxBytes *http.MaxBytesError
	return errors.As(err, &maxBytes) || strings.Contains(err.Error(), "request body too large")
}
