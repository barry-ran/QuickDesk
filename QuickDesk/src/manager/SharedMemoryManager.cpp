// Copyright 2026 QuickDesk Authors

#include "SharedMemoryManager.h"
#include "YUVPlanarVideoBuffer.h"
#include "infra/log/log.h"

namespace quickdesk {

SharedMemoryManager::SharedMemoryManager(QObject* parent)
    : QObject(parent)
{
}

SharedMemoryManager::~SharedMemoryManager()
{
    detachAll();
}

bool SharedMemoryManager::attach(const QString& connectionId, 
                                  const QString& sharedMemoryName)
{
    std::string key = connectionId.toStdString();
    
    // Already attached?
    auto it = m_handles.find(key);
    if (it != m_handles.end()) {
        auto& existing = it->second;
        if (existing->sharedMemoryName == sharedMemoryName && 
            existing->sharedMemory && existing->sharedMemory->isAttached()) {
            return true;  // Already attached to same memory
        }
        // Different memory name - detach first
        detach(connectionId);
    }

    // Create new QSharedMemory instance
    auto shm = std::make_unique<QSharedMemory>();
    
    // Use setNativeKey() to directly use the platform-specific name
    // This allows interoperability with C++ client using native APIs
    shm->setNativeKey(sharedMemoryName);

    // Attach to existing shared memory (created by C++ client)
    if (!shm->attach(QSharedMemory::ReadOnly)) {
        LOG_WARN("Failed to attach to shared memory '{}' for connection {}: {}",
                 sharedMemoryName.toStdString(), connectionId.toStdString(),
                 shm->errorString().toStdString());
        emit frameReadError(connectionId, 
                            QString("Failed to attach: %1").arg(shm->errorString()));
        return false;
    }

    // Create handle entry
    auto handle = std::make_unique<SharedMemoryHandle>();
    handle->connectionId = connectionId;
    handle->sharedMemoryName = sharedMemoryName;
    handle->lastFrameIndex = 0;
    handle->sharedMemory = std::move(shm);

    m_handles[key] = std::move(handle);

    LOG_INFO("Attached to shared memory '{}' for connection {}",
             sharedMemoryName.toStdString(), connectionId.toStdString());
    emit attachmentChanged(connectionId, true);
    return true;
}

void SharedMemoryManager::detach(const QString& connectionId)
{
    std::string key = connectionId.toStdString();
    auto it = m_handles.find(key);
    if (it == m_handles.end()) {
        return;
    }

    closeHandle(*it->second);
    m_handles.erase(it);

    LOG_INFO("Detached from shared memory for connection {}", 
             connectionId.toStdString());
    emit attachmentChanged(connectionId, false);
}

void SharedMemoryManager::detachAll()
{
    for (auto& pair : m_handles) {
        closeHandle(*pair.second);
        emit attachmentChanged(pair.second->connectionId, false);
    }
    m_handles.clear();
    LOG_INFO("Detached from all shared memory regions");
}

bool SharedMemoryManager::isAttached(const QString& connectionId) const
{
    std::string key = connectionId.toStdString();
    auto it = m_handles.find(key);
    if (it == m_handles.end()) {
        return false;
    }
    return it->second->sharedMemory && it->second->sharedMemory->isAttached();
}

QVideoFrame SharedMemoryManager::readVideoFrame(const QString& connectionId)
{
    std::string key = connectionId.toStdString();
    auto it = m_handles.find(key);
    if (it == m_handles.end()) {
        return QVideoFrame();
    }

    auto& handle = it->second;
    if (!handle->sharedMemory || !handle->sharedMemory->isAttached()) {
        return QVideoFrame();
    }

    // Lock shared memory for reading
    if (!handle->sharedMemory->lock()) {
        LOG_WARN("Failed to lock shared memory for connection {}: {}",
                 connectionId.toStdString(), 
                 handle->sharedMemory->errorString().toStdString());
        return QVideoFrame();
    }

    QVideoFrame result;
    const void* data = handle->sharedMemory->constData();
    
    if (data) {
        const SharedFrameHeader* header = 
            static_cast<const SharedFrameHeader*>(data);

        // Validate header
        if (header->magic == kSharedFrameMagic && 
            header->version == kSharedFrameVersion) {
            
            handle->lastFrameIndex = header->frame_index;

            quint32 width = header->width;
            quint32 height = header->height;
            quint32 dataSize = header->data_size;
            SharedFrameFormat frameFormat = static_cast<SharedFrameFormat>(header->format);

            if (width > 0 && height > 0 && width <= 8192 && height <= 8192) {
                const uchar* frameData = static_cast<const uchar*>(data) + 
                                         sizeof(SharedFrameHeader);

                // Read stride information from header
                quint32 ySrcStride = header->y_stride;
                quint32 uSrcStride = header->u_stride;
                quint32 vSrcStride = header->v_stride;
                
                // Calculate expected data size with stride
                quint32 expectedSize = ySrcStride * height + 
                                      uSrcStride * (height / 2) + 
                                      vSrcStride * (height / 2);
                
                if (dataSize == expectedSize) {
                    // Create QVideoFrameFormat for YUV420P
                    QVideoFrameFormat format(QSize(static_cast<int>(width), static_cast<int>(height)),
                                            QVideoFrameFormat::Format_YUV420P);
                    // Set correct color space for HD content (>= 720p uses BT.709)
                    // H.264 hardware encoder outputs BT.709 limited range by default
                    format.setColorSpace(QVideoFrameFormat::ColorSpace_BT709);
                    format.setColorRange(QVideoFrameFormat::ColorRange_Video);

                    // 🚀 Use custom YUVPlanarVideoBuffer to control stride (Qt 6.8)
                    auto planarBuffer = std::make_unique<YUVPlanarVideoBuffer>(format);
                    
                    // Y plane: Single copy with stride
                    int sizeY = ySrcStride * height;
                    planarBuffer->m_data[0] = QByteArray(reinterpret_cast<const char*>(frameData), sizeY);
                    planarBuffer->m_bytesPerLine[0] = ySrcStride;
                    
                    // U plane: Single copy with stride
                    const uchar* uSrc = frameData + ySrcStride * height;
                    int sizeU = uSrcStride * (height / 2);
                    planarBuffer->m_data[1] = QByteArray(reinterpret_cast<const char*>(uSrc), sizeU);
                    planarBuffer->m_bytesPerLine[1] = uSrcStride;
                    
                    // V plane: Single copy with stride
                    const uchar* vSrc = frameData + ySrcStride * height + uSrcStride * (height / 2);
                    int sizeV = vSrcStride * (height / 2);
                    planarBuffer->m_data[2] = QByteArray(reinterpret_cast<const char*>(vSrc), sizeV);
                    planarBuffer->m_bytesPerLine[2] = vSrcStride;
                    
                    planarBuffer->m_planeCount = 3;
                    
                    // Create QVideoFrame with custom buffer (Qt 6.8 requires unique_ptr)
                    result = QVideoFrame(std::move(planarBuffer));
                } else {
                    LOG_WARN("YUV I420 data size mismatch for connection {}: {} vs expected {}",
                             connectionId.toStdString(), dataSize, expectedSize);
                }
            }
        }
    }

    handle->sharedMemory->unlock();
    return result;
}

FrameData SharedMemoryManager::lockFrame(const QString& connectionId)
{
    FrameData frameData;
    std::string key = connectionId.toStdString();
    auto it = m_handles.find(key);
    if (it == m_handles.end()) {
        return frameData;
    }

    auto& handle = it->second;
    if (!handle->sharedMemory || !handle->sharedMemory->isAttached()) {
        return frameData;
    }

    // Already locked?
    if (handle->isLocked) {
        LOG_WARN("Shared memory already locked for connection {}", 
                 connectionId.toStdString());
        return frameData;
    }

    if (!handle->sharedMemory->lock()) {
        LOG_WARN("Failed to lock shared memory for connection {}: {}",
                 connectionId.toStdString(), 
                 handle->sharedMemory->errorString().toStdString());
        return frameData;
    }

    handle->isLocked = true;

    const void* data = handle->sharedMemory->constData();
    if (!data) {
        handle->sharedMemory->unlock();
        handle->isLocked = false;
        return frameData;
    }

    const SharedFrameHeader* header = 
        static_cast<const SharedFrameHeader*>(data);

    // Validate header
    if (header->magic != kSharedFrameMagic || 
        header->version != kSharedFrameVersion) {
        handle->sharedMemory->unlock();
        handle->isLocked = false;
        return frameData;
    }

    quint32 width = header->width;
    quint32 height = header->height;
    quint32 dataSize = header->data_size;
    quint32 expectedSize = width * height + (width / 2) * (height / 2) * 2;  // YUV I420

    if (width == 0 || height == 0 || width > 8192 || height > 8192 ||
        dataSize != expectedSize) {
        handle->sharedMemory->unlock();
        handle->isLocked = false;
        return frameData;
    }

    // Update last frame index
    handle->lastFrameIndex = header->frame_index;

    // Fill frame data
    frameData.valid = true;
    frameData.width = width;
    frameData.height = height;
    frameData.frameIndex = header->frame_index;
    frameData.timestampUs = header->timestamp_us;
    frameData.format = static_cast<SharedFrameFormat>(header->format);
    frameData.data = static_cast<const uchar*>(data) + sizeof(SharedFrameHeader);
    frameData.dataSize = dataSize;

    return frameData;
}

void SharedMemoryManager::unlockFrame(const QString& connectionId)
{
    std::string key = connectionId.toStdString();
    auto it = m_handles.find(key);
    if (it == m_handles.end()) {
        return;
    }

    auto& handle = it->second;
    if (handle->isLocked && handle->sharedMemory) {
        handle->sharedMemory->unlock();
        handle->isLocked = false;
    }
}

QSize SharedMemoryManager::frameSize(const QString& connectionId) const
{
    std::string key = connectionId.toStdString();
    auto it = m_handles.find(key);
    if (it == m_handles.end() || !it->second->sharedMemory || 
        !it->second->sharedMemory->isAttached()) {
        return QSize();
    }

    // We need to read the header to get dimensions
    // This requires a const_cast since we need to lock
    auto& handle = *it->second;
    if (!const_cast<QSharedMemory*>(handle.sharedMemory.get())->lock()) {
        return QSize();
    }

    QSize result;
    const void* data = handle.sharedMemory->constData();
    if (data) {
        const SharedFrameHeader* header = 
            static_cast<const SharedFrameHeader*>(data);
        if (header->magic == kSharedFrameMagic) {
            result = QSize(static_cast<int>(header->width), 
                          static_cast<int>(header->height));
        }
    }

    const_cast<QSharedMemory*>(handle.sharedMemory.get())->unlock();
    return result;
}

quint32 SharedMemoryManager::lastFrameIndex(const QString& connectionId) const
{
    std::string key = connectionId.toStdString();
    auto it = m_handles.find(key);
    if (it == m_handles.end()) {
        return 0;
    }
    return it->second->lastFrameIndex;
}

bool SharedMemoryManager::isNewFrame(const QString& connectionId, 
                                      quint32 currentFrameIndex) const
{
    std::string key = connectionId.toStdString();
    auto it = m_handles.find(key);
    if (it == m_handles.end()) {
        return true;  // Not tracking, assume new
    }
    return currentFrameIndex > it->second->lastFrameIndex;
}

void SharedMemoryManager::closeHandle(SharedMemoryHandle& handle)
{
    if (handle.sharedMemory) {
        if (handle.sharedMemory->isAttached()) {
            handle.sharedMemory->detach();
        }
        handle.sharedMemory.reset();
    }
    handle.lastFrameIndex = 0;
}

} // namespace quickdesk
