#pragma once

#include <QString>

namespace infra {

/**
 * @brief Platform-specific auto startup implementation
 */
class AutoStartManagerPlatform {
public:
    static bool isAutoStartEnabled();
    static bool enableAutoStart(const QString& args);
    static bool disableAutoStart();
};

}
