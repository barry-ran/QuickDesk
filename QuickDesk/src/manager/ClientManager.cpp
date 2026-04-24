// Copyright 2026 QuickDesk Authors
// Client Manager Implementation

#include "ClientManager.h"
#include "NativeMessaging.h"
#include "core/localconfigcenter.h"
#include "infra/log/log.h"
#include <QUuid>
#include <QJsonArray>
#include <QJsonDocument>
#include <QDir>
#include <QFile>
#include <QFileInfo>
#include <QDateTime>
#include <QClipboard>
#include <QDesktopServices>
#include <QGuiApplication>
#include <QMimeData>
#include <QStandardPaths>
#include <QUrl>

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
        QStringList deviceIds = m_connections.keys();
        for (const auto& devId : deviceIds) {
            emit connectionStateChanged(devId, "disconnected", QJsonObject());
            m_sharedMemoryManager->detach(devId);
            emit connectionRemoved(devId);
        }
        m_connections.clear();
        m_connIdToDeviceId.clear();
        m_activeDeviceId.clear();

        if (!deviceIds.isEmpty()) {
            emit connectionCountChanged();
            emit activeConnectionChanged();
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

    // Reject duplicate connections to the same device
    if (m_connections.contains(deviceId)) {
        auto& existing = m_connections[deviceId];
        if (existing.rtcState != RtcStatus::Disconnected &&
            existing.rtcState != RtcStatus::Failed) {
            LOG_INFO("Device {} already connected (connectionId={}), reusing",
                     deviceId.toStdString(), existing.connectionId.toStdString());
            return deviceId;
        }
        // Previous connection is dead — clean up and reconnect
        removeConnection(deviceId);
    }

    QString connectionId = generateConnectionId();

    ConnectionInfo conn;
    conn.connectionId = connectionId;
    conn.deviceId = deviceId;
    conn.rtcState = RtcStatus::Connecting;
    m_connections[deviceId] = conn;
    m_connIdToDeviceId[connectionId] = deviceId;

    QJsonObject message;
    message["type"] = "connectToHost";
    message["connectionId"] = connectionId;
    message["deviceId"] = deviceId;
    message["accessCode"] = accessCode;
    message["serverUrl"] = serverUrl;
    
    // Pass runtime API key so client process can authenticate with signaling server
    QString apiKey = core::LocalConfigCenter::instance().apiKey();
    if (!apiKey.isEmpty()) {
        message["apiKey"] = apiKey;
    }
    
    QString videoCodec = core::LocalConfigCenter::instance().preferredVideoCodec();
    if (!videoCodec.isEmpty()) {
        message["preferredVideoCodec"] = videoCodec;
    }
    
    if (!m_iceConfig.isEmpty()) {
        message["iceConfig"] = m_iceConfig;
        QJsonArray servers = m_iceConfig.value("iceServers").toArray();
        LOG_INFO("Client: Sending ICE config with {} server(s), lifetime={}",
                 servers.size(),
                 m_iceConfig.value("lifetimeDuration").toString("unset").toStdString());
    } else {
        LOG_INFO("Client: No ICE config available, client will use defaults");
    }

    LOG_INFO("Connecting to host: {} connectionId: {}", deviceId.toStdString(), connectionId.toStdString());
    m_messaging->sendMessage(message);

    emit connectionCountChanged();
    emit connectionAdded(deviceId);
    emit connectionListChanged();

    if (m_activeDeviceId.isEmpty()) {
        setActiveDeviceId(deviceId);
    }

    return deviceId;
}

void ClientManager::disconnectFromHost(const QString& deviceId)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }

    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    QJsonObject message;
    message["type"] = "disconnectFromHost";
    message["connectionId"] = connId;
    m_messaging->sendMessage(message);

    if (m_connections.contains(deviceId)) {
        m_connections[deviceId].rtcState = RtcStatus::Disconnected;
        emit connectionStateChanged(deviceId, "disconnected", QJsonObject());
    }

    m_sharedMemoryManager->detach(deviceId);
    removeConnection(deviceId);

    emit connectionCountChanged();
    emit connectionRemoved(deviceId);
    emit connectionListChanged();

    if (m_activeDeviceId == deviceId) {
        if (m_connections.isEmpty()) {
            setActiveDeviceId(QString());
        } else {
            setActiveDeviceId(m_connections.firstKey());
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

    QStringList deviceIds = m_connections.keys();
    for (const auto& devId : deviceIds) {
        m_connections[devId].rtcState = RtcStatus::Disconnected;
        emit connectionStateChanged(devId, "disconnected", QJsonObject());
        m_sharedMemoryManager->detach(devId);
        emit connectionRemoved(devId);
    }
    m_connections.clear();
    m_connIdToDeviceId.clear();
    m_activeDeviceId.clear();

    emit connectionCountChanged();
    emit activeConnectionChanged();
    emit connectionListChanged();
}

void ClientManager::sendHello(const QString& deviceId,
                              const QString& preferredVideoCodec)
{
    if (!m_messaging || !m_messaging->isReady()) {
        emit errorOccurred("", "NOT_READY", "Client process is not ready");
        return;
    }

    QJsonObject message;
    message["type"] = "hello";
    if (!deviceId.isEmpty()) {
        message["deviceId"] = deviceId;
    }
    if (!preferredVideoCodec.isEmpty()) {
        message["preferredVideoCodec"] = preferredVideoCodec;
    }
    m_messaging->sendMessage(message);
}

void ClientManager::sendMouseMove(const QString& deviceId, int x, int y)
{
    sendMouseEvent(deviceId, "move", x, y, 0, 0, 0);
}

void ClientManager::sendMousePress(const QString& deviceId, int x, int y, int button)
{
    sendMouseEvent(deviceId, "press", x, y, button, 0, 0);
}

void ClientManager::sendMouseRelease(const QString& deviceId, int x, int y, int button)
{
    sendMouseEvent(deviceId, "release", x, y, button, 0, 0);
}

void ClientManager::sendMouseWheel(const QString& deviceId, int x, int y, int deltaX, int deltaY)
{
    sendMouseEvent(deviceId, "wheel", x, y, 0, deltaX, deltaY);
}

void ClientManager::sendKeyPress(const QString& deviceId, int nativeScanCode, int lockStates)
{
    sendKeyboardEvent(deviceId, "press", nativeScanCode, lockStates);
}

void ClientManager::sendKeyRelease(const QString& deviceId, int nativeScanCode, int lockStates)
{
    sendKeyboardEvent(deviceId, "release", nativeScanCode, lockStates);
}

void ClientManager::syncClipboard(const QString& deviceId, const QString& text)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    QJsonObject message;
    message["type"] = "clipboardSync";
    message["connectionId"] = connId;
    message["text"] = text;
    m_messaging->sendMessage(message);
}

