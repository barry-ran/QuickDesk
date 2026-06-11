package service

import (
	"context"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

func (s *AuditService) Log(ctx context.Context, adminID uint, adminUsername, action, resourceType, resourceID, details, ip string) {
	entry := &models.AuditLog{
		AdminID:       adminID,
		AdminUsername: adminUsername,
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		Details:       details,
		IP:            ip,
		CreatedAt:     time.Now(),
	}
	s.db.WithContext(ctx).Create(entry)
}

// AuditListParams is the cursor-keyset filter used by GET /v1/admin/audit-logs.
type AuditListParams struct {
	AfterID                 uint
	Limit                   int
	Action, AdminUsername   string
	DateFrom, DateTo        string
}

func (s *AuditService) List(ctx context.Context, p AuditListParams) ([]models.AuditLog, int64, error) {
	query := s.db.WithContext(ctx).Model(&models.AuditLog{})

	if p.Action != "" {
		query = query.Where("action = ?", p.Action)
	}
	if p.AdminUsername != "" {
		query = query.Where("admin_username LIKE ?", "%"+p.AdminUsername+"%")
	}
	if p.DateFrom != "" {
		if t, err := time.Parse(time.RFC3339, p.DateFrom); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if p.DateTo != "" {
		if t, err := time.Parse(time.RFC3339, p.DateTo); err == nil {
			query = query.Where("created_at <= ?", t)
		}
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if p.AfterID > 0 {
		query = query.Where("id < ?", p.AfterID)
	}
	var logs []models.AuditLog
	err := query.Order("id DESC").Limit(p.Limit).Find(&logs).Error
	return logs, total, err
}
