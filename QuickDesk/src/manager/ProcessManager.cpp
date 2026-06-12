// Copyright 2026 QuickDesk Authors

#include "ProcessManager.h"
#include "NativeMessaging.h"
#include "infra/log/log.h"
#include <QCoreApplication>
#include <QDir>
#include <QFileInfo>
#ifdef Q_OS_WIN
#include <qt_windows.h>
#include <winsvc.h>
#endif

namespace quickdesk {

ProcessManager::ProcessManager(QObject* parent)
    : QObject(parent)
{
    m_hostRestartTimer.setSingleShot(true);
    connect(&m_hostRestartTimer, &QTimer::timeout,
            this, &ProcessManager::onHostRestartTimer);
    
    m_clientRestartTimer.setSingleShot(true);
    connect(&m_clientRestartTimer, &QTimer::timeout,
            this, &ProcessManager::onClientRestartTimer);

    m_pipeConnectTimer.setSingleShot(true);
    connect(&m_pipeConnectTimer, &QTimer::timeout,
            this, &ProcessManager::onServicePipeError);
}

ProcessManager::~ProcessManager()
{
    m_hostRestartTimer.stop();
    m_clientRestartTimer.stop();
    m_pipeConnectTimer.stop();

    m_hostAutoRestart = false;
    m_clientAutoRestart = false;

    // Disconnect signals before killing to prevent the finished() signal from
    // cascading into other objects that may already be destroyed during
    // application shutdown.
    if (m_hostProcess) {
        m_hostProcess->disconnect(this);
        if (m_hostProcess->state() != QProcess::NotRunning) {
            m_hostProcess->kill();
            m_hostProcess->waitForFinished(1000);
        }
    }
    if (m_clientProcess) {
        m_clientProcess->disconnect(this);
        if (m_clientProcess->state() != QProcess::NotRunning) {
            m_clientProcess->kill();
            m_clientProcess->waitForFinished(1000);
        }
    }
}

bool ProcessManager::startHostProcess()
{
    if (isHostRunning() || m_serviceConnecting) {
        LOG_WARN("Host process is already running or connecting");
        return true;
    }

    setHostProcessStatus(ProcessStatus::Starting);

#ifdef Q_OS_WIN
    if (isHostServiceRunning()) {
        LOG_INFO("QuickDeskHost service detected, connecting via Named Pipe");
        connectToHostServiceAsync();
        return true;
    }
#endif

    return startHostAsChildProcess();
}

bool ProcessManager::startHostAsChildProcess()
{
    if (m_hostExePath.isEmpty()) {
        emit hostProcessError("Host executable path not set");
        setHostProcessStatus(ProcessStatus::NotStarted);
        return false;
    }

    m_hostProcess = std::make_unique<QProcess>(this);
    
    connect(m_hostProcess.get(), 
            QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished),
            this, &ProcessManager::onHostProcessFinished);
    
    connect(m_hostProcess.get(), &QProcess::started,
            this, &ProcessManager::onHostProcessStarted);
    
    connect(m_hostProcess.get(), &QProcess::errorOccurred,
            this, &ProcessManager::onHostProcessErrorOccurred);

    if (!startProcess(m_hostProcess.get(), m_hostExePath, "Host", m_logDir)) {
        m_hostProcess.reset();
        setHostProcessStatus(ProcessStatus::NotStarted);
        return false;
    }

    return true;
}

bool ProcessManager::startClientProcess()
{
    if (isClientRunning()) {
        LOG_WARN("Client process is already running");
        return true;
    }

    if (m_clientExePath.isEmpty()) {
        emit clientProcessError("Client executable path not set");
        return false;
    }

    m_clientProcess = std::make_unique<QProcess>(this);
    
    connect(m_clientProcess.get(), 
            QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished),
            this, &ProcessManager::onClientProcessFinished);
    
    connect(m_clientProcess.get(), &QProcess::started,
            this, &ProcessManager::onClientProcessStarted);
    
    connect(m_clientProcess.get(), &QProcess::errorOccurred,
            this, &ProcessManager::onClientProcessErrorOccurred);

    setClientProcessStatus(ProcessStatus::Starting);
    if (!startProcess(m_clientProcess.get(), m_clientExePath, "Client", m_logDir)) {
        m_clientProcess.reset();
        setClientProcessStatus(ProcessStatus::NotStarted);
        return false;
    }

    return true;
}

