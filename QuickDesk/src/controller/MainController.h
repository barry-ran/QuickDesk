// Copyright 2026 QuickDesk Authors
// Main controller for QuickDesk Qt application

#ifndef QUICKDESK_CONTROLLER_MAINCONTROLLER_H
#define QUICKDESK_CONTROLLER_MAINCONTROLLER_H

#include <QObject>
#include <QTimer>
#include <memory>

#include "../manager/ServerManager.h"
#include "../manager/HostManager.h"
#include "../manager/ClientManager.h"
#include "../common/ProcessStatus.h"

namespace quickdesk {

class ProcessManager;

/**
 * @brief Main controller that coordinates all managers
 * 
 * Exposed to QML as the primary interface for the application.
 */
class MainController : public QObject {
    Q_OBJECT
    Q_PROPERTY(ServerManager* serverManager READ serverManager CONSTANT)
    Q_PROPERTY(HostManager* hostManager READ hostManager CONSTANT)
    Q_PROPERTY(ClientManager* clientManager READ clientManager CONSTANT)
    
    // Host status
    Q_PROPERTY(ProcessStatus::Status hostProcessStatus READ hostProcessStatus NOTIFY hostProcessStatusChanged)
    Q_PROPERTY(ServerStatus::Status hostServerStatus READ hostServerStatus NOTIFY hostServerStatusChanged)
    
    // Client status
    Q_PROPERTY(ProcessStatus::Status clientProcessStatus READ clientProcessStatus NOTIFY clientProcessStatusChanged)
    Q_PROPERTY(ServerStatus::Status clientServerStatus READ clientServerStatus NOTIFY clientServerStatusChanged)
    
    // Access code auto-refresh info
    Q_PROPERTY(QString nextAccessCodeRefreshTime READ nextAccessCodeRefreshTime NOTIFY nextAccessCodeRefreshTimeChanged)

    // Host properties (convenience for QML)
    Q_PROPERTY(QString deviceId READ deviceId NOTIFY deviceIdChanged)
    Q_PROPERTY(QString accessCode READ accessCode NOTIFY accessCodeChanged)
    Q_PROPERTY(bool isHostConnected READ isHostConnected NOTIFY hostConnectionChanged)
    
    // Signaling state properties (convenience for QML)
    Q_PROPERTY(QString signalingState READ signalingState NOTIFY signalingStateChanged)
    Q_PROPERTY(int signalingRetryCount READ signalingRetryCount NOTIFY signalingStateChanged)
    Q_PROPERTY(int signalingNextRetryIn READ signalingNextRetryIn NOTIFY signalingStateChanged)
    Q_PROPERTY(QString signalingError READ signalingError NOTIFY signalingStateChanged)
    Q_PROPERTY(QString signalingStatusText READ signalingStatusText NOTIFY signalingStateChanged)

public:
    explicit MainController(QObject* parent = nullptr);
    ~MainController() override;

    /**
     * @brief Initialize all components
     * 
     * Starts Host and Client processes.
     */
    Q_INVOKABLE void initialize();

    /**
     * @brief Shutdown all components
     */
    Q_INVOKABLE void shutdown();

    /**
     * @brief Connect to a remote host
     * @return Connection ID
     */
    Q_INVOKABLE QString connectToRemoteHost(const QString& deviceId,
                                            const QString& accessCode,
                                            const QString& serverUrl = QString());

    /**
     * @brief Disconnect from a remote host
     */
    Q_INVOKABLE void disconnectFromRemoteHost(const QString& connectionId);

    /**
     * @brief Refresh access code
     */
    Q_INVOKABLE void refreshAccessCode();
    
    /**
     * @brief Reset access code refresh timer (called after manual refresh)
     */
    void resetAccessCodeRefreshTimer();

    /**
     * @brief Copy text to clipboard
     */
    Q_INVOKABLE void copyToClipboard(const QString& text);

    /**
     * @brief Copy device info (ID and access code) to clipboard
     */
    Q_INVOKABLE void copyDeviceInfo();

    // Property getters
    ServerManager* serverManager() const;
    HostManager* hostManager() const;
    ClientManager* clientManager() const;

    // Host convenience properties
    QString deviceId() const;
    QString accessCode() const;
    bool isHostConnected() const;
    
    // Signaling state convenience properties
    QString signalingState() const;
    int signalingRetryCount() const;
    int signalingNextRetryIn() const;
    QString signalingError() const;
    QString signalingStatusText() const;
    
    // Status getters
    ProcessStatus::Status hostProcessStatus() const;
    ServerStatus::Status hostServerStatus() const;
    ProcessStatus::Status clientProcessStatus() const;
    ServerStatus::Status clientServerStatus() const;
    
    // Access code auto-refresh info
    QString nextAccessCodeRefreshTime() const;

signals:
    void initializationFailed(const QString& error);
    void deviceIdChanged();
    void accessCodeChanged();
    void hostConnectionChanged();
    void signalingStateChanged();
    void hostProcessStatusChanged();
    void hostServerStatusChanged();
    void clientProcessStatusChanged();
    void clientServerStatusChanged();
    void nextAccessCodeRefreshTimeChanged();

private slots:
    void onHostProcessStarted();
    void onHostProcessStopped(int exitCode);
    void onHostProcessError(const QString& error);
    void onHostProcessRestarting(int retryCount, int maxRetries);
    void onHostStatusChanged();
    
    void onClientProcessStarted();
    void onClientProcessStopped(int exitCode);
    void onClientProcessError(const QString& error);
    void onClientProcessRestarting(int retryCount, int maxRetries);
    void onClientStatusChanged();
    void onClientSignalingStateChanged(const QString& connectionId,
                                       const QString& state,
                                       int retryCount,
                                       int nextRetryIn,
                                       const QString& error);
    
    void onHostReady(const QString& deviceId, const QString& accessCode);

private:
    std::unique_ptr<ProcessManager> m_processManager;
    std::unique_ptr<ServerManager> m_serverManager;
    std::unique_ptr<HostManager> m_hostManager;
    std::unique_ptr<ClientManager> m_clientManager;

    QString m_deviceId;
    QString m_accessCode;
    
    // Host status
    ProcessStatus::Status m_hostProcessStatus = ProcessStatus::NotStarted;
    ServerStatus::Status m_hostServerStatus = ServerStatus::Disconnected;
    
    // Client status
    ProcessStatus::Status m_clientProcessStatus = ProcessStatus::NotStarted;
    ServerStatus::Status m_clientServerStatus = ServerStatus::Disconnected;
    QString m_primaryConnectionId;  // Track primary connection for client signaling status
    
    // Access code auto-refresh timer
    QTimer m_accessCodeRefreshTimer;
    int m_accessCodeRefreshIntervalMinutes = -1;  // -1 = disabled
    QDateTime m_nextRefreshTime;  // Next scheduled refresh time
    
    void onAccessCodeRefreshTimer();
    void updateAccessCodeRefreshTimer();
    QString getDefaultServerUrl() const;
};

} // namespace quickdesk

#endif // QUICKDESK_CONTROLLER_MAINCONTROLLER_H
