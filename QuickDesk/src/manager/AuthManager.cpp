// Copyright 2026 QuickDesk Authors
#include "AuthManager.h"
#include "ServerManager.h"
#include "HostManager.h"
#include "infra/http/httprequest.h"
#include "infra/log/log.h"
#include "core/localconfigcenter.h"

#include <QJsonDocument>
#include <QJsonObject>
#include <QUrl>

#ifndef QUICKDESK_API_KEY
#define QUICKDESK_API_KEY ""
#endif

namespace quickdesk {

namespace {
constexpr int kRequestTimeoutMs = 10000;

// Refresh the access token this many seconds before its nominal expiry
// to avoid 401 races. 60s gives plenty of slack for slow networks.
constexpr int kAccessRefreshMarginSeconds = 60;
}

AuthManager::AuthManager(ServerManager* serverManager, QObject* parent)
    : QObject(parent)
    , m_serverManager(serverManager)
{
}

void AuthManager::setHostManager(HostManager* host)
{
    m_hostManager = host;
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

QString AuthManager::httpBaseUrl() const
{
    QString wsUrl = m_serverManager->serverUrl();
    QString httpUrl = wsUrl;
    httpUrl.replace("ws://", "http://");
    httpUrl.replace("wss://", "https://");
    if (!httpUrl.endsWith("/")) httpUrl += "/";
    return httpUrl;
}

QString AuthManager::effectiveApiKey() const
{
    QString runtimeApiKey = core::LocalConfigCenter::instance().apiKey();
    if (!runtimeApiKey.isEmpty()) return runtimeApiKey;
    constexpr const char* kCompileApiKey = QUICKDESK_API_KEY;
    if (kCompileApiKey[0] != '\0') return QString::fromLatin1(kCompileApiKey);
    return QString();
}

QString AuthManager::bearerValue() const
{
    if (m_accessToken.isEmpty()) return QString();
    return QStringLiteral("Bearer ") + m_accessToken;
}

QList<QPair<QString, QString>> AuthManager::publicHeaders() const
{
    QList<QPair<QString, QString>> headers;
    QString apiKey = effectiveApiKey();
    if (!apiKey.isEmpty()) {
        headers.append(qMakePair(QStringLiteral("X-API-Key"), apiKey));
    }
    return headers;
}

QList<QPair<QString, QString>> AuthManager::userAuthHeaders(bool includeContentType) const
{
    QList<QPair<QString, QString>> headers = publicHeaders();
    if (includeContentType) {
        headers.append(qMakePair(QStringLiteral("Content-Type"),
                                  QStringLiteral("application/json")));
    }
    QString bearer = bearerValue();
    if (!bearer.isEmpty()) {
        headers.append(qMakePair(QStringLiteral("Authorization"), bearer));
    }
    return headers;
}

std::pair<QString, QString> AuthManager::parseProblem(const std::string& data,
                                                       const std::string& fallback)
{
    QJsonDocument doc = QJsonDocument::fromJson(QByteArray::fromStdString(data));
    if (doc.isObject()) {
        QJsonObject obj = doc.object();
        // Prefer RFC7807 {code, detail, title} over legacy {error}.
        QString code = obj["code"].toString();
        QString msg = obj["detail"].toString();
        if (msg.isEmpty()) msg = obj["title"].toString();
        if (msg.isEmpty()) msg = obj["error"].toString();
        if (msg.isEmpty()) msg = obj["message"].toString();
        if (!msg.isEmpty() || !code.isEmpty()) return {code, msg};
    }
    return {{}, QString::fromStdString(fallback)};
}

// -----------------------------------------------------------------------
// HTTP helper with 401→refresh→retry-once
// -----------------------------------------------------------------------

void AuthManager::rawRequest(const QString& method,
                              const QUrl& url,
                              const QList<QPair<QString, QString>>& headers,
                              const QString& body,
                              ResponseCallback callback)
{
    auto& http = infra::HttpRequest::instance();
    if (method == QLatin1String("GET")) {
        http.sendGetRequest(url, headers, kRequestTimeoutMs, callback);
    } else if (method == QLatin1String("POST")) {
        http.sendPostRequest(url, headers, body, kRequestTimeoutMs, callback);
    } else if (method == QLatin1String("PUT")) {
        http.sendPutRequest(url, headers, body, kRequestTimeoutMs, callback);
    } else if (method == QLatin1String("PATCH")) {
        http.sendPatchRequest(url, headers, body, kRequestTimeoutMs, callback);
    } else if (method == QLatin1String("DELETE")) {
        http.sendDeleteRequest(url, headers, kRequestTimeoutMs, callback);
    } else {
        LOG_ERROR("[AuthManager] unknown HTTP method: {}", method.toStdString());
        callback(0, "unknown HTTP method", std::string{});
    }
}

void AuthManager::request(const QString& method,
                           const QUrl& url,
                           const QString& body,
                           ResponseCallback callback)
{
    auto headers = userAuthHeaders(!body.isEmpty() || method == QLatin1String("POST")
                                    || method == QLatin1String("PUT")
                                    || method == QLatin1String("PATCH"));

    auto onFirstReply = [this, method, url, body, callback](int statusCode,
                                                              const std::string& errorMsg,
                                                              const std::string& data) {
        QMetaObject::invokeMethod(this, [this, method, url, body, callback, statusCode, errorMsg, data]() {
            if (statusCode != 401 || !m_isLoggedIn) {
                callback(statusCode, errorMsg, data);
                return;
            }
            // Try silent refresh once.
            refreshAccessToken([this, method, url, body, callback](bool ok) {
                if (!ok) {
                    // Session is already cleared by refreshAccessToken.
                    callback(401, "session expired", std::string{});
                    return;
                }
                // Retry with fresh token. Per §2.15, a second 401 after
                // a successful refresh is treated as "session ended" —
                // clear local state and surface the 401 to the caller.
                auto retryHeaders = userAuthHeaders(true);
                rawRequest(method, url, retryHeaders, body,
                           [this, callback](int sc, const std::string& em, const std::string& d) {
                               QMetaObject::invokeMethod(this, [this, callback, sc, em, d]() {
                                   if (sc == 401) {
                                       LOG_WARN("[AuthManager] second 401 after refresh — clearing session");
                                       clearSession();
                                   }
                                   callback(sc, em, d);
                               });
                           });
            });
        });
    };
    rawRequest(method, url, headers, body, onFirstReply);
}

// -----------------------------------------------------------------------
// Restore / bootstrap
// -----------------------------------------------------------------------

void AuthManager::restoreSession()
{
    auto& lcc = core::LocalConfigCenter::instance();
    m_accessToken  = lcc.userToken();
    m_refreshToken = lcc.refreshToken();

    QString accessExp  = lcc.accessTokenExpiresAt();
    QString refreshExp = lcc.refreshTokenExpiresAt();
    m_accessExpiresAt  = QDateTime::fromString(accessExp,  Qt::ISODate);
    m_refreshExpiresAt = QDateTime::fromString(refreshExp, Qt::ISODate);

    // Always fetch features regardless of login state
    fetchFeatures();

    if (m_accessToken.isEmpty() && m_refreshToken.isEmpty()) {
        LOG_INFO("[AuthManager] No saved tokens, not logged in");
        return;
    }

    // If refresh token exists but access is missing/expired → refresh first.
    if (m_refreshToken.isEmpty()) {
        // No refresh — validate existing access token (legacy path).
        // Pre-set m_isLoggedIn optimistically so request()'s 401 refresh
        // branch can engage; clearSession flips it back on failure.
        m_isLoggedIn = !m_accessToken.isEmpty();
        fetchUserInfo();
        return;
    }

    QDateTime now = QDateTime::currentDateTimeUtc();
    bool accessValid = !m_accessToken.isEmpty() && m_accessExpiresAt.isValid() &&
                        now.addSecs(kAccessRefreshMarginSeconds) < m_accessExpiresAt;
    if (accessValid) {
        LOG_INFO("[AuthManager] Access token still valid, fetching user info");
        // Same optimistic flip as above — without it, a 401 on /v1/me
        // wouldn't cascade through request()'s refresh helper because
        // the m_isLoggedIn guard would short-circuit.
        m_isLoggedIn = true;
        fetchUserInfo();
        return;
    }

    LOG_INFO("[AuthManager] Access token near/past expiry, refreshing");
    refreshAccessToken([this](bool ok) {
        if (!ok) return;  // clearSession already emitted loggedOut
        fetchUserInfo();
    });
}

void AuthManager::fetchFeatures()
{
    QUrl url(httpBaseUrl() + "v1/features");
    auto headers = publicHeaders();

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

// -----------------------------------------------------------------------
// Token plumbing
// -----------------------------------------------------------------------

void AuthManager::applyTokens(const QByteArray& responseBody)
{
    QJsonDocument doc = QJsonDocument::fromJson(responseBody);
    QJsonObject root = doc.object();

    m_accessToken       = root["access_token"].toString();
    m_refreshToken      = root["refresh_token"].toString();
    QString accessExp   = root["access_expires_at"].toString();
    QString refreshExp  = root["refresh_expires_at"].toString();
    m_accessExpiresAt   = QDateTime::fromString(accessExp,  Qt::ISODate);
    m_refreshExpiresAt  = QDateTime::fromString(refreshExp, Qt::ISODate);

    QJsonObject userObj = root["user"].toObject();
    if (!userObj.isEmpty()) {
        m_userId   = userObj["id"].toInt();
        m_username = userObj["username"].toString();
    }

    m_isLoggedIn = !m_accessToken.isEmpty();

    auto& lcc = core::LocalConfigCenter::instance();
    lcc.setUserToken(m_accessToken);
    lcc.setRefreshToken(m_refreshToken);
    lcc.setAccessTokenExpiresAt(accessExp);
    lcc.setRefreshTokenExpiresAt(refreshExp);
    if (m_userId != 0) {
        lcc.setUserId(QString::number(m_userId));
    }
    if (!m_username.isEmpty()) {
        lcc.setUsername(m_username);
    }
}

void AuthManager::refreshAccessToken(std::function<void(bool)> done)
{
    if (m_refreshToken.isEmpty()) {
        LOG_WARN("[AuthManager] refresh requested but no refresh token");
        clearSession();
        if (done) done(false);
        return;
    }

    // Coalesce concurrent refreshes so we only make one HTTP call.
    if (m_refreshInFlight) {
        if (done) m_refreshQueue.append(done);
        return;
    }
    m_refreshInFlight = true;
    if (done) m_refreshQueue.append(done);

    QUrl url(httpBaseUrl() + "v1/auth/tokens:refresh");
    auto headers = publicHeaders();
    headers.append(qMakePair(QStringLiteral("Content-Type"),
                              QStringLiteral("application/json")));

    QJsonObject body;
    body["refresh_token"] = m_refreshToken;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    LOG_INFO("[AuthManager] Refreshing access token");

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                bool ok = false;
                if (statusCode == 200 && errorMsg.empty()) {
                    // /v1/auth/tokens:refresh returns just the new tokens
                    // (no user object). applyTokens still handles that.
                    applyTokens(QByteArray::fromStdString(data));
                    LOG_INFO("[AuthManager] Access token refreshed");
                    ok = true;
                } else {
                    // REFRESH_INVALID or expired → hard logout.
                    auto [code, msg] = parseProblem(data, errorMsg);
                    LOG_WARN("[AuthManager] Refresh failed: code={} msg={}",
                             code.toStdString(), msg.toStdString());
                    clearSession();
                }

                auto q = std::move(m_refreshQueue);
                m_refreshQueue.clear();
                m_refreshInFlight = false;
                for (auto& cb : q) {
                    if (cb) cb(ok);
                }
            });
        });
}

