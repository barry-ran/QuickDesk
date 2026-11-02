// Copyright 2026 QuickDesk Authors

#include "api/TrustHandler.h"
#include "core/localconfigcenter.h"
#include "infra/env/applicationcontext.h"

#include <QCoreApplication>
#include <QDebug>
#include <QDir>
#include <QEventLoop>
#include <QJsonDocument>
#include <QSqlDatabase>
#include <QStandardPaths>
#include <QThread>
#include <QThreadPool>
#include <QTimer>
#include <QUuid>

namespace {

int failures = 0;

void check(bool condition, const QString& message)
{
    if (!condition) {
        ++failures;
        qCritical().noquote() << "FAIL:" << message;
    }
}

QJsonObject request()
{
    return {
        {"deviceId", "test-device"},
        {"tool_name", "run_command"},
        {"arguments", QJsonObject{{"command", "whoami"}}},
        {"risk_level", "high"},
        {"timeout_secs", 2}
    };
}

void workerRequest(quickdesk::TrustHandler& handler, const QString& name,
                   bool expectPrompt, bool approve, const QString& expectedReason)
{
    QEventLoop loop;
    QTimer deadline;
    deadline.setSingleShot(true);
    QObject::connect(&deadline, &QTimer::timeout, &loop, [&]() {
        qFatal("Worker request deadlocked: %s", qPrintable(name));
    });

    int promptCount = 0;
    QJsonObject result;
    QObject::connect(&handler, &quickdesk::TrustHandler::confirmationRequested,
                     &loop, [&](const QString& id, const QString&,
                                const QString&, const QString&, const QString&,
                                const QStringList&, int) {
        ++promptCount;
        check(QThread::currentThread() == handler.thread(), name + ": prompt receiver thread");
        handler.resolveConfirmation(id, approve, approve ? "user approved" : "user denied");
    }, Qt::QueuedConnection);

    QThreadPool pool;
    pool.setMaxThreadCount(1);
    deadline.start(5000);
    pool.start([&]() {
        const QJsonObject response = handler.handleRequestConfirmation(request());
        QMetaObject::invokeMethod(&loop, [&, response]() {
            result = response;
            loop.quit();
        }, Qt::QueuedConnection);
    });
    loop.exec();
    pool.waitForDone();

    check(promptCount == (expectPrompt ? 1 : 0), name + ": prompt count");
    const QJsonObject expected{{"approved", approve}, {"reason", expectedReason}};
    check(result == expected,
          name + ": response " + QString::fromUtf8(QJsonDocument(result).toJson(QJsonDocument::Compact)));
    qInfo().noquote() << "Completed:" << name;
}

class TestData {
public:
    TestData()
    {
        QStandardPaths::setTestModeEnabled(true);
        QCoreApplication::setOrganizationName("QuickDeskTests");
        QCoreApplication::setApplicationName("trust-" + QUuid::createUuid().toString(QUuid::WithoutBraces));
        m_path = QStandardPaths::writableLocation(QStandardPaths::AppDataLocation);
        if (QDir(m_path).exists()) {
            qFatal("Refusing to reuse an existing test data directory");
        }
        auto& context = infra::ApplicationContext::instance();
        if (!context.init() || context.localDataPath() != m_path) {
            qFatal("Could not initialize isolated application context");
        }
        if (!core::LocalConfigCenter::instance().init()) {
            qFatal("Could not initialize real SQLite configuration database");
        }
    }

    ~TestData()
    {
        QSqlDatabase::removeDatabase("appconfig.db");
        check(QDir(m_path).removeRecursively(), "remove isolated test data");
    }

private:
    QString m_path;
};

} // namespace

int main(int argc, char* argv[])
{
    QCoreApplication app(argc, argv);
    {
        TestData data;
        auto& config = core::LocalConfigCenter::instance();
        quickdesk::TrustHandler handler(nullptr);

        workerRequest(handler, "unset policy prompts and denies", true, false, "user denied");

        config.setTrustConfirmMode("auto_approve");
        workerRequest(handler, "worker reads persisted auto approval", false, true, "auto_approved_by_policy");

        const QJsonObject direct = handler.handleRequestConfirmation(request());
        check(direct == QJsonObject{{"approved", true}, {"reason", "auto_approved_by_policy"}},
              "same-thread auto approval");

        config.setTrustConfirmMode("manual");
        workerRequest(handler, "live switch to manual preserves denial", true, false, "user denied");
        workerRequest(handler, "manual approval uses the dialog response", true, true, "user approved");

        config.setTrustConfirmMode("auto_approve");
        workerRequest(handler, "live switch back to auto approval", false, true, "auto_approved_by_policy");
        check(handler.handleRequestConfirmation(QJsonObject{}).contains("error"),
              "auto approval does not bypass required tool validation");

        config.setTrustConfirmMode("unknown_policy");
        workerRequest(handler, "unknown policy fails closed", true, false, "user denied");

        config.setTrustConfirmMode("");
        workerRequest(handler, "empty policy falls back to manual", true, false, "user denied");
    }

    if (failures != 0) {
        qCritical() << failures << "trust regression checks failed";
        return 1;
    }
    qInfo() << "All trust handler regression checks passed";
    return 0;
}
