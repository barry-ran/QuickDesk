package config

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var envFile string

func init() {
	flag.StringVar(&envFile, "env", "", "Path to env config file (default: .env in current directory)")
}

type IceConfig struct {
	TurnURLs      []string
	AuthSecret    string // shared secret with coturn (use-auth-secret mode)
	CredentialTTL int    // TURN credential TTL in seconds
	StunURLs      []string
}

type AdminConfig struct {
	User     string
	Password string
}

type SecurityConfig struct {
	APIKey         string   // API Key for client authentication; empty means disabled
	AllowedOrigins []string // Allowed Origin domains for WebClient; empty means disabled
	// TrustedProxies are the peers whose X-Forwarded-For header is believed.
	// Everyone else's header is ignored and the socket address is used, which
	// is what stops a client from choosing its own client_ip.
	TrustedProxies []string
}

// MetricsConfig controls the optional Prometheus text endpoint. It is disabled
// unless a token is configured, so a self-hosted install does not accidentally
// expose operational counters on the public listener.
type MetricsConfig struct {
	Token string
}

// DefaultTrustedProxies is the reverse-proxy topology a self-hosted install
// normally has: the proxy is on the same host or on the LAN. Only these peers
// may set X-Forwarded-For; a request arriving from anywhere else is attributed
// to its socket address no matter what the header says.
var DefaultTrustedProxies = []string{
	"127.0.0.1/32", "::1/128",
	"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
	"fc00::/7",
}

// parseTrustedProxies reads TRUSTED_PROXIES. Empty means the defaults above;
// the literal `none` means trust nobody (every request is attributed to its
// socket address), which is the right setting for a server exposed directly.
func parseTrustedProxies(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return append([]string(nil), DefaultTrustedProxies...)
	}
	if strings.EqualFold(trimmed, "none") {
		return nil
	}
	out := make([]string, 0, 4)
	for _, part := range strings.Split(trimmed, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// HolidayConfig points device access policies at the statutory holiday
// calendar they resolve `holiday` exception entries against. Empty Dir keeps
// the calendar embedded in the binary.
type HolidayConfig struct {
	Dir string
	// URL, when set, makes the server download the holiday-cn year files
	// instead of waiting for someone to copy next year's notice in. Either a
	// template containing `{year}` or a base URL that `/<year>.json` is
	// appended to; a cache directory (Dir) is required alongside it.
	URL string
	// RefreshMinutes is how often the fetcher re-checks. 0 fetches once at
	// startup and never again.
	RefreshMinutes int
}

// RecordingConfig controls where uploaded session-recording artifacts land and
// how large a single upload may be. The directory is deliberately separate from
// the log directory and is never mounted as a static site.
type RecordingConfig struct {
	Dir              string
	MaxUploadMB      int
	URLSecret        string
	TicketTTLSeconds int
	StaleMinutes     int
	RetentionDays    int
	SweepIntervalMin int
}

type SmsConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	Enabled         bool // true when all required fields are configured
}

// UpdateConfig controls the self-hosted update channel's optional artifact
// re-hosting. The index works without it - a release can point at a URL the
// operator hosts - but when Dir is set the server can hold the installer itself
// and hand out its own download URL.
type UpdateConfig struct {
	// ArtifactDir is where re-hosted installers land. Like the recording store
	// it is never mounted as a static site; bytes only leave through the
	// API-key-gated download route.
	ArtifactDir string
	// MaxUploadMB caps a single installer upload.
	MaxUploadMB int
	// PublicBaseURL is the absolute prefix used to build the client-facing
	// download_url for a re-hosted artifact, e.g. https://updates.example.com.
	// Empty falls back to the scheme and Host of the admin upload request, which
	// is only right for a deployment reached at one name without TLS
	// termination in front of it.
	PublicBaseURL string
	// StaleUploadMinutes is how long an abandoned resumable-upload staging file
	// (`.part`) may sit before the sweep deletes it. A paused upload older than
	// this has to restart from zero, which is the visible cost of the window.
	StaleUploadMinutes int
	// SweepIntervalMin is how often the cleanup pass runs. 0 disables the worker;
	// the admin trigger still works.
	SweepIntervalMin int
}