// -----------------------------------------------------------------------
// Auth actions
// -----------------------------------------------------------------------

void AuthManager::login(const QString& identifier, const QString& password)
{
    QUrl url(httpBaseUrl() + "v1/auth/sessions");
    auto headers = publicHeaders();
    headers.append(qMakePair(QStringLiteral("Content-Type"),
                              QStringLiteral("application/json")));

    QJsonObject body;
    body["identifier"] = identifier;  // server accepts username/phone/email
    body["password"]   = password;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    LOG_INFO("[AuthManager] Logging in as: {}", identifier.toStdString());

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    auto [code, msg] = parseProblem(data, errorMsg);
                    LOG_WARN("[AuthManager] Login failed: {}", msg.toStdString());
                    emit loginFailed(code, msg);
                    return;
                }
                applyTokens(QByteArray::fromStdString(data));
                LOG_INFO("[AuthManager] Login successful: {} (id={})",
                         m_username.toStdString(), m_userId);
                emit loginStateChanged();
                emit loginSuccess();
            });
        });
}

void AuthManager::loginWithSms(const QString& phone, const QString& smsCode)
{
    QUrl url(httpBaseUrl() + "v1/auth/sessions:sms");
    auto headers = publicHeaders();
    headers.append(qMakePair(QStringLiteral("Content-Type"),
                              QStringLiteral("application/json")));

    QJsonObject body;
    body["phone"]    = phone;
    body["sms_code"] = smsCode;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    LOG_INFO("[AuthManager] Logging in with SMS: {}", phone.toStdString());

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    auto [code, msg] = parseProblem(data, errorMsg);
                    LOG_WARN("[AuthManager] SMS login failed: {}", msg.toStdString());
                    emit loginFailed(code, msg);
                    return;
                }
                applyTokens(QByteArray::fromStdString(data));
                LOG_INFO("[AuthManager] SMS login successful: {} (id={})",
                         m_username.toStdString(), m_userId);
                emit loginStateChanged();
                emit loginSuccess();
            });
        });
}