void ClientManager::sendSkillCommand(const QString& deviceId,
                                     const QString& jsonData)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    QJsonObject message;
    message["type"] = "skillBridgeSend";
    message["connectionId"] = connId;
    message["data"] = jsonData;
    m_messaging->sendMessage(message);
}

void ClientManager::setTargetFramerate(const QString& deviceId, int framerate)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot set framerate: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    framerate = qBound(1, framerate, 60);
    LOG_INFO("Setting target framerate for {}: {} FPS", deviceId.toStdString(), framerate);

    QJsonObject message;
    message["type"] = "setFramerate";
    message["connectionId"] = connId;
    message["framerate"] = framerate;
    m_messaging->sendMessage(message);
}

void ClientManager::setResolution(const QString& deviceId, int width, int height, int dpi)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot set resolution: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    if (width <= 0 || height <= 0 || width > 8192 || height > 8192) {
        LOG_WARN("Invalid resolution: {}x{}", width, height);
        return;
    }

    LOG_INFO("Setting resolution for {}: {}x{} @ {} DPI", 
             deviceId.toStdString(), width, height, dpi);

    QJsonObject message;
    message["type"] = "setResolution";
    message["connectionId"] = connId;
    message["width"] = width;
    message["height"] = height;
    message["dpi"] = dpi;
    m_messaging->sendMessage(message);
}

void ClientManager::setFramerateBoost(const QString& deviceId, bool enabled, 
                                      int captureIntervalMs, int boostDurationMs)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot set framerate boost: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    captureIntervalMs = qBound(10, captureIntervalMs, 1000);
    boostDurationMs = qBound(100, boostDurationMs, 1000);

    LOG_INFO("Setting framerate boost for {}: enabled={}, interval={}ms, duration={}ms", 
             deviceId.toStdString(), enabled, captureIntervalMs, boostDurationMs);

    QJsonObject message;
    message["type"] = "setFramerateBoost";
    message["connectionId"] = connId;
    message["enabled"] = enabled;
    message["captureIntervalMs"] = captureIntervalMs;
    message["boostDurationMs"] = boostDurationMs;
    m_messaging->sendMessage(message);
}

void ClientManager::setBitrate(const QString& deviceId, int minBitrateBps)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot set bitrate: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    if (minBitrateBps < 0) {
        LOG_WARN("Invalid bitrate: {} (must be >= 0)", minBitrateBps);
        return;
    }

    LOG_INFO("Setting bitrate for {}: {} MiB ({} bps)", 
             deviceId.toStdString(), 
             minBitrateBps / 1024.0 / 1024.0, 
             minBitrateBps);

    QJsonObject message;
    message["type"] = "setBitrate";
    message["connectionId"] = connId;
    message["minBitrateBps"] = minBitrateBps;
    m_messaging->sendMessage(message);
}

void ClientManager::setAudioEnabled(const QString& deviceId, bool enabled)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot set audio: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    LOG_INFO("Setting audio enabled for {}: {}", deviceId.toStdString(), enabled);

    QJsonObject message;
    message["type"] = "setAudioEnabled";
    message["connectionId"] = connId;
    message["enabled"] = enabled;
    m_messaging->sendMessage(message);
}

void ClientManager::selectDisplay(const QString& deviceId, int displayIndex)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot select display: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    LOG_INFO("Selecting display {} for {}", displayIndex, deviceId.toStdString());

    QJsonObject message;
    message["type"] = "selectDisplay";
    message["connectionId"] = connId;
    message["displayId"] = QString::number(displayIndex);
    m_messaging->sendMessage(message);
}

