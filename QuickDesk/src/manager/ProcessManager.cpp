// Copyright 2026 QuickDesk Authors

#include "ProcessManager.h"
#include "NativeMessaging.h"
#include <QCoreApplication>
#include <QDir>
#include <QFileInfo>
#include <QDebug>

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
        qWarning() << "Host process is already running";
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

    if (!startProcess(m_hostProcess.get(), m_hostExePath, "Host")) {
        m_hostProcess.reset();
        return false;
    }

    // Create Native Messaging handler
    m_hostMessaging = std::make_unique<NativeMessaging>(m_hostProcess.get(), this);
    
    setHostStatus("running");
    emit hostProcessStarted();
    qInfo() << "Host process started, PID:" << m_hostProcess->processId();
    
    return true;
}

bool ProcessManager::startClientProcess()
{
    if (isClientRunning()) {
        qWarning() << "Client process is already running";
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

    if (!startProcess(m_clientProcess.get(), m_clientExePath, "Client")) {
        m_clientProcess.reset();
        return false;
    }

    // Create Native Messaging handler
    m_clientMessaging = std::make_unique<NativeMessaging>(m_clientProcess.get(), this);
    
    setClientStatus("running");
    emit clientProcessStarted();
    qInfo() << "Client process started, PID:" << m_clientProcess->processId();
    
    return true;
}

void ProcessManager::stopHostProcess()
{
    m_hostRestartTimer.stop();
    m_hostStoppingIntentionally = true;
    
    if (m_hostProcess && m_hostProcess->state() != QProcess::NotRunning) {
        qInfo() << "Stopping host process...";
        m_hostProcess->closeWriteChannel(); // Close stdin to trigger graceful exit
        if (!m_hostProcess->waitForFinished(1000)) {
            qWarning() << "Host process did not exit gracefully, terminating...";
            m_hostProcess->terminate();
            m_hostProcess->kill();
        }
    }
    m_hostMessaging.reset();
    m_hostProcess.reset();
    m_hostRestartCount = 0;
    setHostStatus("stopped");
}

void ProcessManager::stopClientProcess()
{
    m_clientRestartTimer.stop();
    m_clientStoppingIntentionally = true;
    
    if (m_clientProcess && m_clientProcess->state() != QProcess::NotRunning) {
        qInfo() << "Stopping client process...";
        m_clientProcess->closeWriteChannel();
        if (!m_clientProcess->waitForFinished(1000)) {
            qWarning() << "Client process did not exit gracefully, terminating...";
            m_clientProcess->terminate();
            m_clientProcess->kill();
        }
    }
    m_clientMessaging.reset();
    m_clientProcess.reset();
    m_clientRestartCount = 0;
    setClientStatus("stopped");
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
        qInfo() << "Auto-detected host executable:" << hostPath;
    }

    if (!clientPath.isEmpty()) {
        m_clientExePath = clientPath;
        qInfo() << "Auto-detected client executable:" << clientPath;
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

QString ProcessManager::hostStatus() const
{
    return m_hostStatus;
}

QString ProcessManager::clientStatus() const
{
    return m_clientStatus;
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
    qInfo() << "Host process finished with exit code:" << exitCode 
            << "status:" << (status == QProcess::NormalExit ? "NormalExit" : "CrashExit");
    
    // Emit signal BEFORE destroying messaging so listeners can disconnect first
    emit hostProcessStopped(exitCode);
    m_hostMessaging.reset();
    
    // Check if we should auto-restart
    bool isAbnormalExit = (status == QProcess::CrashExit) || (exitCode != 0);
    
    if (m_hostStoppingIntentionally) {
        // User requested stop, don't restart
        m_hostStoppingIntentionally = false;
        setHostStatus("stopped");
        qInfo() << "Host stopped intentionally, not restarting";
        return;
    }
    
    if (!m_hostAutoRestart) {
        setHostStatus("stopped");
        qInfo() << "Host auto-restart disabled";
        return;
    }
    
    if (!isAbnormalExit) {
        // Normal exit with code 0, don't restart
        setHostStatus("stopped");
        qInfo() << "Host exited normally, not restarting";
        return;
    }
    
    // Abnormal exit - try to restart
    if (m_hostRestartCount >= MAX_RESTART_ATTEMPTS) {
        setHostStatus("failed");
        QString error = QString("Host process crashed %1 times, giving up").arg(MAX_RESTART_ATTEMPTS);
        qWarning() << error;
        emit hostProcessError(error);
        return;
    }
    
    m_hostRestartCount++;
    int delay = calculateRestartDelay(m_hostRestartCount);
    
    setHostStatus(QString("restarting:%1").arg(m_hostRestartCount));
    qInfo() << "Host crashed, restarting in" << delay << "ms (attempt" 
            << m_hostRestartCount << "of" << MAX_RESTART_ATTEMPTS << ")";
    
    emit hostProcessRestarting(m_hostRestartCount, MAX_RESTART_ATTEMPTS);
    m_hostRestartTimer.start(delay);
}

void ProcessManager::onClientProcessFinished(int exitCode, QProcess::ExitStatus status)
{
    qInfo() << "Client process finished with exit code:" << exitCode 
            << "status:" << (status == QProcess::NormalExit ? "NormalExit" : "CrashExit");
    
    // Emit signal BEFORE destroying messaging so listeners can disconnect first
    emit clientProcessStopped(exitCode);
    m_clientMessaging.reset();
    
    // Check if we should auto-restart
    bool isAbnormalExit = (status == QProcess::CrashExit) || (exitCode != 0);
    
    if (m_clientStoppingIntentionally) {
        // User requested stop, don't restart
        m_clientStoppingIntentionally = false;
        setClientStatus("stopped");
        qInfo() << "Client stopped intentionally, not restarting";
        return;
    }
    
    if (!m_clientAutoRestart) {
        setClientStatus("stopped");
        qInfo() << "Client auto-restart disabled";
        return;
    }
    
    if (!isAbnormalExit) {
        // Normal exit with code 0, don't restart
        setClientStatus("stopped");
        qInfo() << "Client exited normally, not restarting";
        return;
    }
    
    // Abnormal exit - try to restart
    if (m_clientRestartCount >= MAX_RESTART_ATTEMPTS) {
        setClientStatus("failed");
        QString error = QString("Client process crashed %1 times, giving up").arg(MAX_RESTART_ATTEMPTS);
        qWarning() << error;
        emit clientProcessError(error);
        return;
    }
    
    m_clientRestartCount++;
    int delay = calculateRestartDelay(m_clientRestartCount);
    
    setClientStatus(QString("restarting:%1").arg(m_clientRestartCount));
    qInfo() << "Client crashed, restarting in" << delay << "ms (attempt" 
            << m_clientRestartCount << "of" << MAX_RESTART_ATTEMPTS << ")";
    
    emit clientProcessRestarting(m_clientRestartCount, MAX_RESTART_ATTEMPTS);
    m_clientRestartTimer.start(delay);
}

void ProcessManager::onHostRestartTimer()
{
    qInfo() << "Attempting to restart Host process...";
    if (!startHostProcess()) {
        // Failed to start, increment count and try again
        if (m_hostRestartCount < MAX_RESTART_ATTEMPTS) {
            m_hostRestartCount++;
            int delay = calculateRestartDelay(m_hostRestartCount);
            setHostStatus(QString("restarting:%1").arg(m_hostRestartCount));
            emit hostProcessRestarting(m_hostRestartCount, MAX_RESTART_ATTEMPTS);
            m_hostRestartTimer.start(delay);
        } else {
            setHostStatus("failed");
            emit hostProcessError("Failed to restart Host after multiple attempts");
        }
    }
}

void ProcessManager::onClientRestartTimer()
{
    qInfo() << "Attempting to restart Client process...";
    if (!startClientProcess()) {
        // Failed to start, increment count and try again
        if (m_clientRestartCount < MAX_RESTART_ATTEMPTS) {
            m_clientRestartCount++;
            int delay = calculateRestartDelay(m_clientRestartCount);
            setClientStatus(QString("restarting:%1").arg(m_clientRestartCount));
            emit clientProcessRestarting(m_clientRestartCount, MAX_RESTART_ATTEMPTS);
            m_clientRestartTimer.start(delay);
        } else {
            setClientStatus("failed");
            emit clientProcessError("Failed to restart Client after multiple attempts");
        }
    }
}

bool ProcessManager::startProcess(QProcess* process, const QString& exePath, 
                                  const QString& processName)
{
    QFileInfo fileInfo(exePath);
    if (!fileInfo.exists() || !fileInfo.isExecutable()) {
        QString error = QString("%1 executable not found: %2").arg(processName, exePath);
        qWarning() << error;
        if (processName == "Host") {
            emit hostProcessError(error);
        } else {
            emit clientProcessError(error);
        }
        return false;
    }

    process->setProgram(exePath);
    process->setWorkingDirectory(fileInfo.absolutePath());
    
    // Native Messaging uses stdin/stdout
    process->setProcessChannelMode(QProcess::SeparateChannels);
    
    process->start();
    
    if (!process->waitForStarted(5000)) {
        QString error = QString("Failed to start %1 process: %2")
                        .arg(processName, process->errorString());
        qWarning() << error;
        if (processName == "Host") {
            emit hostProcessError(error);
        } else {
            emit clientProcessError(error);
        }
        return false;
    }

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
    searchPaths << QDir(appDir).filePath("../../src/out/Debug");
    searchPaths << QDir(appDir).filePath("../../../src/out/Debug");
    searchPaths << QDir(appDir).filePath("../../../../src/out/Debug");
    
    // 3. Absolute development paths
    searchPaths << "D:/mycode/remoting/src/out/Debug";
    
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

void ProcessManager::setHostStatus(const QString& status)
{
    if (m_hostStatus != status) {
        m_hostStatus = status;
        emit hostStatusChanged();
    }
}

void ProcessManager::setClientStatus(const QString& status)
{
    if (m_clientStatus != status) {
        m_clientStatus = status;
        emit clientStatusChanged();
    }
}

} // namespace quickdesk
