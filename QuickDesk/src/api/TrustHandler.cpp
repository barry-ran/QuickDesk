// Copyright 2026 QuickDesk Authors

#include "TrustHandler.h"

#include "../core/localconfigcenter.h"
#include <QJsonArray>
#include <QJsonDocument>
#include <QMetaObject>
#include <QMutexLocker>
#include <QThread>

#include "infra/log/log.h"

namespace quickdesk {

TrustHandler::TrustHandler(MainController* controller, QObject* parent)
    : QObject(parent)
    , m_controller(controller)
{
}

QJsonObject TrustHandler::handleRequestConfirmation(const QJsonObject& params)
{
    QString deviceId = params["deviceId"].toString();
    QString toolName     = params["tool_name"].toString();
    QString argumentsJson = QString::fromUtf8(
        QJsonDocument(params["arguments"].toObject()).toJson(QJsonDocument::Compact));
    QString riskLevel    = params["risk_level"].toString("medium");
    QJsonArray reasonsArr = params["reasons"].toArray();
    int timeoutSecs      = params["timeout_secs"].toInt(60);

    if (toolName.isEmpty()) {
        return {{"error", "requestConfirmation: tool_name is required"}};
    }

    auto& config = core::LocalConfigCenter::instance();
    bool autoApprove = false;
    const auto readConfirmMode = [&config, &autoApprove]() {
        autoApprove = config.trustConfirmMode() == "auto_approve";
    };
    if (QThread::currentThread() == config.thread()) {
        readConfirmMode();
    } else if (!QMetaObject::invokeMethod(&config, readConfirmMode,
                                        Qt::BlockingQueuedConnection)) {
        LOG_WARN("TrustHandler: failed to read confirmation policy; requiring manual confirmation");
    }

    if (autoApprove) {
        return {{"approved", true}, {"reason", "auto_approved_by_policy"}};
    }

    QString confirmId = QString("confirm-%1").arg(m_nextId.fetchAndAddRelaxed(1));
    LOG_INFO("TrustHandler: requestConfirmation tool={} id={}", 
             toolName.toStdString(), confirmId.toStdString());

    QMutex mutex;
    QWaitCondition cond;
    bool approved = false;
    QString reason;
    bool done = false;

    {
        QMutexLocker locker(&m_pendingMutex);
        m_pending.insert(confirmId, PendingConfirmation{
            &mutex, &cond, &approved, &reason, &done
        });
    }

    QStringList reasons;
    for (const auto& r : reasonsArr) {
        reasons.append(r.toString());
    }

    emit confirmationRequested(
        confirmId, deviceId, toolName, argumentsJson,
        riskLevel, reasons, timeoutSecs
    );

    {
        QMutexLocker locker(&mutex);
        if (!done) {
            unsigned long waitMs = static_cast<unsigned long>(timeoutSecs) * 1000;
            cond.wait(&mutex, waitMs);
        }
    }

    {
        QMutexLocker locker(&m_pendingMutex);
        m_pending.remove(confirmId);
    }

    if (!done) {
        LOG_WARN("TrustHandler: confirmation {} timed out", confirmId.toStdString());
        return {{"approved", false}, {"reason", "timeout"}};
    }

    return {{"approved", approved}, {"reason", reason}};
}

QJsonObject TrustHandler::handleEmergencyStop(const QJsonObject& params)
{
    QString reason = params["reason"].toString("manual");

    {
        QMutexLocker locker(&m_pendingMutex);
        for (auto it = m_pending.begin(); it != m_pending.end(); ++it) {
            QMutexLocker resultLock(it.value().mutex);
            *it.value().approved = false;
            *it.value().reason   = "emergency stop";
            *it.value().done     = true;
            it.value().cond->wakeOne();
        }
        m_pending.clear();
    }

    emit emergencyStopActivated(reason);
    LOG_INFO("TrustHandler: emergency stop activated — {}", reason.toStdString());

    return {{"status", "emergency_stop_activated"}, {"reason", reason}};
}

QJsonObject TrustHandler::handleDeactivateEmergency(const QJsonObject& params)
{
    Q_UNUSED(params);
    emit emergencyStopDeactivated();
    LOG_INFO("TrustHandler: emergency stop deactivated");
    return {{"status", "emergency_stop_deactivated"}};
}

void TrustHandler::resolveConfirmation(const QString& confirmationId,
                                        bool approved,
                                        const QString& reason)
{
    QMutexLocker locker(&m_pendingMutex);
    auto it = m_pending.find(confirmationId);
    if (it == m_pending.end()) {
        LOG_WARN("TrustHandler: confirmation {} not found", confirmationId.toStdString());
        return;
    }

    PendingConfirmation& pc = it.value();
    QMutexLocker resultLock(pc.mutex);
    *pc.approved = approved;
    *pc.reason   = reason;
    *pc.done     = true;
    pc.cond->wakeOne();
}

} // namespace quickdesk
