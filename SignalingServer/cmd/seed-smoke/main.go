// Command seed-smoke prepares the account and device that
// scripts/sdk/openapi-smoke.mjs needs against a fresh server, and prints the
// environment the smoke reads.
//
// It exists because the smoke used to require a hand-built account: the device
// it checks is provisioned by a real controlled machine, which CI does not
// have. The user is created through the admin API (so the password is hashed by
// the server's own path, not by a second implementation here) and the device is
// one row bound to that user, which is all the smoke's device-scoped endpoints
// need. The same recipe therefore works on a laptop and in the workflow.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/redis/go-redis/v9"

	"gorm.io/gorm"
)

func main() {
	server := flag.String("server", envOr("QD_SERVER", "http://127.0.0.1:8000"), "signaling server base URL")
	apiKey := flag.String("api-key", envOr("QD_API_KEY", ""), "X-API-Key for the server")
	adminUser := flag.String("admin-user", envOr("ADMIN_USER", "admin"), "admin username")
	adminPassword := flag.String("admin-password", envOr("ADMIN_PASSWORD", ""), "admin password")
	username := flag.String("username", "", "account to create (default: smoke-<unix>)")
	password := flag.String("password", "", "password for that account")
	deviceID := flag.String("device-id", "", "9-digit device id to bind (default: derived from the clock)")
	deviceName := flag.String("device-name", "smoke device", "device label")
	flag.Parse()

	suffix := time.Now().Unix()
	if strings.TrimSpace(*username) == "" {
		*username = fmt.Sprintf("smoke-%d", suffix)
	}
	if strings.TrimSpace(*password) == "" {
		*password = fmt.Sprintf("Smoke-%d!", suffix)
	}
	if strings.TrimSpace(*deviceID) == "" {
		// A 9-digit id that does not start with 0, so it matches the public
		// device-id shape the API validates.
		*deviceID = strconv.FormatInt(100000000+(suffix%900000000), 10)
	}
	if *apiKey == "" {
		log.Fatal("an API key is required (-api-key or QD_API_KEY)")
	}
	if *adminPassword == "" {
		log.Fatal("the admin password is required (-admin-password or ADMIN_PASSWORD)")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	token, err := adminLogin(client, *server, *apiKey, *adminUser, *adminPassword)
	if err != nil {
		log.Fatalf("admin login: %v", err)
	}
	userID, err := createUser(client, *server, *apiKey, token, *username, *password)
	if err != nil {
		log.Fatalf("create the smoke account: %v", err)
	}
	if err := bindDevice(*deviceID, *deviceName, userID); err != nil {
		log.Fatalf("bind the smoke device: %v", err)
	}
	if err := markDeviceOnline(*deviceID); err != nil {
		log.Fatalf("mark the smoke device online: %v", err)
	}

	// Printed as shell assignments so a caller can `eval` them or hand them to
	// the smoke step directly.
	fmt.Printf("QD_IDENTIFIER=%s\n", *username)
	fmt.Printf("QD_PASSWORD=%s\n", *password)
	fmt.Printf("QD_SMOKE_DEVICE=%s\n", *deviceID)
	// The smoke has to run inside the presence TTL, and the device-scoped
	// endpoints answer 409 HOST_OFFLINE without it.
	log.Printf("device %s is online for %s (presence heartbeat TTL); run the smoke now",
		*deviceID, presenceHeartbeatTTL)
}

// presenceHeartbeatTTL mirrors the TTL the presence service applies to a
// heartbeat, so the message above states the real window.
const presenceHeartbeatTTL = 90 * time.Second

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// adminLogin exchanges the seeded administrator credentials for a JWT.
func adminLogin(client *http.Client, server, apiKey, username, password string) (string, error) {
	var out struct {
		AccessToken string `json:"access_token"`
	}
	body := map[string]string{"username": username, "password": password}
	if err := postJSON(client, server+"/v1/admin/auth/sessions", apiKey, "", body, &out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("login succeeded without an access_token")
	}
	return out.AccessToken, nil
}

// createUser creates the account the smoke logs in as. The admin path is used
// deliberately: it goes through the server's own password hashing, so this tool
// never has to know how credentials are stored.
func createUser(client *http.Client, server, apiKey, token, username, password string) (uint, error) {
	var out struct {
		ID uint `json:"id"`
	}
	body := map[string]string{"username": username, "password": password}
	if err := postJSON(client, server+"/v1/admin/users", apiKey, token, body, &out); err != nil {
		return 0, err
	}
	if out.ID == 0 {
		return 0, fmt.Errorf("create returned no user id")
	}
	return out.ID, nil
}

// bindDevice writes one device row owned by the smoke account. The smoke only
// needs the device to exist and belong to the caller: it asserts the *shape* of
// the device-scoped read models, and the one write it performs against a device
// endpoint is a deliberately wrong access code.
func bindDevice(deviceID, deviceName string, userID uint) error {
	cfg := config.Load()
	db := database.InitPostgreSQL(cfg)
	owner := userID
	row := models.Device{
		DeviceID:   deviceID,
		DeviceUUID: fmt.Sprintf("smoke-%s", deviceID),
		OS:         "Windows",
		OSVersion:  "11",
		AppVersion: "smoke",
		UserID:     &owner,
		DeviceName: deviceName,
		// Deliberately not 000000: the smoke pins the rejected-access-code shape
		// by sending exactly that, and a device that happened to use it would
		// turn the expected 403 into a 200.
		AccessCode: "135790",
		LoggedIn:   true,
	}
	// Written as an explicit find-then-write rather than an upsert so re-running
	// with an explicit -device-id is idempotent without GORM building the WHERE
	// clause out of every non-zero field on the row.
	var existing models.Device
	err := db.Where("device_id = ?", deviceID).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return db.Create(&row).Error
	case err != nil:
		return err
	default:
		return db.Model(&existing).Updates(map[string]interface{}{
			"user_id":     userID,
			"device_name": deviceName,
			"logged_in":   true,
		}).Error
	}
}

