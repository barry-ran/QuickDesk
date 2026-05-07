// Copyright 2026 QuickDesk Authors
#include "CloudDeviceManager.h"
#include "ServerManager.h"
#include "AuthManager.h"
#include "infra/http/httprequest.h"
#include "infra/log/log.h"

#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QUrl>
#include <QTimer>

namespace quickdesk {

namespace {
constexpr int kRequestTimeoutMs = 10000;
constexpr int kSyncReconnectDelayMs = 5000;
}

CloudDeviceManager::CloudDeviceManager(ServerManager* serverManager, AuthManager* authManager, QObject* parent)
    : QObject(parent)
    , m_serverManager(serverManager)
    , m_authManager(authManager)
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

QList<QPair<QString, QString>> CloudDeviceManager::authHeaders() const
{
    QList<QPair<QString, QString>> headers;
    headers.append(qMakePair(QStringLiteral("Authorization"),
                              QStringLiteral("Bearer ") + m_authManager->token()));
    headers.append(qMakePair(QStringLiteral("Content-Type"), QStringLiteral("application/json")));
    return headers;
}

// ---- My Devices ----

void CloudDeviceManager::fetchMyDevices()
{
    if (!m_authManager->isLoggedIn()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/devices");
    auto headers = authHeaders();

    infra::HttpRequest::instance().sendGetRequest(
        url, headers, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] fetchMyDevices failed: {}", errorMsg);
                    return;
                }
                QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
                QJsonArray devices = doc.object()["devices"].toArray();
                m_myDevices.clear();
                for (const auto& v : devices) {
                    m_myDevices.append(v.toObject().toVariantMap());
                }
                LOG_INFO("[CloudDeviceManager] Fetched {} devices", m_myDevices.size());
                emit myDevicesChanged();
            });
        });
}

void CloudDeviceManager::autoBindDevice(const QString& deviceId)
{
    if (!m_authManager->isLoggedIn() || deviceId.isEmpty()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/devices/auto-bind");
    auto headers = authHeaders();
    QJsonObject body;
    body["device_id"] = deviceId;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] autoBindDevice failed: {}", errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Device auto-bound successfully");
                fetchMyDevices();
            });
        });
}

void CloudDeviceManager::unbindDevice(const QString& deviceId)
{
    if (!m_authManager->isLoggedIn()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/devices/unbind");
    auto headers = authHeaders();
    QJsonObject body;
    body["device_id"] = deviceId;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] unbindDevice failed: {}", errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Device unbound");
                fetchMyDevices();
            });
        });
}

void CloudDeviceManager::deviceLogin(const QString& deviceId)
{
    if (!m_authManager->isLoggedIn() || deviceId.isEmpty()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/devices/" + deviceId + "/login");
    auto headers = authHeaders();

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, QString(), kRequestTimeoutMs,
        [this, deviceId](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, deviceId]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] deviceLogin failed: {}", errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Device login marked: {}", deviceId.toStdString());
            });
        });
}

void CloudDeviceManager::deviceLogout(const QString& deviceId)
{
    if (deviceId.isEmpty()) return;
    // Allow calling even if not logged in (token may still be valid briefly)
    QString token = m_authManager->token();
    if (token.isEmpty()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/devices/" + deviceId + "/logout");
    auto headers = authHeaders();

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, QString(), kRequestTimeoutMs,
        [this, deviceId](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, deviceId]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] deviceLogout failed: {}", errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Device logout marked: {}", deviceId.toStdString());
            });
        });
}

void CloudDeviceManager::setDeviceRemark(const QString& deviceId, const QString& remark)
{
    if (!m_authManager->isLoggedIn()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/devices/" + deviceId + "/remark");
    auto headers = authHeaders();
    QJsonObject body;
    body["remark"] = remark;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    infra::HttpRequest::instance().sendPutRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] setDeviceRemark failed: {}", errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Remark updated");
                fetchMyDevices();
            });
        });
}

void CloudDeviceManager::syncAccessCode(const QString& deviceId, const QString& accessCode)
{
    if (!m_authManager->isLoggedIn()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/devices/" + deviceId + "/access-code");
    auto headers = authHeaders();
    QJsonObject body;
    body["access_code"] = accessCode;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    infra::HttpRequest::instance().sendPutRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this, deviceId, accessCode](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, deviceId, accessCode]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] syncAccessCode failed: {}", errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Access code synced for device: {}", deviceId.toStdString());
                // Update local cache
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

// ---- Connection Record ----

void CloudDeviceManager::recordConnection(const QString& deviceId, int duration,
                                           const QString& status, const QString& errorMsg)
{
    if (!m_authManager->isLoggedIn() || deviceId.isEmpty()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/devices/record");
    auto headers = authHeaders();
    QJsonObject body;
    body["device_id"] = deviceId;
    body["duration"] = duration;
    body["status"] = status;
    if (!errorMsg.isEmpty()) body["error_msg"] = errorMsg;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this, deviceId, status](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [statusCode, errorMsg, deviceId, status]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] recordConnection failed: {}", errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Connection recorded: device={}, status={}",
                         deviceId.toStdString(), status.toStdString());
            });
        });
}

void CloudDeviceManager::fetchConnectionLogs()
{
    if (!m_authManager->isLoggedIn()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/devices/logs");
    auto headers = authHeaders();

    infra::HttpRequest::instance().sendGetRequest(
        url, headers, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] fetchConnectionLogs failed: {}", errorMsg);
                    return;
                }
                QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
                QJsonArray logs = doc.object()["logs"].toArray();
                m_connectionLogs.clear();
                for (const auto& v : logs) {
                    m_connectionLogs.append(v.toObject().toVariantMap());
                }
                LOG_INFO("[CloudDeviceManager] Fetched {} connection logs", m_connectionLogs.size());
                emit connectionLogsChanged();
            });
        });
}

