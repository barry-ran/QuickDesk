// Copyright 2026 QuickDesk Authors
// QuickDesk Qt Application Entry Point

#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QIcon>
#include <QDebug>

#include "infra/env/applicationcontext.h"
#include "infra/log/log.h"

#include "controller/MainController.h"
#include "manager/ServerManager.h"
#include "manager/HostManager.h"
#include "manager/ClientManager.h"

int main(int argc, char *argv[])
{
    QGuiApplication app(argc, argv);

    // infra
    infra::ApplicationContext::instance().init();
    infra::Log::instance().init(infra::ApplicationContext::instance().logPath());
    LOG_INFO("start app {}, log level:{} ********", infra::ApplicationContext::instance().applicationName().toStdString(), SPDLOG_ACTIVE_LEVEL);
    LOG_INFO("current version:{}", infra::ApplicationContext::instance().applicationVersion().toStdString());

    // QML FileDialog需要这个
    // 放在ApplicationContext::init后面设置，否则OrganizationName会包含在QStandardPaths::AppLocalDataLocation目录中
    app.setOrganizationName("QuickCoder");
    app.setApplicationName("QuickDesk");
    app.setApplicationVersion("0.1.0");

    qInfo() << "QuickDesk starting...";
    qInfo() << "Qt version:" << qVersion();

    // Register C++ types for QML
    qmlRegisterType<quickdesk::MainController>("QuickDesk", 1, 0, "MainController");
    qmlRegisterUncreatableType<quickdesk::ServerManager>("QuickDesk", 1, 0, "ServerManager",
        "ServerManager is accessed through MainController");
    qmlRegisterUncreatableType<quickdesk::HostManager>("QuickDesk", 1, 0, "HostManager",
        "HostManager is accessed through MainController");
    qmlRegisterUncreatableType<quickdesk::ClientManager>("QuickDesk", 1, 0, "ClientManager",
        "ClientManager is accessed through MainController");

    QQmlApplicationEngine engine;
    
    // Handle QML creation failures
    QObject::connect(&engine, &QQmlApplicationEngine::objectCreationFailed,
        &app, []() { 
            qCritical() << "QML object creation failed!";
            QCoreApplication::exit(-1); 
        },
        Qt::QueuedConnection);
    
    // Load main QML
    engine.loadFromModule("QuickDesk", "Main");

    qInfo() << "QuickDesk started successfully";
    
    return app.exec();
}
