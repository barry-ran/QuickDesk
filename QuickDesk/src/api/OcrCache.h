// Copyright 2026 QuickDesk Authors
// 基于帧 hash 的 OCR 结果缓存，避免对同一帧重复识别

#ifndef QUICKDESK_API_OCRCACHE_H
#define QUICKDESK_API_OCRCACHE_H

#include "OcrEngine.h"

#include <QMutex>
#include <QHash>
#include <QQueue>

namespace quickdesk {

// ---------------------------------------------------------------------------
// OcrCache — 线程安全的 LRU 缓存
//
// Key: frameHash（由 OcrEngine::computeFrameHash 计算）
// Value: OcrResult
// 最大条目数: MAX_ENTRIES（默认 30，覆盖约 30 个不同帧）
// ---------------------------------------------------------------------------
class OcrCache {
public:
    static OcrCache& instance();

    // 查询缓存；返回 true 表示命中，out 被填充
    bool get(const QString& frameHash, OcrResult& out) const;

    // 写入缓存
    void put(const QString& frameHash, OcrResult result);

    // 清空所有缓存（连接断开时调用）
    void clear();

    // 当前缓存条目数
    int size() const;

private:
    OcrCache() = default;
    OcrCache(const OcrCache&) = delete;
    OcrCache& operator=(const OcrCache&) = delete;

    static constexpr int MAX_ENTRIES = 30;

    mutable QMutex            m_mutex;
    QHash<QString, OcrResult> m_data;
    QQueue<QString>           m_order;  // LRU 淘汰队列（front = 最旧）
};

} // namespace quickdesk

#endif // QUICKDESK_API_OCRCACHE_H