// ---- Favorites ----

void CloudDeviceManager::fetchFavorites()
{
    if (!m_authManager->isLoggedIn()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/favorites");
    auto headers = authHeaders();

    infra::HttpRequest::instance().sendGetRequest(
        url, headers, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] fetchFavorites failed: {}", errorMsg);
                    return;
                }
                QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
                QJsonArray favorites = doc.object()["favorites"].toArray();
                m_myFavorites.clear();
                for (const auto& v : favorites) {
                    m_myFavorites.append(v.toObject().toVariantMap());
                }
                LOG_INFO("[CloudDeviceManager] Fetched {} favorites", m_myFavorites.size());
                emit myFavoritesChanged();
            });
        });
}

void CloudDeviceManager::addFavorite(const QString& deviceId, const QString& name, const QString& password)
{
    if (!m_authManager->isLoggedIn()) return;

    QString favName = name;
    if (favName.isEmpty()) {
        for (const auto& v : m_myDevices) {
            QVariantMap d = v.toMap();
            if (d["device_id"].toString() == deviceId) {
                favName = d["remark"].toString();
                break;
            }
        }
    }

    QUrl url(httpBaseUrl() + "api/v1/user/favorites");
    auto headers = authHeaders();
    QJsonObject body;
    body["device_id"] = deviceId;
    if (!favName.isEmpty()) body["device_name"] = favName;
    if (!password.isEmpty()) body["access_password"] = password;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] addFavorite failed: {}", errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Favorite added");
                fetchFavorites();
            });
        });
}

void CloudDeviceManager::updateFavorite(const QString& deviceId, const QString& name, const QString& password)
{
    if (!m_authManager->isLoggedIn()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/favorites/" + deviceId);
    auto headers = authHeaders();
    QJsonObject body;
    if (!name.isEmpty()) body["device_name"] = name;
    if (!password.isEmpty()) body["access_password"] = password;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    infra::HttpRequest::instance().sendPutRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] updateFavorite failed: {}", errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Favorite updated");
                fetchFavorites();
            });
        });
}

void CloudDeviceManager::removeFavorite(const QString& deviceId)
{
    if (!m_authManager->isLoggedIn()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/favorites/" + deviceId);
    auto headers = authHeaders();

    infra::HttpRequest::instance().sendDeleteRequest(
        url, headers, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[CloudDeviceManager] removeFavorite failed: {}", errorMsg);
                    return;
                }
                LOG_INFO("[CloudDeviceManager] Favorite removed");
                fetchFavorites();
            });
        });
}

// ---- Sync WebSocket ----

void CloudDeviceManager::startSync()
{
    if (!m_authManager->isLoggedIn()) return;
    stopSync();

    m_syncSocket = new QWebSocket(QString(), QWebSocketProtocol::VersionLatest, this);
    connect(m_syncSocket, &QWebSocket::textMessageReceived,
            this, &CloudDeviceManager::onSyncTextMessageReceived);
    connect(m_syncSocket, &QWebSocket::disconnected,
            this, &CloudDeviceManager::onSyncDisconnected);

    QString wsUrl = m_serverManager->serverUrl();
    if (!wsUrl.endsWith("/")) wsUrl += "/";
    wsUrl += "api/v1/user/sync?token=" + m_authManager->token();

    LOG_INFO("[CloudDeviceManager] Connecting sync WebSocket: {}", wsUrl.toStdString());
    m_syncSocket->open(QUrl(wsUrl));
}

void CloudDeviceManager::stopSync()
{
    if (m_syncSocket) {
        disconnect(m_syncSocket, nullptr, this, nullptr);
        m_syncSocket->close();
        m_syncSocket->deleteLater();
        m_syncSocket = nullptr;
        LOG_INFO("[CloudDeviceManager] Sync WebSocket stopped");
    }
}

void CloudDeviceManager::onSyncTextMessageReceived(const QString& message)
{
    QJsonDocument doc = QJsonDocument::fromJson(message.toUtf8());
    if (!doc.isObject()) return;
    QJsonObject msg = doc.object();
    LOG_INFO("[CloudDeviceManager] Sync message: {}", msg["type"].toString().toStdString());
    handleSyncMessage(msg);
    emit syncMessage(msg);
}

void CloudDeviceManager::onSyncDisconnected()
{
    LOG_WARN("[CloudDeviceManager] Sync WebSocket disconnected, reconnecting in {}ms", kSyncReconnectDelayMs);
    if (m_authManager->isLoggedIn()) {
        QTimer::singleShot(kSyncReconnectDelayMs, this, &CloudDeviceManager::startSync);
    }
}

void CloudDeviceManager::handleSyncMessage(const QJsonObject& msg)
{
    QString type = msg["type"].toString();

    if (type == "device_online" || type == "device_offline" ||
        type == "device_access_code_changed" || type == "device_remark_changed") {
        // Refresh device list
        fetchMyDevices();
    } else if (type == "favorite_added" || type == "favorite_updated" || type == "favorite_removed") {
        // Refresh favorites
        fetchFavorites();
    }
}

QVariantList CloudDeviceManager::myDevices() const { return m_myDevices; }
QVariantList CloudDeviceManager::myFavorites() const { return m_myFavorites; }
QVariantList CloudDeviceManager::connectionLogs() const { return m_connectionLogs; }

} // namespace quickdesk
