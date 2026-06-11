package service

import (
	"context"
	"encoding/json"
	"log"
)

// webhookSubscriber wires the EventBus onto the webhook service so admin-
// scope events fan out to HTTP webhooks. Per §2.17 M4, we only subscribe
// to *system-level* event types — user-scoped events (with a non-zero
// UserID) carry per-user data and would leak if broadcast to a webhook
// that isn't associated with that user.
type webhookSubscriber struct {
	webhooks *WebhookService
}

func NewWebhookSubscriber(w *WebhookService) EventSubscriber {
	return &webhookSubscriber{webhooks: w}
}

func (s *webhookSubscriber) Name() string { return "webhook" }

// webhookEligibleTypes lists the event types webhook subscribers may see.
var webhookEligibleTypes = map[string]struct{}{
	EventTurnConfigChang:     {},
	EventDeviceSecretRotated: {},
	EventSessionRevoked:      {},
}

func (s *webhookSubscriber) HandleEvent(ctx context.Context, evt Event) {
	if _, ok := webhookEligibleTypes[evt.Type]; !ok {
		return
	}
	var data map[string]interface{}
	payload, err := json.Marshal(evt)
	if err != nil {
		log.Printf("[webhook] marshal event: %v", err)
		return
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return
	}
	s.webhooks.Dispatch(evt.Type, data)
}

// auditSubscriber writes admin-originated events to the audit_logs table.
// User-scoped business events (device.remark.changed, favorite.added, ...)
// intentionally don't get written — otherwise the table would balloon.
type auditSubscriber struct {
	audit *AuditService
}

func NewAuditSubscriber(a *AuditService) EventSubscriber {
	return &auditSubscriber{audit: a}
}

func (s *auditSubscriber) Name() string { return "audit" }

func (s *auditSubscriber) HandleEvent(ctx context.Context, evt Event) {
	switch evt.Type {
	case EventDeviceSecretRotated,
		EventTurnConfigChang,
		EventSessionRevoked:
	default:
		return
	}
	details, _ := json.Marshal(evt.Data)
	s.audit.Log(ctx, 0, "", evt.Type, "event", evt.DeviceID, string(details), "")
}
