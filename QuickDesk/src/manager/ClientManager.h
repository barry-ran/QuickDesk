// Copyright 2026 QuickDesk Authors
// Client process communication manager

#ifndef QUICKDESK_MANAGER_CLIENTMANAGER_H
#define QUICKDESK_MANAGER_CLIENTMANAGER_H

#include <memory>

#include <QObject>
#include <QPointer>
#include <QJsonObject>
#include <QJsonArray>
#include <QHash>
#include <QList>
#include <QMap>
#include <QStringList>
#include <QUrl>
#include <QVariantMap>

#include "SharedMemoryManager.h"
#include "common/ProcessStatus.h"

namespace quickdesk {

class NativeMessaging;

/**
 * @brief Connection information for remote hosts.
 *
 * Keyed by deviceId externally. The connectionId field is an internal
 * implementation detail used only for communication with the native
 * client process via NativeMessaging.
 */
struct ConnectionInfo {
    QString connectionId;   // internal only — for NativeMessaging routing
    QString deviceId;
    QString deviceName;
    RtcStatus::Status rtcState = RtcStatus::Disconnected;
    QString connectedAt;
    int width = 0;
    int height = 0;
    
    QString signalingState = "disconnected";
    int signalingRetryCount = 0;
    int signalingNextRetryIn = 0;
    QString signalingError;

    bool supportsSendAttentionSequence = false;
    bool supportsLockWorkstation = false;
    bool supportsFileTransfer = false;
    bool supportsPrivacyScreen = false;
    bool supportsVirtualDisplay = false;
};

/**
 * @brief Manages communication with the Client process.
 *
 * All public APIs identify remote sessions by deviceId (the stable
 * 9-digit device identifier). The internal connectionId (e.g. "conn_1")
 * is used only inside this class for NativeMessaging routing and is
 * never exposed to callers.
 */
class ClientManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(int connectionCount READ connectionCount NOTIFY connectionCountChanged)
    Q_PROPERTY(QString activeDeviceId READ activeDeviceId
               WRITE setActiveDeviceId NOTIFY activeConnectionChanged)
    Q_PROPERTY(QStringList connectedDeviceIds READ connectedDeviceIds NOTIFY connectionListChanged)
    Q_PROPERTY(SharedMemoryManager* sharedMemoryManager READ sharedMemoryManager CONSTANT)

public:
    explicit ClientManager(QObject* parent = nullptr);
    ~ClientManager() override = default;

    void setMessaging(NativeMessaging* messaging);
    
    void setIceConfig(const QJsonObject& iceConfig);
    QJsonObject getIceConfig() const;

    // Connection management — all identified by deviceId
    Q_INVOKABLE QString connectToHost(const QString& deviceId,
                                      const QString& accessCode,
                                      const QString& serverUrl);
    Q_INVOKABLE void disconnectFromHost(const QString& deviceId);
    Q_INVOKABLE void disconnectAll();
    Q_INVOKABLE void sendHello(const QString& deviceId = QString(),
                               const QString& preferredVideoCodec = QString());

    // Input events
    Q_INVOKABLE void sendMouseMove(const QString& deviceId, int x, int y);
    Q_INVOKABLE void sendMousePress(const QString& deviceId, int x, int y, int button);
    Q_INVOKABLE void sendMouseRelease(const QString& deviceId, int x, int y, int button);
    Q_INVOKABLE void sendMouseWheel(const QString& deviceId, int x, int y, int deltaX, int deltaY);
    Q_INVOKABLE void sendKeyPress(const QString& deviceId, int nativeScanCode, int lockStates);
    Q_INVOKABLE void sendKeyRelease(const QString& deviceId, int nativeScanCode, int lockStates);

    // Clipboard
    Q_INVOKABLE void syncClipboard(const QString& deviceId, const QString& text);

    // Skill bridge
    Q_INVOKABLE void sendSkillCommand(const QString& deviceId,
                                      const QString& jsonData);

