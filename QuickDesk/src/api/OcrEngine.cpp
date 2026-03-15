// Copyright 2026 QuickDesk Authors

#include "OcrEngine.h"

#include "infra/log/log.h"

#include <QCoreApplication>
#include <QCryptographicHash>
#include <QDir>
#include <QFile>
#include <QMutexLocker>

#ifdef QUICKDESK_OCR_ENABLED
#include "OcrLiteCApi.h"
#endif

namespace quickdesk {

// ---------------------------------------------------------------------------
// Singleton
// ---------------------------------------------------------------------------
OcrEngine& OcrEngine::instance() {
    static OcrEngine s_instance;
    return s_instance;
}

OcrEngine::~OcrEngine() {
#ifdef QUICKDESK_OCR_ENABLED
    if (m_handle) {
        OcrDestroy(m_handle);
        m_handle = nullptr;
    }
#endif
}

// ---------------------------------------------------------------------------
// 默认模型路径（相对于可执行文件 / macOS Bundle）
// ---------------------------------------------------------------------------
QString OcrEngine::defaultModelDir() {
    QString base = QCoreApplication::applicationDirPath();
#if defined(Q_OS_MACOS)
    // QuickDesk.app/Contents/MacOS/ -> Contents/Resources/models
    base += "/../Resources";
#endif
    return base + "/models";
}

// ---------------------------------------------------------------------------
// computeFrameHash（无论是否启用 OCR 都可用）
// ---------------------------------------------------------------------------
QString OcrEngine::computeFrameHash(const QImage& image) {
    if (image.isNull()) return {};
    QCryptographicHash hash(QCryptographicHash::Md5);
    hash.addData(reinterpret_cast<const char*>(image.constBits()),
                 static_cast<qsizetype>(image.sizeInBytes()));
    return QString::fromLatin1(hash.result().toHex());
}

// ---------------------------------------------------------------------------
// OCR 功能实现（仅当 QUICKDESK_OCR_ENABLED 时编译）
// ---------------------------------------------------------------------------
#ifdef QUICKDESK_OCR_ENABLED

bool OcrEngine::initialize(const QString& modelDir) {
    QMutexLocker lock(&m_mutex);

    if (m_handle) {
        LOG_INFO("OcrEngine already initialized, skipping.");
        return true;
    }

    m_modelDir = modelDir.isEmpty() ? defaultModelDir() : modelDir;

    // 优先 PP-OCRv4；若不存在回退到 v3
    auto pickModel = [&](const QString& v4Name, const QString& v3Name) -> QString {
        QString v4 = m_modelDir + "/" + v4Name;
        if (QFile::exists(v4)) return v4;
        QString v3 = m_modelDir + "/" + v3Name;
        if (QFile::exists(v3)) return v3;
        return QString();
    };

    QString det = pickModel("ch_PP-OCRv4_det_infer.onnx", "ch_PP-OCRv3_det_infer.onnx");
    QString rec = pickModel("ch_PP-OCRv4_rec_infer.onnx", "ch_PP-OCRv3_rec_infer.onnx");
    QString cls = m_modelDir + "/ch_ppocr_mobile_v2.0_cls_infer.onnx";
    QString key = m_modelDir + "/ppocr_keys_v1.txt";

    if (det.isEmpty() || rec.isEmpty()) {
        LOG_ERROR("OcrEngine: detection or recognition model not found in: {}",
                  m_modelDir.toStdString());
        return false;
    }
    if (!QFile::exists(cls)) {
        LOG_WARN("OcrEngine: cls model not found, angle detection disabled: {}",
                 cls.toStdString());
        cls.clear();
    }
    if (!QFile::exists(key)) {
        LOG_ERROR("OcrEngine: ppocr_keys_v1.txt not found: {}", key.toStdString());
        return false;
    }

    constexpr int kThreads = 2;
    m_handle = OcrInit(
        det.toUtf8().constData(),
        cls.isEmpty() ? nullptr : cls.toUtf8().constData(),
        rec.toUtf8().constData(),
        key.toUtf8().constData(),
        kThreads
    );

    if (!m_handle) {
        LOG_ERROR("OcrEngine: OcrInit failed (check model files and ONNX Runtime)");
        return false;
    }

    LOG_INFO("OcrEngine initialized. det={} rec={}", det.toStdString(), rec.toStdString());
    return true;
}

bool OcrEngine::isInitialized() const {
    QMutexLocker lock(&m_mutex);
    return m_handle != nullptr;
}

OcrResult OcrEngine::recognize(const QImage& image) {
    if (image.isNull()) return {};

    QMutexLocker lock(&m_mutex);
    if (!m_handle) return {};

    // QImage -> BGR888（Qt 6.2+ 原生支持，无需 OpenCV 转换）
    QImage bgr = image.convertToFormat(QImage::Format_BGR888);
    if (bgr.isNull()) return {};

    OCR_INPUT input{};
    input.data       = const_cast<uchar*>(bgr.bits());
    input.channels   = 3;
    input.width      = bgr.width();
    input.height     = bgr.height();
    input.dataLength = static_cast<long>(bgr.sizeInBytes());
    input.type       = 0;

    OCR_PARAM param{};
    OCR_RESULT raw{};

    OCR_BOOL ok = OcrDetectInput(m_handle, &input, &param, &raw);
    if (!ok) return {};

    OcrResult result;
    result.imageSize = image.size();

    for (unsigned long long i = 0; i < raw.textBlocksLength; ++i) {
        const TEXT_BLOCK& blk = raw.textBlocks[i];
        if (!blk.text || blk.textLength == 0) continue;

        OcrTextBlock tb;
        tb.text       = QString::fromUtf8(reinterpret_cast<const char*>(blk.text),
                                          static_cast<int>(blk.textLength) - 1);
        tb.confidence = static_cast<double>(blk.boxScore);

        if (blk.boxPointLength >= 4) {
            double minX = blk.boxPoint[0].x, maxX = minX;
            double minY = blk.boxPoint[0].y, maxY = minY;
            for (unsigned long long j = 1; j < blk.boxPointLength; ++j) {
                minX = std::min(minX, blk.boxPoint[j].x);
                maxX = std::max(maxX, blk.boxPoint[j].x);
                minY = std::min(minY, blk.boxPoint[j].y);
                maxY = std::max(maxY, blk.boxPoint[j].y);
            }
            tb.bbox   = QRect(QPoint(static_cast<int>(minX), static_cast<int>(minY)),
                              QPoint(static_cast<int>(maxX), static_cast<int>(maxY)));
            tb.center = tb.bbox.center();
        }

        result.blocks.append(tb);
    }

    OcrFreeResult(&raw);
    return result;
}

#else // QUICKDESK_OCR_ENABLED not defined — stub implementations

bool OcrEngine::initialize(const QString&) { return false; }
bool OcrEngine::isInitialized() const { return false; }
OcrResult OcrEngine::recognize(const QImage&) { return {}; }

#endif // QUICKDESK_OCR_ENABLED

} // namespace quickdesk
