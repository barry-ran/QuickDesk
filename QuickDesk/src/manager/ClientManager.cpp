// Copyright 2026 QuickDesk Authors
// Client Manager Implementation

#include "ClientManager.h"
#include "NativeMessaging.h"
#include "infra/log/log.h"
#include <QUuid>
#include <QJsonArray>
#include <QDir>
#include <QFileInfo>
#include <QDateTime>

namespace quickdesk {

ClientManager::ClientManager(QObject* parent)
    : QObject(parent)
    , m_sharedMemoryManager(std::make_unique<SharedMemoryManager>(this))
{
}

void ClientManager::setMessaging(NativeMessaging* messaging)
{
    if (m_messaging) {
        QObject::disconnect(m_messaging, nullptr, this, nullptr);
    }

    m_messaging = messaging;

    if (m_messaging) {
        connect(m_messaging, &NativeMessaging::messageReceived,
                this, &ClientManager::onMessageReceived);
        connect(m_messaging, &NativeMessaging::errorOccurred,
                this, &ClientManager::onMessagingError);
    } else {
        // Clear all state when messaging is disconnected (process stopped)
        bool hadConnections = !m_connections.isEmpty();
        m_connections.clear();
        m_activeConnectionId.clear();
        // Note: Don't reset m_connectionCounter to avoid ID conflicts after restart
        
        if (hadConnections) {
            emit connectionCountChanged();
            emit connectionListChanged();
        }
    }
}

QString ClientManager::connectToHost(const QString& deviceId,
                                     const QString& accessCode,
                                     const QString& serverUrl)
{
    if (!m_messaging || !m_messaging->isReady()) {
        emit errorOccurred("", "NOT_READY", "Client process is not ready");
        return QString();
    }

    QString connectionId = generateConnectionId();

    // Create connection info
    ConnectionInfo conn;
    conn.connectionId = connectionId;
    conn.deviceId = deviceId;
    conn.rtcState = RtcStatus::Connecting;
    m_connections[connectionId] = conn;

    // Send connect message
    QJsonObject message;
    message["type"] = "connectToHost";
    message["connectionId"] = connectionId;
    message["deviceId"] = deviceId;
    message["accessCode"] = accessCode;
    message["serverUrl"] = serverUrl;
    
    // Include ICE configuration if provided (using standard Chrome Remoting format)
    if (!m_iceServers.isEmpty()) {
        QJsonObject iceConfig;
        iceConfig["iceServers"] = m_iceServers;
        message["iceConfig"] = iceConfig;
        LOG_INFO("Client: Sending ICE config with {} server(s)", m_iceServers.size());
    } else {
        LOG_INFO("Client: No custom ICE servers configured, client will use defaults");
    }

    LOG_INFO("Connecting to host: {} connectionId: {}", deviceId.toStdString(), connectionId.toStdString());
    m_messaging->sendMessage(message);

    emit connectionCountChanged();
    emit connectionAdded(connectionId);
    emit connectionListChanged();

    // Set as active if first connection
    if (m_activeConnectionId.isEmpty()) {
        setActiveConnectionId(connectionId);
    }

    return connectionId;
}

void ClientManager::disconnectFromHost(const QString& connectionId)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }

    QJsonObject message;
    message["type"] = "disconnectFromHost";
    message["connectionId"] = connectionId;
    m_messaging->sendMessage(message);

    m_connections.remove(connectionId);

    emit connectionCountChanged();
    emit connectionRemoved(connectionId);
    emit connectionListChanged();

    // Update active connection if needed
    if (m_activeConnectionId == connectionId) {
        if (m_connections.isEmpty()) {
            setActiveConnectionId(QString());
        } else {
            setActiveConnectionId(m_connections.firstKey());
        }
    }
}

void ClientManager::disconnectAll()
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }

    QJsonObject message;
    message["type"] = "disconnectAll";
    m_messaging->sendMessage(message);

    m_connections.clear();
    m_activeConnectionId.clear();

    emit connectionCountChanged();
    emit activeConnectionChanged();
    emit connectionListChanged();
}

