package handler

import (
	"errors"
	"net/http"
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
//
// The bytes stay wherever the operator hosts them; this is the index of which
// version exists for which channel/platform/arch and what it hashes to.
type AdminUpdateReleaseHandler struct {
	releases *service.UpdateReleaseService
	audit    *service.AuditService
}

func NewAdminUpdateReleaseHandler(releases *service.UpdateReleaseService, audit *service.AuditService) *AdminUpdateReleaseHandler {
	return &AdminUpdateReleaseHandler{releases: releases, audit: audit}
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
	if err := h.releases.Delete(c.Request.Context(), uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Update release not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "",
		"update_release.delete", "update_release", c.Param("id"), "", c.ClientIP())
	c.Status(http.StatusNoContent)
}
