package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

// postWebhookOnce performs a single delivery, returning the HTTP status (0
// when the request never completed) and any transport error. When secret is
// set the body is signed with HMAC-SHA256 and the digest is sent as
// `X-Webhook-Signature: sha256=<hex>`, the same scheme the admin webhooks use
// — receivers can share one verification routine.
//
// headers carries extra identity headers a caller wants on the request (the
// notification outbox uses them to give receivers a stable dedupe key); they
// are applied before the signature header so a caller cannot override the
// digest.
func postWebhookOnce(url, secret string, payload interface{}, headers map[string]string) (int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		if name == "" || value == "" {
			continue
		}
		req.Header.Set(name, value)
	}
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		req.Header.Set("X-Webhook-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// postWebhook is the fire-and-forget variant used by admin webhooks: three
// attempts with backoff, returning the last HTTP status seen.
func postWebhook(url, secret string, payload interface{}) int {
	var lastStatus int
	for attempt := 0; attempt < 3; attempt++ {
		status, err := postWebhookOnce(url, secret, payload, nil)
		lastStatus = status
		if err == nil && status >= 200 && status < 300 {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
	}
	return lastStatus
}
