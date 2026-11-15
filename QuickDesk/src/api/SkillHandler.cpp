// Copyright 2026 QuickDesk Authors
// Skill bridge handler implementation

#include "SkillHandler.h"

#include <QJsonDocument>
#include <QMutexLocker>

#include "../controller/MainController.h"
#include "../manager/ClientManager.h"
#include "infra/log/log.h"

namespace quickdesk {

namespace {

QJsonObject skillBridgeUnsupported(const QString& toolName)
{
    return {{"error", QString("%1: the remote session does not answer on the "
                              "qd-skill-bridge capability").arg(toolName)},
            {"hint", QStringLiteral("The host did not answer a capability probe. The official "
                                    "Windows/macOS remoting cores do not advertise qd-skill-bridge "
                                    "(Linux QuickDesk builds do). Use device_id 'local' to run "
                                    "tools on this machine.")}};
}

} // namespace

SkillHandler::SkillHandler(MainController* controller, QObject* parent)
    : QObject(parent)
    , m_controller(controller)
{
    connect(m_controller->clientManager(),
            &ClientManager::skillBridgeResponseReceived,
            this,
            [this](const QString& deviceId, const QJsonObject& response) {
                onSkillResponse(deviceId, response);
            });

    // Capability support may change with a new session (e.g. a host upgrade),
    // so drop the cached probe result whenever the connection state changes.
    connect(m_controller->clientManager(),
            &ClientManager::connectionStateChanged,
            this,
            [this](const QString& deviceId, const QString&, const QJsonObject&) {
                QMutexLocker locker(&m_bridgeMutex);
                m_bridgeSupport.remove(deviceId);
            });

    // Local skill host responses share the same request-id space; local ids are
    // prefixed "local-" so they can never collide with a remote client's ids.
    if (auto* shm = m_controller->skillHostManager()) {
        connect(shm, &SkillHostManager::skillHostResponseReady,
                this, [this](const QJsonObject& response) {
                    onSkillResponse(QString(), response);
                });
    }
}

// ---- Public API (called from worker threads) ----

QJsonObject SkillHandler::handleSkillExec(const QJsonObject& params)
{
    QString deviceId = params["deviceId"].toString();
    QString tool     = params["tool"].toString();
    QJsonValue args  = params.value("args");

    if (deviceId.isEmpty() || tool.isEmpty()) {
        return {{"error", "skillExec: device_id and tool are required"}};
    }

    if (!isConnectionValid(deviceId)) {
        return {{"error", QString("skillExec: device '%1' not found — use a deviceId from "
                                  "list_connections, or 'local' to run on this machine")
                     .arg(deviceId)}};
    }

    const bool local = isLocalTarget(deviceId);
    if (local && (!m_controller->skillHostManager() ||
                  !m_controller->skillHostManager()->isRunning())) {
        return {{"error", "skillExec: the local skill host is not running — enable it in "
                          "Settings or use a remote deviceId"}};
    }

    if (!local) {
        QJsonObject bridgeError = ensureRemoteBridge(deviceId, "skillExec");
        if (!bridgeError.isEmpty()) {
            return bridgeError;
        }
    }

    QJsonObject payload;
    payload["id"]   = local ? "local-" + nextRequestId() : nextRequestId();
    payload["type"] = "toolCall";
    payload["tool"] = tool;
    payload["args"] = args.isUndefined() ? QJsonValue(QJsonObject{}) : args;

    // Tools such as run_command do their own timeout (60 s by default), so the
    // bridge must wait longer than the 10 s identity-round-trip budget.
    const int toolTimeoutSecs = args.toObject().value("timeout_secs").toInt(60);
    return sendAndWait(deviceId, payload, (toolTimeoutSecs + 15) * 1000);
}

QJsonObject SkillHandler::handleSkillListTools(const QJsonObject& params)
{
    QString deviceId = params["deviceId"].toString();

    if (deviceId.isEmpty()) {
        return {{"error", "skillListTools: device_id is required"}};
    }

    if (!isConnectionValid(deviceId)) {
        return {{"error", QString("skillListTools: device '%1' not found — use a deviceId from "
                                  "list_connections, or 'local' to run on this machine")
                     .arg(deviceId)}};
    }

    const bool local = isLocalTarget(deviceId);
    if (local && (!m_controller->skillHostManager() ||
                  !m_controller->skillHostManager()->isRunning())) {
        return {{"error", "skillListTools: the local skill host is not running"}};
    }

    QJsonObject probeResult;
    if (!local) {
        QJsonObject bridgeError =
            ensureRemoteBridge(deviceId, "skillListTools", &probeResult);
        if (!bridgeError.isEmpty()) {
            return bridgeError;
        }
        if (!probeResult.isEmpty()) {
            return probeResult;
        }
    }

    QJsonObject payload;
    payload["id"]   = local ? "local-" + nextRequestId() : nextRequestId();
    payload["type"] = "listTools";

    // Discovery must not fail on a slow remote before the tool call itself
    // would; use the same widened budget as skillExec.
    return sendAndWait(deviceId, payload, 25000);
}

// ---- Callback from ClientManager (main thread) ----

void SkillHandler::onSkillResponse(const QString& /*deviceId*/,
                                   const QJsonObject& response)
{
    QString id = response["id"].toString();
    if (id.isEmpty()) return;

    QMutexLocker pendingLock(&m_pendingMutex);
    auto it = m_pending.find(id);
    if (it == m_pending.end()) return;

    PendingRequest& req = it.value();
    QMutexLocker resultLock(req.mutex);
    *req.result = response;
    *req.done   = true;
    req.cond->wakeOne();
}

// ---- Private helpers ----

bool SkillHandler::isLocalTarget(const QString& deviceId) const
{
    if (deviceId.compare("local", Qt::CaseInsensitive) == 0) {
        return true;
    }
    auto* hm = m_controller->hostManager();
    return hm && !hm->deviceId().isEmpty() && hm->deviceId() == deviceId;
}

bool SkillHandler::isConnectionValid(const QString& deviceId) const
{
    if (isLocalTarget(deviceId)) {
        return true;
    }
    auto* cm = m_controller->clientManager();
    return cm && cm->connectedDeviceIds().contains(deviceId);
}

QJsonObject SkillHandler::ensureRemoteBridge(const QString& deviceId,
                                             const QString& toolName,
                                             QJsonObject* probeResult)
{
    {
        QMutexLocker locker(&m_bridgeMutex);
        auto it = m_bridgeSupport.constFind(deviceId);
        if (it != m_bridgeSupport.constEnd()) {
            if (it.value() == BridgeSupport::Supported) {
                return {};
            }
            if (it.value() == BridgeSupport::Unsupported) {
                return skillBridgeUnsupported(toolName);
            }
        }
    }

    QJsonObject probe;
    probe["id"]   = nextRequestId();
    probe["type"] = "listTools";
    QJsonObject result = sendAndWait(deviceId, probe, 8000);

    QMutexLocker locker(&m_bridgeMutex);
    if (result.contains("tools")) {
        m_bridgeSupport[deviceId] = BridgeSupport::Supported;
        if (probeResult) {
            *probeResult = result;
        }
        return {};
    }
    m_bridgeSupport[deviceId] = BridgeSupport::Unsupported;
    return skillBridgeUnsupported(toolName);
}

QJsonObject SkillHandler::sendAndWait(const QString& deviceId,
                                      const QJsonObject& payload,
                                      int timeoutMs)
{
    QString id = payload["id"].toString();
    QByteArray bytes = QJsonDocument(payload).toJson(QJsonDocument::Compact);
    QString jsonData = QString::fromUtf8(bytes);

    QMutex mutex;
    QWaitCondition cond;
    QJsonObject result;
    bool done = false;

    {
        QMutexLocker locker(&m_pendingMutex);
        m_pending.insert(id, PendingRequest{&mutex, &cond, &result, &done});
    }

    // NativeMessaging is not thread-safe — send from the main thread.
    if (isLocalTarget(deviceId)) {
        auto* controller = m_controller;
        QMetaObject::invokeMethod(controller, [controller, jsonData]() {
            if (auto* shm = controller->skillHostManager()) {
                shm->forwardFromLocal(jsonData);
            }
        }, Qt::QueuedConnection);
    } else {
        auto* cm = m_controller->clientManager();
        QMetaObject::invokeMethod(cm, [cm, deviceId, jsonData]() {
            cm->sendSkillCommand(deviceId, jsonData);
        }, Qt::QueuedConnection);
    }

    // Block the *worker thread* (not the main thread) until response or timeout.
    {
        QMutexLocker locker(&mutex);
        if (!done) {
            cond.wait(&mutex, timeoutMs);
        }
    }

    {
        QMutexLocker locker(&m_pendingMutex);
        m_pending.remove(id);
    }

    if (!done) {
        const QString op = payload["type"].toString() == "listTools"
                               ? QStringLiteral("skillListTools")
                               : QStringLiteral("skillExec");
        return {{"error", QString("%1 timed out after %2 ms").arg(op).arg(timeoutMs)},
                {"hint", isLocalTarget(deviceId)
                             ? QStringLiteral("The local skill host did not answer.")
                             : QStringLiteral("The remote app did not answer on the skill bridge. "
                                              "Not every remote build supports it: the official "
                                              "Windows/macOS remoting cores do not advertise the "
                                              "qd-skill-bridge capability (Linux QuickDesk builds do). "
                                              "Use device_id 'local' to run tools on this machine.")}};
    }
    return result;
}

QString SkillHandler::nextRequestId()
{
    return QString("skill-%1").arg(m_nextId.fetchAndAddRelaxed(1));
}

} // namespace quickdesk
