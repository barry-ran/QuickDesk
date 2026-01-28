// Copyright 2026 QuickDesk Authors

#include "VideoFrameProvider.h"
#include "CursorImageProvider.h"
#include "infra/log/log.h"

#include <QDateTime>

namespace quickdesk {

VideoFrameProvider::VideoFrameProvider(QObject* parent)
    : QObject(parent)
{
}

VideoFrameProvider::~VideoFrameProvider()
{
}

void VideoFrameProvider::setVideoSink(QVideoSink* sink)
{
    if (m_videoSink == sink) {
        return;
    }

    m_videoSink = sink;
    emit videoSinkChanged();
    
    LOG_DEBUG("VideoFrameProvider: videoSink set for connection {}", 
              m_connectionId.toStdString());
}

void VideoFrameProvider::setConnectionId(const QString& connectionId)
{
    if (m_connectionId == connectionId) {
        return;
    }

    m_connectionId = connectionId;
    emit connectionIdChanged();
    
    LOG_DEBUG("VideoFrameProvider: connectionId set to {}", 
              connectionId.toStdString());
}

void VideoFrameProvider::setSharedMemoryManager(SharedMemoryManager* manager)
{
    if (m_sharedMemoryManager == manager) {
        return;
    }

    m_sharedMemoryManager = manager;
    emit sharedMemoryManagerChanged();
}

void VideoFrameProvider::setActive(bool active)
{
    if (m_active == active) {
        return;
    }

    m_active = active;
    emit activeChanged();
    
    LOG_DEBUG("VideoFrameProvider: active={} for connection {}", 
              active, m_connectionId.toStdString());
}

void VideoFrameProvider::onVideoFrameReady(quint32 frameIndex)
{
    Q_UNUSED(frameIndex)
    
    if (!m_active) {
        return;
    }
    
    pushFrame();
}

void VideoFrameProvider::pushFrame()
{
    if (!m_active || !m_videoSink || !m_sharedMemoryManager || 
        m_connectionId.isEmpty()) {
        return;
    }

    // Check if attached
    if (!m_sharedMemoryManager->isAttached(m_connectionId)) {
        return;
    }

    // Read video frame (efficient GPU path)
    QVideoFrame frame = m_sharedMemoryManager->readVideoFrame(m_connectionId);
    
    if (!frame.isValid()) {
        return;
    }

    // Update frame size if changed
    QSize newSize = frame.size();
    if (m_frameSize != newSize) {
        m_frameSize = newSize;
        emit frameSizeChanged();
    }

    // Push frame to video sink for GPU rendering
    m_videoSink->setVideoFrame(frame);

    emit frameReceived();
    
    // Update frame rate statistics
    updateFrameRate();
}

void VideoFrameProvider::updateFrameRate()
{
    qint64 now = QDateTime::currentMSecsSinceEpoch();
    
    if (m_frameRateStartTime == 0) {
        m_frameRateStartTime = now;
        m_frameCount = 0;
    }
    
    m_frameCount++;
    
    // Calculate frame rate every second
    qint64 elapsed = now - m_frameRateStartTime;
    if (elapsed >= 1000) {
        int newFps = static_cast<int>(m_frameCount * 1000 / elapsed);
        if (m_frameRate != newFps) {
            m_frameRate = newFps;
            emit frameRateChanged();
        }
        
        // Reset counters
        m_frameRateStartTime = now;
        m_frameCount = 0;
    }
    
    m_lastFrameTime = now;
}

void VideoFrameProvider::onCursorShapeChanged(int width, int height,
                                              int hotspotX, int hotspotY,
                                              const QByteArray& data)
{
    if (width <= 0 || height <= 0) {
        // Clear cursor
        m_cursorImage = QImage();
        m_cursorHotspot = QPoint(0, 0);
        emit cursorChanged();
        return;
    }
    
    // Expected data size: width * height * 4 (BGRA)
    int expectedSize = width * height * 4;
    if (data.size() < expectedSize) {
        LOG_WARN("Cursor data size mismatch: expected {} got {}", 
                 expectedSize, data.size());
        return;
    }
    
    // Create QImage from BGRA data
    // Note: QImage::Format_ARGB32 on little-endian systems is actually BGRA in memory
    m_cursorImage = QImage(reinterpret_cast<const uchar*>(data.constData()),
                           width, height, width * 4,
                           QImage::Format_ARGB32);
    // Make a deep copy since the original data buffer may be temporary
    m_cursorImage = m_cursorImage.copy();
    
    m_cursorHotspot = QPoint(hotspotX, hotspotY);
    
    // LOG_DEBUG("Cursor updated: {}x{} hotspot({}, {})", 
    //           width, height, hotspotX, hotspotY);
    
    // Update the image provider so QML can access the cursor
    if (CursorImageProvider::instance()) {
        CursorImageProvider::instance()->setCursor(m_connectionId, m_cursorImage, m_cursorHotspot);
    }
    
    emit cursorChanged();
}

} // namespace quickdesk
