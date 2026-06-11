package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"sync"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/redis/go-redis/v9"
)

// SmsScene is the high-level reason a verification code is being requested.
// We bind a code to (phone, scene) so a code minted for "login" can never
// be used to satisfy "reset_password" verification.
type SmsScene string

const (
	SmsSceneLogin         SmsScene = "login"
	SmsSceneRegister      SmsScene = "register"
	SmsSceneResetPassword SmsScene = "reset_password"
	SmsSceneBindPhone     SmsScene = "bind_phone"
)

func ValidScene(s string) bool {
	switch SmsScene(s) {
	case SmsSceneLogin, SmsSceneRegister, SmsSceneResetPassword, SmsSceneBindPhone:
		return true
	}
	return false
}

var phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

func ValidatePhone(phone string) bool {
	return phoneRegex.MatchString(phone)
}

// Rate-limit constants per §2.16.
const (
	smsCodeTTL     = 5 * time.Minute
	smsRateMinute  = time.Minute   // ≤1 per minute
	smsRateBurst   = 10 * time.Minute
	smsBurstLimit  = 3
	smsDailyTTL    = 24 * time.Hour
	smsDailyLimit  = 10
	smsMaxAttempts = 3
)

// Sentinel errors so handlers can map to RFC7807 codes.
var (
	ErrSmsRateLimit = errors.New("sms send rate exceeded")
	ErrSmsDaily     = errors.New("sms daily limit exceeded")
	ErrSmsCodeExpired = errors.New("sms code expired")
	ErrSmsCodeWrong   = errors.New("sms code mismatch")
	ErrSmsCodeAttempts = errors.New("too many sms verification attempts")
	ErrSmsDisabled  = errors.New("sms not configured")
)

type smsCodeData struct {
	Code     string `json:"code"`
	Attempts int    `json:"attempts"`
}

// SmsSettingsProvider is the interface SmsService needs to read live SMS config.
type SmsSettingsProvider interface {
	GetSmsAccessKeyID() string
	GetSmsAccessKeySecret() string
	GetSmsSignName() string
	GetSmsTemplateCode() string
	IsSmsEnabled() bool
}

type SmsService struct {
	rdb      *redis.Client
	settings SmsSettingsProvider

	mu          sync.Mutex
	smsClient   *dysmsapi.Client
	fingerprint [32]byte
}

func NewSmsService(rdb *redis.Client, settings SmsSettingsProvider) *SmsService {
	s := &SmsService{rdb: rdb, settings: settings}
	if settings.IsSmsEnabled() {
		s.ensureClient()
	}
	return s
}

func (s *SmsService) IsEnabled() bool { return s.settings.IsSmsEnabled() }

// ensureClient lazily creates / recreates the Aliyun client when credentials change.
func (s *SmsService) ensureClient() *dysmsapi.Client {
	keyID := s.settings.GetSmsAccessKeyID()
	keySecret := s.settings.GetSmsAccessKeySecret()
	fp := sha256.Sum256([]byte(keyID + "|" + keySecret))

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.smsClient != nil && s.fingerprint == fp {
		return s.smsClient
	}
	cfg := &openapi.Config{
		AccessKeyId:     tea.String(keyID),
		AccessKeySecret: tea.String(keySecret),
		Endpoint:        tea.String("dysmsapi.aliyuncs.com"),
	}
	client, err := dysmsapi.NewClient(cfg)
	if err != nil {
		log.Printf("[SmsService] Failed to init Aliyun client: %v", err)
		s.smsClient = nil
		return nil
	}
	s.smsClient = client
	s.fingerprint = fp
	log.Println("[SmsService] Aliyun SMS client (re)initialised")
	return client
}

// Redis key helpers (qd: namespace, §S4/S5).

func (s *SmsService) codeKey(phone string, scene SmsScene) string {
	return fmt.Sprintf("qd:sms:code:%s:%s", phone, scene)
}
func (s *SmsService) rateKey(phone string) string {
	return fmt.Sprintf("qd:sms:rate:%s", phone)
}
func (s *SmsService) burstKey(phone string) string {
	return fmt.Sprintf("qd:sms:burst:%s", phone)
}
func (s *SmsService) dailyKey(phone string) string {
	return fmt.Sprintf("qd:sms:daily:%s", phone)
}

