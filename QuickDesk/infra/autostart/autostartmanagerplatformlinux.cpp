#include "autostartmanagerplatform.h"

#include <QCoreApplication>
#include <QDir>
#include <QFile>
#include <QFileInfo>
#include <QStandardPaths>
#include <QTextStream>

namespace infra {

namespace {

QString getAutostartFilePath()
{
    QString configDir = QStandardPaths::writableLocation(QStandardPaths::ConfigLocation);
    if (configDir.isEmpty()) {
        return QString();
    }
    QString appName = QCoreApplication::applicationName();
    return configDir + "/autostart/" + appName + ".desktop";
}

} // namespace

bool AutoStartManagerPlatform::isAutoStartEnabled()
{
    QString path = getAutostartFilePath();
    if (path.isEmpty()) {
        return false;
    }
    return QFile::exists(path);
}

bool AutoStartManagerPlatform::enableAutoStart(const QString& args)
{
    QString path = getAutostartFilePath();
    if (path.isEmpty()) {
        return false;
    }

    // Inside an AppImage applicationFilePath() is the temporary mount point;
    // $APPIMAGE holds the stable path the desktop entry must reference.
    QString exePath = QString::fromLocal8Bit(qgetenv("APPIMAGE"));
    if (exePath.isEmpty()) {
        exePath = QCoreApplication::applicationFilePath();
    }

    if (!QDir().mkpath(QFileInfo(path).absolutePath())) {
        return false;
    }

    QFile file(path);
    if (!file.open(QIODevice::WriteOnly | QIODevice::Truncate | QIODevice::Text)) {
        return false;
    }
    QTextStream out(&file);
    out << "[Desktop Entry]\n"
        << "Type=Application\n"
        << "Name=" << QCoreApplication::applicationName() << "\n"
        << "Exec=\"" << exePath << "\" " << args << "\n"
        << "X-GNOME-Autostart-enabled=true\n";
    return out.status() == QTextStream::Ok;
}

bool AutoStartManagerPlatform::disableAutoStart()
{
    QString path = getAutostartFilePath();
    if (path.isEmpty()) {
        return false;
    }
    if (!QFile::exists(path)) {
        return true;
    }
    return QFile::remove(path);
}

}