// Copyright 2026 QuickDesk Authors
#ifndef QUICKDESK_MANAGER_FILETRANSFERQUEUE_H
#define QUICKDESK_MANAGER_FILETRANSFERQUEUE_H

#include <QObject>
#include <QHash>
#include <QJsonObject>
#include <QList>
#include <QSet>
#include <QUrl>
#include <QVariantList>

namespace quickdesk {

// Product-layer scheduler. No file contents cross this class and no native
// protocol fields are added. One upload is dispatched per connection because
// the existing protocol does not echo a caller-supplied request identifier.
class FileTransferQueue final : public QObject {
    Q_OBJECT
public:
    explicit FileTransferQueue(QObject* parent = nullptr);
    void beginSession(const QString& connectionId, const QString& deviceId);
    void endSession(const QString& connectionId);
    void reconcileSession(const QString& connectionId, const QString& deviceId,
                          bool connected, bool terminated);
    bool sessionActive(const QString& connectionId) const;
    void clearCompleted();
    void endAllSessions();
    bool enqueueUpload(const QString& connectionId, const QUrl& url);
    void registerDownload(const QString& connectionId, const QString& saveDir);
    void handleMessage(const QJsonObject& message);
    void cancel(const QString& taskId);
    QVariantList tasks() const;
    QString completedDownloadPath(const QString& taskId) const;
    // Public for deterministic clock-free tests; called by the cancellation timer.
    void cancellationTimedOut(const QString& taskId);

signals:
    void changed();
    void commandRequested(const QJsonObject& command);
    void notice(const QString& deviceId, const QString& message);

private:
    struct Task {
        QString id, connectionId, deviceId, transferId;
        QString filename, path, direction, status, error, savePath, saveRoot;
        double progress = 0;
        bool cancelSent = false;
        bool cancelRequested = false;
    };
    QList<Task> m_tasks;
    QHash<QString, QString> m_sessions;
    QHash<QString, QString> m_downloadRoots;
    QSet<QString> m_retiredConnections;
    // Preserve retired native IDs even if presentation history is cleared.
    QSet<QString> m_retiredTransfers;
    QSet<QString> m_blockedUploads;
    static bool terminal(const QString& status);
    static QString wireKey(const QString& connectionId, const QString& direction,
                           const QString& transferId);
    int taskIndex(const QString& taskId) const;
    void pump(const QString& connectionId);
    void sendCancel(int index);
    void finish(int index, const QString& status, const QString& error = QString());
};

} // namespace quickdesk
#endif