// SendCode dispatches a 6-digit code via Aliyun and stores it under
// (phone, scene). Returns ErrSmsRateLimit / ErrSmsDaily if limits trip.
func (s *SmsService) SendCode(ctx context.Context, phone string, scene SmsScene) error {
	if !s.IsEnabled() {
		return ErrSmsDisabled
	}
	client := s.ensureClient()
	if client == nil {
		return ErrSmsDisabled
	}

	// 1-minute rate limit.
	if s.rdb.Exists(ctx, s.rateKey(phone)).Val() > 0 {
		return ErrSmsRateLimit
	}
	// 10-minute burst limit.
	burstCount, _ := s.rdb.Get(ctx, s.burstKey(phone)).Int()
	if burstCount >= smsBurstLimit {
		return ErrSmsRateLimit
	}
	// 24-hour cap.
	dailyCount, _ := s.rdb.Get(ctx, s.dailyKey(phone)).Int()
	if dailyCount >= smsDailyLimit {
		return ErrSmsDaily
	}

	code := fmt.Sprintf("%06d", rand.Intn(1_000_000))
	tplParam, _ := json.Marshal(map[string]string{"code": code})
	req := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(s.settings.GetSmsSignName()),
		TemplateCode:  tea.String(s.settings.GetSmsTemplateCode()),
		TemplateParam: tea.String(string(tplParam)),
	}
	resp, err := client.SendSms(req)
	if err != nil {
		log.Printf("[SmsService] Aliyun SendSms error: %v", err)
		return fmt.Errorf("send sms: %w", err)
	}
	if resp.Body != nil && resp.Body.Code != nil && *resp.Body.Code != "OK" {
		log.Printf("[SmsService] Aliyun rejected: code=%s msg=%s",
			tea.StringValue(resp.Body.Code), tea.StringValue(resp.Body.Message))
		return fmt.Errorf("aliyun rejected: %s", tea.StringValue(resp.Body.Message))
	}

	// Persist code + bump rate counters.
	body, _ := json.Marshal(smsCodeData{Code: code})
	s.rdb.Set(ctx, s.codeKey(phone, scene), string(body), smsCodeTTL)
	s.rdb.Set(ctx, s.rateKey(phone), "1", smsRateMinute)

	pipe := s.rdb.Pipeline()
	pipe.Incr(ctx, s.burstKey(phone))
	pipe.Expire(ctx, s.burstKey(phone), smsRateBurst)
	pipe.Incr(ctx, s.dailyKey(phone))
	pipe.Expire(ctx, s.dailyKey(phone), smsDailyTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[SmsService] rate counter exec failed: %v", err)
	}

	log.Printf("[SmsService] Code sent to %s (scene=%s)", phone, scene)
	return nil
}

// VerifyCode validates the code for (phone, scene). On match, the code is
// deleted (single-use). On mismatch, the attempt counter is bumped; after
// smsMaxAttempts the code is invalidated.
func (s *SmsService) VerifyCode(ctx context.Context, phone string, scene SmsScene, code string) error {
	key := s.codeKey(phone, scene)
	val, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return ErrSmsCodeExpired
	}
	var data smsCodeData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		s.rdb.Del(ctx, key)
		return ErrSmsCodeExpired
	}
	if data.Attempts >= smsMaxAttempts {
		s.rdb.Del(ctx, key)
		return ErrSmsCodeAttempts
	}
	if data.Code != code {
		data.Attempts++
		updated, _ := json.Marshal(data)
		ttl := s.rdb.TTL(ctx, key).Val()
		if ttl > 0 {
			s.rdb.Set(ctx, key, string(updated), ttl)
		}
		if data.Attempts >= smsMaxAttempts {
			s.rdb.Del(ctx, key)
			return ErrSmsCodeAttempts
		}
		return ErrSmsCodeWrong
	}
	s.rdb.Del(ctx, key)
	return nil
}
