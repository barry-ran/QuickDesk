// Copyright 2026 QuickDesk Authors
#include "FileTransferQueue.h"
#include "ClipboardFiles.h"
#include <QFile>
#include <QSignalSpy>
#include <QTemporaryDir>
#include <QtTest>

using namespace quickdesk;

class FileTransferTest : public QObject {
    Q_OBJECT
    QTemporaryDir dir;

    QString file(const QString& name, qint64 size = 1) {
        const QString path = dir.filePath(name);
        QDir().mkpath(QFileInfo(path).absolutePath());
        QFile f(path);
        if (!f.open(QIODevice::WriteOnly) || !f.resize(size)) return {};
        return path;
    }
    QVariantMap row(const FileTransferQueue& q, int i) { return q.tasks().at(i).toMap(); }
    QString id(const FileTransferQueue& q, int i) { return row(q, i).value("taskId").toString(); }
    QString status(const FileTransferQueue& q, int i) { return row(q, i).value("status").toString(); }
    QJsonObject event(const QString& type, const QString& conn, const QString& transfer,
                      const QString& filename = QString()) {
        return {{"type", type}, {"connectionId", conn}, {"transferId", transfer},
                {"filename", filename}, {"bytesSent", 5}, {"totalBytes", 10},
                {"errorMessage", "test error"}};
    }

private slots:
    void sameNameAcrossSessionsAndPaths() {
        FileTransferQueue q;
        QSignalSpy commands(&q, &FileTransferQueue::commandRequested);
        q.beginSession("a1", "a");
        q.beginSession("b1", "b");
        QVERIFY(q.enqueueUpload("a1", QUrl::fromLocalFile(file("left/same.txt"))));
        QVERIFY(q.enqueueUpload("a1", QUrl::fromLocalFile(file("right/same.txt"))));
        QVERIFY(q.enqueueUpload("b1", QUrl::fromLocalFile(file("other/same.txt"))));
        QCOMPARE(commands.count(), 2);
        QCOMPARE(status(q, 1), QString("queued"));
        q.handleMessage(event("fileTransferProgress", "b1", "1", "same.txt"));
        QCOMPARE(row(q, 0).value("transferId").toString(), QString());
        QCOMPARE(row(q, 2).value("transferId").toString(), QString("1"));
        q.handleMessage(event("fileTransferComplete", "a1", "1", "same.txt"));
        QTRY_COMPARE(commands.count(), 3);
        QCOMPARE(status(q, 0), QString("complete"));
        QCOMPARE(status(q, 1), QString("uploading"));
        q.handleMessage(event("fileTransferError", "a1", "1"));
        q.handleMessage(event("fileTransferProgress", "a1", "1", "same.txt"));
        QCOMPARE(row(q, 1).value("transferId").toString(), QString());
        QCOMPARE(status(q, 0), QString("complete"));
        QCOMPARE(status(q, 2), QString("uploading"));
    }

    void pendingCancelBindsThenWaitsForTerminal() {
        FileTransferQueue q;
        QSignalSpy commands(&q, &FileTransferQueue::commandRequested);
        q.beginSession("a", "device");
        q.enqueueUpload("a", QUrl::fromLocalFile(file("first")));
        q.enqueueUpload("a", QUrl::fromLocalFile(file("second")));
        q.cancel(id(q, 0));
        QCOMPARE(commands.count(), 1);
        QCOMPARE(status(q, 0), QString("cancelling"));
        q.handleMessage(event("fileTransferProgress", "a", "t", "first"));
        QCOMPARE(commands.count(), 2);
        QCOMPARE(commands.at(1).at(0).toJsonObject().value("type").toString(), QString("cancelFileUpload"));
        QCOMPARE(commands.at(1).at(0).toJsonObject().value("transferId").toString(), QString("t"));
        q.handleMessage(event("fileTransferProgress", "a", "t", "first"));
        QCOMPARE(commands.count(), 2); // cancellation is idempotent
        q.cancellationTimedOut(id(q, 0));
        QCOMPARE(status(q, 0), QString("uncertain"));
        QCOMPARE(status(q, 1), QString("queued"));
        q.handleMessage(event("fileTransferError", "a", "t"));
        QTRY_COMPARE(commands.count(), 3);
        QCOMPARE(status(q, 0), QString("cancelled"));
        q.handleMessage(event("fileTransferComplete", "a", "t", "first"));
        QCOMPARE(status(q, 0), QString("cancelled"));
    }

    void queuedCancelNeverDispatchesAndCompletionCanWin() {
        FileTransferQueue q;
        QSignalSpy commands(&q, &FileTransferQueue::commandRequested);
        q.beginSession("a", "device");
        q.enqueueUpload("a", QUrl::fromLocalFile(file("race")));
        q.enqueueUpload("a", QUrl::fromLocalFile(file("queued")));
        q.cancel(id(q, 1));
        q.cancel(id(q, 0));
        q.handleMessage(event("fileTransferComplete", "a", "t", "race"));
        QCoreApplication::processEvents();
        QCOMPARE(commands.count(), 1);
        QCOMPARE(status(q, 0), QString("complete"));
        QCOMPARE(status(q, 1), QString("cancelled"));
    }