void ClientManager::sendHello(const QString& deviceId)
{
    if (!m_messaging || !m_messaging->isReady()) {
        emit errorOccurred("", "NOT_READY", "Client process is not ready");
        return;
    }

    QJsonObject message;
    message["type"] = "hello";
    // Send local device_id so client process can use it for signaling identification
    if (!deviceId.isEmpty()) {
        message["deviceId"] = deviceId;
    }
    m_messaging->sendMessage(message);
}

void ClientManager::sendMouseMove(const QString& connectionId, int x, int y)
{
    sendMouseEvent(connectionId, "move", x, y, 0, 0);
}

void ClientManager::sendMousePress(const QString& connectionId, int x, int y, int button)
{
    sendMouseEvent(connectionId, "press", x, y, button, 0);
}

void ClientManager::sendMouseRelease(const QString& connectionId, int x, int y, int button)
{
    sendMouseEvent(connectionId, "release", x, y, button, 0);
}

void ClientManager::sendMouseWheel(const QString& connectionId, int x, int y, int delta)
{
    sendMouseEvent(connectionId, "wheel", x, y, 0, delta);
}

void ClientManager::sendKeyPress(const QString& connectionId, int keyCode, int modifiers)
{
    sendKeyboardEvent(connectionId, "press", keyCode, modifiers);
}

void ClientManager::sendKeyRelease(const QString& connectionId, int keyCode, int modifiers)
{
    sendKeyboardEvent(connectionId, "release", keyCode, modifiers);
}

void ClientManager::syncClipboard(const QString& connectionId, const QString& text)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }

    QJsonObject message;
    message["type"] = "clipboardSync";
    message["connectionId"] = connectionId;
    message["text"] = text;
    m_messaging->sendMessage(message);
}

void ClientManager::setTargetFramerate(const QString& connectionId, int framerate)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot set framerate: messaging not ready");
        return;
    }

    // Clamp to valid range
    framerate = qBound(1, framerate, 60);

    LOG_INFO("Setting target framerate for {}: {} FPS", 
             connectionId.toStdString(), framerate);

    QJsonObject message;
    message["type"] = "setFramerate";
    message["connectionId"] = connectionId;
    message["framerate"] = framerate;
    m_messaging->sendMessage(message);
}

void ClientManager::setResolution(const QString& connectionId, int width, int height, int dpi)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot set resolution: messaging not ready");
        return;
    }

    // Validate dimensions
    if (width <= 0 || height <= 0 || width > 8192 || height > 8192) {
        LOG_WARN("Invalid resolution: {}x{}", width, height);
        return;
    }

    LOG_INFO("Setting resolution for {}: {}x{} @ {} DPI", 
             connectionId.toStdString(), width, height, dpi);

    QJsonObject message;
    message["type"] = "setResolution";
    message["connectionId"] = connectionId;
    message["width"] = width;
    message["height"] = height;
    message["dpi"] = dpi;
    m_messaging->sendMessage(message);
}

void ClientManager::setFramerateBoost(const QString& connectionId, bool enabled, 
                                      int captureIntervalMs, int boostDurationMs)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot set framerate boost: messaging not ready");
        return;
    }

    // Clamp to valid ranges
    captureIntervalMs = qBound(10, captureIntervalMs, 1000);
    boostDurationMs = qBound(100, boostDurationMs, 1000);

    LOG_INFO("Setting framerate boost for {}: enabled={}, interval={}ms, duration={}ms", 
             connectionId.toStdString(), enabled, captureIntervalMs, boostDurationMs);

    QJsonObject message;
    message["type"] = "setFramerateBoost";
    message["connectionId"] = connectionId;
    message["enabled"] = enabled;
    message["captureIntervalMs"] = captureIntervalMs;
    message["boostDurationMs"] = boostDurationMs;
    m_messaging->sendMessage(message);
}

