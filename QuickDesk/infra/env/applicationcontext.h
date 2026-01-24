#pragma once

#include <QString>

#include "base/singleton.h"

namespace infra {

class ApplicationContext : public base::Singleton<ApplicationContext> {
public:
    bool init();

    QString applicationName() const;
    QString applicationDirPath() const;
    QString applicationFilePath() const;
    QString applicationVersion() const;
    QString lastRunApplicationVersion() const;
    void setLastRunApplicationVersion(const QString& version);
    // 本地运行过最大的app版本（不包含当前运行的版本）
    QString maxRunApplicationVersion() const;
    void setMaxRunApplicationVersion(const QString& version);
    QString applicationMd5() const;

    QString localPicturesPath() const;
    QString localDataPath() const;
    QString localDownloadPath() const;
    QString logPath() const;
    QString dbPath() const;

private:
    QString m_localPicturesPath;
    QString m_localDataPath;
    QString m_localDownloadPath;
    QString m_logPath;
    QString m_dbPath;
    QString m_applicationMd5;
    QString m_lastApplicationVersion;
    QString m_maxApplicationVersion;
};

}
