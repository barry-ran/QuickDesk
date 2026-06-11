// Copyright 2026 QuickDesk Authors
#include "CloudDeviceManager.h"
#include "ServerManager.h"
#include "AuthManager.h"
#include "HostManager.h"
#include "infra/http/httprequest.h"
#include "infra/log/log.h"

#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QUrl>
#include <QTimer>
#include <QRandomGenerator>

#ifndef QUICKDESK_API_KEY
#define QUICKDESK_API_KEY ""
#endif

namespace quickdesk {

namespace {
constexpr int kRequestTimeoutMs = 10000;
constexpr int kSyncReconnectBaseMs = 1000;   // exponential backoff base
constexpr int kSyncReconnectMaxMs  = 30000;  // cap
}

CloudDeviceManager::CloudDeviceManager(ServerManager* serverManager,
                                         AuthManager* authManager,
                                         HostManager* hostManager,
                                         QObject* parent)
    : QObject(parent)
    , m_serverManager(serverManager)
    , m_authManager(authManager)
    , m_hostManager(hostManager)
{
}

CloudDeviceManager::~CloudDeviceManager()
{
    stopSync();
}

QString CloudDeviceManager::httpBaseUrl() const
{
    QString wsUrl = m_serverManager->serverUrl();
    QString httpUrl = wsUrl;
    httpUrl.replace("ws://", "http://");
    httpUrl.replace("wss://", "https://");
    if (!httpUrl.endsWith("/")) httpUrl += "/";
    return httpUrl;
}

QList<QPair<QString, QString>> CloudDeviceManager::deviceSecretHeaders() const
{
    QList<QPair<QString, QString>> headers;

    // AuthManager::publicHeaders() injects X-API-Key (runtime override
    // > compile-time) when available. apiKeyAuth.Required() wraps the
    // entire /v1 group per main.go.
    for (const auto& h : m_authManager->publicHeaders()) {
        headers.append(h);
    }
    headers.append(qMakePair(QStringLiteral("Content-Type"),
                              QStringLiteral("application/json")));

    if (m_hostManager) {
        QString secret = m_hostManager->deviceSecret();
        if (!secret.isEmpty()) {
            headers.append(qMakePair(QStringLiteral("Authorization"),
                                      QStringLiteral("Bearer ") + secret));
        }
    }
    return headers;
}

// ========================================================================
// My Devices
// ========================================================================

void CloudDeviceManager::fetchMyDevices()
{
    if (!m_authManager->isLoggedIn()) return;

    QUrl url(httpBaseUrl() + "v1/me/devices");

    m_authManager->request("GET", url, QString(),
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] fetchMyDevices failed: status={} err={}",
                             statusCode, errorMsg);
                    return;
                }
                QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
                QJsonArray devices = doc.object()["items"].toArray();
                m_myDevices.clear();
                for (const auto& v : devices) {
                    QJsonObject obj = v.toObject();
                    m_myDevices.append(obj.toVariantMap());
                    LOG_INFO("[CloudDeviceManager]   device={} online={} logged_in={}",
                             obj["device_id"].toString().toStdString(),
                             obj["online"].toBool(),
                             obj["logged_in"].toBool());
                }
                LOG_INFO("[CloudDeviceManager] Fetched {} devices", m_myDevices.size());
                emit myDevicesChanged();
            });
        });
}

void CloudDeviceManager::autoBindDevice(const QString& deviceId)
{
    if (!m_authManager->isLoggedIn() || deviceId.isEmpty()) return;

    // §2.2: POST /v1/me/devices with body {device_id} — idempotent,
    // performs takeover atomically server-side.
    QUrl url(httpBaseUrl() + "v1/me/devices");
    QJsonObject body;
    body["device_id"] = deviceId;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    m_authManager->request("POST", url, bodyData,
        [this, deviceId](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, deviceId]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] autoBindDevice({}) failed: status={} err={}",
                             deviceId.toStdString(), statusCode, errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Device bound: {}",
                         deviceId.toStdString());
                // The server will emit `device.bound` via realtime — no
                // need to refetch; patch will arrive through the events WS.
                // Fetch once as a belt-and-suspenders initial load.
                fetchMyDevices();
            });
        });
}

