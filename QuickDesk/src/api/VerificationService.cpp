// Copyright 2026 QuickDesk Authors

#include "VerificationService.h"
#include "OcrCache.h"
#include "controller/MainController.h"
#include "manager/ClientManager.h"
#include "manager/SharedMemoryManager.h"
#include "infra/log/log.h"

#include <QJsonArray>
#include <QJsonObject>
#include <QThread>
#include <QVideoFrame>

#ifdef Q_OS_WIN
#include <windows.h>
#endif

namespace quickdesk {

// ---------------------------------------------------------------------------
// VerificationService
// ---------------------------------------------------------------------------

VerificationService::VerificationService(MainController* controller)
    : m_controller(controller)
{}

// ---------------------------------------------------------------------------
// OCR helpers
// ---------------------------------------------------------------------------

bool VerificationService::runOcr(const QString& connectionId,
                                  OcrResult& result,
                                  QString& errorOut)
{
    if (!OcrEngine::instance().isInitialized()) {
        errorOut = "OCR engine not ready";
        return false;
    }

    auto* shm = m_controller->clientManager()->sharedMemoryManager();
    if (!shm || !shm->isAttached(connectionId)) {
        errorOut = QString("No video frame available for: %1").arg(connectionId);
        return false;
    }

    QVideoFrame vf = shm->readVideoFrame(connectionId);
    if (!vf.isValid()) { errorOut = "Failed to read video frame"; return false; }

    QImage image = vf.toImage();
    if (image.isNull()) { errorOut = "Failed to convert frame to image"; return false; }

    QString hash = OcrEngine::computeFrameHash(image);
    OcrResult cached;
    if (OcrCache::instance().get(hash, cached)) {
        result = cached;
        return true;
    }

    result           = OcrEngine::instance().recognize(image);
    result.frameHash = hash;
    OcrCache::instance().put(hash, result);
    return true;
}

// static
QString VerificationService::activeWindowTitle()
{
#ifdef Q_OS_WIN
    HWND hwnd = GetForegroundWindow();
    if (!hwnd) return {};
    int len = GetWindowTextLengthW(hwnd);
    if (len <= 0) return {};
    std::wstring buf(static_cast<size_t>(len + 1), L'\0');
    GetWindowTextW(hwnd, buf.data(), len + 1);
    return QString::fromStdWString(buf);
#else
    return {};
#endif
}

// ---------------------------------------------------------------------------
// Condition evaluation
// ---------------------------------------------------------------------------

// static
ConditionResult VerificationService::evaluate(const VerificationCondition& cond,
                                               const OcrResult& ocr,
                                               const QString& windowTitle)
{
    ConditionResult r;
    r.condition = cond;

    const QString& type  = cond.type;
    const QString& value = cond.value;

    if (type == "text_present") {
        for (const auto& blk : ocr.blocks) {
            if (blk.text.contains(value, Qt::CaseInsensitive)) {
                r.passed = true;
                r.actual = blk.text;
                r.reason = QString("Found \"%1\" in block \"%2\"").arg(value, blk.text);
                return r;
            }
        }
        r.reason = QString("Text \"%1\" not found on screen").arg(value);

    } else if (type == "text_absent") {
        for (const auto& blk : ocr.blocks) {
            if (blk.text.contains(value, Qt::CaseInsensitive)) {
                r.actual = blk.text;
                r.reason = QString("Text \"%1\" is still present: \"%2\"").arg(value, blk.text);
                return r;
            }
        }
        r.passed = true;
        r.reason = QString("Text \"%1\" is absent (as expected)").arg(value);

    } else if (type == "text_present_exact") {
        for (const auto& blk : ocr.blocks) {
            if (blk.text.compare(value, Qt::CaseSensitive) == 0) {
                r.passed = true;
                r.actual = blk.text;
                r.reason = QString("Exact match \"%1\" found").arg(value);
                return r;
            }
        }
        r.reason = QString("Exact text \"%1\" not found on screen").arg(value);

    } else if (type == "window_title_contains") {
        if (windowTitle.contains(value, Qt::CaseInsensitive)) {
            r.passed = true;
            r.actual = windowTitle;
            r.reason = QString("Window title \"%1\" contains \"%2\"").arg(windowTitle, value);
        } else {
            r.actual = windowTitle;
            r.reason = QString("Window title \"%1\" does not contain \"%2\"").arg(windowTitle, value);
        }

    } else if (type == "window_title_equals") {
        if (windowTitle.compare(value, Qt::CaseSensitive) == 0) {
            r.passed = true;
            r.actual = windowTitle;
            r.reason = QString("Window title matches \"%1\"").arg(value);
        } else {
            r.actual = windowTitle;
            r.reason = QString("Window title is \"%1\", expected \"%2\"").arg(windowTitle, value);
        }

    } else {
        r.reason = QString("Unknown condition type: \"%1\"").arg(type);
    }

    return r;
}

// ---------------------------------------------------------------------------
// verifyActionResult — poll until all conditions pass or timeout
// ---------------------------------------------------------------------------

VerificationResult VerificationService::verifyActionResult(
    const QString& connectionId,
    const QList<VerificationCondition>& conditions,
    int timeoutMs)
{
    constexpr int kPollMs = 200;
    int elapsed = 0;
    VerificationResult vr;

    while (true) {
        OcrResult ocr;
        QString err;
        if (!runOcr(connectionId, ocr, err)) {
            vr.summary = QString("OCR error: %1").arg(err);
            return vr;
        }
        QString title = activeWindowTitle();

        vr.results.clear();
        bool allPass = true;
        for (const auto& cond : conditions) {
            ConditionResult r = evaluate(cond, ocr, title);
            if (!r.passed) allPass = false;
            vr.results.append(r);
        }

        if (allPass) {
            vr.allPassed = true;
            vr.summary   = QString("All %1 condition(s) passed").arg(conditions.size());
            return vr;
        }

        if (elapsed >= timeoutMs) {
            vr.timedOut = false;  // we simply ran out of retries
            // build failure summary
            QStringList failures;
            for (const auto& r : vr.results) {
                if (!r.passed) failures << r.reason;
            }
            vr.summary = QString("Verification failed: %1").arg(failures.join("; "));
            return vr;
        }

        int wait = qMin(kPollMs, timeoutMs - elapsed);
        QThread::msleep(static_cast<unsigned long>(wait));
        elapsed += kPollMs;
    }
}

// ---------------------------------------------------------------------------
// assertScreenState — immediate, no polling
// ---------------------------------------------------------------------------

VerificationResult VerificationService::assertScreenState(
    const QString& connectionId,
    const QList<VerificationCondition>& conditions)
{
    VerificationResult vr;
    OcrResult ocr;
    QString err;
    if (!runOcr(connectionId, ocr, err)) {
        vr.summary = QString("OCR error: %1").arg(err);
        return vr;
    }
    QString title = activeWindowTitle();

    bool allPass = true;
    for (const auto& cond : conditions) {
        ConditionResult r = evaluate(cond, ocr, title);
        if (!r.passed) allPass = false;
        vr.results.append(r);
    }

    vr.allPassed = allPass;
    if (allPass) {
        vr.summary = QString("All %1 condition(s) satisfied").arg(conditions.size());
    } else {
        QStringList failures;
        for (const auto& r : vr.results) {
            if (!r.passed) failures << r.reason;
        }
        vr.summary = QString("Assertion failed: %1").arg(failures.join("; "));
    }
    return vr;
}

// ---------------------------------------------------------------------------
// screenDiffSummary
// ---------------------------------------------------------------------------

ScreenDiff VerificationService::screenDiffSummary(
    const QString& connectionId,
    const QString& fromHash,
    QString& errorOut)
{
    ScreenDiff diff;
    diff.fromHash = fromHash;

    // Fetch "to" state (current frame)
    OcrResult toOcr;
    if (!runOcr(connectionId, toOcr, errorOut)) return diff;
    diff.toHash = toOcr.frameHash;

    // Identical frame — nothing to diff
    if (diff.fromHash == diff.toHash) {
        diff.summary = "No screen change detected (same frame)";
        return diff;
    }

    // Fetch "from" state from cache (may be empty if evicted)
    OcrResult fromOcr;
    bool hasFrome = !fromHash.isEmpty() && OcrCache::instance().get(fromHash, fromOcr);

    if (!hasFrome) {
        // Can't diff without the baseline — just describe the current state
        diff.added    = toOcr.blocks;
        diff.hasChanges = true;
        diff.summary  = QString("No baseline available for hash \"%1\"; "
                                "current frame has %2 text block(s)")
                        .arg(fromHash).arg(toOcr.blocks.size());
        return diff;
    }

    // Build lookup sets by text content
    QSet<QString> fromTexts, toTexts;
    for (const auto& b : fromOcr.blocks) fromTexts.insert(b.text);
    for (const auto& b : toOcr.blocks)   toTexts.insert(b.text);

    for (const auto& b : toOcr.blocks) {
        if (!fromTexts.contains(b.text)) diff.added.append(b);
    }
    for (const auto& b : fromOcr.blocks) {
        if (!toTexts.contains(b.text)) diff.removed.append(b);
    }

    diff.hasChanges = !diff.added.isEmpty() || !diff.removed.isEmpty();

    if (!diff.hasChanges) {
        diff.summary = "Screen text content unchanged (same text blocks, possibly repositioned)";
    } else {
        QStringList parts;
        if (!diff.added.isEmpty()) {
            QStringList texts;
            for (const auto& b : diff.added) texts << QString("\"%1\"").arg(b.text);
            parts << QString("appeared: %1").arg(texts.join(", "));
        }
        if (!diff.removed.isEmpty()) {
            QStringList texts;
            for (const auto& b : diff.removed) texts << QString("\"%1\"").arg(b.text);
            parts << QString("disappeared: %1").arg(texts.join(", "));
        }
        diff.summary = parts.join("; ");
    }

    return diff;
}

// ---------------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------------

// static
VerificationCondition VerificationService::conditionFromJson(const QJsonObject& obj)
{
    VerificationCondition c;
    c.type  = obj["type"].toString();
    c.value = obj["value"].toString();
    return c;
}

// static
QJsonObject VerificationService::conditionResultToJson(const ConditionResult& r)
{
    return QJsonObject{
        {"type",   r.condition.type},
        {"value",  r.condition.value},
        {"passed", r.passed},
        {"actual", r.actual},
        {"reason", r.reason}
    };
}

// static
QJsonObject VerificationService::verificationResultToJson(const VerificationResult& vr)
{
    QJsonArray arr;
    for (const auto& r : vr.results) arr.append(conditionResultToJson(r));
    return QJsonObject{
        {"allPassed", vr.allPassed},
        {"timedOut",  vr.timedOut},
        {"results",   arr},
        {"summary",   vr.summary}
    };
}

static QJsonObject ocrBlockToJsonV(const OcrTextBlock& blk)
{
    return QJsonObject{
        {"text",       blk.text},
        {"confidence", blk.confidence},
        {"bbox",       QJsonObject{
            {"x", blk.bbox.x()}, {"y", blk.bbox.y()},
            {"w", blk.bbox.width()}, {"h", blk.bbox.height()}
        }},
        {"center", QJsonObject{{"x", blk.center.x()}, {"y", blk.center.y()}}}
    };
}

// static
QJsonObject VerificationService::screenDiffToJson(const ScreenDiff& d)
{
    auto blocksToArr = [](const QList<OcrTextBlock>& list) {
        QJsonArray a;
        for (const auto& b : list) a.append(ocrBlockToJsonV(b));
        return a;
    };

    return QJsonObject{
        {"fromHash",   d.fromHash},
        {"toHash",     d.toHash},
        {"hasChanges", d.hasChanges},
        {"added",      blocksToArr(d.added)},
        {"removed",    blocksToArr(d.removed)},
        {"summary",    d.summary}
    };
}

} // namespace quickdesk
