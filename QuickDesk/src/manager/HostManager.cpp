// Copyright 2026 QuickDesk Authors

#include "HostManager.h"
#include "NativeMessaging.h"
#include <QDebug>
#include <QJsonArray>

namespace quickdesk {

HostManager::HostManager(QObject* parent)
    : QObject(parent)
{
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
    }
}

void HostManager::connectToServer(const QString& serverUrl)
{
    if (!m_messaging || !m_messaging->isReady()) {
        emit errorOccurred("NOT_READY", "Host process is not ready");
        return;
    }

    QJsonObject message;
    message["type"] = "connect";
    // Only serverUrl is needed - Host will auto-generate deviceId and accessCode
    message["signalingServerUrl"] = serverUrl;

    qInfo() << "Sending connect message to host, serverUrl:" << serverUrl;
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

void HostManager::refreshTempPassword()
{
    if (!m_messaging || !m_messaging->isReady()) {
        emit refreshTempPasswordResult(false, "NOT_READY", "Host process not ready");
        return;
    }

    // Check if connected to signaling server
    if (m_signalingState != "connected") {
        emit refreshTempPasswordResult(false, "NOT_CONNECTED", 
            "Not connected to signaling server");
        return;
    }

    QJsonObject message;
    message["type"] = "refreshTempPassword";
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
    
    qDebug() << "Host received message:" << type;

    if (type == "helloResponse") {
        handleHelloResponse(message);
    } else if (type == "hostReady") {
        handleHostReady(message);
    } else if (type == "temporaryPasswordChanged") {
        handleTemporaryPasswordChanged(message);
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
    } else if (type == "refreshTempPasswordResponse") {
        handleRefreshTempPasswordResponse(message);
    } else {
        qWarning() << "Unknown message type from host:" << type;
    }
}

void HostManager::onMessagingError(const QString& error)
{
    emit errorOccurred("MESSAGING_ERROR", error);
}

void HostManager::handleHelloResponse(const QJsonObject& message)
{
    QString version = message["version"].toString();
    qInfo() << "Host hello response, version:" << version;
    emit helloResponseReceived(version);
}

void HostManager::handleHostReady(const QJsonObject& message)
{
    m_deviceId = message["deviceId"].toString();
    m_accessCode = message["accessCode"].toString();
    m_isConnected = true;

    qInfo() << "Host ready - Device ID:" << m_deviceId 
            << "Access Code:" << m_accessCode;

    emit deviceIdChanged();
    emit accessCodeChanged();
    emit connectionStatusChanged();
    emit hostReady(m_deviceId, m_accessCode);
}

void HostManager::handleTemporaryPasswordChanged(const QJsonObject& message)
{
    QString newPassword = message["accessCode"].toString();
    
    if (!newPassword.isEmpty()) {
        m_accessCode = newPassword;
        
        qInfo() << "Temporary password changed to:" << m_accessCode;
        
        emit accessCodeChanged();
        emit temporaryPasswordChanged(m_accessCode);
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
    
    // Also support nested clientInfo for backward compatibility
    QJsonObject clientInfo = message["clientInfo"].toObject();
    if (clientUsername.isEmpty()) {
        clientUsername = clientInfo["username"].toString();
    }

    SessionInfo session;
    session.connectionId = clientId;
    session.username = clientUsername;
    session.ip = clientInfo["ip"].toString();
    session.deviceName = clientInfo["deviceName"].toString();
    session.state = "connected";

    m_clients[clientId] = session;

    qInfo() << "Client connected:" << clientId << session.username;

    emit clientCountChanged();
    emit clientListChanged();
    emit clientConnected(clientId, message);
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

    qInfo() << "Client disconnected:" << clientId << clientUsername << reason;

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

    qInfo() << "Authorization request from:" << username << ip;

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
    
    qWarning() << "Host error:" << code << errorMsg;
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

void HostManager::handleRefreshTempPasswordResponse(const QJsonObject& message)
{
    bool success = message["success"].toBool();
    
    if (success) {
        QString newPassword = message["newPassword"].toString();
        qInfo() << "Temporary password refreshed:" << newPassword;
        
        m_accessCode = newPassword;
        emit accessCodeChanged();
        emit temporaryPasswordChanged(newPassword);
        emit refreshTempPasswordResult(true, "", "");
    } else {
        QString errorCode = message["error"].toString();
        QString errorMessage = message["errorMessage"].toString();
        qWarning() << "Failed to refresh password:" << errorCode << errorMessage;
        emit refreshTempPasswordResult(false, errorCode, errorMessage);
    }
}

} // namespace quickdesk
