// Copyright 2026 QuickDesk Authors
// Signaling Server configuration

#ifndef QUICKDESK_MANAGER_SERVERMANAGER_H
#define QUICKDESK_MANAGER_SERVERMANAGER_H

#include <QObject>

namespace quickdesk {

/**
 * @brief Manages signaling server configuration
 * 
 * This class only stores the server URL configuration.
 * The signaling server runs independently on a remote server.
 */
class ServerManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString serverUrl READ serverUrl WRITE setServerUrl NOTIFY serverUrlChanged)
    Q_PROPERTY(QString status READ status NOTIFY statusChanged)

public:
    explicit ServerManager(QObject* parent = nullptr);
    ~ServerManager() override = default;

    // Server URL (e.g., ws://your-server.com:8000)
    QString serverUrl() const;
    void setServerUrl(const QString& url);

    // Status string for UI
    QString status() const;

    // Load/save settings
    void loadSettings();
    void saveSettings();

signals:
    void serverUrlChanged();
    void statusChanged();

private:
    QString m_serverUrl = "ws://localhost:8000";  // Default for development
    QString m_status = "已配置";

    void updateStatus(const QString& status);
};

} // namespace quickdesk

#endif // QUICKDESK_MANAGER_SERVERMANAGER_H
