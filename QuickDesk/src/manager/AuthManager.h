// Copyright 2026 QuickDesk Authors
#ifndef QUICKDESK_MANAGER_AUTHMANAGER_H
#define QUICKDESK_MANAGER_AUTHMANAGER_H

#include <QObject>
#include <QString>
#include <QDateTime>
#include <functional>

namespace quickdesk {

class ServerManager;
class HostManager;

/**
 * AuthManager — Qt client side of the v1 auth surface (§2.1 / §2.2).
 *
 *   Access token   2h, used as `Authorization: Bearer` on /v1/me/*
 *   Refresh token  30d, used ONLY on POST /v1/auth/tokens:refresh (rotation)
 *
 * Both tokens are persisted via LocalConfigCenter so restoreSession() on
 * next launch can run a silent refresh before any user-visible UI.
 *
 * 401 interception: CloudDeviceManager calls `request(...)` below, which
 * retries once after a silent refresh. On refresh failure the session is
 * cleared and `loggedOut()` fires (→ QML pops back to the login page).
 *
 * Two-step logout (§2.11): logout() first DELETEs
 *   /v1/me/devices/{deviceId}/session
 * and then DELETEs /v1/me/sessions/current. Any single step may fail —
 * the user's intent is "log out", so local session is always cleared at
 * the end.
 */
class AuthManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(bool isLoggedIn READ isLoggedIn NOTIFY loginStateChanged)
    Q_PROPERTY(QString username READ username NOTIFY loginStateChanged)
    Q_PROPERTY(uint userId READ userId NOTIFY loginStateChanged)
    Q_PROPERTY(QString token READ token NOTIFY loginStateChanged)
    Q_PROPERTY(bool smsEnabled READ smsEnabled NOTIFY smsEnabledChanged)

public:
    explicit AuthManager(ServerManager* serverManager, QObject* parent = nullptr);
    ~AuthManager() override = default;

    // HostManager is resolved late (MainController wires it after construction)
    // so logout() has a way to fetch the live device_id from the host process.
    void setHostManager(HostManager* host);

    // Restore session state from persisted tokens. If refresh token is
    // present, runs a silent refresh before fetching user profile.
    void restoreSession();

    // Fetch server feature flags (sms_enabled, etc.). Used by login UI to
    // decide whether to show the SMS login tab.
    Q_INVOKABLE void fetchFeatures();

    Q_INVOKABLE void login(const QString& identifier, const QString& password);
    Q_INVOKABLE void loginWithSms(const QString& phone, const QString& smsCode);
    Q_INVOKABLE void sendSmsCode(const QString& phone, const QString& scene);
    Q_INVOKABLE void registerUser(const QString& username, const QString& password,
                                   const QString& phone, const QString& email,
                                   const QString& smsCode = QString());
    // §2.11 two-step logout. deviceId is optional; when empty, falls back
    // to HostManager::deviceId() then LocalConfigCenter::lastDeviceId().
    Q_INVOKABLE void logout(const QString& deviceId = QString());
    Q_INVOKABLE void fetchUserInfo();

    bool isLoggedIn() const;
    QString username() const;
    uint userId() const;
    QString token() const;
    bool smsEnabled() const;

    // ----- shared HTTP helper for user-scoped calls -----

    // Base URL (HTTP), guarantees trailing slash.
    QString httpBaseUrl() const;

    // Standard headers for user-scoped calls: Content-Type + Bearer
    // access_token (when logged in) + X-API-Key (runtime override first,
    // else compile-time injected QUICKDESK_API_KEY).
    QList<QPair<QString, QString>> userAuthHeaders(bool includeContentType = true) const;

    // Headers with X-API-Key only (no Bearer). Used for unauthenticated
    // public calls like /v1/features and /v1/preset.
    QList<QPair<QString, QString>> publicHeaders() const;

    // Perform a GET/POST/PUT/PATCH/DELETE with automatic 401→refresh→retry-once.
    // `method` ∈ {"GET","POST","PUT","PATCH","DELETE"}. `body` ignored for GET/DELETE.
    using ResponseCallback = std::function<void(int statusCode,
                                                  const std::string& errorMsg,
                                                  const std::string& data)>;
    void request(const QString& method,
                 const QUrl& url,
                 const QString& body,
                 ResponseCallback callback);

signals:
    void loginStateChanged();
    void smsEnabledChanged();
    void loginSuccess();
    void loginFailed(const QString& errorCode, const QString& errorMsg);
    void registerSuccess();
    void registerFailed(const QString& errorCode, const QString& errorMsg);
    void loggedOut();
    void smsCodeSent();
    void smsCodeFailed(const QString& errorCode, const QString& errorMsg);

private:
    // Issue a session from /v1/auth/sessions(-sms|register) response JSON.
    void applyTokens(const QByteArray& responseBody);

    // Refresh silently. Calls `done(true)` on success, `done(false)` on
    // failure (in which case session is already cleared).
    void refreshAccessToken(std::function<void(bool)> done);

    // Clear local session + emit loggedOut. Safe to call multiple times.
    void clearSession();

    // Perform a single raw HTTP request (no refresh logic).
    void rawRequest(const QString& method,
                    const QUrl& url,
                    const QList<QPair<QString, QString>>& headers,
                    const QString& body,
                    ResponseCallback callback);

    // Extract RFC7807 {code, detail/title} or fallback to errorMsg.
    static std::pair<QString, QString> parseProblem(const std::string& data,
                                                     const std::string& fallback);

    // Construct the "Bearer access_token" header value (empty if no token).
    QString bearerValue() const;

    // Helper: pick the X-API-Key (runtime override > compile-time > empty).
    QString effectiveApiKey() const;

    ServerManager* m_serverManager;
    HostManager*   m_hostManager = nullptr;

    bool    m_isLoggedIn = false;
    bool    m_smsEnabled = false;
    QString m_username;
    uint    m_userId = 0;
    QString m_accessToken;
    QString m_refreshToken;
    QDateTime m_accessExpiresAt;
    QDateTime m_refreshExpiresAt;

    // Prevent concurrent refresh storms: queue callbacks while a refresh
    // is in flight.
    bool m_refreshInFlight = false;
    QList<std::function<void(bool)>> m_refreshQueue;
};

} // namespace quickdesk

#endif // QUICKDESK_MANAGER_AUTHMANAGER_H
