package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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

var phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

const (
	smsCodeTTL     = 5 * time.Minute
	smsRateTTL     = 60 * time.Second
	smsDailyTTL    = 24 * time.Hour
	smsDailyLimit  = 10
	smsMaxAttempts = 3
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
	fingerprint [32]byte // hash of credentials to detect changes
}

func NewSmsService(rdb *redis.Client, settings SmsSettingsProvider) *SmsService {
	s := &SmsService{rdb: rdb, settings: settings}
	if settings.IsSmsEnabled() {
		s.ensureClient()
	}
	return s
}

func (s *SmsService) IsEnabled() bool {
	return s.settings.IsSmsEnabled()
}

// ensureClient lazily creates or recreates the Aliyun client when credentials change.
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
		log.Printf("[SmsService] Failed to create Aliyun SMS client: %v", err)
		s.smsClient = nil
		return nil
	}

	s.smsClient = client
	s.fingerprint = fp
	log.Println("[SmsService] Aliyun SMS client (re)initialized")
	return client
}

func ValidatePhone(phone string) bool {
	return phoneRegex.MatchString(phone)
}

func (s *SmsService) SendCode(ctx context.Context, phone string) error {
	if !s.IsEnabled() {
		return fmt.Errorf("SMS service is not enabled")
	}

	client := s.ensureClient()
	if client == nil {
		return fmt.Errorf("SMS service initialization failed")
	}

	rateKey := fmt.Sprintf("sms_rate:%s", phone)
	if s.rdb.Exists(ctx, rateKey).Val() > 0 {
		return fmt.Errorf("发送太频繁，请稍后再试")
	}

	dailyKey := fmt.Sprintf("sms_daily:%s", phone)
	count, _ := s.rdb.Get(ctx, dailyKey).Int()
	if count >= smsDailyLimit {
		return fmt.Errorf("今日验证码发送次数已达上限")
	}

	code := fmt.Sprintf("%04d", rand.Intn(10000))

	templateParam, _ := json.Marshal(map[string]string{"code": code})
	req := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(s.settings.GetSmsSignName()),
		TemplateCode:  tea.String(s.settings.GetSmsTemplateCode()),
		TemplateParam: tea.String(string(templateParam)),
	}

	resp, err := client.SendSms(req)
	if err != nil {
		log.Printf("[SmsService] Aliyun SendSms error: %v", err)
		return fmt.Errorf("短信发送失败")
	}
	if resp.Body != nil && resp.Body.Code != nil && *resp.Body.Code != "OK" {
		log.Printf("[SmsService] Aliyun SendSms rejected: code=%s msg=%s", tea.StringValue(resp.Body.Code), tea.StringValue(resp.Body.Message))
		return fmt.Errorf("短信发送失败: %s", tea.StringValue(resp.Body.Message))
	}

	codeKey := fmt.Sprintf("sms_code:%s", phone)
	data, _ := json.Marshal(smsCodeData{Code: code, Attempts: 0})
	s.rdb.Set(ctx, codeKey, string(data), smsCodeTTL)

	s.rdb.Set(ctx, rateKey, "1", smsRateTTL)

	pipe := s.rdb.Pipeline()
	pipe.Incr(ctx, dailyKey)
	pipe.Expire(ctx, dailyKey, smsDailyTTL)
	pipe.Exec(ctx)

	log.Printf("[SmsService] Code sent to %s", phone)
	return nil
}

func (s *SmsService) VerifyCode(ctx context.Context, phone, code string) error {
	codeKey := fmt.Sprintf("sms_code:%s", phone)
	val, err := s.rdb.Get(ctx, codeKey).Result()
	if err != nil {
		return fmt.Errorf("验证码已过期，请重新获取")
	}

	var data smsCodeData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		s.rdb.Del(ctx, codeKey)
		return fmt.Errorf("验证码已过期，请重新获取")
	}

	if data.Attempts >= smsMaxAttempts {
		s.rdb.Del(ctx, codeKey)
		return fmt.Errorf("错误次数过多，请重新获取验证码")
	}

	if data.Code != code {
		data.Attempts++
		updated, _ := json.Marshal(data)
		ttl := s.rdb.TTL(ctx, codeKey).Val()
		if ttl > 0 {
			s.rdb.Set(ctx, codeKey, string(updated), ttl)
		}
		remaining := smsMaxAttempts - data.Attempts
		if remaining <= 0 {
			s.rdb.Del(ctx, codeKey)
			return fmt.Errorf("错误次数过多，请重新获取验证码")
		}
		return fmt.Errorf("验证码错误，还可尝试%d次", remaining)
	}

	s.rdb.Del(ctx, codeKey)
	return nil
}
