// Copyright 2026 QuickDesk Authors
// Client process communication manager

#ifndef QUICKDESK_MANAGER_CLIENTMANAGER_H
#define QUICKDESK_MANAGER_CLIENTMANAGER_H

#include <QObject>
#include <QJsonObject>
#include <QJsonArray>
#include <QList>
#include <QMap>
#include <QStringList>
#include <QVariantMap>
#include <memory>

#include "SharedMemoryManager.h"
#include "common/ProcessStatus.h"

namespace quickdesk {

class NativeMessaging;

/**
 * @brief Connection information for remote hosts
 */
struct ConnectionInfo {
    QString connectionId;
    QString deviceId;
    QString deviceName;
    RtcStatus::Status rtcState = RtcStatus::Disconnected;  // WebRTC P2P connection state
    QString connectedAt;
    int width = 0;
    int height = 0;
    
    // Signaling state for this connection
    QString signalingState = "disconnected";
    int signalingRetryCount = 0;
    int signalingNextRetryIn = 0;
    QString signalingError;
};

/**
 * @brief Manages communication with the Client process
 */
class ClientManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(int connectionCount READ connectionCount NOTIFY connectionCountChanged)
    Q_PROPERTY(QString activeConnectionId READ activeConnectionId 
               WRITE setActiveConnectionId NOTIFY activeConnectionChanged)
    Q_PROPERTY(QStringList connectionIds READ connectionIds NOTIFY connectionListChanged)
    Q_PROPERTY(SharedMemoryManager* sharedMemoryManager READ sharedMemoryManager CONSTANT)

public:
    explicit ClientManager(QObject* parent = nullptr);
    ~ClientManager() override = default;

    // Set the Native Messaging handler
    void setMessaging(NativeMessaging* messaging);
    
    // ICE server configuration
    void setIceServers(const QJsonArray& iceServers);
    QJsonArray getIceServers() const;

    // Connection management
    Q_INVOKABLE QString connectToHost(const QString& deviceId,
                                      const QString& accessCode,
                                      const QString& serverUrl);
    Q_INVOKABLE void disconnectFromHost(const QString& connectionId);
    Q_INVOKABLE void disconnectAll();
    Q_INVOKABLE void sendHello(const QString& deviceId = QString());

    // Input events
    Q_INVOKABLE void sendMouseMove(const QString& connectionId, int x, int y);
    Q_INVOKABLE void sendMousePress(const QString& connectionId, int x, int y, int button);
    Q_INVOKABLE void sendMouseRelease(const QString& connectionId, int x, int y, int button);
    Q_INVOKABLE void sendMouseWheel(const QString& connectionId, int x, int y, int delta);
    Q_INVOKABLE void sendKeyPress(const QString& connectionId, int keyCode, int modifiers);
    Q_INVOKABLE void sendKeyRelease(const QString& connectionId, int keyCode, int modifiers);

    // Clipboard
    Q_INVOKABLE void syncClipboard(const QString& connectionId, const QString& text);

    // Video control
    Q_INVOKABLE void setTargetFramerate(const QString& connectionId, int framerate);
    Q_INVOKABLE void setResolution(const QString& connectionId, int width, int height, int dpi = 96);
    Q_INVOKABLE void setFramerateBoost(const QString& connectionId, bool enabled, 
                                       int captureIntervalMs = 30, int boostDurationMs = 300);
    Q_INVOKABLE void setBitrate(const QString& connectionId, int minBitrateBps);

    // State getters
    int connectionCount() const;
    QString activeConnectionId() const;
    void setActiveConnectionId(const QString& id);
    QList<ConnectionInfo> connections() const;
    ConnectionInfo getConnection(const QString& connectionId) const;
    QStringList connectionIds() const;
    Q_INVOKABLE RtcStatus::Status getConnectionRtcState(const QString& connectionId) const;
    
    // Get signaling state for a specific connection
    Q_INVOKABLE QString getSignalingState(const QString& connectionId) const;
    Q_INVOKABLE int getSignalingRetryCount(const QString& connectionId) const;
    Q_INVOKABLE int getSignalingNextRetryIn(const QString& connectionId) const;
    Q_INVOKABLE QString getSignalingError(const QString& connectionId) const;
    
    // Get device ID for a connection
    Q_INVOKABLE QString getConnectionDeviceId(const QString& connectionId) const;

    // Shared memory access
    SharedMemoryManager* sharedMemoryManager() const { return m_sharedMemoryManager.get(); }

    // Frame testing - save current frame to file (for debugging)
    Q_INVOKABLE bool saveFrameToFile(const QString& connectionId, 
                                     const QString& filePath);

signals:
    void connectionCountChanged();
    void activeConnectionChanged();
    
    void helloResponseReceived(const QString& version);
    
    // Signaling state changed for a specific connection
    void signalingStateChanged(const QString& connectionId,
                               const QString& state,
                               int retryCount,
                               int nextRetryIn,
                               const QString& error);
    
    void connectionStateChanged(const QString& connectionId, 
                                const QString& state,
                                const QJsonObject& hostInfo);
    void connectionAdded(const QString& connectionId);
    void connectionRemoved(const QString& connectionId);
    void connectionListChanged();
    void videoFrameReady(const QString& connectionId, int frameIndex);
    void clipboardReceived(const QString& connectionId, const QString& text);
    void errorOccurred(const QString& connectionId, 
                       const QString& code, 
                       const QString& message);
    void cursorShapeChanged(const QString& connectionId, 
                            int width, int height,
                            int hotspotX, int hotspotY,
                            const QByteArray& data);
    
    // Detailed performance statistics update (QVariantMap keys match JSON field names)
    void performanceStatsUpdated(const QString& connectionId,
                                 const QVariantMap& stats);

private slots:
    void onMessageReceived(const QJsonObject& message);
    void onMessagingError(const QString& error);

private:
    NativeMessaging* m_messaging = nullptr;
    std::unique_ptr<SharedMemoryManager> m_sharedMemoryManager;
    QMap<QString, ConnectionInfo> m_connections;  // Each connection has its own signaling state
    QString m_activeConnectionId;
    int m_connectionCounter = 0;
    
    // ICE server configuration
    QJsonArray m_iceServers;

    QString generateConnectionId();
    void handleHelloResponse(const QJsonObject& message);
    void handleSignalingStateChanged(const QJsonObject& message);
    void handleConnectToHostResponse(const QJsonObject& message);
    void handleConnectionStateChanged(const QJsonObject& message);
    void handleConnectionListChanged(const QJsonObject& message);
    void handleVideoFrameReady(const QJsonObject& message);
    void handleClipboardReceived(const QJsonObject& message);
    void handleError(const QJsonObject& message);
    void handleConnectionFailed(const QJsonObject& message);
    void handleHostConnected(const QJsonObject& message);
    void handleHostDisconnected(const QJsonObject& message);
    void handleHostConnectionFailed(const QJsonObject& message);
    void handleDisconnectFromHostResponse(const QJsonObject& message);
    void handleDisconnectAllResponse(const QJsonObject& message);
    void handleCursorShapeChanged(const QJsonObject& message);
    void handlePerformanceStatsUpdate(const QJsonObject& message);
    
    void sendMouseEvent(const QString& connectionId, const QString& eventType,
                        int x, int y, int button, int wheelDelta);
    void sendKeyboardEvent(const QString& connectionId, const QString& eventType,
                           int keyCode, int modifiers);
};

} // namespace quickdesk

#endif // QUICKDESK_MANAGER_CLIENTMANAGER_H