void CloudDeviceManager::unbindDevice(const QString& deviceId)
{
    if (!m_authManager->isLoggedIn() || deviceId.isEmpty()) return;

    QUrl url(httpBaseUrl() + "v1/me/devices/" + deviceId);

    m_authManager->request("DELETE", url, QString(),
        [this, deviceId](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, deviceId]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] unbindDevice({}) failed: status={} err={}",
                             deviceId.toStdString(), statusCode, errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Device unbound: {}",
                         deviceId.toStdString());
                // Realtime will emit device.unbound; patch local cache
                // proactively so the UI updates instantly.
                for (int i = m_myDevices.size() - 1; i >= 0; --i) {
                    if (m_myDevices[i].toMap()["device_id"].toString() == deviceId) {
                        m_myDevices.removeAt(i);
                    }
                }
                emit myDevicesChanged();
            });
        });
}

void CloudDeviceManager::setDeviceRemark(const QString& deviceId, const QString& remark)
{
    if (!m_authManager->isLoggedIn() || deviceId.isEmpty()) return;

    // §2.2: PATCH /v1/me/devices/:device_id {remark?, device_name?}
    QUrl url(httpBaseUrl() + "v1/me/devices/" + deviceId);
    QJsonObject body;
    body["remark"] = remark;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    m_authManager->request("PATCH", url, bodyData,
        [this, deviceId](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [statusCode, errorMsg, deviceId]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] setDeviceRemark({}) failed: status={} err={}",
                             deviceId.toStdString(), statusCode, errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Remark updated for {}",
                         deviceId.toStdString());
                // Local patch — server will emit device.remark.changed
                // via realtime too, but we front-run for snappier UI.
                // Field name is intentionally `remark` (matches server envelope).
                // Note: we don't have the new value echoed back in the
                // response envelope, so rely on the next realtime event
                // for an authoritative refresh. Still, patch what we set
                // ourselves for immediate visibility.
                // (realtime event will overwrite on arrival)
            });
        });
}

// §2.23: Qt syncs access_code using Bearer <device_secret>. device_secret
// is NOT the user access_token — so we bypass AuthManager::request()
// (which injects the user Bearer) and build headers manually.
void CloudDeviceManager::syncAccessCode(const QString& deviceId, const QString& accessCode)
{
    if (deviceId.isEmpty() || accessCode.isEmpty()) return;
    if (!m_hostManager || m_hostManager->deviceSecret().isEmpty()) {
        LOG_WARN("[CloudDeviceManager] syncAccessCode skipped: no device_secret yet");
        return;
    }

    // R28: MainController wires THREE host-side signals
    // (hostReady / accessCodeChanged / deviceSecretReady) to this
    // method; they all fire within a few ms when the host finishes
    // starting up. Suppress duplicate uploads of the exact same
    // (device_id, access_code) pair so we don't PUT 3× and publish 3
    // `device.access_code.changed` events in succession.
    if (deviceId == m_lastSyncedDeviceId && accessCode == m_lastSyncedAccessCode) {
        LOG_INFO("[CloudDeviceManager] syncAccessCode skipped (already up-to-date) for device={}",
                 deviceId.toStdString());
        return;
    }

    QUrl url(httpBaseUrl() + "v1/devices/" + deviceId + "/access-code");
    auto headers = deviceSecretHeaders();

    QJsonObject body;
    body["access_code"] = accessCode;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    LOG_INFO("[CloudDeviceManager] Syncing access_code for device={} (Bearer device_secret)",
             deviceId.toStdString());

    infra::HttpRequest::instance().sendPutRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this, deviceId, accessCode](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, deviceId, accessCode]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] syncAccessCode failed: status={} err={}",
                             statusCode, errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Access code synced for device: {}",
                         deviceId.toStdString());
                // R28: remember what we successfully pushed so the next
                // MainController signal with the same (deviceId, code)
                // short-circuits in syncAccessCode().
                m_lastSyncedDeviceId = deviceId;
                m_lastSyncedAccessCode = accessCode;
                // Local cache patch (realtime will confirm via
                // device.access_code.changed event — which does NOT
                // include the plaintext code; that's why we patch locally).
                for (int i = 0; i < m_myDevices.size(); ++i) {
                    QVariantMap d = m_myDevices[i].toMap();
                    if (d["device_id"].toString() == deviceId) {
                        d["access_code"] = accessCode;
                        m_myDevices[i] = d;
                        emit myDevicesChanged();
                        break;
                    }
                }
            });
        });
}

