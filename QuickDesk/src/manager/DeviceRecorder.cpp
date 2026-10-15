// Copyright 2026 QuickDesk Authors
// Control-side session recorder implementation.

#include "DeviceRecorder.h"

#include "AuthManager.h"
#include "ServerManager.h"
#include "SharedMemoryManager.h"
#include "infra/log/log.h"

#include <QDateTime>
#include <QDir>
#include <QFile>
#include <QFileInfo>
#include <QHttpMultiPart>
#include <QHttpPart>
#include <QImage>
#include <QJsonDocument>
#include <QJsonObject>
#include <QMediaFormat>
#include <QNetworkReply>
#include <QNetworkRequest>
#include <QStandardPaths>
#include <QUrl>
#include <QVideoFrame>

namespace quickdesk {

namespace {

// Sampling rate for pushing decoded frames into the encoder. The remote stream
// may arrive at a different rate; the encoder is fed at a steady cadence and
// timestamps are generated locally so the resulting file has correct timing.
constexpr int kFrameIntervalMs = 1000 / 15;
// QVideoFrame presentation timestamps are in microseconds, not milliseconds.
constexpr qint64 kFrameDurationUs = 1000000 / 15;
// Upload retries. The first entry is "send immediately"; the rest are the
// backoff applied before each subsequent attempt. Four attempts span ~8s, which
// is long enough to ride out a dropped connection without stalling the UI.
constexpr int kMaxUploadAttempts = 4;
constexpr int kRetryBackoffMs[kMaxUploadAttempts] = {0, 800, 2000, 5000};
// A preview is a thumbnail: cap the long edge so a 4K session does not produce a
// 4K JPEG that every recordings-list render has to decode.
constexpr int kPosterMaxEdgePx = 640;

QString isoNow() {
    return QDateTime::currentDateTimeUtc().toString(Qt::ISODate);
}

} // namespace

DeviceRecorder::DeviceRecorder(SharedMemoryManager* sharedMemory,
                               AuthManager* auth,
                               ServerManager* server,
                               QObject* parent)
    : QObject(parent)
    , m_sharedMemory(sharedMemory)
    , m_auth(auth)
    , m_server(server) {
    m_frameTimer.setInterval(kFrameIntervalMs);
    m_frameTimer.setTimerType(Qt::PreciseTimer);
    connect(&m_frameTimer, &QTimer::timeout, this, &DeviceRecorder::onFrameTick);
    connect(&m_recorder, &QMediaRecorder::recorderStateChanged,
            this, &DeviceRecorder::onRecorderStateChanged);
    connect(&m_recorder, &QMediaRecorder::errorOccurred,
            this, &DeviceRecorder::onRecorderError);
}

DeviceRecorder::~DeviceRecorder() {
    if (m_recording) {
        m_frameTimer.stop();
        m_recorder.stop();
    }
}

bool DeviceRecorder::startRecording(const QString& deviceId) {
    if (m_recording) {
        return true;
    }
    if (deviceId.isEmpty()) {
        fail(QStringLiteral("no active device to record"));
        return false;
    }
    if (!m_sharedMemory || !m_auth || !m_server) {
        fail(QStringLiteral("recorder is not wired to the client managers"));
        return false;
    }

    const QString dir = QStandardPaths::writableLocation(QStandardPaths::TempLocation);
    QDir().mkpath(dir);
    m_outputPath = QDir(dir).filePath(
        QStringLiteral("quickdesk-rec-%1.mp4").arg(QDateTime::currentMSecsSinceEpoch()));
    QFile::remove(m_outputPath);
    // The poster lives beside the capture so it shares the temp-file lifetime
    // and is never mistaken for a recording artifact.
    m_posterPath = m_outputPath + QStringLiteral(".thumb.jpg");
    QFile::remove(m_posterPath);
    m_posterCaptured = false;

    QMediaFormat format;
    format.setFileFormat(QMediaFormat::MPEG4);
    format.setVideoCodec(QMediaFormat::VideoCodec::H264);
    m_recorder.setMediaFormat(format);
    m_recorder.setQuality(QMediaRecorder::HighQuality);
    m_recorder.setVideoFrameRate(15);
    m_recorder.setOutputLocation(QUrl::fromLocalFile(m_outputPath));

    // Video-only: the decoded remote stream is fed as frames, so there is no
    // capture source and no audio input to attach.
    m_session.setVideoFrameInput(&m_frameInput);
    m_session.setRecorder(&m_recorder);

    m_deviceId = deviceId;
    m_startedAtMs = QDateTime::currentMSecsSinceEpoch();
    m_lastError.clear();
    emit lastErrorChanged();

    m_recorder.record();
    m_recording = true;
    emit recordingChanged();
    m_frameTimer.start();
    LOG_INFO("DeviceRecorder: recording started device={} path={}",
             deviceId.toStdString(), m_outputPath.toStdString());
    return true;
}

void DeviceRecorder::stopRecording() {
    if (!m_recording) {
        return;
    }
    m_frameTimer.stop();
    m_recording = false;
    emit recordingChanged();
    // The upload is kicked off from the StoppedState transition so the file is
    // already finalised on disk.
    m_recorder.stop();
}

void DeviceRecorder::onFrameTick() {
    if (!m_recording || m_deviceId.isEmpty() || !m_sharedMemory) {
        return;
    }
    QVideoFrame frame = m_sharedMemory->readVideoFrame(m_deviceId);
    if (!frame.isValid()) {
        return;
    }
    capturePoster(frame);
    // readVideoFrame() copies the planes into an owned buffer, so the frame can
    // be handed to the encoder without racing the producer.
    // Qt expects microsecond presentation timestamps here; feeding milliseconds
    // collapses a multi-second recording into a few milliseconds of container
    // time (caught by probing the produced MP4, not by the build).
    const qint64 nowUs = QDateTime::currentMSecsSinceEpoch() * 1000;
    const qint64 elapsedUs = nowUs - (m_startedAtMs * 1000);
    frame.setStartTime(elapsedUs);
    frame.setEndTime(elapsedUs + kFrameDurationUs);
    m_frameInput.sendVideoFrame(frame);
}

void DeviceRecorder::capturePoster(const QVideoFrame& frame) {
    if (m_posterCaptured || m_posterPath.isEmpty()) {
        return;
    }
    // Capture the first frame the operator sees, not the last: grabbing it at
    // stop time would race the teardown of the decoder.
    const QImage image = frame.toImage();
    if (image.isNull()) {
        return;
    }
    const QImage poster =
        (image.width() > kPosterMaxEdgePx || image.height() > kPosterMaxEdgePx)
            ? image.scaled(kPosterMaxEdgePx, kPosterMaxEdgePx, Qt::KeepAspectRatio,
                           Qt::SmoothTransformation)
            : image;
    if (poster.save(m_posterPath, "JPEG", 82)) {
        m_posterCaptured = true;
    } else {
        LOG_WARN("DeviceRecorder: could not write the poster to {}", m_posterPath.toStdString());
    }
}

void DeviceRecorder::onRecorderStateChanged(QMediaRecorder::RecorderState state) {
    if (state != QMediaRecorder::StoppedState || m_outputPath.isEmpty()) {
        return;
    }
    const QString finishedPath = m_outputPath;
    m_outputPath.clear();
    uploadRecording(finishedPath);
}

void DeviceRecorder::onRecorderError(QMediaRecorder::Error error, const QString& errorString) {
    Q_UNUSED(error);
    fail(errorString.isEmpty() ? QStringLiteral("recorder error") : errorString);
}

void DeviceRecorder::fail(const QString& message) {
    m_lastError = message;
    emit lastErrorChanged();
    LOG_WARN("DeviceRecorder: {}", message.toStdString());
}

QString DeviceRecorder::httpBaseUrl() const {
    QString url = m_server ? m_server->serverUrl() : QString();
    url.replace(QStringLiteral("ws://"), QStringLiteral("http://"));
    url.replace(QStringLiteral("wss://"), QStringLiteral("https://"));
    if (!url.endsWith(QLatin1Char('/'))) {
        url += QLatin1Char('/');
    }
    return url;
}

void DeviceRecorder::uploadRecording(const QString& filePath) {
    const QFileInfo info(filePath);
    if (!info.exists() || info.size() <= 0) {
        fail(QStringLiteral("recording produced no data"));
        QFile::remove(filePath);
        QFile::remove(m_posterPath);
        m_posterPath.clear();
        return;
    }
    createIndexRowThenUpload(filePath, info.size());
}

void DeviceRecorder::createIndexRowThenUpload(const QString& filePath, qint64 sizeBytes) {
    Q_UNUSED(sizeBytes);
    const qint64 durationSeconds =
        qMax<qint64>(1, (QDateTime::currentMSecsSinceEpoch() - m_startedAtMs) / 1000);

    QJsonObject payload;
    payload.insert(QStringLiteral("device_id"), m_deviceId);
    payload.insert(QStringLiteral("device_name"), QString());
    payload.insert(QStringLiteral("session_id"),
                   QStringLiteral("qt-%1").arg(QDateTime::currentMSecsSinceEpoch()));
    // The row starts as "recording": the capture is on disk but the media has
    // not reached the server yet. A successful upload flips it to "completed"
    // server-side, and a failure explicitly marks it "failed" below — so the
    // index never claims a recording exists when it does not.
    payload.insert(QStringLiteral("status"), QStringLiteral("recording"));
    payload.insert(QStringLiteral("format"), QStringLiteral("mp4"));
    payload.insert(QStringLiteral("duration_seconds"), durationSeconds);
    payload.insert(QStringLiteral("started_at"),
                   QDateTime::fromMSecsSinceEpoch(m_startedAtMs).toUTC().toString(Qt::ISODate));
    payload.insert(QStringLiteral("ended_at"), isoNow());

    const QString body = QString::fromUtf8(QJsonDocument(payload).toJson(QJsonDocument::Compact));
    const QUrl url(httpBaseUrl() + QStringLiteral("v1/me/recordings"));

    m_auth->request(QStringLiteral("POST"), url, body,
                    [this, filePath](int statusCode, const std::string& errorMsg,
                                     const std::string& data) {
                        const QString message = QString::fromStdString(errorMsg);
                        const QString payloadText = QString::fromStdString(data);
                        QMetaObject::invokeMethod(this, [this, statusCode, message, payloadText, filePath]() {
                            if (statusCode != 200 && statusCode != 201) {
                                fail(QStringLiteral("could not index recording: HTTP %1 %2")
                                         .arg(statusCode)
                                         .arg(message));
                                return;
                            }
                            const QJsonObject row =
                                QJsonDocument::fromJson(payloadText.toUtf8()).object();
                            const QJsonValue idValue = row.value(QStringLiteral("id"));
                            if (idValue.isUndefined() || idValue.isNull()) {
                                fail(QStringLiteral("index response carried no recording id"));
                                return;
                            }
                            uploadBytes(filePath, QString::number(idValue.toInt()));
                        });
                    });
}

void DeviceRecorder::uploadBytes(const QString& filePath, const QString& recordingId) {
    const QString url =
        httpBaseUrl() + QStringLiteral("v1/me/recordings/%1/artifact").arg(recordingId);
    postFileWithRetry(url, filePath, QStringLiteral("video/mp4"), QStringLiteral("recording.mp4"),
                      QStringLiteral("artifact"),
                      [this, filePath, recordingId](bool ok, int status, const QString& detail) {
                          if (!ok) {
                              fail(QStringLiteral("recording upload failed: HTTP %1 %2")
                                       .arg(status)
                                       .arg(detail));
                              // Leave the index honest and drop the orphaned capture: a row
                              // that says "recording" forever is worse than an explicit failure.
                              markFailed(recordingId, filePath);
                              return;
                          }
                          m_lastRecordingId = recordingId;
                          emit uploadFinished(m_lastRecordingId);
                          LOG_INFO("DeviceRecorder: uploaded recording id={} file={}",
                                   recordingId.toStdString(), filePath.toStdString());
                          // The temp file only exists to carry the bytes to the server.
                          QFile::remove(filePath);
                          uploadThumbnail(recordingId);
                      },
                      0);
}

void DeviceRecorder::uploadThumbnail(const QString& recordingId) {
    const QString poster = m_posterPath;
    m_posterPath.clear();
    if (poster.isEmpty() || !QFileInfo::exists(poster)) {
        return;
    }
    // Best effort: the recording itself is already stored, so a missing preview
    // must not turn a completed capture into a reported failure.
    const QString url =
        httpBaseUrl() + QStringLiteral("v1/me/recordings/%1/thumbnail").arg(recordingId);
    postFileWithRetry(url, poster, QStringLiteral("image/jpeg"), QStringLiteral("thumb.jpg"),
                      QStringLiteral("thumbnail"),
                      [poster](bool ok, int status, const QString& detail) {
                          if (!ok) {
                              LOG_WARN("DeviceRecorder: thumbnail upload failed: HTTP {} {}",
                                       status, detail.toStdString());
                          }
                          QFile::remove(poster);
                      },
                      0);
}

void DeviceRecorder::postFileWithRetry(
    const QString& url,
    const QString& filePath,
    const QString& mimeType,
    const QString& fileName,
    const QString& label,
    std::function<void(bool ok, int status, const QString& detail)> done,
    int attempt) {
    auto* file = new QFile(filePath);
    if (!file->open(QIODevice::ReadOnly)) {
        delete file;
        done(false, 0, QStringLiteral("cannot reopen %1").arg(filePath));
        return;
    }

    auto* multipart = new QHttpMultiPart(QHttpMultiPart::FormDataType);
    QHttpPart filePart;
    filePart.setHeader(QNetworkRequest::ContentTypeHeader, mimeType);
    filePart.setHeader(QNetworkRequest::ContentDispositionHeader,
                       QStringLiteral("form-data; name=\"file\"; filename=\"%1\"").arg(fileName));
    filePart.setBodyDevice(file);
    file->setParent(multipart);
    multipart->append(filePart);

    // Braces, not parentheses: `QNetworkRequest request(QUrl(url));` parses as a
    // function declaration (most vexing parse) and the follow-up calls then fail
    // to compile with C2228.
    QNetworkRequest request{QUrl(url)};
    for (const auto& header : m_auth->userAuthHeaders(false)) {
        request.setRawHeader(header.first.toUtf8(), header.second.toUtf8());
    }

    QNetworkReply* reply = m_network.post(request, multipart);
    multipart->setParent(reply);
    connect(reply, &QNetworkReply::finished, this,
            [this, reply, file, url, filePath, mimeType, fileName, label, done, attempt]() {
                const int status = reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
                const QByteArray responseBody = reply->readAll();
                const QNetworkReply::NetworkError networkError = reply->error();
                reply->deleteLater();

                // Close the source file before any retry or removal: Windows
                // refuses to delete (or reopen) an open handle.
                if (file && file->isOpen()) {
                    file->close();
                }

                if (networkError == QNetworkReply::NoError && (status == 200 || status == 201)) {
                    done(true, status, QString());
                    return;
                }

                // A 4xx means the server rejected this payload; replaying it
                // would fail identically. Only transport errors and 5xx/429 are
                // worth another attempt.
                const bool transient = networkError != QNetworkReply::NoError || status >= 500
                                       || status == 429 || status == 0;
                if (transient && attempt + 1 < kMaxUploadAttempts) {
                    const int delayMs = kRetryBackoffMs[attempt + 1];
                    LOG_INFO("DeviceRecorder: retrying {} upload (attempt {} of {}) in {}ms",
                             label.toStdString(), attempt + 2, kMaxUploadAttempts, delayMs);
                    QTimer::singleShot(
                        delayMs, this,
                        [this, url, filePath, mimeType, fileName, label, done, attempt]() {
                            postFileWithRetry(url, filePath, mimeType, fileName, label, done,
                                              attempt + 1);
                        });
                    return;
                }
                done(false, status, QString::fromUtf8(responseBody.left(200)));
            });
}

void DeviceRecorder::markFailed(const QString& recordingId, const QString& filePath) {
    // The capture never reached the server, so drop it immediately: keeping an
    // orphan file around would only hide the failure.
    QFile::remove(filePath);
    if (!m_posterPath.isEmpty()) {
        QFile::remove(m_posterPath);
        m_posterPath.clear();
    }
    if (!m_auth || recordingId.isEmpty()) {
        return;
    }
    QJsonObject payload;
    payload.insert(QStringLiteral("status"), QStringLiteral("failed"));
    const QString body = QString::fromUtf8(QJsonDocument(payload).toJson(QJsonDocument::Compact));
    const QUrl url(httpBaseUrl() + QStringLiteral("v1/me/recordings/%1").arg(recordingId));
    m_auth->request(QStringLiteral("PATCH"), url, body,
                    [recordingId](int statusCode, const std::string& errorMsg, const std::string&) {
                        if (statusCode != 200) {
                            LOG_WARN("DeviceRecorder: could not mark recording {} failed: HTTP {} {}",
                                     recordingId.toStdString(), statusCode, errorMsg);
                        }
                    });
}

} // namespace quickdesk