void ProcessManager::stopHostProcess()
{
    m_hostRestartTimer.stop();
    m_hostStoppingIntentionally = true;

    // Cancel pending async connection.
    if (m_serviceConnecting) {
        m_serviceConnecting = false;
        m_pipeConnectTimer.stop();
    }

    m_hostLaunchMode = HostLaunchMode::Unknown;
    emit hostLaunchModeChanged();

    // Service mode: disconnect both sockets.
    if (m_hostReadSocket || m_hostWriteSocket) {
        LOG_INFO("Disconnecting from host service pipes...");
        cleanupServiceConnection();
        m_hostRestartCount = 0;
        setHostProcessStatus(ProcessStatus::NotStarted);
        return;
    }
    
    if (!m_hostProcess || m_hostProcess->state() == QProcess::NotRunning) {
        m_hostMessaging.reset();
        m_hostProcess.reset();
        m_hostRestartCount = 0;
        setHostProcessStatus(ProcessStatus::NotStarted);
        return;
    }

    LOG_INFO("Initiating async host process shutdown...");
    m_hostProcess->closeWriteChannel();

    // Phase 1→2→3 escalation via single-shot timers.
    // Cleanup happens in onHostProcessFinished when the process actually exits.
    QProcess* proc = m_hostProcess.get();
    QTimer::singleShot(5000, proc, [proc]() {
        if (proc->state() == QProcess::NotRunning) return;
        LOG_WARN("Host process did not exit gracefully, terminating...");
        proc->terminate();

        QTimer::singleShot(3000, proc, [proc]() {
            if (proc->state() == QProcess::NotRunning) return;
            LOG_WARN("Host process did not terminate, killing...");
            proc->kill();
        });
    });
}

void ProcessManager::stopClientProcess()
{
    m_clientRestartTimer.stop();
    m_clientStoppingIntentionally = true;

    if (!m_clientProcess || m_clientProcess->state() == QProcess::NotRunning) {
        m_clientMessaging.reset();
        m_clientProcess.reset();
        m_clientRestartCount = 0;
        setClientProcessStatus(ProcessStatus::NotStarted);
        return;
    }

    LOG_INFO("Initiating async client process shutdown...");
    m_clientProcess->closeWriteChannel();

    QProcess* proc = m_clientProcess.get();
    QTimer::singleShot(3000, proc, [proc]() {
        if (proc->state() == QProcess::NotRunning) return;
        LOG_WARN("Client process did not exit gracefully, terminating...");
        proc->terminate();

        QTimer::singleShot(2000, proc, [proc]() {
            if (proc->state() == QProcess::NotRunning) return;
            LOG_WARN("Client process did not terminate, killing...");
            proc->kill();
        });
    });
}

void ProcessManager::stopAllProcesses()
{
    stopHostProcess();
    stopClientProcess();
}

bool ProcessManager::isHostRunning() const
{
    if (m_hostMessaging && m_hostReadSocket &&
        m_hostReadSocket->state() == QLocalSocket::ConnectedState)
        return true;
    return m_hostProcess && m_hostProcess->state() == QProcess::Running;
}

bool ProcessManager::isClientRunning() const
{
    return m_clientProcess && m_clientProcess->state() == QProcess::Running;
}

NativeMessaging* ProcessManager::hostMessaging() const
{
    return m_hostMessaging.get();
}

NativeMessaging* ProcessManager::clientMessaging() const
{
    return m_clientMessaging.get();
}

void ProcessManager::setHostExePath(const QString& path)
{
    m_hostExePath = path;
}

void ProcessManager::setClientExePath(const QString& path)
{
    m_clientExePath = path;
}

void ProcessManager::setLogDir(const QString& logDir)
{
    m_logDir = logDir;
}

QString ProcessManager::logDir() const
{
    return m_logDir;
}

void ProcessManager::setConfigDir(const QString& configDir)
{
    m_configDir = configDir;
}

