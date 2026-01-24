#include "log.h"

#include <QDir>
#include <QFileInfo>

#include "spdlog/sinks/daily_file_sink.h"
#include "spdlog/sinks/msvc_sink.h"

namespace infra {

bool Log::init(const QString& path)
{
    spdlog::set_pattern("[%Y-%m-%d %H:%M:%S.%e] [%n] [%P] [%t] [%l] [%s:%#] %v");
    spdlog::set_level(spdlog::level::trace);

    QString filePath = path + QDir::separator() + "app.log";
    // toLocal8Bit修复windows中文用户名导致崩溃 https://github.com/gabime/spdlog/issues/2510
    m_logger = spdlog::daily_logger_mt("app", filePath.toLocal8Bit().data(), 0, 0);

#ifdef Q_OS_WIN32
    // windows下使用windebug(内部用OutDebugString，可以输出到QtCreator控制台)代替color_stdout(default_logger的默认值)
    auto windebug_sink = std::make_shared<spdlog::sinks::windebug_sink_mt>();
    auto windebug_logger = std::make_shared<spdlog::logger>("windebug_logger", windebug_sink);
    windebug_logger->set_pattern("[%Y-%m-%d %H:%M:%S.%e] [%n] [%P] [%t] [%l] [%s:%#] %v");
    windebug_logger->set_level(spdlog::level::trace);
    spdlog::set_default_logger(windebug_logger);
#endif

    LOG_INFO("init log:{}", SPDLOG_ACTIVE_LEVEL);    

    return true;
}

std::shared_ptr<spdlog::logger> Log::logger()
{
    return m_logger;
}

}