void ClientManager::sendAction(const QString& deviceId, const QString& action)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot send action: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    LOG_INFO("Sending action '{}' for {}", action.toStdString(), deviceId.toStdString());

    QJsonObject message;
    message["type"] = "sendAction";
    message["connectionId"] = connId;
    message["action"] = action;
    m_messaging->sendMessage(message);
}

bool ClientManager::supportsSendAttentionSequence(const QString& deviceId) const
{
    auto it = m_connections.find(deviceId);
    if (it == m_connections.end())
        return false;
    return it.value().supportsSendAttentionSequence;
}

bool ClientManager::supportsLockWorkstation(const QString& deviceId) const
{
    auto it = m_connections.find(deviceId);
    if (it == m_connections.end())
        return false;
    return it.value().supportsLockWorkstation;
}

void ClientManager::togglePrivacyScreen(const QString& deviceId, bool enabled)
{
    sendAction(deviceId, enabled ? "enablePrivacyScreen" : "disablePrivacyScreen");
}

bool ClientManager::supportsPrivacyScreen(const QString& deviceId) const
{
    auto it = m_connections.find(deviceId);
    if (it == m_connections.end())
        return false;
    return it.value().supportsPrivacyScreen;
}

void ClientManager::createVirtualDisplay(const QString& deviceId,
                                          int width, int height, int refreshRate)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot create virtual display: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    QJsonObject data;
    data["action"] = "create";
    data["width"] = width;
    data["height"] = height;
    data["refreshRate"] = refreshRate;

    QJsonObject msg;
    msg["type"] = "virtualDisplayCommand";
    msg["connectionId"] = connId;
    msg["data"] = QString::fromUtf8(QJsonDocument(data).toJson(QJsonDocument::Compact));
    m_messaging->sendMessage(msg);
}

void ClientManager::removeVirtualDisplay(const QString& deviceId, int index)
{
    if (!m_messaging || !m_messaging->isReady()) return;
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    QJsonObject data;
    data["action"] = "remove";
    data["index"] = index;

    QJsonObject msg;
    msg["type"] = "virtualDisplayCommand";
    msg["connectionId"] = connId;
    msg["data"] = QString::fromUtf8(QJsonDocument(data).toJson(QJsonDocument::Compact));
    m_messaging->sendMessage(msg);
}

void ClientManager::removeAllVirtualDisplays(const QString& deviceId)
{
    if (!m_messaging || !m_messaging->isReady()) return;
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    QJsonObject data;
    data["action"] = "removeAll";

    QJsonObject msg;
    msg["type"] = "virtualDisplayCommand";
    msg["connectionId"] = connId;
    msg["data"] = QString::fromUtf8(QJsonDocument(data).toJson(QJsonDocument::Compact));
    m_messaging->sendMessage(msg);
}

void ClientManager::queryVirtualDisplays(const QString& deviceId)
{
    if (!m_messaging || !m_messaging->isReady()) return;
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    QJsonObject data;
    data["action"] = "query";

    QJsonObject msg;
    msg["type"] = "virtualDisplayCommand";
    msg["connectionId"] = connId;
    msg["data"] = QString::fromUtf8(QJsonDocument(data).toJson(QJsonDocument::Compact));
    m_messaging->sendMessage(msg);
}

bool ClientManager::supportsVirtualDisplay(const QString& deviceId) const
{
    auto it = m_connections.find(deviceId);
    if (it == m_connections.end())
        return false;
    return it.value().supportsVirtualDisplay;
}

void ClientManager::startFileUpload(const QString& deviceId, const QUrl& fileUrl)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot start file upload: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    QString filePath = fileUrl.toLocalFile();
    LOG_INFO("Starting file upload for {}: {}", deviceId.toStdString(),
             filePath.toStdString());

    QJsonObject message;
    message["type"] = "startFileUpload";
    message["connectionId"] = connId;
    message["filePath"] = filePath;
    m_messaging->sendMessage(message);
}

void ClientManager::cancelFileUpload(const QString& deviceId,
                                     const QString& transferId)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot cancel file upload: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    LOG_INFO("Cancelling file upload: transfer={} device={}",
             transferId.toStdString(), deviceId.toStdString());

    QJsonObject message;
    message["type"] = "cancelFileUpload";
    message["connectionId"] = connId;
    message["transferId"] = transferId;
    m_messaging->sendMessage(message);
}

void ClientManager::startFileDownload(const QString& deviceId)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot start file download: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    QString saveDir = QStandardPaths::writableLocation(QStandardPaths::DownloadLocation);
    LOG_INFO("Starting file download for {}, saveDir={}",
             deviceId.toStdString(), saveDir.toStdString());

    QJsonObject message;
    message["type"] = "startFileDownload";
    message["connectionId"] = connId;
    message["saveDir"] = saveDir;
    m_messaging->sendMessage(message);
}