QString ProcessManager::configDir() const
{
    return m_configDir;
}

QString ProcessManager::hostExePath() const
{
    return m_hostExePath;
}

QString ProcessManager::clientExePath() const
{
    return m_clientExePath;
}

bool ProcessManager::autoDetectPaths()
{
    QString hostPath = findExecutable("quickdesk_host");
    QString clientPath = findExecutable("quickdesk_client");

    if (!hostPath.isEmpty()) {
        m_hostExePath = hostPath;
        LOG_INFO("Auto-detected host executable: {}", hostPath.toStdString());
    }

    if (!clientPath.isEmpty()) {
        m_clientExePath = clientPath;
        LOG_INFO("Auto-detected client executable: {}", clientPath.toStdString());
    }

    return !hostPath.isEmpty() && !clientPath.isEmpty();
}

bool ProcessManager::hostAutoRestart() const
{
    return m_hostAutoRestart;
}

void ProcessManager::setHostAutoRestart(bool enabled)
{
    if (m_hostAutoRestart != enabled) {
        m_hostAutoRestart = enabled;
        emit hostAutoRestartChanged();
    }
}

bool ProcessManager::clientAutoRestart() const
{
    return m_clientAutoRestart;
}

void ProcessManager::setClientAutoRestart(bool enabled)
{
    if (m_clientAutoRestart != enabled) {
        m_clientAutoRestart = enabled;
        emit clientAutoRestartChanged();
    }
}

ProcessStatus::Status ProcessManager::hostProcessStatus() const
{
    return m_hostProcessStatus;
}

ProcessStatus::Status ProcessManager::clientProcessStatus() const
{
    return m_clientProcessStatus;
}

HostLaunchMode::Mode ProcessManager::hostLaunchMode() const
{
    return m_hostLaunchMode;
}

void ProcessManager::resetHostRetryCount()
{
    m_hostRestartCount = 0;
}

void ProcessManager::resetClientRetryCount()
{
    m_clientRestartCount = 0;
}

void ProcessManager::onHostProcessFinished(int exitCode, QProcess::ExitStatus status)
{
    LOG_INFO("Host process finished with exit code: {} status: {}", 
             exitCode, 
             (status == QProcess::NormalExit ? "NormalExit" : "CrashExit"));
    
    // Emit signal BEFORE destroying messaging so listeners can disconnect first
    emit hostProcessStopped(exitCode);
    m_hostMessaging.reset();
    
    // Check if we should auto-restart
    bool isAbnormalExit = (status == QProcess::CrashExit) || (exitCode != 0);
    
    if (m_hostStoppingIntentionally) {
        m_hostStoppingIntentionally = false;
        m_hostProcess.reset();
        m_hostRestartCount = 0;
        setHostProcessStatus(ProcessStatus::NotStarted);
        LOG_INFO("Host stopped intentionally, not restarting");
        return;
    }
    
    if (!m_hostAutoRestart) {
        setHostProcessStatus(ProcessStatus::NotStarted);
        LOG_INFO("Host auto-restart disabled");
        return;
    }
    
    if (!isAbnormalExit) {
        // Normal exit with code 0, don't restart
        setHostProcessStatus(ProcessStatus::NotStarted);
        LOG_INFO("Host exited normally, not restarting");
        return;
    }
    
    // Abnormal exit - try to restart
    if (m_hostRestartCount >= MAX_RESTART_ATTEMPTS) {
        setHostProcessStatus(ProcessStatus::Failed);
        QString error = QString("Host process crashed %1 times, giving up").arg(MAX_RESTART_ATTEMPTS);
        LOG_WARN("{}", error.toStdString());
        emit hostProcessError(error);
        return;
    }
    
    m_hostRestartCount++;
    int delay = calculateRestartDelay(m_hostRestartCount);
    
    setHostProcessStatus(ProcessStatus::Restarting);
    LOG_INFO("Host crashed, restarting in {} ms (attempt {} of {})", 
             delay, m_hostRestartCount, MAX_RESTART_ATTEMPTS);
    
    emit hostProcessRestarting(m_hostRestartCount, MAX_RESTART_ATTEMPTS);
    m_hostRestartTimer.start(delay);
}

