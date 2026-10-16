-- QuickDesk Signaling Server — canonical v1 schema
--
-- NOTE: tables are auto-created by GORM AutoMigrate on startup; this file
-- is the single source of truth for column types/indexes/defaults and is
-- what you'd run by hand on a fresh PostgreSQL instance.
--
-- The schema follows docs/dev/信令服务器API重构方案.md §4 exactly.
-- Cross-cutting design rules to keep in mind:
--   * `devices.online` is INTENTIONALLY ABSENT — online state is derived
--     from Redis presence keys (`qd:presence:device:{id}:hb` and
--     `qd:presence:device:{id}:ws:*`).
--   * `devices.logged_in_intent` only flips on explicit user actions
--     (bind / unbind / session logout); WebSocket churn never touches it.
--   * `devices.device_secret_hash` stores argon2id(device_secret); the
--     plaintext secret is only returned to the host once, at provision.
--   * Auth tokens (access, refresh, admin sessions, SMS codes, signal
--     tokens, rate-limit counters) live entirely in Redis — no DB rows.

-- ---------------------------------------------------------------
-- Users
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(64)  UNIQUE NOT NULL,
    phone         VARCHAR(32)  UNIQUE,
    email         VARCHAR(128) UNIQUE,
    password      VARCHAR(128) NOT NULL,           -- bcrypt hash
    level         VARCHAR(10)  NOT NULL DEFAULT 'V1',     -- V1..V5
    device_count  INTEGER      NOT NULL DEFAULT 0,        -- device quota
    channel_type  VARCHAR(20)  NOT NULL DEFAULT '全球',   -- 全球 / 中国大陆
    status        BOOLEAN      NOT NULL DEFAULT TRUE,     -- enabled
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------
-- Devices
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS devices (
    id                   BIGSERIAL PRIMARY KEY,
    device_id            VARCHAR(9)   UNIQUE NOT NULL,
    device_uuid          VARCHAR(64)  UNIQUE NOT NULL,
    device_secret_hash   VARCHAR(128) NOT NULL DEFAULT '',
    machine_fingerprint  VARCHAR(128) NOT NULL DEFAULT '',
    os                   VARCHAR(32),
    os_version           VARCHAR(32),
    app_version          VARCHAR(32),
    user_id              BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    device_name          VARCHAR(128),
    access_code          VARCHAR(32),
    logged_in            BOOLEAN      NOT NULL DEFAULT FALSE,  -- user intent; NEVER touched by WS churn
    last_seen_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_devices_user_id   ON devices(user_id);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen_at);

-- ---------------------------------------------------------------
-- User ↔ Device bindings
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_devices (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id       VARCHAR(9) NOT NULL,
    remark          VARCHAR(128),
    first_bound_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_connect_at TIMESTAMPTZ,
    connect_count   INTEGER     NOT NULL DEFAULT 0,
    status          BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_device UNIQUE (user_id, device_id)
);
CREATE INDEX IF NOT EXISTS idx_user_devices_user_id   ON user_devices(user_id);
CREATE INDEX IF NOT EXISTS idx_user_devices_device_id ON user_devices(device_id);

