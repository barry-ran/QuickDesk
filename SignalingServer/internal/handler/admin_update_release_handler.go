package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminUpdateReleaseHandler serves the publishing half of the self-hosted
// update channel:
//
//	GET    /v1/admin/update-releases
//	POST   /v1/admin/update-releases
//	DELETE /v1/admin/update-releases/:id
//	POST   /v1/admin/update-releases/:id/artifact
//	PATCH  /v1/admin/update-releases/:id/artifact
//
// This is the index of which version exists for which channel/platform/arch and
// what it hashes to. The bytes normally stay wherever the operator hosts them;
// UploadArtifact (one request) and AppendArtifact (one chunk per request) are
// the optional paths that let this server hold them instead.
type AdminUpdateReleaseHandler struct {
	releases *service.UpdateReleaseService
	audit    *service.AuditService
	store    UpdateArtifactStore
}

// UpdateArtifactStore describes where re-hosted installers land and how large
// one upload may be. Like the recording store, the directory is never mounted as
// a static site - bytes only leave through the checked download route.
type UpdateArtifactStore struct {
	Dir      string
	MaxBytes int64
	// PublicBaseURL is the absolute prefix used to build the client-facing
	// download_url. Empty falls back to the scheme and Host of the upload
	// request, which is only right for a deployment reached at one name with
	// TLS termination in front of it; set UPDATE_PUBLIC_BASE_URL otherwise.
	PublicBaseURL string
}

func NewAdminUpdateReleaseHandler(
	releases *service.UpdateReleaseService,
	audit *service.AuditService,
	store UpdateArtifactStore,
) *AdminUpdateReleaseHandler {
	return &AdminUpdateReleaseHandler{releases: releases, audit: audit, store: store}
}

func (h *AdminUpdateReleaseHandler) List(c *gin.Context) {
	limit := 0
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "limit must be a non-negative number")
			return
		}
		limit = parsed
	}
	items, err := h.releases.List(c.Request.Context(), service.UpdateReleaseFilter{
		Channel:  strings.TrimSpace(c.Query("channel")),
		Platform: strings.TrimSpace(c.Query("platform")),
		Arch:     strings.TrimSpace(c.Query("arch")),
		Limit:    limit,
	})
	if err != nil {
		writeUpdateReleaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type updateReleasePublishReq struct {
	Channel     string     `json:"channel"`
	Platform    string     `json:"platform"`
	Arch        string     `json:"arch"`
	Version     string     `json:"version"`
	DownloadURL string     `json:"download_url"`
	SHA256      string     `json:"sha256"`
	SizeBytes   int64      `json:"size_bytes"`
	Notes       string     `json:"notes"`
	Mandatory   bool       `json:"mandatory"`
	PublishedAt *time.Time `json:"published_at"`
}

func (h *AdminUpdateReleaseHandler) Publish(c *gin.Context) {
	var req updateReleasePublishReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	input := service.PublishUpdateReleaseInput{
		Channel:     req.Channel,
		Platform:    req.Platform,
		Arch:        req.Arch,
		Version:     req.Version,
		DownloadURL: req.DownloadURL,
		SHA256:      req.SHA256,
		SizeBytes:   req.SizeBytes,
		Notes:       req.Notes,
		Mandatory:   req.Mandatory,
	}
	if req.PublishedAt != nil {
		input.PublishedAt = *req.PublishedAt
	}
	release, err := h.releases.Publish(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrUpdateReleaseExists) {
			ProblemConflict(c, ProblemCodeConflict, err.Error())
			return
		}
		writeUpdateReleaseError(c, err)
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "",
		"update_release.publish", "update_release", strconv.FormatUint(uint64(release.ID), 10),
		release.Channel+"/"+release.Platform+"/"+release.Arch+"/"+release.Version, c.ClientIP())
	c.JSON(http.StatusCreated, release)
}

func (h *AdminUpdateReleaseHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid release id")
		return
	}
	release, err := h.releases.Delete(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Update release not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	// Drop the stored installer along with the row. A failure here is not fatal:
	// the release is already gone, so a stray file costs disk, not correctness.
	if release.ArtifactPath != "" {
		removeStoredArtifact(release.ArtifactPath)
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "",
		"update_release.delete", "update_release", c.Param("id"), "", c.ClientIP())
	c.Status(http.StatusNoContent)
}