type Config struct {
	Server        ServerConfig   `mapstructure:"server"`
	Database      DatabaseConfig `mapstructure:"database"`
	Redis         RedisConfig    `mapstructure:"redis"`
	Ice           IceConfig
	Admin         AdminConfig
	Security      SecurityConfig
	Metrics       MetricsConfig
	Recording     RecordingConfig
	Notification  NotificationConfig
	Audit         AuditConfig
	InstanceLog   InstanceLogConfig
	OfflineAlert  OfflineAlertConfig
	ScheduledTask ScheduledTaskConfig
	Approval      ApprovalConfig
	Holiday       HolidayConfig
	Sms           SmsConfig
	Update        UpdateConfig
	Runtime       RuntimeConfig
}

// InstanceLogConfig controls shipping this process's log lines into the shared
// database, which is what lets one search cover every instance in a
// multi-instance deployment. It is off by default: shipping writes rows, so a
// single-instance deployment should not pay for them.
type InstanceLogConfig struct {
	// Sink enables shipping.
	Sink bool
	// RetentionDays bounds how long shipped lines are kept. The default is
	// short on purpose - this table is a convenience copy for cross-instance
	// search, not evidence; the authoritative log is the rotating file on the
	// instance that wrote it.
	RetentionDays int
	// SweepIntervalMin is how often retention runs. 0 disables the sweep.
	SweepIntervalMin int
}

// NotificationConfig controls how long the user-notification outbox keeps
// terminal delivery rows. RetentionDays = 0 keeps history forever.
type NotificationConfig struct {
	RetentionDays    int
	SweepIntervalMin int
	// ArchiveDir, when set, makes retention append expired rows to a daily
	// JSONL file there before deleting them. Empty = plain delete.
	ArchiveDir string
	// ArchiveColdDir, when set, moves expired daily archive files there instead
	// of deleting them. Empty keeps the old delete-on-rotation behavior.
	ArchiveColdDir string
	// ArchiveRetentionDays bounds how long those archive files themselves are
	// kept. 0 = keep them forever (the safe default: the archive exists so a
	// purged delivery stays recoverable, so deleting it has to be asked for).
	ArchiveRetentionDays int
	// AggregateSeconds collapses bursts: a delivery is held for this long and
	// matching events that arrive during the hold increment its occurrence
	// count instead of becoming their own rows. 0 disables it, which keeps the
	// historical "one row per event, delivered immediately" behaviour.
	AggregateSeconds int
}

// AuditConfig controls how long the `audit_logs` table keeps entries. The table
// only ever grows otherwise - it has no natural terminal state - so retention is
// the only thing that bounds it. RetentionDays = 0 keeps every row forever.
type AuditConfig struct {
	RetentionDays    int
	SweepIntervalMin int
	// MaxRows, when positive, is a hard row cap independent of age: the sweep
	// trims the oldest rows until at most this many remain. It exists because a
	// quiet deployment can still accumulate rows faster than any single age
	// window implies, and because the age window alone cannot answer "how big
	// is this table allowed to get". 0 = no cap.
	MaxRows int
	// MaxRowsPerActor, when positive, caps how many rows any single actor may
	// hold. The table-wide cap alone does not stop one noisy actor - a user
	// whose failing token refreshes log in a loop, an admin running a bulk
	// migration - from filling the table and evicting every other actor's
	// trail. 0 = no per-actor cap.
	MaxRowsPerActor int
	// MaxRowsPerDevice, when positive, caps how many rows any single device may
	// hold. Device ids come from resource_id on both explicit device actions
	// and device-scoped events; batch operations that do not name one device
	// remain governed by the action and table caps. 0 = no per-device cap.
	MaxRowsPerDevice int
	// MaxRowsPerAction, when positive, caps how many rows any single action may
	// hold. This catches one action type - a failing login, a noisy export -
	// dominating the trail even when its actors are otherwise quiet. 0 = no
	// per-action cap.
	MaxRowsPerAction int
	// ArchiveDir, when set, makes retention append expired rows to a daily
	// JSONL file there before deleting them. Empty = plain delete. Audit rows
	// are evidence, so an operator turning retention on should normally set it.
	ArchiveDir string
	// ArchiveColdDir, when set, moves expired daily archive files there instead
	// of deleting them. Empty keeps the old delete-on-rotation behavior.
	ArchiveColdDir string
	// ArchiveRetentionDays bounds how long those archive files themselves are
	// kept. 0 = keep them forever (the safe default: archiving exists so a
	// purged row stays recoverable, so deleting the archives has to be asked
	// for). Only files named audit-logs-YYYY-MM-DD.jsonl are touched.
	ArchiveRetentionDays int
}