void ClientManager::setBitrate(const QString& connectionId, int minBitrateBps)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot set bitrate: messaging not ready");
        return;
    }

    // Validate bitrate (allow 0 to disable, or reasonable range)
    if (minBitrateBps < 0) {
        LOG_WARN("Invalid bitrate: {} (must be >= 0)", minBitrateBps);
        return;
    }

    LOG_INFO("Setting bitrate for {}: {} MiB ({} bps)", 
             connectionId.toStdString(), 
             minBitrateBps / 1024.0 / 1024.0, 
             minBitrateBps);

    QJsonObject message;
    message["type"] = "setBitrate";
    message["connectionId"] = connectionId;
    message["minBitrateBps"] = minBitrateBps;
    m_messaging->sendMessage(message);
}

int ClientManager::connectionCount() const
{
    return m_connections.size();
}

QString ClientManager::activeConnectionId() const
{
    return m_activeConnectionId;
}

void ClientManager::setActiveConnectionId(const QString& id)
{
    if (m_activeConnectionId != id) {
        m_activeConnectionId = id;
        emit activeConnectionChanged();
    }
}

QList<ConnectionInfo> ClientManager::connections() const
{
    return m_connections.values();
}

ConnectionInfo ClientManager::getConnection(const QString& connectionId) const
{
    return m_connections.value(connectionId);
}

QStringList ClientManager::connectionIds() const
{
    return m_connections.keys();
}

RtcStatus::Status ClientManager::getConnectionRtcState(const QString& connectionId) const
{
    if (m_connections.contains(connectionId)) {
        return m_connections[connectionId].rtcState;
    }
    return RtcStatus::Disconnected;
}

QString ClientManager::getSignalingState(const QString& connectionId) const
{
    if (m_connections.contains(connectionId)) {
        return m_connections[connectionId].signalingState;
    }
    return "disconnected";
}

int ClientManager::getSignalingRetryCount(const QString& connectionId) const
{
    if (m_connections.contains(connectionId)) {
        return m_connections[connectionId].signalingRetryCount;
    }
    return 0;
}

int ClientManager::getSignalingNextRetryIn(const QString& connectionId) const
{
    if (m_connections.contains(connectionId)) {
        return m_connections[connectionId].signalingNextRetryIn;
    }
    return 0;
}

QString ClientManager::getSignalingError(const QString& connectionId) const
{
    if (m_connections.contains(connectionId)) {
        return m_connections[connectionId].signalingError;
    }
    return QString();
}

QString ClientManager::getConnectionDeviceId(const QString& connectionId) const
{
    if (m_connections.contains(connectionId)) {
        return m_connections[connectionId].deviceId;
    }
    return QString();
}

void ClientManager::onMessageReceived(const QJsonObject& message)
{
    QString type = message["type"].toString();

    if (type != "videoFrameReady" && type != "performanceStatsUpdate") {
        LOG_DEBUG("Client received message: {}", type.toStdString());
    }

    if (type == "helloResponse") {
        handleHelloResponse(message);
    } else if (type == "signalingStateChanged") {
        handleSignalingStateChanged(message);
    } else if (type == "connectToHostResponse") {
        handleConnectToHostResponse(message);
    } else if (type == "connectionStateChanged") {
        handleConnectionStateChanged(message);
    } else if (type == "connectionListChanged") {
        handleConnectionListChanged(message);
    } else if (type == "videoFrameReady") {
        handleVideoFrameReady(message);
    } else if (type == "clipboardReceived") {
        handleClipboardReceived(message);
    } else if (type == "error") {
        handleError(message);
    } else if (type == "connectionFailed") {
        handleConnectionFailed(message);
    } else if (type == "onHostConnected") {
        handleHostConnected(message);
    } else if (type == "onHostDisconnected") {
        handleHostDisconnected(message);
    } else if (type == "onHostConnectionFailed") {
        handleHostConnectionFailed(message);
    } else if (type == "disconnectFromHostResponse") {
        handleDisconnectFromHostResponse(message);
    } else if (type == "disconnectAllResponse") {
        handleDisconnectAllResponse(message);
    } else if (type == "cursorShapeChanged") {
        handleCursorShapeChanged(message);
    } else if (type == "performanceStatsUpdate") {
        handlePerformanceStatsUpdate(message);
    } else if (type == "setFramerateResponse" || type == "setResolutionResponse" || type == "setFramerateBoostResponse" || type == "setBitrateResponse") {
        // Acknowledgement responses - just log success/failure
        bool success = message["success"].toBool();
        if (!success) {
            QString error = message["error"].toString();
            LOG_WARN("{} failed: {}", type.toStdString(), error.toStdString());
        }
    } else {
        LOG_WARN("Unknown message type from client: {}", type.toStdString());
    }
}

