package handler

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/httpx"
	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SessionRecordingHandler serves the user-scoped recording index under
// /v1/me/recordings. Media files are produced by the control engine; this
// surface owns discovery, ownership, and lifecycle metadata.
type SessionRecordingHandler struct {
	recordings *service.SessionRecordingService
	bus        *service.EventBus
	store      SessionRecordingStore
	audit      *service.AuditService
}

// SessionRecordingStore describes where uploaded media files live and how
// large they may be. Files are written outside any statically served directory
// and only ever handed back through the ownership-checked routes below.
type SessionRecordingStore struct {
	Dir      string
	MaxBytes int64
	// URLSecret signs the short-lived playback tickets. Empty means tickets are
	// disabled and only the Bearer-authenticated routes can fetch media.
	URLSecret string
	// TicketTTL is how long a playback ticket stays valid.
	TicketTTL time.Duration
}

func NewSessionRecordingHandler(
	recordings *service.SessionRecordingService,
	bus *service.EventBus,
	store SessionRecordingStore,
	audit *service.AuditService,
) *SessionRecordingHandler {
	return &SessionRecordingHandler{recordings: recordings, bus: bus, store: store, audit: audit}
}

// recordingAuditWindow collapses the burst of requests one playback produces
// into a single audit row. A browser fetching a recording issues several range
// requests and re-renders thumbnails on every scroll; without a window the
// trail would record transport details instead of "this actor read this
// recording around this time".
const recordingAuditWindow = 5 * time.Minute

// auditMediaRead records a user reading recording media. The admin routes
// audit the same act under the same action names, so `recording.download`
// finds every download; the actor column says whether it was the owner or an
// administrator. A missing audit service is not an error - the media read is
// the point of the route, and the write is best-effort like every other audit
// call in this package.
func (h *SessionRecordingHandler) auditMediaRead(
	c *gin.Context,
	actorID uint,
	action string,
	recordingID uint,
	details string,
) {
	if h.audit == nil {
		return
	}
	h.audit.LogActorDeduped(
		c.Request.Context(),
		actorID,
		action,
		"session_recording",
		strconv.FormatUint(uint64(recordingID), 10),
		details,
		c.ClientIP(),
		recordingAuditWindow,
	)
}