    // Video control
    Q_INVOKABLE void setTargetFramerate(const QString& deviceId, int framerate);
    Q_INVOKABLE void setResolution(const QString& deviceId, int width, int height, int dpi = 96);
    Q_INVOKABLE void setFramerateBoost(const QString& deviceId, bool enabled, 
                                       int captureIntervalMs = 30, int boostDurationMs = 300);
    Q_INVOKABLE void setBitrate(const QString& deviceId, int minBitrateBps);

    // Audio control
    Q_INVOKABLE void setAudioEnabled(const QString& deviceId, bool enabled);

    // Display selection (multi-monitor)
    Q_INVOKABLE void selectDisplay(const QString& deviceId, int displayIndex);

    // Remote actions (Ctrl+Alt+Del, Lock Screen)
    Q_INVOKABLE void sendAction(const QString& deviceId, const QString& action);
    Q_INVOKABLE bool supportsSendAttentionSequence(const QString& deviceId) const;
    Q_INVOKABLE bool supportsLockWorkstation(const QString& deviceId) const;

    // Privacy screen
    Q_INVOKABLE void togglePrivacyScreen(const QString& deviceId, bool enabled);
    Q_INVOKABLE bool supportsPrivacyScreen(const QString& deviceId) const;

    // Virtual display
    Q_INVOKABLE void createVirtualDisplay(const QString& deviceId,
                                           int width, int height, int refreshRate);
    Q_INVOKABLE void removeVirtualDisplay(const QString& deviceId, int index);
    Q_INVOKABLE void removeAllVirtualDisplays(const QString& deviceId);
    Q_INVOKABLE void queryVirtualDisplays(const QString& deviceId);
    Q_INVOKABLE bool supportsVirtualDisplay(const QString& deviceId) const;

    // File transfer (Client -> Host upload)
    Q_INVOKABLE void startFileUpload(const QString& deviceId, const QUrl& fileUrl);
    Q_INVOKABLE void cancelFileUpload(const QString& deviceId, const QString& transferId);
    Q_INVOKABLE bool supportsFileTransfer(const QString& deviceId) const;

    // File download (Host -> Client)
    Q_INVOKABLE void startFileDownload(const QString& deviceId);
    Q_INVOKABLE void cancelFileDownload(const QString& deviceId, const QString& transferId);

    // Downloaded file operations
    Q_INVOKABLE void openDownloadedFile(const QString& filePath);
    Q_INVOKABLE void openContainingFolder(const QString& filePath);
    Q_INVOKABLE bool deleteDownloadedFile(const QString& filePath);

    // Clipboard file paste
    Q_INVOKABLE bool pasteFilesFromClipboard(const QString& deviceId);

    // State getters
    int connectionCount() const;
    QString activeDeviceId() const;
    void setActiveDeviceId(const QString& deviceId);
    QList<ConnectionInfo> connections() const;
    ConnectionInfo getConnection(const QString& deviceId) const;
    QStringList connectedDeviceIds() const;
    Q_INVOKABLE RtcStatus::Status getConnectionRtcState(const QString& deviceId) const;
    
    Q_INVOKABLE QString getSignalingState(const QString& deviceId) const;
    Q_INVOKABLE int getSignalingRetryCount(const QString& deviceId) const;
    Q_INVOKABLE int getSignalingNextRetryIn(const QString& deviceId) const;
    Q_INVOKABLE QString getSignalingError(const QString& deviceId) const;

    SharedMemoryManager* sharedMemoryManager() const { return m_sharedMemoryManager.get(); }

    Q_INVOKABLE bool saveFrameToFile(const QString& deviceId, 
                                     const QString& filePath);

signals:
    void connectionCountChanged();
    void activeConnectionChanged();
    
    void helloResponseReceived(const QString& version);
    
    void signalingStateChanged(const QString& deviceId,
                               const QString& state,
                               int retryCount,
                               int nextRetryIn,
                               const QString& error);
    
    void connectionStateChanged(const QString& deviceId, 
                                const QString& state,
                                const QJsonObject& hostInfo);
    void connectionAdded(const QString& deviceId);
    void connectionRemoved(const QString& deviceId);
    void connectionListChanged();
    void videoFrameReady(const QString& deviceId, int frameIndex);
    void clipboardReceived(const QString& deviceId, const QString& text);

