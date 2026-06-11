// Copyright 2026 QuickDesk Authors

#include "HostManager.h"
#include "NativeMessaging.h"
#include "core/localconfigcenter.h"
#include "infra/env/applicationcontext.h"
#include "infra/log/log.h"
#include <QJsonArray>

namespace quickdesk {

namespace {
// §2.25 native-messaging protocol version. Bumped to 2 for the
// signaling-server v1 refactor (host now ships device_secret + uses
// first-frame auth for signaling WS).
constexpr int kNativeMessagingProtocolVersion = 2;
}

HostManager::HostManager(QObject* parent)
    : QObject(parent)
{
    // Don't hardcode TURN servers here anymore
    // They will be set from TurnServerManager when connecting
    LOG_INFO("HostManager initialized");
}

void HostManager::setMessaging(NativeMessaging* messaging)
{
    if (m_messaging) {
        QObject::disconnect(m_messaging, nullptr, this, nullptr);
    }

    m_messaging = messaging;

    if (m_messaging) {
        connect(m_messaging, &NativeMessaging::messageReceived,
                this, &HostManager::onMessageReceived);
        connect(m_messaging, &NativeMessaging::errorOccurred,
                this, &HostManager::onMessagingError);
    } else {
        // Clear all state when messaging is disconnected (process stopped)
        bool hadClients = !m_clients.isEmpty();
        m_clients.clear();
        m_deviceId.clear();
        m_accessCode.clear();
        // device_secret is runtime-only (§2.22). Clear it so a restarted
        // host can deliver a new one via helloResponse.
        m_deviceSecret.clear();
        m_isConnected = false;
        m_signalingState = "disconnected";
        m_signalingRetryCount = 0;
        m_signalingNextRetryIn = 0;
        m_signalingError.clear();
        
        emit deviceIdChanged();
        emit accessCodeChanged();
        emit connectionStatusChanged();
        emit signalingStateChanged();
        if (hadClients) {
            emit clientCountChanged();
            emit clientListChanged();
        }
    }
}

void HostManager::connectToServer(const QString& serverUrl, const QString& savedAccessCode)
{
    if (!m_messaging || !m_messaging->isReady()) {
        emit errorOccurred("NOT_READY", "Host process is not ready");
        return;
    }

    QJsonObject message;
    message["type"] = "connect";
    // Only serverUrl is needed - Host will auto-generate deviceId and accessCode
    message["signalingServerUrl"] = serverUrl;

    // Send app version so host can report it to signaling server
    message["appVersion"] = infra::ApplicationContext::instance().applicationVersion();

    // Pass runtime API key so host process can authenticate with signaling server
    QString apiKey = core::LocalConfigCenter::instance().apiKey();
    if (!apiKey.isEmpty()) {
        message["apiKey"] = apiKey;
    }

#ifdef Q_OS_WIN
    // 改用system模式提升权限了，这里不再区分是否使用UIAccess了，统一告诉Host不使用提升的权限运行
    message["useElevatedHost"] = false;
#endif

    // If savedAccessCode is provided (never refresh mode), include it
    if (!savedAccessCode.isEmpty()) {
        message["accessCode"] = savedAccessCode;
        LOG_INFO("Sending connect message with saved access code: {}", savedAccessCode.toStdString());
    } else {
        LOG_INFO("Sending connect message to host, serverUrl: {}", serverUrl.toStdString());
    }
    
    if (!m_iceConfig.isEmpty()) {
        message["iceConfig"] = m_iceConfig;
        QJsonArray servers = m_iceConfig.value("iceServers").toArray();
        LOG_INFO("Sending ICE config with {} server(s), lifetime={}",
                 servers.size(),
                 m_iceConfig.value("lifetimeDuration").toString("unset").toStdString());
    } else {
        LOG_INFO("No ICE config available, host will use defaults");
    }

    m_messaging->sendMessage(message);
}

void HostManager::disconnectFromServer()
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }

    QJsonObject message;
    message["type"] = "disconnect";
    m_messaging->sendMessage(message);

    m_isConnected = false;
    m_deviceId.clear();
    m_accessCode.clear();
    m_clients.clear();

    emit connectionStatusChanged();
    emit deviceIdChanged();
    emit accessCodeChanged();
    emit clientCountChanged();
    emit clientListChanged();
}