// markDeviceOnline uses the presence service's own calls - the same ones a real
// host heartbeat and signaling WebSocket use - so a seeded device reads as
// online to the device-scoped endpoints without anyone hand-writing Redis keys.
//
// The instance it claims the device for matters: PresenceState.Online is
// "heartbeat key AND at least one ws instance whose instance key is alive", and
// only a *running* server refreshes its instance key. So the device is attached
// to a live instance found in Redis rather than to this short-lived process,
// which would make the device fall offline again seconds later.
func markDeviceOnline(deviceID string) error {
	cfg := config.Load()
	rdb := database.InitRedis(cfg)
	ctx := context.Background()
	instance, err := liveInstance(ctx, rdb)
	if err != nil {
		return err
	}
	presence := service.NewPresenceService(rdb, instance)
	if err := presence.MarkWSConnected(ctx, deviceID); err != nil {
		return err
	}
	return presence.Heartbeat(ctx, deviceID)
}

// liveInstance returns one signaling instance that is currently refreshing its
// presence key.
func liveInstance(ctx context.Context, rdb *redis.Client) (string, error) {
	var cursor uint64
	for {
		keys, next, err := rdb.Scan(ctx, cursor, "qd:presence:instance:*", 50).Result()
		if err != nil {
			return "", err
		}
		for _, key := range keys {
			if id := strings.TrimPrefix(key, "qd:presence:instance:"); id != "" {
				return id, nil
			}
		}
		cursor = next
		if cursor == 0 {
			return "", errors.New("no live signaling instance found in Redis; start the server before seeding")
		}
	}
}

func postJSON(client *http.Client, url, apiKey, token string, body interface{}, out interface{}) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("%s -> %d %s", url, res.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}