void ClientManager::onMessagingError(const QString& error)
{
    emit errorOccurred("", "MESSAGING_ERROR", error);
}

QString ClientManager::generateConnectionId()
{
    return QString("conn_%1").arg(++m_connectionCounter);
}

void ClientManager::handleHelloResponse(const QJsonObject& message)
{
    QString version = message["version"].toString();
    LOG_INFO("Client hello response, version: {}", version.toStdString());
    emit helloResponseReceived(version);
}

void ClientManager::handleSignalingStateChanged(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    QString state = message["state"].toString();
    int retryCount = message["retryCount"].toInt();
    int nextRetryIn = message["nextRetryIn"].toInt();
    QString error = message["error"].toString();

    LOG_INFO("Client signaling state changed: connection={}, state={}, retry={}, next={}s, error={}",
             connectionId.toStdString(), state.toStdString(), retryCount, nextRetryIn, error.toStdString());

    // Update signaling state for this specific connection
    if (m_connections.contains(connectionId)) {
        m_connections[connectionId].signalingState = state;
        m_connections[connectionId].signalingRetryCount = retryCount;
        m_connections[connectionId].signalingNextRetryIn = nextRetryIn;
        m_connections[connectionId].signalingError = error;
        
        // Emit signal with connection ID
        emit signalingStateChanged(connectionId, state, retryCount, nextRetryIn, error);
    } else {
        LOG_WARN("Signaling state changed for unknown connection: {}", connectionId.toStdString());
    }
}

void ClientManager::handleConnectToHostResponse(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();

    if (connectionId.isEmpty()) {
        LOG_WARN("connectToHostResponse missing connectionId");
        return;
    }

    if (!m_connections.contains(connectionId)) {
        ConnectionInfo conn;
        conn.connectionId = connectionId;
        conn.rtcState = RtcStatus::Connecting;
        m_connections[connectionId] = conn;

        emit connectionCountChanged();
        emit connectionAdded(connectionId);
        emit connectionListChanged();

        if (m_activeConnectionId.isEmpty()) {
            setActiveConnectionId(connectionId);
        }
    }
}

void ClientManager::handleConnectionStateChanged(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    QString stateStr = message["state"].toString();
    QJsonObject hostInfo = message["hostInfo"].toObject();

    if (m_connections.contains(connectionId)) {
        // Convert string to enum
        if (stateStr == "connecting") {
            m_connections[connectionId].rtcState = RtcStatus::Connecting;
        } else if (stateStr == "connected") {
            m_connections[connectionId].rtcState = RtcStatus::Connected;
        } else if (stateStr == "disconnected") {
            m_connections[connectionId].rtcState = RtcStatus::Disconnected;
        } else if (stateStr == "failed") {
            m_connections[connectionId].rtcState = RtcStatus::Failed;
        }
        
        if (hostInfo.contains("resolution")) {
            QString resolution = hostInfo["resolution"].toString();
            QStringList parts = resolution.split('x');
            if (parts.size() == 2) {
                m_connections[connectionId].width = parts[0].toInt();
                m_connections[connectionId].height = parts[1].toInt();
            }
        }
        if (hostInfo.contains("deviceName")) {
            m_connections[connectionId].deviceName = hostInfo["deviceName"].toString();
        }
    }

    LOG_INFO("Connection {} RTC state changed to: {}", connectionId.toStdString(), stateStr.toStdString());
    emit connectionStateChanged(connectionId, stateStr, hostInfo);
    emit connectionListChanged();
}

