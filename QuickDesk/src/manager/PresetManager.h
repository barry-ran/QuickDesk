// Copyright 2026 QuickDesk Authors
// Server preset configuration manager

#ifndef QUICKDESK_MANAGER_PRESETMANAGER_H
#define QUICKDESK_MANAGER_PRESETMANAGER_H

#include <QObject>
#include <QTimer>
#include <QVariantList>
#include <QDateTime>

namespace quickdesk {

class ServerManager;

class PresetManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString announcement READ announcement NOTIFY announcementChanged)
    Q_PROPERTY(QVariantList links READ links NOTIFY linksChanged)
    Q_PROPERTY(bool presetLoaded READ presetLoaded NOTIFY presetLoadedChanged)

public:
    explicit PresetManager(ServerManager* serverManager, QObject* parent = nullptr);
    ~PresetManager() override = default;

    void start();
    void stop();

    QString announcement() const;
    QVariantList links() const;
    bool presetLoaded() const;

signals:
    void announcementChanged();
    void linksChanged();
    void presetLoadedChanged();
    void presetLoadFailed(const QString& error);
    void forceUpgradeRequired(const QString& minVersion);

private:
    void fetchPreset();
    void onPresetResponse(int statusCode, const std::string& errorMsg, const std::string& data);
    void parsePresetData(const QByteArray& data);
    QString currentLanguage() const;
    static bool isVersionLower(const QString& current, const QString& required);

    ServerManager* m_serverManager;
    QTimer m_pollTimer;
    QTimer m_initialTimer;
    QDateTime m_firstRequestTime;
    bool m_presetLoaded = false;
    bool m_started = false;

    QString m_announcement;
    QVariantList m_links;
    QString m_lastMinVersion;
};

} // namespace quickdesk

#endif // QUICKDESK_MANAGER_PRESETMANAGER_H
