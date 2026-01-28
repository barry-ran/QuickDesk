// Copyright 2026 QuickDesk Authors
// QuickDesk Qt Application Entry Point

#include <QFontDatabase>
#include <QGuiApplication>
#include <QIcon>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QJSEngine>
#include <QtQuickControls2/QQuickStyle>

#include "core/localconfigcenter.h"
#include "core/userdatacenter.h"
#include "infra/env/applicationcontext.h"
#include "infra/log/log.h"

#include "controller/MainController.h"
#include "manager/ServerManager.h"
#include "manager/HostManager.h"
#include "manager/ClientManager.h"
#include "manager/SharedMemoryManager.h"
#include "component/VideoFrameProvider.h"
#include "component/KeycodeMapper.h"
#include "component/CursorImageProvider.h"

int main(int argc, char *argv[])
{
    QGuiApplication app(argc, argv);

    // infra
    infra::ApplicationContext::instance().init();
    infra::Log::instance().init(infra::ApplicationContext::instance().applicationDirPath());
    LOG_INFO("start app {}, log level:{} ********", infra::ApplicationContext::instance().applicationName().toStdString(), SPDLOG_ACTIVE_LEVEL);
    infra::ApplicationContext::instance().setApplicationVersion(APP_VERSION_STR);
    LOG_INFO("current version:{}", infra::ApplicationContext::instance().applicationVersion().toStdString());

    // QML FileDialog需要这个
    // 放在ApplicationContext::init后面设置，否则OrganizationName会包含在QStandardPaths::AppLocalDataLocation目录中
    app.setOrganizationName("QuickCoder");
    app.setApplicationName("QuickDesk");

    // 设置 Qt Quick Controls 使用 Basic 样式（允许完全自定义）
    QQuickStyle::setStyle("Basic");
    
    LOG_INFO("QuickDesk starting...");
    LOG_INFO("Qt version: {}", qVersion());

    core::LocalConfigCenter::instance().init();
    core::UserDataCenter::instance().init();

    // Register C++ types for QML
    qmlRegisterType<quickdesk::MainController>("QuickDesk", 1, 0, "MainController");
    qmlRegisterUncreatableType<quickdesk::ServerManager>("QuickDesk", 1, 0, "ServerManager",
        "ServerManager is accessed through MainController");
    qmlRegisterUncreatableType<quickdesk::HostManager>("QuickDesk", 1, 0, "HostManager",
        "HostManager is accessed through MainController");
    qmlRegisterUncreatableType<quickdesk::ClientManager>("QuickDesk", 1, 0, "ClientManager",
        "ClientManager is accessed through MainController");
    qmlRegisterUncreatableType<quickdesk::SharedMemoryManager>("QuickDesk", 1, 0, "SharedMemoryManager",
        "SharedMemoryManager is accessed through ClientManager");
    qmlRegisterType<quickdesk::VideoFrameProvider>("QuickDesk", 1, 0, "VideoFrameProvider");
    
    // Register KeycodeMapper as singleton
    qmlRegisterSingletonType<quickdesk::KeycodeMapper>("QuickDesk", 1, 0, "KeycodeMapper",
        [](QQmlEngine*, QJSEngine*) -> QObject* {
            return quickdesk::KeycodeMapper::instance();
        });

    QQmlApplicationEngine engine;
    
    // Register cursor image provider
    engine.addImageProvider("cursor", new quickdesk::CursorImageProvider());
    
    // Expose version to QML
    engine.rootContext()->setContextProperty("APP_VERSION", APP_VERSION_STR);
    
    QFontDatabase::addApplicationFont(":/res/font/SegoeFluentIcons.ttf");

    // Handle QML creation failures
    QObject::connect(&engine, &QQmlApplicationEngine::objectCreationFailed,
        &app, []() {
            LOG_CRITICAL("QML object creation failed!");
            QCoreApplication::exit(-1);
        },
        Qt::QueuedConnection);

    // Load main QML
    engine.loadFromModule("QuickDesk", "MainWindow");

    LOG_INFO("QuickDesk started successfully");

    return app.exec();
}