void ProcessManager::onClientProcessFinished(int exitCode, QProcess::ExitStatus status)
{
    LOG_INFO("Client process finished with exit code: {} status: {}", 
             exitCode,
             (status == QProcess::NormalExit ? "NormalExit" : "CrashExit"));
    
    // Emit signal BEFORE destroying messaging so listeners can disconnect first
    emit clientProcessStopped(exitCode);
    m_clientMessaging.reset();
    
    // Check if we should auto-restart
    bool isAbnormalExit = (status == QProcess::CrashExit) || (exitCode != 0);
    
    if (m_clientStoppingIntentionally) {
        m_clientStoppingIntentionally = false;
        m_clientProcess.reset();
        m_clientRestartCount = 0;
        setClientProcessStatus(ProcessStatus::NotStarted);
        LOG_INFO("Client stopped intentionally, not restarting");
        return;
    }
    
    if (!m_clientAutoRestart) {
        setClientProcessStatus(ProcessStatus::NotStarted);
        LOG_INFO("Client auto-restart disabled");
        return;
    }
    
    if (!isAbnormalExit) {
        // Normal exit with code 0, don't restart
        setClientProcessStatus(ProcessStatus::NotStarted);
        LOG_INFO("Client exited normally, not restarting");
        return;
    }
    
    // Abnormal exit - try to restart
    if (m_clientRestartCount >= MAX_RESTART_ATTEMPTS) {
        setClientProcessStatus(ProcessStatus::Failed);
        QString error = QString("Client process crashed %1 times, giving up").arg(MAX_RESTART_ATTEMPTS);
        LOG_WARN("{}", error.toStdString());
        emit clientProcessError(error);
        return;
    }
    
    m_clientRestartCount++;
    int delay = calculateRestartDelay(m_clientRestartCount);
    
    setClientProcessStatus(ProcessStatus::Restarting);
    LOG_INFO("Client crashed, restarting in {} ms (attempt {} of {})",
             delay, m_clientRestartCount, MAX_RESTART_ATTEMPTS);
    
    emit clientProcessRestarting(m_clientRestartCount, MAX_RESTART_ATTEMPTS);
    m_clientRestartTimer.start(delay);
}

void ProcessManager::onHostRestartTimer()
{
    LOG_INFO("Attempting to restart Host process...");
    // startHostProcess now returns false only if exe path is not set or file doesn't exist
    // Actual start failures are handled by errorOccurred signal
    startHostProcess();
}

void ProcessManager::onClientRestartTimer()
{
    LOG_INFO("Attempting to restart Client process...");
    // startClientProcess now returns false only if exe path is not set or file doesn't exist
    // Actual start failures are handled by errorOccurred signal
    startClientProcess();
}

void ProcessManager::onHostProcessStarted()
{
    // Create Native Messaging handler
    m_hostMessaging = std::make_unique<NativeMessaging>(m_hostProcess.get(), this);
    
    m_hostRestartCount = 0;
    m_hostLaunchMode = HostLaunchMode::ChildProcess;
    emit hostLaunchModeChanged();
    setHostProcessStatus(ProcessStatus::Running);
    emit hostProcessStarted();
    LOG_INFO("Host process started successfully, PID: {}", m_hostProcess->processId());
}