void ClientManager::handleConnectionListChanged(const QJsonObject& message)
{
    m_connections.clear();
    
    QJsonArray connections = message["connections"].toArray();
    for (const QJsonValue& value : connections) {
        QJsonObject obj = value.toObject();
        ConnectionInfo conn;
        conn.connectionId = obj["connectionId"].toString();
        conn.deviceId = obj["deviceId"].toString();
        conn.deviceName = obj["deviceName"].toString();
        conn.connectedAt = obj["connectedAt"].toString();
        
        // Convert string state to enum
        QString stateStr = obj["state"].toString();
        if (stateStr == "connecting") {
            conn.rtcState = RtcStatus::Connecting;
        } else if (stateStr == "connected") {
            conn.rtcState = RtcStatus::Connected;
        } else if (stateStr == "disconnected") {
            conn.rtcState = RtcStatus::Disconnected;
        } else if (stateStr == "failed") {
            conn.rtcState = RtcStatus::Failed;
        } else {
            conn.rtcState = RtcStatus::Disconnected;
        }
        
        m_connections[conn.connectionId] = conn;
    }

    emit connectionCountChanged();
    emit connectionListChanged();
}

void ClientManager::handleVideoFrameReady(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    int frameIndex = message["frameIndex"].toInt();
    int width = message["width"].toInt();
    int height = message["height"].toInt();
    QString sharedMemoryName = message["sharedMemoryName"].toString();
    
    // Attach to shared memory if not already attached
    if (!m_sharedMemoryManager->isAttached(connectionId)) {
        if (!m_sharedMemoryManager->attach(connectionId, sharedMemoryName)) {
            LOG_WARN("Failed to attach to shared memory for connection {}", 
                     connectionId.toStdString());
            return;
        }
        LOG_INFO("Attached to shared memory: {} ({}x{})", 
                 sharedMemoryName.toStdString(), width, height);
    }
    
    // Update connection info with resolution
    if (m_connections.contains(connectionId)) {
        m_connections[connectionId].width = width;
        m_connections[connectionId].height = height;
    }
    
    emit videoFrameReady(connectionId, frameIndex);
}

void ClientManager::handleClipboardReceived(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    QString text = message["text"].toString();
    
    emit clipboardReceived(connectionId, text);
}

void ClientManager::handleError(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    QString code = message["code"].toString();
    QString errorMsg = message["message"].toString();
    
    LOG_WARN("Client error: {} {} {}", connectionId.toStdString(), code.toStdString(), errorMsg.toStdString());
    emit errorOccurred(connectionId, code, errorMsg);
}

void ClientManager::handleConnectionFailed(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    QString errorCode = message["errorCode"].toString();
    QString errorMsg = message["message"].toString();
    
    qWarning() << "Connection failed:" << connectionId 
               << "error:" << errorCode << "-" << errorMsg;
    
    // Update connection state
    if (m_connections.contains(connectionId)) {
        m_connections[connectionId].rtcState = RtcStatus::Failed;
        emit connectionStateChanged(connectionId, "failed", QJsonObject());
    }
    
    // Emit error with specific error code
    emit errorOccurred(connectionId, errorCode, errorMsg);
    
    // Remove failed connection from list
    m_connections.remove(connectionId);
    emit connectionCountChanged();
    emit connectionRemoved(connectionId);
    emit connectionListChanged();
    
    // Update active connection if needed
    if (m_activeConnectionId == connectionId) {
        if (m_connections.isEmpty()) {
            setActiveConnectionId(QString());
        } else {
            setActiveConnectionId(m_connections.firstKey());
        }
    }
}

