// Copyright 2026 QuickDesk Authors

#include "NativeMessaging.h"
#include <QJsonDocument>
#include <QDataStream>
#include <QThread>

namespace quickdesk {

// --- Transport 1: QProcess (stdin/stdout) ---

NativeMessaging::NativeMessaging(QProcess* process, QObject* parent)
    : QObject(parent)
    , m_process(process)
{
    Q_ASSERT(process);
    connect(m_process, &QProcess::readyReadStandardOutput,
            this, &NativeMessaging::onReadyRead);
    connect(m_process, &QProcess::errorOccurred,
            this, &NativeMessaging::onProcessError);
}

// --- Transport 2: dual QLocalSocket (separate read/write Named Pipes) ---

NativeMessaging::NativeMessaging(QLocalSocket* readSocket,
                                 QLocalSocket* writeSocket,
                                 QObject* parent)
    : QObject(parent)
    , m_readSocket(readSocket)
    , m_writeSocket(writeSocket)
{
    Q_ASSERT(readSocket);
    Q_ASSERT(writeSocket);
    connect(m_readSocket, &QLocalSocket::readyRead,
            this, &NativeMessaging::onReadyRead);
    connect(m_readSocket, &QLocalSocket::errorOccurred,
            this, &NativeMessaging::onSocketError);
    connect(m_writeSocket, &QLocalSocket::errorOccurred,
            this, &NativeMessaging::onSocketError);
}

// --- Shared implementation ---

void NativeMessaging::sendMessage(const QJsonObject& message)
{
    // Async WebSocket API handlers (clickText, skillExec, ...) call this from
    // a QThreadPool thread. QProcess/QLocalSocket writes are not thread-safe,
    // and an interleaved length-prefixed frame wedges the core's stdin parser
    // permanently, so every write is serialized on the owning thread.
    if (QThread::currentThread() != thread()) {
        QMetaObject::invokeMethod(
            this, [this, message]() { sendMessage(message); },
            Qt::QueuedConnection);
        return;
    }

    QByteArray encoded = encodeMessage(message);
    qint64 written = -1;

    if (m_process) {
        if (m_process->state() != QProcess::Running) {
            emit errorOccurred("Process is not running");
            return;
        }
        written = m_process->write(encoded);
    } else if (m_writeSocket) {
        if (m_writeSocket->state() != QLocalSocket::ConnectedState) {
            emit errorOccurred("Socket is not connected");
            return;
        }
        written = m_writeSocket->write(encoded);
    } else {
        emit errorOccurred("No transport configured (process or socket)");
        return;
    }

    if (written != encoded.size()) {
        emit errorOccurred(QString("Failed to write message: wrote %1 of %2 bytes")
                          .arg(written).arg(encoded.size()));
    }
}

bool NativeMessaging::isReady() const
{
    if (m_process)
        return m_process->state() == QProcess::Running;
    if (m_readSocket && m_writeSocket) {
        return m_readSocket->state() == QLocalSocket::ConnectedState &&
               m_writeSocket->state() == QLocalSocket::ConnectedState;
    }
    return false;
}

void NativeMessaging::onReadyRead()
{
    if (m_process)
        m_buffer.append(m_process->readAllStandardOutput());
    else if (m_readSocket)
        m_buffer.append(m_readSocket->readAll());
    parseBuffer();
}

void NativeMessaging::onProcessError(QProcess::ProcessError error)
{
    QString errorMsg;
    switch (error) {
    case QProcess::FailedToStart:
        errorMsg = "Process failed to start";
        break;
    case QProcess::Crashed:
        errorMsg = "Process crashed";
        break;
    case QProcess::Timedout:
        errorMsg = "Process timed out";
        break;
    case QProcess::WriteError:
        errorMsg = "Process write error";
        break;
    case QProcess::ReadError:
        errorMsg = "Process read error";
        break;
    default:
        errorMsg = "Unknown process error";
        break;
    }
    emit errorOccurred(errorMsg);
}

void NativeMessaging::onSocketError(QLocalSocket::LocalSocketError error)
{
    auto* socket = qobject_cast<QLocalSocket*>(sender());
    QString errorMsg = socket
        ? socket->errorString()
        : QString("Socket error %1").arg(static_cast<int>(error));
    emit errorOccurred(errorMsg);
}

void NativeMessaging::parseBuffer()
{
    // Native Messaging format: 4-byte length (little-endian) + JSON
    while (m_buffer.size() >= 4) {
        // Read length (little-endian)
        quint32 length = 0;
        length |= static_cast<quint8>(m_buffer[0]);
        length |= static_cast<quint8>(m_buffer[1]) << 8;
        length |= static_cast<quint8>(m_buffer[2]) << 16;
        length |= static_cast<quint8>(m_buffer[3]) << 24;

        // Check if we have the complete message
        if (static_cast<quint32>(m_buffer.size()) < 4 + length) {
            break; // Wait for more data
        }

        // Extract JSON
        QByteArray jsonData = m_buffer.mid(4, length);
        m_buffer.remove(0, 4 + length);

        // Parse JSON
        QJsonParseError parseError;
        QJsonDocument doc = QJsonDocument::fromJson(jsonData, &parseError);
        
        if (parseError.error != QJsonParseError::NoError) {
            emit errorOccurred(QString("JSON parse error: %1").arg(parseError.errorString()));
            continue;
        }

        if (!doc.isObject()) {
            emit errorOccurred("Received non-object JSON");
            continue;
        }

        emit messageReceived(doc.object());
    }
}

QByteArray NativeMessaging::encodeMessage(const QJsonObject& message)
{
    QJsonDocument doc(message);
    QByteArray json = doc.toJson(QJsonDocument::Compact);
    
    quint32 length = json.size();
    
    QByteArray result;
    result.reserve(4 + length);
    
    // Write length (little-endian)
    result.append(static_cast<char>(length & 0xFF));
    result.append(static_cast<char>((length >> 8) & 0xFF));
    result.append(static_cast<char>((length >> 16) & 0xFF));
    result.append(static_cast<char>((length >> 24) & 0xFF));
    
    // Write JSON
    result.append(json);
    
    return result;
}

} // namespace quickdesk
