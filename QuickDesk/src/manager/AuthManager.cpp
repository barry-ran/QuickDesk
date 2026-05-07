// Copyright 2026 QuickDesk Authors
#include "AuthManager.h"
#include "ServerManager.h"
#include "infra/http/httprequest.h"
#include "infra/log/log.h"
#include "core/localconfigcenter.h"

#include <QJsonDocument>
#include <QJsonObject>
#include <QUrl>

namespace quickdesk {

namespace {
constexpr int kRequestTimeoutMs = 10000;

// Extract {code, error} from JSON error response
std::pair<QString, QString> parseError(const std::string& data, const std::string& fallback) {
    QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
    if (doc.isObject()) {
        QString code = doc.object()["code"].toString();
        QString msg = doc.object()["error"].toString();
        if (!msg.isEmpty()) return {code, msg};
    }
    return {{}, QString::fromStdString(fallback)};
}
}

AuthManager::AuthManager(ServerManager* serverManager, QObject* parent)
    : QObject(parent)
    , m_serverManager(serverManager)
{
}

void AuthManager::restoreSession()
{
    auto& lcc = core::LocalConfigCenter::instance();
    m_token = lcc.userToken();

    // Always fetch features regardless of login state
    fetchFeatures();

    if (m_token.isEmpty()) {
        LOG_INFO("[AuthManager] No saved token, not logged in");
        return;
    }

    LOG_INFO("[AuthManager] Restoring session from saved token");
    // Validate token by fetching user info
    fetchUserInfo();
}

void AuthManager::fetchFeatures()
{
    QUrl url(httpBaseUrl() + "api/v1/features");
    QList<QPair<QString, QString>> headers;

    infra::HttpRequest::instance().sendGetRequest(
        url, headers, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[AuthManager] Failed to fetch features: {}", errorMsg);
                    return;
                }

                QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
                QJsonObject root = doc.object();
                bool smsEnabled = root["sms_enabled"].toBool(false);
                if (m_smsEnabled != smsEnabled) {
                    m_smsEnabled = smsEnabled;
                    LOG_INFO("[AuthManager] Features: sms_enabled={}", m_smsEnabled);
                    emit smsEnabledChanged();
                }
            });
        });
}

QString AuthManager::httpBaseUrl() const
{
    QString wsUrl = m_serverManager->serverUrl();
    QString httpUrl = wsUrl;
    httpUrl.replace("ws://", "http://");
    httpUrl.replace("wss://", "https://");
    if (!httpUrl.endsWith("/")) httpUrl += "/";
    return httpUrl;
}

void AuthManager::login(const QString& username, const QString& password)
{
    QUrl url(httpBaseUrl() + "api/v1/user/login");
    QList<QPair<QString, QString>> headers;
    headers.append(qMakePair(QStringLiteral("Content-Type"), QStringLiteral("application/json")));

    QJsonObject body;
    body["username"] = username;
    body["password"] = password;
    QByteArray bodyData = QJsonDocument(body).toJson(QJsonDocument::Compact);

    LOG_INFO("[AuthManager] Logging in as: {}", username.toStdString());

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, QString::fromUtf8(bodyData), kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    auto [code, msg] = parseError(data, errorMsg);
                    LOG_WARN("[AuthManager] Login failed: {}", msg.toStdString());
                    emit loginFailed(code, msg);
                    return;
                }

                QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
                QJsonObject root = doc.object();
                m_token = root["token"].toString();
                QJsonObject userObj = root["user"].toObject();
                m_userId = userObj["id"].toInt();
                m_username = userObj["username"].toString();
                m_isLoggedIn = true;

                // Persist token
                auto& lcc = core::LocalConfigCenter::instance();
                lcc.setUserToken(m_token);
                lcc.setUserId(QString::number(m_userId));
                lcc.setUsername(m_username);

                LOG_INFO("[AuthManager] Login successful: {} (id={})", m_username.toStdString(), m_userId);
                emit loginStateChanged();
                emit loginSuccess();
            });
        });
}

void AuthManager::registerUser(const QString& username, const QString& password,
                                const QString& phone, const QString& email,
                                const QString& smsCode)
{
    QUrl url(httpBaseUrl() + "api/v1/user/register");
    QList<QPair<QString, QString>> headers;
    headers.append(qMakePair(QStringLiteral("Content-Type"), QStringLiteral("application/json")));

    QJsonObject body;
    body["username"] = username;
    body["password"] = password;
    if (!phone.isEmpty()) body["phone"] = phone;
    if (!email.isEmpty()) body["email"] = email;
    if (!smsCode.isEmpty()) body["sms_code"] = smsCode;
    QByteArray bodyData = QJsonDocument(body).toJson(QJsonDocument::Compact);

    LOG_INFO("[AuthManager] Registering user: {}", username.toStdString());

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, QString::fromUtf8(bodyData), kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    auto [code, msg] = parseError(data, errorMsg);
                    LOG_WARN("[AuthManager] Registration failed: {}", msg.toStdString());
                    emit registerFailed(code, msg);
                    return;
                }

                LOG_INFO("[AuthManager] Registration successful");
                emit registerSuccess();
            });
        });
}