-- ---------------------------------------------------------------
-- Connection history
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS connection_histories (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id   VARCHAR(9)  NOT NULL,
    device_name VARCHAR(128),
    connect_ip  VARCHAR(45),
    duration    INTEGER,
    status      VARCHAR(16) NOT NULL,
    error_msg   VARCHAR(255),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_connhist_user_created ON connection_histories(user_id, created_at DESC);

-- ---------------------------------------------------------------
-- Session recording index (media artifact owned by engine)
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS session_recordings (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id        VARCHAR(9)   NOT NULL,
    device_name      VARCHAR(128),
    session_id       VARCHAR(64),
    status           VARCHAR(20)  NOT NULL,
    format           VARCHAR(20),
    artifact_path    VARCHAR(500),
    artifact_url     VARCHAR(1000),
    thumbnail_path   VARCHAR(500),
    thumbnail_url    VARCHAR(1000),
    duration_seconds INTEGER      NOT NULL DEFAULT 0,
    size_bytes       BIGINT       NOT NULL DEFAULT 0,
    started_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    ended_at         TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_session_recordings_user_created ON session_recordings(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_session_recordings_device ON session_recordings(device_id);
CREATE INDEX IF NOT EXISTS idx_session_recordings_session ON session_recordings(session_id);
-- Added after the table shipped: keep existing installs in step with the
-- CREATE TABLE above without a separate migration file.
ALTER TABLE session_recordings ADD COLUMN IF NOT EXISTS thumbnail_url VARCHAR(1000);

-- ---------------------------------------------------------------
-- Favorites
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_favorites (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id       VARCHAR(9) NOT NULL,
    device_name     VARCHAR(128),
    access_password VARCHAR(32),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_favorite UNIQUE (user_id, device_id)
);
CREATE INDEX IF NOT EXISTS idx_user_favorites_user_id ON user_favorites(user_id);

-- ---------------------------------------------------------------
-- Admin accounts
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS admin_users (
    id           BIGSERIAL PRIMARY KEY,
    username     VARCHAR(50)  UNIQUE NOT NULL,
    password     VARCHAR(255) NOT NULL,
    email        VARCHAR(100),
    role         VARCHAR(20)  NOT NULL DEFAULT 'admin',
    status       BOOLEAN      NOT NULL DEFAULT TRUE,
    totp_secret  VARCHAR(200) NOT NULL DEFAULT '',
    totp_enabled BOOLEAN      NOT NULL DEFAULT FALSE,
    last_login   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_admin_users_email ON admin_users(email);

-- ---------------------------------------------------------------
-- Audit log
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_logs (
    id              BIGSERIAL PRIMARY KEY,
    admin_id        BIGINT,
    admin_username  VARCHAR(100),
    action          VARCHAR(50)  NOT NULL,
    resource_type   VARCHAR(50),
    resource_id     VARCHAR(100),
    details         TEXT,
    ip              VARCHAR(50),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_admin_id   ON audit_logs(admin_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action     ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- End-user actors. Added after the table shipped: until this column existed an
-- audited user action (deleting a recording, changing a device policy) had no
-- way to record *who* did it, because admin_id means "which administrator".
ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS actor_user_id BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_user_id ON audit_logs(actor_user_id);

-- ---------------------------------------------------------------
-- Dynamic settings (single row)
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS settings (
    id                    BIGSERIAL PRIMARY KEY,
    site_enabled          BOOLEAN      NOT NULL DEFAULT TRUE,
    site_name             VARCHAR(100) NOT NULL DEFAULT 'QuickDesk',
    login_logo            VARCHAR(500),
    small_logo            VARCHAR(500),
    favicon               VARCHAR(500),
    turn_urls             TEXT,
    turn_auth_secret      VARCHAR(500),
    turn_credential_ttl   INTEGER      NOT NULL DEFAULT 86400,
    stun_urls             TEXT,
    turn_config_version   BIGINT       NOT NULL DEFAULT 1,
    api_key               VARCHAR(500),
    allowed_origins       TEXT,
    admin_ip_whitelist    TEXT,
    sms_access_key_id     VARCHAR(200),
    sms_access_key_secret VARCHAR(200),
    sms_sign_name         VARCHAR(100),
    sms_template_code     VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------
-- Device groups
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS device_groups (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) UNIQUE NOT NULL,
    description VARCHAR(500),
    color       VARCHAR(20),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS device_group_members (
    id         BIGSERIAL PRIMARY KEY,
    group_id   BIGINT     NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    device_id  VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_group_device UNIQUE (group_id, device_id)
);

-- ---------------------------------------------------------------
-- Device access log (owner-scoped audit of incoming session attempts)
-- ---------------------------------------------------------------
-- Written by the server at access-code verification time: a controller
-- cannot hide or forge its own attempts. `owner_id` is the device owner at
-- that moment (0 when the device is unbound); the caller is identified by
-- IP + client_id because `access-code:verify` authenticates the desktop
-- client with the API key rather than a user session.
CREATE TABLE IF NOT EXISTS device_access_logs (
    id         BIGSERIAL PRIMARY KEY,
    device_id  VARCHAR(20) NOT NULL,
    owner_id   BIGINT      NOT NULL DEFAULT 0,
    client_id  VARCHAR(64) NOT NULL DEFAULT '',
    session_id VARCHAR(64) NOT NULL DEFAULT '',
    ip         VARCHAR(50) NOT NULL DEFAULT '',
    result     VARCHAR(32) NOT NULL,
    reason     VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_device_access_logs_device_id ON device_access_logs (device_id);
CREATE INDEX IF NOT EXISTS idx_device_access_logs_owner_id ON device_access_logs (owner_id);
CREATE INDEX IF NOT EXISTS idx_device_access_logs_result ON device_access_logs (result);

-- ---------------------------------------------------------------
-- Device access policy (owner-controlled gate for new sessions)
-- ---------------------------------------------------------------
-- Enforced on `POST /v1/devices/:id/access-code:verify` before the access
-- code is compared. A missing row means "no policy", so devices that existed
-- before this feature keep accepting connections.
CREATE TABLE IF NOT EXISTS device_access_policies (
    id              BIGSERIAL PRIMARY KEY,
    device_id       VARCHAR(20) NOT NULL UNIQUE,
    owner_id        BIGINT      NOT NULL DEFAULT 0,
    remote_disabled BOOLEAN     NOT NULL DEFAULT FALSE,
    allow_ips       TEXT        NOT NULL DEFAULT '[]',
    max_sessions    INTEGER     NOT NULL DEFAULT 0,
    allowed_windows TEXT        NOT NULL DEFAULT '[]',
    timezone        VARCHAR(64) NOT NULL DEFAULT '',
    exception_dates TEXT        NOT NULL DEFAULT '[]',
    approval_required BOOLEAN   NOT NULL DEFAULT FALSE,
    approval_mode   VARCHAR(16) NOT NULL DEFAULT '',
    approval_trusted_ips TEXT   NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_device_access_policies_owner_id ON device_access_policies (owner_id);

-- Second factor: a matched access code becomes a pending owner decision.
-- Added after the table shipped, so existing installs need the ALTER too.
ALTER TABLE device_access_policies
    ADD COLUMN IF NOT EXISTS approval_required BOOLEAN NOT NULL DEFAULT FALSE;

-- Approval triggers: off / always / grant / untrusted_ip, plus the trusted
-- addresses that "untrusted_ip" exempts.
ALTER TABLE device_access_policies
    ADD COLUMN IF NOT EXISTS approval_mode VARCHAR(16) NOT NULL DEFAULT '';

ALTER TABLE device_access_policies
    ADD COLUMN IF NOT EXISTS approval_trusted_ips TEXT NOT NULL DEFAULT '[]';

-- ---------------------------------------------------------------
-- Device access grants (owner-issued, time-boxed, optionally single-use)
-- An alternative to the device's own rotating access code: the policy gate
-- still runs first, and a use is only counted once a session actually starts.
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS device_access_grants (
    id          BIGSERIAL PRIMARY KEY,
    device_id   VARCHAR(20) NOT NULL,
    owner_id    BIGINT      NOT NULL DEFAULT 0,
    code        VARCHAR(64) NOT NULL,
    label       VARCHAR(100) NOT NULL DEFAULT '',
    expires_at  TIMESTAMPTZ NOT NULL,
    max_uses    INTEGER     NOT NULL DEFAULT 0,
    uses        INTEGER     NOT NULL DEFAULT 0,
    revoked_at  TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    bound_client_id VARCHAR(64)  NOT NULL DEFAULT '',
    bound_client_public_key VARCHAR(100) NOT NULL DEFAULT '',
    bound_ips       TEXT         NOT NULL DEFAULT '[]',
    last_used_client_id VARCHAR(64) NOT NULL DEFAULT '',
    last_used_ip    VARCHAR(50)  NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_device_access_grants_device_id
    ON device_access_grants (device_id);

CREATE INDEX IF NOT EXISTS idx_device_access_grants_code
    ON device_access_grants (code);

CREATE INDEX IF NOT EXISTS idx_device_access_grants_owner_id
    ON device_access_grants (owner_id);

-- Optional guest binding: which client_id / address may redeem the code, and
-- who actually did. Added after the table shipped, hence the ALTERs.
ALTER TABLE device_access_grants
    ADD COLUMN IF NOT EXISTS bound_client_id VARCHAR(64) NOT NULL DEFAULT '';

ALTER TABLE device_access_grants
    ADD COLUMN IF NOT EXISTS bound_ips TEXT NOT NULL DEFAULT '[]';

-- Optional proof of possession: when set, the controller bound by client id
-- must also sign the server's challenge with the matching private key.
ALTER TABLE device_access_grants
    ADD COLUMN IF NOT EXISTS bound_client_public_key VARCHAR(100) NOT NULL DEFAULT '';

ALTER TABLE device_access_grants
    ADD COLUMN IF NOT EXISTS last_used_client_id VARCHAR(64) NOT NULL DEFAULT '';

ALTER TABLE device_access_grants
    ADD COLUMN IF NOT EXISTS last_used_ip VARCHAR(50) NOT NULL DEFAULT '';

-- ---------------------------------------------------------------
-- Device access approvals (second factor for high-risk sessions)
-- Created by `access-code:verify` when the policy has approval_required set.
-- The controller polls/claims with a one-time secret (stored as a SHA-256
-- hash); the owner approves or denies. The approved -> consumed transition is
-- a single UPDATE, so only one claim can mint a signal token.
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS device_access_approvals (
    id          BIGSERIAL PRIMARY KEY,
    device_id   VARCHAR(20) NOT NULL,
    owner_id    BIGINT      NOT NULL DEFAULT 0,
    client_id   VARCHAR(64) NOT NULL DEFAULT '',
    client_ip   VARCHAR(50) NOT NULL DEFAULT '',
    source      VARCHAR(16) NOT NULL DEFAULT 'device_code',
    grant_id    BIGINT,
    secret_hash VARCHAR(64) NOT NULL DEFAULT '',
    status      VARCHAR(16) NOT NULL DEFAULT 'pending',
    expires_at  TIMESTAMPTZ NOT NULL,
    decided_at  TIMESTAMPTZ,
    decided_by  BIGINT      NOT NULL DEFAULT 0,
    reason      VARCHAR(255) NOT NULL DEFAULT '',
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_device_access_approvals_device_id
    ON device_access_approvals (device_id);

CREATE INDEX IF NOT EXISTS idx_device_access_approvals_owner_id
    ON device_access_approvals (owner_id);

CREATE INDEX IF NOT EXISTS idx_device_access_approvals_status
    ON device_access_approvals (status);

CREATE INDEX IF NOT EXISTS idx_device_access_approvals_expires_at
    ON device_access_approvals (expires_at);

CREATE INDEX IF NOT EXISTS idx_device_access_approvals_grant_id
    ON device_access_approvals (grant_id);

-- ---------------------------------------------------------------
-- Webhooks (system/admin scope, per §2.17 M4)
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS webhooks (
    id             BIGSERIAL PRIMARY KEY,
    name           VARCHAR(100) NOT NULL,
    url            VARCHAR(500) NOT NULL,
    secret         VARCHAR(200),
    events         TEXT         NOT NULL,
    enabled        BOOLEAN      NOT NULL DEFAULT TRUE,
    last_triggered TIMESTAMPTZ,
    last_status    INTEGER      NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------
-- User notification channels (per-user webhook endpoints)
-- ---------------------------------------------------------------
-- The per-user counterpart of `webhooks`: admin webhooks only carry
-- system-scope events (§2.17 M4), so anything tied to one user's devices is
-- delivered to that user's own channel instead. `secret` signs outbound
-- deliveries (X-Webhook-Signature: sha256=...) and is never returned by the
-- API.
CREATE TABLE IF NOT EXISTS user_notification_channels (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT       NOT NULL,
    name           VARCHAR(100) NOT NULL DEFAULT 'channel',
    url            VARCHAR(500) NOT NULL,
    secret         VARCHAR(200) NOT NULL DEFAULT '',
    events         TEXT         NOT NULL,
    enabled        BOOLEAN      NOT NULL DEFAULT TRUE,
    last_triggered TIMESTAMPTZ,
    last_status    INTEGER      NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_notification_channels_user_id
    ON user_notification_channels (user_id);

-- ---------------------------------------------------------------
-- Per-user quiet hours ("do not page me")
-- One row per user, current state only. `snoozed_until` is the one-shot form
-- and `windows` a JSON array of recurring {days, start, end} periods evaluated
-- in `timezone`. The delivery worker holds a due alert while either is active
-- and posts it once the period ends; nothing is dropped.
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_quiet_hours (
    user_id       BIGINT PRIMARY KEY,
    snoozed_until TIMESTAMPTZ,
    windows       TEXT        NOT NULL DEFAULT '',
    timezone      VARCHAR(64) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------
-- Notification delivery outbox (durable, at-least-once webhook retries)
-- One row per (event, channel). The delivery worker claims due rows with
-- SELECT ... FOR UPDATE SKIP LOCKED and retries with exponential backoff
-- until the endpoint answers 2xx or the attempt budget is exhausted.
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_deliveries (
    id              BIGSERIAL PRIMARY KEY,
    channel_id      BIGINT       NOT NULL,
    user_id         BIGINT       NOT NULL,
    event_type      VARCHAR(64)  NOT NULL,
    device_id       VARCHAR(20)  NOT NULL DEFAULT '',
    payload         TEXT         NOT NULL,
    status          VARCHAR(16)  NOT NULL DEFAULT 'pending',
    attempts        INTEGER      NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_status     INTEGER      NOT NULL DEFAULT 0,
    last_error      VARCHAR(255) NOT NULL DEFAULT '',
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_due
    ON notification_deliveries (status, next_attempt_at);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_channel_id
    ON notification_deliveries (channel_id);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_user_id
    ON notification_deliveries (user_id);

-- Denormalised device id so a bulk replay can target one device with an index
-- instead of scanning the payload JSON. Added after the table shipped; rows
-- written before this column have an empty value and only match an unfiltered
-- replay.
ALTER TABLE notification_deliveries
    ADD COLUMN IF NOT EXISTS device_id VARCHAR(20) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_device_id
    ON notification_deliveries (device_id);

-- Notification aggregation: one row can stand for several occurrences of the
-- same (channel, event type, device) when NOTIFICATION_AGGREGATE_SECONDS is
-- set. Rows written before this column existed are single occurrences.
ALTER TABLE notification_deliveries
    ADD COLUMN IF NOT EXISTS occurrence_count INTEGER NOT NULL DEFAULT 1;

-- Delivery acknowledgement: when a human confirmed they saw a delivery, as
-- opposed to an endpoint accepting it with a 2xx. Kept beside delivered_at
-- rather than derived from it - the whole point is that the two can disagree.
-- Rows written before this column existed are unconfirmed (NULL).
ALTER TABLE notification_deliveries
    ADD COLUMN IF NOT EXISTS acked_at TIMESTAMPTZ;

-- Quiet hours park a due alert instead of sending it: held_until marks "this
-- row is waiting for the owner's quiet period to end", which is a different
-- state from a retry backoff and has to be released the moment the setting
-- changes. NULL means "not held".
ALTER TABLE notification_deliveries
    ADD COLUMN IF NOT EXISTS held_until TIMESTAMPTZ;

-- ---------------------------------------------------------------
-- Per-device offline alerts ("tell me when this stays offline")
-- The row carries the evaluator's episode state because presence only answers
-- "is it online right now": offline_since is set on the first offline
-- observation and alerted_at when the alert for that episode was published,
-- and both are cleared when the device comes back so the next outage alerts
-- again. Alerts themselves go out through the notification outbox above.
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS device_offline_alert_rules (
    id            BIGSERIAL PRIMARY KEY,
    device_id     VARCHAR(20) NOT NULL UNIQUE,
    owner_id      BIGINT      NOT NULL DEFAULT 0,
    enabled       BOOLEAN     NOT NULL DEFAULT TRUE,
    after_minutes INTEGER     NOT NULL DEFAULT 10,
    repeat_minutes INTEGER    NOT NULL DEFAULT 0,
    offline_since TIMESTAMPTZ,
    alerted_at    TIMESTAMPTZ,
    snoozed_until TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_device_offline_alert_rules_owner_id
    ON device_offline_alert_rules (owner_id);

-- Repeat reminders and maintenance snoozes were added after the table
-- shipped, so existing installs need the ALTERs too.
ALTER TABLE device_offline_alert_rules
    ADD COLUMN IF NOT EXISTS repeat_minutes INTEGER NOT NULL DEFAULT 0;

ALTER TABLE device_offline_alert_rules
    ADD COLUMN IF NOT EXISTS snoozed_until TIMESTAMPTZ;

-- Recurring maintenance windows (weekly quiet periods) were added after the
-- one-shot snooze, so keep a JSON list of windows plus the zone they are
-- evaluated in. Empty string means "no recurring window".
ALTER TABLE device_offline_alert_rules
    ADD COLUMN IF NOT EXISTS maintenance_windows TEXT NOT NULL DEFAULT '';

ALTER TABLE device_offline_alert_rules
    ADD COLUMN IF NOT EXISTS maintenance_timezone VARCHAR(64) NOT NULL DEFAULT '';

-- ---------------------------------------------------------------
-- Preset (single-row server config distributed to clients)
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS presets (
    id            BIGSERIAL PRIMARY KEY,
    notice        TEXT         NOT NULL DEFAULT '',
    links         TEXT         NOT NULL DEFAULT '',
    min_version   VARCHAR(32)  NOT NULL DEFAULT '',
    webclient_url VARCHAR(500) NOT NULL DEFAULT '',
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------
-- Owner-scheduled control-plane tasks ("log out everywhere at 22:00").
-- The row carries both the schedule and the outcome: the worker picks due rows
-- with FOR UPDATE SKIP LOCKED, runs the kind's action, then either advances
-- next_run_at by repeat_seconds or disables a one-shot row. Finished rows are
-- kept so "did it fire, and what happened" stays answerable.
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT       NOT NULL,
    kind           VARCHAR(64)  NOT NULL,
    next_run_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    repeat_seconds INTEGER      NOT NULL DEFAULT 0,
    enabled        BOOLEAN      NOT NULL DEFAULT TRUE,
    -- Optional calendar recurrence: daily@HH:MM, weekdays@HH:MM,
    -- weekly@<dow>@HH:MM, monthly@<dom>@HH:MM or every@<N>@days@HH:MM. When set
    -- it *is* the period, so repeat_seconds must be 0 and next_run_at is
    -- recomputed from it after every run.
    schedule       VARCHAR(64)  NOT NULL DEFAULT '',
    -- Move a calendar occurrence off a statutory day off, keeping the time of
    -- day. Only meaningful with a calendar schedule.
    skip_holidays  BOOLEAN      NOT NULL DEFAULT FALSE,
    -- IANA zone the calendar schedule is evaluated in; '' means UTC.
    timezone       VARCHAR(64)  NOT NULL DEFAULT '',
    max_attempts   INTEGER      NOT NULL DEFAULT 1,
    attempts       INTEGER      NOT NULL DEFAULT 0,
    last_run_at    TIMESTAMPTZ,
    last_status    VARCHAR(16)  NOT NULL DEFAULT '',
    last_error     VARCHAR(255) NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_due
    ON scheduled_tasks (enabled, next_run_at);

CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_user_id
    ON scheduled_tasks (user_id);

-- Added after the table shipped: existing deployments get the column from
-- AutoMigrate, and this keeps the schema file self-sufficient for a fresh one.
ALTER TABLE scheduled_tasks
    ADD COLUMN IF NOT EXISTS skip_holidays BOOLEAN NOT NULL DEFAULT FALSE;

-- ---------------------------------------------------------------
-- Cross-instance log shipping ("search the log of the instance that actually
-- served the request"). Each instance copies its own log lines here when
-- INSTANCE_LOG_SINK is on, and an admin search with scope=all reads this table
-- instead of only the files on the machine that answered.
--
-- The row is a *copy*: the authoritative log is still the rotating file on the
-- instance that wrote the line, which is why retention here deletes without
-- archiving. (instance_id, source, seq) is unique so a re-sent batch is
-- ignored rather than duplicated.
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS instance_log_entries (
    id          BIGSERIAL PRIMARY KEY,
    instance_id VARCHAR(64)  NOT NULL,
    source      VARCHAR(64)  NOT NULL DEFAULT '',
    seq         BIGINT       NOT NULL,
    logged_at   TIMESTAMPTZ  NOT NULL,
    line        TEXT         NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_instance_log_seq
    ON instance_log_entries (instance_id, source, seq);

CREATE INDEX IF NOT EXISTS idx_instance_log_entries_logged_at
    ON instance_log_entries (logged_at);

-- Search indexes. GORM's AutoMigrate does not create expression indexes or
-- generated columns, so the server ensures these itself at startup
-- (internal/database/log_search_index.go) and logs a warning if a managed
-- Postgres refuses the extension: the queries stay correct either way, they
-- just fall back to a sequential scan.
--
-- pg_trgm backs the substring search (`q=`), which keeps its ILIKE semantics.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_instance_log_entries_line_trgm
    ON instance_log_entries USING gin (line gin_trgm_ops);

-- The generated tsvector backs the full-text search (`match=`); 'simple'
-- because log lines are identifiers, not prose, and stemming would rewrite
-- them.
ALTER TABLE instance_log_entries
    ADD COLUMN IF NOT EXISTS line_tsv tsvector
    GENERATED ALWAYS AS (to_tsvector('simple', line)) STORED;

CREATE INDEX IF NOT EXISTS idx_instance_log_entries_line_tsv
    ON instance_log_entries USING gin (line_tsv);

-- Self-hosted update channel. The server owns the index (which version exists
-- for which channel/platform/arch, where the bytes live and what they hash to);
-- the artifact itself stays on whatever static host the operator controls.
-- (channel, platform, arch, version) is unique so a re-run cannot silently
-- replace the bytes behind a version clients have already installed.
CREATE TABLE IF NOT EXISTS update_releases (
    id           BIGSERIAL     PRIMARY KEY,
    channel      VARCHAR(32)   NOT NULL DEFAULT 'stable',
    platform     VARCHAR(32)   NOT NULL,
    arch         VARCHAR(32)   NOT NULL,
    version      VARCHAR(64)   NOT NULL,
    download_url VARCHAR(1024) NOT NULL,
    sha256       VARCHAR(64)   NOT NULL,
    size_bytes   BIGINT        NOT NULL DEFAULT 0,
    notes        TEXT          NOT NULL DEFAULT '',
    mandatory    BOOLEAN       NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_update_release_version
    ON update_releases (channel, platform, arch, version);
