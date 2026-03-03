#pragma once

#include <QString>

namespace infra {

/**
 * @brief Auto startup manager - cross-platform interface
 *
 * Provides two main interfaces:
 * 1. isAutoStartEnabled() - Check if auto startup is enabled
 * 2. setAutoStartEnabled() - Enable or disable auto startup
 */
class AutoStartManager {
public:
    /**
     * @brief Check if auto startup is enabled
     * @return true if auto startup is enabled, false otherwise
     */
    static bool isAutoStartEnabled();

    /**
     * @brief Enable or disable auto startup
     * @param enable true to enable, false to disable
     * @param args Optional arguments to pass when starting the application
     * @return true if operation succeeded, false otherwise
     */
    static bool setAutoStartEnabled(bool enable, const QString& args = QString());
};

}
