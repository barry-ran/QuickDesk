// Copyright 2026 QuickDesk Authors

#include "ProcessManager.h"
#include "NativeMessaging.h"
#include "infra/log/log.h"
#include <QCoreApplication>
#include <QDir>
#include <QFileInfo>

namespace quickdesk {

ProcessManager::ProcessManager(QObject* parent)
    : QObject(parent)
{
    // Setup restart timers
    m_hostRestartTimer.setSingleShot(true);
    connect(&m_hostRestartTimer, &QTimer::timeout,
            this, &ProcessManager::onHostRestartTimer);
    
    m_clientRestartTimer.setSingleShot(true);
    connect(&m_clientRestartTimer, &QTimer::timeout,
            this, &ProcessManager::onClientRestartTimer);
}

ProcessManager::~ProcessManager()
{
    // Stop timers first
    m_hostRestartTimer.stop();
    m_clientRestartTimer.stop();
    
    // Disable auto-restart during destruction
    m_hostAutoRestart = false;
    m_clientAutoRestart = false;
    
    stopAllProcesses();
}

bool ProcessManager::startHostProcess()
{
    if (isHostRunning()) {
        LOG_WARN("Host process is already running");
        return true;
    }

    if (m_hostExePath.isEmpty()) {
        emit hostProcessError("Host executable path not set");
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

    setHostProcessStatus(ProcessStatus::Starting);
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
    
    if (m_hostProcess && m_hostProcess->state() != QProcess::NotRunning) {
        LOG_INFO("begin stop host process...");
        m_hostProcess->closeWriteChannel(); // Close stdin to trigger graceful exit
        bool finished = m_hostProcess->waitForFinished(10000);
        if (!finished) {
            LOG_WARN("Host process did not exit gracefully, terminating...");
            m_hostProcess->terminate();
            finished = m_hostProcess->waitForFinished(5000);
        }
        if (!finished) {
            LOG_WARN("Host process did not terminate, killing...");
            m_hostProcess->kill();
            finished = m_hostProcess->waitForFinished(5000);
        }
        LOG_INFO("end stop host process...");
        if (!finished && m_hostProcess->state() != QProcess::NotRunning) {
            LOG_WARN("Host process still running, skipping destroy");
            return;
        }
    }
    m_hostMessaging.reset();
    m_hostProcess.reset();
    m_hostRestartCount = 0;
    setHostProcessStatus(ProcessStatus::NotStarted);
}

void ProcessManager::stopClientProcess()
{
    m_clientRestartTimer.stop();
    m_clientStoppingIntentionally = true;
    
    if (m_clientProcess && m_clientProcess->state() != QProcess::NotRunning) {
        LOG_INFO("begin stop client process...");
        m_clientProcess->closeWriteChannel();
        bool finished = m_clientProcess->waitForFinished(3000);
        if (!finished) {
            LOG_WARN("Client process did not exit gracefully, terminating...");
            m_clientProcess->terminate();
            finished = m_clientProcess->waitForFinished(3000);
        }
        if (!finished) {
            LOG_WARN("Client process did not terminate, killing...");
            m_clientProcess->kill();
            finished = m_clientProcess->waitForFinished(3000);
        }
        LOG_INFO("end stop client process...");
        if (!finished && m_clientProcess->state() != QProcess::NotRunning) {
            LOG_WARN("Client process still running, skipping destroy");
            return;
        }
    }
    m_clientMessaging.reset();
    m_clientProcess.reset();
    m_clientRestartCount = 0;
    setClientProcessStatus(ProcessStatus::NotStarted);
}

void ProcessManager::stopAllProcesses()
{
    stopHostProcess();
    stopClientProcess();
}

bool ProcessManager::isHostRunning() const
{
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
        // User requested stop, don't restart
        m_hostStoppingIntentionally = false;
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
        // User requested stop, don't restart
        m_clientStoppingIntentionally = false;
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
    
    setHostProcessStatus(ProcessStatus::Running);
    emit hostProcessStarted();
    LOG_INFO("Host process started successfully, PID: {}", m_hostProcess->processId());
}

void ProcessManager::onHostProcessErrorOccurred(QProcess::ProcessError error)
{
    QString errorString = m_hostProcess ? m_hostProcess->errorString() : "Unknown error";
    LOG_WARN("Host process error occurred: {} - {}", error, errorString.toStdString());
    
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
    
    setClientProcessStatus(ProcessStatus::Running);
    emit clientProcessStarted();
    LOG_INFO("Client process started successfully, PID: {}", m_clientProcess->processId());
}

void ProcessManager::onClientProcessErrorOccurred(QProcess::ProcessError error)
{
    QString errorString = m_clientProcess ? m_clientProcess->errorString() : "Unknown error";
    LOG_WARN("Client process error occurred: {} - {}", error, errorString.toStdString());
    
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
        // Use --log-dir=path format (Chromium style)
        arguments << QString("--log-dir=%1").arg(logDir);
    }

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
    
    // 1. Same directory as Qt app
    searchPaths << QCoreApplication::applicationDirPath();
    
    // 2. Relative to workspace (for development)
    QString appDir = QCoreApplication::applicationDirPath();
#ifdef QT_DEBUG
    searchPaths << QDir(appDir).filePath("../../../../src/out/Debug");
#else
    searchPaths << QDir(appDir).filePath("../../../../src/out/Release");
#endif
    
#ifdef Q_OS_WIN
    QString exeName = name + ".exe";
#else
    QString exeName = name;
#endif

    for (const QString& path : searchPaths) {
        QString fullPath = QDir(path).filePath(exeName);
        QFileInfo fileInfo(fullPath);
        if (fileInfo.exists() && fileInfo.isExecutable()) {
            return fileInfo.absoluteFilePath();
        }
    }

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

} // namespace quickdesk