void ClientManager::cancelFileDownload(const QString& deviceId,
                                       const QString& transferId)
{
    if (!m_messaging || !m_messaging->isReady()) {
        LOG_WARN("Cannot cancel file download: messaging not ready");
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    LOG_INFO("Cancelling file download: transfer={} device={}",
             transferId.toStdString(), deviceId.toStdString());

    QJsonObject message;
    message["type"] = "cancelFileDownload";
    message["connectionId"] = connId;
    message["transferId"] = transferId;
    m_messaging->sendMessage(message);
}

void ClientManager::openDownloadedFile(const QString& filePath)
{
    QDesktopServices::openUrl(QUrl::fromLocalFile(filePath));
}

void ClientManager::openContainingFolder(const QString& filePath)
{
    QFileInfo fi(filePath);
    QDesktopServices::openUrl(QUrl::fromLocalFile(fi.absolutePath()));
}

bool ClientManager::deleteDownloadedFile(const QString& filePath)
{
    return QFile::remove(filePath);
}

bool ClientManager::pasteFilesFromClipboard(const QString& deviceId)
{
    const QMimeData* mimeData = QGuiApplication::clipboard()->mimeData();
    if (!mimeData || !mimeData->hasUrls()) {
        return false;
    }

    QList<QUrl> urls = mimeData->urls();
    bool anyStarted = false;
    for (const QUrl& url : urls) {
        if (url.isLocalFile()) {
            QFileInfo fi(url.toLocalFile());
            if (fi.exists() && fi.isFile()) {
                startFileUpload(deviceId, url);
                anyStarted = true;
            }
        }
    }
    return anyStarted;
}

bool ClientManager::supportsFileTransfer(const QString& deviceId) const
{
    auto it = m_connections.find(deviceId);
    if (it == m_connections.end())
        return false;
    return it.value().supportsFileTransfer;
}

int ClientManager::connectionCount() const
{
    return m_connections.size();
}

QString ClientManager::activeDeviceId() const
{
    return m_activeDeviceId;
}

void ClientManager::setActiveDeviceId(const QString& deviceId)
{
    if (m_activeDeviceId != deviceId) {
        m_activeDeviceId = deviceId;
        emit activeConnectionChanged();
    }
}

QList<ConnectionInfo> ClientManager::connections() const
{
    return m_connections.values();
}

ConnectionInfo ClientManager::getConnection(const QString& deviceId) const
{
    return m_connections.value(deviceId);
}

QStringList ClientManager::connectedDeviceIds() const
{
    return m_connections.keys();
}

RtcStatus::Status ClientManager::getConnectionRtcState(const QString& deviceId) const
{
    if (m_connections.contains(deviceId)) {
        return m_connections[deviceId].rtcState;
    }
    return RtcStatus::Disconnected;
}

QString ClientManager::getSignalingState(const QString& deviceId) const
{
    if (m_connections.contains(deviceId)) {
        return m_connections[deviceId].signalingState;
    }
    return "disconnected";
}

int ClientManager::getSignalingRetryCount(const QString& deviceId) const
{
    if (m_connections.contains(deviceId)) {
        return m_connections[deviceId].signalingRetryCount;
    }
    return 0;
}

int ClientManager::getSignalingNextRetryIn(const QString& deviceId) const
{
    if (m_connections.contains(deviceId)) {
        return m_connections[deviceId].signalingNextRetryIn;
    }
    return 0;
}

QString ClientManager::getSignalingError(const QString& deviceId) const
{
    if (m_connections.contains(deviceId)) {
        return m_connections[deviceId].signalingError;
    }
    return QString();
}

// --- Message handling (internal connectionId → external deviceId) ---

void ClientManager::onMessageReceived(const QJsonObject& message)
{
    QString type = message["type"].toString();

    if (type != "videoFrameReady" && type != "performanceStatsUpdate" && type != "displayListChanged" && type != "routeChanged") {
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
    } else if (type == "displayListChanged") {
        handleDisplayListChanged(message);
    } else if (type == "routeChanged") {
        handleRouteChanged(message);
    } else if (type == "hostCapabilities") {
        handleHostCapabilities(message);
    } else if (type == "fileTransferProgress") {
        handleFileTransferProgress(message);
    } else if (type == "fileTransferComplete") {
        handleFileTransferComplete(message);
    } else if (type == "fileTransferError") {
        handleFileTransferError(message);
    } else if (type == "fileDownloadStarted") {
        handleFileDownloadStarted(message);
    } else if (type == "fileDownloadProgress") {
        handleFileDownloadProgress(message);
    } else if (type == "fileDownloadComplete") {
        handleFileDownloadComplete(message);
    } else if (type == "fileDownloadError") {
        handleFileDownloadError(message);
    } else if (type == "skillBridgeResponse") {
        handleSkillBridgeResponse(message);
    } else if (type == "virtualDisplayState") {
        handleVirtualDisplayState(message);
    } else if (type == "setFramerateResponse" || type == "setResolutionResponse" || type == "setFramerateBoostResponse" || type == "setBitrateResponse") {
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

// --- Private helpers ---

QString ClientManager::connectionIdFor(const QString& deviceId) const
{
    auto it = m_connections.find(deviceId);
    if (it != m_connections.end())
        return it->connectionId;
    return {};
}

QString ClientManager::findDeviceId(const QString& connectionId) const
{
    return m_connIdToDeviceId.value(connectionId);
}

QString ClientManager::generateConnectionId()
{
    return QString("conn_%1").arg(++m_connectionCounter);
}

void ClientManager::removeConnection(const QString& deviceId)
{
    auto it = m_connections.find(deviceId);
    if (it != m_connections.end()) {
        m_connIdToDeviceId.remove(it->connectionId);
        m_connections.erase(it);
    }
}

// --- Message handlers ---

void ClientManager::handleHelloResponse(const QJsonObject& message)
{
    QString version = message["version"].toString();
    LOG_INFO("Client hello response, version: {}", version.toStdString());
    emit helloResponseReceived(version);
}

void ClientManager::handleSignalingStateChanged(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) {
        LOG_WARN("Signaling state changed for unknown connection: {}", connId.toStdString());
        return;
    }

    QString state = message["state"].toString();
    int retryCount = message["retryCount"].toInt();
    int nextRetryIn = message["nextRetryIn"].toInt();
    QString error = message["error"].toString();

    LOG_INFO("Client signaling state changed: device={}, state={}, retry={}, next={}s, error={}",
             deviceId.toStdString(), state.toStdString(), retryCount, nextRetryIn, error.toStdString());

    m_connections[deviceId].signalingState = state;
    m_connections[deviceId].signalingRetryCount = retryCount;
    m_connections[deviceId].signalingNextRetryIn = nextRetryIn;
    m_connections[deviceId].signalingError = error;
    
    emit signalingStateChanged(deviceId, state, retryCount, nextRetryIn, error);
}

void ClientManager::handleConnectToHostResponse(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    if (connId.isEmpty()) {
        LOG_WARN("connectToHostResponse missing connectionId");
        return;
    }

    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) {
        // Unexpected response — we don't know which device this belongs to
        LOG_WARN("connectToHostResponse for unknown connectionId: {}", connId.toStdString());
        return;
    }
    // Connection already tracked from connectToHost()
}

void ClientManager::handleConnectionStateChanged(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString stateStr = message["state"].toString();
    QJsonObject hostInfo = message["hostInfo"].toObject();

    if (stateStr == "connecting") {
        m_connections[deviceId].rtcState = RtcStatus::Connecting;
    } else if (stateStr == "connected") {
        m_connections[deviceId].rtcState = RtcStatus::Connected;
    } else if (stateStr == "disconnected") {
        m_connections[deviceId].rtcState = RtcStatus::Disconnected;
    } else if (stateStr == "failed") {
        m_connections[deviceId].rtcState = RtcStatus::Failed;
    }
    
    if (hostInfo.contains("resolution")) {
        QString resolution = hostInfo["resolution"].toString();
        QStringList parts = resolution.split('x');
        if (parts.size() == 2) {
            m_connections[deviceId].width = parts[0].toInt();
            m_connections[deviceId].height = parts[1].toInt();
        }
    }
    if (hostInfo.contains("deviceName")) {
        m_connections[deviceId].deviceName = hostInfo["deviceName"].toString();
    }

    LOG_INFO("Connection {} (device {}) RTC state changed to: {}", connId.toStdString(), deviceId.toStdString(), stateStr.toStdString());
    emit connectionStateChanged(deviceId, stateStr, hostInfo);
    emit connectionListChanged();
}

void ClientManager::handleConnectionListChanged(const QJsonObject& message)
{
    m_connections.clear();
    m_connIdToDeviceId.clear();
    
    QJsonArray connections = message["connections"].toArray();
    for (const QJsonValue& value : connections) {
        QJsonObject obj = value.toObject();
        ConnectionInfo conn;
        conn.connectionId = obj["connectionId"].toString();
        conn.deviceId = obj["deviceId"].toString();
        conn.deviceName = obj["deviceName"].toString();
        conn.connectedAt = obj["connectedAt"].toString();
        
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
        
        m_connections[conn.deviceId] = conn;
        m_connIdToDeviceId[conn.connectionId] = conn.deviceId;
    }

    emit connectionCountChanged();
    emit connectionListChanged();
}

void ClientManager::handleVideoFrameReady(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    int frameIndex = message["frameIndex"].toInt();
    int width = message["width"].toInt();
    int height = message["height"].toInt();
    QString sharedMemoryName = message["sharedMemoryName"].toString();
    
    if (!m_sharedMemoryManager->isAttached(deviceId)) {
        if (!m_sharedMemoryManager->attach(deviceId, sharedMemoryName)) {
            LOG_WARN("Failed to attach to shared memory for device {}", 
                     deviceId.toStdString());
            return;
        }
        LOG_INFO("Attached to shared memory: {} ({}x{})", 
                 sharedMemoryName.toStdString(), width, height);
    }
    
    if (m_connections.contains(deviceId)) {
        m_connections[deviceId].width = width;
        m_connections[deviceId].height = height;
    }
    
    emit videoFrameReady(deviceId, frameIndex);
}

void ClientManager::handleClipboardReceived(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString text = message["text"].toString();
    emit clipboardReceived(deviceId, text);
}

void ClientManager::handleError(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    QString code = message["code"].toString();
    QString errorMsg = message["message"].toString();
    
    LOG_WARN("Client error: device={} {} {}", deviceId.toStdString(), code.toStdString(), errorMsg.toStdString());
    emit errorOccurred(deviceId, code, errorMsg);
}

void ClientManager::handleConnectionFailed(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString errorCode = message["errorCode"].toString();
    QString errorMsg = message["message"].toString();
    
    qWarning() << "Connection failed: device" << deviceId
               << "error:" << errorCode << "-" << errorMsg;
    
    if (m_connections.contains(deviceId)) {
        m_connections[deviceId].rtcState = RtcStatus::Failed;
        emit connectionStateChanged(deviceId, "failed", QJsonObject());
    }
    
    emit errorOccurred(deviceId, errorCode, errorMsg);
    
    removeConnection(deviceId);
    emit connectionCountChanged();
    emit connectionRemoved(deviceId);
    emit connectionListChanged();
    
    if (m_activeDeviceId == deviceId) {
        if (m_connections.isEmpty()) {
            setActiveDeviceId(QString());
        } else {
            setActiveDeviceId(m_connections.firstKey());
        }
    }
}

void ClientManager::handleHostConnected(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    LOG_INFO("Host connected: device={}", deviceId.toStdString());
    
    if (m_connections.contains(deviceId)) {
        m_connections[deviceId].rtcState = RtcStatus::Connected;
        emit connectionStateChanged(deviceId, "connected", QJsonObject());
    }
    emit connectionListChanged();
}

void ClientManager::handleHostDisconnected(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    LOG_INFO("Host disconnected: device={}", deviceId.toStdString());
    
    if (m_connections.contains(deviceId)) {
        m_connections[deviceId].rtcState = RtcStatus::Disconnected;
        emit connectionStateChanged(deviceId, "disconnected", QJsonObject());
    }
    
    m_sharedMemoryManager->detach(deviceId);
    
    removeConnection(deviceId);
    emit connectionCountChanged();
    emit connectionRemoved(deviceId);
    emit connectionListChanged();
    
    if (m_activeDeviceId == deviceId) {
        if (m_connections.isEmpty()) {
            setActiveDeviceId(QString());
        } else {
            setActiveDeviceId(m_connections.firstKey());
        }
    }
}

void ClientManager::handleHostConnectionFailed(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    int errorCode = message["errorCode"].toInt();
    
    LOG_WARN("Host connection failed: device={} error code: {}", deviceId.toStdString(), errorCode);
    
    if (m_connections.contains(deviceId)) {
        m_connections[deviceId].rtcState = RtcStatus::Failed;
        emit connectionStateChanged(deviceId, "failed", QJsonObject());
    }
    
    QString errorMsg;
    switch (errorCode) {
        case 1: errorMsg = tr("Authentication failed"); break;
        case 2: errorMsg = tr("Channel error"); break;
        case 3: errorMsg = tr("Connection timeout"); break;
        case 4: errorMsg = tr("Network error"); break;
        default: errorMsg = tr("Connection failed (error code: %1)").arg(errorCode); break;
    }
    
    emit errorOccurred(deviceId, "CONNECTION_FAILED", errorMsg);
    
    removeConnection(deviceId);
    emit connectionCountChanged();
    emit connectionRemoved(deviceId);
    emit connectionListChanged();
    
    if (m_activeDeviceId == deviceId) {
        if (m_connections.isEmpty()) {
            setActiveDeviceId(QString());
        } else {
            setActiveDeviceId(m_connections.firstKey());
        }
    }
}

void ClientManager::handleDisconnectFromHostResponse(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    if (!connId.isEmpty()) {
        LOG_INFO("Disconnect response received for connection: {}", connId.toStdString());
    }
}

void ClientManager::handleDisconnectAllResponse(const QJsonObject& message)
{
    Q_UNUSED(message);
    LOG_INFO("Disconnect all response received");
}

void ClientManager::handleCursorShapeChanged(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    int width = message["width"].toInt();
    int height = message["height"].toInt();
    int hotspotX = message["hotspotX"].toInt();
    int hotspotY = message["hotspotY"].toInt();
    QString base64Data = message["data"].toString();
    
    QByteArray data = QByteArray::fromBase64(base64Data.toLatin1());
    
    emit cursorShapeChanged(deviceId, width, height, hotspotX, hotspotY, data);
}

void ClientManager::handlePerformanceStatsUpdate(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QVariantMap stats;
    stats["captureMs"]      = message["captureMs"].toDouble();
    stats["encodeMs"]       = message["encodeMs"].toDouble();
    stats["networkDelayMs"] = message["networkDelayMs"].toDouble();
    stats["decodeMs"]       = message["decodeMs"].toDouble();
    stats["paintMs"]        = message["paintMs"].toDouble();
    stats["totalLatencyMs"] = message["totalLatencyMs"].toDouble();
    stats["inputRoundtripMs"] = message["inputRoundtripMs"].toDouble();
    stats["bandwidthKbps"]  = message["bandwidthKbps"].toDouble();
    stats["frameRate"]      = message["frameRate"].toDouble();
    stats["packetRate"]     = message["packetRate"].toDouble();
    stats["codec"]              = message["codec"].toString("Unknown");
    stats["frameQuality"]       = message["frameQuality"].toInt(-1);
    stats["encodedRectWidth"]   = message["encodedRectWidth"].toInt();
    stats["encodedRectHeight"]  = message["encodedRectHeight"].toInt();
    
    emit performanceStatsUpdated(deviceId, stats);
}

void ClientManager::handleDisplayListChanged(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QJsonArray displays = message["displays"].toArray();
    int activeDisplayIndex = message["activeDisplayIndex"].toInt(0);

    LOG_DEBUG("DisplayList changed: device={}, displays={}, active={}",
              deviceId.toStdString(), displays.size(), activeDisplayIndex);

    // Emit the new displayListChanged signal with full display info.
    emit displayListChanged(deviceId, displays, activeDisplayIndex);

    // Also emit the legacy videoLayoutChanged for backward compatibility.
    // Use the active display's dimensions.
    if (activeDisplayIndex >= 0 && activeDisplayIndex < displays.size()) {
        QJsonObject activeDisplay = displays[activeDisplayIndex].toObject();
        int widthDips = activeDisplay["width"].toInt();
        int heightDips = activeDisplay["height"].toInt();
        emit videoLayoutChanged(deviceId, widthDips, heightDips);
    }
}

void ClientManager::handleRouteChanged(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QVariantMap routeInfo;
    routeInfo["routeType"] = message["routeType"].toString();
    routeInfo["transportProtocol"] = message["transportProtocol"].toString();
    routeInfo["localCandidateType"] = message["localCandidateType"].toString();
    routeInfo["remoteCandidateType"] = message["remoteCandidateType"].toString();
    routeInfo["localAddress"] = message["localAddress"].toString();
    routeInfo["remoteAddress"] = message["remoteAddress"].toString();

    QVariantList localCandidates;
    for (const auto& val : message["localCandidates"].toArray()) {
        QJsonObject obj = val.toObject();
        QVariantMap c;
        c["address"] = obj["address"].toString();
        c["type"] = obj["type"].toString();
        c["protocol"] = obj["protocol"].toString();
        c["isIpv6"] = obj["isIpv6"].toBool();
        c["priority"] = obj["priority"].toInt();
        localCandidates.append(c);
    }
    routeInfo["localCandidates"] = localCandidates;

    QVariantList remoteCandidates;
    for (const auto& val : message["remoteCandidates"].toArray()) {
        QJsonObject obj = val.toObject();
        QVariantMap c;
        c["address"] = obj["address"].toString();
        c["type"] = obj["type"].toString();
        c["protocol"] = obj["protocol"].toString();
        c["isIpv6"] = obj["isIpv6"].toBool();
        c["priority"] = obj["priority"].toInt();
        remoteCandidates.append(c);
    }
    routeInfo["remoteCandidates"] = remoteCandidates;

    LOG_INFO("Route changed: device={}, type={}, local={}, remote={}",
             deviceId.toStdString(),
             routeInfo["routeType"].toString().toStdString(),
             routeInfo["localCandidateType"].toString().toStdString(),
             routeInfo["remoteCandidateType"].toString().toStdString());

    emit routeChanged(deviceId, routeInfo);
}

void ClientManager::sendMouseEvent(const QString& deviceId, const QString& eventType,
                                   int x, int y, int button,
                                   int wheelDeltaX, int wheelDeltaY)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    QJsonObject message;
    message["type"] = "mouseEvent";
    message["connectionId"] = connId;
    message["eventType"] = eventType;
    message["x"] = x;
    message["y"] = y;
    message["button"] = button;
    message["wheelDeltaX"] = wheelDeltaX;
    message["wheelDeltaY"] = wheelDeltaY;
    m_messaging->sendMessage(message);
}