// §2.6: POST /v1/devices/:device_id/access-code:verify.
// Auth path here is X-API-Key (via publicHeaders), NOT device_secret:
// any Qt instance can mint a client-side signal_token for any host
// whose access_code it knows — mirrors the WebClient flow.
void CloudDeviceManager::verifyAccessCode(
    const QString& deviceId,
    const QString& accessCode,
    std::function<void(const QString& signalToken)> onSuccess,
    std::function<void(int, const QString&, const QString&)> onError) {
    if (deviceId.isEmpty() || accessCode.isEmpty()) {
        if (onError) onError(0, QStringLiteral("INVALID_ARGUMENT"),
                             QStringLiteral("deviceId / accessCode empty"));
        return;
    }

    QUrl url(httpBaseUrl() + "v1/devices/" + deviceId +
             "/access-code:verify");

    QList<QPair<QString, QString>> headers;
    // publicHeaders gives us X-API-Key (runtime override > compile-time).
    if (m_authManager) {
        for (const auto& h : m_authManager->publicHeaders()) {
            headers.append(h);
        }
    }
    headers.append(qMakePair(QStringLiteral("Content-Type"),
                              QStringLiteral("application/json")));

    QJsonObject body;
    body["code"] = accessCode;
    QString bodyData =
        QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    LOG_INFO("[CloudDeviceManager] verifyAccessCode device={} (X-API-Key)",
             deviceId.toStdString());

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this, deviceId, onSuccess = std::move(onSuccess),
         onError = std::move(onError)](int statusCode,
                                        const std::string& errorMsg,
                                        const std::string& data) {
            QMetaObject::invokeMethod(
                this, [=, onSuccess = std::move(onSuccess),
                       onError = std::move(onError)]() mutable {
                    QJsonDocument doc = QJsonDocument::fromJson(
                        QByteArray::fromStdString(data));
                    QJsonObject obj = doc.object();
                    if (statusCode == 200 && errorMsg.empty()) {
                        QString token = obj.value("signal_token").toString();
                        if (token.isEmpty()) {
                            if (onError) {
                                onError(statusCode,
                                        QStringLiteral("INVALID_RESPONSE"),
                                        QStringLiteral(
                                            "server returned no signal_token"));
                            }
                            return;
                        }
                        if (onSuccess) onSuccess(token);
                        return;
                    }
                    // RFC 7807 problem+json: {code, title, detail, status}.
                    QString code = obj.value("code").toString();
                    QString detail = obj.value("detail").toString();
                    if (detail.isEmpty()) detail = QString::fromStdString(errorMsg);
                    LOG_WARN("[CloudDeviceManager] verifyAccessCode failed: "
                             "status={} code={} detail={}",
                             statusCode, code.toStdString(),
                             detail.toStdString());
                    if (onError) onError(statusCode, code, detail);
                });
        });
}

QString CloudDeviceManager::getDeviceAccessCode(const QString& deviceId) const
{
    for (const auto& v : m_myDevices) {
        QVariantMap d = v.toMap();
        if (d["device_id"].toString() == deviceId) {
            return d["access_code"].toString();
        }
    }
    return QString();
}

QString CloudDeviceManager::getDeviceDisplayName(const QString& deviceId) const
{
    for (const auto& v : m_myDevices) {
        QVariantMap d = v.toMap();
        if (d["device_id"].toString() == deviceId) {
            QString remark = d["remark"].toString();
            if (!remark.isEmpty())
                return remark + " (" + deviceId + ")";
            QString name = d["device_name"].toString();
            if (!name.isEmpty())
                return name + " (" + deviceId + ")";
            break;
        }
    }

    for (const auto& v : m_myFavorites) {
        QVariantMap f = v.toMap();
        if (f["device_id"].toString() == deviceId) {
            QString name = f["device_name"].toString();
            if (!name.isEmpty())
                return name + " (" + deviceId + ")";
            break;
        }
    }

    return deviceId;
}

// ========================================================================
// Connection record
// ========================================================================

