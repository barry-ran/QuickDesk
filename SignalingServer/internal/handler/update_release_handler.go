package handler

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UpdateReleaseHandler serves the client-facing half of the self-hosted update
// channel:
//
//	GET /v1/update:check?platform=&arch=&channel=&current_version=
//	GET /v1/update-releases/:id/artifact
//
// It sits in the API-key gated v1 group but is not user-authenticated: the
// controlled machine checks for updates before anyone logs in. The answer is
// the newest published release for that channel/platform/arch and whether it is
// strictly newer than the caller's version. The artifact route serves installers
// the operator uploaded to this server; externally hosted ones never touch it.
type UpdateReleaseHandler struct {
	releases *service.UpdateReleaseService
}

func NewUpdateReleaseHandler(releases *service.UpdateReleaseService) *UpdateReleaseHandler {
	return &UpdateReleaseHandler{releases: releases}
}

func (h *UpdateReleaseHandler) Check(c *gin.Context) {
	result, err := h.releases.Check(
		c.Request.Context(),
		c.Query("platform"),
		c.Query("arch"),
		c.Query("channel"),
		c.Query("current_version"),
	)
	if err != nil {
		writeUpdateReleaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// DownloadArtifact serves POST-uploaded installer bytes. Clients reach it with
// the same API key they used for the check, which is why it can be a plain GET:
// no browser navigation and therefore no ticket is involved.
//
// The path comes from the release row, never from the request, so a caller
// cannot walk outside the update store. http.ServeContent adds Range and
// If-Modified-Since, so a download that drops halfway can resume instead of
// starting a multi-hundred-megabyte transfer again.
func (h *UpdateReleaseHandler) DownloadArtifact(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid release id")
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
	if release.ArtifactPath == "" {
		ProblemNotFound(c, ProblemCodeNotFound,
			"This release is not hosted on this server; fetch it from download_url")
		return
	}
	file, err := os.Open(release.ArtifactPath)
	if err != nil {
		ProblemNotFound(c, ProblemCodeNotFound, "Update artifact is unavailable on this server")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		ProblemNotFound(c, ProblemCodeNotFound, "Update artifact is unavailable on this server")
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", filepath.Base(release.ArtifactPath)))
	http.ServeContent(c.Writer, c.Request, filepath.Base(release.ArtifactPath), info.ModTime(), file)
}

// writeUpdateReleaseError keeps a client typo (an unknown platform, a version
// that is not dotted numbers) a 400, and reserves 500 for the database.
func writeUpdateReleaseError(c *gin.Context, err error) {
	var invalid *service.UpdateReleaseValidationError
	if errors.As(err, &invalid) {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, invalid.Error())
		return
	}
	ProblemInternal(c, err.Error())
}