// UploadArtifact handles POST /v1/admin/update-releases/:id/artifact.
//
// This is the re-hosting half of the channel: the operator hands the installer
// to the server instead of pointing download_url at a static host, and the
// channel then serves it from the same origin as the API. The bytes must hash
// to the digest the release already advertises, so "the server now hosts this
// release" can never quietly mean "the server now hosts different bytes".
//
// Two shapes, mirroring the recording upload: multipart/form-data with a `file`
// part (what the admin UI sends), or the raw body with `?filename=` for a CLI.
func (h *AdminUpdateReleaseHandler) UploadArtifact(c *gin.Context) {
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
	if h.store.MaxBytes > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.store.MaxBytes)
	}

	var (
		src  io.ReadCloser
		name string
	)
	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "multipart field 'file' is required")
			return
		}
		name = fileHeader.Filename
		opened, err := fileHeader.Open()
		if err != nil {
			ProblemInternal(c, err.Error())
			return
		}
		defer opened.Close()
		src = opened
	} else {
		name = c.Query("filename")
		if strings.TrimSpace(name) == "" {
			ProblemBadRequest(c, ProblemCodeInvalidRequest,
				"raw uploads need ?filename= (or post multipart/form-data with a 'file' part)")
			return
		}
		src = c.Request.Body
	}

	safeName := sanitizeUpdateArtifactName(name)
	if safeName == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "filename has no usable characters")
		return
	}
	dest := filepath.Join(h.store.Dir, strconv.FormatUint(uint64(id), 10), safeName)
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	// Write beside the target and rename, so a dropped upload can never leave a
	// half-written installer under the name the download route serves.
	tmp := dest + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	digest := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(out, digest), src)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(tmp)
		if h.store.MaxBytes > 0 && written >= h.store.MaxBytes {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "artifact exceeds the configured upload limit")
			return
		}
		ProblemInternal(c, copyErr.Error())
		return
	}
	if closeErr != nil {
		os.Remove(tmp)
		ProblemInternal(c, closeErr.Error())
		return
	}
	if written <= 0 {
		os.Remove(tmp)
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "artifact is empty")
		return
	}
	actual := hex.EncodeToString(digest.Sum(nil))
	if want := strings.ToLower(strings.TrimSpace(release.SHA256)); want != "" && want != actual {
		os.Remove(tmp)
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			"uploaded artifact hashes to "+actual+", but the release advertises "+want)
		return
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		ProblemInternal(c, err.Error())
		return
	}
	updated, err := h.releases.SetArtifact(
		c.Request.Context(), uint(id), dest, actual, h.artifactDownloadURL(c, uint(id)), written)
	if err != nil {
		os.Remove(dest)
		ProblemInternal(c, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "",
		"update_release.artifact.upload", "update_release", c.Param("id"),
		actual+"/"+strconv.FormatInt(written, 10)+" bytes", c.ClientIP())
	c.JSON(http.StatusOK, updated)
}

// artifactDownloadURL is the client-facing URL for a re-hosted installer. A
// self-hosted deployment normally sits behind one name, so the request's own
// host is a good default; a deployment that terminates TLS elsewhere sets
// UPDATE_PUBLIC_BASE_URL and the stored URL is absolute from the start.
func (h *AdminUpdateReleaseHandler) artifactDownloadURL(c *gin.Context, id uint) string {
	route := "/v1/update-releases/" + strconv.FormatUint(uint64(id), 10) + "/artifact"
	if base := strings.TrimRight(strings.TrimSpace(h.store.PublicBaseURL), "/"); base != "" {
		return base + route
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host + route
}

// sanitizeUpdateArtifactName keeps only the leaf of an uploaded name and strips
// separators and control characters, so a hostile filename cannot influence
// where the bytes land. It returns "" when nothing usable is left.
func sanitizeUpdateArtifactName(name string) string {
	leaf := filepath.Base(strings.TrimSpace(name))
	leaf = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '/' || r == '\\' {
			return -1
		}
		return r
	}, leaf)
	leaf = strings.TrimLeft(strings.TrimSpace(leaf), ".")
	if leaf == "" || leaf == "." || leaf == ".." {
		return ""
	}
	runes := []rune(leaf)
	if len(runes) > 120 {
		ext := filepath.Ext(leaf)
		keep := 120 - len([]rune(ext))
		if keep < 1 {
			keep = 1
		}
		leaf = string(runes[:keep]) + ext
	}
	return leaf
}

// removeStoredArtifact deletes a re-hosted installer, any staging file left by
// an interrupted resumable upload, and the per-release directory that held them.
// Best effort: the row is already gone.
func removeStoredArtifact(path string) {
	dir := filepath.Dir(path)
	_ = os.Remove(path)
	_ = os.Remove(path + ".part")
	if entries, err := os.ReadDir(dir); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".part") {
				continue
			}
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	_ = os.Remove(dir)
}
