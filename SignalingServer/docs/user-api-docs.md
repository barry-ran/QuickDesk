# QuickDesk Signaling Server v1 API

> Updated: 2026-05-12 (v1 refactor; aligned with `docs/dev/信令服务器API重构方案.md` §2.2)
> All routes are version-prefixed under `/v1`. Pre-refactor `/api/v1/*` endpoints are removed.

This document is the canonical client-facing reference for the SignalingServer
HTTP/WS surface. Server source of truth lives in
[`SignalingServer/cmd/signaling/main.go`](../cmd/signaling/main.go); the design
rationale for every decision is in
[`QuickDesk/docs/dev/信令服务器API重构方案.md`](../../docs/dev/信令服务器API重构方案.md).

### Machine-readable contract

[`SignalingServer/api/openapi.yaml`](../api/openapi.yaml) is an OpenAPI 3.1
contract for the stable, integrator-facing subset of this API (authenticate,
enumerate devices, verify an access code, and manage the access policy, guest
access codes, approvals including approve / deny / claim, offline-alert rule,
notification channels, and the session-recording index including media
download and playback tickets) — generate a client in any language from it.

Four generated clients are committed and kept in step with the spec by CI:
[`sdk/ts`](../../sdk/ts) (TypeScript, `fetch`), [`sdk/go`](../../sdk/go) (Go),
[`sdk/python`](../../sdk/python) (Python) and [`sdk/java`](../../sdk/java)
(Java, okhttp-gson). All four come from a pinned openapi-generator jar, so a
local regeneration and a CI regeneration are byte-identical:

```bash
scripts/sdk/generate-client.sh sdk/ts
scripts/sdk/generate-client.sh sdk/go go
scripts/sdk/generate-client.sh sdk/python python
scripts/sdk/generate-client.sh sdk/java java
scripts/sdk/check-generated-sdk.sh          # fails when sdk/ts is stale
scripts/sdk/check-generated-sdk.sh go       # fails when sdk/go is stale
scripts/sdk/check-generated-sdk.sh python   # fails when sdk/python is stale
scripts/sdk/check-generated-sdk.sh java     # fails when sdk/java is stale
```