void AuthManager::sendSmsCode(const QString& phone, const QString& scene)
{
    QUrl url(httpBaseUrl() + "v1/verification-codes");
    auto headers = publicHeaders();
    headers.append(qMakePair(QStringLiteral("Content-Type"),
                              QStringLiteral("application/json")));

    QJsonObject body;
    body["phone"] = phone;
    body["scene"] = scene;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    LOG_INFO("[AuthManager] Sending SMS code to {} (scene={})",
             phone.toStdString(), scene.toStdString());

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    auto [code, msg] = parseProblem(data, errorMsg);
                    LOG_WARN("[AuthManager] SMS send failed: {}", msg.toStdString());
                    emit smsCodeFailed(code, msg);
                    return;
                }
                LOG_INFO("[AuthManager] SMS code sent successfully");
                emit smsCodeSent();
            });
        });
}

void AuthManager::registerUser(const QString& username, const QString& password,
                                const QString& phone, const QString& email,
                                const QString& smsCode)
{
    QUrl url(httpBaseUrl() + "v1/auth/register");
    auto headers = publicHeaders();
    headers.append(qMakePair(QStringLiteral("Content-Type"),
                              QStringLiteral("application/json")));

    QJsonObject body;
    body["username"] = username;
    body["password"] = password;
    if (!phone.isEmpty()) body["phone"] = phone;
    if (!email.isEmpty()) body["email"] = email;
    if (!smsCode.isEmpty()) body["sms_code"] = smsCode;
    QString bodyData = QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact));

    LOG_INFO("[AuthManager] Registering user: {}", username.toStdString());

    infra::HttpRequest::instance().sendPostRequest(
        url, headers, bodyData, kRequestTimeoutMs,
        [this](int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                if (statusCode != 200 || !errorMsg.empty()) {
                    auto [code, msg] = parseProblem(data, errorMsg);
                    LOG_WARN("[AuthManager] Registration failed: {}",
                             msg.toStdString());
                    emit registerFailed(code, msg);
                    return;
                }
                // /v1/auth/register returns the same envelope as
                // /v1/auth/sessions (user + tokens). Apply tokens so the
                // user is immediately logged in.
                applyTokens(QByteArray::fromStdString(data));
                LOG_INFO("[AuthManager] Registration successful: {} (id={})",
                         m_username.toStdString(), m_userId);
                emit registerSuccess();
                emit loginStateChanged();
                emit loginSuccess();
            });
        });
}