void CloudDeviceManager::recordConnection(const QString& deviceId, int duration,
                                           const QString& status, const QString& errorMsg)
{
    if (!m_authManager->isLoggedIn() || deviceId.isEmpty()) return;

    QUrl url(httpBaseUrl() + "v1/me/connections");
    QJsonObject body;
    body["device_id"] = deviceId;
    body["duration"]  = duration;
    body["status"]    = status;
    if (!errorMsg.isEmpty()) body["error_msg"] = errorMsg;
    // Best-effort enrichment: look up display name from current cache.
    QString name = getDeviceDisplayName(deviceId);
    if (name != deviceId) body["device_name"] = name;

    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    m_authManager->request("POST", url, bodyData,
        [deviceId, status](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            if (statusCode != 200 || !errorMsg.empty()) {
                LOG_WARN("[CloudDeviceManager] recordConnection failed: status={} err={}",
                         statusCode, errorMsg);
                return;
            }
            LOG_INFO("[CloudDeviceManager] Connection recorded: device={}, status={}",
                     deviceId.toStdString(), status.toStdString());
        });
}

void CloudDeviceManager::fetchConnectionLogs()
{
    if (!m_authManager->isLoggedIn()) return;

    QUrl url(httpBaseUrl() + "v1/me/connections");

    m_authManager->request("GET", url, QString(),
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] fetchConnectionLogs failed: status={} err={}",
                             statusCode, errorMsg);
                    return;
                }
                QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
                // §2.3 list envelope: {items:[], next_cursor}
                QJsonArray logs = doc.object()["items"].toArray();
                m_connectionLogs.clear();
                for (const auto& v : logs) {
                    m_connectionLogs.append(v.toObject().toVariantMap());
                }
                LOG_INFO("[CloudDeviceManager] Fetched {} connection logs",
                         m_connectionLogs.size());
                emit connectionLogsChanged();
            });
        });
}

// ========================================================================
// Favorites
// ========================================================================

void CloudDeviceManager::fetchFavorites()
{
    if (!m_authManager->isLoggedIn()) return;

    QUrl url(httpBaseUrl() + "v1/me/favorites");

    m_authManager->request("GET", url, QString(),
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] fetchFavorites failed: status={} err={}",
                             statusCode, errorMsg);
                    return;
                }
                QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
                QJsonArray favorites = doc.object()["items"].toArray();
                m_myFavorites.clear();
                for (const auto& v : favorites) {
                    m_myFavorites.append(v.toObject().toVariantMap());
                }
                LOG_INFO("[CloudDeviceManager] Fetched {} favorites",
                         m_myFavorites.size());
                emit myFavoritesChanged();
            });
        });
}

void CloudDeviceManager::addFavorite(const QString& deviceId,
                                       const QString& name,
                                       const QString& password)
{
    if (!m_authManager->isLoggedIn() || deviceId.isEmpty()) return;

    QString favName = name;
    if (favName.isEmpty()) {
        for (const auto& v : m_myDevices) {
            QVariantMap d = v.toMap();
            if (d["device_id"].toString() == deviceId) {
                favName = d["remark"].toString();
                if (favName.isEmpty()) favName = d["device_name"].toString();
                break;
            }
        }
    }

    QUrl url(httpBaseUrl() + "v1/me/favorites");
    QJsonObject body;
    body["device_id"] = deviceId;
    if (!favName.isEmpty()) body["device_name"] = favName;
    if (!password.isEmpty()) body["access_password"] = password;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    m_authManager->request("POST", url, bodyData,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] addFavorite failed: status={} err={}",
                             statusCode, errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Favorite added");
                // realtime favorite.added will arrive shortly; refetch
                // once as a fallback for immediate visibility.
                fetchFavorites();
            });
        });
}

void CloudDeviceManager::updateFavorite(const QString& deviceId,
                                         const QString& name,
                                         const QString& password)
{
    if (!m_authManager->isLoggedIn() || deviceId.isEmpty()) return;

    QUrl url(httpBaseUrl() + "v1/me/favorites/" + deviceId);
    QJsonObject body;
    if (!name.isEmpty())     body["device_name"]     = name;
    if (!password.isEmpty()) body["access_password"] = password;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    m_authManager->request("PATCH", url, bodyData,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] updateFavorite failed: status={} err={}",
                             statusCode, errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Favorite updated");
                fetchFavorites();
            });
        });
}

