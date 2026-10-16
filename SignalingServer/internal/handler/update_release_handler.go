package handler

import (
	"errors"
	"net/http"

	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// UpdateReleaseHandler serves the client-facing half of the self-hosted update
// channel:
//
//	GET /v1/update:check?platform=&arch=&channel=&current_version=
//
// It sits in the API-key gated v1 group but is not user-authenticated: the
// controlled machine checks for updates before anyone logs in. The answer is
// the newest published release for that channel/platform/arch and whether it is
// strictly newer than the caller's version.
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