void ProcessManager::onHostProcessErrorOccurred(QProcess::ProcessError error)
{
    QString errorString = m_hostProcess ? m_hostProcess->errorString() : "Unknown error";
    LOG_WARN("Host process error occurred: {} - {}", (int)error, errorString.toStdString());
    
    if (error == QProcess::FailedToStart) {
        QString errorMsg = QString("Failed to start Host process: %1").arg(errorString);
        emit hostProcessError(errorMsg);
        
        // Check if we're in auto-restart mode and should retry
        if (m_hostAutoRestart && !m_hostStoppingIntentionally) {
            if (m_hostRestartCount < MAX_RESTART_ATTEMPTS) {
                m_hostRestartCount++;
                int delay = calculateRestartDelay(m_hostRestartCount);
                setHostProcessStatus(ProcessStatus::Restarting);
                LOG_INFO("Host failed to start, retrying in {} ms (attempt {} of {})",
                        delay, m_hostRestartCount, MAX_RESTART_ATTEMPTS);
                emit hostProcessRestarting(m_hostRestartCount, MAX_RESTART_ATTEMPTS);
                m_hostRestartTimer.start(delay);
            } else {
                setHostProcessStatus(ProcessStatus::Failed);
                emit hostProcessError("Failed to start Host after multiple attempts");
            }
        } else {
            setHostProcessStatus(ProcessStatus::NotStarted);
        }
    } else if (error == QProcess::Crashed) {
        LOG_WARN("Host process crashed");
        // Will be handled by onHostProcessFinished
    } else {
        QString errorMsg = QString("Host process error: %1").arg(errorString);
        emit hostProcessError(errorMsg);
    }
}

void ProcessManager::onClientProcessStarted()
{
    // Create Native Messaging handler
    m_clientMessaging = std::make_unique<NativeMessaging>(m_clientProcess.get(), this);
    
    m_clientRestartCount = 0;
    setClientProcessStatus(ProcessStatus::Running);
    emit clientProcessStarted();
    LOG_INFO("Client process started successfully, PID: {}", m_clientProcess->processId());
}

void ProcessManager::onClientProcessErrorOccurred(QProcess::ProcessError error)
{
    QString errorString = m_clientProcess ? m_clientProcess->errorString() : "Unknown error";
    LOG_WARN("Client process error occurred: {} - {}", (int)error, errorString.toStdString());
    
    if (error == QProcess::FailedToStart) {
        QString errorMsg = QString("Failed to start Client process: %1").arg(errorString);
        emit clientProcessError(errorMsg);
        
        // Check if we're in auto-restart mode and should retry
        if (m_clientAutoRestart && !m_clientStoppingIntentionally) {
            if (m_clientRestartCount < MAX_RESTART_ATTEMPTS) {
                m_clientRestartCount++;
                int delay = calculateRestartDelay(m_clientRestartCount);
                setClientProcessStatus(ProcessStatus::Restarting);
                LOG_INFO("Client failed to start, retrying in {} ms (attempt {} of {})",
                        delay, m_clientRestartCount, MAX_RESTART_ATTEMPTS);
                emit clientProcessRestarting(m_clientRestartCount, MAX_RESTART_ATTEMPTS);
                m_clientRestartTimer.start(delay);
            } else {
                setClientProcessStatus(ProcessStatus::Failed);
                emit clientProcessError("Failed to start Client after multiple attempts");
            }
        } else {
            setClientProcessStatus(ProcessStatus::NotStarted);
        }
    } else if (error == QProcess::Crashed) {
        LOG_WARN("Client process crashed");
        // Will be handled by onClientProcessFinished
    } else {
        QString errorMsg = QString("Client process error: %1").arg(errorString);
        emit clientProcessError(errorMsg);
    }
}

bool ProcessManager::startProcess(QProcess* process, const QString& exePath, 
                                  const QString& processName, const QString& logDir)
{
    QFileInfo fileInfo(exePath);
    if (!fileInfo.exists() || !fileInfo.isExecutable()) {
        QString error = QString("%1 executable not found or not executable: %2").arg(processName, exePath);
        LOG_WARN("{}", error.toStdString());
        if (processName == "Host") {
            emit hostProcessError(error);
        } else {
            emit clientProcessError(error);
        }
        return false;
    }

    connect(process, &QProcess::readyReadStandardError, this, [process, processName]() {
        QByteArray err = process->readAllStandardError();
        if (!err.isEmpty()) {
            // Add clear prefix to distinguish subprocess logs
            QString prefix = QString("========== BEGIN %1 PROCESS OUTPUT ==========").arg(processName.toUpper());
            LOG_INFO("{}", prefix.toStdString());
            
            // Split by newline and output each line separately
            QString errStr = QString::fromUtf8(err);
            QStringList lines = errStr.split('\n', Qt::SkipEmptyParts);
            for (const QString& lineErr : lines) {
                LOG_INFO("{}: {}", processName.toStdString(), lineErr.toStdString());
            }
            
            LOG_INFO("========== END %1 PROCESS OUTPUT ==========", processName.toStdString());
        }
    });

    // Prepare command line arguments
    QStringList arguments;
    if (!logDir.isEmpty()) {
        arguments << QString("--log-dir=%1").arg(logDir);
    }
    if (!m_configDir.isEmpty()) {
        arguments << QString("--config-dir=%1").arg(m_configDir);
    }
#ifdef Q_OS_MAC
    if (process == m_hostProcess.get()) {
        if (!m_macHostPermissionRequestSent) {
            arguments << QStringLiteral("--request-mac-permissions");
            m_macHostPermissionRequestSent = true;
        }
    }
#endif

    process->setProgram(exePath);
    process->setArguments(arguments);
    process->setWorkingDirectory(fileInfo.absolutePath());
    
    // Native Messaging uses stdin/stdout
    process->setProcessChannelMode(QProcess::SeparateChannels);
    
    LOG_INFO("Starting {} process: {}", processName.toStdString(), exePath.toStdString());
    process->start();
    return true;
}