All four clients carry hand-written `examples/` code, and the smoke job runs
each against a live server before the conformance smoke takes its device
offline. `sdk/go/examples/recordinglifecycle` runs a whole write workflow
(create a recording, upload the media, patch the row, find it in the index, mint
a playback ticket, read the bytes back through that ticket - the route a
`<video>` tag uses, with no Authorization header - and delete the row).
`sdk/go/examples/resumableupload`, `sdk/python/examples/resumable_upload.py`,
`sdk/ts/examples/resumable-upload.js` and
`sdk/java/src/main/java/org/openapitools/client/examples/ResumableUpload.java`
cover the chunked half of the same contract: two PATCH chunks with a declared
total, an `Upload-Probe` offset probe, a 409 resynchronisation, server-side
finalisation, and a byte-identical read back through a ticket.
`sdk/go/examples/approvalflow`, `sdk/python/examples/approval_flow.py`,
`sdk/ts/examples/approval-flow.js` and
`sdk/java/src/main/java/org/openapitools/client/examples/ApprovalFlow.java` all
drive the same second-factor flow through their own generated client: require
approval, receive the pending answer, poll a claim while it waits, see it in the
owner's list, approve, exchange it for a signal token, and restore the policy.
The Python example is also the one that caught the spec's unquoted `off` enum
(YAML read it as boolean `false`, so strict clients could not deserialize the
server's default policy). The freshness check excludes those `examples/`
directories; the Java client is otherwise compiled with Maven in CI
(`mvn -DskipTests package`). The generated trees are otherwise untouched, so
regenerating after a spec change is the whole maintenance story.

The generator applies the normalisations the templates get wrong for a nested
module: the Go client's `go.mod` module path is pinned to
`github.com/ilh-github/go-remote-next/sdk/go`, and the Python `pyproject.toml`
license placeholder `NoLicense` (not an SPDX identifier, which makes modern
setuptools refuse to build) becomes `MIT`. The Java template additionally
ignores the git ids and writes a build date into every file, so the generator
normalises the pom's url/scm to this repository and pins
`hideGenerationTimestamp=true`; it also emits Maven, Gradle and SBT builds at
once, and only Maven is kept because only Maven is verified. All of it lives in
`scripts/sdk/generate-client.sh` so the freshness check reproduces it.

It covers
a documented subset rather than every route: operations are added as they are
pinned by a conformance check, which is
[`scripts/sdk/openapi-smoke.mjs`](../../scripts/sdk/openapi-smoke.mjs). That
script calls the documented operations against a running server and asserts the
fields the spec promises, so a rename or shape change fails the check instead of
silently invalidating the file. It is read-only by default; pass
`QD_SMOKE_DEVICE=<device_id>` to also exercise the offline-alert and
access-policy `PUT`s (both restored to their prior state), issue-then-revoke a
short-lived guest code, and the rejected-access-code shape. The guest-code probe
deliberately leaves one **revoked, never-used** grant row behind, because
revocation rather than deletion is how a grant is disabled.

The approval **decision** routes (`approve`, `deny`, `access-approvals:claim`)
are in the contract, but the always-runnable smoke only pins their `404` / `400`
shapes (it uses an id that cannot exist, so nothing is created or decided). Their
happy path needs a real pending approval - an online device plus
`approval_mode` - so it is verified by the end-to-end run recorded in
`docs/dev/自托管端到端联调记录.md` §67 rather than by the smoke.

The recording media routes are covered too, and unlike the approval decisions
they are pinned end to end by the always-runnable smoke: it mints an artifact
ticket and a thumbnail ticket for the newest recording that has a stored
artifact, asserts the returned `url` shape, fetches **one byte** through the
ticket URL with no `Authorization` header (proving the ticket is the
authorization), and checks the `401` for a bogus ticket. Minting a ticket is
stateless, so nothing is written by these checks. The recording **write** verbs
(`POST`/`PATCH`/`DELETE` and the multipart uploads) are deliberately still out
of the contract: they need a body the smoke cannot safely invent on a real
account.

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

### `GET /metrics` (Prometheus, opt-in)

Prometheus text exposition is disabled unless `METRICS_TOKEN` is set. When it
is set, scrapers must send `Authorization: Bearer <METRICS_TOKEN>`; a missing
or wrong token is `401`. The route lives outside `/v1` and is not counted by
the API request counter, so the metric endpoint does not inflate its own
traffic.

The response is `text/plain; version=0.0.4` and currently exposes:
`quickdesk_signaling_build_info`, `quickdesk_signaling_uptime_seconds`,
`quickdesk_signaling_api_requests_total`,
`quickdesk_signaling_api_requests_today`,
`quickdesk_signaling_http_requests_by_route_total`,
`quickdesk_signaling_websocket_connections`,
`quickdesk_signaling_signal_hosts`,
`quickdesk_signaling_signal_clients`,
`quickdesk_signaling_event_streams`,
`quickdesk_signaling_go_goroutines`,
`quickdesk_signaling_go_mem_alloc_bytes`, and
`quickdesk_signaling_go_mem_sys_bytes`. `api_requests_total` is process-local
and monotonic; `api_requests_today` is a UTC-day gauge that resets at midnight.

`http_requests_by_route_total` is the family alert rules can be written
against: it carries `route` (gin's matched **pattern**, so
`/v1/devices/:device_id/heartbeat` is one series and not one per device),
`method` and `status` (the exact code). A request that matched no route is
labelled `route="unmatched"` rather than dropped, so a 404 storm stays visible.
The label set is bounded by the route table, and the series are emitted in a
stable order so two scrapes of an idle process are comparable line by line.

Ready-made alert rules live in
[`deploy/prometheus/quickdesk-signaling.rules.yml`](../deploy/prometheus/quickdesk-signaling.rules.yml)
- an API error-ratio alert, a per-route 5xx burst, a goroutine-growth check and
a "no signaling hosts" check. A unit test parses that file and fails if any rule
names a metric the endpoint does not export, so a rename cannot leave a silent
rule behind.

**A counter reset is expected, not a gap.** Every `quickdesk_signaling_*`
counter starts at zero when the process restarts; Prometheus handles that
inside `rate()`, which is why the shipped rules are rate/ratio/gauge based and
never `counter > N`. What the metrics deliberately do **not** carry is
per-user, per-device or per-action business dimensions: those live in the audit
trail and the access timeline, where they are queryable and retained, rather
than in a time series whose cardinality would follow the customer base.

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
| `GET` | `/v1/me/devices/:device_id/access-logs?cursor=&limit=` | — | Owner-only timeline of access to **this device**: `{items: [{id, device_id, owner_id, client_id, session_id, ip, result, reason, created_at}], next_cursor}`. |

### 5.3.1 Device access log (who tried to reach my machine)

Every `POST /v1/devices/:device_id/access-code:verify` writes one row, so the
device owner can answer "did anyone try to open my machine, from where, and
did it work". The controller cannot fabricate or suppress these rows — they are
written server-side, never reported by the client. `result` is one of:

| `result` | Meaning |
|---|---|
| `granted` | Access code matched and the host was online; a `signal_token` was minted. |
| `denied_invalid_code` | Wrong access code. |
| `denied_rate_limited` | Blocked by the per-IP / per-device / per-(device,ip) limiter, or the failure counter tripped. |
| `host_offline` | Code matched but the host was not connected; no session started. |
| `session_started` | The controller actually opened the signaling channel with the token from a `granted` attempt — i.e. the attempt turned into a real control session. Recorded by the realtime handler; `session_id` identifies that session. |
| `denied_policy` | The device's own access policy refused the client (remote access disabled, or the client IP is outside `allow_ips`). See 5.3.3. |
| `denied_session_limit` | The access code was accepted, but the device already had as many controllers as `max_sessions` allows. Recorded when the controller opens its signaling socket. |

`owner_id` is the owner at the moment of the attempt (`0` when the device is
unbound). Rows are newest-first and paginated by the `next_cursor` keyset.
Each attempt also fans a `device.access.denied` / `device.access.granted` event
to that owner's realtime WS stream, and a real control session adds
`device.access.session_started`; rejected attempts additionally land in the
admin audit log.

### 5.3.2 Notification channels (per-user webhooks)

Admin webhooks only carry system-scope events, so a user could not be told
that someone tried to open *their* device. A notification channel is a
user-owned webhook endpoint that receives that user's own device events.

| Method | Path | Body | Notes |
|---|---|---|---|
| `GET` | `/v1/me/notification-channels` | — | `{items, supported_events, signature_header, signature_scheme}`. The signing secret is never returned. |
| `POST` | `/v1/me/notification-channels` | `{name?, url, secret?, events: [...], enabled?}` | `enabled` defaults to `true`. `events` must come from `supported_events`. |
| `PATCH` | `/v1/me/notification-channels/:id` | same as create | Omit `enabled` to keep the current state; omit `secret` to keep the stored one. |
| `DELETE` | `/v1/me/notification-channels/:id` | — | Owner-scoped. |
| `POST` | `/v1/me/notification-channels/:id/test` | — | Fires a synthetic `test` event so the endpoint can be verified without waiting for device traffic. |
| `GET` | `/v1/me/notification-channels/:id/deliveries?limit=` | — | Delivery history for the channel (newest first, default 50, max 200), as `{items, unacked}`. `unacked` counts this channel's rows that were delivered (2xx) and never confirmed; a row with `held_until` is parked by the owner's quiet hours rather than waiting on a retry. Owner-scoped. |
| `POST` | `/v1/me/notification-channels/:id/deliveries/:deliveryId/retry` | — | Re-arms a parked `failed` delivery (dead-letter replay). Returns `400` for rows that are not `failed`. Owner-scoped. |
| `POST` | `/v1/me/notification-channels/:id/deliveries/:deliveryId/ack` | — | Records that somebody confirmed the delivery (`acked_at`). Idempotent — the first ack owns the timestamp and later ones return the same row. `409` for a row that is still `pending`/`delivering`, `404` when the id does not belong to that channel. Owner-scoped. |
| `POST` | `/v1/me/notification-channels/:id/deliveries:retry-failed` | optional body | Re-arms parked `failed` deliveries for the channel. Owner-scoped. |
| `POST` | `/v1/me/notification-channels:retry-failed` | optional body | Re-arms parked `failed` deliveries the caller owns, **across all of their channels** (one call after an outage that hit several endpoints). |
| `GET` | `/v1/me/notification-quiet-hours` | — | The caller's quiet hours: `{snoozed_until, quiet_hours, timezone, active, active_until}`. A user who never saved one gets the documented defaults (`snoozed_until: null`, empty `quiet_hours`, `active: false`). |
| `PUT` | `/v1/me/notification-quiet-hours` | `{snoozed_until?, quiet_hours: [{"days":[0-6],"start":"HH:MM","end":"HH:MM"}], timezone?}` | Full replace: omitting a field clears it, so `{}` turns quiet hours off. `quiet_hours` is capped at 20 windows, `timezone` is an IANA zone (empty = the server's local zone). Malformed input is a `400` and nothing is written. |

The two bulk-replay endpoints take the same optional body. Omitting it replays
everything in scope, which is how they behaved before filters existed.

| Field | Effect |
|---|---|
| `event_type` | only rows of this event type |
| `device_id` | only rows about this device. Rows written before the `device_id` column existed have an empty value and only match an unfiltered replay |
| `limit` | re-arm at most this many rows, **oldest first**, so repeated calls drain the backlog deterministically. `0` = no cap; anything above `1000` is clamped to `1000` and the response reports the effective value; negative is a `400` |
| `spread_over_minutes` | pace the replay across this many minutes instead of making the whole backlog due at once. `0` = everything due immediately (the default); anything above `1440` is clamped and the response reports the effective value; negative is a `400` |
| `channel_ids` | only these channels — the cross-channel endpoint's answer to "one of my endpoints is broken, leave the others alone". At most 50 ids, all positive; on the per-channel endpoint it can only narrow, never widen |
| `created_after` / `created_before` | **inclusive** bounds on when the event was queued (RFC3339). A window selects what the backlog *was* at a point in time, so it is deliberately read from `created_at` rather than `updated_at` (which moves on every attempt). `created_after` later than `created_before` is a `400` rather than an empty result |
| `dry_run` | count what *would* be re-armed and change nothing |

Response: `{success, matched, requeued, dry_run, limit, spread_over_minutes,
filters}`, where `filters` echoes back `event_type`, `device_id`, `channel_ids`
and the window bounds that were actually applied. `matched` is how many parked
rows satisfy the filters before `limit`; `requeued` is how many were actually
re-armed (always `0` on a dry run). The dry run exists because re-arming wakes
the delivery worker immediately, and "replay everything" is rarely the right
first move when one endpoint is broken and another is healthy.

`spread_over_minutes` is the tool for the other half of that problem: after an
outage the whole backlog is due at once, which can knock the endpoint over again
the moment it recovers. A spread replay re-arms every matched row as `pending`
straight away but staggers `next_attempt_at` across the window, so the existing
worker delivers the backlog in waves. The first wave is due one bucket interval
after the call and the last at the end of the window; nothing is due at the
moment of the call.

Payload shape (same as admin webhooks):

```json
{"id": "evt_1a2b3c4d-...", "event": "device.access.denied", "timestamp": "2026-10-16T03:35:23Z", "data": {"device_id": "...", "ip": "...", "reason": "..."}}
```

`id` is the event envelope's stable identity. It is the same value across every
delivery attempt of that event, so a receiver can key a dedupe table on it
without depending on the shape of `data`.

**Acknowledging is a different fact from delivering.** The outbox marks a row
`delivered` when the endpoint answers `2xx`, which says the HTTP call succeeded
and nothing more: a webhook that returns `200` and then drops the event looks
exactly like one that acted on it. `POST .../deliveries/:deliveryId/ack`
records the stronger claim — somebody (or the receiving system on their behalf)
actually saw the alert — in `acked_at`, next to `delivered_at` rather than
replacing it. The first ack wins the timestamp; asking again returns the same
row instead of moving the clock, so a retry or two operators clicking at once
cannot rewrite when the confirmation happened. A row that has not finished
(`pending`, `delivering`) is a `409`: "somebody saw this" is a claim nothing
supports yet, and recording it would turn the column into a lie. A parked
`failed` row *can* be acknowledged, because an operator who saw the alert in
their list has confirmed it even though the network never cooperated.

The number to watch is `unacked` from the deliveries list (and `unacked` /
`acked` on the admin `notification-deliveries:stats`): a channel that answers
`200` into a void shows up there and nowhere else. Every other counter — status
counts, last HTTP status, attempt counts — looks perfectly healthy in that
case.

**Quiet hours hold alerts instead of dropping them.** `snoozed_until` is the
one-shot form ("not tonight") and `quiet_hours` the recurring one (a backup
window, a night shift, a holiday), both evaluated in the setting's own
timezone. While either is active the delivery worker does not post: a due row
goes back to `pending` with `held_until` set to when the quiet period ends, and
the row is attempted normally once that moment passes. Three consequences are
worth knowing:

- **A hold is not an attempt.** `attempts` and `last_status` are untouched, so
  a quiet night cannot consume a healthy channel's retry budget or park its
  rows as `failed` for something the endpoint never did.
- **Changing the setting moves what it was holding.** Clearing the snooze
  releases the held rows immediately (`held_until` cleared, due now) rather
  than leaving them asleep until a deadline that no longer exists; shortening a
  recurring window pulls them forward. Rows waiting on a delivery *backoff* are
  untouched — that is why `held_until` exists as a separate marker.
- **`held_until` answers "why is this alert still pending".** It is a current
  fact, not history: it is set when the row is held and cleared when the hold
  is released.

A recurring window is re-checked every five minutes rather than computing the
exact boundary of the window, so an alert held by one is delivered at most five
minutes after the window ends. A snooze is an exact instant and is honoured to
the second.

Supported events today: `device.access.denied`, `device.access.granted`,
`device.access.session_started`, `device.online.changed`,
`device.offline.sustained` (§5.3.6), `device.bound`, `device.unbound`,
`device.access_approval.requested`, `device.access_approval.decided`,
`scheduled_task.failed` (§5.6).

When `secret` is set the raw body is signed with HMAC-SHA256 and sent as
`X-Webhook-Signature: sha256=<hex>`, so receivers can share one verification
routine with the admin webhooks. URLs must be absolute `http(s)`; operators
exposing this as multi-tenant SaaS should also apply an egress allowlist (a
user-supplied URL is an SSRF surface).

**Delivery is durable (at-least-once).** Every event is first written to the
`notification_deliveries` outbox, so a channel that is briefly down no longer
loses the event. A background worker polls the outbox every 2s, claims due
rows with `SELECT ... FOR UPDATE SKIP LOCKED` (safe across two server
instances) and posts them one at a time:

- the claim is **per-channel fair**: phase one picks the channels holding the
  oldest due work, phase two takes at most 5 rows from each. Ordering the whole
  table by id would let one channel with a deep backlog fill every batch, so
  every other endpoint's notifications would wait behind it no matter how small
  their own backlog is. The trade-off is that a batch comes out shorter than 20
  rows when few channels have due work;
- success (`2xx`) → `status: "delivered"`, `delivered_at` is stamped;
- failure → the row is rescheduled with exponential backoff
  (2s, 4s, 8s … capped at 5m);
- after 8 attempts the row is parked as `status: "failed"` (kept, never
  silently dropped) and still shows up in the history.

Each attempt also records `last_status` (the HTTP code, `0` on a transport
error) and `last_error` on both the row and the channel row.

**Dedupe contract.** Delivery is *at-least-once*: a retry, a bulk replay, or a
delivery whose response was lost re-POSTs a body the receiver may already have
processed. Every POST therefore carries two identity headers:

- `X-Webhook-Delivery-Id` — the outbox row id. It is stable across the retry
  loop *and* across a manual replay of that same row, so it identifies "this
  one attempt to deliver this one row". Two distinct rows for the same event
  (for example the same event delivered to two channels) have different
  delivery ids.
- `X-Webhook-Event-Id` — the event envelope `id`, identical to the `id` field
  in the body. It is stable for the event itself, so a receiver that wants
  "process each event once" dedupes on this rather than on the delivery id.

Either header is omitted when its value is unknown (a row queued before the
event id existed has no `X-Webhook-Event-Id`), so a receiver must treat a
missing header as "no key", not as the empty string. The simplest correct
receiver stores `X-Webhook-Event-Id` in a unique index and ignores a repeat.
The `:id/deliveries` endpoint is the operator-facing log:
it shows what was queued, how many attempts it took, and whether it is still
`pending`, `delivering`, `delivered` or `failed`.

The claim is written as `SELECT ... FOR UPDATE SKIP LOCKED`, so the outbox is
safe to drain from more than one worker or more than one server instance — two
workers never take the same row. A row that exhausts its budget is parked as
`failed` rather than dropped; `POST .../deliveries/:deliveryId/retry` re-arms
one for another delivery pass (resetting its attempt budget to zero).

Retention is opt-in: `NOTIFICATION_RETENTION_DAYS` (default `0` = keep history
forever) deletes finished rows — `delivered` and `failed` only — whose
`updated_at` is older than the window. In-flight rows (`pending`, `delivering`)
are never touched, so a long backoff is not mistaken for history. The sweep runs
every `NOTIFICATION_SWEEP_INTERVAL_MINUTES` (default `60`); when retention is `0`
the worker is not started at all.

Set `NOTIFICATION_ARCHIVE_DIR` to make retention **archive before deleting**:
expired rows are appended (one JSON object per line) to a daily
`notification-deliveries-YYYY-MM-DD.jsonl` under that directory and only removed
after the file is flushed to disk. If the archive cannot be written the purge
aborts with an error and the rows stay, so retention can never be the step that
destroys the only copy of a delivery history. Leave it empty for the plain
delete.

The archive directory is bounded too. `NOTIFICATION_ARCHIVE_RETENTION_DAYS`
(default `0` = keep the archives forever) deletes daily archive files older than
the window on the same timer, and `{"archive_retention_days": N}` on the sweep
endpoint does it on demand; `archive_removed` reports how many files left the
hot directory. Set `NOTIFICATION_ARCHIVE_COLD_DIR` to **move** expired files
there instead of deleting them; `archive_moved` and `archive_deleted` report the
split. Only files named `notification-deliveries-YYYY-MM-DD.jsonl` are
candidates, so a shared archive directory can also hold the audit archives
without this sweep touching them. Asking for archive retention while
`NOTIFICATION_ARCHIVE_DIR` is unset is a `400` rather than a silent no-op. The
default is "keep forever" on purpose: the archive exists so a purged delivery
stays recoverable, so deleting it has to be requested.

**Aggregation is opt-in noise reduction.** `NOTIFICATION_AGGREGATE_SECONDS`
defaults to `0`, which keeps the behaviour above: one row per event, delivered
on the next worker tick. A positive value holds each new delivery for that many
seconds before its first attempt and folds later events with the same
`(channel, event type, device)` into it; the merged row keeps the first event's
body and gains `occurrence_count` in the delivery log, and the POST carries
`aggregated_count` so a receiver can tell one occurrence from forty. The hold
is fixed rather than extended by each merge, so the worst-case delay is the
window and not the length of the burst — the price of aggregation is promptness,
which is why it is off by default. Only rows still `pending` inside their hold
are merged, so a row the worker is about to send (or has already claimed) is
never mutated.

### 5.3.3 Device access policy (who is allowed to connect)

The access log records who *tried*; the policy decides who *may*. The owner of
the controlled machine sets it, and it is enforced on the device-side
`access-code:verify` endpoint **before** the access code is compared — a
blocked client never gets to guess codes.

| Method | Path | Body | Notes |
|---|---|---|---|
| `GET` | `/v1/me/devices/:device_id/access-policy` | — | Returns `{device_id, remote_disabled, allow_ips: [], max_sessions, allowed_windows: [], timezone, exception_dates: [], approval_required, approval_mode, approval_trusted_ips: [], updated_at?}`. A device with no policy returns `remote_disabled: false, allow_ips: [], max_sessions: 0, allowed_windows: [], exception_dates: [], approval_required: false, approval_mode: "off", approval_trusted_ips: []`. |
| `PUT` | `/v1/me/devices/:device_id/access-policy` | `{remote_disabled: bool, allow_ips: ["10.0.0.0/8"], max_sessions: 1, allowed_windows: [{"days":[1,2,3,4,5],"start":"09:00","end":"18:00"}], timezone: "Asia/Shanghai", exception_dates: [{"holiday":"春节","mode":"deny"},{"repeat":"lunar","lunar_month":8,"lunar_day":15,"mode":"deny"},{"date":"2026-10-17","mode":"allow"}], approval_mode: "grant", approval_trusted_ips: ["10.211.55.0/24"]}` | Full replace. IP entries must be bare IPs or CIDR blocks; `0` max sessions means unlimited; negative is rejected. An unknown `approval_mode` is a `400`. |

`max_sessions` is a concurrency cap on *controllers* (viewers count too, the
host socket does not). It is enforced when a controller authenticates its
signaling socket — after the access code was already accepted — so the log
reads `granted` followed by `denied_session_limit`, and the rejected socket
gets `SESSION_LIMIT_REACHED` **instead of** `auth_ok`.

`allowed_windows` restricts *when* sessions may start, in the policy's
`timezone` (IANA name, default = the server's local zone; the server embeds
the IANA database so this works on Windows and scratch containers):

- `days` are `0`=Sunday … `6`=Saturday; omit `days` for "every day";
- `start`/`end` are `HH:MM` wall-clock times, `end` exclusive;
- `start` later than `end` wraps past midnight and the day filter applies to
  the day the window *starts* (`{"days":[5],"start":"22:00","end":"06:00"}`
  covers Friday night through Saturday 06:00);
- an empty list means "any time".

`exception_dates` override the weekly schedule for specific calendar dates,
evaluated in the same `timezone`:

- `mode: "deny"` blocks the covered day(s) (holidays, maintenance windows) even
  if a window would otherwise allow them — reason
  `today is blocked by an access exception date`;
- `mode: "allow"` permits the covered day(s) and **overrides**
  `allowed_windows` (a one-off work day);
- an `allow` and a `deny` entry may not cover the same day (overlap is rejected
  as ambiguous, with the two offending indexes named); same-mode overlap is
  allowed. An empty list means "no exceptions".

Each entry carries **exactly one** of three date sources:

1. **Fixed date** — `date` is `YYYY-MM-DD`; `end_date` is an optional
   **inclusive** range end and defaults to `date`; `repeat: "yearly"` makes it
   recur every year on the same month-day(s), and a yearly range may wrap the
   new year (`2026-12-30` → `2026-01-02`). A non-yearly range must be ordered.
2. **Statutory holiday** — `holiday` names an entry in the server's holiday
   calendar, e.g. `"春节"`, `"国庆节"`, `"元旦"`. The dates come from the data,
   not from the policy, so the whole holiday break is one entry:
   `{"holiday":"春节","mode":"deny"}`. `holiday_day: "work"` selects the
   make-up workdays (调休上班) instead of the days off, which is how a Saturday
   that the government turned into a workday can be admitted by a weekly
   window that would otherwise skip it:
   `{"holiday":"春节","holiday_day":"work","mode":"allow"}`. An unknown holiday
   name is rejected at write time rather than silently never matching.
3. **Lunar date** — `repeat: "lunar"` with `lunar_month` (1–12) and `lunar_day`
   (1–30) follows the lunar calendar, so `{"repeat":"lunar","lunar_month":8,"lunar_day":15,"mode":"deny"}`
   blocks 中秋 every year without an annual edit. `12/30` falls back to `12/29`
   when that month is short, which keeps a 除夕 entry working across years.

The holiday calendar is the 2026 CN arrangement compiled into the binary
(sourced from the State Council notice) plus, when `HOLIDAY_CALENDAR_DIR` is
set, every `YYYY.json` found there. The files use the public `holiday-cn`
format (`{year, papers, days:[{name,date,isOffDay}]}`), so next year's file can
be dropped in and the server restarted without a rebuild.

`HOLIDAY_CALENDAR_URL` removes the "somebody has to notice the notice" step:
when it is set the server downloads the year files itself into
`HOLIDAY_CALENDAR_DIR` (which is then required) before loading them, so a fresh
deployment starts with next year's arrangement already in place.
`HOLIDAY_CALENDAR_REFRESH_MINUTES` re-checks on that interval; `0` means "fetch
once at startup". The URL is either a template containing `{year}` or a base URL
that `/<year>.json` is appended to, so the published dataset works directly:

    HOLIDAY_CALENDAR_DIR=/var/lib/quickdesk/holidays
    HOLIDAY_CALENDAR_URL=https://cdn.jsdelivr.net/npm/holiday-cn@latest/{year}.json
    HOLIDAY_CALENDAR_REFRESH_MINUTES=1440

Three failure rules matter here. A fetch that fails is logged and **never
fatal**: the embedded year and whatever is already on disk stay in use. A file
is validated before it is written - it has to parse *and* contain dates for the
year it was requested for - so a redirect to an HTML error page cannot replace a
good calendar. And a file whose bytes have not changed is not rewritten, so a
refresh that finds nothing new costs one GET and no churn.

Exception dates only decide the *time* rule. `remote_disabled` and `allow_ips`
are still evaluated first, so an allow date cannot bypass either of them.

`approval_mode` adds a second factor on top of a correct access code
(§5.3.5): instead of a signal token, the session becomes a pending request the
owner must approve. It is not a substitute for the code — the code is still
compared, and the policy gate above still runs. The mode decides *which*
sessions are worth interrupting the owner for:

| `approval_mode` | Asks for approval when… |
|---|---|
| `off` (default) | never |
| `always` | every session, whatever the credential |
| `grant` | the session used a time-boxed guest grant, not the device's own code — the "I lent someone a code" case |
| `untrusted_ip` | the controller is outside `approval_trusted_ips` — the "someone from somewhere new" case |

`approval_trusted_ips` is a list of IPs/CIDR blocks (same syntax as
`allow_ips`) and is **required** for `untrusted_ip`; an empty list there is a
`400` rather than a silent "trust everyone". `approval_required: true` is kept
for compatibility and means `approval_mode: "always"`; the response always
reports the resolved mode, so `{"approval_required": true}` reads back as
`{"approval_required": true, "approval_mode": "always"}`. An unrecognised mode
is rejected at write time — a typo must never quietly turn the gate off.

Approval requests carry a `trigger` (`always` / `grant` / `untrusted_ip`) on
the `device.access_approval.requested` event, and the access timeline says
which rule asked, e.g. `waiting for owner approval (guest code)`.

A controller outside every window is rejected at verify time with
`ACCESS_POLICY_DENIED` / "outside the allowed access window" and shows up as
`denied_policy` on the access timeline — the access code is never compared.

Rejected attempts are recorded in the access log as `denied_policy` (with the
reason) and fan out as the usual `device.access.denied` event, so notification
channels subscribed to denials see policy blocks too. Policy changes are
audited as `device.access_policy.changed`.

> **Where `client_ip` comes from.** Everything that keys on a client address —
> `allow_ips`, `approval_trusted_ips`, the per-IP rate limiters, the access log,
> and the per-source half of the approval queue guard (§5.3.5) — derives it
> from the request peer, honouring `X-Forwarded-For` **only when that peer is a
> trusted proxy**. The default list is loopback plus the private ranges
> (`127.0.0.1/32`, `::1/128`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`,
> `fc00::/7`), which is the usual "reverse proxy on this host or on the LAN"
> topology. Set `TRUSTED_PROXIES` to name a different proxy, or to the literal
> `none` when the server is exposed directly; an unparseable entry is fatal at
> startup rather than ignored, because falling back silently would leave the IP
> gates spoofable while looking configured. Verified with the same
> `X-Forwarded-For: 198.51.100.77` on two peers in one run: from the LAN
> (`192.168.8.81`, untrusted) the server recorded and enforced
> `192.168.8.81` and denied the request on `allow_ips`, while from loopback
> (trusted) it recorded `198.51.100.77` and let the request through to the code
> check. The header is a claim; only a proxy's claim is believed.

**A multi-hop chain is walked right to left.** Each proxy appends to
`X-Forwarded-For`, so the entries nearest the server are the ones a client
cannot forge; the server therefore takes the **rightmost entry that is not
itself a trusted proxy**, falling back to the socket address when the chain is
empty or malformed. Two consequences are worth stating because they are the
whole point:

- A client that **prepends** a fake address does not change its attributed IP.
  With the default trusted list, `X-Forwarded-For: 10.0.0.250, 203.0.113.9`
  from a trusted peer records `203.0.113.9` — the prepended `10.0.0.250` is
  skipped only because it is closer to the *client*, not because it is trusted.
- A real chain of proxies is resolved to the true client. `X-Forwarded-For:
  198.51.100.7, 10.0.0.9` (the inner hop a trusted private proxy) records
  `198.51.100.7`.

If every entry is trusted the leftmost is used, because there is no untrusted
hop to stop at. Verified against the live access-log path (§5.3.3): both chains
above landed as those IPs in `device_access_logs`, and the multi-hop matrix is
pinned by `TestClientIPWalksMultiHopForwardedFor` in
`internal/handler/trusted_proxy_test.go`.

### 5.3.4 Device access grants (time-boxed guest codes)

Handing a guest the device's own access code means handing them a code that
keeps working. A grant is a separate, time-boxed (optionally single-use) code
the owner can issue, hand out, and revoke.

| Method | Path | Body | Notes |
|---|---|---|---|
| `GET` | `/v1/me/devices/:device_id/access-grants` | — | `{items: [...]}` newest first. Each item carries the derived `active` flag (so clients never re-implement the expiry/quota/revocation rules), the `bound_client_id` / `bound_client_public_key` / `bound_ips` binding, and `last_used_client_id` / `last_used_ip` showing who actually redeemed it. |
| `POST` | `/v1/me/devices/:device_id/access-grants` | `{label?, expires_in_minutes, max_uses?, code?, bound_client_id?, bound_client_public_key?, bound_ips?}` | `expires_in_minutes` must be positive; `max_uses` defaults to `0` = unlimited within the window; `code` is generated (8 chars, unambiguous alphabet) when omitted. Returns `201` with the code, which the owner must relay to the guest. A `bound_ips` entry that is not an IP/CIDR, a `bound_client_id` over 64 characters, a `bound_client_public_key` that is not base64 of a raw 32-byte Ed25519 public key, or a public key without a `bound_client_id`, is a `400`. |
| `DELETE` | `/v1/me/devices/:device_id/access-grants/:grant_id` | — | Revokes. Revoking an already-revoked grant is a no-op. |

**Optional guest binding.** A grant is a bearer code by default: whoever holds
it can redeem it. `bound_client_id` and `bound_ips` narrow that:

| Field | Enforced against | Strength |
|---|---|---|
| `bound_client_id` | the `client_id` the controller sends to `access-code:verify` | client-asserted friction: it stops a code being pasted into a different client, but a client can claim any id, so on its own it is not proof of identity — pair it with `bound_client_public_key` to get one |
| `bound_client_public_key` | a signature over the server's challenge, verified against this key | real proof of possession: the code plus the client id are no longer enough, because the redeemer also has to hold the matching private key |
| `bound_ips` | the address the server actually saw | a real boundary: a forwarded code stays useless from anywhere else |

All three are optional and ANDed when set; empty means "anyone holding the
code", which is how grants behaved before the fields existed. A controller that
matches the code but not the binding gets the ordinary `403 INVALID_CODE` — the
same answer as a wrong code, so a leaked code does not confirm itself.

**Proof of possession (`bound_client_public_key`).** When the owner sets this,
the grant can only be redeemed by a controller that proves it holds the private
half. The flow is two steps, and the second one is what actually needs the key:

1. `POST /v1/devices/:device_id/controller-challenges` with `{code, client_id}`
   returns `{challenge, expires_at}` (two minutes). The request carries the code
   and goes through the same rate limiter and device policy as a redemption, so
   this is not a free oracle; the client id must already match the binding. A
   code that is not key-bound answers `400 PROOF_NOT_REQUIRED`, which lets a
   client fall back to plain verification instead of treating it as a failure.
2. `POST /v1/devices/:device_id/access-code:verify` with
   `{code, client_id, challenge, signature}`, where `signature` is base64 of the
   bound key's Ed25519 signature over **the challenge string exactly as
   returned**. Missing, expired, re-bound (another device, grant or client id)
   or wrong-key proofs answer `403 CONTROLLER_PROOF_INVALID` and count as a
   failed attempt, so a leaked code cannot be used to grind against the key.

The path is `controller-challenges` rather than `access-code:challenge` because
gin's router reads the `:` in `access-code:verify` as the start of a wildcard
segment: a sibling sharing that static prefix panics at route-registration
time. It is the same limitation the `webhooks/:id/test` note below describes.

This is the **server half** of the binding. No shipped client generates or uses
these keys yet (that work is in the WebClient/Qt tree, which tracks upstream
PR #23), so today the mechanism is for API integrators; a grant without the key
behaves exactly as it did before.

`access-code:verify` accepts **either** the device's access code or the code of
a usable grant. Two properties matter:

- **The policy still wins.** The device policy (remote switch, IP allow list,
  windows, exception dates) is evaluated before the code comparison, so a grant
  can never open a device the owner has switched off or restricted.
- **A use is only counted when a session actually starts.** Matching a grant
  does not consume it; the counter is incremented after the host-online check,
  so a guest who arrives while the host is offline does not burn a use. The
  increment re-checks the window and quota inside the `UPDATE`, so two
  controllers racing the last slot of a single-use grant cannot both win — the
  loser gets `403 INVALID_CODE`.

Admissions via a grant land on the access timeline as `granted` with the reason
`access grant <id>`, so the owner can tell a guest code from their own code.

Grants also emit realtime events to the owner (§11):

- `device.access_grant.changed` on create and revoke (`action: "created"` /
  `"revoked"`). It deliberately omits the code — the owner already received it
  in the create response and events must not carry credentials. This event is
  also written to the admin audit trail, the same way a policy change is.
- `device.access_grant.used` when a grant admits a session, carrying
  `{grant_id, label, uses, max_uses, client_id}`. Successful admissions are
  intentionally *not* audited (that would add an admin row per healthy session);
  they are visible on the owner-scoped access timeline instead.

### 5.3.5 Device access approvals (second factor)

`approval_mode` on the device policy (§5.3.3) turns "right access code" into
"right access code **and** the owner said yes", for the sessions the owner
chose to gate (`always`, `grant`, or `untrusted_ip`). It exists for the case
the code alone cannot cover: a guest who was given a code walks up at a bad
moment, or someone connects from an address the owner does not recognise.

The flow:

1. the controller calls `access-code:verify` (§6.5) with a correct code or
   grant while the host is online;
2. instead of a signal token, the server creates a pending approval and
   answers `202` with `{approval_id, approval_secret, status: "pending",
   expires_at, poll_after_ms}`. The secret is shown **once** and stored only as
   a SHA-256 hash;
3. the owner sees it on `GET /v1/me/devices/:device_id/access-approvals`
   (the request also fans out as `device.access_approval.requested`), and
   approves or denies;
4. the controller polls `POST /v1/devices/:device_id/access-approvals:claim`
   with `{approval_id, secret}` until it is granted, then receives the usual
   `{signal_token, expires_at}`.

| Method | Path | Auth | Body / notes |
|---|---|---|---|
| `GET` | `/v1/me/devices/:device_id/access-approvals` | user token | `?status=pending&limit=50`. `status` is a comma-separated subset of `pending,approved,denied,expired,consumed` and defaults to `pending`. Returns `{items: [...]}` newest first. |
| `GET` | `/v1/me/devices/:device_id/access-approvals:summary` | user token | `?since=` (RFC3339, default the last 24 hours) and `&limit=` (how many refused sources, default 5, capped at 10). Returns `{since, total, by_status, denied_sources, denial_cooldown_seconds, denial_cooldown_multiplier, denial_cooldown_max_seconds}` — see below. |
| `POST` | `/v1/me/devices/:device_id/access-approvals/:approval_id/approve` | user token | Optional `{reason}`. `200` with the updated row. A request that was already decided (or timed out) returns `409` with the current `status` instead of silently flipping it. |
| `POST` | `/v1/me/devices/:device_id/access-approvals/:approval_id/deny` | user token | Optional `{reason}`. Same `200` / `409` shape. |
| `POST` | `/v1/devices/:device_id/access-approvals:claim` | `X-API-Key` | `{approval_id, secret}` from step 2. `202` with `{status: pending, expires_at, poll_after_ms}` while pending, `200` with `{status: granted, signal_token, expires_at}` once approved, `403` when denied or expired, `404` for an unknown id *or* a wrong secret (they are deliberately indistinguishable), `409` when the approval was already used. |

**The summary is the answer to "is somebody hammering me".** The row list shows
requests one at a time, so noticing that a single address has been refused six
times this afternoon means counting by eye. `access-approvals:summary`
aggregates a window into `{since, total, by_status, denied_sources,
denial_cooldown_seconds}`: `by_status` always carries all five states (zeroes
included, so a missing key never has to mean "unknown"), and `denied_sources`
lists the refused sources most-refusals-first with each one's
`last_denied_at`. The window is keyed on `created_at` for every count — "the
requests that arrived in this window" — so a denial of an older request does not
appear even if the decision happened after the window opened.
`denial_cooldown_seconds` echoes the live `APPROVAL_DENIAL_COOLDOWN_SECONDS`
and `denial_cooldown_multiplier` / `denial_cooldown_max_seconds` echo the two
escalation settings (§5.3.5 rules below). Together they are what lets an owner
tell "this source is being refused right now" from "it was refused a while
ago", and — when the window escalates — understand why a repeat offender is
waiting longer than the base interval suggests.

Rules that matter:

- The approval is **single-use**. The `approved → consumed` transition is a
  single `UPDATE` re-checking the status, so two controllers racing the same
  approval cannot both mint a token — the loser gets `409`.
- **The host is re-checked at claim time.** An approval is not consumed while
  the host is offline; the controller gets `409 HOST_OFFLINE` and can claim
  again once the device is back.
- **A grant still spends its use only when the session can start.** For a
  grant-backed approval, the grant counter is incremented during the claim,
  after the host-online check — the same property as the non-approval path.
- A pending approval **expires** after 2 minutes (`DeviceAccessApprovalDefaultTTL`);
  the client then sees `403` on claim and the row reads `expired`.
- **A request is refused while one is already waiting.** A correct code is not
  a licence to fill the owner's decision queue: the same client asking again
  gets `429 RATE_LIMITED` with `Retry-After` until its pending request expires,
  and a device never accumulates more than 5 unanswered requests from any mix
  of sources (the sixth is `429` too). Both refusals mean "wait", not "wrong
  code" — the code was right. Once the pending request expires, the same client
  may ask again.
  The per-source half keys on the address the server saw, which is the
  `X-Forwarded-For`-derived client IP — see the trusted-proxy caveat in
  §5.3.3: behind a proxy that does not strip or overwrite the header, a client
  can choose its own `client_ip`.
- **A denial is followed by a cooldown.** Otherwise "no" would just clear the
  pending slot and the same source could ask again on the next tick, turning a
  correct access code into a way to keep paging the owner. After the owner
  denies, that source is refused for `APPROVAL_DENIAL_COOLDOWN_SECONDS`
  (default `300`; `0` disables it) with `429 RATE_LIMITED` and a `Retry-After`
  that says exactly when the block lifts. The window is per `(device, source)`,
  so one stranger's denial cannot lock out anybody else, and it is keyed on the
  same `client_ip` as the per-source guard above.
- **Repeated denials can cost more.**
  `APPROVAL_DENIAL_COOLDOWN_MULTIPLIER` (default `1`) multiplies the window for
  a source that has been refused again and again: with a multiplier of `3`, the
  third consecutive refusal waits `3 × 3 × 300s` instead of `300s`.
  `APPROVAL_DENIAL_COOLDOWN_MAX_SECONDS` (default `0`, meaning uncapped) is the
  ceiling that stops the growth; set it whenever the multiplier is above `1`,
  since otherwise the window grows without bound as the streak grows. The
  streak counts consecutive refusals **for that `(device, source)` pair** and
  is read from the decision history, so an approval — or a request that merely
  expired because nobody answered — ends it: only an explicit "no" escalates,
  and the source is never permanently banned. The default multiplier of `1`
  reproduces the fixed window exactly, so a deployment that sets neither
  variable keeps the behaviour it had.
- Access-timeline rows read `pending_approval` on request, `denied_approval`
  when the owner refuses, and `granted` when the claim succeeds. Approving or
  denying is also written to the admin audit trail (it is an admission
  decision), while healthy admissions stay owner-scoped.

### 5.3.6 Device offline alerts ("tell me when this stays offline")

`device.online.changed` fires the moment a heartbeat stops, which is the same
event whether the machine is rebooting, asleep, temporarily unplugged or dead.
An offline alert adds the missing dimension — **how long** — as an owner-set
rule per device:

| Method | Path | Auth | Body / notes |
|---|---|---|---|
| `GET` | `/v1/me/devices/:device_id/offline-alert` | user token | The rule, or the defaults (`enabled: false`, `after_minutes: 10`) when none is configured. Also returns the evaluator's episode state. |
| `PUT` | `/v1/me/devices/:device_id/offline-alert` | user token | `{enabled, after_minutes, repeat_minutes?, snoozed_until?, maintenance_windows?, maintenance_timezone?}`. `after_minutes` is 1…10080 (7 days); omitting it (or sending `0`) stores the default of 10. `repeat_minutes` is 0 (one alert per outage, the default) or 1…10080. `snoozed_until` is an RFC3339 instant; omitting or sending `null` clears an existing snooze. `maintenance_windows` is a list of recurring quiet periods, each `{"days":[0-6],"start":"HH:MM","end":"HH:MM"}` (at most 20); `maintenance_timezone` is an IANA zone name (empty = the server's local zone). Omitted/empty clears them. Out-of-range or malformed values are `400`. |
| `DELETE` | `/v1/me/devices/:device_id/offline-alert` | user token | `204`. Idempotent: deleting a rule that does not exist is still `204`. |

Response:

```json
{"device_id": "256983850", "enabled": true, "after_minutes": 10, "repeat_minutes": 0,
 "maintenance_windows": [{"days":[0],"start":"02:00","end":"04:00"}], "maintenance_timezone": "Asia/Shanghai",
 "in_maintenance_window": false,
 "offline_since": "2026-10-16T04:10:17Z", "last_alerted_at": "2026-10-16T04:11:17Z"}
```

`in_maintenance_window` is computed at read time, so a client can show "quiet
right now because of a recurring window" without re-implementing the schedule.
The window shape and its midnight-wrap rule (`start` later than `end` means the
window runs into the next day; `start == end` means all day on the selected
days; an empty `days` list means every day) are identical to device access
windows, so operators only have to learn one model.

A background evaluator (`OFFLINE_ALERT_INTERVAL_SECONDS`, default 30; `0`
disables it) turns presence into an episode state machine:

1. **first** tick that sees the device offline records `offline_since` — this
   is what makes "offline for ten minutes" answerable, because Redis only knows
   whether the heartbeat exists *right now*;
2. while `now - offline_since < after_minutes` nothing happens;
3. once the threshold is crossed the evaluator publishes
   `device.offline.sustained` and stamps `last_alerted_at`. With
   `repeat_minutes: 0` that is the only message for the episode; with a
   positive value it keeps reminding at that interval until the device comes
   back. The event is user-scoped, so it reaches the owner's realtime stream
   and — if a channel subscribes to it — the durable notification outbox;
4. when the device comes back, `offline_since` and `last_alerted_at` are
   cleared, so the **next** outage alerts again instead of being suppressed by
   the first one.

Payload of the alert:

```json
{"device_id": "256983850", "device_name": "办公机", "offline_since": "...",
 "offline_minutes": 11, "threshold_minutes": 10,
 "reminder": false, "repeat_minutes": 0}
```

`reminder` distinguishes "this device just crossed the threshold" from "this is
still down" (`offline_minutes` is `> threshold_minutes` on a reminder), so a
receiver can route the two differently without diffing timestamps.

Four behaviours are worth knowing because the obvious implementation gets
them wrong:

- **A deliberately logged-out device is not an outage.** When
  `devices.logged_in` is false the episode is cleared and nothing is published;
  alerting about a machine the owner parked would only train them to ignore the
  alerts.
- **Changing the threshold does not restart the clock.** `offline_since` is
  preserved across a `PUT`, so raising the threshold from 1 to 10 minutes keeps
  the episode's real start; lowering it re-evaluates immediately.
- **Disabling the rule clears the episode.** Switching it off means "stop
  telling me about this device", so switching it back on gives a clean slate
  rather than an alert about an outage that began while it was off.
- **A snooze silences messages, not the clock.** `snoozed_until` suppresses
  publishing while it is in the future, but `offline_since` keeps running and
  the moment it expires the alert (or the next reminder) goes out — a
  maintenance window should not make the outage invisible, only quiet.
- **A recurring window behaves like a snooze that repeats.** While `now` falls
  inside any configured `maintenance_windows` entry the evaluator publishes
  nothing, but the episode clock keeps running exactly as with a snooze, so a
  weekly backup window is one setting instead of a fresh `snoozed_until` every
  time. `in_maintenance_window` in the response tells the owner why it is
  quiet.

Both `offline_since` and `last_alerted_at` are written with conditional
`UPDATE`s (`... WHERE offline_since IS NULL` and `... WHERE alerted_at IS NULL`),
so two signaling instances cannot both start the same episode or both publish
one alert for it.

### 5.3.7 Self-hosted update channel

The installed clients ask this server which build they should be running,
instead of asking upstream. The server owns the **index** - which version
exists for which channel/platform/arch, where the bytes live and what they hash
to. The artifact normally stays on whatever static host the operator controls
(`download_url`, an absolute http(s) URL), but the operator may also hand the
installer to this server, which then serves it itself. Publishing is a
super-admin operation.

| Method | Path | Body | Notes |
|---|---|---|---|
| `GET` | `/v1/update:check?platform=&arch=&channel=&current_version=` | — | Client-facing. API-key gated, **not** user-authenticated: a controlled machine checks before anyone logs in. Returns `{update_available, mandatory, channel, platform, arch, current_version?, latest}`, where `latest` is always present and is `null` when the channel has no release for that platform/arch. `current_version` is optional; without it any release counts as an update. A publish that is not strictly newer than `current_version` reports `update_available: false`. |
| `GET` | `/v1/update-releases/:id/artifact` | — | Client-facing. Streams an installer this server hosts itself; the same API key as the check, and supports `Range`. `404` when the release's bytes live on an external host or the release is unknown. |
| `GET` | `/v1/admin/update-releases?channel=&platform=&arch=&limit=` | — | Super-admin. Newest first. |
| `POST` | `/v1/admin/update-releases` | `{channel?, platform, arch, version, download_url?, sha256, size_bytes?, notes?, mandatory?, published_at?}` | Super-admin. `channel` defaults to `stable`; a non-empty `download_url` must be absolute http(s); `sha256` must be the 64-character hex digest of the artifact. Publishing the same channel/platform/arch/version twice is `409` rather than a silent replacement. Audited as `update_release.publish`. |
| `POST` | `/v1/admin/update-releases/:id/artifact` | Installer bytes. `multipart/form-data` with a `file` part, or a raw body plus `?filename=`. | Super-admin. Re-hosting: stores the bytes under `UPDATE_ARTIFACT_DIR` and points `download_url` at `GET /v1/update-releases/:id/artifact`. The upload is refused with `400` unless it hashes to the `sha256` the release already advertises, so re-hosting can never swap the bytes clients were told to expect. Audited as `update_release.artifact.upload`. |
| `PATCH` | `/v1/admin/update-releases/:id/artifact?filename=` | One chunk (raw body). | Super-admin. Resumable upload, same protocol as the recording artifact: `Upload-Offset` (required) is where this chunk starts, `Upload-Length` (optional, `*` while unknown) declares the total, `Upload-Complete: 1` ends the upload when no total was declared and `Upload-Probe: 1` asks where the upload stands without staging bytes. `202` stages the chunk and returns the release row with the authoritative offset in the `Upload-Offset` response header; `200` means this chunk finished the upload and the digest matched; `409` means the offset did not match and that same header carries the offset the server actually holds. The digest is only knowable once the last chunk lands, so a mismatch is reported then (`400`) and the staged bytes are discarded rather than hosted. |
| `POST` | `/v1/admin/update-releases:sweep` | — | Super-admin. Runs the artifact-store cleanup once and returns `{uploads_reaped, bytes_reaped, orphans_removed, orphan_bytes}`. The periodic worker calls the same pass. Audited as `update_release.sweep`. |
| `DELETE` | `/v1/admin/update-releases/:id` | — | Super-admin. Deletes the row **and** any stored installer. `404` for an unknown id. Audited as `update_release.delete`. |

A release published without a `download_url` is a **placeholder**: the operator
has decided the version exists but has not stored the bytes yet. It is listed in
the admin index and is invisible to clients - `update:check` skips it - because
announcing an update nobody can download is worse than announcing nothing. The
upload hashing to the published `sha256` is what makes it fetchable, and
`hosted_here` in the check response says whether `download_url` is this server's
own artifact route or the operator's static host.

Re-hosting is configured by `UPDATE_ARTIFACT_DIR` (default `data/updates`, never
mounted as a static site), `UPDATE_MAX_UPLOAD_MB` (default 4096) and
`UPDATE_PUBLIC_BASE_URL`. Set the last one whenever a reverse proxy terminates
TLS in front of this server: the stored `download_url` is absolute, and without
an explicit base it is built from the host on the admin upload request (which
would otherwise be `http://` behind an HTTPS proxy).

A resumable upload is keyed by the release id **and** the file name, because that
name is what the download route sends back. Starting a second upload under a
different name while one is staged is a `400` that names the staged file, so a
typo cannot silently abandon hundreds of megabytes. A single-shot `POST` to the
same release starts over from zero, which is what "here is the whole file"
means.

The store is cleaned by a periodic pass (`UPDATE_SWEEP_INTERVAL_MINUTES`, default
60; `0` disables the worker but keeps the manual trigger) with two rules, which
is also what `POST /v1/admin/update-releases:sweep` reports separately:

- a staging file (`<name>.part`) untouched for `UPDATE_STALE_UPLOAD_MINUTES`
  (default 360) is an upload nobody came back to and is deleted. Age alone
  decides: a half-finished upload legitimately belongs to a live release. The
  trade-off is visible to clients — a paused upload older than the window has to
  restart from zero, which is the operation a resumable upload exists to make
  cheap;
- a stored artifact that is old enough (one hour) and that no release row claims
  is an orphan and is deleted. Every write path removes its own file on failure,
  so this is the crash-shaped case: a rename that happened, a database write that
  did not. The row is compared by path, so re-uploading a release under a new
  name leaves the old file reclaimable.

Versions are dotted numbers (`4.2.5.0`); a leading `v` is accepted and missing
trailing parts count as zero, so `4.2` and `4.2.0.0` are equal. `channel` is
`stable` (the default), `beta` or `nightly`; `platform` is `windows`, `macos`
or `linux`; `arch` is `x64`, `arm64` or `universal`. `mandatory` is echoed only
when an update is actually available, so a client can tell "this update is
required" from "there is nothing to do".

### 5.3.8 Remote sessions (who is in my machine right now)

A **remote session** is one controller holding a signaling session against a
device. It is opened when that controller's signaling socket completes
authentication, and closed when that socket ends - which is the whole lifetime
this server can observe, because the media path is peer to peer. A session whose
signaling channel is gone is over as far as the control plane is concerned,
whatever the peers do with their RTP afterwards.

This is the queryable half of what the device access log already records: the
log says "someone got in" and never says when they left, so `status=active`
could not be answered from it.

| Method | Path | Notes |
|---|---|---|
| `GET` | `/v1/me/remote-sessions?status=&device_id=&cursor=&limit=` | Owner only. `status` is `active` (who is in right now), `ended` (history) or omitted/`all` for both; anything else is a `400`. Cursor-paginated, newest first, `next_cursor` absent on the last page. |

Each row carries `session_id` (the signaling id the server assigned),
`device_id`, `client_id` (the controller-declared identifier, the same value the
access log records, which is what lets the two be read together), `client_ip`,
`started_at`, and - once it is over - `ended_at` and `end_reason`. Two derived
fields save every client from inventing its own answer: `active` is
`ended_at == null`, and `duration_seconds` is the run length so far for a session
that is still counting and the total once it has stopped.

`end_reason` is `disconnected` in the ordinary case (the controller closed, the
client crashed, or the network dropped - the server cannot tell those apart and
does not pretend to) and `server_restart` for rows a restart found still open: a
signaling socket cannot outlive the process that held it, so those sessions
really did end, they just ended without the handler's close running. Unbound
devices are recorded with `user_id: 0` and never appear in an owner's list.

The owner's realtime stream (`/v1/realtime/events`, §6) carries the transitions
as `remote_session.started` and `remote_session.ended`, both with the same fields
the list returns, so a client can keep a live view without polling. This is
deliberately not the `session.*` namespace, which is the caller's own login
sessions (`GET /v1/me/sessions`, `session.revoked`); those are a different
resource with a different lifetime.

### 5.4 Connections (history)

| Method | Path | Body |
|---|---|---|
| `GET` | `/v1/me/connections?cursor=&limit=&since=` | — |
| `POST` | `/v1/me/connections` | `{device_id, duration, status: "success"\|"failed", error_msg?}` |
| `GET` | `/v1/me/recordings?cursor=&limit=` | — |
| `POST` | `/v1/me/recordings` | `{device_id, device_name?, session_id?, status, format?, artifact_path?, artifact_url?, thumbnail_path?, duration_seconds?, size_bytes?, started_at?, ended_at?}` |
| `GET` | `/v1/me/recordings/:id` | — |
| `PATCH` | `/v1/me/recordings/:id` | `{status?, format?, artifact_path?, artifact_url?, thumbnail_path?, duration_seconds?, size_bytes?, ended_at?}` |
| `DELETE` | `/v1/me/recordings/:id` | Deletes the row **and the stored media/preview files** it points at. |
| `POST` | `/v1/me/recordings/:id/artifact` | Media bytes. `multipart/form-data` with a `file` part, or a raw body plus `?ext=.mp4\|.mkv\|.webm\|.h264\|.grf`. |
| `PATCH` | `/v1/me/recordings/:id/artifact?ext=.mp4` | Resumable upload: one chunk per request, positioned by the `Upload-Offset` header and optionally bounded by `Upload-Length` (total, or `*` while unknown). `202` stages the chunk and returns the recording row with the authoritative offset in the `Upload-Offset` response header; `200` means this chunk finished the upload and returns the finalized row. Both success statuses use the same `SessionRecording` schema because a generated client declares one success type per operation. `409` means the offset did not match and the header carries the offset the server actually holds. A status probe asks where the upload stands without staging anything: send `Upload-Probe: 1` with a valid offset, and the server ignores the body (the generated clients use this because they cannot send a zero-length body). |
| `GET` | `/v1/me/recordings/:id/artifact` | Streams the stored media back. Owner only. |
| `POST` | `/v1/me/recordings/:id/artifact-ticket` | Mints `{url, expires_at}` for playback without an `Authorization` header. |
| `POST` | `/v1/me/recordings/:id/thumbnail` | Preview image (JPEG or PNG) for the recordings list. `multipart/form-data` with a `file` part, or a raw body plus `?ext=.jpg\|.jpeg\|.png`. |
| `GET` | `/v1/me/recordings/:id/thumbnail` | Streams the stored preview back. Owner only. |
| `POST` | `/v1/me/recordings/:id/thumbnail-ticket` | Mints `{url, expires_at}` for the preview, same rules as the artifact ticket. |

`session_recordings` is the control-plane index for remote-session media. The
recording engine owns the artifact and reports its metadata through
`POST /v1/me/recordings`. The bytes themselves are pushed with
`POST /v1/me/recordings/:id/artifact` and read back through the matching `GET`
route — there is deliberately **no static directory mount**, every download is
ownership-checked.

`GET /v1/me/recordings` is cursor-paginated: it returns
`{items, next_cursor}` newest-first, `limit` defaults to `50` and caps at `200`,
and the envelope carries **no** `total` because the endpoint does not count the
whole set on every page. Follow `next_cursor` until it is absent rather than
trusting a count. The machine-readable contract covers the list endpoint plus
the media half of this surface: `GET .../artifact`, `GET .../thumbnail`,
`POST .../artifact-ticket`, `POST .../thumbnail-ticket`, and the ticket-gated
`GET /v1/recordings/{id}/artifact|thumbnail?t=`, and the whole write side —
`POST`/`PATCH`/`DELETE`, both multipart uploads and the resumable `PATCH
.../artifact`.

Signed playback links: `<video src>`, `window.open` and `<a download>` cannot
carry an `Authorization` header, so those two routes have ticket variants.
`POST .../artifact-ticket` and `POST .../thumbnail-ticket` return a URL of the
form `/v1/recordings/:id/artifact?t=...` (or `/thumbnail?t=...`) signed with
`RECORDING_URL_SECRET` (falling back to `API_KEY`), valid for
`RECORDING_TICKET_TTL_SECONDS` (default `600`). The two routes live outside the
API-key and user-auth groups because a top-level browser navigation carries
neither header; the ticket is the credential. Each ticket is scoped to one
media kind, so a playback link cannot be replayed against the preview route.

**Media reads are audited, once per burst.** Every artifact and thumbnail read
on the owner routes *and* on the ticket routes writes a `recording.download` /
`recording.thumbnail.download` row with the reader in `actor_user_id` and their
observed address in `ip` — the same action names the admin download routes use,
so one filter shows every read and the actor column says whether it was the
owner or an administrator. Because a browser fetches a recording in several
range requests and re-renders thumbnails as the list scrolls, the write is
deduplicated to one row per `(actor, action, recording)` within 5 minutes; the
ticket route adds `via=ticket` to the details. The trail therefore answers "who
opened this recording, roughly when, and where from" without recording transport
details, and a leaked playback link shows up as a read by the user the ticket
was minted for.

Upload behaviour worth knowing:

- the stored file name is derived from `(user_id, recording_id)` and a
  whitelisted extension (`.mp4 .m4v .mkv .webm .h264 .grf`); the uploaded
  filename never reaches the filesystem, so traversal is not possible;
- bytes are written to `<name>.part` and renamed, so a dropped upload cannot
  leave a half-written artifact under the real name;
- the upload sets `artifact_url` (server-relative), `size_bytes` and `format`,
  and flips `status` from `recording` to `completed`. A `failed` status is left
  alone so a client can still report failure;
- the size limit is `RECORDING_MAX_UPLOAD_MB` (default `2048`), and files live
  under `RECORDING_DIR` (default `data/recordings`).
- **A dropped upload can resume instead of starting over.**
  `PATCH /v1/me/recordings/:id/artifact?ext=.mp4` takes one chunk per request
  and answers with the number of bytes it actually holds, in the
  `Upload-Offset` response header. The client tells the server where the chunk
  starts; when the two disagree the answer is `409` with the authoritative
  offset in that same header, so a client that lost its bookkeeping resyncs
  without re-sending anything. `Upload-Length` (total, or `*` while unknown)
  lets the server finish by itself when the last byte lands and refuses a chunk
  that would overshoot; `Upload-Complete: 1` finishes an upload whose total was
  never declared. An empty body with a valid offset is a probe — it stages
  nothing and reports the current offset, which is how a client that crashed
  re-learns where it stopped. A staging file that a client starts under one
  extension cannot be resumed under another (that is a `400` naming the
  extension already staged), the total is capped exactly like the single-shot
  upload, and an empty artifact is refused on every path.
- Staging files are not immortal: the recording lifecycle deletes a `.part`
  file that has not been written for six hours, reported as `uploads_reaped` by
  `POST /v1/admin/recordings:sweep`. A paused upload older than that must
  restart, which is the operation resumable upload exists to make cheap.

Previews follow the same storage rules with a much smaller budget: the cap is
2 MiB (`min(2 MiB, RECORDING_MAX_UPLOAD_MB)`), the file is named
`thumb-<recording_id>.<ext>` next to the media, and only `.jpg`/`.jpeg`/`.png`
are accepted. Unlike an artifact, the bytes are **sniffed** (JPEG `FF D8 FF`,
PNG signature) and the declared extension must match them, because the
recordings list renders this image inline. A successful upload writes
`thumbnail_url` and removes any preview left under the other extension. Only
the relative URL is stored, so `thumbnail_url` is never an absolute host and
the value survives a redeploy behind a different name.

Status lifecycle: a client opens the row as `recording`, uploads the bytes, and
the server flips it to `completed` when they arrive. If the upload never lands the
client marks the row `failed`. If a client dies mid-capture the row is repaired by
the stale sweep below, so `recording` is never a terminal state.

Retention is opt-in: `RECORDING_RETENTION_DAYS` (default `0` = keep media
forever) deletes media files older than the window while **keeping the index
row**; `artifact_path`/`artifact_url`/`size_bytes` are cleared, so playback
stops being offered and the artifact route answers `404`. A stored preview is
deleted by the same pass and `thumbnail_path`/`thumbnail_url` are cleared with
it, so a purge never leaves an image orphaned on disk. `RECORDING_STALE_MINUTES`
(default `240`) is how long `recording` may stay open before it is marked
`failed`. Both passes run every `RECORDING_SWEEP_INTERVAL_MINUTES` (default `30`)
and can be triggered on demand through the admin endpoints below.

Deleting a recording (`DELETE /v1/me/recordings/:id` or the admin equivalent)
drops the media and the preview along with the row. The same pass also reaps
**orphan files** - stored files under `RECORDING_DIR` whose index row is gone,
for example because an unlink failed earlier - once they are more than an hour
old. Upload temporaries (`.part`) are never matched, so an in-flight upload
cannot be reaped.

### 5.5 Favorites

| Method | Path | Body |
|---|---|---|
| `GET` | `/v1/me/favorites` | — |
| `POST` | `/v1/me/favorites` | `{device_id, device_name?, access_password?}` |
| `PATCH` | `/v1/me/favorites/:device_id` | `{device_name?, access_password?}` |
| `DELETE` | `/v1/me/favorites/:device_id` | — |

`FavoriteItem` shape: `{device_id, device_name, access_password, created_at}`.

### 5.6 Scheduled tasks (owner-scheduled control-plane actions)

| Method | Path | Body |
|---|---|---|
| `GET` | `/v1/me/scheduled-tasks` | — |
| `POST` | `/v1/me/scheduled-tasks` | `{kind, run_at?, repeat_seconds?, max_attempts?, schedule?, timezone?, skip_holidays?}` |
| `DELETE` | `/v1/me/scheduled-tasks/:id` | — |

`run_at` is RFC3339 and must be in the future; `repeat_seconds` is `0` for a
one-shot task or `60`…`31536000` to repeat. `GET` also returns
`supported_kinds` — which is the authoritative list of what may be scheduled —
and `supported_schedules`, the accepted calendar forms. A kind the server cannot
run is rejected at create time rather than stored as a row nothing will ever
execute.

#### Calendar schedules

`repeat_seconds` covers "every 15 minutes". For "every night at 22:00" or "the
1st of the month", send `schedule` instead:

| Form | Meaning |
|---|---|
| `daily@HH:MM` | every day at that local time |
| `weekdays@HH:MM` | Monday to Friday at that local time |
| `weekly@<dow>@HH:MM` | that weekday — `mon`…`sun`, or `0`…`6` with `0` = Sunday |
| `monthly@<dom>@HH:MM` | that day of the month, `1`…`31` |
| `every@<N>@days@HH:MM` | every `N` days (`1`…`365`), phased from the task's creation date |

`HH:MM` is a 24-hour clock time. `timezone` is an IANA zone name
(`Asia/Shanghai`), and it is the local clock the schedule is read on; omitting it
means UTC. The first occurrence is derived from the schedule too — you do not
send `run_at` with a `schedule`, and the API rejects the two together, or a
`schedule` combined with `repeat_seconds`, because those would be two
contradictory answers to "when".

Two rules the server states up front rather than leaving to inspection:

- `monthly@31` **clamps** to the last day of a shorter month, so it runs on
  February 28 (29 in a leap year) and April 30 instead of silently skipping
  those months. A maintenance task that must run every month is the more useful
  contract.
- A wall-clock time that a DST transition deletes still runs: the task fires at
  the first real instant at or after it, so a `daily@02:30` in a US spring
  forward runs at 03:00 rather than an hour early. This is the rule cron and
  systemd timers use for a skipped hour.
- `every@<N>@days@HH:MM` keeps its **phase**: it is anchored to the day the
  task was created, so "every 3 days" runs on the same dates forever even if
  one occurrence runs late or the server restarts. It is not "N days from the
  last run", which would drift.

**`skip_holidays`** moves a calendar occurrence off a statutory day off — the
same calendar the device access policy uses for its `holiday` exception
entries — keeping the time of day, and only applies together with `schedule`
(an interval task has no calendar day to skip, so the combination is a `400`).
The walk is day by day because the Spring Festival break is longer than a week,
and it is bounded so a broken calendar file cannot trap a task forever.
`weekdays@HH:MM` and `skip_holidays` are separate on purpose: "Monday to
Friday" and "not a national holiday" are different questions, and a 调休
make-up workday is still a weekday.

Every response row carries `schedule`, `timezone` and `skip_holidays` back, and `next_run_at` is
always the concrete occurrence the worker will fire — which is also how you can
confirm that "05:30 Shanghai" was understood as 05:30 Shanghai and not 05:30
UTC.

Only kinds whose whole effect lives in the control plane are registered, so a
task never silently depends on a client that is not there. Today that is one
kind:

- **`sign_out_all_sessions`** — revokes every active session family of the
  owner, i.e. "log out everywhere at 22:00". It is the same action
  `PUT /v1/me/password` performs, just scheduled, and it works because both the
  session records and the revocation live in the control plane.

Each row carries its outcome: `last_run_at`, `last_status` (`ok` / `failed`) and
`last_error`, so a caller can see that a task fired and what happened without
reading server logs. One-shot tasks are **disabled** after they run rather than
deleted, so the history survives.

**Failures can be retried.** `max_attempts` (default `1`, allowed `1`…`10`) is
how many times one occurrence may be attempted; `1` keeps the original
"fire once and record the error" behaviour. When a run fails and the budget is
not spent, the task is re-armed after a short backoff (30s, 1m, 2m, 4m, 8m, then
held at 10m) instead of waiting a whole period — which matters most for a
one-shot task, where the next period never comes. A retry never pushes a
recurring task past its own next period, so it recovers the failed occurrence
without drifting the schedule. The live counter is `attempts`: reset to `0` on
success, kept on a parked one-shot so the operator can see how many tries it
got, and reset for a recurring task so the next period starts fresh. A parked
one-shot stays `enabled: false` with `last_error` set.

**Giving up is announced.** When a task spends its retry budget the scheduler
publishes a user-scoped `scheduled_task.failed` event carrying `task_id`,
`kind`, `status`, `attempts` and `error`. It fans out to the owner's realtime
WebSocket and to any notification channel that subscribed to it (§5.5), so
"I scheduled something and it silently never ran" is answerable without polling
the row. Per-attempt failures are deliberately *not* events - they live on the
row (`attempts`, `last_error`) and in the audit trail - because an alert per
retry teaches the owner to ignore it.

The worker (`SCHEDULED_TASK_INTERVAL_SECONDS`, default `30`; `0` disables it)
claims due rows with `FOR UPDATE SKIP LOCKED` and advances the schedule *inside*
the claiming transaction, before the action runs. That makes the schedule safe
across instances — the same occurrence can never fire twice — at the cost of
at-most-once semantics: a crash in the window between the claim and the action
skips that occurrence. A recurring task whose worker was down for several
periods fires once and reschedules from now rather than replaying every missed
occurrence; a calendar task reschedules from now by recomputing the next
calendar occurrence, not by adding a fixed period. Every run writes an audit
row, `scheduled_task.run`, with the kind, the outcome and the attempt number
(`attempt=2/3`), so a retry is visible in the trail and not only in the row's
counter. A row whose `schedule` cannot be parsed — a hand-edit, or a value
written by a newer server — is parked (`enabled: false`, `last_status: failed`,
`last_error` naming the problem) instead of being claimed on every tick.

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

`code` may be the device's access code **or** the code of an active access grant
(§5.3.4). The device policy gate runs first either way, and a grant only spends
one of its uses once a session actually starts.

Auth: `X-API-Key` **or** `Origin` in the admin-configured `allowed_origins`
whitelist (whichever the request can satisfy). Browsers depend on the second
form.

Request:

```json
{ "code": "633732" }
```

`client_id` names the controller and is what a grant's `bound_client_id` is
compared against. When the matched grant also carries a
`bound_client_public_key`, the request must add `challenge` and `signature`
(both from `POST /v1/devices/:device_id/controller-challenges`, §5.3.4); without
them the answer is `403 CONTROLLER_PROOF_INVALID`.

Response (200):

```json
{ "status": "granted", "signal_token": "stk_01H...", "expires_at": "2026-05-12T08:53:39Z" }
```

When the device policy has `approval_required` (§5.3.3), a correct code returns
`202` instead, and no token is minted until the owner approves and the
controller claims the approval (§5.3.5):

```json
{
  "status": "pending",
  "approval_id": 42,
  "approval_secret": "9c5cabcd...",
  "expires_at": "2026-10-16T00:40:18Z",
  "poll_after_ms": 2000
}
```

Both answers are the same schema (`AccessCodeVerifyResponse`, distinguished by
`status`), and the claim route's `200`/`202` pair works the same way. That is not
cosmetic: a generated client declares a single success type per operation, so a
second schema would leave the `202` body undecodable in every committed SDK -
the Go client would return a token-shaped struct with empty approval fields and
the caller could not poll.

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
5. Server sends:  {"type":"bootstrap_done", "server_rev":4218}
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
in-place. Clients must apply snapshot/replay/events only when their
`server_rev` is newer than the last applied revision; stale frames are ignored.
Clients must not depend on HTTP `/v1/me/devices` responses to order or repair
realtime state, because HTTP and WS frames can arrive in either order.

If the gap is too large (Redis stream has been trimmed), the server replies
`{type:"snapshot_required"}` and then sends a fresh authoritative `snapshot`
on the same connection. Clients may also choose to reconnect without
`since_rev`, but they must remain compatible with the same-connection snapshot
path.

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
| `GET` | `/v1/admin/system/logs` — the operational files an operator may read: `{items: [{name, size_bytes, modified_at}]}` from a fixed server-side allow-list (`signaling.log`, plus `postgres.log` / `redis.log` when `POSTGRES_LOG_FILE` / `REDIS_LOG_FILE` are set). Super-admin only |
| `GET` | `/v1/admin/system/logs:search` — search a log instead of downloading it. `?q=` (case-insensitive substring), `&match=` (**full text** on both scopes: words are ANDed, `"a phrase"` is a phrase, `OR` and a leading `-` negation are supported; the two modes are separate because a substring finds a request id while full text finds words without matching them apart; `scope=local` compiles the expression in Go and `scope=all` evaluates it with `websearch_to_tsquery('simple', …)`), `&file=` (which configured log: `signaling.log` by default, `postgres.log` / `redis.log` when `POSTGRES_LOG_FILE` / `REDIS_LOG_FILE` are set, or **`all`** to search every configured source in one pass; an unknown name is `400`), `&component=` (whole-token match on the structured `component=` field, so `component=api` does not match `api_auth`), `&since=` / `&until=` (RFC3339, inclusive, compared against the line's own UTC timestamp), `&limit=` (default 200, clamped to 2000), `&scope=` (`local`, the default, reads this instance's files; **`all`** reads the lines every instance shipped into the shared store, and needs `INSTANCE_LOG_SINK=true` or it is a `400` naming the setting; with the sink on, the PostgreSQL and Redis logs are shipped too when `POSTGRES_LOG_FILE` / `REDIS_LOG_FILE` are configured, so `&file=postgres.log` narrows the aggregate the same way it narrows a local scan), `&instance=` (with `scope=all`, narrow to one instance id). Local returns `{items: [{timestamp, line, file}], count, limit, scope, file, match, files, scanned_bytes, scanned_lines, truncated}`; `scope=all` returns `{items: [{instance, source, timestamp, line}], count, limit, scope, instance, match, instances, truncated, stored_lines, dropped_lines}` (`match` echoes the full-text expression that was applied). Audited as `log.search` (`log_file` resource locally, `log_store:aggregated` for the merged view). Super-admin only |
| `GET` | `/v1/admin/system/logs/:name` — download one allow-listed file; audited as `log.download`. Super-admin only |
| `GET` | `/v1/admin/connections` |
| `GET` | `/v1/admin/activity` |
| `GET` | `/v1/admin/trends?range=24h\|7d\|30d` |
| `GET` | `/v1/admin/audit-logs` (filters `action` / `admin` / `username` / `actor_user_id` / `date_from` / `date_to`; `username` is a case-insensitive substring that matches either an administrator name or an end-user account in one pass, so the caller does not need to know which table the account lives in; a non-numeric `actor_user_id` is `400` rather than a filter that quietly matches everything) |
| `POST` | `/v1/admin/audit-logs:sweep` — run the audit-retention purge now. Optional body `{older_than_days: N}` overrides the configured age window, `{max_rows: N}` the table-wide row cap, `{max_rows_per_actor: N}` the per-actor row cap, `{max_rows_per_device: N}` the per-device row cap, `{max_rows_per_action: N}` the per-action row cap and `{archive_retention_days: N}` how long the daily archive files are kept; any combination may be passed. Returns `{purged, older_than_days, cutoff, max_rows, cap_removed, max_rows_per_actor, actor_cap_removed, max_rows_per_device, device_cap_removed, max_rows_per_action, action_cap_removed, archive_retention_days, archive_removed, archive_moved, archive_deleted, archive_dir, archive_cold_dir}` and audits `audit.sweep`. `400` when none of the config or body supplies any bound (`AUDIT_RETENTION_DAYS` / `AUDIT_MAX_ROWS` / `AUDIT_MAX_ROWS_PER_ACTOR` / `AUDIT_MAX_ROWS_PER_DEVICE` / `AUDIT_MAX_ROWS_PER_ACTION` / `AUDIT_ARCHIVE_RETENTION_DAYS` or the matching body fields), and `400` when archive retention is asked for while `AUDIT_ARCHIVE_DIR` is unset; when `AUDIT_ARCHIVE_DIR` is set an archive write failure deletes nothing |
| `GET` | `/v1/admin/audit-archive:search` — read back a day's archive that retention wrote. `?date=` (`YYYY-MM-DD`, default today in UTC), `&q=` (case-insensitive substring over action/resource/details/ip/admin), `&action=` (exact), `&admin=` (substring), `&actor_user_id=` (positive integer), `&limit=` (default 50, clamped to 500). Returns `{items, count, date, file, scanned_bytes, scanned_lines, truncated, has_more, archive_dir}` newest first and audits `audit.archive.search`. `400` when `AUDIT_ARCHIVE_DIR` is unset or `date`/`limit`/`actor_user_id` is malformed; `404` when that day has no archive file |
| `GET` `DELETE` | `/v1/admin/recordings[/:id]` (list filters `user_id` / `device_id` / `status` / `q`; delete writes audit action `recording.delete`) |
| `GET` | `/v1/admin/recordings/:id/artifact` (cross-user download; audit action `recording.download`) |
| `GET` | `/v1/admin/recordings/:id/thumbnail` (cross-user preview; audit action `recording.thumbnail.download`) |
| `GET` | `/v1/admin/recordings:storage` — `{artifact_count, total_bytes, retention_days, stale_after_minutes}` |
| `POST` | `/v1/admin/recordings:sweep` — runs the stale + retention + orphan pass now; returns `{stale_marked_failed, artifacts_purged, bytes_freed, orphans_removed, uploads_reaped}` (`uploads_reaped` counts abandoned resumable-upload staging files older than six hours) and audits `recording.sweep` |
| `GET` | `/v1/admin/notification-deliveries:stats` — `{total, by_status, acked, unacked, retention_days, archive_retention_days, archive_dir}`; `by_status` always carries all four states (`pending`/`delivering`/`delivered`/`failed`), zeroes included; `acked` counts rows with `acked_at`, `unacked` counts rows that were delivered but never confirmed |
| `POST` | `/v1/admin/notification-deliveries:sweep` — runs the outbox retention pass now. Optional body `{older_than_days: N}` overrides the configured window and `{archive_retention_days: N}` overrides how long the daily archive files are kept; either or both may be passed. Returns `{purged, older_than_days, cutoff, archive_retention_days, archive_removed, archive_moved, archive_deleted, archive_dir, archive_cold_dir}` and audits `notification_delivery.sweep`. `400` when none of the config or body supplies a bound (`NOTIFICATION_RETENTION_DAYS` / `NOTIFICATION_ARCHIVE_RETENTION_DAYS` or the matching body fields) and `400` when archive retention is asked for with `NOTIFICATION_ARCHIVE_DIR` unset; on an archive write failure nothing is deleted and the response is an error |
| `GET` `PUT` | `/v1/admin/preset` |
| `GET` `PUT` | `/v1/admin/settings` |
| `GET` `POST` `PATCH` `DELETE` | `/v1/admin/webhooks[/:id]` |
| `POST` | `/v1/admin/webhooks/:id/test` |

**Audit rows carry two actor columns.** `admin_id` / `admin_username` identify
an administrator; `actor_user_id` identifies an end user who acted on their own
resource. Exactly one of them is set. They are separate columns rather than one
"actor" on purpose: the two live in different tables, and overloading
`admin_id` with a user id would make "which administrator did this?"
unanswerable. The action name is shared where the same fact can be caused by
either party — a recording deletion is `recording.delete` whether the owner or
an admin did it, and the actor columns are what tell them apart, so one filter
shows every deletion.

Both actor columns are filterable, so the two questions an incident asks are
both answerable: `?admin=` narrows to an administrator and `?actor_user_id=`
narrows to an end user (`recording.delete` rows written by owners carry
`actor_user_id`, so filtering on it is how "what did this user do" is asked).
User-side rows also record the actor's observed client IP, the same column the
admin path fills, so the trail answers "from where" regardless of who acted.
The CSV export (`/v1/admin/audit-logs:export?format=csv`) carries
`actor_user_id` as its own column; without it a mixed export could not tell an
owner's deletion from an administrator's. The export accepts the same filters
as the list, `username` included, so the narrowed view can be taken away rather
than only looked at.

A third filter, `?username=`, covers the case where the operator knows the
account name but neither its id nor which side of the trail it sits on: it
matches `admin_username` and resolves an end-user name through the `users`
table in one query, so "what did wei do" returns the administrator's rows and
the end user's rows together. Resolving the user side through a join rather
than copying a name onto every row is deliberate - a renamed account's history
follows the account, and the write path pays nothing. Like `?admin=`, it is a
case-insensitive substring.

**Audit retention is opt-in and off by default.** `audit_logs` has no terminal
state, so without retention it only ever grows. `AUDIT_RETENTION_DAYS` (default
`0` = keep everything) deletes rows older than the window on a timer
(`AUDIT_SWEEP_INTERVAL_MINUTES`, default `60`), and
`POST /v1/admin/audit-logs:sweep` runs the same pass on demand - with an
optional `{"older_than_days": N}` so a window can be exercised without changing
configuration. Set `AUDIT_ARCHIVE_DIR` and expired rows are appended to a daily
JSONL file, fsynced, and only then deleted; if that write fails nothing is
removed and the endpoint reports the error, because an audit trail is evidence
and retention must not be the step that destroys the only copy. A sweep with no
configured retention and no `older_than_days` is a `400` rather than a silent
no-op or a "keep zero days".

**The age window is not the only bound: `AUDIT_MAX_ROWS` caps the table.**
Retention answers "how long do we keep rows"; it cannot answer "how big is this
table allowed to get", and a busy deployment can cross any row count long before
the window matters. `AUDIT_MAX_ROWS` (default `0` = no cap) is a hard row cap
independent of age: each sweep trims the oldest rows until at most that many
remain, archiving them first when `AUDIT_ARCHIVE_DIR` is set. Either bound alone
is enough to start the sweep, so "keep 30 days **or** 100k rows, whichever bites
first" is just two settings; `{"max_rows": N}` on
`POST /v1/admin/audit-logs:sweep` exercises the cap without changing config. The
cap is enforced *at sweep time*: rows written afterwards - including the sweep's
own `audit.sweep` row - can push the table one or a few rows over until the next
tick, so it bounds growth rather than freezing an exact count. There is
deliberately no byte-sized cap: PostgreSQL does not return freed space until
`VACUUM`, so a byte threshold would either over-delete or never converge, while
a row count is exact.

**A table-wide cap is not enough: `AUDIT_MAX_ROWS_PER_ACTOR` charges the noisy
actor.** The table cap has no idea who wrote a row, so a single actor - a user
whose failing token refreshes log in a loop, an admin running a bulk migration -
can fill the table and evict *everybody else's* trail. `AUDIT_MAX_ROWS_PER_ACTOR`
(default `0` = no cap) caps each actor separately: an actor holding more than the
limit loses their own oldest rows, and other actors' rows are untouched. Admins
and end users are counted apart, so admin 7 and user 7 never spend each other's
budget, and rows with no actor at all (the server's own entries) are exempt -
there is nobody to charge them to, and `AUDIT_MAX_ROWS` still bounds them. The
per-actor cap runs *before* the table-wide one on the same sweep, so the targeted
rule absorbs the overflow before the blunt rule starts deleting someone else's
history; `{"max_rows_per_actor": N}` on the sweep endpoint exercises it without
changing config. One pass inspects at most 20 actors and removes at most 1000
rows, so the sweep loops instead of holding one long statement, and when
`AUDIT_ARCHIVE_DIR` is set the removed rows are archived and fsynced first,
exactly like the other retention bounds.

**The same targeted logic covers a hot device and a hot action.**
`AUDIT_MAX_ROWS_PER_DEVICE` (default `0` = no cap) caps each device's trail. A
device row is identified by `resource_id` on `resource_type=device` (explicit
admin device actions) or `resource_type=event` (device-scoped events; the audit
subscriber uses the device id as the default resource). A batch action with an
empty `resource_id` names no single device and is therefore left to the action
cap, and recording/user/resource ids are not mistaken for device ids.
`AUDIT_MAX_ROWS_PER_ACTION` (default `0` = no cap) caps one action name, which
is the bound that stops a failing login, a noisy export or another single action
type from filling the table while every actor looks quiet. Empty action names
are exempt because there is no action to charge. Both caps run before the
table-wide cap, archive-and-fsync their removed rows when `AUDIT_ARCHIVE_DIR` is
set, and inspect at most 20 scopes / remove at most 1000 rows per pass so a
large cleanup drains over repeated passes. `{"max_rows_per_device": N}` and
`{"max_rows_per_action": N}` on the sweep endpoint exercise them without
changing configuration.

**The archive directory is bounded too, and off by default.**
Archiving keeps a purged row recoverable, but the daily files themselves only
ever grew. `AUDIT_ARCHIVE_RETENTION_DAYS` (default `0` = keep them forever)
deletes archive files older than the window on the same timer, and
`{"archive_retention_days": N}` on the sweep endpoint does it on demand;
`archive_removed` reports how many files left the hot directory. Set
`AUDIT_ARCHIVE_COLD_DIR` to **move** expired files there instead of deleting
them; `archive_moved` and `archive_deleted` report the split. A move that cannot
create the cold directory stops the sweep and leaves the hot file in place.
Only files named `audit-logs-YYYY-MM-DD.jsonl` are candidates - anything else in
the directory (a README, a hand-rotated copy) is left alone, because the
directory belongs to the operator as much as to this server. Asking for archive
retention without an `AUDIT_ARCHIVE_DIR` is a `400` rather than a silent no-op.
The default is "keep forever" on purpose: archiving exists so a purged row
stays recoverable, so deleting the archives has to be asked for explicitly.

**The archive is searchable, not write-only.**
`GET /v1/admin/audit-archive:search` reads a day's JSONL archive back with the
same bounded reverse scan the log search uses: `date` picks the file (default
today in UTC), `q` / `action` / `admin` / `actor_user_id` filter it, and the
response is newest-first with `scanned_bytes`, `scanned_lines`, `truncated`
(the scan budget ran out) and `has_more` (the limit was reached). Retention
deletes the live rows as it archives them, so without this endpoint "what did
administrator X do last month" became an SSH-and-grep job the moment the window
closed. The endpoint refuses to run when no `AUDIT_ARCHIVE_DIR` is configured
(`400`): an empty result must never mean "retention deleted it and you never
turned the archive on". A missing day is a `404`, distinct from a day that
exists but holds no matching rows. The path is `audit-archive:search` rather
than `audit-logs:archive` because gin keys routes by method and cannot hold two
different wildcard names (`:export` and `:archive`) under the same `GET` prefix.

**`logs:search` reads the log backwards.** The application log is a rotating
file (`signaling.log`, `signaling.log.1`, …), so the newest line is at the end
of a file that may be tens of megabytes; the endpoint scans from the end in
64 KiB blocks and stops after an 8 MiB budget, which makes "what happened
around 3pm" a query instead of a download. Two consequences are worth knowing:
the token-boundary `component` match exists so `component=api` does not pull in
`component=api_auth`; and `truncated: true` means the scan ran out of budget
before reaching the oldest file, so an empty result then is *not* proof the
event never happened. Paging backwards is done with `until` — pass the oldest
timestamp from one page to get the previous one — rather than a second scan
direction.

The same scan serves the PostgreSQL and Redis logs once `POSTGRES_LOG_FILE` /
`REDIS_LOG_FILE` are set: pick one with `file=`, and the response's `file` field
echoes the selection while `files` lists the rotated siblings that were walked.
An unknown `file=` is a `400` rather than a silent fallback to the application
log, because "no results" and "you searched a different file than you thought"
are very different answers during an incident. The `log.search` audit row
records the file name as its resource id, so "who read the database log" is
answerable afterwards.

**`file=all` searches every source at once, with one budget.** An incident
usually spans more than one log, and making the operator guess whether to look
at the application log or the database log first is work the server can do
instead. `file=all` walks each configured source newest-file-first, in
registration order, and every returned item carries a `file` field naming the
source it came from. The 8 MiB budget is **shared** across the sources rather
than granted per file, so "search everything" is still one bounded read -
`truncated: true` means that shared budget ran out, and the entries already
found are still returned.

**`scope=all` searches every instance, not just the one that answered.** A
multi-instance deployment behind a load balancer answers `logs:search` on
whichever instance took the request, so `scope=local` (the default, and the
behaviour every earlier version had) can only ever see that machine's files -
"the error is in the other instance's log" was unfixable from the admin UI. With
`INSTANCE_LOG_SINK=true`, each instance copies its own log lines into the shared
database and `scope=all` reads that table instead:

- every entry carries `instance` (the id the server logs at startup and registers
  with presence) and `source`, so a merged view is still attributable, and
  `instances` lists everyone who has shipped at least one line - which is what
  makes an empty result interpretable;
- `instance=<id>` narrows to one instance, and `component=` keeps its whole-token
  rule, so the aggregate answers the same questions as the file scan;
- **two search modes, on purpose.** `q=` is the case-insensitive substring
  search it always was - the right tool for a request id or a fragment - and it
  is now index-backed by a `pg_trgm` GIN index on `line`, which is what makes
  `ILIKE '%…%'` usable on a table that grows with the fleet's log volume.
  `match=` is PostgreSQL full text via `websearch_to_tsquery('simple', …)`:
  words are ANDed, `"quoted text"` is a phrase, `OR` works and a leading `-`
  negates, so `match="connection refused"` does not return a line that merely
  contains both words apart. The `simple` configuration is deliberate - log
  lines are identifiers and stack frames, and language stemming would rewrite
  `connection` into `connect` and match lines nobody asked for. `match` works on
  **both** scopes: `scope=all` evaluates the expression in PostgreSQL, and
  `scope=local` compiles the same expression in Go over the same simple lexer
  (`internal/service/log_match.go`), so "where do I search" no longer decides
  "how do I search". Both responses echo the expression back in `match`;
- the full-text column is a **generated** `tsvector` with its own GIN index.
  Neither it nor the trigram index is created by GORM's AutoMigrate - it manages
  columns, not expression indexes or generated columns - so the server ensures
  both at startup and logs the outcome. If a managed Postgres refuses
  `pg_trgm`, the queries stay correct and simply fall back to a sequential scan,
  which is logged as a warning at boot rather than silently accepted;
- shipping is **best effort by design**. The authoritative log is still the
  rotating file on the instance that wrote the line, so shipping must never
  block or fail a caller: lines queue in a 4096-line buffer, and if that fills -
  or the database refuses a batch - the copy is dropped and counted. The
  response reports this instance's `stored_lines` and `dropped_lines`, so "the
  line is missing" can be told apart from "shipping dropped it under load";
- a re-sent batch is ignored rather than duplicated: every shipped line carries
  a per-instance sequence number and `(instance_id, source, seq)` is unique;
- lines logged **before** the database is open are not shipped - the startup
  banner included. Shipping begins when the sink attaches, and the local file
  remains the record of everything before that;
- the PostgreSQL and Redis logs are **tailed**, not written through: each
  configured file gets its own reader that starts at the end of the file, ships
  only whole lines (a fragment without its newline waits for the next poll), and
  identifies a line by `(source, generation, offset)` rather than by the offset
  alone. The generation increments when the file is rotated (the inode behind the
  path changed) or truncated (the size went backwards), because offsets restart
  after either and a post-rotation line at the same offset as a pre-rotation one
  would otherwise be dropped as a duplicate. Rotation detection by inode is
  Linux/macOS only; on Windows only the size-shrink case is visible, so a
  rename-and-recreate that immediately grows past the old offset is not
  detected there;
- `INSTANCE_LOG_RETENTION_DAYS` (default `7`) bounds the shared table and
  `INSTANCE_LOG_SWEEP_INTERVAL_MINUTES` (default `30`) sets how often it is
  swept. Shipped lines are deleted without archiving - unlike the audit trail,
  the row is a copy of a line that is still in the file on the instance that
  wrote it. With the sink off, `scope=all` is a `400` rather than an empty
  result, because "nobody is shipping" must not read as "nothing happened".

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
| `device.offline.sustained` | the device stayed offline past its owner-set threshold (§5.3.6), published once per outage | `{device_id, device_name, offline_since, offline_minutes, threshold_minutes}` |
| `device.session.updated` | `logged_in_intent` flipped (bind / clear-session) | `{device_id, online, logged_in}` |
| `device.access_code.changed` | host reported a new code | `{device_id, changed: true}` (no plaintext) |
| `device.bound` / `device.added` | new device for this user | `{device_id}` (clients fetch detail once if not present) |
| `device.ownership.lost` | another account took over this device | `{device_id}` |
| `device.unbound` | unbound by user or admin | `{device_id}` |
| `device.remark.changed` / `device.display_name.changed` | `PATCH /v1/me/devices/:id` | `{device_id, remark?, device_name?}` |
| `favorite.added` / `favorite.updated` / `favorite.removed` | favorites mutated | `{device_id, ...}` |
| `recording.created` | `POST /v1/me/recordings` | `{recording_id, device_id, session_id, status, format}` |
| `recording.updated` | `PATCH /v1/me/recordings/:id` | `{recording_id, device_id, session_id, status}` |
| `recording.deleted` | `DELETE /v1/me/recordings/:id` | `{recording_id}` |
| `session.revoked` | this user's session was killed | `{}` |
| `device.secret.rotated` | admin rotated `device_secret` (sent to host owner) | `{device_id}` (host should re-provision on next 401) |
| `device.access.granted` / `device.access.denied` / `device.access.session_started` | an access attempt or session start on this user's device (§5.3.1) | `{device_id, client_id, ip, result, reason?, log_id}` |
| `device.access_policy.changed` | owner changed the device access policy (§5.3.3) | `{device_id, remote_disabled, allow_ips, max_sessions, allowed_windows, timezone, exception_dates}` |
| `device.access_grant.changed` | owner created or revoked a guest grant (§5.3.4) | `{action: "created"\|"revoked", grant_id, label, expires_at, max_uses, uses}` (never the code) |
| `device.access_grant.used` | a grant just admitted a session | `{grant_id, label, uses, max_uses, client_id}` |
| `device.access_approval.requested` | a correct code is waiting for the owner's yes/no (§5.3.5) | `{approval_id, client_id, client_ip, source, trigger, expires_at}`, where `trigger` is `always` / `grant` / `untrusted_ip` (never the secret) |
| `device.access_approval.decided` | owner approved or denied a waiting session | `{approval_id, status: "approved"\|"denied", source, client_id, reason?}` |
| `remote_session.started` | a controller's signaling socket authenticated against this user's device (§5.3.8) | `{session_id, device_id, client_id, client_ip, started_at, active, duration_seconds}` |
| `remote_session.ended` | that signaling socket went away | `{session_id, device_id, client_id, client_ip, started_at, ended_at, end_reason, active, duration_seconds}` |

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
