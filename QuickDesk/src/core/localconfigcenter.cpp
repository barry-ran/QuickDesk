#include "localconfigcenter.h"

#include "core/db/appconfigdatabase.h"

namespace core {

#define LCC_FUNCTION_IMP_FLOAT(getName, setName)                                                 \
    float LocalConfigCenter::getName(const float& defaultValue)                                  \
    {                                                                                            \
        QString value = m_configDatabase->get(#getName);                                         \
        if (value.isEmpty()) {                                                                   \
            return defaultValue;                                                                 \
        }                                                                                        \
        return value.toFloat();                                                                  \
    }                                                                                            \
                                                                                                 \
    void LocalConfigCenter::set##setName(const float& value)                                     \
    {                                                                                            \
        if (!m_configDatabase->get(#getName).isEmpty() && qFuzzyCompare(getName(0.0f), value)) { \
            return;                                                                              \
        }                                                                                        \
                                                                                                 \
        m_configDatabase->set(#getName, QString::number(value));                                 \
        Q_EMIT signal##setName##Changed(value);                                                  \
    }

#define LCC_FUNCTION_IMP_INT(getName, setName)                                   \
    int LocalConfigCenter::getName(const int& defaultValue)                      \
    {                                                                            \
        QString value = m_configDatabase->get(#getName);                         \
        if (value.isEmpty()) {                                                   \
            return defaultValue;                                                 \
        }                                                                        \
        return value.toInt();                                                    \
    }                                                                            \
                                                                                 \
    void LocalConfigCenter::set##setName(const int& value)                       \
    {                                                                            \
        if (!m_configDatabase->get(#getName).isEmpty() && getName(0) == value) { \
            return;                                                              \
        }                                                                        \
                                                                                 \
        m_configDatabase->set(#getName, QString::number(value));                 \
        Q_EMIT signal##setName##Changed(value);                                  \
    }

#define LCC_FUNCTION_IMP_BOOL(getName, setName)                                      \
    bool LocalConfigCenter::getName(const bool& defaultValue)                        \
    {                                                                                \
        QString value = m_configDatabase->get(#getName);                             \
        if (value.isEmpty()) {                                                       \
            return defaultValue;                                                     \
        }                                                                            \
        return value.toInt();                                                        \
    }                                                                                \
                                                                                     \
    void LocalConfigCenter::set##setName(const bool& value)                          \
    {                                                                                \
        if (!m_configDatabase->get(#getName).isEmpty() && getName(false) == value) { \
            return;                                                                  \
        }                                                                            \
                                                                                     \
        m_configDatabase->set(#getName, QString::number(value));                     \
        Q_EMIT signal##setName##Changed(value);                                      \
    }

#define LCC_FUNCTION_IMP_STRING(getName, setName)                                 \
    QString LocalConfigCenter::getName(const QString& defaultValue)               \
    {                                                                             \
        QString value = m_configDatabase->get(#getName);                          \
        if (value.isEmpty()) {                                                    \
            return defaultValue;                                                  \
        }                                                                         \
        return value;                                                             \
    }                                                                             \
                                                                                  \
    void LocalConfigCenter::set##setName(const QString& value)                    \
    {                                                                             \
        if (!m_configDatabase->get(#getName).isEmpty() && getName("") == value) { \
            return;                                                               \
        }                                                                         \
                                                                                  \
        m_configDatabase->set(#getName, value);                                   \
        Q_EMIT signal##setName##Changed(value);                                   \
    }

LocalConfigCenter::LocalConfigCenter(QObject* parent)
    : QObject(parent)
{
    m_configDatabase = new AppConfigDataBase();
}

LocalConfigCenter::~LocalConfigCenter()
{
    delete m_configDatabase;
    m_configDatabase = nullptr;
}

bool LocalConfigCenter::init()
{
    return m_configDatabase->init();
}

LCC_FUNCTION_IMP_BOOL(groupWindowVerticalScreen, GroupWindowVerticalScreen);

LCC_FUNCTION_IMP_INT(accessCodeRefreshInterval, AccessCodeRefreshInterval);
LCC_FUNCTION_IMP_INT(darkTheme, DarkTheme);

LCC_FUNCTION_IMP_STRING(language, Language);
LCC_FUNCTION_IMP_STRING(savedAccessCode, SavedAccessCode);
LCC_FUNCTION_IMP_STRING(preferredVideoCodec, PreferredVideoCodec);
LCC_FUNCTION_IMP_STRING(accessCodeNextRefreshTime, AccessCodeNextRefreshTime);
LCC_FUNCTION_IMP_STRING(signalingServerUrl, SignalingServerUrl);
LCC_FUNCTION_IMP_STRING(turnServersJson, TurnServersJson);
LCC_FUNCTION_IMP_STRING(apiKey, ApiKey);

LCC_FUNCTION_IMP_BOOL(trayMessageShown, TrayMessageShown);

LCC_FUNCTION_IMP_BOOL(skillHostEnabled, SkillHostEnabled);
LCC_FUNCTION_IMP_STRING(mcpTransportMode, McpTransportMode);
LCC_FUNCTION_IMP_STRING(extraSkillsDirs, ExtraSkillsDirs);
LCC_FUNCTION_IMP_STRING(trustConfirmMode, TrustConfirmMode);

LCC_FUNCTION_IMP_BOOL(autoPrivacyScreenOnConnect, AutoPrivacyScreenOnConnect);

LCC_FUNCTION_IMP_STRING(userToken, UserToken);
LCC_FUNCTION_IMP_STRING(refreshToken, RefreshToken);
LCC_FUNCTION_IMP_STRING(accessTokenExpiresAt, AccessTokenExpiresAt);
LCC_FUNCTION_IMP_STRING(refreshTokenExpiresAt, RefreshTokenExpiresAt);
LCC_FUNCTION_IMP_STRING(userId, UserId);
LCC_FUNCTION_IMP_STRING(username, Username);
LCC_FUNCTION_IMP_STRING(lastDeviceId, LastDeviceId);

}