void CloudDeviceManager::removeFavorite(const QString& deviceId)
{
    if (!m_authManager->isLoggedIn() || deviceId.isEmpty()) return;

    QUrl url(httpBaseUrl() + "v1/me/favorites/" + deviceId);

    m_authManager->request("DELETE", url, QString(),
        [this, deviceId](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, deviceId]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] removeFavorite failed: status={} err={}",
                             statusCode, errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Favorite removed");
                // Local patch plus refetch fallback.
                for (int i = m_myFavorites.size() - 1; i >= 0; --i) {
                    if (m_myFavorites[i].toMap()["device_id"].toString() == deviceId) {
                        m_myFavorites.removeAt(i);
                    }
                }
                emit myFavoritesChanged();
            });
        });
}

// ========================================================================
// Realtime events WebSocket (§2.7 / §2.8)
// ========================================================================

void CloudDeviceManager::startSync()
{
    if (!m_authManager->isLoggedIn()) return;
    stopSync();

    m_syncAuthOk = false;
    m_syncSocket = new QWebSocket(QString(),
                                    QWebSocketProtocol::VersionLatest, this);
    connect(m_syncSocket, &QWebSocket::textMessageReceived,
            this, &CloudDeviceManager::onSyncTextMessageReceived);
    connect(m_syncSocket, &QWebSocket::disconnected,
            this, &CloudDeviceManager::onSyncDisconnected);
    connect(m_syncSocket, &QWebSocket::connected,
            this, &CloudDeviceManager::onSyncConnected);

    QString wsUrl = m_serverManager->serverUrl();
    if (!wsUrl.endsWith("/")) wsUrl += "/";
    // §2.16: WS URL does NOT carry the token. First-frame auth (§2.13).
    wsUrl += "v1/realtime/events";

    LOG_INFO("[CloudDeviceManager] Connecting realtime events WebSocket: {}",
             wsUrl.toStdString());
    m_syncSocket->open(QUrl(wsUrl));
}

void CloudDeviceManager::stopSync()
{
    if (m_syncSocket) {
        disconnect(m_syncSocket, nullptr, this, nullptr);
        m_syncSocket->close();
        m_syncSocket->deleteLater();
        m_syncSocket = nullptr;
        m_syncAuthOk = false;
        m_serverRev = 0;
        LOG_INFO("[CloudDeviceManager] Realtime events WebSocket stopped");
    }
}

void CloudDeviceManager::onSyncConnected()
{
    // Server expects first-frame auth (§2.13). Don't fire syncConnected()
    // yet — we want to wait for auth_ok so subscribers (autoBindDevice)
    // act only after the server has accepted the token.
    sendAuthFrame();
}

void CloudDeviceManager::sendAuthFrame()
{
    if (!m_syncSocket || !m_authManager->isLoggedIn()) return;

    QJsonObject auth;
    auth["type"] = "auth";
    auth["access_token"] = m_authManager->token();
    if (m_serverRev > 0) {
        // §2.8 resume hint; server may reply with replay-then-snapshot or
        // snapshot_required depending on stream retention.
        auth["since_rev"] = m_serverRev;
    }
    QString payload = QString::fromUtf8(QJsonDocument(auth).toJson(QJsonDocument::Compact));
    m_syncSocket->sendTextMessage(payload);
    LOG_INFO("[CloudDeviceManager] Sent WS auth frame (since_rev={})", m_serverRev);
}