void ClientManager::sendKeyboardEvent(const QString& deviceId, const QString& eventType,
                                      int nativeScanCode, int lockStates)
{
    if (!m_messaging || !m_messaging->isReady()) {
        return;
    }
    QString connId = connectionIdFor(deviceId);
    if (connId.isEmpty()) return;

    QJsonObject message;
    message["type"] = "keyboardEvent";
    message["connectionId"] = connId;
    message["eventType"] = eventType;
    message["nativeScanCode"] = nativeScanCode;
    message["lockStates"] = lockStates;
    m_messaging->sendMessage(message);
}

bool ClientManager::saveFrameToFile(const QString& deviceId, 
                                     const QString& filePath)
{
    if (!m_sharedMemoryManager->isAttached(deviceId)) {
        LOG_WARN("Cannot save frame: not attached to shared memory for {}", 
                 deviceId.toStdString());
        return false;
    }
    
    QVideoFrame videoFrame = m_sharedMemoryManager->readVideoFrame(deviceId);
    if (!videoFrame.isValid()) {
        LOG_WARN("Cannot save frame: failed to read video frame for {}", 
                 deviceId.toStdString());
        return false;
    }
    
    QImage frame = videoFrame.toImage();
    if (frame.isNull()) {
        LOG_WARN("Cannot save frame: failed to convert video frame to image for {}", 
                 deviceId.toStdString());
        return false;
    }
    
    QFileInfo fileInfo(filePath);
    QDir dir = fileInfo.dir();
    if (!dir.exists()) {
        dir.mkpath(".");
    }
    
    bool success = frame.save(filePath);
    if (success) {
        LOG_INFO("Saved frame to: {} ({}x{})", 
                 filePath.toStdString(), frame.width(), frame.height());
    } else {
        LOG_WARN("Failed to save frame to: {}", filePath.toStdString());
    }
    
    return success;
}

