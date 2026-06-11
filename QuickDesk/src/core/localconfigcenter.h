#pragma once

#include <QObject>
#include <QString>

#include "base/singleton.h"
#include "infra/env/applicationcontext.h"

namespace core {

#define LCC_FUNCTION_DEC_FLOAT(getName, setName, defaultValue) LCC_FUNCTION_DEC(float, getName, setName, defaultValue)
#define LCC_FUNCTION_DEC_INT(getName, setName, defaultValue) LCC_FUNCTION_DEC(int, getName, setName, defaultValue)
#define LCC_FUNCTION_DEC_BOOL(getName, setName, defaultValue) LCC_FUNCTION_DEC(bool, getName, setName, defaultValue)
#define LCC_FUNCTION_DEC_STRING(getName, setName, defaultValue) LCC_FUNCTION_DEC(QString, getName, setName, defaultValue)
#define LCC_FUNCTION_DEC(type, getName, setName, defaultValue) \
    type getName(const type& value = defaultValue);            \
    void set##setName(const type& value);                      \
    Q_SIGNAL void signal##setName##Changed(const type& value);

class AppConfigDataBase;
class LocalConfigCenter : public QObject, public base::Singleton<LocalConfigCenter> {
    Q_OBJECT
public:
    LocalConfigCenter(QObject* parent = nullptr);
    ~LocalConfigCenter();

    bool init();

    LCC_FUNCTION_DEC_BOOL(groupWindowVerticalScreen, GroupWindowVerticalScreen, true)

    LCC_FUNCTION_DEC_INT(accessCodeRefreshInterval, AccessCodeRefreshInterval, 120)  // minutes: never=-1, 1, 30, 120(default), 360, 720, 1440
    LCC_FUNCTION_DEC_INT(darkTheme, DarkTheme, 1)  // 0=Light, 1=Dark, default=Dark

    LCC_FUNCTION_DEC_STRING(language, Language, "Auto")
    LCC_FUNCTION_DEC_STRING(savedAccessCode, SavedAccessCode, "")  // Saved access code for "never refresh" mode
    LCC_FUNCTION_DEC_STRING(preferredVideoCodec, PreferredVideoCodec, "AV1")  // Video codec: "H264", "VP8", "VP9", "AV1"
    LCC_FUNCTION_DEC_STRING(accessCodeNextRefreshTime, AccessCodeNextRefreshTime, "")  // ISO datetime of next scheduled refresh
    LCC_FUNCTION_DEC_STRING(signalingServerUrl, SignalingServerUrl, "ws://qdsignaling.quickcoder.cc:8000") // Signaling server URL
    LCC_FUNCTION_DEC_STRING(turnServersJson, TurnServersJson, "") // TURN/STUN servers configuration in JSON format
    LCC_FUNCTION_DEC_STRING(apiKey, ApiKey, "") // API Key for signaling server authentication (runtime override, takes precedence over compile-time key)

    LCC_FUNCTION_DEC_BOOL(trayMessageShown, TrayMessageShown, false) // Whether the tray minimize hint has been shown

    LCC_FUNCTION_DEC_BOOL(skillHostEnabled, SkillHostEnabled, true) // AI Agent on/off
    LCC_FUNCTION_DEC_STRING(mcpTransportMode, McpTransportMode, "stdio") // MCP transport: "stdio" or "http"
    LCC_FUNCTION_DEC_STRING(extraSkillsDirs, ExtraSkillsDirs, "") // JSON array of user-added skills directories
    LCC_FUNCTION_DEC_STRING(trustConfirmMode, TrustConfirmMode, "manual") // "manual" = show dialog, "auto_approve" = approve all

    // Privacy screen
    LCC_FUNCTION_DEC_BOOL(autoPrivacyScreenOnConnect, AutoPrivacyScreenOnConnect, false) // Auto-enable privacy screen when a client connects

    // User authentication
    LCC_FUNCTION_DEC_STRING(userToken, UserToken, "")    // Persisted user access token (short-lived; used to bootstrap restoreSession only)
    LCC_FUNCTION_DEC_STRING(refreshToken, RefreshToken, "")    // Persisted refresh token (30d) — refresh via POST /v1/auth/tokens:refresh
    LCC_FUNCTION_DEC_STRING(accessTokenExpiresAt, AccessTokenExpiresAt, "")  // ISO8601 UTC expiry of current access token
    LCC_FUNCTION_DEC_STRING(refreshTokenExpiresAt, RefreshTokenExpiresAt, "") // ISO8601 UTC expiry of current refresh token
    LCC_FUNCTION_DEC_STRING(userId, UserId, "")          // Persisted user ID
    LCC_FUNCTION_DEC_STRING(username, Username, "")      // Persisted username
    LCC_FUNCTION_DEC_STRING(lastDeviceId, LastDeviceId, "") // Last known device_id of local host (§2.11 two-step logout bootstrap fallback)

private:
    AppConfigDataBase* m_configDatabase = nullptr;
};

}