// OfflineAlertConfig controls the background evaluator behind per-device
// "tell me when this stays offline" rules. IntervalSeconds = 0 disables it.
//
// The interval is a trade-off against the heartbeat TTL (90s): a device is
// only considered offline once the heartbeat key expires, so evaluating much
// faster than that only burns queries.
type OfflineAlertConfig struct {
	IntervalSeconds int
}

// ScheduledTaskConfig controls the worker behind owner-scheduled control-plane
// actions ("sign out everywhere at 22:00"). The default interval is 30s;
// IntervalSeconds = 0 disables the worker for deployments that never schedule
// anything.
type ScheduledTaskConfig struct {
	IntervalSeconds int
}

// ApprovalConfig tunes the second-factor approval queue.
type ApprovalConfig struct {
	// DenialCooldownSeconds is how long a source whose request the owner denied
	// must wait before asking the same device again. 0 disables it, letting a
	// denial be followed by another request immediately - the historical
	// behaviour, and the abuse the cooldown exists to close.
	DenialCooldownSeconds int
	// DenialCooldownMultiplier makes each further consecutive refusal cost
	// more: a source with a streak of n denials waits
	// DenialCooldownSeconds * multiplier^(n-1). 1 keeps the window constant,
	// which is why it is the default.
	DenialCooldownMultiplier int
	// DenialCooldownMaxSeconds caps that escalation. 0 means "no ceiling", so
	// only the multiplier bounds the window.
	DenialCooldownMaxSeconds int
}

type RuntimeConfig struct {
	RunAutoMigrate  bool
	PprofAddr       string
	LogDir          string
	LogMaxSizeMB    int
	LogMaxBackups   int
	PostgresLogFile string
	RedisLogFile    string
	// ControllerChallengeSecret signs the proof-of-possession challenges that
	// a key-bound access grant requires. It falls back to the API key, so a
	// deployment that already has one needs no new setting.
	ControllerChallengeSecret string
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
}

