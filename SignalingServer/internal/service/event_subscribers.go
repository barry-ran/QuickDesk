package service

import (
	"context"
	"encoding/json"
	"fmt"
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
		EventSessionRevoked,
		// Rejected remote-access attempts are the security-relevant business
		// events operators want in the admin audit trail. Successful ones
		// live in `device_access_logs` (owner-scoped) instead, so the audit
		// table does not grow a row per healthy session.
		EventDeviceAccessDenied,
		// Changing who may reach a device is itself an auditable act.
		EventDevicePolicyChanged,
		// Issuing or cancelling a shared guest code is the same kind of act.
		EventDeviceAccessGrantChanged,
		// Approving or denying a waiting session is a deliberate admission
		// decision and belongs in the admin trail alongside policy changes.
		EventDeviceAccessApprovalDecided,
		// Deleting a recording destroys evidence, so "it was deleted, by this
		// actor, at this time" is exactly what a later review needs. Creating
		// one is deliberately not audited: the recording index already records
		// it, and a row per session would only dilute the trail.
		EventRecordingDeleted:
	default:
		return
	}
	details, _ := json.Marshal(evt.Data)
	action := auditAction(evt)
	resourceType, resourceID := auditResource(evt)
	if actor := auditActorUserID(evt); actor != 0 {
		s.audit.LogActor(ctx, actor, action, resourceType, resourceID, string(details), eventClientIP(evt))
		return
	}
	s.audit.Log(ctx, 0, "", action, resourceType, resourceID, string(details), "")
}

// eventClientIP pulls the actor's observed address out of the event data.
// Producers that attribute an event to a user put `ip` there so the audit row
// can answer "from where" the same way the admin path does. A missing or
// non-string value yields "" rather than a stringified map, because an audit
// row that says `ip="map[]"` is worse than one that says nothing.
func eventClientIP(evt Event) string {
	if v, ok := evt.Data["ip"].(string); ok {
		return v
	}
	return ""
}

// auditAction is the name written to the trail. It defaults to the event type,
// but where an administrator action already exists for the same fact the two
// have to agree: an operator filtering `recording.delete` must see every
// deletion, whoever performed it. Which id column is set - admin_id or
// actor_user_id - is what tells them apart.
func auditAction(evt Event) string {
	if evt.Type == EventRecordingDeleted {
		return "recording.delete"
	}
	return evt.Type
}

// auditResource names what an audited event is about. The device id is the
// default because most of these events are about a device; a recording deletion
// is about the recording, and the device is incidental.
func auditResource(evt Event) (resourceType, resourceID string) {
	if evt.Type == EventRecordingDeleted {
		if v, ok := evt.Data["recording_id"]; ok && v != nil {
			return "session_recording", fmt.Sprintf("%v", v)
		}
		return "session_recording", evt.DeviceID
	}
	return "event", evt.DeviceID
}

// auditActorUserID reports the user who *did* the thing, or 0 when the event's
// user is only the subject of it.
//
// The distinction is the difference between a useful trail and a misleading
// one: a rejected access attempt is routed to the device owner but was caused
// by somebody else, so recording the owner as its actor would file a stranger's
// action under the owner's history. Only the events below are acts a user
// performs on their own resources.
func auditActorUserID(evt Event) uint {
	switch evt.Type {
	case EventRecordingDeleted,
		EventDevicePolicyChanged,
		EventDeviceAccessGrantChanged,
		EventDeviceAccessApprovalDecided:
		return evt.UserID
	default:
		return 0
	}
}

// userNotificationSubscriber delivers user-scoped events to that user's own
// notification channels. It is the user-level counterpart of the admin
// webhook subscriber: admin webhooks only carry system-scope events, so
// without this subscriber a user could never be told that somebody tried to
// open their device.
type userNotificationSubscriber struct {
	channels *NotificationChannelService
}

func NewUserNotificationSubscriber(c *NotificationChannelService) EventSubscriber {
	return &userNotificationSubscriber{channels: c}
}

func (s *userNotificationSubscriber) Name() string { return "user_notifications" }

func (s *userNotificationSubscriber) HandleEvent(ctx context.Context, evt Event) {
	if evt.UserID == 0 {
		return
	}
	s.channels.DispatchEvent(ctx, evt)
}
