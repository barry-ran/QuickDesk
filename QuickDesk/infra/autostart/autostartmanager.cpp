#include "autostartmanager.h"
#include "autostartmanagerplatform.h"

namespace infra {

bool AutoStartManager::isAutoStartEnabled()
{
    return AutoStartManagerPlatform::isAutoStartEnabled();
}

bool AutoStartManager::setAutoStartEnabled(bool enable, const QString& args)
{
    if (enable) {
        return AutoStartManagerPlatform::enableAutoStart(args);
    } else {
        return AutoStartManagerPlatform::disableAutoStart();
    }
}

}