QString ProcessManager::findExecutable(const QString& name)
{
    // Search paths in order of priority
    QStringList searchPaths;
        
    QString appDir = QCoreApplication::applicationDirPath();    

    // Relative to workspace (for development)
    //    Workspace root is the parent of QuickDesk/output/x64/{Debug|Release}
    //    On Windows: applicationDirPath = .../QuickDesk/output/x64/Debug
    //    On Mac:     applicationDirPath = .../QuickDesk/output/x64/Debug/QuickDesk.app/Contents/MacOS    
#ifdef Q_OS_MAC
    // Go up 3 extra levels for .app/Contents/MacOS
    static const QString kRelPrefix = "../../../../../../../src/out/";
#else
    static const QString kRelPrefix = "../../../../src/out/";
#endif

#ifdef QT_DEBUG
#if defined(__x86_64__) && defined(Q_OS_MAC)
    searchPaths << QDir(appDir).filePath(kRelPrefix + "Debug-x64");
#endif
    searchPaths << QDir(appDir).filePath(kRelPrefix + "Debug");
#else
#if defined(__x86_64__) && defined(Q_OS_MAC)
    searchPaths << QDir(appDir).filePath(kRelPrefix + "Release-x64");
#endif
    searchPaths << QDir(appDir).filePath(kRelPrefix + "Release");
#endif

#ifdef Q_OS_WIN
    // 3rdparty directory (for development)
    searchPaths << QDir(appDir).filePath("../../../QuickDesk/3rdparty/quickdesk-remoting/x64");

    // Same directory as Qt exec
    searchPaths << appDir;
#endif

#ifdef Q_OS_MAC
    // 3rdparty directory (for development)
#if defined(__x86_64__)
    searchPaths << QDir(appDir).filePath("../../../QuickDesk/3rdparty/quickdesk-remoting/x64");
#else
    searchPaths << QDir(appDir).filePath("../../../QuickDesk/3rdparty/quickdesk-remoting/arm64");
#endif
    // Contents/Frameworks/ for .app bundles (publish layout)
    searchPaths << QDir(appDir).filePath("../Frameworks");
#endif
    
#ifdef Q_OS_WIN
    QString exeName = name + ".exe";

    for (const QString& path : searchPaths) {
        QString fullPath = QDir(path).filePath(exeName);
        QFileInfo fileInfo(fullPath);
        if (fileInfo.exists() && fileInfo.isExecutable()) {
            return fileInfo.absoluteFilePath();
        }
    }
#elif defined(Q_OS_MAC)
    // On Mac, host is an .app bundle; client is a plain executable.
    // Try .app bundle first, then plain executable.
    for (const QString& path : searchPaths) {
        // Try as .app bundle: name.app/Contents/MacOS/name
        QString bundlePath = QDir(path).filePath(
            name + ".app/Contents/MacOS/" + name);
        QFileInfo bundleInfo(bundlePath);
        if (bundleInfo.exists() && bundleInfo.isExecutable()) {
            return bundleInfo.absoluteFilePath();
        }

        // Try as plain executable
        QString plainPath = QDir(path).filePath(name);
        QFileInfo plainInfo(plainPath);
        if (plainInfo.exists() && plainInfo.isExecutable()) {
            return plainInfo.absoluteFilePath();
        }
    }
#endif
    return QString();
}

