// Copyright 2026 QuickDesk Authors
// Process lifecycle management

#ifndef QUICKDESK_MANAGER_PROCESSMANAGER_H
#define QUICKDESK_MANAGER_PROCESSMANAGER_H

#include <QObject>
#include <QProcess>
#include <QLocalSocket>
#include <QTimer>
#include <memory>

#include "../common/ProcessStatus.h"

namespace quickdesk {

class NativeMessaging;

/**
 * @brief Manages the lifecycle of Host and Client processes
 */
class ProcessManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(bool hostAutoRestart READ hostAutoRestart WRITE setHostAutoRestart NOTIFY hostAutoRestartChanged)
    Q_PROPERTY(bool clientAutoRestart READ clientAutoRestart WRITE setClientAutoRestart NOTIFY clientAutoRestartChanged)
    Q_PROPERTY(ProcessStatus::Status hostProcessStatus READ hostProcessStatus NOTIFY hostProcessStatusChanged)
    Q_PROPERTY(ProcessStatus::Status clientProcessStatus READ clientProcessStatus NOTIFY clientProcessStatusChanged)
    Q_PROPERTY(HostLaunchMode::Mode hostLaunchMode READ hostLaunchMode NOTIFY hostLaunchModeChanged)

public:
    explicit ProcessManager(QObject* parent = nullptr);
    ~ProcessManager() override;

    // Process management
    bool startHostProcess();
    bool startClientProcess();
    
    void stopHostProcess();
    void stopClientProcess();
    void stopAllProcesses();

    bool isHostRunning() const;
    bool isClientRunning() const;

    // Get Native Messaging handlers
    NativeMessaging* hostMessaging() const;
    NativeMessaging* clientMessaging() const;

    // Executable paths
    void setHostExePath(const QString& path);
    void setClientExePath(const QString& path);
    QString hostExePath() const;
    QString clientExePath() const;

    // Log directory
    void setLogDir(const QString& logDir);
    QString logDir() const;

    // Config directory (for host config.json)
    void setConfigDir(const QString& configDir);
    QString configDir() const;

    // Auto-detect executable paths
    bool autoDetectPaths();

    // Auto-restart settings
    bool hostAutoRestart() const;
    void setHostAutoRestart(bool enabled);
    bool clientAutoRestart() const;
    void setClientAutoRestart(bool enabled);

    // Process status
    ProcessStatus::Status hostProcessStatus() const;
    ProcessStatus::Status clientProcessStatus() const;

    // Host launch mode
    HostLaunchMode::Mode hostLaunchMode() const;

    // Reset retry counts (call after successful connection)
    void resetHostRetryCount();
    void resetClientRetryCount();

signals:
    void hostProcessStarted();
    void hostProcessStopped(int exitCode);
    void hostProcessError(const QString& error);
    void hostProcessRestarting(int retryCount, int maxRetries);
    void hostAutoRestartChanged();
    void hostProcessStatusChanged();

    void clientProcessStarted();
    void clientProcessStopped(int exitCode);
    void clientProcessError(const QString& error);
    void clientProcessRestarting(int retryCount, int maxRetries);
    void clientAutoRestartChanged();
    void clientProcessStatusChanged();
    void hostLaunchModeChanged();

private slots:
    void onHostProcessFinished(int exitCode, QProcess::ExitStatus status);
    void onClientProcessFinished(int exitCode, QProcess::ExitStatus status);
    void onHostProcessStarted();
    void onHostProcessErrorOccurred(QProcess::ProcessError error);
    void onClientProcessStarted();
    void onClientProcessErrorOccurred(QProcess::ProcessError error);
    void onHostRestartTimer();
    void onClientRestartTimer();

private:
    std::unique_ptr<QProcess> m_hostProcess;
    std::unique_ptr<QProcess> m_clientProcess;

    // Service mode: connect to host worker via two Named Pipes
    std::unique_ptr<QLocalSocket> m_hostReadSocket;
    std::unique_ptr<QLocalSocket> m_hostWriteSocket;

    std::unique_ptr<NativeMessaging> m_hostMessaging;
    std::unique_ptr<NativeMessaging> m_clientMessaging;

    QString m_hostExePath;
    QString m_clientExePath;
    QString m_logDir;
    QString m_configDir;

    // Auto-restart settings
    bool m_hostAutoRestart = true;
    bool m_clientAutoRestart = true;
    static constexpr int MAX_RESTART_ATTEMPTS = 5;
    static const int BASE_RESTART_DELAY_MS = 2000; // 2 seconds

    // Host restart state
    int m_hostRestartCount = 0;
    bool m_hostStoppingIntentionally = false;
    bool m_macHostPermissionRequestSent = false;
    QTimer m_hostRestartTimer;
    ProcessStatus::Status m_hostProcessStatus = ProcessStatus::NotStarted;

    // Client restart state
    int m_clientRestartCount = 0;
    bool m_clientStoppingIntentionally = false;
    QTimer m_clientRestartTimer;
    ProcessStatus::Status m_clientProcessStatus = ProcessStatus::NotStarted;
    HostLaunchMode::Mode m_hostLaunchMode = HostLaunchMode::Unknown;

    bool startProcess(QProcess* process, const QString& exePath, 
                      const QString& processName, const QString& logDir);
    bool startHostAsChildProcess();
    void connectToHostServiceAsync();
    void cleanupServiceConnection();
    void onServicePipeConnected();
    void onServicePipeError();
    bool isHostServiceRunning() const;
    bool m_serviceConnecting = false;
    QTimer m_pipeConnectTimer;

    QString findExecutable(const QString& name);
    int calculateRestartDelay(int retryCount) const;
    void setHostProcessStatus(ProcessStatus::Status status);
    void setClientProcessStatus(ProcessStatus::Status status);
};

} // namespace quickdesk

#endif // QUICKDESK_MANAGER_PROCESSMANAGER_H
