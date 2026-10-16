package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newGrantTestDB opens the throwaway database and clears the grants table.
func newGrantTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the grant tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// One pool per test against a scratch Postgres with a connection limit;
	// closing it with the test keeps a long package run from exhausting them.
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.DeviceAccessGrant{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE device_access_grants").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func TestDeviceAccessGrantUsableAt(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	revoked := now.Add(-time.Minute)
	cases := []struct {
		name  string
		grant models.DeviceAccessGrant
		want  bool
	}{
		{"fresh", models.DeviceAccessGrant{ExpiresAt: now.Add(time.Hour)}, true},
		{"expired", models.DeviceAccessGrant{ExpiresAt: now.Add(-time.Second)}, false},
		{"expires exactly now is over", models.DeviceAccessGrant{ExpiresAt: now}, false},
		{"revoked", models.DeviceAccessGrant{ExpiresAt: now.Add(time.Hour), RevokedAt: &revoked}, false},
		{"used up", models.DeviceAccessGrant{ExpiresAt: now.Add(time.Hour), MaxUses: 1, Uses: 1}, false},
		{"one use left", models.DeviceAccessGrant{ExpiresAt: now.Add(time.Hour), MaxUses: 2, Uses: 1}, true},
		{"unlimited uses", models.DeviceAccessGrant{ExpiresAt: now.Add(time.Hour), MaxUses: 0, Uses: 99}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.grant.UsableAt(now); got != tc.want {
				t.Errorf("UsableAt = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCreateDeviceAccessGrantValidation(t *testing.T) {
	db := newGrantTestDB(t)
	svc := NewDeviceAccessGrantService(db)
	ctx := context.Background()

	if _, err := svc.Create(ctx, "dev-1", 1, CreateDeviceAccessGrantInput{ExpiresInMinutes: 0}); !errors.Is(err, ErrInvalidAccessGrant) {
		t.Errorf("zero lifetime err = %v, want ErrInvalidAccessGrant", err)
	}
	if _, err := svc.Create(ctx, "dev-1", 1, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30, MaxUses: -1}); !errors.Is(err, ErrInvalidAccessGrant) {
		t.Errorf("negative max_uses err = %v, want ErrInvalidAccessGrant", err)
	}

	// A generated code is the right length and stays inside the unambiguous
	// alphabet, so nobody has to guess between O and 0.
	row, err := svc.Create(ctx, "dev-1", 1, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(row.Code) != 8 {
		t.Errorf("generated code %q has length %d, want 8", row.Code, len(row.Code))
	}
	for _, ch := range row.Code {
		if !strings.ContainsRune(grantCodeAlphabet, ch) {
			t.Errorf("generated code %q contains %q, outside the alphabet", row.Code, ch)
		}
	}

	// A caller-supplied code is kept as-is (trimmed).
	custom, err := svc.Create(ctx, "dev-1", 1, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30, Code: "  GUEST123  "})
	if err != nil {
		t.Fatalf("create with custom code: %v", err)
	}
	if custom.Code != "GUEST123" {
		t.Errorf("custom code = %q, want GUEST123", custom.Code)
	}
}

func TestAccessGrantLifecycle(t *testing.T) {
	db := newGrantTestDB(t)
	svc := NewDeviceAccessGrantService(db)
	ctx := context.Background()

	row, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{
		Label:            "guest",
		ExpiresInMinutes: 60,
		MaxUses:          1,
		Code:             "ONCEONLY",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if matched, err := svc.Match(ctx, "dev-1", "ONCEONLY", "controller-1", "10.0.0.1"); err != nil || matched == nil || matched.ID != row.ID {
		t.Fatalf("match before use = (%v, %v), want the grant", matched, err)
	}
	// A different device must not see this grant, and a wrong code matches nothing.
	if matched, _ := svc.Match(ctx, "dev-2", "ONCEONLY", "controller-1", "10.0.0.1"); matched != nil {
		t.Error("grant leaked to another device")
	}
	if matched, _ := svc.Match(ctx, "dev-1", "WRONGCODE", "controller-1", "10.0.0.1"); matched != nil {
		t.Error("wrong code matched a grant")
	}

	ok, err := svc.Consume(ctx, row.ID, "controller-1", "10.0.0.1")
	if err != nil || !ok {
		t.Fatalf("first consume = (%v, %v), want true", ok, err)
	}
	// Single-use grant is now spent: no match, no second consume.
	if matched, _ := svc.Match(ctx, "dev-1", "ONCEONLY", "controller-1", "10.0.0.1"); matched != nil {
		t.Error("a spent single-use grant still matched")
	}
	if ok, _ := svc.Consume(ctx, row.ID, "controller-1", "10.0.0.1"); ok {
		t.Error("second consume succeeded on a single-use grant")
	}

	var reloaded models.DeviceAccessGrant
	if err := db.First(&reloaded, row.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Uses != 1 || reloaded.LastUsedAt == nil {
		t.Errorf("after use: uses=%d last_used_at=%v, want 1 and set", reloaded.Uses, reloaded.LastUsedAt)
	}
	// The redeemer is written with the use count, so the owner can see who.
	if reloaded.LastUsedClientID != "controller-1" || reloaded.LastUsedIP != "10.0.0.1" {
		t.Errorf("redeemer = (%q, %q), want (controller-1, 10.0.0.1)",
			reloaded.LastUsedClientID, reloaded.LastUsedIP)
	}
}

func TestAccessGrantExpiredAndRevoked(t *testing.T) {
	db := newGrantTestDB(t)
	svc := NewDeviceAccessGrantService(db)
	ctx := context.Background()

	expired, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30, Code: "EXPIRED1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := db.Model(&models.DeviceAccessGrant{}).Where("id = ?", expired.ID).
		Update("expires_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("expire: %v", err)
	}
	if matched, _ := svc.Match(ctx, "dev-1", "EXPIRED1", "c", "10.0.0.1"); matched != nil {
		t.Error("an expired grant matched")
	}
	if ok, _ := svc.Consume(ctx, expired.ID, "c", "10.0.0.1"); ok {
		t.Error("an expired grant was consumed")
	}

	live, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30, Code: "REVOKEME"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Revoke(ctx, "dev-1", live.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if matched, _ := svc.Match(ctx, "dev-1", "REVOKEME", "c", "10.0.0.1"); matched != nil {
		t.Error("a revoked grant matched")
	}
	if ok, _ := svc.Consume(ctx, live.ID, "c", "10.0.0.1"); ok {
		t.Error("a revoked grant was consumed")
	}
	// Revoking through the wrong device is a no-op, not a cross-device revoke.
	other, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30, Code: "OTHERDEV"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Revoke(ctx, "dev-2", other.ID); err != nil {
		t.Fatalf("revoke from another device: %v", err)
	}
	if matched, _ := svc.Match(ctx, "dev-1", "OTHERDEV", "c", "10.0.0.1"); matched == nil {
		t.Error("revoking through another device should not revoke the grant")
	}
}

// TestAccessGrantConsumeIsAtomic covers the race that matters for a single-use
// link: two controllers redeeming the last slot must not both win.
func TestAccessGrantConsumeIsAtomic(t *testing.T) {
	db := newGrantTestDB(t)
	svc := NewDeviceAccessGrantService(db)
	ctx := context.Background()

	row, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{
		ExpiresInMinutes: 30, MaxUses: 1, Code: "RACEONCE",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	const workers = 8
	var mu sync.Mutex
	wins := 0
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := svc.Consume(ctx, row.ID, "controller-1", "10.0.0.1")
			if err != nil {
				t.Errorf("consume: %v", err)
				return
			}
			if ok {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Errorf("%d concurrent consumes won, want exactly 1", wins)
	}

	var reloaded models.DeviceAccessGrant
	if err := db.First(&reloaded, row.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Uses != 1 {
		t.Errorf("uses = %d, want 1", reloaded.Uses)
	}
}

// An unbound grant keeps the original "anyone holding the code" behaviour, so
// existing rows and clients are unaffected by the binding fields.
func TestGrantBindingAllowsEmptyAdmitsAnyone(t *testing.T) {
	grant := models.DeviceAccessGrant{}
	if !GrantBindingAllows(grant, "", "") {
		t.Fatal("an unbound grant must admit any controller")
	}
	if !GrantBindingAllows(grant, "someone", "203.0.113.9") {
		t.Fatal("an unbound grant must admit a named controller too")
	}
}

func TestGrantBindingAllowsClientAndIP(t *testing.T) {
	tests := []struct {
		name     string
		grant    models.DeviceAccessGrant
		clientID string
		clientIP string
		want     bool
	}{
		{"client id matches", models.DeviceAccessGrant{BoundClientID: "guest-1"}, "guest-1", "203.0.113.9", true},
		{"client id differs", models.DeviceAccessGrant{BoundClientID: "guest-1"}, "guest-2", "203.0.113.9", false},
		// client_id is an opaque token, so it is compared exactly rather than
		// case-folded; "GUEST-1" is a different client.
		{"client id is case sensitive", models.DeviceAccessGrant{BoundClientID: "guest-1"}, "GUEST-1", "203.0.113.9", false},
		{"client id is trimmed", models.DeviceAccessGrant{BoundClientID: " guest-1 "}, "guest-1", "203.0.113.9", true},
		{"ip inside the cidr", models.DeviceAccessGrant{BoundIPs: `["10.211.55.0/24"]`}, "guest-1", "10.211.55.4", true},
		{"ip outside the cidr", models.DeviceAccessGrant{BoundIPs: `["10.211.55.0/24"]`}, "guest-1", "203.0.113.9", false},
		{"exact ip", models.DeviceAccessGrant{BoundIPs: `["203.0.113.9"]`}, "guest-1", "203.0.113.9", true},
		// A bound grant fails closed when the address cannot be parsed.
		{"unparseable ip is refused", models.DeviceAccessGrant{BoundIPs: `["10.0.0.0/8"]`}, "guest-1", "not-an-ip", false},
		{"both halves must match", models.DeviceAccessGrant{BoundClientID: "guest-1", BoundIPs: `["10.0.0.0/8"]`}, "guest-1", "10.1.2.3", true},
		{"wrong client with right ip", models.DeviceAccessGrant{BoundClientID: "guest-1", BoundIPs: `["10.0.0.0/8"]`}, "guest-2", "10.1.2.3", false},
		{"right client with wrong ip", models.DeviceAccessGrant{BoundClientID: "guest-1", BoundIPs: `["10.0.0.0/8"]`}, "guest-1", "203.0.113.9", false},
		{"empty unparseable json means unbound", models.DeviceAccessGrant{BoundIPs: "[]"}, "guest-1", "203.0.113.9", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GrantBindingAllows(tt.grant, tt.clientID, tt.clientIP); got != tt.want {
				t.Fatalf("GrantBindingAllows = %v, want %v", got, tt.want)
			}
		})
	}
}

// Match enforces the binding: the same code is a plain invalid code for a
// controller the grant was not issued to.
func TestAccessGrantBindingIsEnforced(t *testing.T) {
	db := newGrantTestDB(t)
	svc := NewDeviceAccessGrantService(db)
	ctx := context.Background()

	row, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{
		ExpiresInMinutes: 30,
		Code:             "BOUNDME1",
		BoundClientID:    "guest-1",
		BoundIPs:         []string{"10.211.55.0/24"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if row.BoundClientID != "guest-1" || row.BoundIPs != `["10.211.55.0/24"]` {
		t.Fatalf("binding not stored: %+v", row)
	}

	if matched, _ := svc.Match(ctx, "dev-1", "BOUNDME1", "guest-1", "10.211.55.4"); matched == nil {
		t.Fatal("the bound guest should match")
	}
	if matched, _ := svc.Match(ctx, "dev-1", "BOUNDME1", "guest-2", "10.211.55.4"); matched != nil {
		t.Error("a different client_id must not redeem a client-bound grant")
	}
	if matched, _ := svc.Match(ctx, "dev-1", "BOUNDME1", "guest-1", "203.0.113.9"); matched != nil {
		t.Error("an address outside bound_ips must not redeem the grant")
	}
}

func TestCreateAccessGrantValidatesBinding(t *testing.T) {
	db := newGrantTestDB(t)
	svc := NewDeviceAccessGrantService(db)
	ctx := context.Background()
	publicKey, _ := testControllerKey(t)

	if _, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{
		ExpiresInMinutes: 30, Code: "BADBIND1", BoundIPs: []string{"example.com"},
	}); !errors.Is(err, ErrInvalidAccessGrant) {
		t.Fatalf("a hostname in bound_ips should be rejected, got %v", err)
	}
	if _, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{
		ExpiresInMinutes: 30, Code: "BADBIND2", BoundClientID: strings.Repeat("x", 65),
	}); !errors.Is(err, ErrInvalidAccessGrant) {
		t.Fatalf("an over-long bound_client_id should be rejected, got %v", err)
	}
	// A trailing-space-only client id is "no binding", not a 65-char id.
	row, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{
		ExpiresInMinutes: 30, Code: "OKBIND01", BoundClientID: "   ",
	})
	if err != nil {
		t.Fatalf("create with blank binding: %v", err)
	}
	if row.BoundClientID != "" {
		t.Fatalf("blank binding should normalise to empty, got %q", row.BoundClientID)
	}

	// A proof-bound grant needs both halves: without the client id the binding
	// and the audit value would disagree, and a key that is not a raw Ed25519
	// public key can never be satisfied.
	if _, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{
		ExpiresInMinutes: 30, Code: "BADKEY01", BoundClientPublicKey: publicKey,
	}); !errors.Is(err, ErrInvalidAccessGrant) {
		t.Fatalf("a public key without a client id should be rejected, got %v", err)
	}
	if _, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{
		ExpiresInMinutes: 30, Code: "BADKEY02", BoundClientID: "guest-1",
		BoundClientPublicKey: "not-a-key",
	}); !errors.Is(err, ErrInvalidAccessGrant) {
		t.Fatalf("a malformed public key should be rejected, got %v", err)
	}
	bound, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{
		ExpiresInMinutes: 30, Code: "KEYBIND1", BoundClientID: "guest-1",
		BoundClientPublicKey: publicKey,
	})
	if err != nil {
		t.Fatalf("create with a bound key: %v", err)
	}
	if bound.BoundClientPublicKey != publicKey {
		t.Fatalf("bound key stored as %q, want %q", bound.BoundClientPublicKey, publicKey)
	}
}