func Load() *Config {
	if !flag.Parsed() {
		flag.Parse()
	}

	if envFile != "" {
		viper.SetConfigFile(envFile)
		viper.SetConfigType("env")
		log.Printf("Using config file: %s", envFile)
	} else {
		viper.SetConfigName(".env")
		viper.SetConfigType("env")
		viper.AddConfigPath(".")
	}
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("SERVER_HOST", "0.0.0.0")
	viper.SetDefault("SERVER_PORT", 8000)
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_USER", "quickdesk")
	viper.SetDefault("DB_PASSWORD", "quickdesk123")
	viper.SetDefault("DB_NAME", "quickdesk")
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", 6379)
	viper.SetDefault("REDIS_PASSWORD", "")

	viper.SetDefault("TURN_URLS", "")
	viper.SetDefault("TURN_AUTH_SECRET", "")
	viper.SetDefault("TURN_CREDENTIAL_TTL", 86400)
	viper.SetDefault("STUN_URLS", "")
	// NOTE: relay bandwidth is capped in coturn (`max-bps`, bytes per second),
	// not here - the signaling server only signs credentials. See
	// scripts/e2e/coturn-local.conf.example and the E2E log section 28.

	viper.SetDefault("ADMIN_USER", "admin")
	viper.SetDefault("ADMIN_PASSWORD", "admin")

	viper.SetDefault("API_KEY", "")
	viper.SetDefault("ALLOWED_ORIGINS", "")
	// Reverse proxies whose X-Forwarded-For is believed. Empty = loopback and
	// private ranges; `none` = trust nobody.
	viper.SetDefault("TRUSTED_PROXIES", "")
	// Prometheus text endpoint. Empty disables /metrics entirely; when set,
	// scrapers must send `Authorization: Bearer <token>`.
	viper.SetDefault("METRICS_TOKEN", "")

	viper.SetDefault("RECORDING_DIR", "data/recordings")
	viper.SetDefault("RECORDING_MAX_UPLOAD_MB", 2048)
	viper.SetDefault("RECORDING_URL_SECRET", "")
	viper.SetDefault("CONTROLLER_CHALLENGE_SECRET", "")
	viper.SetDefault("RECORDING_TICKET_TTL_SECONDS", 600)
	// Recordings stuck in "recording" longer than this are marked failed.
	viper.SetDefault("RECORDING_STALE_MINUTES", 240)
	// 0 keeps media forever; any positive value deletes files past the window
	// while keeping their index rows.
	viper.SetDefault("RECORDING_RETENTION_DAYS", 0)
	viper.SetDefault("RECORDING_SWEEP_INTERVAL_MINUTES", 30)

	// Self-hosted update channel: where re-hosted installers land, how large one
	// may be, and the absolute prefix used to build their download URLs.
	viper.SetDefault("UPDATE_ARTIFACT_DIR", "data/updates")
	viper.SetDefault("UPDATE_MAX_UPLOAD_MB", 4096)
	viper.SetDefault("UPDATE_PUBLIC_BASE_URL", "")
	// Abandoned staging files: the same 6h window the recording store uses, and
	// an hourly pass. 0 minutes of interval disables the worker.
	viper.SetDefault("UPDATE_STALE_UPLOAD_MINUTES", 360)
	viper.SetDefault("UPDATE_SWEEP_INTERVAL_MINUTES", 60)

	// Notification outbox retention. 0 keeps terminal delivery rows forever;
	// any positive value deletes delivered/failed rows older than the window
	// (pending/delivering rows are never touched).
	viper.SetDefault("NOTIFICATION_RETENTION_DAYS", 0)
	viper.SetDefault("NOTIFICATION_SWEEP_INTERVAL_MINUTES", 60)
	// When set, expired delivery rows are appended to a daily JSONL file here
	// before being deleted, so retention never destroys the only copy.
	viper.SetDefault("NOTIFICATION_ARCHIVE_DIR", "")
	// When set, expired daily archive files are moved here instead of deleted.
	viper.SetDefault("NOTIFICATION_ARCHIVE_COLD_DIR", "")
	// How long those daily archive files themselves are kept. 0 = forever.
	viper.SetDefault("NOTIFICATION_ARCHIVE_RETENTION_DAYS", 0)
	// Notification aggregation. 0 (the default) delivers every event as its own
	// row immediately; a positive value holds a delivery for that many seconds
	// and folds matching events into it, trading promptness for fewer messages.
	viper.SetDefault("NOTIFICATION_AGGREGATE_SECONDS", 0)

	// Audit-log retention. 0 keeps every entry forever; any positive value
	// deletes rows older than the window. AUDIT_ARCHIVE_DIR, when set, appends
	// them to a daily JSONL file first, which is the recommended posture for a
	// trail that exists to be evidence.
	viper.SetDefault("AUDIT_RETENTION_DAYS", 0)
	viper.SetDefault("AUDIT_SWEEP_INTERVAL_MINUTES", 60)
	viper.SetDefault("AUDIT_ARCHIVE_DIR", "")
	// When set, expired daily archive files are moved here instead of deleted.
	viper.SetDefault("AUDIT_ARCHIVE_COLD_DIR", "")
	// How long the daily archive files themselves are kept. 0 = forever.
	viper.SetDefault("AUDIT_ARCHIVE_RETENTION_DAYS", 0)
	// Hard row cap, independent of the age window: the sweep trims the oldest
	// rows until at most this many remain. 0 = no cap.
	viper.SetDefault("AUDIT_MAX_ROWS", 0)
	// Per-actor row cap: the sweep trims each actor's oldest rows until nobody
	// holds more than this, so one noisy actor cannot evict everyone else's
	// trail. 0 = no per-actor cap.
	viper.SetDefault("AUDIT_MAX_ROWS_PER_ACTOR", 0)
	// Per-device and per-action row caps: targeted bounds for a hot device or a
	// noisy action. Both are independent of every other audit bound.
	viper.SetDefault("AUDIT_MAX_ROWS_PER_DEVICE", 0)
	viper.SetDefault("AUDIT_MAX_ROWS_PER_ACTION", 0)

	// Cross-instance log shipping. Off by default: enabling it writes every log
	// line into the shared database so `scope=all` can search every instance.
	viper.SetDefault("INSTANCE_LOG_SINK", false)
	// How long shipped log lines are kept. Short by default - this table is a
	// convenience copy for cross-instance search, not the authoritative log.
	viper.SetDefault("INSTANCE_LOG_RETENTION_DAYS", 7)
	viper.SetDefault("INSTANCE_LOG_SWEEP_INTERVAL_MINUTES", 30)

	// Background evaluator for per-device offline alerts. 0 disables it.
	viper.SetDefault("OFFLINE_ALERT_INTERVAL_SECONDS", 30)

	// Worker behind owner-scheduled control-plane actions. 0 disables it for a
	// deployment that never schedules anything; the worker itself is one
	// indexed query per tick when no task is due.
	viper.SetDefault("SCHEDULED_TASK_INTERVAL_SECONDS", 30)

	// Second-factor approval: after the owner denies a request, the same source
	// waits this long before it may ask the same device again. The default is
	// non-zero because a denial that clears the way for the next request is
	// what turns a correct access code into a way to keep paging the owner.
	viper.SetDefault("APPROVAL_DENIAL_COOLDOWN_SECONDS", 300)

	// Repeated denials from one source can lengthen that window. The default
	// multiplier of 1 keeps the window constant: escalation is opt-in because
	// it changes how a well-behaved-but-unlucky client is treated.
	viper.SetDefault("APPROVAL_DENIAL_COOLDOWN_MULTIPLIER", 1)
	// 0 means "no ceiling"; set it whenever the multiplier is above 1, or the
	// window grows without bound as the streak grows.
	viper.SetDefault("APPROVAL_DENIAL_COOLDOWN_MAX_SECONDS", 0)

	// Statutory holiday calendar used by device access policies. Empty keeps
	// the year(s) embedded in the binary; point it at a directory of
	// holiday-cn style YYYY.json files to add years without a rebuild.
	viper.SetDefault("HOLIDAY_CALENDAR_DIR", "")
	// Set the URL to have the server fetch those year files itself (the
	// holiday-cn dataset publishes one JSON per year). 0 refresh minutes means
	// "fetch once at startup".
	viper.SetDefault("HOLIDAY_CALENDAR_URL", "")
	viper.SetDefault("HOLIDAY_CALENDAR_REFRESH_MINUTES", 0)

	// Aliyun SMS (optional – leave empty to disable phone features)
	viper.SetDefault("ALIYUN_SMS_ACCESS_KEY_ID", "")
	viper.SetDefault("ALIYUN_SMS_ACCESS_KEY_SECRET", "")
	viper.SetDefault("ALIYUN_SMS_SIGN_NAME", "")
	viper.SetDefault("ALIYUN_SMS_TEMPLATE_CODE", "")

	viper.SetDefault("RUN_AUTO_MIGRATE", false)
	viper.SetDefault("PPROF_ADDR", "")
	viper.SetDefault("LOG_DIR", "logs")
	viper.SetDefault("LOG_MAX_SIZE_MB", 50)
	viper.SetDefault("LOG_MAX_BACKUPS", 5)
	viper.SetDefault("POSTGRES_LOG_FILE", "")
	viper.SetDefault("REDIS_LOG_FILE", "")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found, using defaults and environment variables")
		} else {
			log.Fatalf("Error reading config file: %v", err)
		}
	} else {
		absPath, _ := filepath.Abs(viper.ConfigFileUsed())
		log.Printf("Loaded config from: %s", absPath)
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: viper.GetString("SERVER_HOST"),
			Port: viper.GetInt("SERVER_PORT"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetInt("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			DBName:   viper.GetString("DB_NAME"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetInt("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
		},
	}

	cfg.Ice = parseIceConfig()
	cfg.Admin = AdminConfig{
		User:     viper.GetString("ADMIN_USER"),
		Password: viper.GetString("ADMIN_PASSWORD"),
	}
	var allowedOrigins []string
	if origins := viper.GetString("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = splitAndTrim(origins)
	}
	cfg.Security = SecurityConfig{
		APIKey:         viper.GetString("API_KEY"),
		AllowedOrigins: allowedOrigins,
		TrustedProxies: parseTrustedProxies(viper.GetString("TRUSTED_PROXIES")),
	}
	cfg.Metrics = MetricsConfig{
		Token: viper.GetString("METRICS_TOKEN"),
	}
	// Playback tickets reuse the API key when no dedicated secret is set, so a
	// self-hosted deployment that already gates its API gets working playback
	// without extra configuration.
	recordingSecret := viper.GetString("RECORDING_URL_SECRET")
	if recordingSecret == "" {
		recordingSecret = cfg.Security.APIKey
	}
	cfg.Recording = RecordingConfig{
		Dir:              viper.GetString("RECORDING_DIR"),
		MaxUploadMB:      viper.GetInt("RECORDING_MAX_UPLOAD_MB"),
		URLSecret:        recordingSecret,
		TicketTTLSeconds: viper.GetInt("RECORDING_TICKET_TTL_SECONDS"),
		StaleMinutes:     viper.GetInt("RECORDING_STALE_MINUTES"),
		RetentionDays:    viper.GetInt("RECORDING_RETENTION_DAYS"),
		SweepIntervalMin: viper.GetInt("RECORDING_SWEEP_INTERVAL_MINUTES"),
	}
	cfg.Update = UpdateConfig{
		ArtifactDir:        viper.GetString("UPDATE_ARTIFACT_DIR"),
		MaxUploadMB:        viper.GetInt("UPDATE_MAX_UPLOAD_MB"),
		PublicBaseURL:      viper.GetString("UPDATE_PUBLIC_BASE_URL"),
		StaleUploadMinutes: viper.GetInt("UPDATE_STALE_UPLOAD_MINUTES"),
		SweepIntervalMin:   viper.GetInt("UPDATE_SWEEP_INTERVAL_MINUTES"),
	}
	cfg.Notification = NotificationConfig{
		RetentionDays:        viper.GetInt("NOTIFICATION_RETENTION_DAYS"),
		SweepIntervalMin:     viper.GetInt("NOTIFICATION_SWEEP_INTERVAL_MINUTES"),
		ArchiveDir:           viper.GetString("NOTIFICATION_ARCHIVE_DIR"),
		ArchiveColdDir:       viper.GetString("NOTIFICATION_ARCHIVE_COLD_DIR"),
		ArchiveRetentionDays: viper.GetInt("NOTIFICATION_ARCHIVE_RETENTION_DAYS"),
		AggregateSeconds:     viper.GetInt("NOTIFICATION_AGGREGATE_SECONDS"),
	}
	cfg.Audit = AuditConfig{
		RetentionDays:        viper.GetInt("AUDIT_RETENTION_DAYS"),
		SweepIntervalMin:     viper.GetInt("AUDIT_SWEEP_INTERVAL_MINUTES"),
		MaxRows:              viper.GetInt("AUDIT_MAX_ROWS"),
		MaxRowsPerActor:      viper.GetInt("AUDIT_MAX_ROWS_PER_ACTOR"),
		MaxRowsPerDevice:     viper.GetInt("AUDIT_MAX_ROWS_PER_DEVICE"),
		MaxRowsPerAction:     viper.GetInt("AUDIT_MAX_ROWS_PER_ACTION"),
		ArchiveDir:           viper.GetString("AUDIT_ARCHIVE_DIR"),
		ArchiveColdDir:       viper.GetString("AUDIT_ARCHIVE_COLD_DIR"),
		ArchiveRetentionDays: viper.GetInt("AUDIT_ARCHIVE_RETENTION_DAYS"),
	}

	cfg.InstanceLog = InstanceLogConfig{
		Sink:             viper.GetBool("INSTANCE_LOG_SINK"),
		RetentionDays:    viper.GetInt("INSTANCE_LOG_RETENTION_DAYS"),
		SweepIntervalMin: viper.GetInt("INSTANCE_LOG_SWEEP_INTERVAL_MINUTES"),
	}
	cfg.OfflineAlert = OfflineAlertConfig{
		IntervalSeconds: viper.GetInt("OFFLINE_ALERT_INTERVAL_SECONDS"),
	}
	cfg.ScheduledTask = ScheduledTaskConfig{
		IntervalSeconds: viper.GetInt("SCHEDULED_TASK_INTERVAL_SECONDS"),
	}
	cfg.Approval = ApprovalConfig{
		DenialCooldownSeconds:    viper.GetInt("APPROVAL_DENIAL_COOLDOWN_SECONDS"),
		DenialCooldownMultiplier: viper.GetInt("APPROVAL_DENIAL_COOLDOWN_MULTIPLIER"),
		DenialCooldownMaxSeconds: viper.GetInt("APPROVAL_DENIAL_COOLDOWN_MAX_SECONDS"),
	}
	cfg.Holiday = HolidayConfig{
		Dir:            viper.GetString("HOLIDAY_CALENDAR_DIR"),
		URL:            viper.GetString("HOLIDAY_CALENDAR_URL"),
		RefreshMinutes: viper.GetInt("HOLIDAY_CALENDAR_REFRESH_MINUTES"),
	}

	// Aliyun SMS – enabled only when all four fields are set
	smsKeyID := viper.GetString("ALIYUN_SMS_ACCESS_KEY_ID")
	smsKeySecret := viper.GetString("ALIYUN_SMS_ACCESS_KEY_SECRET")
	smsSignName := viper.GetString("ALIYUN_SMS_SIGN_NAME")
	smsTemplate := viper.GetString("ALIYUN_SMS_TEMPLATE_CODE")
	smsEnabled := smsKeyID != "" && smsKeySecret != "" && smsSignName != "" && smsTemplate != ""
	cfg.Sms = SmsConfig{
		AccessKeyID:     smsKeyID,
		AccessKeySecret: smsKeySecret,
		SignName:        smsSignName,
		TemplateCode:    smsTemplate,
		Enabled:         smsEnabled,
	}
	// Proof-of-possession challenges are signed with their own secret when one
	// is configured, and with the API key otherwise, so an existing deployment
	// can bind a grant to a key without adding a setting.
	challengeSecret := strings.TrimSpace(viper.GetString("CONTROLLER_CHALLENGE_SECRET"))
	if challengeSecret == "" {
		challengeSecret = cfg.Security.APIKey
	}
	cfg.Runtime = RuntimeConfig{
		RunAutoMigrate:            viper.GetBool("RUN_AUTO_MIGRATE"),
		PprofAddr:                 viper.GetString("PPROF_ADDR"),
		LogDir:                    viper.GetString("LOG_DIR"),
		LogMaxSizeMB:              viper.GetInt("LOG_MAX_SIZE_MB"),
		LogMaxBackups:             viper.GetInt("LOG_MAX_BACKUPS"),
		PostgresLogFile:           viper.GetString("POSTGRES_LOG_FILE"),
		RedisLogFile:              viper.GetString("REDIS_LOG_FILE"),
		ControllerChallengeSecret: challengeSecret,
	}

	apiKeyStatus := "disabled"
	if cfg.Security.APIKey != "" {
		apiKeyStatus = "enabled"
	}
	log.Printf("Loaded config: Server=%s:%d, DB=%s:%d/%s, ICE TURN=%d STUN=%d TTL=%ds, APIKey=%s, AllowedOrigins=%d, SMS=%v, AutoMigrate=%v, Pprof=%v, LogDir=%s",
		cfg.Server.Host, cfg.Server.Port,
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName,
		len(cfg.Ice.TurnURLs), len(cfg.Ice.StunURLs), cfg.Ice.CredentialTTL,
		apiKeyStatus, len(cfg.Security.AllowedOrigins), cfg.Sms.Enabled, cfg.Runtime.RunAutoMigrate, cfg.Runtime.PprofAddr != "", cfg.Runtime.LogDir)

	return cfg
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Password, c.DBName)
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func parseIceConfig() IceConfig {
	ice := IceConfig{
		CredentialTTL: viper.GetInt("TURN_CREDENTIAL_TTL"),
		AuthSecret:    viper.GetString("TURN_AUTH_SECRET"),
	}
	if urls := viper.GetString("TURN_URLS"); urls != "" {
		ice.TurnURLs = splitAndTrim(urls)
	}
	if urls := viper.GetString("STUN_URLS"); urls != "" {
		ice.StunURLs = splitAndTrim(urls)
	}
	return ice
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
