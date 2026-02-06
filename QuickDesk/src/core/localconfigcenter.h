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

    LCC_FUNCTION_DEC_BOOL(groupWindowVerticalScreen, GroupWindowVerticalScreen, true);

    LCC_FUNCTION_DEC_INT(accessCodeRefreshInterval, AccessCodeRefreshInterval, 120);  // minutes: never=-1, 1, 30, 120(default), 360, 720, 1440
    LCC_FUNCTION_DEC_INT(darkTheme, DarkTheme, 1);  // 0=Light, 1=Dark, default=Dark

    LCC_FUNCTION_DEC_STRING(language, Language, "Auto");
    LCC_FUNCTION_DEC_STRING(savedAccessCode, SavedAccessCode, "");  // Saved access code for "never refresh" mode
    LCC_FUNCTION_DEC_STRING(signalingServerUrl, SignalingServerUrl, "ws://qdsignaling.quickcoder.cc:8000");  // Signaling server URL
    LCC_FUNCTION_DEC_STRING(turnServersJson, TurnServersJson, "");  // TURN/STUN servers configuration in JSON format

private:
    AppConfigDataBase* m_configDatabase = nullptr;
};

}