void HostManager::sendHello()
{
    if (!m_messaging || !m_messaging->isReady()) {
        emit errorOccurred("NOT_READY", "Host process is not ready");
        return;
    }

    QJsonObject message;
    message["type"] = "hello";
    // §2.25 native-messaging protocol versioning. The host must echo
    // this in helloResponse; mismatch → nativeMessagingProtocolMismatch.
    message["protocol_version"] = kNativeMessagingProtocolVersion;
    m_messaging->sendMessage(message);
}

void HostManager::authorizeClient(const QString& connectionId, bool authorized)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }

    QJsonObject message;
    message["type"] = "authorizationResponse";
    message["connectionId"] = connectionId;
    message["authorized"] = authorized;
    m_messaging->sendMessage(message);
}

void HostManager::kickClient(const QString& connectionId)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }

    QJsonObject message;
    message["type"] = "kickClient";
    message["connectionId"] = connectionId;
    m_messaging->sendMessage(message);
}

void HostManager::refreshAccessCode()
{
    if (!m_messaging || !m_messaging->isReady()) {
        emit refreshAccessCodeResult(false, "NOT_READY", "Host process not ready");
        return;
    }

    // Check if connected to signaling server
    if (m_signalingState != "connected") {
        emit refreshAccessCodeResult(false, "NOT_CONNECTED", 
            "Not connected to signaling server");
        return;
    }

    QJsonObject message;
    message["type"] = "refreshAccessCode";
    m_messaging->sendMessage(message);
}

QString HostManager::deviceId() const
{
    return m_deviceId;
}

QString HostManager::accessCode() const
{
    return m_accessCode;
}

QString HostManager::deviceSecret() const
{
    return m_deviceSecret;
}

bool HostManager::isConnected() const
{
    return m_isConnected;
}

int HostManager::clientCount() const
{
    return m_clients.size();
}

QStringList HostManager::clientIds() const
{
    return m_clients.keys();
}

QList<SessionInfo> HostManager::connectedClients() const
{
    return m_clients.values();
}

QString HostManager::getClientUsername(const QString& clientId) const
{
    if (m_clients.contains(clientId)) {
        return m_clients[clientId].username;
    }
    return QString();
}

QString HostManager::getClientDeviceId(const QString& clientId) const
{
    if (m_clients.contains(clientId)) {
        QString deviceId = m_clients[clientId].deviceId;
        if (!deviceId.isEmpty()) {
            return deviceId;
        }
        // Fallback: if deviceId is empty, show a friendly label
        int index = m_clients.keys().indexOf(clientId) + 1;
        return tr("Remote Device %1").arg(index);
    }
    return tr("Unknown Device");
}

QString HostManager::getClientState(const QString& clientId) const
{
    if (m_clients.contains(clientId)) {
        QString state = m_clients[clientId].state;
        if (state == "connected") return "已连接";
        if (state == "authenticating") return "认证中...";
        if (state == "disconnected") return "已断开";
        return state;
    }
    return QString();
}

void HostManager::onMessageReceived(const QJsonObject& message)
{
    QString type = message["type"].toString();
    
    LOG_DEBUG("Host received message: {}", type.toStdString());

    if (type == "helloResponse") {
        handleHelloResponse(message);
    } else if (type == "connectResponse") {
        handleConnectResponse(message);
    } else if (type == "hostReady") {
        handleHostReady(message);
    } else if (type == "accessCodeChanged") {
        handleAccessCodeChanged(message);
    } else if (type == "natPolicyChanged") {
        handleNatPolicyChanged(message);
    } else if (type == "clientConnected") {
        handleClientConnected(message);
    } else if (type == "clientDisconnected") {
        handleClientDisconnected(message);
    } else if (type == "authorizationRequest") {
        handleAuthorizationRequest(message);
    } else if (type == "clientListChanged") {
        handleClientListChanged(message);
    } else if (type == "error") {
        handleError(message);
    } else if (type == "signalingStateChanged") {
        handleSignalingStateChanged(message);
    } else if (type == "refreshAccessCodeResponse") {
        handleRefreshAccessCodeResponse(message);
    } else if (type == "disconnectResponse") {
        handleDisconnectResponse(message);
    } else if (type == "skillMessage") {
        handleSkillMessage(message);
    } else if (type == "privacyScreenStateChanged") {
        handlePrivacyScreenStateChanged(message);
    } else {
        LOG_WARN("Unknown message type from host: {}", type.toStdString());
    }
}