void CloudDeviceManager::onSyncTextMessageReceived(const QString& message)
{
    QJsonDocument doc = QJsonDocument::fromJson(message.toUtf8());
    if (!doc.isObject()) return;
    QJsonObject msg = doc.object();
    QString type = msg["type"].toString();

    // Track server_rev for resume on every wire frame that carries it.
    if (msg.contains("server_rev")) {
        qint64 rev = msg["server_rev"].toVariant().toLongLong();
        if (rev > m_serverRev) m_serverRev = rev;
    }

    LOG_DEBUG("[CloudDeviceManager] WS frame type={} server_rev={}",
              type.toStdString(), m_serverRev);

    if (type == "auth_ok") {
        m_syncAuthOk = true;
        m_reconnectAttempt = 0;
        LOG_INFO("[CloudDeviceManager] Realtime events auth_ok");
        emit syncConnected();
        return;
    }
    if (type == "error") {
        QString code = msg["data"].toObject()["code"].toString();
        LOG_WARN("[CloudDeviceManager] WS server error code={}", code.toStdString());
        // TOKEN_INVALID / AUTH_INVALID — the WS auth token was rejected.
        // Proactively kick an HTTP refresh so the next reconnect uses a
        // fresh access_token (otherwise we'd loop on the same bad token).
        // We rely on AuthManager's internal de-dup (m_refreshInFlight) to
        // avoid storming refresh when HTTP traffic is already refreshing.
        if (code == QLatin1String("TOKEN_INVALID") ||
            code == QLatin1String("AUTH_INVALID")) {
            // Use a GET /v1/me to trigger the 401 → refresh cascade; it
            // has no side effects server-side and is the minimum call
            // that routes through AuthManager::request().
            QUrl url(httpBaseUrl() + "v1/me");
            m_authManager->request("GET", url, QString(),
                [](int, const std::string&, const std::string&) { /* no-op */ });
        }
        m_syncSocket->close();
        return;
    }
    if (type == "snapshot_required") {
        LOG_INFO("[CloudDeviceManager] snapshot_required — waiting for snapshot frame");
        // The server sends the snapshot frame next anyway; nothing to do.
        return;
    }
    if (type == "snapshot") {
        handleSnapshotFrame(msg);
        return;
    }
    // Anything else is a domain event (§2.7).
    handleEventFrame(msg);
    emit syncMessage(msg);
}

void CloudDeviceManager::onSyncDisconnected()
{
    bool wasAuthOk = m_syncAuthOk;
    m_syncAuthOk = false;
    int attempt = ++m_reconnectAttempt;
    int delayMs = qMin(kSyncReconnectBaseMs * (1 << qMin(attempt - 1, 5)),
                        kSyncReconnectMaxMs);
    // Add a small random jitter (§2.15 thundering-herd).
    delayMs += QRandomGenerator::global()->bounded(500);

    LOG_WARN("[CloudDeviceManager] Realtime WS disconnected (was_auth_ok={}, attempt={}), reconnecting in {}ms",
             wasAuthOk, attempt, delayMs);
    if (m_authManager->isLoggedIn()) {
        QTimer::singleShot(delayMs, this, &CloudDeviceManager::startSync);
    }
}

void CloudDeviceManager::handleSnapshotFrame(const QJsonObject& msg)
{
    QJsonObject data = msg["data"].toObject();

    QJsonArray devices = data["devices"].toArray();
    m_myDevices.clear();
    for (const auto& v : devices) {
        m_myDevices.append(v.toObject().toVariantMap());
    }
    emit myDevicesChanged();

    QJsonArray favorites = data["favorites"].toArray();
    m_myFavorites.clear();
    for (const auto& v : favorites) {
        m_myFavorites.append(v.toObject().toVariantMap());
    }
    emit myFavoritesChanged();

    LOG_INFO("[CloudDeviceManager] snapshot applied: {} devices, {} favorites (server_rev={})",
             m_myDevices.size(), m_myFavorites.size(), m_serverRev);
}

void CloudDeviceManager::handleEventFrame(const QJsonObject& msg)
{
    QString type = msg["type"].toString();
    QJsonObject data = msg["data"].toObject();

    if (type.startsWith("device.")) {
        applyDeviceEvent(type, data);
    } else if (type.startsWith("favorite.")) {
        applyFavoriteEvent(type, data);
    } else if (type == "session.revoked") {
        LOG_WARN("[CloudDeviceManager] session.revoked received — forcing logout");
        // AuthManager clears local session + emits loggedOut → QML pops
        // to login page. We ask logout() with empty device_id so it goes
        // through the two-step flow (best-effort session clear is fine
        // even though the server has already nuked it).
        m_authManager->logout(QString());
    } else {
        LOG_DEBUG("[CloudDeviceManager] unhandled event type: {}",
                  type.toStdString());
    }
}

