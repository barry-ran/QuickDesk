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

// AdminAdminsHandler serves /v1/admin/admins/* — CRUD over admin user
// accounts themselves. 2FA lives in AdminTOTPHandler.
type AdminAdminsHandler struct {
	admins *service.AdminUserService
	tokens *service.TokenService
	audit  *service.AuditService
}

func NewAdminAdminsHandler(admins *service.AdminUserService, tokens *service.TokenService, audit *service.AuditService) *AdminAdminsHandler {
	return &AdminAdminsHandler{admins: admins, tokens: tokens, audit: audit}
}

func (h *AdminAdminsHandler) List(c *gin.Context) {
	users, err := h.admins.GetAllAdminUsers(c.Request.Context())
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	items := make([]models.AdminUserResponse, 0, len(users))
	for _, u := range users {
		items = append(items, u.ToResponse())
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *AdminAdminsHandler) Create(c *gin.Context) {
	var req models.CreateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	u, err := h.admins.CreateAdminUser(c.Request.Context(), &req)
	if err != nil {
		ProblemConflict(c, ProblemCodeConflict, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "admin.create", "admin_user", formatUint(u.ID), "", c.ClientIP())
	c.JSON(http.StatusCreated, u.ToResponse())
}

func (h *AdminAdminsHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	u, err := h.admins.GetAdminUserByID(c.Request.Context(), uint(id))
	if err != nil {
		ProblemNotFound(c, ProblemCodeNotFound, "Admin not found")
		return
	}
	c.JSON(http.StatusOK, u.ToResponse())
}

func (h *AdminAdminsHandler) Patch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	var req models.UpdateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	u, passwordChanged, err := h.admins.UpdateAdminUser(c.Request.Context(), uint(id), &req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Admin not found")
			return
		}
		ProblemConflict(c, ProblemCodeConflict, err.Error())
		return
	}
	// §2.16: Admin session 改密码后 revoke 本账号全部 session（包括自己）.
	if passwordChanged {
		h.tokens.RevokeAllForSubject(c.Request.Context(), service.ScopeAdmin, u.ID)
		h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "admin.password.reset", "admin_user", formatUint(u.ID), "", c.ClientIP())
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "admin.update", "admin_user", formatUint(u.ID), "", c.ClientIP())
	c.JSON(http.StatusOK, u.ToResponse())
}

func (h *AdminAdminsHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	if err := h.admins.DeleteAdminUser(c.Request.Context(), uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Admin not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "admin.delete", "admin_user", c.Param("id"), "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
