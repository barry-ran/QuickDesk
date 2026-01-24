#include "applicationcontext.h"

#include <QCoreApplication>
#include <QDir>
#include <QStandardPaths>

#include "base/util.h"

namespace infra {

constexpr char kLogPath[] = "logs";
constexpr char kDbPath[] = "db";

bool ApplicationContext::init()
{
    m_localPicturesPath = QStandardPaths::writableLocation(QStandardPaths::PicturesLocation);
    m_localDataPath = QStandardPaths::writableLocation(QStandardPaths::AppDataLocation);
    m_localDownloadPath = QStandardPaths::writableLocation(QStandardPaths::DownloadLocation);
    m_logPath = m_localDataPath + QDir::separator() + kLogPath;
    m_dbPath = m_localDataPath + QDir::separator() + kDbPath;

    m_applicationMd5 = base::Util::fileMd5(applicationFilePath());

    QDir localDataDir(m_localDataPath);
    if (!localDataDir.mkpath(m_localDataPath)) {
        return false;
    }
    QDir dbDir(m_dbPath);
    if (!dbDir.mkpath(m_dbPath)) {
        return false;
    }

    return true;
}

QString ApplicationContext::applicationName() const
{
    return QCoreApplication::applicationName();
}

QString ApplicationContext::applicationDirPath() const
{
    return QCoreApplication::applicationDirPath();
}

QString ApplicationContext::applicationFilePath() const
{
    return QCoreApplication::applicationFilePath();
}

QString ApplicationContext::applicationVersion() const
{
    return QCoreApplication::applicationVersion();
}

QString ApplicationContext::lastRunApplicationVersion() const
{
    return m_lastApplicationVersion;
}

void ApplicationContext::setLastRunApplicationVersion(const QString& version)
{
    m_lastApplicationVersion = version;
}

QString ApplicationContext::maxRunApplicationVersion() const
{
    return m_maxApplicationVersion;
}

void ApplicationContext::setMaxRunApplicationVersion(const QString& version)
{
    m_maxApplicationVersion = version;
}

QString ApplicationContext::applicationMd5() const
{
    return m_applicationMd5;
}

QString ApplicationContext::localPicturesPath() const
{
    return m_localPicturesPath;
}

QString ApplicationContext::localDataPath() const
{
    return m_localDataPath;
}

QString ApplicationContext::localDownloadPath() const
{
    return m_localDownloadPath;
}

QString ApplicationContext::logPath() const
{
    return m_logPath;
}

QString ApplicationContext::dbPath() const
{
    return m_dbPath;
}
}