    void skillBridgeResponseReceived(const QString& deviceId,
                                     const QJsonObject& response);
    void errorOccurred(const QString& deviceId, 
                       const QString& code, 
                       const QString& message);
    void cursorShapeChanged(const QString& deviceId, 
                            int width, int height,
                            int hotspotX, int hotspotY,
                            const QByteArray& data);
    
    void performanceStatsUpdated(const QString& deviceId,
                                 const QVariantMap& stats);

    void videoLayoutChanged(const QString& deviceId,
                            int widthDips, int heightDips);

    void displayListChanged(const QString& deviceId,
                            const QJsonArray& displays,
                            int activeDisplayIndex);

    void routeChanged(const QString& deviceId,
                      const QVariantMap& routeInfo);

    void hostCapabilitiesChanged(const QString& deviceId,
                                 bool supportsSendAttentionSequence,
                                 bool supportsLockWorkstation,
                                 bool supportsFileTransfer,
                                 bool supportsPrivacyScreen,
                                 bool supportsVirtualDisplay);

    void fileTransferProgress(const QString& deviceId,
                              const QString& transferId,
                              const QString& filename,
                              double bytesSent,
                              double totalBytes);
    void fileTransferComplete(const QString& deviceId,
                              const QString& transferId,
                              const QString& filename);
    void fileTransferError(const QString& deviceId,
                           const QString& transferId,
                           const QString& errorMessage);

    void fileDownloadStarted(const QString& deviceId,
                             const QString& transferId,
                             const QString& filename,
                             double totalBytes);
    void fileDownloadProgress(const QString& deviceId,
                              const QString& transferId,
                              const QString& filename,
                              double bytesReceived,
                              double totalBytes);
    void fileDownloadComplete(const QString& deviceId,
                              const QString& transferId,
                              const QString& filename,
                              const QString& savePath);
    void fileDownloadError(const QString& deviceId,
                           const QString& transferId,
                           const QString& errorMessage);

    void virtualDisplayStateChanged(const QString& deviceId,
                                    const QJsonObject& state);

private slots:
    void onMessageReceived(const QJsonObject& message);
    void onMessagingError(const QString& error);

private:
    QPointer<NativeMessaging> m_messaging;
    std::unique_ptr<SharedMemoryManager> m_sharedMemoryManager;
    QMap<QString, ConnectionInfo> m_connections;  // key = deviceId
    QHash<QString, QString> m_connIdToDeviceId;   // connectionId -> deviceId reverse lookup
    QString m_activeDeviceId;
    int m_connectionCounter = 0;
    
    QJsonObject m_iceConfig;

    // Resolve internal connectionId from a deviceId
    QString connectionIdFor(const QString& deviceId) const;
    // Reverse-lookup: find deviceId from an internal connectionId (used in message handlers)
    QString findDeviceId(const QString& connectionId) const;
    QString generateConnectionId();
    void removeConnection(const QString& deviceId);

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
    void handleDisplayListChanged(const QJsonObject& message);
    void handleRouteChanged(const QJsonObject& message);
    void handleHostCapabilities(const QJsonObject& message);
    void handleFileTransferProgress(const QJsonObject& message);
    void handleFileTransferComplete(const QJsonObject& message);
    void handleFileTransferError(const QJsonObject& message);
    void handleFileDownloadStarted(const QJsonObject& message);
    void handleFileDownloadProgress(const QJsonObject& message);
    void handleFileDownloadComplete(const QJsonObject& message);
    void handleFileDownloadError(const QJsonObject& message);
    void handleSkillBridgeResponse(const QJsonObject& message);
    void handleVirtualDisplayState(const QJsonObject& message);
    
    void sendMouseEvent(const QString& deviceId, const QString& eventType,
                        int x, int y, int button,
                        int wheelDeltaX, int wheelDeltaY);
    void sendKeyboardEvent(const QString& deviceId, const QString& eventType,
                           int nativeScanCode, int lockStates);
};

} // namespace quickdesk

#endif // QUICKDESK_MANAGER_CLIENTMANAGER_H