    void errorRoutingAndUnknownIdentifierPause() {
        FileTransferQueue q;
        QSignalSpy commands(&q, &FileTransferQueue::commandRequested);
        q.beginSession("a", "A"); q.beginSession("b", "B");
        q.enqueueUpload("a", QUrl::fromLocalFile(file("error-a")));
        q.enqueueUpload("a", QUrl::fromLocalFile(file("after-error")));
        q.enqueueUpload("b", QUrl::fromLocalFile(file("error-b")));
        q.handleMessage(event("fileTransferError", "b", "tb"));
        QCOMPARE(status(q, 0), QString("uploading"));
        QCOMPARE(status(q, 2), QString("error"));
        q.handleMessage(event("fileTransferError", "a", ""));
        QCoreApplication::processEvents();
        QCOMPARE(status(q, 0), QString("error"));
        QCOMPARE(status(q, 1), QString("queued"));
        QCOMPARE(commands.count(), 2); // no unsafe reuse of an unidentified slot
    }

    void disconnectReconnectRejectsOldGeneration() {
        FileTransferQueue q;
        QSignalSpy commands(&q, &FileTransferQueue::commandRequested);
        const auto url = QUrl::fromLocalFile(file("reconnect"));
        q.beginSession("old", "device");
        q.enqueueUpload("old", url);
        q.endSession("old");
        q.beginSession("new", "device");
        q.enqueueUpload("new", url);
        q.beginSession("old", "device"); // delayed connected notification
        QVERIFY(!q.enqueueUpload("old", url));
        q.handleMessage(event("fileTransferComplete", "old", "t", "reconnect"));
        QCOMPARE(status(q, 0), QString("disconnected"));
        QCOMPARE(status(q, 1), QString("uploading"));
        q.handleMessage(event("fileTransferComplete", "new", "t", "reconnect"));
        QCOMPARE(status(q, 1), QString("complete"));
        QCOMPARE(commands.count(), 2);
    }

    void duplicatePasteAndInvalidFiles() {
        FileTransferQueue q;
        QSignalSpy commands(&q, &FileTransferQueue::commandRequested);
        QSignalSpy notices(&q, &FileTransferQueue::notice);
        const auto url = QUrl::fromLocalFile(file("dedup"));
        QVERIFY(!q.enqueueUpload("missing", url));
        q.beginSession("a", "A");
        QVERIFY(q.enqueueUpload("a", url));
        QVERIFY(q.enqueueUpload("a", url));
        QCOMPARE(q.tasks().size(), 1);
        QCOMPARE(commands.count(), 1);
        QVERIFY(!q.enqueueUpload("a", QUrl::fromLocalFile(dir.path())));
        QVERIFY(!q.enqueueUpload("a", QUrl("https://example.com/file")));
        QVERIFY(!q.enqueueUpload("a", QUrl::fromLocalFile(dir.filePath("does-not-exist"))));
        QCOMPARE(notices.count(), 3);
    }

    void emptyAndLargeFilesAreOnlyPaths() {
        FileTransferQueue q;
        QSignalSpy commands(&q, &FileTransferQueue::commandRequested);
        q.beginSession("a", "A");
        const QString empty = file("empty", 0);
        q.enqueueUpload("a", QUrl::fromLocalFile(empty));
        q.handleMessage(event("fileTransferComplete", "a", "empty-id", "empty"));
        QCOMPARE(status(q, 0), QString("complete"));
        const QString large = file("large");
        q.enqueueUpload("a", QUrl::fromLocalFile(large));
        QTRY_COMPARE(commands.count(), 2);
        auto progress = event("fileTransferProgress", "a", "large-id", "large");
        progress.insert("bytesSent", double(qint64(1) << 32));
        progress.insert("totalBytes", double(qint64(1) << 33));
        q.handleMessage(progress);
        QCOMPARE(row(q, 1).value("progress").toDouble(), 0.5);
        const auto command = commands.last().at(0).toJsonObject();
        QCOMPARE(command.size(), 3); // command carries only a path, never contents
        QCOMPARE(command.value("filePath").toString(), QFileInfo(large).canonicalFilePath());
    }

    void clipboardOnlyAfterExplicitActionAndConfirmedDownload() {
        FileTransferQueue q;
        q.beginSession("a", "A");
        q.registerDownload("a", dir.path());
        const QString path = file("downloaded");
        QGuiApplication::clipboard()->setText("keep this");
        q.handleMessage(event("fileDownloadStarted", "a", "d", "downloaded"));
        QVERIFY(q.completedDownloadPath(id(q, 0)).isEmpty());
        auto done = event("fileDownloadComplete", "a", "d", "downloaded");
        done.insert("savePath", path);
        q.handleMessage(done);
        QCOMPARE(QGuiApplication::clipboard()->text(), QString("keep this"));
        q.endSession("a");
        QCOMPARE(q.completedDownloadPath(id(q, 0)), path);
        QVERIFY(copyLocalFileToClipboard(q.completedDownloadPath(id(q, 0))));
        QCOMPARE(QGuiApplication::clipboard()->mimeData()->urls(), QList<QUrl>{QUrl::fromLocalFile(path)});
        QVERIFY(QFile::remove(path));
        QGuiApplication::clipboard()->setText("new clipboard");
        QVERIFY(q.completedDownloadPath(id(q, 0)).isEmpty());
        QVERIFY(!copyLocalFileToClipboard(path));
        QCOMPARE(QGuiApplication::clipboard()->text(), QString("new clipboard"));
    }

    void downloadCannotCopyOutsideRequestedDirectory() {
        FileTransferQueue q;
        q.beginSession("a", "A");
        const QString outside = file("outside/file");
        QDir().mkpath(dir.filePath("inside"));
        q.registerDownload("a", dir.filePath("inside"));
        q.handleMessage(event("fileDownloadStarted", "a", "d", "file"));
        auto done = event("fileDownloadComplete", "a", "d", "file");
        done.insert("savePath", outside);
        q.handleMessage(done);
        QVERIFY(q.completedDownloadPath(id(q, 0)).isEmpty());
    }
};
QTEST_MAIN(FileTransferTest)
#include "file_transfer_test.moc"