void HostManager::onMessagingError(const QString& error)
{
    emit errorOccurred("MESSAGING_ERROR", error);
}

void HostManager::handleHelloResponse(const QJsonObject& message)
{
    QString version = message["version"].toString();

    // §2.25: check native-messaging protocol version. If the host didn't
    // send one, assume legacy (v1) and log a warning — we can't safely
    // rely on the new fields but we don't outright refuse either,
    // because the host may be mid-upgrade.
    int hostProtocolVersion = message.value("protocol_version").toInt(1);
    if (hostProtocolVersion != kNativeMessagingProtocolVersion) {
        LOG_WARN("Native-messaging protocol mismatch: host={} qt={}",
                 hostProtocolVersion, kNativeMessagingProtocolVersion);
        emit nativeMessagingProtocolMismatch(hostProtocolVersion,
                                              kNativeMessagingProtocolVersion);
    }

    // helloResponse may ALSO carry {device_id, device_secret, access_code}
    // when the host has already finished provisioning by the time Qt
    // sends its hello (host cold start < Qt hello round-trip). Apply any
    // present fields so downstream flows (CloudDeviceManager::syncAccessCode)
    // don't have to wait for a separate hostReady.
    QString deviceId     = message.value("device_id").toString();
    if (deviceId.isEmpty()) deviceId = message.value("deviceId").toString();
    QString accessCode   = message.value("access_code").toString();
    if (accessCode.isEmpty()) accessCode = message.value("accessCode").toString();
    QString deviceSecret = message.value("device_secret").toString();

    bool deviceIdChangedFlag = false;
    bool accessCodeChangedFlag = false;
    if (!deviceId.isEmpty() && deviceId != m_deviceId) {
        m_deviceId = deviceId;
        deviceIdChangedFlag = true;
    }
    if (!accessCode.isEmpty() && accessCode != m_accessCode) {
        m_accessCode = accessCode;
        accessCodeChangedFlag = true;
    }
    if (!deviceSecret.isEmpty() && deviceSecret != m_deviceSecret) {
        m_deviceSecret = deviceSecret;
        LOG_INFO("Host device_secret received (len={})", m_deviceSecret.size());
        emit deviceSecretReady(m_deviceSecret);
    }
    if (deviceIdChangedFlag) emit deviceIdChanged();
    if (accessCodeChangedFlag) emit accessCodeChanged();

    LOG_INFO("Host hello response, version: {}, protocol_version: {}",
             version.toStdString(), hostProtocolVersion);
    emit helloResponseReceived(version);
}

void HostManager::handleConnectResponse(const QJsonObject& message)
{
    Q_UNUSED(message);
    LOG_INFO("Host connect response received");
}

void HostManager::handleNatPolicyChanged(const QJsonObject& message)
{
    Q_UNUSED(message);
    LOG_INFO("Host NAT policy changed");
}

void HostManager::handleHostReady(const QJsonObject& message)
{
    m_deviceId = message["deviceId"].toString();
    if (m_deviceId.isEmpty()) m_deviceId = message["device_id"].toString();
    m_accessCode = message["accessCode"].toString();
    if (m_accessCode.isEmpty()) m_accessCode = message["access_code"].toString();
    m_isConnected = true;

    // §2.22 / §2.23: host ships device_secret on hostReady so Qt can
    // call device-level APIs (syncAccessCode). Runtime-only, never
    // persisted; cleared on setMessaging(null).
    QString deviceSecret = message.value("device_secret").toString();
    if (!deviceSecret.isEmpty() && deviceSecret != m_deviceSecret) {
        m_deviceSecret = deviceSecret;
        LOG_INFO("Host device_secret received via hostReady (len={})",
                 m_deviceSecret.size());
        emit deviceSecretReady(m_deviceSecret);
    }

    qInfo() << "Host ready - Device ID:" << m_deviceId 
            << "Access Code:" << m_accessCode;

    emit deviceIdChanged();
    emit accessCodeChanged();
    emit connectionStatusChanged();
    emit hostReady(m_deviceId, m_accessCode);
}

