// Copyright 2026 QuickDesk Authors
// API request handler for WebSocket API

#ifndef QUICKDESK_API_APIHANDLER_H
#define QUICKDESK_API_APIHANDLER_H

#include "OcrEngine.h"
#include "OcrCache.h"

#include <QObject>
#include <QJsonObject>
#include <functional>
#include <QMap>

namespace quickdesk {

class MainController;

class ApiHandler : public QObject {
    Q_OBJECT

public:
    explicit ApiHandler(MainController* controller, QObject* parent = nullptr);

    QJsonObject handleRequest(const QJsonObject& request);

private:
    using Handler = std::function<QJsonObject(const QJsonObject& params)>;

    void registerHandlers();

    // Host info
    QJsonObject handleGetHostInfo(const QJsonObject& params);
    QJsonObject handleGetHostClients(const QJsonObject& params);
    QJsonObject handleRefreshAccessCode(const QJsonObject& params);
    QJsonObject handleKickClient(const QJsonObject& params);

    // Status
    QJsonObject handleGetStatus(const QJsonObject& params);
    QJsonObject handleGetSignalingStatus(const QJsonObject& params);

    // Connection management
    QJsonObject handleListConnections(const QJsonObject& params);
    QJsonObject handleGetConnectionInfo(const QJsonObject& params);
    QJsonObject handleConnectToHost(const QJsonObject& params);
    QJsonObject handleDisconnectFromHost(const QJsonObject& params);
    QJsonObject handleDisconnectAll(const QJsonObject& params);

    // Remote desktop operations
    QJsonObject handleScreenshot(const QJsonObject& params);
    QJsonObject handleMouseClick(const QJsonObject& params);
    QJsonObject handleMouseDoubleClick(const QJsonObject& params);
    QJsonObject handleMouseMove(const QJsonObject& params);
    QJsonObject handleMouseScroll(const QJsonObject& params);
    QJsonObject handleKeyboardType(const QJsonObject& params);
    QJsonObject handleKeyboardHotkey(const QJsonObject& params);
    QJsonObject handleMouseDrag(const QJsonObject& params);
    QJsonObject handleKeyPress(const QJsonObject& params);
    QJsonObject handleKeyRelease(const QJsonObject& params);
    QJsonObject handleGetClipboard(const QJsonObject& params);
    QJsonObject handleSetClipboard(const QJsonObject& params);
    QJsonObject handleGetScreenSize(const QJsonObject& params);

    // OCR / UI 状态
    QJsonObject handleGetScreenText(const QJsonObject& params);
    QJsonObject handleFindElement(const QJsonObject& params);
    QJsonObject handleClickText(const QJsonObject& params);

    static int keyNameToScanCode(const QString& keyName);

    // Helpers
    QJsonObject makeResult(const QJsonObject& data);
    QJsonObject makeError(int code, const QString& message);

    MainController* m_controller;
    QMap<QString, Handler> m_handlers;
    QMap<QString, QString> m_clipboardCache;  // connectionId -> last received text
};

} // namespace quickdesk

#endif // QUICKDESK_API_APIHANDLER_H
