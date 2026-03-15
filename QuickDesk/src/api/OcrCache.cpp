// Copyright 2026 QuickDesk Authors

#include "OcrCache.h"

#include <QMutexLocker>

namespace quickdesk {

OcrCache& OcrCache::instance() {
    static OcrCache s_instance;
    return s_instance;
}

bool OcrCache::get(const QString& frameHash, OcrResult& out) const {
    if (frameHash.isEmpty()) return false;
    QMutexLocker lock(&m_mutex);
    auto it = m_data.find(frameHash);
    if (it == m_data.end()) return false;
    out = it.value();
    return true;
}

void OcrCache::put(const QString& frameHash, OcrResult result) {
    if (frameHash.isEmpty()) return;
    QMutexLocker lock(&m_mutex);

    if (m_data.contains(frameHash)) {
        // 已存在，直接更新（不调整队列顺序，可接受）
        m_data[frameHash] = std::move(result);
        return;
    }

    // 超出容量时淘汰最旧条目
    while (m_data.size() >= MAX_ENTRIES && !m_order.isEmpty()) {
        m_data.remove(m_order.dequeue());
    }

    m_data.insert(frameHash, std::move(result));
    m_order.enqueue(frameHash);
}

void OcrCache::clear() {
    QMutexLocker lock(&m_mutex);
    m_data.clear();
    m_order.clear();
}

int OcrCache::size() const {
    QMutexLocker lock(&m_mutex);
    return m_data.size();
}

} // namespace quickdesk
