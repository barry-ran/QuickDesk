// Copyright 2026 QuickDesk Authors
// Skill bridge handler — routes skillExec / skillListTools from MCP to the
// remote skill host via the qd-skill-bridge WebRTC data channel.

#ifndef QUICKDESK_API_SKILLHANDLER_H
#define QUICKDESK_API_SKILLHANDLER_H

#include <QAtomicInt>
#include <QHash>
#include <QJsonObject>
#include <QMap>
#include <QMutex>
#include <QObject>
#include <QString>
#include <QWaitCondition>

namespace quickdesk {

class MainController;

/**
 * @brief Handles skillExec and skillListTools MCP tool calls.
 *
 * Thread safety: handleSkillExec / handleSkillListTools are designed to be
 * called from *worker threads* (e.g. QThreadPool).  They block the calling
 * worker thread via QWaitCondition (NOT QEventLoop) until the remote skill host
 * replies or the timeout elapses.
 *
 * onSkillResponse runs on the *main thread* (direct signal connection from
 * ClientManager) and wakes the waiting worker thread.
 */
class SkillHandler : public QObject {
    Q_OBJECT

public:
    explicit SkillHandler(MainController* controller, QObject* parent = nullptr);

    // Thread-safe: can be called from any thread (dispatched via QThreadPool).
    QJsonObject handleSkillExec(const QJsonObject& params);
    QJsonObject handleSkillListTools(const QJsonObject& params);

    // Called on the main thread when a skillBridgeResponse arrives.
    void onSkillResponse(const QString& deviceId,
                         const QJsonObject& response);

private:
    enum class BridgeSupport { Unknown, Supported, Unsupported };

    bool isConnectionValid(const QString& deviceId) const;
    bool isLocalTarget(const QString& deviceId) const;

    // The core cannot tell whether the host advertised qd-skill-bridge, so a
    // remote call to a host without it would otherwise stall until the full
    // skill timeout. Probe once per device with a cheap listTools request.
    QJsonObject ensureRemoteBridge(const QString& deviceId,
                                   const QString& toolName,
                                   QJsonObject* probeResult = nullptr);

    QJsonObject sendAndWait(const QString& deviceId,
                            const QJsonObject& payload,
                            int timeoutMs = 10000);

    QString nextRequestId();

    MainController* m_controller;
    QAtomicInt      m_nextId{1};

    QMutex m_bridgeMutex;
    QHash<QString, BridgeSupport> m_bridgeSupport;

    struct PendingRequest {
        QMutex*         mutex  = nullptr;
        QWaitCondition* cond   = nullptr;
        QJsonObject*    result = nullptr;
        bool*           done   = nullptr;
    };
    QMutex m_pendingMutex;
    QMap<QString, PendingRequest> m_pending;
};

} // namespace quickdesk

#endif // QUICKDESK_API_SKILLHANDLER_H