int ProcessManager::calculateRestartDelay(int retryCount) const
{
    // Exponential backoff: 2s, 4s, 8s, 16s, 32s (capped at 32s)
    int delay = BASE_RESTART_DELAY_MS * (1 << (retryCount - 1));
    return qMin(delay, 32000);
}

void ProcessManager::setHostProcessStatus(ProcessStatus::Status status)
{
    if (m_hostProcessStatus != status) {
        m_hostProcessStatus = status;
        emit hostProcessStatusChanged();
    }
}

void ProcessManager::setClientProcessStatus(ProcessStatus::Status status)
{
    if (m_clientProcessStatus != status) {
        m_clientProcessStatus = status;
        emit clientProcessStatusChanged();
    }
}

bool ProcessManager::isHostServiceRunning() const
{
#ifdef Q_OS_WIN
    SC_HANDLE scm = OpenSCManagerW(nullptr, nullptr, SC_MANAGER_CONNECT);
    if (!scm)
        return false;
    SC_HANDLE svc = OpenServiceW(scm, L"QuickDeskHost", SERVICE_QUERY_STATUS);
    bool running = false;
    if (svc) {
        SERVICE_STATUS ss;
        if (QueryServiceStatus(svc, &ss) &&
            ss.dwCurrentState == SERVICE_RUNNING) {
            running = true;
        }
        CloseServiceHandle(svc);
    }
    CloseServiceHandle(scm);
    return running;
#else
    return false;
#endif
}

void ProcessManager::connectToHostServiceAsync()
{
#ifdef Q_OS_WIN
    DWORD sessionId = 0;
    ProcessIdToSessionId(GetCurrentProcessId(), &sessionId);
    QString pipeBase = QString("quickdesk_host_%1").arg(sessionId);

    m_hostReadSocket = std::make_unique<QLocalSocket>(this);
    m_hostWriteSocket = std::make_unique<QLocalSocket>(this);
    m_serviceConnecting = true;

    QString readPipeName = pipeBase + "_out";
    QString writePipeName = pipeBase + "_in";

    LOG_INFO("Connecting to host service pipes: {}", pipeBase.toStdString());

    // Phase 1: write socket connected → initiate read socket connection.
    connect(m_hostWriteSocket.get(), &QLocalSocket::connected, this,
        [this, readPipeName]() {
            if (!m_serviceConnecting) return;
            LOG_INFO("Write pipe connected, connecting read pipe...");
            m_hostReadSocket->connectToServer(readPipeName, QIODevice::ReadOnly);
        });

    // Phase 2: read socket connected → both pipes ready.
    connect(m_hostReadSocket.get(), &QLocalSocket::connected, this,
        &ProcessManager::onServicePipeConnected);

    // Error during connection phase or runtime.
    auto handleError = [this](QLocalSocket::LocalSocketError err) {
        if (m_serviceConnecting) {
            // Defer cleanup: connectToServer() may still be on the call stack.
            // Destroying the socket synchronously here causes use-after-free.
            QTimer::singleShot(0, this, &ProcessManager::onServicePipeError);
            return;
        }
        if (err != QLocalSocket::PeerClosedError) {
            auto* sock = qobject_cast<QLocalSocket*>(sender());
            QString msg = sock ? sock->errorString()
                               : QStringLiteral("Unknown pipe error");
            LOG_WARN("Host pipe error: {}", msg.toStdString());
            emit hostProcessError(msg);
        }
    };
    connect(m_hostWriteSocket.get(), &QLocalSocket::errorOccurred,
            this, handleError);
    connect(m_hostReadSocket.get(), &QLocalSocket::errorOccurred,
            this, handleError);

    // Runtime disconnect.
    connect(m_hostReadSocket.get(), &QLocalSocket::disconnected, this,
        [this]() {
            if (m_serviceConnecting) return;
            LOG_INFO("Disconnected from host service pipe");
            emit hostProcessStopped(0);
            cleanupServiceConnection();

            if (m_hostStoppingIntentionally) {
                m_hostStoppingIntentionally = false;
                setHostProcessStatus(ProcessStatus::NotStarted);
                return;
            }

            if (!m_hostAutoRestart) {
                setHostProcessStatus(ProcessStatus::NotStarted);
                return;
            }

            // Retry connecting to the service pipe as long as the service
            // is running. The worker should restart quickly after a clean
            // pipe disconnect.
            if (isHostServiceRunning()) {
                m_hostRestartCount++;
                if (m_hostRestartCount <= MAX_RESTART_ATTEMPTS) {
                    int delay = calculateRestartDelay(m_hostRestartCount);
                    setHostProcessStatus(ProcessStatus::Restarting);
                    LOG_INFO("Reconnecting to host service in {} ms (attempt {} of {})",
                             delay, m_hostRestartCount, MAX_RESTART_ATTEMPTS);
                    emit hostProcessRestarting(m_hostRestartCount, MAX_RESTART_ATTEMPTS);
                    m_hostRestartTimer.start(delay);
                    return;
                }
                LOG_WARN("Service pipe reconnect failed after {} attempts, "
                         "falling back to child process", MAX_RESTART_ATTEMPTS);
            }

            // Service not running or retries exhausted — fallback.
            m_hostRestartTimer.stop();
            m_hostLaunchMode = HostLaunchMode::Unknown;
            emit hostLaunchModeChanged();
            startHostAsChildProcess();
        });

    m_pipeConnectTimer.start(5000);
    m_hostWriteSocket->connectToServer(writePipeName, QIODevice::WriteOnly);
#endif
}

