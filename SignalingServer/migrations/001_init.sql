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
