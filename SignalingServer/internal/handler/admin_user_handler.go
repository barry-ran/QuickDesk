package handler

import (
	"net/http"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AdminUserHandler handles admin user management API
type AdminUserHandler struct {
	service *service.AdminUserService
}

// NewAdminUserHandler creates a new AdminUserHandler
func NewAdminUserHandler(service *service.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{service: service}
}

// GetAdminUsers handles GET /api/v1/admin/users
func (h *AdminUserHandler) GetAdminUsers(c *gin.Context) {
	ctx := c.Request.Context()

	users, err := h.service.GetAllAdminUsers(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get admin users"})
		return
	}

	var response []models.AdminUserResponse
	for _, user := range users {
		response = append(response, user.ToResponse())
	}

	c.JSON(http.StatusOK, gin.H{"users": response})
}

// CreateAdminUser handles POST /api/v1/admin/users
func (h *AdminUserHandler) CreateAdminUser(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.CreateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.CreateAdminUser(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user.ToResponse())
}

// UpdateAdminUser handles PUT /api/v1/admin/users/:id
func (h *AdminUserHandler) UpdateAdminUser(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req models.UpdateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.UpdateAdminUser(ctx, uint(id), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// DeleteAdminUser handles DELETE /api/v1/admin/users/:id
func (h *AdminUserHandler) DeleteAdminUser(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.service.DeleteAdminUser(ctx, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete admin user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin user deleted successfully"})
}