void ProcessManager::onServicePipeConnected()
{
    if (!m_serviceConnecting) return;
    m_serviceConnecting = false;
    m_pipeConnectTimer.stop();

    m_hostMessaging = std::make_unique<NativeMessaging>(
        m_hostReadSocket.get(), m_hostWriteSocket.get(), this);
    m_hostRestartCount = 0;
    m_hostLaunchMode = HostLaunchMode::Service;
    emit hostLaunchModeChanged();
    setHostProcessStatus(ProcessStatus::Running);
    emit hostProcessStarted();
    LOG_INFO("Connected to host service via Named Pipes");
}

void ProcessManager::onServicePipeError()
{
    if (!m_serviceConnecting) return;
    m_serviceConnecting = false;
    m_pipeConnectTimer.stop();
    cleanupServiceConnection();

    // If service is still running, retry pipe connection instead of falling
    // back to child process. The worker may be restarting after a clean exit.
    if (isHostServiceRunning()) {
        m_hostRestartCount++;
        if (m_hostRestartCount <= MAX_RESTART_ATTEMPTS) {
            int delay = calculateRestartDelay(m_hostRestartCount);
            setHostProcessStatus(ProcessStatus::Restarting);
            LOG_INFO("Service pipe not ready, retrying in {} ms (attempt {} of {})",
                     delay, m_hostRestartCount, MAX_RESTART_ATTEMPTS);
            emit hostProcessRestarting(m_hostRestartCount, MAX_RESTART_ATTEMPTS);
            m_hostRestartTimer.start(delay);
            return;
        }
        LOG_WARN("Service pipe unavailable after {} retries, falling back to "
                 "child process", MAX_RESTART_ATTEMPTS);
    } else {
        LOG_INFO("Service not running, starting as child process");
    }

    m_hostLaunchMode = HostLaunchMode::Unknown;
    emit hostLaunchModeChanged();
    startHostAsChildProcess();
}

void ProcessManager::cleanupServiceConnection()
{
    m_hostMessaging.reset();
    if (m_hostReadSocket) {
        m_hostReadSocket->disconnect(this);
        m_hostReadSocket->disconnectFromServer();
        m_hostReadSocket.reset();
    }
    if (m_hostWriteSocket) {
        m_hostWriteSocket->disconnect(this);
        m_hostWriteSocket->disconnectFromServer();
        m_hostWriteSocket.reset();
    }
}

} // namespace quickdesk
