package service

import (
	"log"
	"strings"
	"sync"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// SettingsService provides thread-safe access to system settings with in-memory cache.
// Dynamic settings (ICE, security) are read from the cache, which is refreshed
// whenever settings are updated via the admin API.
type SettingsService struct {
	db    *gorm.DB
	mu    sync.RWMutex
	cache *models.Settings
}

func NewSettingsService(db *gorm.DB) *SettingsService {
	s := &SettingsService{db: db}
	s.Reload()
	return s
}

// Reload reads settings from DB into cache.
func (s *SettingsService) Reload() {
	var settings models.Settings
	if err := s.db.First(&settings).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			settings = models.Settings{
				SiteEnabled:       true,
				SiteName:          "QuickDesk",
				TurnCredentialTTL: 86400,
			}
		} else {
			log.Printf("Failed to load settings: %v", err)
			return
		}
	}
	s.mu.Lock()
	s.cache = &settings
	s.mu.Unlock()
}

// Get returns a snapshot of the current settings.
func (s *SettingsService) Get() models.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil {
		return models.Settings{SiteEnabled: true, SiteName: "QuickDesk", TurnCredentialTTL: 86400}
	}
	return *s.cache
}

// Save persists settings to DB and refreshes the cache.
func (s *SettingsService) Save(settings *models.Settings) error {
	var existing models.Settings
	result := s.db.First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		if err := s.db.Create(settings).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	} else {
		settings.ID = existing.ID
		if err := s.db.Save(settings).Error; err != nil {
			return err
		}
	}

	s.mu.Lock()
	s.cache = settings
	s.mu.Unlock()
	return nil
}

// EnvSeed holds all .env values for first-run seeding.
type EnvSeed struct {
	TurnURLs, TurnAuthSecret       string
	TurnTTL                        int
	StunURLs, APIKey, AllowedOrigins string
	SmsKeyID, SmsKeySecret         string
	SmsSignName, SmsTemplateCode   string
}

// SeedFromEnv initializes DB with .env values if settings row doesn't exist yet.
func (s *SettingsService) SeedFromEnv(seed EnvSeed) {
	var count int64
	s.db.Model(&models.Settings{}).Count(&count)
	if count > 0 {
		return
	}

	log.Println("Seeding settings from .env configuration...")
	settings := &models.Settings{
		SiteEnabled:        true,
		SiteName:           "QuickDesk",
		TurnURLs:           seed.TurnURLs,
		TurnAuthSecret:     seed.TurnAuthSecret,
		TurnCredentialTTL:  seed.TurnTTL,
		StunURLs:           seed.StunURLs,
		APIKey:             seed.APIKey,
		AllowedOrigins:     seed.AllowedOrigins,
		SmsAccessKeyID:     seed.SmsKeyID,
		SmsAccessKeySecret: seed.SmsKeySecret,
		SmsSignName:        seed.SmsSignName,
		SmsTemplateCode:    seed.SmsTemplateCode,
	}
	if err := s.Save(settings); err != nil {
		log.Printf("Failed to seed settings: %v", err)
	}
}

// GetTurnURLs returns TURN URLs as a slice.
func (s *SettingsService) GetTurnURLs() []string {
	return splitLines(s.Get().TurnURLs)
}

func (s *SettingsService) GetTurnAuthSecret() string {
	return s.Get().TurnAuthSecret
}

func (s *SettingsService) GetTurnCredentialTTL() int {
	ttl := s.Get().TurnCredentialTTL
	if ttl <= 0 {
		return 86400
	}
	return ttl
}

// GetStunURLs returns STUN URLs as a slice.
func (s *SettingsService) GetStunURLs() []string {
	return splitLines(s.Get().StunURLs)
}

func (s *SettingsService) GetAPIKey() string {
	return s.Get().APIKey
}

func (s *SettingsService) GetAllowedOrigins() []string {
	return splitLines(s.Get().AllowedOrigins)
}

// SMS getters

func (s *SettingsService) GetSmsAccessKeyID() string     { return s.Get().SmsAccessKeyID }
func (s *SettingsService) GetSmsAccessKeySecret() string { return s.Get().SmsAccessKeySecret }
func (s *SettingsService) GetSmsSignName() string        { return s.Get().SmsSignName }
func (s *SettingsService) GetSmsTemplateCode() string    { return s.Get().SmsTemplateCode }

func (s *SettingsService) IsSmsEnabled() bool {
	st := s.Get()
	return st.SmsAccessKeyID != "" && st.SmsAccessKeySecret != "" &&
		st.SmsSignName != "" && st.SmsTemplateCode != ""
}

// splitLines splits a string by newlines or commas into trimmed non-empty parts.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	// Support both newline-separated (from UI) and comma-separated (from .env)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, ",", "\n")
	parts := strings.Split(s, "\n")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