type createSessionRecordingReq struct {
	DeviceID        string     `json:"device_id" binding:"required"`
	DeviceName      string     `json:"device_name"`
	SessionID       string     `json:"session_id"`
	Status          string     `json:"status" binding:"required"`
	Format          string     `json:"format"`
	ArtifactPath    string     `json:"artifact_path"`
	ArtifactURL     string     `json:"artifact_url"`
	ThumbnailPath   string     `json:"thumbnail_path"`
	DurationSeconds int        `json:"duration_seconds"`
	SizeBytes       int64      `json:"size_bytes"`
	StartedAt       *time.Time `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
}

type updateSessionRecordingReq struct {
	Status          *string    `json:"status"`
	Format          *string    `json:"format"`
	ArtifactPath    *string    `json:"artifact_path"`
	ArtifactURL     *string    `json:"artifact_url"`
	ThumbnailPath   *string    `json:"thumbnail_path"`
	DurationSeconds *int       `json:"duration_seconds"`
	SizeBytes       *int64     `json:"size_bytes"`
	EndedAt         *time.Time `json:"ended_at"`
}

func (h *SessionRecordingHandler) Create(c *gin.Context) {
	uid := middleware.MustUserID(c)
	var req createSessionRecordingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	var startedAt time.Time
	if req.StartedAt != nil {
		startedAt = *req.StartedAt
	}
	row, err := h.recordings.Create(c.Request.Context(), uid, service.CreateSessionRecordingInput{
		DeviceID:        req.DeviceID,
		DeviceName:      req.DeviceName,
		SessionID:       req.SessionID,
		Status:          req.Status,
		Format:          req.Format,
		ArtifactPath:    req.ArtifactPath,
		ArtifactURL:     req.ArtifactURL,
		ThumbnailPath:   req.ThumbnailPath,
		DurationSeconds: req.DurationSeconds,
		SizeBytes:       req.SizeBytes,
		StartedAt:       startedAt,
		EndedAt:         req.EndedAt,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidRecordingStatus) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if h.bus != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventRecordingCreated,
			UserID:   uid,
			DeviceID: row.DeviceID,
			Data: map[string]interface{}{
				"recording_id": row.ID,
				"session_id":   row.SessionID,
				"status":       row.Status,
				"format":       row.Format,
			},
		})
	}
	c.JSON(http.StatusCreated, row)
}

func (h *SessionRecordingHandler) List(c *gin.Context) {
	uid := middleware.MustUserID(c)
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	rows, err := h.recordings.List(c.Request.Context(), uid, service.SessionRecordingListParams{
		BeforeID: cur.OffsetID,
		Limit:    p.Limit + 1,
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	var next string
	if len(rows) > p.Limit {
		next = EncodeCursor(CursorPayload{OffsetID: rows[p.Limit-1].ID})
		rows = rows[:p.Limit]
	}
	c.JSON(http.StatusOK, NewCursorPage(rows, next, -1))
}

func (h *SessionRecordingHandler) Get(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	row, err := h.recordings.Get(c.Request.Context(), uid, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, row)
}

func (h *SessionRecordingHandler) Update(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	var req updateSessionRecordingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	row, err := h.recordings.Update(c.Request.Context(), uid, uint(id), service.UpdateSessionRecordingInput{
		Status:          req.Status,
		Format:          req.Format,
		ArtifactPath:    req.ArtifactPath,
		ArtifactURL:     req.ArtifactURL,
		ThumbnailPath:   req.ThumbnailPath,
		DurationSeconds: req.DurationSeconds,
		SizeBytes:       req.SizeBytes,
		EndedAt:         req.EndedAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
		case errors.Is(err, service.ErrInvalidRecordingStatus):
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		default:
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		}
		return
	}
	if h.bus != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventRecordingUpdated,
			UserID:   uid,
			DeviceID: row.DeviceID,
			Data: map[string]interface{}{
				"recording_id": row.ID,
				"session_id":   row.SessionID,
				"status":       row.Status,
			},
		})
	}
	c.JSON(http.StatusOK, row)
}

func (h *SessionRecordingHandler) Delete(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	if err := h.recordings.Delete(c.Request.Context(), uid, uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if h.bus != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:   service.EventRecordingDeleted,
			UserID: uid,
			// `ip` is the actor's observed address; the audit subscriber lifts
			// it onto the row so a deletion can be explained by who and where.
			Data: map[string]interface{}{"recording_id": uint(id), "ip": c.ClientIP()},
		})
	}
	c.Status(http.StatusNoContent)
}

// UploadArtifact handles POST /v1/me/recordings/:id/artifact.
//
// Two shapes are accepted because the two clients differ: browsers and Qt both
// can post multipart/form-data with a `file` part (the normal path), while a
// headless/CLI producer can stream the raw body with `?ext=.mp4`.
func (h *SessionRecordingHandler) UploadArtifact(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	// Ownership check first: never accept bytes for a row the caller does not own.
	if _, err := h.recordings.Get(c.Request.Context(), uid, uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if h.store.MaxBytes > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.store.MaxBytes)
	}

	var (
		src io.ReadCloser
		ext string
	)
	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "multipart field 'file' is required")
			return
		}
		ext, err = service.NormalizeArtifactExtension(fileHeader.Filename)
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		opened, err := fileHeader.Open()
		if err != nil {
			ProblemInternal(c, err.Error())
			return
		}
		defer opened.Close()
		src = opened
	} else {
		ext, err = service.NormalizeArtifactExtension(c.Query("ext"))
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest,
				"raw uploads need ?ext=.mp4|.mkv|.webm|.h264|.grf")
			return
		}
		src = c.Request.Body
	}

	dest := service.ArtifactPath(h.store.Dir, uid, uint(id), ext)
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	// Write next to the target and rename, so a dropped upload can never leave
	// a half-written file behind under the real name.
	tmp := dest + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	written, copyErr := io.Copy(out, src)
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
		ProblemBadRequest(c, ProblemCodeInvalidRequest, service.ErrArtifactEmpty.Error())
		return
	}

	row, err := h.finalizeArtifact(c, uid, uint(id), tmp, dest, ext, written)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, row)
}

// finalizeArtifact promotes a finished upload into place and records it on the
// row. Both the single-shot POST body and the resumable PATCH path end here, so
// the two shapes cannot disagree about what "the artifact is stored" means: one
// rename, one AttachArtifact call, one realtime event.
//
// A failure removes the temporary file, so a rejected finish cannot leave a
// half-written upload behind for the reaper to find later.
func (h *SessionRecordingHandler) finalizeArtifact(
	c *gin.Context,
	uid uint,
	id uint,
	tmp string,
	dest string,
	ext string,
	size int64,
) (*models.SessionRecording, error) {
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return nil, err
	}
	row, err := h.recordings.AttachArtifact(c.Request.Context(), uid, id, dest, ext, size)
	if err != nil {
		return nil, err
	}
	if h.bus != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventRecordingUpdated,
			UserID:   uid,
			DeviceID: row.DeviceID,
			Data: map[string]interface{}{
				"recording_id": row.ID,
				"session_id":   row.SessionID,
				"status":       row.Status,
				"size_bytes":   row.SizeBytes,
			},
		})
	}
	return row, nil
}

// thumbnailMaxBytes caps a preview image independently of the (much larger)
// artifact limit. A preview is a small still; an oversized one means the client
// sent the wrong file.
const thumbnailMaxBytes int64 = 2 << 20

// UploadThumbnail handles POST /v1/me/recordings/:id/thumbnail.
//
// It accepts the same two request shapes as UploadArtifact (multipart `file`,
// or a raw body with `?ext=`), but the bytes are sniffed before they are
// stored: the recordings list renders this image inline, so a file that only
// claims to be a JPEG must not be accepted on the strength of its name.
func (h *SessionRecordingHandler) UploadThumbnail(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	// Ownership check first: never accept bytes for a row the caller does not own.
	if _, err := h.recordings.Get(c.Request.Context(), uid, uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}

	limit := thumbnailMaxBytes
	if h.store.MaxBytes > 0 && h.store.MaxBytes < limit {
		limit = h.store.MaxBytes
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)

	var (
		src          io.ReadCloser
		declaredType string
	)
	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "multipart field 'file' is required")
			return
		}
		ext, err := service.NormalizeThumbnailExtension(fileHeader.Filename)
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "thumbnail must be .jpg, .jpeg or .png")
			return
		}
		declaredType = service.ThumbnailContentType(ext)
		opened, err := fileHeader.Open()
		if err != nil {
			ProblemInternal(c, err.Error())
			return
		}
		defer opened.Close()
		src = opened
	} else {
		ext, err := service.NormalizeThumbnailExtension(c.Query("ext"))
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "raw uploads need ?ext=.jpg|.jpeg|.png")
			return
		}
		declaredType = service.ThumbnailContentType(ext)
		src = c.Request.Body
	}

	head := make([]byte, 8)
	n, readErr := io.ReadFull(src, head)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		ProblemInternal(c, readErr.Error())
		return
	}
	head = head[:n]
	sniffed, err := service.SniffThumbnailType(head)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if sniffed != declaredType {
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			"thumbnail content does not match its declared type")
		return
	}
	// The stored extension comes from the bytes, so it always describes the file
	// that is actually on disk.
	ext := service.ThumbnailExtensionForType(sniffed)

	dest := service.ThumbnailPath(h.store.Dir, uid, uint(id), ext)
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	tmp := dest + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if _, err := out.Write(head); err != nil {
		out.Close()
		os.Remove(tmp)
		ProblemInternal(c, err.Error())
		return
	}
	rest, copyErr := io.Copy(out, src)
	written := int64(len(head)) + rest
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(tmp)
		if written >= limit {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "thumbnail exceeds the configured upload limit")
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
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		ProblemInternal(c, err.Error())
		return
	}
	// A previous preview may have used the other allowed extension; drop it so
	// the store keeps exactly one thumbnail per recording.
	for _, other := range []string{".jpg", ".png"} {
		if stale := service.ThumbnailPath(h.store.Dir, uid, uint(id), other); stale != dest {
			os.Remove(stale)
		}
	}

	row, err := h.recordings.AttachThumbnail(c.Request.Context(), uid, uint(id), dest, ext, written)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if h.bus != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventRecordingUpdated,
			UserID:   uid,
			DeviceID: row.DeviceID,
			Data: map[string]interface{}{
				"recording_id":  row.ID,
				"session_id":    row.SessionID,
				"thumbnail_url": row.ThumbnailURL,
			},
		})
	}
	c.JSON(http.StatusOK, row)
}

// DownloadThumbnail handles GET /v1/me/recordings/:id/thumbnail.
func (h *SessionRecordingHandler) DownloadThumbnail(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	row, err := h.recordings.ThumbnailFor(c.Request.Context(), uid, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording thumbnail not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	h.auditMediaRead(c, uid, "recording.thumbnail.download", uint(id), "")
	serveThumbnail(c, row)
}

// DownloadArtifact handles GET /v1/me/recordings/:id/artifact.
func (h *SessionRecordingHandler) DownloadArtifact(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	row, err := h.recordings.ArtifactFor(c.Request.Context(), uid, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording artifact not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	h.auditMediaRead(c, uid, "recording.download", uint(id), fmt.Sprintf("size_bytes=%d", row.SizeBytes))
	serveArtifact(c, row)
}

// DownloadArtifactAdmin handles GET /v1/admin/recordings/:id/artifact and
// writes an audit entry, matching the admin delete behaviour.
func (h *AdminSessionRecordingHandler) DownloadArtifactAdmin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	row, err := h.recordings.ArtifactForAdmin(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording artifact not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if h.audit != nil {
		h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "recording.download",
			"session_recording", c.Param("id"), fmt.Sprintf("size_bytes=%d", row.SizeBytes), c.ClientIP())
	}
	serveArtifact(c, row)
}

// DownloadThumbnailAdmin handles GET /v1/admin/recordings/:id/thumbnail. Like
// the artifact variant it writes an audit entry, because reading a user's screen
// capture is exactly the kind of access an admin log should be able to explain.
func (h *AdminSessionRecordingHandler) DownloadThumbnailAdmin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	row, err := h.recordings.ThumbnailForAdmin(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording thumbnail not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if h.audit != nil {
		h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "recording.thumbnail.download",
			"session_recording", c.Param("id"), "", c.ClientIP())
	}
	serveThumbnail(c, row)
}

func serveThumbnail(c *gin.Context, row *models.SessionRecording) {
	path := row.ThumbnailPath
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		ProblemNotFound(c, ProblemCodeNotFound, "Recording thumbnail is unavailable on this server")
		return
	}
	if ct := service.ThumbnailContentType(filepath.Ext(path)); ct != "" {
		c.Header("Content-Type", ct)
	}
	// Previews are per-user and can change under the same URL, so let the browser
	// keep it briefly but never let an intermediary share it across users.
	c.Header("Cache-Control", "private, max-age=300")
	c.File(path)
}

func serveArtifact(c *gin.Context, row *models.SessionRecording) {
	path := row.ArtifactPath
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		ProblemNotFound(c, ProblemCodeNotFound, "Recording artifact is unavailable on this server")
		return
	}
	if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
		c.Header("Content-Type", ct)
	}
	c.Header("Content-Disposition",
		fmt.Sprintf("inline; filename=%q", filepath.Base(path)))
	// Serve the exact file we stat'ed; anything outside the recording store is
	// impossible because the path is derived from the row, not from the request.
	c.File(path)
}

// CreateArtifactTicket handles POST /v1/me/recordings/:id/artifact-ticket.
//
// Browsers cannot attach an Authorization header to <video src>, window.open or
// <a download>, so playback/download needs a URL that authenticates by itself.
// The ticket is short-lived, bound to (recording, user) and signature-checked.
func (h *SessionRecordingHandler) CreateArtifactTicket(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	if h.store.URLSecret == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			"playback tickets are disabled: set RECORDING_URL_SECRET or API_KEY")
		return
	}
	row, err := h.recordings.ArtifactFor(c.Request.Context(), uid, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording artifact not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	ticket, exp := service.SignRecordingTicket(h.store.URLSecret, row.ID, uid,
		service.TicketKindArtifact, h.ticketTTL())
	c.JSON(http.StatusOK, gin.H{
		"url":        fmt.Sprintf("/v1/recordings/%d/artifact?t=%s", row.ID, ticket),
		"expires_at": exp,
	})
}

// CreateThumbnailTicket mints the same kind of short-lived credential as the
// artifact route, but scoped to the preview image.
func (h *SessionRecordingHandler) CreateThumbnailTicket(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	if h.store.URLSecret == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			"playback tickets are disabled: set RECORDING_URL_SECRET or API_KEY")
		return
	}
	row, err := h.recordings.ThumbnailFor(c.Request.Context(), uid, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording thumbnail not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	ticket, exp := service.SignRecordingTicket(h.store.URLSecret, row.ID, uid,
		service.TicketKindThumbnail, h.ticketTTL())
	c.JSON(http.StatusOK, gin.H{
		"url":        fmt.Sprintf("/v1/recordings/%d/thumbnail?t=%s", row.ID, ticket),
		"expires_at": exp,
	})
}

func (h *SessionRecordingHandler) ticketTTL() time.Duration {
	if h.store.TicketTTL > 0 {
		return h.store.TicketTTL
	}
	return 10 * time.Minute
}

// DownloadArtifactByTicket handles GET /v1/recordings/:id/artifact?t=...
//
// This route is intentionally registered outside the API-key and user-auth
// groups: a top-level browser navigation carries neither an X-API-Key nor an
// Origin header, so the signed ticket is the only credential available.
func (h *SessionRecordingHandler) DownloadArtifactByTicket(c *gin.Context) {
	if h.store.URLSecret == "" {
		ProblemNotFound(c, ProblemCodeNotFound, "Recording artifact not found")
		return
	}
	ticket, err := service.VerifyRecordingTicket(h.store.URLSecret, c.Query("t"))
	if err != nil {
		httpx.Unauthorized(c, httpx.CodeUnauthorized, "invalid or expired recording ticket")
		return
	}
	// A playback link must not be replayable against the preview route; the
	// ticket carries its own purpose for exactly this check.
	if ticket.Kind != service.TicketKindArtifact {
		ProblemNotFound(c, ProblemCodeNotFound, "Recording artifact not found")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || uint(id) != ticket.RecordingID {
		ProblemNotFound(c, ProblemCodeNotFound, "Recording artifact not found")
		return
	}
	row, err := h.recordings.ArtifactFor(c.Request.Context(), ticket.UserID, ticket.RecordingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording artifact not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	// The playback ticket is the authorization for this read, so the actor is
	// the user the ticket was minted for - recording it here is what makes a
	// leaked ticket's use visible.
	h.auditMediaRead(c, ticket.UserID, "recording.download", ticket.RecordingID,
		fmt.Sprintf("size_bytes=%d via=ticket", row.SizeBytes))
	serveArtifact(c, row)
}

// DownloadThumbnailByTicket handles GET /v1/recordings/:id/thumbnail?t=...
//
// Same reasoning as the artifact route: the user's own recordings page renders
// previews with an <img> tag, which cannot carry an Authorization header.
func (h *SessionRecordingHandler) DownloadThumbnailByTicket(c *gin.Context) {
	if h.store.URLSecret == "" {
		ProblemNotFound(c, ProblemCodeNotFound, "Recording thumbnail not found")
		return
	}
	ticket, err := service.VerifyRecordingTicket(h.store.URLSecret, c.Query("t"))
	if err != nil {
		httpx.Unauthorized(c, httpx.CodeUnauthorized, "invalid or expired recording ticket")
		return
	}
	if ticket.Kind != service.TicketKindThumbnail {
		ProblemNotFound(c, ProblemCodeNotFound, "Recording thumbnail not found")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || uint(id) != ticket.RecordingID {
		ProblemNotFound(c, ProblemCodeNotFound, "Recording thumbnail not found")
		return
	}
	row, err := h.recordings.ThumbnailFor(c.Request.Context(), ticket.UserID, ticket.RecordingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording thumbnail not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	h.auditMediaRead(c, ticket.UserID, "recording.thumbnail.download", ticket.RecordingID, "via=ticket")
	serveThumbnail(c, row)
}
