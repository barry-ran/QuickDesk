// Copyright 2026 QuickDesk Authors

#include "ServerManager.h"
#include "infra/log/log.h"
#include "../core/localconfigcenter.h"
#include <QCoreApplication>

namespace quickdesk {

namespace {

constexpr char kLegacyOfficialServerUrl[] = "ws://qdsignaling.quickcoder.cc:8060";
constexpr char kOfficialServerUrl[] = "wss://qdsignaling.quickcoder.cc";

} // namespace

ServerManager::ServerManager(QObject* parent)
    : QObject(parent)
{
    loadSettings();
}

QString ServerManager::serverUrl() const
{
    return m_serverUrl;
}

void ServerManager::setServerUrl(const QString& url)
{
    if (m_serverUrl != url) {
        m_serverUrl = url;
        saveSettings();
        emit serverUrlChanged();
        LOG_INFO("Server URL changed to: {}", url.toStdString());
    }
}

QString ServerManager::status() const
{
    return m_status;
}

void ServerManager::loadSettings()
{
    m_serverUrl = core::LocalConfigCenter::instance().signalingServerUrl();
    if (m_serverUrl == QLatin1String(kLegacyOfficialServerUrl)) {
        m_serverUrl = QLatin1String(kOfficialServerUrl);
        saveSettings();
        LOG_INFO("Migrated signaling server URL to: {}", m_serverUrl.toStdString());
    }
    LOG_INFO("Loaded signaling server URL: {}", m_serverUrl.toStdString());
}

void ServerManager::saveSettings()
{
    core::LocalConfigCenter::instance().setSignalingServerUrl(m_serverUrl);
    LOG_INFO("Saved signaling server URL: {}", m_serverUrl.toStdString());
}

void ServerManager::updateStatus(const QString& status)
{
    if (m_status != status) {
        m_status = status;
        emit statusChanged();
    }
}

} // namespace quickdesk