// -----------------------------------------------------------------------
// Logout (§2.11 two-step coordination)
// -----------------------------------------------------------------------

void AuthManager::logout(const QString& deviceId)
{
    if (!m_isLoggedIn && m_accessToken.isEmpty() && m_refreshToken.isEmpty()) {
        clearSession();
        return;
    }

    // §2.11 step 1: resolve device_id.
    QString effectiveDeviceId = deviceId;
    if (effectiveDeviceId.isEmpty() && m_hostManager) {
        effectiveDeviceId = m_hostManager->deviceId();
    }
    if (effectiveDeviceId.isEmpty()) {
        // Fallback to the last known device_id persisted by MainController
        // on every hostReady; covers "Qt started, host not ready yet,
        // user clicks logout immediately" (scenario §5 #19).
        effectiveDeviceId = core::LocalConfigCenter::instance().lastDeviceId();
    }

    auto step2DeleteSession = [this]() {
        // §2.11 step 2: revoke current user session regardless of step 1 outcome.
        QUrl url(httpBaseUrl() + "v1/me/sessions/current");
        auto headers = userAuthHeaders(false);
        infra::HttpRequest::instance().sendDeleteRequest(
            url, headers, kRequestTimeoutMs,
            [this](int statusCode, const std::string& errorMsg, const std::string& data) {
                Q_UNUSED(data);
                QMetaObject::invokeMethod(this, [this, statusCode, errorMsg]() {
                    if (statusCode != 200 || !errorMsg.empty()) {
                        // Non-fatal: user intent is logout, so proceed anyway.
                        LOG_WARN("[AuthManager] DELETE /v1/me/sessions/current failed: status={} err={}",
                                 statusCode, errorMsg);
                    } else {
                        LOG_INFO("[AuthManager] Current session revoked");
                    }
                    clearSession();
                });
            });
    };

    if (effectiveDeviceId.isEmpty()) {
        // No device to clear — skip straight to session delete.
        LOG_INFO("[AuthManager] Logout: no device_id available, skipping session/clear");
        step2DeleteSession();
        return;
    }

    // §2.11 step 1: clear logged_in_intent on this host's device row.
    QString path = QStringLiteral("v1/me/devices/") + effectiveDeviceId + QStringLiteral("/session");
    QUrl url(httpBaseUrl() + path);
    auto headers = userAuthHeaders(false);

    LOG_INFO("[AuthManager] Logout step 1: DELETE /v1/me/devices/{}/session",
             effectiveDeviceId.toStdString());

    infra::HttpRequest::instance().sendDeleteRequest(
        url, headers, kRequestTimeoutMs,
        [this, step2DeleteSession, effectiveDeviceId](int statusCode,
                                                        const std::string& errorMsg,
                                                        const std::string& data) {
            Q_UNUSED(data);
            QMetaObject::invokeMethod(this, [statusCode, errorMsg, step2DeleteSession, effectiveDeviceId]() {
                if (statusCode != 200 && statusCode != 404) {
                    // 404 is expected when a stale lastDeviceId no longer
                    // exists server-side; do not treat as fatal.
                    LOG_WARN("[AuthManager] device/session clear failed: status={} err={}",
                             statusCode, errorMsg);
                } else {
                    LOG_INFO("[AuthManager] device/session cleared for {}",
                             effectiveDeviceId.toStdString());
                }
                step2DeleteSession();
            });
        });
}

