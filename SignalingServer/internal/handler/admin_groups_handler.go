package handler

import (
	"errors"
	"net/http"
	"strconv"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminGroupsHandler serves /v1/admin/groups/*.
type AdminGroupsHandler struct {
	groups *service.DeviceGroupService
	audit  *service.AuditService
}

func NewAdminGroupsHandler(g *service.DeviceGroupService, a *service.AuditService) *AdminGroupsHandler {
	return &AdminGroupsHandler{groups: g, audit: a}
}

func (h *AdminGroupsHandler) List(c *gin.Context) {
	groups, err := h.groups.GetAll(c.Request.Context())
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	type groupWithCount struct {
		models.DeviceGroup
		DeviceCount int64 `json:"device_count"`
	}
	items := make([]groupWithCount, len(groups))
	for i, g := range groups {
		count, _ := h.groups.CountDevices(c.Request.Context(), g.ID)
		items[i] = groupWithCount{DeviceGroup: g, DeviceCount: count}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type groupReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

func (h *AdminGroupsHandler) Create(c *gin.Context) {
	var req groupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if req.Name == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "name required")
		return
	}
	g := &models.DeviceGroup{Name: req.Name, Description: req.Description, Color: req.Color}
	if err := h.groups.Create(c.Request.Context(), g); err != nil {
		ProblemConflict(c, ProblemCodeConflict, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "group.create", "group", strconv.FormatUint(uint64(g.ID), 10), "", c.ClientIP())
	c.JSON(http.StatusCreated, g)
}

func (h *AdminGroupsHandler) Patch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	g, err := h.groups.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Group not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	var req groupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if req.Name != "" {
		g.Name = req.Name
	}
	if req.Description != "" {
		g.Description = req.Description
	}
	if req.Color != "" {
		g.Color = req.Color
	}
	if err := h.groups.Update(c.Request.Context(), g); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "group.update", "group", c.Param("id"), "", c.ClientIP())
	c.JSON(http.StatusOK, g)
}

func (h *AdminGroupsHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	if err := h.groups.Delete(c.Request.Context(), uint(id)); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "group.delete", "group", c.Param("id"), "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type groupDevicesReq struct {
	DeviceIDs []string `json:"device_ids" binding:"required"`
}

func (h *AdminGroupsHandler) AddDevices(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	var req groupDevicesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if err := h.groups.AddDevices(c.Request.Context(), uint(id), req.DeviceIDs); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AdminGroupsHandler) RemoveDevices(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	var req groupDevicesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if err := h.groups.RemoveDevices(c.Request.Context(), uint(id), req.DeviceIDs); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AdminGroupsHandler) ListDevices(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	devs, err := h.groups.GetDeviceIDs(c.Request.Context(), uint(id))
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": devs})
}
