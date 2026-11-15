// Copyright 2026 QuickDesk Authors
// API request handler for WebSocket API

#ifndef QUICKDESK_API_APIHANDLER_H
#define QUICKDESK_API_APIHANDLER_H

#include "OcrEngine.h"
#include "OcrCache.h"
#include "UiStateService.h"
#include "VerificationService.h"
#include "SkillHandler.h"
#include "TrustHandler.h"

#include <QObject>
#include <QJsonObject>
#include <QHash>
#include <QSet>
#include <functional>
#include <QMap>

namespace quickdesk {

class MainController;

class ApiHandler : public QObject {
    Q_OBJECT

public:
    explicit ApiHandler(MainController* controller, QObject* parent = nullptr);

    QJsonObject handleRequest(const QJsonObject& request);
    TrustHandler* trustHandler() { return &m_trust; }

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
    QJsonObject handleGetHeldKeys(const QJsonObject& params);
    QJsonObject handleReleaseAllKeys(const QJsonObject& params);
    QJsonObject handleGetClipboard(const QJsonObject& params);
    QJsonObject handleSetClipboard(const QJsonObject& params);
    QJsonObject handleGetScreenSize(const QJsonObject& params);

    // OCR / UI 状态
    QJsonObject handleGetScreenText(const QJsonObject& params);
    QJsonObject handleFindElement(const QJsonObject& params);
    QJsonObject handleClickText(const QJsonObject& params);
    QJsonObject handleGetUiState(const QJsonObject& params);
    QJsonObject handleWaitForText(const QJsonObject& params);
    QJsonObject handleAssertTextPresent(const QJsonObject& params);

    // 验证与自愈
    QJsonObject handleVerifyActionResult(const QJsonObject& params);
    QJsonObject handleScreenDiffSummary(const QJsonObject& params);
    QJsonObject handleAssertScreenState(const QJsonObject& params);

    // Skill bridge
    QJsonObject handleSkillExec(const QJsonObject& params);
    QJsonObject handleSkillListTools(const QJsonObject& params);

    // Trust layer
    QJsonObject handleRequestConfirmation(const QJsonObject& params);
    QJsonObject handleEmergencyStop(const QJsonObject& params);
    QJsonObject handleDeactivateEmergency(const QJsonObject& params);
    QJsonObject handleResolveConfirmation(const QJsonObject& params);

    static int keyNameToNativeKeycode(const QString& keyName);
    static bool resolveKey(const QString& keyName, int* nativeKeycode, bool* needsShift);

    // Releases keys pressed through this API and still held down. Called
    // before keyboard_type/keyboard_hotkey so a forgotten key_release cannot
    // corrupt the next action with a stuck modifier.
    void releaseHeldKeys(const QString& deviceId, QStringList* released = nullptr);

    // Helpers
    QJsonObject makeResult(const QJsonObject& data);
    QJsonObject makeError(int code, const QString& message);

    MainController* m_controller;
    QMap<QString, Handler> m_handlers;
    QMap<QString, QString> m_clipboardCache;  // deviceId -> last received text
    QHash<QString, QSet<QString>> m_heldKeys; // deviceId -> key names held via keyPress
    UiStateService       m_uiState;
    VerificationService  m_verification;
    SkillHandler         m_skillHandler;
    TrustHandler         m_trust;
};

} // namespace quickdesk

#endif // QUICKDESK_API_APIHANDLER_H
