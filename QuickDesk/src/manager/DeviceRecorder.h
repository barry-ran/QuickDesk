// Copyright 2026 QuickDesk Authors
// Control-side session recorder.

#ifndef QUICKDESK_MANAGER_DEVICERECORDER_H
#define QUICKDESK_MANAGER_DEVICERECORDER_H

#include <QMediaCaptureSession>
#include <QMediaRecorder>
#include <QNetworkAccessManager>
#include <QObject>
#include <QString>
#include <QTimer>
#include <QVideoFrameInput>

#include <functional>

class QVideoFrame;

namespace quickdesk {

class AuthManager;
class ServerManager;
class SharedMemoryManager;

/**
 * @brief Records a remote session on the control side.
 *
 * The decoded remote frames already arrive as QVideoFrame through
 * SharedMemoryManager (the same call VideoFrameProvider uses for display), so
 * they can be pushed straight into Qt Multimedia's encoder through
 * QVideoFrameInput. That keeps the remoting engine untouched and avoids
 * depending on an FFmpeg SDK: Qt's own backend muxes the MP4.
 *
 * Lifecycle: startRecording() -> frames -> stopRecording() -> upload, which
 * creates a `session_recordings` row and then posts the bytes to
 * /v1/me/recordings/:id/artifact. The first decoded frame is also saved as a
 * small JPEG and posted to /v1/me/recordings/:id/thumbnail so the recordings
 * list has a preview.
 */
class DeviceRecorder : public QObject {
    Q_OBJECT
    Q_PROPERTY(bool recording READ isRecording NOTIFY recordingChanged)
    Q_PROPERTY(QString lastError READ lastError NOTIFY lastErrorChanged)
    Q_PROPERTY(QString lastRecordingId READ lastRecordingId NOTIFY uploadFinished)

public:
    DeviceRecorder(SharedMemoryManager* sharedMemory,
                   AuthManager* auth,
                   ServerManager* server,
                   QObject* parent = nullptr);
    ~DeviceRecorder() override;

    bool isRecording() const { return m_recording; }
    QString lastError() const { return m_lastError; }
    QString lastRecordingId() const { return m_lastRecordingId; }

    Q_INVOKABLE bool startRecording(const QString& deviceId);
    Q_INVOKABLE void stopRecording();

signals:
    void recordingChanged();
    void lastErrorChanged();
    void uploadFinished(const QString& recordingId);

private slots:
    void onFrameTick();
    void onRecorderStateChanged(QMediaRecorder::RecorderState state);
    void onRecorderError(QMediaRecorder::Error error, const QString& errorString);

private:
    void fail(const QString& message);
    void capturePoster(const QVideoFrame& frame);
    QString httpBaseUrl() const;
    void uploadRecording(const QString& filePath);
    void createIndexRowThenUpload(const QString& filePath, qint64 sizeBytes);
    void uploadBytes(const QString& filePath, const QString& recordingId);
    void uploadThumbnail(const QString& recordingId);
    /**
     * POST one file as multipart/form-data, retrying only failures a retry can
     * actually fix (transport errors, 5xx, 429). A 4xx is a verdict on the
     * payload, so it is reported immediately instead of being retried.
     */
    void postFileWithRetry(const QString& url,
                           const QString& filePath,
                           const QString& mimeType,
                           const QString& fileName,
                           const QString& label,
                           std::function<void(bool ok, int status, const QString& detail)> done,
                           int attempt);
    void markFailed(const QString& recordingId, const QString& filePath);

    SharedMemoryManager* m_sharedMemory = nullptr;
    AuthManager* m_auth = nullptr;
    ServerManager* m_server = nullptr;

    QMediaCaptureSession m_session;
    QMediaRecorder m_recorder;
    QVideoFrameInput m_frameInput;
    QNetworkAccessManager m_network;
    QTimer m_frameTimer;

    QString m_deviceId;
    QString m_outputPath;
    QString m_posterPath;
    QString m_lastError;
    QString m_lastRecordingId;
    bool m_recording = false;
    bool m_posterCaptured = false;
    qint64 m_startedAtMs = 0;
};

} // namespace quickdesk

#endif // QUICKDESK_MANAGER_DEVICERECORDER_H
