#pragma once

#include <memory>

#include <QString>

#include "base/singleton.h"
#include "spdlog/spdlog.h"

#ifdef QT_DEBUG
#define LOG_TRACE(...)                                                     \
    if (infra::Log::instance().logger()) {                                 \
        SPDLOG_LOGGER_TRACE(infra::Log::instance().logger(), __VA_ARGS__); \
        SPDLOG_LOGGER_INFO(spdlog::default_logger(), __VA_ARGS__);         \
    }
#define LOG_DEBUG(...)                                                     \
    if (infra::Log::instance().logger()) {                                 \
        SPDLOG_LOGGER_DEBUG(infra::Log::instance().logger(), __VA_ARGS__); \
        SPDLOG_LOGGER_DEBUG(spdlog::default_logger(), __VA_ARGS__);        \
    }
#define LOG_INFO(...)                                                     \
    if (infra::Log::instance().logger()) {                                \
        SPDLOG_LOGGER_INFO(infra::Log::instance().logger(), __VA_ARGS__); \
        SPDLOG_LOGGER_INFO(spdlog::default_logger(), __VA_ARGS__);        \
    }
#define LOG_WARN(...)                                                     \
    if (infra::Log::instance().logger()) {                                \
        SPDLOG_LOGGER_WARN(infra::Log::instance().logger(), __VA_ARGS__); \
        SPDLOG_LOGGER_WARN(spdlog::default_logger(), __VA_ARGS__);        \
    }
#define LOG_ERROR(...)                                                     \
    if (infra::Log::instance().logger()) {                                 \
        SPDLOG_LOGGER_ERROR(infra::Log::instance().logger(), __VA_ARGS__); \
        SPDLOG_LOGGER_ERROR(spdlog::default_logger(), __VA_ARGS__);        \
    }
#define LOG_CRITICAL(...)                                                     \
    if (infra::Log::instance().logger()) {                                    \
        SPDLOG_LOGGER_CRITICAL(infra::Log::instance().logger(), __VA_ARGS__); \
        SPDLOG_LOGGER_CRITICAL(spdlog::default_logger(), __VA_ARGS__);        \
    }
#else
#define LOG_TRACE(...)                                                     \
    if (infra::Log::instance().logger()) {                                 \
        SPDLOG_LOGGER_TRACE(infra::Log::instance().logger(), __VA_ARGS__); \
    }
#define LOG_DEBUG(...)                                                     \
    if (infra::Log::instance().logger()) {                                 \
        SPDLOG_LOGGER_DEBUG(infra::Log::instance().logger(), __VA_ARGS__); \
    }
#define LOG_INFO(...)                                                     \
    if (infra::Log::instance().logger()) {                                \
        SPDLOG_LOGGER_INFO(infra::Log::instance().logger(), __VA_ARGS__); \
    }
#define LOG_WARN(...)                                                     \
    if (infra::Log::instance().logger()) {                                \
        SPDLOG_LOGGER_WARN(infra::Log::instance().logger(), __VA_ARGS__); \
    }
#define LOG_ERROR(...)                                                     \
    if (infra::Log::instance().logger()) {                                 \
        SPDLOG_LOGGER_ERROR(infra::Log::instance().logger(), __VA_ARGS__); \
    }
#define LOG_CRITICAL(...)                                                     \
    if (infra::Log::instance().logger()) {                                    \
        SPDLOG_LOGGER_CRITICAL(infra::Log::instance().logger(), __VA_ARGS__); \
    }
#endif

namespace infra {

class Log : public base::Singleton<Log> {
public:
    bool init(const QString& path);
    std::shared_ptr<spdlog::logger> logger();

private:
    std::shared_ptr<spdlog::logger> m_logger;
};

}