void ClientManager::setIceConfig(const QJsonObject& iceConfig)
{
    m_iceConfig = iceConfig;
    QJsonArray servers = m_iceConfig.value("iceServers").toArray();
    LOG_INFO("Client: ICE config updated: {} server(s)", servers.size());
}

QJsonObject ClientManager::getIceConfig() const
{
    return m_iceConfig;
}

void ClientManager::handleHostCapabilities(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    bool supportsSAS = message["supportsSendAttentionSequence"].toBool();
    bool supportsLock = message["supportsLockWorkstation"].toBool();
    bool supportsFile = message["supportsFileTransfer"].toBool();
    bool supportsPrivacy = message["supportsPrivacyScreen"].toBool();
    bool supportsVD = message["supportsVirtualDisplay"].toBool();

    auto it = m_connections.find(deviceId);
    if (it != m_connections.end()) {
        it->supportsSendAttentionSequence = supportsSAS;
        it->supportsLockWorkstation = supportsLock;
        it->supportsFileTransfer = supportsFile;
        it->supportsPrivacyScreen = supportsPrivacy;
        it->supportsVirtualDisplay = supportsVD;
    }

    LOG_INFO("Host capabilities for {}: SAS={} Lock={} FileTransfer={} PrivacyScreen={} VirtualDisplay={}",
             deviceId.toStdString(), supportsSAS, supportsLock, supportsFile, supportsPrivacy, supportsVD);

    emit hostCapabilitiesChanged(deviceId, supportsSAS, supportsLock, supportsFile, supportsPrivacy, supportsVD);
}