void HostManager::handleAccessCodeChanged(const QJsonObject& message)
{
    QString newAccessCode = message["accessCode"].toString();
    if (newAccessCode.isEmpty()) newAccessCode = message["access_code"].toString();
    
    if (!newAccessCode.isEmpty()) {
        m_accessCode = newAccessCode;
        
        LOG_INFO("Access code changed to: {}", m_accessCode.toStdString());
        
        emit accessCodeChanged();
    }
}

void HostManager::handleClientConnected(const QJsonObject& message)
{
    // Support both "clientId" (new format) and "connectionId" (old format)
    QString clientId = message["clientId"].toString();
    if (clientId.isEmpty()) {
        clientId = message["connectionId"].toString();
    }
    QString clientUsername = message["clientUsername"].toString();
    QString deviceId = message["deviceId"].toString();  // Client's device ID
    
    // Also support nested clientInfo for backward compatibility
    QJsonObject clientInfo = message["clientInfo"].toObject();
    if (clientUsername.isEmpty()) {
        clientUsername = clientInfo["username"].toString();
    }

    SessionInfo session;
    session.connectionId = clientId;
    session.username = clientUsername;
    session.deviceId = deviceId;
    session.ip = clientInfo["ip"].toString();
    session.deviceName = clientInfo["deviceName"].toString();
    session.state = "connected";

    m_clients[clientId] = session;

    LOG_INFO("Client connected: {} {} device_id: {}", 
             clientId.toStdString(), 
             session.username.toStdString(),
             deviceId.toStdString());

    emit clientCountChanged();
    emit clientListChanged();
    emit clientConnected(clientId, message);

    // Auto-enable privacy screen if configured
    if (core::LocalConfigCenter::instance().autoPrivacyScreenOnConnect()) {
        LOG_INFO("Auto-enabling privacy screen on client connect");
        togglePrivacyScreen(true);
    }
}

void HostManager::handleClientDisconnected(const QJsonObject& message)
{
    // Support both "clientId" (new format) and "connectionId" (old format)
    QString clientId = message["clientId"].toString();
    if (clientId.isEmpty()) {
        clientId = message["connectionId"].toString();
    }
    QString clientUsername = message["clientUsername"].toString();
    QString reason = message["reason"].toString();

    m_clients.remove(clientId);

    LOG_INFO("Client disconnected: {} {} {}", clientId.toStdString(), clientUsername.toStdString(), reason.toStdString());

    emit clientCountChanged();
    emit clientListChanged();
    emit clientDisconnected(clientId, reason);
}

void HostManager::handleAuthorizationRequest(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    QJsonObject clientInfo = message["clientInfo"].toObject();
    
    QString username = clientInfo["username"].toString();
    QString ip = clientInfo["ip"].toString();

    LOG_INFO("Authorization request from: {} {}", username.toStdString(), ip.toStdString());

    emit authorizationRequested(connectionId, username, ip);
}

void HostManager::handleClientListChanged(const QJsonObject& message)
{
    m_clients.clear();
    
    QJsonArray clients = message["clients"].toArray();
    for (const QJsonValue& value : clients) {
        QJsonObject obj = value.toObject();
        SessionInfo session;
        session.connectionId = obj["connectionId"].toString();
        session.username = obj["username"].toString();
        session.ip = obj["ip"].toString();
        session.deviceName = obj["deviceName"].toString();
        session.connectedAt = obj["connectedAt"].toString();
        session.state = obj["state"].toString();
        
        m_clients[session.connectionId] = session;
    }

    emit clientCountChanged();
    emit clientListChanged();
}

