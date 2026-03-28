// Copyright 2026 QuickDesk Authors

#include "PresetManager.h"
#include "ServerManager.h"
#include "infra/http/httprequest.h"
#include "infra/log/log.h"
#include "../language/languagemanage.h"

#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QUrl>

#ifndef QUICKDESK_API_KEY
#define QUICKDESK_API_KEY ""
#endif

namespace quickdesk {

namespace {
constexpr int kInitialDelayMs = 2000;
constexpr int kPollIntervalMs = 60000;
constexpr int kRequestTimeoutMs = 10000;
constexpr int kLoadDeadlineMinutes = 10;
}

PresetManager::PresetManager(ServerManager* serverManager, QObject* parent)
    : QObject(parent)
    , m_serverManager(serverManager)
{
    m_pollTimer.setInterval(kPollIntervalMs);
    connect(&m_pollTimer, &QTimer::timeout, this, &PresetManager::fetchPreset);

    m_initialTimer.setSingleShot(true);
    m_initialTimer.setInterval(kInitialDelayMs);
    connect(&m_initialTimer, &QTimer::timeout, this, [this]() {
        m_firstRequestTime = QDateTime::currentDateTime();
        fetchPreset();
        m_pollTimer.start();
    });
}

void PresetManager::start()
{
    if (m_started) return;
    m_started = true;
    m_initialTimer.start();
    LOG_INFO("PresetManager started, first request in {}ms", kInitialDelayMs);
}

void PresetManager::stop()
{
    m_started = false;
    m_initialTimer.stop();
    m_pollTimer.stop();
    LOG_INFO("PresetManager stopped");
}

QString PresetManager::announcement() const { return m_announcement; }
QVariantList PresetManager::links() const { return m_links; }
bool PresetManager::presetLoaded() const { return m_presetLoaded; }
QString PresetManager::webclientUrl() const { return m_webclientUrl; }

void PresetManager::fetchPreset()
{
    QString wsUrl = m_serverManager->serverUrl();
    QString httpUrl = wsUrl;
    httpUrl.replace("ws://", "http://");
    httpUrl.replace("wss://", "https://");
    if (!httpUrl.endsWith("/")) httpUrl += "/";
    httpUrl += "api/v1/preset";

    QUrl url(httpUrl);
    QList<QPair<QString, QString>> headers;
    constexpr const char* kApiKey = QUICKDESK_API_KEY;
    if (kApiKey[0] != '\0') {
        headers.append(qMakePair(QStringLiteral("X-API-Key"), QString::fromLatin1(kApiKey)));
    }

    LOG_INFO("Fetching preset from: {}", httpUrl.toStdString());

    infra::HttpRequest::instance().sendGetRequest(
        url, headers, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                onPresetResponse(statusCode, errorMsg, data);
            });
        });
}

void PresetManager::onPresetResponse(int statusCode, const std::string& errorMsg, const std::string& data)
{
    if (statusCode != 200 || !errorMsg.empty()) {
        LOG_WARN("Preset request failed: status={}, error={}", statusCode, errorMsg);

        if (!m_presetLoaded && m_firstRequestTime.isValid()) {
            auto elapsed = m_firstRequestTime.secsTo(QDateTime::currentDateTime());
            if (elapsed >= kLoadDeadlineMinutes * 60) {
                LOG_ERROR("Preset load deadline exceeded ({} minutes), emitting presetLoadFailed", kLoadDeadlineMinutes);
                m_pollTimer.stop();
                emit presetLoadFailed(tr("Unable to connect to server for %1 minutes").arg(kLoadDeadlineMinutes));
            }
        }
        return;
    }

    LOG_INFO("Preset response received, data size={}", data.size());
    parsePresetData(QByteArray::fromStdString(data));
}

void PresetManager::parsePresetData(const QByteArray& data)
{
    QJsonParseError parseError;
    QJsonDocument doc = QJsonDocument::fromJson(data, &parseError);
    if (parseError.error != QJsonParseError::NoError || !doc.isObject()) {
        LOG_WARN("Preset JSON parse error: {}", parseError.errorString().toStdString());
        return;
    }

    QJsonObject root = doc.object();
    QString lang = currentLanguage();

    // Parse announcement
    QJsonObject noticeObj = root["notice"].toObject();
    QString newAnnouncement = noticeObj[lang].toString();
    if (newAnnouncement.isEmpty() && lang != "en_US") {
        newAnnouncement = noticeObj["en_US"].toString();
    }
    if (m_announcement != newAnnouncement) {
        m_announcement = newAnnouncement;
        emit announcementChanged();
        LOG_INFO("Announcement updated (lang={}): {}", lang.toStdString(),
                 m_announcement.left(50).toStdString());
    }

    // Parse links
    QJsonObject linksObj = root["links"].toObject();
    QJsonArray linksArray = linksObj[lang].toArray();
    if (linksArray.isEmpty() && lang != "en_US") {
        linksArray = linksObj["en_US"].toArray();
    }
    QVariantList newLinks;
    for (const QJsonValue& val : linksArray) {
        QJsonObject linkObj = val.toObject();
        QVariantMap link;
        link["icon"] = linkObj["icon"].toString();
        link["text"] = linkObj["text"].toString();
        link["url"] = linkObj["url"].toString();
        newLinks.append(link);
    }
    if (m_links != newLinks) {
        m_links = newLinks;
        emit linksChanged();
        LOG_INFO("Links updated: {} items", m_links.size());
    }

    // Parse webclient_url
    QString webclientUrl = root["webclient_url"].toString();
    if (m_webclientUrl != webclientUrl) {
        m_webclientUrl = webclientUrl;
        emit webclientUrlChanged();
        LOG_INFO("WebClient URL updated: {}", m_webclientUrl.toStdString());
    }

    // Mark as loaded
    if (!m_presetLoaded) {
        m_presetLoaded = true;
        emit presetLoadedChanged();
        LOG_INFO("Preset loaded successfully");
    }

    // Check min_version
    QString minVersion = root["min_version"].toString();
    if (!minVersion.isEmpty() && minVersion != m_lastMinVersion) {
        m_lastMinVersion = minVersion;
        QString currentVersion = QString(APP_VERSION_STR);
        LOG_INFO("Version check: current={}, min_required={}", currentVersion.toStdString(), minVersion.toStdString());
        if (isVersionLower(currentVersion, minVersion)) {
            LOG_WARN("Current version {} is lower than min required {}, force upgrade",
                     currentVersion.toStdString(), minVersion.toStdString());
            m_pollTimer.stop();
            emit forceUpgradeRequired(minVersion);
        }
    }
}

QString PresetManager::currentLanguage() const
{
    QString lang = LanguageManage::instance().getCurrentRealLanguage();
    if (lang != "zh_CN" && lang != "en_US") {
        return "en_US";
    }
    return lang;
}

bool PresetManager::isVersionLower(const QString& current, const QString& required)
{
    QStringList curParts = current.split('.');
    QStringList reqParts = required.split('.');

    int maxLen = qMax(curParts.size(), reqParts.size());
    for (int i = 0; i < maxLen; ++i) {
        int curNum = (i < curParts.size()) ? curParts[i].toInt() : 0;
        int reqNum = (i < reqParts.size()) ? reqParts[i].toInt() : 0;
        if (curNum < reqNum) return true;
        if (curNum > reqNum) return false;
    }
    return false;
}

} // namespace quickdesk
