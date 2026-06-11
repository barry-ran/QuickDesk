# QuickDesk Signaling Server v1 API

> Updated: 2026-05-12 (v1 refactor; aligned with `docs/dev/信令服务器API重构方案.md` §2.2)
> All routes are version-prefixed under `/v1`. Pre-refactor `/api/v1/*` endpoints are removed.

This document is the canonical client-facing reference for the SignalingServer
HTTP/WS surface. Server source of truth lives in
[`SignalingServer/cmd/signaling/main.go`](../cmd/signaling/main.go); the design
rationale for every decision is in
[`QuickDesk/docs/dev/信令服务器API重构方案.md`](../../docs/dev/信令服务器API重构方案.md).

---

## Table of contents

1. [Conventions](#1-conventions)
2. [Authentication layers](#2-authentication-layers)
3. [Public surface](#3-public-surface)
4. [Auth (register / login / refresh / password reset)](#4-auth)
5. [Current user (`/v1/me/*`)](#5-current-user-v1me)
6. [Devices (host / client side)](#6-devices)
7. [WebSockets](#7-websockets)
8. [Admin surface](#8-admin-surface)
9. [Error format](#9-error-format)
10. [Rate limits](#10-rate-limits)
11. [Realtime event types](#11-realtime-event-types)

---

## 1. Conventions

### 1.1 Wire format

- All requests and responses are `application/json` unless noted.
- All field names on the wire are **`snake_case`** (`device_id`, `access_code`,
  `last_seen_at`, `next_cursor`, `device_count`, `channel_type`, `date_from`,
  …). i18n dictionary keys and language-internal identifiers may keep their
  native casing — only fields that cross the network boundary are normalized.
- Timestamps are RFC 3339 / ISO 8601 with timezone (`2026-05-12T08:52:39Z`).
- All non-2xx responses are RFC 7807 problem details with
  `Content-Type: application/problem+json` (see [§9](#9-error-format)).

### 1.2 Envelopes

- **Single object**: bare object — `{"id": 1, …}`.
- **List**: cursor-paginated — `{"items": [...], "next_cursor": "..." | null, "total": 123?}`.
- **Errors**: RFC 7807 (see §9).

### 1.3 Pagination

List endpoints accept:

- `cursor` — opaque token from the previous response's `next_cursor`. Empty/absent
  means "first page".
- `limit` — page size (default 20, max 100).

Server returns `next_cursor: null` when the cursor reaches the end. Cursors are
order-sensitive: changing sort/filter parameters invalidates the cursor stack —
clients should reset.

### 1.4 Request ID

Every HTTP request gets an `X-Request-ID` header:

- Client may send `traceparent` / `X-Request-ID`; server propagates it.
- Server-generated ULIDs otherwise.
- Error responses echo it as `trace_id` for correlation.

---

## 2. Authentication layers

Three orthogonal layers:

| Layer | Credential | Where it goes | Lifetime |
|---|---|---|---|
| **Server admission** | `X-API-Key` (compile-injected on official binaries; runtime overridable) | All `/v1/*` HTTP requests | Compile-time fixed (or admin override) |
| **Device identity** | `device_secret` (per-device, server-issued at provision time) | `Authorization: Bearer <device_secret>` (host-side device APIs only) | Permanent until rotation |
| **User identity** | `access_token` (short) + `refresh_token` (long) | `Authorization: Bearer <access_token>` (user APIs); refresh only at `POST /v1/auth/tokens:refresh` | access 2 h, refresh 30 d |

Notes:

- `X-API-Key` is the global "official binary admits to official server" gate.
  When the server has no API key configured, the middleware short-circuits to
  pass-through — self-hosted deployments work without any client config.
- WebSocket upgrades (`/v1/realtime/*`) are **not** subject to `X-API-Key`,
  because browsers can't attach custom headers to a WS handshake. They
  authenticate via first-frame auth (§7).
- `device_secret` and the user `access_token` are completely independent. An
  unbound device can connect host signaling without any user; a logged-in user
  can read their devices without any device's secret.
- The "client" role of access-code verify accepts **either** `X-API-Key`
  **or** an `Origin` in the admin-configured `allowed_origins` whitelist —
  this is what lets a browser-only WebClient call the verify endpoint.

---

## 3. Public surface

### 3.1 `GET /health`

Liveness + readiness probe. Public, no auth.

```json
{
  "status": "ok",
  "version": "v2.10.0",
  "components": {
    "postgres": "ok",
    "redis": "ok"
  }
}
```

Returns `503` with `status: "degraded"` if any component is non-`ok`. The
`version` value comes from the `Version` ldflag (`go build -ldflags
"-X main.Version=$(git describe --tags --always)"`); defaults to `"dev"` when
unset.

Use as Kubernetes readiness/liveness probe directly.

### 3.2 `GET /v1/preset`

Returns the unified preset bundle (announcements, links, WebClient URL,
minimum native-client version). Consumed by `PresetManager` in Qt and by the
WebClient bootstrap.

```json
{
  "announcement": { "zh_CN": "...", "en_US": "..." },
  "links":        { "zh_CN": [...], "en_US": [...] },
  "webclient_url": "https://web.quickdesk.cc",
  "min_version":   "2.10"
}
```

### 3.3 `GET /v1/settings/public`

Subset of the runtime settings safe for unauthenticated clients (site name,
brand, login logo URL, registration enabled flag, etc.).

### 3.4 `GET /v1/features`

Feature flag map.

```json
{
  "sms_enabled":      true,
  "register_enabled": true
}
```

### 3.5 `POST /v1/verification-codes`

Single entrypoint for all SMS scenarios. The legacy `/api/v1/sms/send` route is
removed.

Request:

```json
{ "phone": "13800138000", "scene": "login" }
```

Valid `scene` values: `login`, `register`, `reset_password`, `bind_phone`.

Response:

```json
{ "request_id": "vc_01H...", "expires_at": "2026-05-12T08:57:39Z" }
```

Rate limits per phone (Redis `qd:sms:rate:{phone}`):

- 1 / minute
- 3 / 10 minutes
- 10 / 24 hours

---

## 4. Auth

All endpoints under `/v1/auth/*` are public (no user token required) but still
sit behind `X-API-Key` when the server has one configured.

### 4.1 `POST /v1/auth/register`

Registers a new account and **automatically logs in** — the response carries
the same envelope as `/v1/auth/sessions` so the client doesn't need to
re-prompt for password.

Request:

```json
{
  "username":  "barry",
  "password":  "...",
  "phone":     "13800138000",
  "email":     "barry@example.com",
  "sms_code":  "123456"
}
```

`phone` / `email` / `sms_code` are optional unless the server requires phone
verification on register (see `/v1/features.register_enabled` semantics).

Response (200):

```json
{
  "user": {
    "id":         1,
    "username":   "barry",
    "phone":      "13800138000",
    "email":      "barry@example.com",
    "level":      "V1",
    "device_count": 0,
    "channel_type": "全球",
    "status":     true,
    "created_at": "2026-05-12T08:52:39Z",
    "updated_at": "2026-05-12T08:52:39Z"
  },
  "access_token":       "...",
  "access_expires_at":  "2026-05-12T10:52:39Z",
  "refresh_token":      "...",
  "refresh_expires_at": "2026-06-11T08:52:39Z"
}
```

### 4.2 `POST /v1/auth/sessions` — password login

```json
{ "identifier": "barry", "password": "..." }
```

`identifier` may be username, phone, or email. Response is identical to
`POST /v1/auth/register`.

### 4.3 `POST /v1/auth/sessions:sms` — SMS login

```json
{ "phone": "13800138000", "sms_code": "123456" }
```

Same response envelope.

### 4.4 `POST /v1/auth/tokens:refresh`

```json
{ "refresh_token": "..." }
```

Response:

```json
{
  "access_token":       "...",
  "access_expires_at":  "...",
  "refresh_token":      "...",
  "refresh_expires_at": "..."
}
```

**Rotation semantics**: the server invalidates the old refresh token
(Redis `GETDEL`) and issues a new one in the same family. Reusing a
previously rotated refresh token causes the entire family to be revoked and
returns `401 REFRESH_INVALID` — the leaked client must re-login.

### 4.5 `POST /v1/auth/password-resets`

```json
{ "phone": "13800138000" }
```

Sends an SMS code (scene `reset_password`).

### 4.6 `POST /v1/auth/password-resets:confirm`

```json
{ "phone": "13800138000", "sms_code": "123456", "new_password": "..." }
```

On success, every existing session of the user is revoked (clients receive a
`session.revoked` realtime event and must clear local credentials).

---

## 5. Current user (`/v1/me/*`)

Requires `Authorization: Bearer <access_token>`. On `401 TOKEN_EXPIRED`,
clients must silently call `/v1/auth/tokens:refresh` and retry once; a second
401 means the session has ended and the client should clear local state and
return to the login flow.

### 5.1 Profile

| Method | Path | Body | Notes |
|---|---|---|---|
| `GET` | `/v1/me` | — | Returns the user object (same shape as `register`). |
| `PUT` | `/v1/me/password` | `{old_password, new_password}` | Revokes all of the user's sessions. |
| `PUT` | `/v1/me/username` | `{username}` | |
| `PUT` | `/v1/me/phone` | `{phone, sms_code}` | SMS verification required. |
| `PUT` | `/v1/me/email` | `{email}` | |

### 5.2 Sessions

| Method | Path | Notes |
|---|---|---|
| `GET` | `/v1/me/sessions` | Lists active sessions (`{id, user_agent, ip, last_seen_at, created_at}`). |
| `DELETE` | `/v1/me/sessions/current` | Logs out the current session. |
| `DELETE` | `/v1/me/sessions/:session_id` | Kicks another session. |

### 5.3 My devices

The unified envelope is `{items: [DeviceItem], next_cursor}`. Every
`DeviceItem` has the canonical shape:

```jsonc
{
  "device_id":   "176017615",
  "device_name": "Barry-PC",
  "remark":      "office desktop",
  "online":      true,    // derived: hb && wsconn — never persisted
  "logged_in":   true,    // derived: logged_in_intent && online
  "access_code": "633732",
  "os":          "win",
  "os_version":  "11",
  "app_version": "2.10.0",
  "last_seen_at":"2026-05-12T08:52:39Z"
}
```

| Method | Path | Body | Notes |
|---|---|---|---|
| `GET` | `/v1/me/devices?cursor=&limit=` | — | Cursor-paginated list. |
| `POST` | `/v1/me/devices` | `{device_id}` | Bind / take-over. Idempotent — re-binding a device already owned by `me` is a 200 no-op. Take-over publishes `device.ownership.lost` to the previous owner. |
| `GET` | `/v1/me/devices/:device_id` | — | One-shot device detail. |
| `PATCH` | `/v1/me/devices/:device_id` | `{remark?, device_name?}` | Partial update. |
| `DELETE` | `/v1/me/devices/:device_id` | — | Unbind (clears `user_id` and `logged_in_intent`). |
| `DELETE` | `/v1/me/devices/:device_id/session` | — | Marks `logged_in_intent = false` without unbinding. Used by the Qt logout flow. |

### 5.4 Connections (history)

| Method | Path | Body |
|---|---|---|
| `GET` | `/v1/me/connections?cursor=&limit=&since=` | — |
| `POST` | `/v1/me/connections` | `{device_id, duration, status: "success"\|"failed", error_msg?}` |

### 5.5 Favorites

| Method | Path | Body |
|---|---|---|
| `GET` | `/v1/me/favorites` | — |
| `POST` | `/v1/me/favorites` | `{device_id, device_name?, access_password?}` |
| `PATCH` | `/v1/me/favorites/:device_id` | `{device_name?, access_password?}` |
| `DELETE` | `/v1/me/favorites/:device_id` | — |

`FavoriteItem` shape: `{device_id, device_name, access_password, created_at}`.

---

## 6. Devices

These endpoints serve the **device side** (Chromium host, headless agents,
official Qt instances acting on behalf of the host). All require
`X-API-Key`; most additionally require `Authorization: Bearer <device_secret>`.

### 6.1 `POST /v1/devices:provision`

First-run provisioning. Allocates `(device_id, device_secret)` and persists
the secret hash.

Auth: `X-API-Key` only.

Request:

```json
{
  "device_uuid":  "01H...-deterministic-from-hardware",
  "os":           "win",
  "os_version":   "11",
  "app_version":  "2.10.0"
}
```

Response (200):

```json
{
  "device_id":     "176017615",
  "device_secret": "<plaintext, returned only here>",
  "device_uuid":   "01H...",
  "issued_at":     "2026-05-12T08:52:39Z"
}
```

Re-provisioning the same `device_uuid` is allowed and rotates the secret
(old secret becomes invalid). Owner / access-code / `logged_in_intent` are
preserved.

### 6.2 `POST /v1/devices/:device_id/heartbeat`

Auth: `X-API-Key` + `Authorization: Bearer <device_secret>`.

Request (all fields optional):

```json
{ "app_version": "2.10.0", "os": "win", "stats": { "cpu": 12.5 } }
```

Response:

```json
{
  "server_time":                "2026-05-12T08:52:39Z",
  "turn_config_version":        4,
  "suggested_heartbeat_interval_sec": 30
}
```

The server may set a `Retry-After: <n>` header to throttle the host. Clients
**must** honor it.

The host watches `turn_config_version` for changes; on bump, it re-fetches
`/v1/ice-config` (this is the only signal a host needs to pick up admin TURN
changes — see §2.19 of the design doc).

### 6.3 `POST /v1/devices/:device_id/signal-tokens`

Auth: `X-API-Key` + `Authorization: Bearer <device_secret>`.

Issues a one-shot `signal_token` valid for 300 seconds, used by the host to
authenticate its `/v1/realtime/signal` first frame. **Hosts must request a
fresh token before every WS connection — never cache.**

Response:

```json
{ "signal_token": "stk_01H...", "expires_at": "2026-05-12T08:57:39Z" }
```

### 6.4 `PUT /v1/devices/:device_id/access-code`

Auth: `X-API-Key` + `Authorization: Bearer <device_secret>`.

Reports the current plaintext access code. Issued by Qt (which receives the
`device_secret` over native-messaging from the Chromium host at process
startup; held only in Qt process memory, never persisted).

Request:

```json
{ "access_code": "633732" }
```

Response: `204 No Content`. Server publishes `device.access_code.changed` to
the owning user's realtime events stream (the event payload **never** contains
the plaintext code — only a `changed` flag).

### 6.5 `POST /v1/devices/:device_id/access-code:verify`

The **client-side** verify entrypoint. Used by the WebClient (browser) and Qt
to validate an access code and obtain a one-shot `signal_token`.

Auth: `X-API-Key` **or** `Origin` in the admin-configured `allowed_origins`
whitelist (whichever the request can satisfy). Browsers depend on the second
form.

Request:

```json
{ "code": "633732" }
```

Response (200):

```json
{ "signal_token": "stk_01H...", "expires_at": "2026-05-12T08:53:39Z" }
```

Error path:

| HTTP | `code` | Meaning |
|---|---|---|
| `404` | `DEVICE_NOT_FOUND` | No such `device_id` registered |
| `409` | `HOST_OFFLINE` | Device exists but is not online (no host hb + wsconn) |
| `403` | `INVALID_CODE` | Access code is wrong (counts toward rate-limit budget) |
| `403` | `TOO_MANY_ATTEMPTS` | Rate limit exceeded; honor the `Retry-After` header |

Rate limits (Redis `INCR`-based, 60-second windows):

- `(device_id, ip)` — 5 wrong attempts → 403 `TOO_MANY_ATTEMPTS`
- `ip` (any device) — 60 total attempts → 429
- `device_id` (any ip) — 30 wrong attempts → 403

### 6.6 `GET /v1/ice-config`

Returns the global ICE configuration. Auth: `X-API-Key` (or hosts may
additionally present `Authorization: Bearer <device_secret>`).

```json
{
  "ice_servers": [
    { "urls": ["stun:stun.l.google.com:19302"] },
    { "urls": ["turn:turn.example.com:3478"], "username": "...", "credential": "..." }
  ],
  "turn_config_version": 4
}
```

---

## 7. WebSockets

Two long-lived endpoints. Both authenticate via **first-frame auth** because
browsers can't attach custom headers to WS upgrades.

Both endpoints `close(4401)` connections that don't send a valid auth frame
within 5 seconds of the upgrade.

### 7.1 `GET /v1/realtime/events` — user event stream

Used by Qt and the WebClient to receive snapshot + incremental updates for
my devices and favorites.

Handshake:

```
1. Client opens WS.
2. Client sends:   {"type":"auth", "access_token":"...", "since_rev": 4217?}
3. Server replies: {"type":"auth_ok", "server_rev": 4218}
4. Server immediately sends a snapshot OR (if `since_rev` is recent enough)
   replays missed events from Redis stream `qd:events:user:{user_id}`.
```

Snapshot frame:

```json
{
  "type": "snapshot",
  "server_rev": 4218,
  "data": {
    "devices":   [DeviceItem, ...],
    "favorites": [FavoriteItem, ...]
  }
}
```

Clients must **replace** local state on snapshot — never merge. Subsequent
events with `device.*` / `favorite.*` types are patches and must be applied
in-place; **clients must not refetch `/v1/me/devices` on every event** (anti-
thundering-herd rule).

If the gap is too large (Redis stream has been trimmed), the server replies
`{type:"snapshot_required"}` and the client must re-send the auth frame to
trigger a fresh snapshot.

`session.revoked` events on this stream mean the user's session has been
terminated by an admin (or family rotation detected a leak). Clients should
clear local credentials and exit to login.

### 7.2 `GET /v1/realtime/signal` — SDP/ICE relay

Used by hosts and clients (browser / native) for WebRTC signaling.

Handshake:

```
1. Client/host opens WS.
2. First frame:
   {"type":"auth", "signal_token":"...", "role":"host"|"client",
    "device_id":"176017615", "client_id":"cli-..." }     // client_id for clients only
3. Server replies: {"type":"auth_ok", "session_id":"..."} on success.
4. SDP/ICE relay begins (existing jingle envelope reused).
```

Sending SDP/ICE before `auth_ok` causes the server to drop and close the
connection. `signal_token` is single-use — it is `GETDEL`'d at first-frame
auth and may not be re-presented on a different WS.

A host that goes offline mid-session causes the server to push
`{"type":"error", "code":"PEER_DISCONNECTED"}` to all bound clients and close
their WS. The client should record a `connections` entry with
`status: "failed"`.

---

## 8. Admin surface

Mounted at `/v1/admin/*`. All endpoints require an admin access token; writes
additionally require 2FA-enabled super-admin (see §2.16 of the design doc).
The admin web SPA in [`SignalingServer/web/`](../web/) is the reference
consumer. Client implementers should consult the admin web source for exact
request shapes; this section documents only the route table.

### 8.1 Admin auth

| Method | Path | Body | Notes |
|---|---|---|---|
| `POST` | `/v1/admin/auth/sessions` | `{username, password, totp_code?}` | If account has 2FA enabled and `totp_code` is omitted, server returns `401 TOTP_REQUIRED` with `pre_token`. |
| `POST` | `/v1/admin/auth/sessions:totp` | `{pre_token, totp_code}` | Two-step completion. |
| `POST` | `/v1/admin/auth/tokens:refresh` | `{refresh_token}` | |
| `DELETE` | `/v1/admin/auth/sessions/current` | — | |

### 8.2 Admin accounts (CRUD over admins themselves)

| Method | Path |
|---|---|
| `GET` | `/v1/admin/admins` |
| `POST` | `/v1/admin/admins` |
| `GET` | `/v1/admin/admins/:id` |
| `PATCH` | `/v1/admin/admins/:id` |
| `DELETE` | `/v1/admin/admins/:id` |
| `POST` | `/v1/admin/admins/me/2fa/setup` |
| `POST` | `/v1/admin/admins/me/2fa/verify` |
| `DELETE` | `/v1/admin/admins/me/2fa` |

### 8.3 Business users

| Method | Path | Notes |
|---|---|---|
| `GET` | `/v1/admin/users` | Filters: `channel_type`, `status`, `level`, `q`. |
| `POST` | `/v1/admin/users` | |
| `POST` | `/v1/admin/users:batch` | `{ids: [...], op: "enable"\|"disable"\|"delete"\|"set_level"}` |
| `GET` | `/v1/admin/users/:id` | |
| `GET` | `/v1/admin/users/:id/details` | Includes bound devices + active sessions. |
| `PATCH` | `/v1/admin/users/:id` | |
| `DELETE` | `/v1/admin/users/:id` | Cascades: `UPDATE devices SET user_id=NULL, logged_in_intent=false WHERE user_id=?`. |
| `POST` | `/v1/admin/users/:id/sessions:revoke` | Kicks every session of that user. |
| `PATCH` | `/v1/admin/users/:id/device-count` | `{deviceCount: N}` (kept camelCase for backward compat — the only such field). |

### 8.4 Devices

| Method | Path | Notes |
|---|---|---|
| `GET` | `/v1/admin/devices` | |
| `POST` | `/v1/admin/devices:batch` | `{ids: [...], op: "delete"\|"assign_group"\|"remove_group"}` |
| `GET` | `/v1/admin/devices/:device_id` | |
| `DELETE` | `/v1/admin/devices/:device_id` | Hard delete. |
| `POST` | `/v1/admin/devices/:device_id/unbind` | Force-unbind, keep device record. |
| `POST` | `/v1/admin/devices/:device_id/secret:rotate` | Invalidates `device_secret`; host re-provisions on next 401. |

### 8.5 Bindings / groups / stats / audit / preset / settings / webhooks

| Method | Path |
|---|---|
| `GET` | `/v1/admin/device-bindings` |
| `GET` `POST` `PATCH` `DELETE` | `/v1/admin/groups[/:id]` |
| `GET` `POST` `DELETE` | `/v1/admin/groups/:id/devices` |
| `GET` | `/v1/admin/stats` |
| `GET` | `/v1/admin/system/status` |
| `GET` | `/v1/admin/connections` |
| `GET` | `/v1/admin/activity` |
| `GET` | `/v1/admin/trends?range=24h\|7d\|30d` |
| `GET` | `/v1/admin/audit-logs` (filters `date_from` / `date_to` / `q`) |
| `GET` `PUT` | `/v1/admin/preset` |
| `GET` `PUT` | `/v1/admin/settings` |
| `GET` `POST` `PATCH` `DELETE` | `/v1/admin/webhooks[/:id]` |
| `POST` | `/v1/admin/webhooks/:id/test` |

> **Why `webhooks/:id/test` and not `webhooks/:id:test`** — gin's httprouter
> cannot register a literal suffix on a wildcard segment. The AIP-136-style
> `:test` would silently fail to register. Sub-resource form is RESTful and
> consistent with what gin can express. See §6 W1 in the design doc for the
> historical record. All other "colon action" endpoints (`sessions:sms`,
> `tokens:refresh`, `users:batch`, `secret:rotate`, …) have a
> literal segment before the colon and work fine. The `2fa/setup` and
> `2fa/verify` endpoints are exceptions — they use sub-resource form because
> `2fa:setup` vs `2fa:verify` would be two competing wildcards on the same
> segment (same issue as `webhooks/:id:test`).

---

## 9. Error format

Every non-2xx response is RFC 7807 problem-details with
`Content-Type: application/problem+json`:

```json
{
  "type":     "https://quickdesk.io/problems/device-not-found",
  "title":    "Device not found",
  "status":   404,
  "detail":   "Device 176017615 is not registered",
  "instance": "/v1/me/devices/176017615",
  "code":     "DEVICE_NOT_FOUND",
  "trace_id": "01H..."
}
```

Common `code` values:

| Code | HTTP | When |
|---|---|---|
| `INVALID_REQUEST` | 400 | Body / query validation failed |
| `UNAUTHORIZED` | 401 | Missing or malformed auth |
| `TOKEN_EXPIRED` | 401 | `access_token` expired — try refresh once |
| `TOKEN_INVALID` | 401 | `access_token` revoked or never existed |
| `REFRESH_INVALID` | 401 | `refresh_token` expired / leaked / replayed |
| `TOTP_REQUIRED` | 401 | Admin login needs second factor |
| `FORBIDDEN` | 403 | Authenticated but not allowed |
| `INVALID_CODE` | 403 | Wrong access code (counts toward rate budget) |
| `TOO_MANY_ATTEMPTS` | 403 | Rate limit hit; see `Retry-After` |
| `DEVICE_NOT_FOUND` | 404 | No such `device_id` |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Generic conflict |
| `HOST_OFFLINE` | 409 | Verify reached an offline host |
| `RATE_LIMITED` | 429 | IP-level rate limit |
| `INTERNAL` | 500 | Server bug — file an issue with `trace_id` |
| `UNAVAILABLE` | 503 | Component degraded; see `/health` |

Clients must read both `status` and `code`. Display `detail` to the user when
no localized i18n key matches `code`.

---

## 10. Rate limits

| Surface | Window | Quota |
|---|---|---|
| `POST /v1/verification-codes` per phone | 1 min / 10 min / 24 h | 1 / 3 / 10 |
| `POST /v1/devices/:id/access-code:verify` per `(device, ip)` (errors only) | 60 s | 5 |
| same per `ip` (any device) | 60 s | 60 |
| same per `device_id` (errors only, any ip) | 60 s | 30 |
| `POST /v1/devices/:id/heartbeat` per device | min interval | 1 s |
| `POST /v1/devices/:id/signal-tokens` per device | min interval | 1 s |

Hits return `403 TOO_MANY_ATTEMPTS` or `429 RATE_LIMITED` with `Retry-After`.

---

## 11. Realtime event types

Pushed over `/v1/realtime/events` to the owning user. Event payloads never
include access-code plaintext, `device_secret`, or `refresh_token`.

| `type` | When | Payload sketch |
|---|---|---|
| `device.online.changed` | hb-TTL expiry or wsconn DEL → `online` flips | `{device_id, online}` |
| `device.session.updated` | `logged_in_intent` flipped (bind / clear-session) | `{device_id, online, logged_in}` |
| `device.access_code.changed` | host reported a new code | `{device_id, changed: true}` (no plaintext) |
| `device.bound` / `device.added` | new device for this user | `{device_id}` (clients fetch detail once if not present) |
| `device.ownership.lost` | another account took over this device | `{device_id}` |
| `device.unbound` | unbound by user or admin | `{device_id}` |
| `device.remark.changed` / `device.display_name.changed` | `PATCH /v1/me/devices/:id` | `{device_id, remark?, device_name?}` |
| `favorite.added` / `favorite.updated` / `favorite.removed` | favorites mutated | `{device_id, ...}` |
| `session.revoked` | this user's session was killed | `{}` |
| `device.secret.rotated` | admin rotated `device_secret` (sent to host owner) | `{device_id}` (host should re-provision on next 401) |

Event envelope (always present):

```json
{
  "id":         "evt_01HXY...",
  "type":       "device.session.updated",
  "ts":         "2026-05-12T08:52:39Z",
  "server_rev": 4218,
  "data":       { ... }
}
```

`server_rev` increments per published event and supports the resume protocol
(§7.1).

---

## Appendix: removed pre-refactor routes

For migration awareness only. Do not call these on a v1 server — they 404.

```
/api/v1/user/register                  →  /v1/auth/register
/api/v1/user/login                     →  /v1/auth/sessions
/api/v1/user/login-sms                 →  /v1/auth/sessions:sms
/api/v1/user/logout                    →  DELETE /v1/me/sessions/current
/api/v1/user/reset-password            →  /v1/auth/password-resets[:confirm]
/api/v1/user/me                        →  /v1/me
/api/v1/user/devices                   →  /v1/me/devices
/api/v1/user/devices/unbind            →  DELETE /v1/me/devices/:id
/api/v1/user/devices/auto-bind         →  POST /v1/me/devices
/api/v1/user/devices/record            →  POST /v1/me/connections
/api/v1/user/devices/logs              →  GET /v1/me/connections
/api/v1/user/favorites                 →  /v1/me/favorites
/api/v1/devices/register               →  /v1/devices:provision
/api/v1/devices/:id/status             →  derived from realtime events
/api/v1/auth/verify                    →  /v1/devices/:id/access-code:verify
/api/v1/ice-config                     →  /v1/ice-config
/api/v1/sms/send                       →  /v1/verification-codes
/signal/:device_id                     →  /v1/realtime/signal (first-frame auth)
/host/:device_id                       →  /v1/realtime/signal
/client/:device_id/:access_code        →  /v1/realtime/signal (after :verify)
/api/v1/user/sync?token=               →  /v1/realtime/events (first-frame auth)
```