void ClientManager::handleFileTransferProgress(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString transferId = message["transferId"].toString();
    QString filename = message["filename"].toString();
    double bytesSent = message["bytesSent"].toDouble();
    double totalBytes = message["totalBytes"].toDouble();

    emit fileTransferProgress(deviceId, transferId, filename, bytesSent, totalBytes);
}

void ClientManager::handleFileTransferComplete(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString transferId = message["transferId"].toString();
    QString filename = message["filename"].toString();

    LOG_INFO("File transfer complete: {} (transfer={})",
             filename.toStdString(), transferId.toStdString());

    emit fileTransferComplete(deviceId, transferId, filename);
}

void ClientManager::handleFileTransferError(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString transferId = message["transferId"].toString();
    QString errorMessage = message["errorMessage"].toString();

    LOG_ERROR("File transfer error: {} (transfer={})",
              errorMessage.toStdString(), transferId.toStdString());

    emit fileTransferError(deviceId, transferId, errorMessage);
}

void ClientManager::handleFileDownloadStarted(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString transferId = message["transferId"].toString();
    QString filename = message["filename"].toString();
    double totalBytes = message["totalBytes"].toDouble();

    LOG_INFO("File download started: {} ({} bytes, transfer={})",
             filename.toStdString(), static_cast<uint64_t>(totalBytes),
             transferId.toStdString());

    emit fileDownloadStarted(deviceId, transferId, filename, totalBytes);
}

