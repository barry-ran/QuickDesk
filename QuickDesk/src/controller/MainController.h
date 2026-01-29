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

namespace quickdesk {

class ProcessManager;

/**
 * @brief Main controller that coordinates all managers
 * 
 * Exposed to QML as the primary interface for the application.
 */
class MainController : public QObject {
    Q_OBJECT
    Q_PROPERTY(bool isInitialized READ isInitialized NOTIFY initializedChanged)
    Q_PROPERTY(QString initStatus READ initStatus NOTIFY initStatusChanged)
    Q_PROPERTY(ServerManager* serverManager READ serverManager CONSTANT)
    Q_PROPERTY(HostManager* hostManager READ hostManager CONSTANT)
    Q_PROPERTY(ClientManager* clientManager READ clientManager CONSTANT)
    Q_PROPERTY(QString hostProcessStatus READ hostProcessStatus NOTIFY hostProcessStatusChanged)
    Q_PROPERTY(QString clientProcessStatus READ clientProcessStatus NOTIFY clientProcessStatusChanged)
    
    // Password auto-refresh info
    Q_PROPERTY(QString nextPasswordRefreshTime READ nextPasswordRefreshTime NOTIFY nextPasswordRefreshTimeChanged)

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
     * @brief Connect host to signaling server
     */
    Q_INVOKABLE void startHosting(const QString& serverUrl = QString());

    /**
     * @brief Disconnect host from signaling server
     */
    Q_INVOKABLE void stopHosting();

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
     * @brief Refresh temporary password
     */
    Q_INVOKABLE void refreshTempPassword();
    
    /**
     * @brief Reset password refresh timer (called after manual refresh)
     */
    void resetPasswordRefreshTimer();

    /**
     * @brief Copy text to clipboard
     */
    Q_INVOKABLE void copyToClipboard(const QString& text);

    /**
     * @brief Copy device info (ID and access code) to clipboard
     */
    Q_INVOKABLE void copyDeviceInfo();

    // Property getters
    bool isInitialized() const;
    QString initStatus() const;
    
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
    
    // Process status
    QString hostProcessStatus() const;
    QString clientProcessStatus() const;
    
    // Password auto-refresh info
    QString nextPasswordRefreshTime() const;

signals:
    void initializedChanged();
    void initStatusChanged();
    void initializationFailed(const QString& error);
    void deviceIdChanged();
    void accessCodeChanged();
    void hostConnectionChanged();
    void signalingStateChanged();
    void hostProcessStatusChanged();
    void clientProcessStatusChanged();
    void hostProcessRestarting(int retryCount, int maxRetries);
    void clientProcessRestarting(int retryCount, int maxRetries);
    void nextPasswordRefreshTimeChanged();

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

    void onHostReady(const QString& deviceId, const QString& accessCode);

private:
    std::unique_ptr<ProcessManager> m_processManager;
    std::unique_ptr<ServerManager> m_serverManager;
    std::unique_ptr<HostManager> m_hostManager;
    std::unique_ptr<ClientManager> m_clientManager;

    bool m_isInitialized = false;
    QString m_initStatus = "未初始化";
    QString m_deviceId;
    QString m_accessCode;
    QString m_lastServerUrl;  // For auto-reconnect after Host restart
    bool m_hostWasHosting = false;  // Was Host connected before restart
    
    // Password auto-refresh timer
    QTimer m_passwordRefreshTimer;
    int m_passwordRefreshIntervalMinutes = -1;  // -1 = disabled
    QDateTime m_nextRefreshTime;  // Next scheduled refresh time
    
    void onPasswordRefreshTimer();
    void updatePasswordRefreshTimer();

    void updateInitStatus(const QString& status);
    void checkInitialized();
    QString getDefaultServerUrl() const;
};

} // namespace quickdesk

#endif // QUICKDESK_CONTROLLER_MAINCONTROLLER_H
