package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

type WebhookService struct {
	db *gorm.DB
}

func NewWebhookService(db *gorm.DB) *WebhookService {
	return &WebhookService{db: db}
}

func (s *WebhookService) Create(ctx context.Context, webhook *models.Webhook) error {
	return s.db.WithContext(ctx).Create(webhook).Error
}

func (s *WebhookService) GetAll(ctx context.Context) ([]models.Webhook, error) {
	var webhooks []models.Webhook
	err := s.db.WithContext(ctx).Order("created_at desc").Find(&webhooks).Error
	return webhooks, err
}

func (s *WebhookService) GetByID(ctx context.Context, id uint) (*models.Webhook, error) {
	var webhook models.Webhook
	err := s.db.WithContext(ctx).First(&webhook, id).Error
	return &webhook, err
}

func (s *WebhookService) Update(ctx context.Context, webhook *models.Webhook) error {
	return s.db.WithContext(ctx).Save(webhook).Error
}

func (s *WebhookService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Webhook{}, id).Error
}

func (s *WebhookService) Dispatch(event string, data interface{}) {
	var webhooks []models.Webhook
	if err := s.db.Where("enabled = ?", true).Find(&webhooks).Error; err != nil {
		return
	}

	payload := map[string]interface{}{
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      data,
	}

	for _, wh := range webhooks {
		var events []string
		if err := json.Unmarshal([]byte(wh.Events), &events); err != nil {
			continue
		}
		subscribed := false
		for _, e := range events {
			if e == event {
				subscribed = true
				break
			}
		}
		if !subscribed {
			continue
		}
		go s.deliver(wh, payload)
	}
}

func (s *WebhookService) deliver(webhook models.Webhook, payload map[string]interface{}) {
	lastStatus := postWebhook(webhook.URL, webhook.Secret, payload)

	now := time.Now()
	if err := s.db.Model(&webhook).Updates(map[string]interface{}{
		"last_triggered": now,
		"last_status":    lastStatus,
	}).Error; err != nil {
		log.Printf("Failed to update webhook status: %v", err)
	}
}

func (s *WebhookService) Test(ctx context.Context, id uint) error {
	webhook, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"event":     "test",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      map[string]string{"message": "This is a test webhook event"},
	}

	go s.deliver(*webhook, payload)
	return nil
}