void ClientManager::handleFileDownloadProgress(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString transferId = message["transferId"].toString();
    QString filename = message["filename"].toString();
    double bytesReceived = message["bytesReceived"].toDouble();
    double totalBytes = message["totalBytes"].toDouble();

    emit fileDownloadProgress(deviceId, transferId, filename, bytesReceived, totalBytes);
}

void ClientManager::handleFileDownloadComplete(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString transferId = message["transferId"].toString();
    QString filename = message["filename"].toString();
    QString savePath = message["savePath"].toString();

    LOG_INFO("File download complete: {} -> {} (transfer={})",
             filename.toStdString(), savePath.toStdString(),
             transferId.toStdString());

    emit fileDownloadComplete(deviceId, transferId, filename, savePath);
}

void ClientManager::handleFileDownloadError(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString transferId = message["transferId"].toString();
    QString errorMessage = message["errorMessage"].toString();

    LOG_ERROR("File download error: {} (transfer={})",
              errorMessage.toStdString(), transferId.toStdString());

    emit fileDownloadError(deviceId, transferId, errorMessage);
}

void ClientManager::handleSkillBridgeResponse(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString data = message["data"].toString();
    QJsonDocument doc = QJsonDocument::fromJson(data.toUtf8());
    QJsonObject response = doc.isObject() ? doc.object() : QJsonObject{{"raw", data}};

    emit skillBridgeResponseReceived(deviceId, response);
}

void ClientManager::handleVirtualDisplayState(const QJsonObject& message)
{
    QString connId = message["connectionId"].toString();
    QString deviceId = findDeviceId(connId);
    if (deviceId.isEmpty()) return;

    QString data = message["data"].toString();
    QJsonDocument doc = QJsonDocument::fromJson(data.toUtf8());
    QJsonObject state = doc.isObject() ? doc.object() : QJsonObject{{"raw", data}};

    LOG_INFO("Virtual display state for {}: {}", deviceId.toStdString(),
             data.toStdString());

    emit virtualDisplayStateChanged(deviceId, state);
}

} // namespace quickdesk
