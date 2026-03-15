// Copyright 2026 QuickDesk Authors
// OCR engine wrapper around RapidOCR (PP-OCRv4 + ONNX Runtime)

#ifndef QUICKDESK_API_OCRENGINE_H
#define QUICKDESK_API_OCRENGINE_H

#include <QImage>
#include <QList>
#include <QMutex>
#include <QPoint>
#include <QRect>
#include <QSize>
#include <QString>

namespace quickdesk {

struct OcrTextBlock {
    QString text;
    QRect   bbox;    // 文本框包围矩形（原始像素坐标）
    QPoint  center;  // 包围矩形中心
    double  confidence = 0.0;
};

struct OcrResult {
    QSize               imageSize;
    QList<OcrTextBlock> blocks;
    QString             frameHash;  // 由 OcrCache 填充，用于缓存命中判断
};

// ---------------------------------------------------------------------------
// OcrEngine — 单例，封装 RapidOCR C API
//
// 线程安全：recognize() 内部加锁，可从多个线程调用（但会串行执行）
// ---------------------------------------------------------------------------
class OcrEngine {
public:
    static OcrEngine& instance();

    // 初始化 OCR 引擎，加载 PP-OCRv4 模型
    // modelDir: 包含 .onnx 和 ppocr_keys_v1.txt 的目录
    // 返回 false 表示模型文件缺失或加载失败
    bool initialize(const QString& modelDir = QString());

    bool isInitialized() const;

    // 对 image 执行 OCR，返回文本块列表
    // 调用者应先通过 OcrCache 检查缓存，避免重复识别
    OcrResult recognize(const QImage& image);

    // 工具：计算图像内容 hash（用于缓存 key）
    static QString computeFrameHash(const QImage& image);

    // 返回当前使用的模型目录
    QString modelDir() const { return m_modelDir; }

private:
    OcrEngine() = default;
    ~OcrEngine();

    // 禁止拷贝
    OcrEngine(const OcrEngine&) = delete;
    OcrEngine& operator=(const OcrEngine&) = delete;

    static QString defaultModelDir();

    void*    m_handle  = nullptr;  // OCR_HANDLE（opaque pointer）
    QString  m_modelDir;
    mutable QMutex m_mutex;
};

} // namespace quickdesk

#endif // QUICKDESK_API_OCRENGINE_H
