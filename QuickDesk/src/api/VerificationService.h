// Copyright 2026 QuickDesk Authors
// Post-action verification and screen diff service

#ifndef QUICKDESK_API_VERIFICATIONSERVICE_H
#define QUICKDESK_API_VERIFICATIONSERVICE_H

#include "OcrEngine.h"

#include <QJsonObject>
#include <QString>
#include <QList>

namespace quickdesk {

class MainController;

// ---------------------------------------------------------------------------
// Condition types supported by verifyActionResult / assertScreenState
// ---------------------------------------------------------------------------
//   text_present            value = text to find (partial, case-insensitive by default)
//   text_absent             value = text that must NOT be on screen
//   text_present_exact      value = text to find (exact, case-sensitive)
//   window_title_contains   value = substring of active window title
//   window_title_equals     value = exact active window title
// ---------------------------------------------------------------------------

struct VerificationCondition {
    QString type;   // see above
    QString value;
};

struct ConditionResult {
    VerificationCondition condition;
    bool  passed  = false;
    QString actual;   // actual text/title found (empty if nothing matched)
    QString reason;   // human-readable explanation
};

struct VerificationResult {
    bool allPassed   = false;
    bool timedOut    = false;
    QList<ConditionResult> results;
    QString summary;  // one-line description for Agent
};

// ---------------------------------------------------------------------------
// ScreenDiff — comparison between two OCR snapshots
// ---------------------------------------------------------------------------

struct ScreenDiff {
    QString fromHash;
    QString toHash;
    QList<OcrTextBlock> added;    // blocks present in 'to' but not 'from'
    QList<OcrTextBlock> removed;  // blocks present in 'from' but not 'to'
    bool    hasChanges = false;
    QString summary;
};

// ---------------------------------------------------------------------------
// VerificationService
// ---------------------------------------------------------------------------

class VerificationService {
public:
    explicit VerificationService(MainController* controller);

    // Check all conditions against the current screen.
    // If not all pass, polls until timeoutMs elapses (0 = no wait).
    // Returns the verification result.
    VerificationResult verifyActionResult(
        const QString& connectionId,
        const QList<VerificationCondition>& conditions,
        int timeoutMs);

    // Immediately check all conditions; no polling.
    VerificationResult assertScreenState(
        const QString& connectionId,
        const QList<VerificationCondition>& conditions);

    // Compare the OCR snapshot stored under fromHash with the current frame.
    // If fromHash is empty, captures a fresh snapshot as the "from" state.
    ScreenDiff screenDiffSummary(
        const QString& connectionId,
        const QString& fromHash,
        QString& errorOut);

    // ---- helpers used by ApiHandler ----
    static VerificationCondition conditionFromJson(const QJsonObject& obj);
    static QJsonObject conditionResultToJson(const ConditionResult& r);
    static QJsonObject verificationResultToJson(const VerificationResult& r);
    static QJsonObject screenDiffToJson(const ScreenDiff& d);

private:
    // Evaluate a single condition against the given OCR result + window title.
    static ConditionResult evaluate(const VerificationCondition& cond,
                                    const OcrResult& ocr,
                                    const QString& windowTitle);

    // Run OCR on the current frame (using cache).
    bool runOcr(const QString& connectionId,
                OcrResult& result,
                QString& errorOut);

    // Get active window title (Windows only, empty on other platforms).
    static QString activeWindowTitle();

    MainController* m_controller;
};

} // namespace quickdesk

#endif // QUICKDESK_API_VERIFICATIONSERVICE_H