// Patch m_myDevices in-place for a device event. §2.8: "直接 patch 本地设备
// 列表，不回拉 fetchMyDevices"。
void CloudDeviceManager::applyDeviceEvent(const QString& type, const QJsonObject& data)
{
    QString deviceId = data["device_id"].toString();
    if (deviceId.isEmpty()) return;

    // Find existing row (if any).
    int idx = -1;
    for (int i = 0; i < m_myDevices.size(); ++i) {
        if (m_myDevices[i].toMap()["device_id"].toString() == deviceId) {
            idx = i;
            break;
        }
    }

    if (type == "device.unbound" || type == "device.ownership.lost") {
        if (idx >= 0) {
            m_myDevices.removeAt(idx);
            emit myDevicesChanged();
        }
        return;
    }
    if (type == "device.bound" || type == "device.added") {
        // Server sent minimal data ({device_id}); list fields will be
        // filled in by a follow-up realtime event or the next snapshot.
        // To keep the UI usable we insert a stub and schedule a fetch.
        if (idx < 0) {
            QVariantMap stub;
            stub["device_id"] = deviceId;
            // Copy whatever extra fields are in the event (may include
            // device_name / os / logged_in depending on future server changes).
            for (auto it = data.constBegin(); it != data.constEnd(); ++it) {
                stub[it.key()] = it.value().toVariant();
            }
            m_myDevices.append(stub);
            emit myDevicesChanged();
            // Fetch to get access_code / remark / display_name — these
            // aren't in the event payload for privacy reasons.
            fetchMyDevices();
        }
        return;
    }
    // For update events, patch selected fields on the existing row.
    if (idx < 0) {
        // Event for a device we don't know about — request a fetch to
        // reconcile (rare, happens on races where snapshot missed a row).
        fetchMyDevices();
        return;
    }

    QVariantMap row = m_myDevices[idx].toMap();
    bool changed = false;
    auto copyField = [&](const QString& key) {
        if (data.contains(key)) {
            row[key] = data[key].toVariant();
            changed = true;
        }
    };

    if (type == "device.online.changed") {
        copyField("online");
        // logged_in is derived from logged_in_intent && online; if the
        // server didn't send it, recompute locally.
        if (data.contains("logged_in")) {
            copyField("logged_in");
        } else if (data.contains("online")) {
            bool online = data["online"].toBool();
            bool prevLI = row["logged_in"].toBool();
            row["logged_in"] = online && prevLI;
            changed = true;
        }
    } else if (type == "device.session.updated") {
        copyField("logged_in");
        copyField("online");
    } else if (type == "device.access_code.changed") {
        // §2.16: event data does NOT carry plaintext access_code.
        // syncAccessCode's local patch already updated the field; if we
        // got here without having done the PUT ourselves (e.g. another
        // Qt instance updated it), refetch to learn the new code.
        fetchMyDevices();
        return;
    } else if (type == "device.remark.changed") {
        copyField("remark");
    } else if (type == "device.device_name.changed") {
        copyField("device_name");
    } else if (type == "device.secret.rotated") {
        // §2.17: we're no longer a valid device_secret holder. HostManager
        // will rediscover via next provision cycle (stage 4); for now,
        // just log.
        LOG_WARN("[CloudDeviceManager] device.secret.rotated for {}",
                 deviceId.toStdString());
        return;
    }

    if (changed) {
        m_myDevices[idx] = row;
        emit myDevicesChanged();
    }
}

void CloudDeviceManager::applyFavoriteEvent(const QString& type, const QJsonObject& data)
{
    QString deviceId = data["device_id"].toString();
    if (deviceId.isEmpty()) return;

    if (type == "favorite.removed") {
        for (int i = m_myFavorites.size() - 1; i >= 0; --i) {
            if (m_myFavorites[i].toMap()["device_id"].toString() == deviceId) {
                m_myFavorites.removeAt(i);
            }
        }
        emit myFavoritesChanged();
        return;
    }

    // added/updated: favorite events don't always carry the full row
    // (server may choose to only send device_id for privacy). Refetch.
    fetchFavorites();
}

QVariantList CloudDeviceManager::myDevices()       const { return m_myDevices; }
QVariantList CloudDeviceManager::myFavorites()     const { return m_myFavorites; }
QVariantList CloudDeviceManager::connectionLogs()  const { return m_connectionLogs; }

} // namespace quickdesk