void ClientManager::handleHostConnected(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    
    LOG_INFO("Host connected: {}", connectionId.toStdString());
    
    if (m_connections.contains(connectionId)) {
        m_connections[connectionId].rtcState = RtcStatus::Connected;
        emit connectionStateChanged(connectionId, "connected", QJsonObject());
    }
    emit connectionListChanged();
}

void ClientManager::handleHostDisconnected(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    
    LOG_INFO("Host disconnected: {}", connectionId.toStdString());
    
    if (m_connections.contains(connectionId)) {
        m_connections[connectionId].rtcState = RtcStatus::Disconnected;
        emit connectionStateChanged(connectionId, "disconnected", QJsonObject());
    }
    
    // Detach from shared memory
    m_sharedMemoryManager->detach(connectionId);
    
    // Remove disconnected connection
    m_connections.remove(connectionId);
    emit connectionCountChanged();
    emit connectionRemoved(connectionId);
    emit connectionListChanged();
    
    // Update active connection if needed
    if (m_activeConnectionId == connectionId) {
        if (m_connections.isEmpty()) {
            setActiveConnectionId(QString());
        } else {
            setActiveConnectionId(m_connections.firstKey());
        }
    }
}

void ClientManager::handleHostConnectionFailed(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    int errorCode = message["errorCode"].toInt();
    
    LOG_WARN("Host connection failed: {} error code: {}", connectionId.toStdString(), errorCode);
    
    // Update connection state
    if (m_connections.contains(connectionId)) {
        m_connections[connectionId].rtcState = RtcStatus::Failed;
        emit connectionStateChanged(connectionId, "failed", QJsonObject());
    }
    
    // Map protocol::ErrorCode to user-friendly message
    QString errorMsg;
    switch (errorCode) {
        case 1: errorMsg = tr("Authentication failed"); break;
        case 2: errorMsg = tr("Channel error"); break;
        case 3: errorMsg = tr("Connection timeout"); break;
        case 4: errorMsg = tr("Network error"); break;
        default: errorMsg = tr("Connection failed (error code: %1)").arg(errorCode); break;
    }
    
    emit errorOccurred(connectionId, "CONNECTION_FAILED", errorMsg);
    
    // Remove failed connection
    m_connections.remove(connectionId);
    emit connectionCountChanged();
    emit connectionRemoved(connectionId);
    emit connectionListChanged();
    
    // Update active connection if needed
    if (m_activeConnectionId == connectionId) {
        if (m_connections.isEmpty()) {
            setActiveConnectionId(QString());
        } else {
            setActiveConnectionId(m_connections.firstKey());
        }
    }
}

void ClientManager::handleDisconnectFromHostResponse(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    if (!connectionId.isEmpty()) {
        LOG_INFO("Disconnect response received for connection: {}", connectionId.toStdString());
    }
}

void ClientManager::handleDisconnectAllResponse(const QJsonObject& message)
{
    Q_UNUSED(message);
    LOG_INFO("Disconnect all response received");
}

void ClientManager::handleCursorShapeChanged(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    int width = message["width"].toInt();
    int height = message["height"].toInt();
    int hotspotX = message["hotspotX"].toInt();
    int hotspotY = message["hotspotY"].toInt();
    QString base64Data = message["data"].toString();
    
    // Decode base64 data
    QByteArray data = QByteArray::fromBase64(base64Data.toLatin1());
    
    // LOG_DEBUG("Cursor shape changed for connection {}: {}x{} hotspot({}, {}) data size: {}",
    //           connectionId.toStdString(), width, height, hotspotX, hotspotY, data.size());
    
    emit cursorShapeChanged(connectionId, width, height, hotspotX, hotspotY, data);
}

