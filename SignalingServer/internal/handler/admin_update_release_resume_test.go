package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// The resumable installer upload spans requests, so the only honest test of it
// is one that sends chunks to a real handler over a real database and then reads
// the bytes back off disk. The arithmetic behind it is pinned separately in
// session_recording_resume_test.go, which this route shares.
func newUpdateArtifactTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the resumable upload tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.UpdateRelease{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE update_releases").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

type updateArtifactRouter struct {
	router   *gin.Engine
	releases *service.UpdateReleaseService
	dir      string
	path     string
	id       uint
}

func newUpdateArtifactRouter(t *testing.T, digest string) updateArtifactRouter {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := newUpdateArtifactTestDB(t)
	releases := service.NewUpdateReleaseService(db)
	dir := t.TempDir()
	h := NewAdminUpdateReleaseHandler(releases, service.NewAuditService(db),
		UpdateArtifactStore{Dir: dir, MaxBytes: 1 << 20})
	router := gin.New()
	router.PATCH("/v1/admin/update-releases/:id/artifact", h.AppendArtifact)

	release, err := releases.Publish(context.Background(), service.PublishUpdateReleaseInput{
		Platform: "windows",
		Arch:     "x64",
		Version:  "4.2.7.0",
		SHA256:   digest,
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	id := strconv.FormatUint(uint64(release.ID), 10)
	return updateArtifactRouter{
		router:   router,
		releases: releases,
		dir:      dir,
		path:     "/v1/admin/update-releases/" + id + "/artifact?filename=QuickDesk-4.2.7.0.exe",
		id:       release.ID,
	}
}

func (r updateArtifactRouter) patch(t *testing.T, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("PATCH", r.path, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()
	r.router.ServeHTTP(rec, req)
	return rec
}

func TestUpdateArtifactResumableUpload(t *testing.T) {
	payload := bytes.Repeat([]byte("quickdesk-installer-"), 500)
	sum := sha256.Sum256(payload)
	h := newUpdateArtifactRouter(t, hex.EncodeToString(sum[:]))
	total := strconv.Itoa(len(payload))
	half := len(payload) / 2

	// First chunk: staged, not finished. 202 carries the authoritative offset.
	first := h.patch(t, payload[:half], map[string]string{
		"Upload-Offset": "0",
		"Upload-Length": total,
	})
	if first.Code != http.StatusAccepted {
		t.Fatalf("first chunk status = %d, want 202: %s", first.Code, first.Body.String())
	}
	if got := first.Header().Get("Upload-Offset"); got != strconv.Itoa(half) {
		t.Fatalf("first chunk Upload-Offset = %q, want %d", got, half)
	}

	// A probe asks where the upload stands and must not move it or read a body.
	probe := h.patch(t, nil, map[string]string{
		"Upload-Offset": strconv.Itoa(half),
		"Upload-Length": total,
		"Upload-Probe":  "1",
	})
	if probe.Code != http.StatusAccepted {
		t.Fatalf("probe status = %d, want 202: %s", probe.Code, probe.Body.String())
	}
	if got := probe.Header().Get("Upload-Offset"); got != strconv.Itoa(half) {
		t.Fatalf("probe Upload-Offset = %q, want the unchanged %d", got, half)
	}

	// A chunk that starts somewhere else is refused before its body is read, and
	// the header tells the client exactly where to resume from.
	mismatch := h.patch(t, payload[half:], map[string]string{
		"Upload-Offset": "0",
		"Upload-Length": total,
	})
	if mismatch.Code != http.StatusConflict {
		t.Fatalf("offset mismatch status = %d, want 409: %s", mismatch.Code, mismatch.Body.String())
	}
	if got := mismatch.Header().Get("Upload-Offset"); got != strconv.Itoa(half) {
		t.Fatalf("mismatch Upload-Offset = %q, want %d", got, half)
	}
	staged, err := artifactStagedBytes(storedUpdateArtifactPath(h, "QuickDesk-4.2.7.0.exe") + ".part")
	if err != nil {
		t.Fatalf("staged bytes: %v", err)
	}
	if staged != int64(half) {
		t.Fatalf("a refused chunk changed the staging file to %d bytes, want %d", staged, half)
	}

	// The last chunk finishes the upload, and only then does the release become
	// fetchable.
	last := h.patch(t, payload[half:], map[string]string{
		"Upload-Offset":   strconv.Itoa(half),
		"Upload-Length":   total,
		"Upload-Complete": "1",
	})
	if last.Code != http.StatusOK {
		t.Fatalf("finishing chunk status = %d, want 200: %s", last.Code, last.Body.String())
	}
	var finalized models.UpdateRelease
	if err := json.Unmarshal(last.Body.Bytes(), &finalized); err != nil {
		t.Fatalf("decode finalized release: %v", err)
	}
	if !finalized.HostedHere || finalized.DownloadURL == "" {
		t.Fatalf("finished upload = %+v, want a hosted release with a download url", finalized)
	}
	if finalized.SizeBytes != int64(len(payload)) {
		t.Fatalf("size_bytes = %d, want %d", finalized.SizeBytes, len(payload))
	}

	stored, err := os.ReadFile(storedUpdateArtifactPath(h, "QuickDesk-4.2.7.0.exe"))
	if err != nil {
		t.Fatalf("read stored artifact: %v", err)
	}
	if !bytes.Equal(stored, payload) {
		t.Fatalf("stored %d bytes, want the %d uploaded bytes", len(stored), len(payload))
	}
	if _, err := os.Stat(storedUpdateArtifactPath(h, "QuickDesk-4.2.7.0.exe") + ".part"); !os.IsNotExist(err) {
		t.Fatalf("the staging file survived a finished upload: %v", err)
	}

	// The release is now what a client sees.
	result, err := h.releases.Check(context.Background(), "windows", "x64", "stable", "4.2.0.0")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !result.UpdateAvailable || result.Latest == nil || !result.Latest.HostedHere {
		t.Fatalf("check after a resumable upload = %+v, want the hosted release", result)
	}
}

// The whole point of hosting the bytes is that the channel can promise they are
// the artifact it advertises. A resumable upload only learns the digest at the
// end, so that is where it has to refuse - and it has to throw the staged bytes
// away rather than leave a rejected payload on disk.
func TestUpdateArtifactResumableUploadRejectsADifferentPayload(t *testing.T) {
	advertised := bytes.Repeat([]byte("a"), 2048)
	uploaded := bytes.Repeat([]byte("b"), 2048)
	sum := sha256.Sum256(advertised)
	h := newUpdateArtifactRouter(t, hex.EncodeToString(sum[:]))
	total := strconv.Itoa(len(uploaded))

	first := h.patch(t, uploaded[:1024], map[string]string{
		"Upload-Offset": "0",
		"Upload-Length": total,
	})
	if first.Code != http.StatusAccepted {
		t.Fatalf("first chunk status = %d, want 202: %s", first.Code, first.Body.String())
	}

	last := h.patch(t, uploaded[1024:], map[string]string{
		"Upload-Offset":   "1024",
		"Upload-Length":   total,
		"Upload-Complete": "1",
	})
	if last.Code != http.StatusBadRequest {
		t.Fatalf("mismatched payload status = %d, want 400: %s", last.Code, last.Body.String())
	}

	stagedPath := storedUpdateArtifactPath(h, "QuickDesk-4.2.7.0.exe")
	if _, err := os.Stat(stagedPath); !os.IsNotExist(err) {
		t.Fatalf("a rejected payload was promoted into place: %v", err)
	}
	if _, err := os.Stat(stagedPath + ".part"); !os.IsNotExist(err) {
		t.Fatalf("a rejected payload was left staged: %v", err)
	}
	release, err := h.releases.Get(context.Background(), h.id)
	if err != nil {
		t.Fatalf("reload release: %v", err)
	}
	if release.HostedHere || release.DownloadURL != "" {
		t.Fatalf("a rejected upload changed the release: %+v", release)
	}
}

func storedUpdateArtifactPath(h updateArtifactRouter, name string) string {
	return h.dir + "/" + strconv.FormatUint(uint64(h.id), 10) + "/" + name
}
