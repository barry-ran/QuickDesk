package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// The resumable upload protocol for update artifacts.
//
// An installer is routinely hundreds of megabytes, and the single-shot POST
// has to start over from zero whenever the connection drops - which is exactly
// what happens at the size where it hurts. This PATCH path is the recording
// artifact protocol (see session_recording_resume.go for the header semantics;
// both routes share parseArtifactUploadHeaders and planArtifactChunk): the
// client sends one chunk at a time, declares the offset it starts at, and reads
// the offset the server actually holds from the Upload-Offset response header.
//
// One difference is worth naming: a recording's staged file is keyed by the
// recording id and its extension, while an update's is keyed by the release id
// and the uploaded file name, because that name is what the download route
// serves back. Changing the name mid-upload would silently abandon the staged
// bytes, so the route refuses the change and names the staged file instead.

// AppendArtifact handles PATCH /v1/admin/update-releases/:id/artifact: one chunk
// of a resumable installer upload.
func (h *AdminUpdateReleaseHandler) AppendArtifact(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid release id")
		return
	}
	if strings.TrimSpace(h.store.Dir) == "" {
		ProblemInternal(c, "update artifact storage is not configured (set UPDATE_ARTIFACT_DIR)")
		return
	}
	release, err := h.releases.Get(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Update release not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}

	name, err := parseUpdateArtifactName(c.Query("filename"))
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
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

	dir := filepath.Join(h.store.Dir, strconv.FormatUint(uint64(id), 10))
	dest := filepath.Join(dir, name)
	tmp := dest + ".part"
	if err := os.MkdirAll(dir, 0o750); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	staged, err := artifactStagedBytes(tmp)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if staged == 0 && req.Offset == 0 {
		if other := otherUpdateArtifactTemp(dir, filepath.Base(tmp)); other != "" {
			ProblemBadRequest(c, ProblemCodeInvalidRequest,
				"an upload for this release is already staged as "+other+"; resume with that file name")
			return
		}
	}

	// Refuse a chunk that starts somewhere else before reading its body: the
	// client gets the authoritative offset in the header and can resync without
	// uploading bytes it would have to throw away.
	if req.Offset != staged {
		c.Header("Upload-Offset", strconv.FormatInt(staged, 10))
		ProblemConflict(c, ProblemCodeConflict, errArtifactOffsetMismatch.Error())
		return
	}

	// Bound the write by whichever limit is closer, so a client cannot make the
	// server read an unbounded body just because the cap is generous.
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
		if errors.Is(err, errArtifactOffsetMismatch) {
			ProblemConflict(c, ProblemCodeConflict, err.Error())
			return
		}
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}

	newSize, err := artifactStagedBytes(tmp)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.Header("Upload-Offset", strconv.FormatInt(newSize, 10))
	if !finish {
		c.JSON(http.StatusAccepted, release)
		return
	}

	finalized, err := h.finalizeResumableArtifact(c, release, tmp, dest, newSize)
	if err != nil {
		if errors.Is(err, errUpdateArtifactDigestMismatch) || errors.Is(err, errUpdateArtifactUnreadable) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, finalized)
}

// errUpdateArtifactDigestMismatch is the resumable twin of the single-shot
// digest check: the staged bytes are not the artifact the release advertises.
var errUpdateArtifactDigestMismatch = errors.New("the staged upload does not match the published sha256")

// errUpdateArtifactUnreadable marks a staging file the server cannot hash, which
// is a client-visible failure because the fix is to upload again.
var errUpdateArtifactUnreadable = errors.New("staged artifact could not be read back")

// finalizeResumableArtifact hashes the staged file, refuses it unless it is the
// artifact the release promises, and only then promotes it into place.
//
// The single-shot POST can hash while it writes; a resumable upload spans
// requests, so the digest is only knowable once the last chunk lands. That is
// why a mismatch discards the staged bytes rather than keeping them: the channel
// must not host a file that contradicts the sha256 its clients verify against,
// and a rejected multi-hundred-megabyte payload is not worth keeping around.
func (h *AdminUpdateReleaseHandler) finalizeResumableArtifact(
	c *gin.Context,
	release *models.UpdateRelease,
	tmp string,
	dest string,
	size int64,
) (*models.UpdateRelease, error) {
	actual, err := hashFile(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return nil, errUpdateArtifactUnreadable
	}
	if want := strings.ToLower(strings.TrimSpace(release.SHA256)); want != "" && want != actual {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("%w: staged artifact hashes to %s, but the release advertises %s",
			errUpdateArtifactDigestMismatch, actual, want)
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	updated, err := h.releases.SetArtifact(
		c.Request.Context(), release.ID, dest, actual, h.artifactDownloadURL(c, release.ID), size)
	if err != nil {
		_ = os.Remove(dest)
		return nil, err
	}
	if h.audit != nil {
		h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "",
			"update_release.artifact.upload", "update_release", strconv.FormatUint(uint64(release.ID), 10),
			actual+"/"+strconv.FormatInt(size, 10)+" bytes (resumable)", c.ClientIP())
	}
	return updated, nil
}

// parseUpdateArtifactName validates the ?filename= a resumable upload declares.
// The name is what the download route sends back, so it is required and it has
// to survive the same sanitiser the single-shot multipart path uses.
func parseUpdateArtifactName(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("resumable uploads need ?filename= (the name the download should send back)")
	}
	safe := sanitizeUpdateArtifactName(raw)
	if safe == "" {
		return "", errors.New("filename has no usable characters")
	}
	return safe, nil
}

// otherUpdateArtifactTemp finds a staging file for the same release under a
// different name. Uploading under a new name would otherwise start a second
// temporary and leave the first one behind forever.
func otherUpdateArtifactTemp(dir string, ownName string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || name == ownName || !strings.HasSuffix(name, ".part") {
			continue
		}
		return name
	}
	return ""
}

// hashFile returns the lowercase hex SHA-256 of a staged upload.
func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