void ClientManager::handlePerformanceStatsUpdate(const QJsonObject& message)
{
    QString connectionId = message["connectionId"].toString();
    
    QVariantMap stats;
    // Timing breakdown (ms)
    stats["captureMs"]      = message["captureMs"].toDouble();
    stats["encodeMs"]       = message["encodeMs"].toDouble();
    stats["decodeMs"]       = message["decodeMs"].toDouble();
    stats["paintMs"]        = message["paintMs"].toDouble();
    stats["totalLatencyMs"] = message["totalLatencyMs"].toDouble();
    stats["roundTripMs"]    = message["roundTripMs"].toDouble();
    // Throughput
    stats["bandwidthKbps"]  = message["bandwidthKbps"].toDouble();
    stats["frameRate"]      = message["frameRate"].toDouble();
    stats["packetRate"]     = message["packetRate"].toDouble();
    // Codec info
    stats["codec"]              = message["codec"].toString("Unknown");
    stats["frameQuality"]       = message["frameQuality"].toInt(-1);
    stats["encodedRectWidth"]   = message["encodedRectWidth"].toInt();
    stats["encodedRectHeight"]  = message["encodedRectHeight"].toInt();
    
    emit performanceStatsUpdated(connectionId, stats);
}

void ClientManager::sendMouseEvent(const QString& connectionId, const QString& eventType,
                                   int x, int y, int button, int wheelDelta)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }

    QJsonObject message;
    message["type"] = "mouseEvent";
    message["connectionId"] = connectionId;
    message["eventType"] = eventType;
    message["x"] = x;
    message["y"] = y;
    message["button"] = button;
    message["wheelDelta"] = wheelDelta;
    m_messaging->sendMessage(message);
}

void ClientManager::sendKeyboardEvent(const QString& connectionId, const QString& eventType,
                                      int keyCode, int modifiers)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }

    QJsonObject message;
    message["type"] = "keyboardEvent";
    message["connectionId"] = connectionId;
    message["eventType"] = eventType;
    message["keyCode"] = keyCode;
    message["modifiers"] = modifiers;
    m_messaging->sendMessage(message);
}

bool ClientManager::saveFrameToFile(const QString& connectionId, 
                                     const QString& filePath)
{
    if (!m_sharedMemoryManager->isAttached(connectionId)) {
        LOG_WARN("Cannot save frame: not attached to shared memory for {}", 
                 connectionId.toStdString());
        return false;
    }
    
    // Read YUV frame and convert to QImage for saving
    QVideoFrame videoFrame = m_sharedMemoryManager->readVideoFrame(connectionId);
    if (!videoFrame.isValid()) {
        LOG_WARN("Cannot save frame: failed to read video frame for {}", 
                 connectionId.toStdString());
        return false;
    }
    
    // Convert YUV frame to QImage
    QImage frame = videoFrame.toImage();
    if (frame.isNull()) {
        LOG_WARN("Cannot save frame: failed to convert video frame to image for {}", 
                 connectionId.toStdString());
        return false;
    }
    
    // Ensure directory exists
    QFileInfo fileInfo(filePath);
    QDir dir = fileInfo.dir();
    if (!dir.exists()) {
        dir.mkpath(".");
    }
    
    // Save to file
    bool success = frame.save(filePath);
    if (success) {
        LOG_INFO("Saved frame to: {} ({}x{})", 
                 filePath.toStdString(), frame.width(), frame.height());
    } else {
        LOG_WARN("Failed to save frame to: {}", filePath.toStdString());
    }
    
    return success;
}

void ClientManager::setIceServers(const QJsonArray& iceServers)
{
    m_iceServers = iceServers;
    LOG_INFO("Client: ICE servers configuration updated: {} server(s)", m_iceServers.size());
}

QJsonArray ClientManager::getIceServers() const
{
    return m_iceServers;
}

} // namespace quickdesk
