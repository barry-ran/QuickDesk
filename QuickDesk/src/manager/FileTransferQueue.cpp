// Copyright 2026 QuickDesk Authors
#include "FileTransferQueue.h"

#include <QDir>
#include <QFileInfo>
#include <QTimer>
#include <QUuid>
#include <algorithm>

namespace quickdesk {

FileTransferQueue::FileTransferQueue(QObject* parent) : QObject(parent) {}

bool FileTransferQueue::terminal(const QString& status)
{
    return status == "complete" || status == "error" || status == "cancelled" ||
           status == "disconnected";
}

QString FileTransferQueue::wireKey(const QString& connectionId, const QString& direction,
                                   const QString& transferId)
{
    return connectionId + QChar(0) + direction + QChar(0) + transferId;
}

void FileTransferQueue::beginSession(const QString& connectionId, const QString& deviceId)
{
    // A disconnected generation cannot be resurrected by a delayed connected event.
    if (!connectionId.isEmpty() && !m_retiredConnections.contains(connectionId))
        m_sessions.insert(connectionId, deviceId);
}

void FileTransferQueue::endSession(const QString& connectionId)
{
    if (connectionId.isEmpty()) return;
    m_sessions.remove(connectionId);
    m_downloadRoots.remove(connectionId);
    m_retiredConnections.insert(connectionId);
    m_blockedUploads.remove(connectionId);
    bool updated = false;
    for (auto& task : m_tasks) {
        if (task.connectionId == connectionId && !terminal(task.status)) {
            task.status = "disconnected";
            task.error = tr("Connection closed. This transfer was not confirmed complete.");
            updated = true;
        }
    }
    if (updated) emit changed();
}

void FileTransferQueue::reconcileSession(const QString& connectionId, const QString& deviceId,
                                         bool connected, bool terminated)
{
    // An initial Connecting snapshot is not the end of a session. Once active,
    // leaving Connected invalidates that generation, even if native reconnects.
    if (terminated || (!connected && m_sessions.contains(connectionId)))
        endSession(connectionId);
    else if (connected)
        beginSession(connectionId, deviceId);
}

bool FileTransferQueue::sessionActive(const QString& connectionId) const
{
    return m_sessions.contains(connectionId);
}

void FileTransferQueue::clearCompleted()
{
    // changed() is synchronous and can be emitted while a method holds a task
    // index/reference. Prune only after that stack unwinds. Keep wire-ID
    // tombstones so late native events cannot bind to the next pending upload.
    QTimer::singleShot(0, this, [this] {
        const auto oldSize = m_tasks.size();
        m_tasks.erase(std::remove_if(m_tasks.begin(), m_tasks.end(),
            [](const Task& task) { return terminal(task.status); }), m_tasks.end());
        if (oldSize != m_tasks.size()) emit changed();
    });
}

void FileTransferQueue::endAllSessions()
{
    const auto ids = m_sessions.keys();
    for (const auto& id : ids) endSession(id);
}

bool FileTransferQueue::enqueueUpload(const QString& connectionId, const QUrl& url)
{
    if (!m_sessions.contains(connectionId)) return false;
    const QString deviceId = m_sessions.value(connectionId);
    const QFileInfo file(url.toLocalFile());
    if (!url.isLocalFile() || !file.exists() || !file.isFile() || !file.isReadable()) {
        emit notice(deviceId, file.isDir()
            ? tr("Folders are not supported. Select individual files to upload.")
            : tr("Select an existing, readable local file."));
        return false;
    }
    // Canonical paths deduplicate repeated paste/drop while a task is pending.
    const QString path = file.canonicalFilePath();
    int pending = 0;
    for (const auto& task : m_tasks) {
        if (task.connectionId == connectionId && task.direction == "upload" &&
            !terminal(task.status)) {
            ++pending;
            if (task.path == path) return true;
        }
    }
    if (pending >= 256) {
        emit notice(deviceId, tr("The upload queue is full. Wait for pending files to finish."));
        return false;
    }
    Task task;
    task.id = QUuid::createUuid().toString(QUuid::WithoutBraces);
    task.connectionId = connectionId;
    task.deviceId = deviceId;
    task.filename = file.fileName();
    task.path = path;
    task.direction = "upload";
    task.status = "queued";
    m_tasks.append(task);
    emit changed();
    pump(connectionId);
    return true;
}

void FileTransferQueue::registerDownload(const QString& connectionId, const QString& saveDir)
{
    if (m_sessions.contains(connectionId))
        m_downloadRoots.insert(connectionId, QDir(saveDir).absolutePath());
}

int FileTransferQueue::taskIndex(const QString& taskId) const
{
    for (int i = 0; i < m_tasks.size(); ++i)
        if (m_tasks[i].id == taskId) return i;
    return -1;
}

void FileTransferQueue::pump(const QString& connectionId)
{
    if (!m_sessions.contains(connectionId) || m_blockedUploads.contains(connectionId)) return;
    for (const auto& task : m_tasks)
        if (task.connectionId == connectionId && task.direction == "upload" &&
            task.status != "queued" && !terminal(task.status)) return;
    for (int i = 0; i < m_tasks.size(); ++i) {
        if (m_tasks[i].connectionId != connectionId || m_tasks[i].status != "queued")
            continue;
        const QFileInfo file(m_tasks[i].path);
        if (!file.exists() || !file.isFile() || !file.isReadable()) {
            m_tasks[i].status = "error";
            m_tasks[i].error = tr("The local file is no longer available.");
            emit changed();
            continue;
        }
        m_tasks[i].status = "uploading";
        const QJsonObject command{{"type", "startFileUpload"},
                                  {"connectionId", connectionId},
                                  {"filePath", m_tasks[i].path}};
        emit changed();
        emit commandRequested(command);
        return;
    }
}

void FileTransferQueue::cancel(const QString& taskId)
{
    const int index = taskIndex(taskId);
    if (index < 0 || terminal(m_tasks[index].status) ||
        m_tasks[index].status == "cancelling" || m_tasks[index].status == "uncertain") return;
    if (m_tasks[index].status == "queued") {
        finish(index, "cancelled", tr("Cancelled before upload started."));
        return;
    }
    m_tasks[index].cancelRequested = true;
    m_tasks[index].status = "cancelling";
    m_tasks[index].error = tr("Cancellation requested. Waiting for the remote transfer result.");
    emit changed();
    sendCancel(index);
    QTimer::singleShot(15000, this, [this, taskId] { cancellationTimedOut(taskId); });
}

void FileTransferQueue::sendCancel(int index)
{
    if (m_tasks[index].transferId.isEmpty() || m_tasks[index].cancelSent) return;
    m_tasks[index].cancelSent = true;
    emit commandRequested(QJsonObject{
        {"type", m_tasks[index].direction == "upload" ? "cancelFileUpload" : "cancelFileDownload"},
        {"connectionId", m_tasks[index].connectionId}, {"transferId", m_tasks[index].transferId}});
}

void FileTransferQueue::cancellationTimedOut(const QString& taskId)
{
    const int index = taskIndex(taskId);
    if (index < 0 || m_tasks[index].status != "cancelling") return;
    m_tasks[index].status = "uncertain";
    m_tasks[index].error = tr("Cancellation was not confirmed. Reconnect to reset this session's "
                             "queue; the remote file may be incomplete.");
    emit changed();
}

void FileTransferQueue::finish(int index, const QString& status, const QString& error)
{
    if (terminal(m_tasks[index].status)) return;
    const QString connectionId = m_tasks[index].connectionId;
    if (!m_tasks[index].transferId.isEmpty())
        m_retiredTransfers.insert(wireKey(connectionId, m_tasks[index].direction,
                                         m_tasks[index].transferId));
    m_tasks[index].status = status;
    m_tasks[index].error = error;
    if (status == "complete") m_tasks[index].progress = 1;
    emit changed();
    // Defer dispatch until the current native message has unwound.
    QTimer::singleShot(0, this, [this, connectionId] { pump(connectionId); });
}

void FileTransferQueue::handleMessage(const QJsonObject& message)
{
    const QString connectionId = message.value("connectionId").toString();
    if (!m_sessions.contains(connectionId)) return;
    const QString type = message.value("type").toString();
    const bool download = type.startsWith("fileDownload");
    if (type != "fileDownloadStarted" && type != "fileDownloadProgress" &&
        type != "fileDownloadComplete" && type != "fileDownloadError" &&
        type != "fileTransferProgress" && type != "fileTransferComplete" &&
        type != "fileTransferError") return;
    const QString direction = download ? "download" : "upload";
    const QString transferId = message.value("transferId").toString();
    const QString filename = message.value("filename").toString();
    if (!transferId.isEmpty() &&
        m_retiredTransfers.contains(wireKey(connectionId, direction, transferId))) return;

    if (!download && transferId.isEmpty()) {
        if (type.endsWith("Error") || type.endsWith("Complete")) {
            // Without an ID, even a terminal-looking message cannot prove
            // which dispatch finished. Preserve the slot instead of guessing.
            for (auto& task : m_tasks) {
                if (task.connectionId == connectionId && task.direction == "upload" &&
                    task.status != "queued" && !terminal(task.status)) {
                    task.status = "uncertain";
                    task.error = tr("A transfer result arrived without an identifier. "
                                    "Reconnect to reset this session's queue.");
                    m_blockedUploads.insert(connectionId);
                    emit changed();
                    return;
                }
            }
        }
        return;
    }

    int index = -1;
    for (int i = 0; i < m_tasks.size(); ++i) {
        const auto& task = m_tasks[i];
        if (task.connectionId == connectionId && task.direction == direction &&
            !terminal(task.status) && task.status != "queued" &&
            !transferId.isEmpty() && task.transferId == transferId) {
            index = i;
            break;
        }
    }
    if (download && type == "fileDownloadStarted" && index < 0) {
        if (transferId.isEmpty() || !m_downloadRoots.contains(connectionId)) return;
        Task task;
        task.id = QUuid::createUuid().toString(QUuid::WithoutBraces);
        task.connectionId = connectionId;
        task.deviceId = m_sessions.value(connectionId);
        task.transferId = transferId;
        task.filename = filename;
        task.direction = direction;
        task.status = "downloading";
        task.saveRoot = m_downloadRoots.value(connectionId);
        m_tasks.append(task);
        emit changed();
        return;
    }
    if (!download && index < 0) {
        // The sole dispatched upload may acquire its native ID from its first
        // progress OR terminal event (zero-byte/failed-open files need no progress).
        for (int i = 0; i < m_tasks.size(); ++i) {
            const auto& task = m_tasks[i];
            if (task.connectionId != connectionId || task.direction != "upload" ||
                terminal(task.status) || task.status == "queued" || !task.transferId.isEmpty())
                continue;
            if (!filename.isEmpty() && filename != task.filename) return;
            index = i;
            break;
        }
    }
    if (index < 0) {
        if (download && type == "fileDownloadError" && m_downloadRoots.contains(connectionId))
            emit notice(m_sessions.value(connectionId),
                        tr("Download could not start: %1").arg(message.value("errorMessage").toString()));
        return;
    }
    if (!transferId.isEmpty()) m_tasks[index].transferId = transferId;
    const bool cancelling = m_tasks[index].cancelRequested;
    if (type.endsWith("Complete")) {
        if (download) m_tasks[index].savePath = message.value("savePath").toString();
        // A completion that wins the cancellation race is still a completion.
        finish(index, "complete", cancelling ? tr("Completed before cancellation took effect.") : QString());
    } else if (type.endsWith("Error")) {
        QString error = message.value("errorMessage").toString();
        if (error.isEmpty()) error = tr("File transfer failed.");
        if (cancelling)
            error = tr("Transfer ended after a cancellation request: %1").arg(error);
        finish(index, "error", error);
    } else if (type.endsWith("Progress")) {
        const double total = message.value("totalBytes").toDouble();
        const double bytes = message.value(download ? "bytesReceived" : "bytesSent").toDouble();
        m_tasks[index].progress = total > 0 ? std::clamp(bytes / total, 0.0, 1.0) : 0;
        emit changed();
        if (cancelling) sendCancel(index);
    }
}

QVariantList FileTransferQueue::tasks() const
{
    QVariantList result;
    for (const auto& task : m_tasks) {
        result.append(QVariantMap{{"taskId", task.id}, {"transferId", task.transferId},
            {"deviceId", task.deviceId}, {"filename", task.filename}, {"direction", task.direction},
            {"progress", task.progress}, {"status", task.status}, {"errorMessage", task.error},
            {"savePath", task.savePath}});
    }
    return result;
}

QString FileTransferQueue::completedDownloadPath(const QString& taskId) const
{
    const int index = taskIndex(taskId);
    if (index < 0) return {};
    const auto& task = m_tasks[index];
    if (task.direction != "download" || task.status != "complete") return {};
    const QFileInfo file(task.savePath);
    if (!file.exists() || !file.isFile() || !file.isReadable()) return {};
    const QString root = QFileInfo(task.saveRoot).canonicalFilePath();
    const QString path = file.canonicalFilePath();
    if (root.isEmpty() || path.isEmpty()) return {};
    const QString relative = QDir(root).relativeFilePath(path);
    if (relative == ".." || relative.startsWith("../") || QDir::isAbsolutePath(relative)) return {};
    return path;
}

} // namespace quickdesk