// -----------------------------------------------------------------------
// fetchUserInfo / clearSession / getters
// -----------------------------------------------------------------------

void AuthManager::fetchUserInfo()
{
    if (m_accessToken.isEmpty()) return;

    QUrl url(httpBaseUrl() + "v1/me");
    // Go through request() so a 401 transparently triggers the
    // refresh-and-retry path; a second 401 after refresh clears the
    // session (request() does this for us) and we still log the failure.
    request("GET", url, QString(),
            [this](int statusCode, const std::string& errorMsg, const std::string& data) {
                QMetaObject::invokeMethod(this, [this, statusCode, errorMsg, data]() {
                    if (statusCode != 200 || !errorMsg.empty()) {
                        LOG_WARN("[AuthManager] Fetch /v1/me failed ({}): {}",
                                 statusCode, errorMsg);
                        if (statusCode == 401) {
                            clearSession();
                        }
                        return;
                    }
                    // §2.2 says /v1/me returns the user object directly
                    // (no enclosing "user" key). Be liberal and accept both.
                    QJsonDocument doc = QJsonDocument::fromJson(
                        QByteArray::fromStdString(data));
                    QJsonObject obj = doc.object();
                    if (obj.contains("user")) obj = obj["user"].toObject();
                    m_userId   = obj["id"].toInt();
                    m_username = obj["username"].toString();
                    m_isLoggedIn = true;

                    auto& lcc = core::LocalConfigCenter::instance();
                    if (m_userId != 0) lcc.setUserId(QString::number(m_userId));
                    if (!m_username.isEmpty()) lcc.setUsername(m_username);

                    LOG_INFO("[AuthManager] Session restored: {} (id={})",
                             m_username.toStdString(), m_userId);
                    emit loginStateChanged();
                    emit loginSuccess();
                });
            });
}

void AuthManager::clearSession()
{
    bool wasLoggedIn = m_isLoggedIn ||
                       !m_accessToken.isEmpty() ||
                       !m_refreshToken.isEmpty();

    m_isLoggedIn = false;
    m_username.clear();
    m_userId = 0;
    m_accessToken.clear();
    m_refreshToken.clear();
    m_accessExpiresAt = QDateTime();
    m_refreshExpiresAt = QDateTime();

    auto& lcc = core::LocalConfigCenter::instance();
    lcc.setUserToken("");
    lcc.setRefreshToken("");
    lcc.setAccessTokenExpiresAt("");
    lcc.setRefreshTokenExpiresAt("");
    lcc.setUserId("");
    lcc.setUsername("");

    if (wasLoggedIn) {
        LOG_INFO("[AuthManager] Session cleared");
        emit loginStateChanged();
        emit loggedOut();
    }
}

bool    AuthManager::isLoggedIn() const { return m_isLoggedIn; }
QString AuthManager::username()   const { return m_username; }
uint    AuthManager::userId()     const { return m_userId; }
QString AuthManager::token()      const { return m_accessToken; }
bool    AuthManager::smsEnabled() const { return m_smsEnabled; }

} // namespace quickdesk