void HostManager::handleError(const QJsonObject& message)
{
    QString code = message["code"].toString();
    QString errorMsg = message["message"].toString();
    
    LOG_WARN("Host error: {} {}", code.toStdString(), errorMsg.toStdString());
    emit errorOccurred(code, errorMsg);
}

void HostManager::handleSignalingStateChanged(const QJsonObject& message)
{
    QString state = message["state"].toString();
    int retryCount = message["retryCount"].toInt();
    int nextRetryIn = message["nextRetryIn"].toInt();
    QString error = message["error"].toString();
    
    qInfo() << "Signaling state changed:" << state
            << "retry:" << retryCount
            << "next:" << nextRetryIn << "s"
            << "error:" << error;
    
    bool changed = false;
    
    if (m_signalingState != state) {
        m_signalingState = state;
        changed = true;
    }
    if (m_signalingRetryCount != retryCount) {
        m_signalingRetryCount = retryCount;
        changed = true;
    }
    if (m_signalingNextRetryIn != nextRetryIn) {
        m_signalingNextRetryIn = nextRetryIn;
        changed = true;
    }
    if (m_signalingError != error) {
        m_signalingError = error;
        changed = true;
    }
    
    // Update connected status based on signaling state
    bool wasConnected = m_isConnected;
    m_isConnected = (state == "connected");
    
    if (changed) {
        emit signalingStateChanged();
    }
    
    if (wasConnected != m_isConnected) {
        emit connectionStatusChanged();
    }
}

void HostManager::handleDisconnectResponse(const QJsonObject& message)
{
    Q_UNUSED(message);
    LOG_INFO("Host disconnect response received");
}

QString HostManager::signalingState() const
{
    return m_signalingState;
}

int HostManager::signalingRetryCount() const
{
    return m_signalingRetryCount;
}

int HostManager::signalingNextRetryIn() const
{
    return m_signalingNextRetryIn;
}

QString HostManager::signalingError() const
{
    return m_signalingError;
}

void HostManager::handleRefreshAccessCodeResponse(const QJsonObject& message)
{
    bool success = message["success"].toBool();
    
    if (success) {
        QString newAccessCode = message["accessCode"].toString();
        LOG_INFO("Access code refreshed: {}", newAccessCode.toStdString());
        
        m_accessCode = newAccessCode;
        emit accessCodeChanged();
        emit refreshAccessCodeResult(true, "", "");
    } else {
        QString errorCode = message["error"].toString();
        QString errorMessage = message["errorMessage"].toString();
        LOG_WARN("Failed to refresh access code: {} {}", errorCode.toStdString(), errorMessage.toStdString());
        emit refreshAccessCodeResult(false, errorCode, errorMessage);
    }
}

void HostManager::setIceConfig(const QJsonObject& iceConfig)
{
    m_iceConfig = iceConfig;
    QJsonArray servers = m_iceConfig.value("iceServers").toArray();
    LOG_INFO("ICE config updated: {} server(s)", servers.size());
}

QJsonObject HostManager::getIceConfig() const
{
    return m_iceConfig;
}

void HostManager::sendSkillBridgeSend(const QString& jsonData)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }

    QJsonObject message;
    message["type"] = "skillBridgeSend";
    message["data"] = jsonData;
    m_messaging->sendMessage(message);
}

void HostManager::togglePrivacyScreen(bool enabled)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }

    QJsonObject message;
    message["type"] = "togglePrivacyScreen";
    message["enabled"] = enabled;
    m_messaging->sendMessage(message);
}

void HostManager::handlePrivacyScreenStateChanged(const QJsonObject& message)
{
    bool enabled = message["enabled"].toBool();
    LOG_INFO("Privacy screen state changed: {}", enabled ? "ON" : "OFF");
    emit privacyScreenStateChanged(enabled);
}

void HostManager::handleSkillMessage(const QJsonObject& message)
{
    QString data = message["data"].toString();
    emit skillMessage(data);
}

} // namespace quickdesk