void AuthManager::sendSmsCode(const QString& phone, const QString& scene)
{
    QUrl url(httpBaseUrl() + "api/v1/sms/send");
    QList<QPair<QString, QString>> headers;
    headers.append(qMakePair(QStringLiteral("Content-Type"), QStringLiteral("application/json")));

    QJsonObject body;
    body["phone"] = phone;
    body["scene"] = scene;
    QByteArray bodyData = QJsonDocument(body).toJson(QJsonDocument::Compact);

    LOG_INFO("[AuthManager] Sending SMS code to {} (scene={})", phone.toStdString(), scene.toStdString());

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, QString::fromUtf8(bodyData), kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    auto [code, msg] = parseError(data, errorMsg);
                    LOG_WARN("[AuthManager] SMS send failed: {}", msg.toStdString());
                    emit smsCodeFailed(code, msg);
                    return;
                }

                LOG_INFO("[AuthManager] SMS code sent successfully");
                emit smsCodeSent();
            });
        });
}

void AuthManager::loginWithSms(const QString& phone, const QString& smsCode)
{
    QUrl url(httpBaseUrl() + "api/v1/user/login-sms");
    QList<QPair<QString, QString>> headers;
    headers.append(qMakePair(QStringLiteral("Content-Type"), QStringLiteral("application/json")));

    QJsonObject body;
    body["phone"] = phone;
    body["sms_code"] = smsCode;
    QByteArray bodyData = QJsonDocument(body).toJson(QJsonDocument::Compact);

    LOG_INFO("[AuthManager] Logging in with SMS: {}", phone.toStdString());

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, QString::fromUtf8(bodyData), kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    auto [code, msg] = parseError(data, errorMsg);
                    LOG_WARN("[AuthManager] SMS login failed: {}", msg.toStdString());
                    emit loginFailed(code, msg);
                    return;
                }

                QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
                QJsonObject root = doc.object();
                m_token = root["token"].toString();
                QJsonObject userObj = root["user"].toObject();
                m_userId = userObj["id"].toInt();
                m_username = userObj["username"].toString();
                m_isLoggedIn = true;

                // Persist token
                auto& lcc = core::LocalConfigCenter::instance();
                lcc.setUserToken(m_token);
                lcc.setUserId(QString::number(m_userId));
                lcc.setUsername(m_username);

                LOG_INFO("[AuthManager] SMS login successful: {} (id={})", m_username.toStdString(), m_userId);
                emit loginStateChanged();
                emit loginSuccess();
            });
        });
}

void AuthManager::logout()
{
    if (m_token.isEmpty()) {
        clearSession();
        return;
    }

    QUrl url(httpBaseUrl() + "api/v1/user/logout");
    QList<QPair<QString, QString>> headers;
    headers.append(qMakePair(QStringLiteral("Authorization"), QStringLiteral("Bearer ") + m_token));

    LOG_INFO("[AuthManager] Logging out user: {}", m_username.toStdString());

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, QString(), kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            Q_UNUSED(statusCode); Q_UNUSED(errorMsg); Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [this]() {
                clearSession();
            });
        });
}

void AuthManager::fetchUserInfo()
{
    if (m_token.isEmpty()) return;

    QUrl url(httpBaseUrl() + "api/v1/user/me");
    QList<QPair<QString, QString>> headers;
    headers.append(qMakePair(QStringLiteral("Authorization"), QStringLiteral("Bearer ") + m_token));

    infra::HttpRequest::instance().sendGetRequest(
        url, headers, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    LOG_WARN("[AuthManager] Token validation failed, clearing session");
                    clearSession();
                    return;
                }

                QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
                QJsonObject userObj = doc.object()["user"].toObject();
                m_userId = userObj["id"].toInt();
                m_username = userObj["username"].toString();
                m_isLoggedIn = true;

                LOG_INFO("[AuthManager] Session restored: {} (id={})", m_username.toStdString(), m_userId);
                emit loginStateChanged();
                emit loginSuccess();
            });
        });
}

void AuthManager::clearSession()
{
    emit loggingOut();

    m_isLoggedIn = false;
    m_username.clear();
    m_userId = 0;
    m_token.clear();

    auto& lcc = core::LocalConfigCenter::instance();
    lcc.setUserToken("");
    lcc.setUserId("");
    lcc.setUsername("");

    LOG_INFO("[AuthManager] Session cleared");
    emit loginStateChanged();
    emit loggedOut();
}

bool AuthManager::isLoggedIn() const { return m_isLoggedIn; }
QString AuthManager::username() const { return m_username; }
uint AuthManager::userId() const { return m_userId; }
QString AuthManager::token() const { return m_token; }
bool AuthManager::smsEnabled() const { return m_smsEnabled; }

} // namespace quickdesk
