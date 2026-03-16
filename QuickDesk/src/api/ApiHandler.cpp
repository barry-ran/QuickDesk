// Copyright 2026 QuickDesk Authors

#include "ApiHandler.h"
#include "controller/MainController.h"
#include "manager/HostManager.h"
#include "manager/ClientManager.h"
#include "manager/ServerManager.h"
#include "manager/ProcessManager.h"
#include "manager/SharedMemoryManager.h"
#include "common/ProcessStatus.h"
#include "infra/log/log.h"
#include <QBuffer>
#include <QImage>
#include <QJsonArray>
#include <QVideoFrame>

namespace quickdesk {

namespace {

QString processStatusToString(ProcessStatus::Status s) {
    switch (s) {
    case ProcessStatus::NotStarted: return "notStarted";
    case ProcessStatus::Starting:   return "starting";
    case ProcessStatus::Running:    return "running";
    case ProcessStatus::Failed:     return "failed";
    case ProcessStatus::Restarting: return "restarting";
    }
    return "unknown";
}

QString rtcStatusToString(RtcStatus::Status s) {
    switch (s) {
    case RtcStatus::Disconnected: return "disconnected";
    case RtcStatus::Connecting:   return "connecting";
    case RtcStatus::Connected:    return "connected";
    case RtcStatus::Failed:       return "failed";
    }
    return "unknown";
}

} // namespace

ApiHandler::ApiHandler(MainController* controller, QObject* parent)
    : QObject(parent)
    , m_controller(controller)
    , m_uiState(controller)
    , m_verification(controller) {
    registerHandlers();

    connect(m_controller->clientManager(), &ClientManager::clipboardReceived,
            this, [this](const QString& connectionId, const QString& text) {
                m_clipboardCache[connectionId] = text;
            });

    connect(m_controller->clientManager(), &ClientManager::connectionRemoved,
            this, [this](const QString& connectionId) {
                m_clipboardCache.remove(connectionId);
            });

    // 在后台初始化 OCR 引擎（加载 ONNX 模型，约 500ms~1s）
    QMetaObject::invokeMethod(this, [this]() {
        if (!OcrEngine::instance().initialize()) {
            LOG_WARN("ApiHandler: OCR engine initialization failed. "
                     "getScreenText/findElement will return error until models are present.");
        }
    }, Qt::QueuedConnection);
}

void ApiHandler::registerHandlers() {
    // Host info
    m_handlers["getHostInfo"] = [this](const QJsonObject& p) {
        return handleGetHostInfo(p);
    };
    m_handlers["getHostClients"] = [this](const QJsonObject& p) {
        return handleGetHostClients(p);
    };
    m_handlers["refreshAccessCode"] = [this](const QJsonObject& p) {
        return handleRefreshAccessCode(p);
    };
    m_handlers["kickClient"] = [this](const QJsonObject& p) {
        return handleKickClient(p);
    };

    // Status
    m_handlers["getStatus"] = [this](const QJsonObject& p) {
        return handleGetStatus(p);
    };
    m_handlers["getSignalingStatus"] = [this](const QJsonObject& p) {
        return handleGetSignalingStatus(p);
    };

    // Connection management
    m_handlers["listConnections"] = [this](const QJsonObject& p) {
        return handleListConnections(p);
    };
    m_handlers["getConnectionInfo"] = [this](const QJsonObject& p) {
        return handleGetConnectionInfo(p);
    };
    m_handlers["connectToHost"] = [this](const QJsonObject& p) {
        return handleConnectToHost(p);
    };
    m_handlers["disconnectFromHost"] = [this](const QJsonObject& p) {
        return handleDisconnectFromHost(p);
    };
    m_handlers["disconnectAll"] = [this](const QJsonObject& p) {
        return handleDisconnectAll(p);
    };

    // Remote desktop operations
    m_handlers["screenshot"] = [this](const QJsonObject& p) {
        return handleScreenshot(p);
    };
    m_handlers["mouseClick"] = [this](const QJsonObject& p) {
        return handleMouseClick(p);
    };
    m_handlers["mouseDoubleClick"] = [this](const QJsonObject& p) {
        return handleMouseDoubleClick(p);
    };
    m_handlers["mouseMove"] = [this](const QJsonObject& p) {
        return handleMouseMove(p);
    };
    m_handlers["mouseScroll"] = [this](const QJsonObject& p) {
        return handleMouseScroll(p);
    };
    m_handlers["keyboardType"] = [this](const QJsonObject& p) {
        return handleKeyboardType(p);
    };
    m_handlers["keyboardHotkey"] = [this](const QJsonObject& p) {
        return handleKeyboardHotkey(p);
    };
    m_handlers["mouseDrag"] = [this](const QJsonObject& p) {
        return handleMouseDrag(p);
    };
    m_handlers["keyPress"] = [this](const QJsonObject& p) {
        return handleKeyPress(p);
    };
    m_handlers["keyRelease"] = [this](const QJsonObject& p) {
        return handleKeyRelease(p);
    };
    m_handlers["getClipboard"] = [this](const QJsonObject& p) {
        return handleGetClipboard(p);
    };
    m_handlers["setClipboard"] = [this](const QJsonObject& p) {
        return handleSetClipboard(p);
    };
    m_handlers["getScreenSize"] = [this](const QJsonObject& p) {
        return handleGetScreenSize(p);
    };

    // OCR / UI 状态
    m_handlers["getScreenText"] = [this](const QJsonObject& p) {
        return handleGetScreenText(p);
    };
    m_handlers["findElement"] = [this](const QJsonObject& p) {
        return handleFindElement(p);
    };
    m_handlers["clickText"] = [this](const QJsonObject& p) {
        return handleClickText(p);
    };
    m_handlers["getUiState"] = [this](const QJsonObject& p) {
        return handleGetUiState(p);
    };
    m_handlers["waitForText"] = [this](const QJsonObject& p) {
        return handleWaitForText(p);
    };
    m_handlers["assertTextPresent"] = [this](const QJsonObject& p) {
        return handleAssertTextPresent(p);
    };
    // 验证与自愈
    m_handlers["verifyActionResult"] = [this](const QJsonObject& p) {
        return handleVerifyActionResult(p);
    };
    m_handlers["screenDiffSummary"] = [this](const QJsonObject& p) {
        return handleScreenDiffSummary(p);
    };
    m_handlers["assertScreenState"] = [this](const QJsonObject& p) {
        return handleAssertScreenState(p);
    };
}

QJsonObject ApiHandler::handleRequest(const QJsonObject& request) {
    auto method = request["method"].toString();
    if (method.isEmpty()) {
        return makeError(-32600, "Missing 'method' field");
    }

    auto it = m_handlers.find(method);
    if (it == m_handlers.end()) {
        return makeError(-32601, QString("Unknown method: %1").arg(method));
    }

    auto params = request["params"].toObject();
    return it.value()(params);
}

// --- Host Info ---

QJsonObject ApiHandler::handleGetHostInfo(const QJsonObject&) {
    auto* host = m_controller->hostManager();
    QJsonObject data;
    data["deviceId"] = host->deviceId();
    data["accessCode"] = host->accessCode();
    data["signalingState"] = host->signalingState();
    data["signalingRetryCount"] = host->signalingRetryCount();
    data["signalingNextRetryIn"] = host->signalingNextRetryIn();
    data["signalingError"] = host->signalingError();
    data["clientCount"] = host->clientCount();
    data["nextAccessCodeRefreshTime"] =
        m_controller->nextAccessCodeRefreshTime();
    return makeResult(data);
}

QJsonObject ApiHandler::handleGetHostClients(const QJsonObject&) {
    auto* host = m_controller->hostManager();
    QJsonArray clients;
    for (const auto& session : host->connectedClients()) {
        QJsonObject obj;
        obj["connectionId"] = session.connectionId;
        obj["username"] = session.username;
        obj["deviceId"] = session.deviceId;
        obj["deviceName"] = session.deviceName;
        obj["ip"] = session.ip;
        obj["state"] = session.state;
        obj["connectedAt"] = session.connectedAt;
        clients.append(obj);
    }
    QJsonObject data;
    data["clients"] = clients;
    return makeResult(data);
}

QJsonObject ApiHandler::handleRefreshAccessCode(const QJsonObject&) {
    m_controller->refreshAccessCode();
    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

QJsonObject ApiHandler::handleKickClient(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }
    m_controller->hostManager()->kickClient(connectionId);
    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

// --- Status ---

QJsonObject ApiHandler::handleGetStatus(const QJsonObject&) {
    auto* host = m_controller->hostManager();
    auto* client = m_controller->clientManager();
    auto* server = m_controller->serverManager();

    QJsonObject hostObj;
    hostObj["processStatus"] =
        processStatusToString(m_controller->hostProcessStatus());
    hostObj["signalingState"] = host->signalingState();
    hostObj["deviceId"] = host->deviceId();
    hostObj["accessCode"] = host->accessCode();
    hostObj["clientCount"] = host->clientCount();

    QJsonObject clientObj;
    clientObj["processStatus"] =
        processStatusToString(m_controller->clientProcessStatus());
    clientObj["connectionCount"] = client->connectionCount();
    clientObj["activeConnectionId"] = client->activeConnectionId();

    QJsonObject serverObj;
    serverObj["url"] = server->serverUrl();
    serverObj["configured"] = !server->serverUrl().isEmpty();

    QJsonObject data;
    data["host"] = hostObj;
    data["client"] = clientObj;
    data["server"] = serverObj;
    return makeResult(data);
}

QJsonObject ApiHandler::handleGetSignalingStatus(const QJsonObject&) {
    auto* host = m_controller->hostManager();

    QJsonObject hostSig;
    hostSig["state"] = host->signalingState();
    hostSig["retryCount"] = host->signalingRetryCount();
    hostSig["nextRetryIn"] = host->signalingNextRetryIn();
    hostSig["error"] = host->signalingError();

    QJsonObject data;
    data["url"] = m_controller->serverManager()->serverUrl();
    data["hostSignaling"] = hostSig;
    return makeResult(data);
}

// --- Connection Management ---

static QJsonObject connectionInfoToJson(const ConnectionInfo& info) {
    QJsonObject obj;
    obj["connectionId"] = info.connectionId;
    obj["deviceId"] = info.deviceId;
    obj["deviceName"] = info.deviceName;
    obj["rtcState"] = rtcStatusToString(info.rtcState);
    obj["signalingState"] = info.signalingState;
    obj["signalingRetryCount"] = info.signalingRetryCount;
    obj["signalingNextRetryIn"] = info.signalingNextRetryIn;
    obj["signalingError"] = info.signalingError;
    obj["width"] = info.width;
    obj["height"] = info.height;
    obj["connectedAt"] = info.connectedAt;
    return obj;
}

QJsonObject ApiHandler::handleListConnections(const QJsonObject&) {
    auto* client = m_controller->clientManager();
    QJsonArray arr;
    for (const auto& conn : client->connections()) {
        arr.append(connectionInfoToJson(conn));
    }
    QJsonObject data;
    data["connections"] = arr;
    data["activeConnectionId"] = client->activeConnectionId();
    return makeResult(data);
}

QJsonObject ApiHandler::handleGetConnectionInfo(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }
    auto* client = m_controller->clientManager();
    auto info = client->getConnection(connectionId);
    if (info.connectionId.isEmpty()) {
        return makeError(404, QString("Connection not found: %1").arg(connectionId));
    }
    return makeResult(connectionInfoToJson(info));
}

QJsonObject ApiHandler::handleConnectToHost(const QJsonObject& params) {
    auto deviceId = params["deviceId"].toString();
    auto accessCode = params["accessCode"].toString();
    if (deviceId.isEmpty() || accessCode.isEmpty()) {
        return makeError(400, "Missing 'deviceId' or 'accessCode'");
    }

    auto serverUrl = params["serverUrl"].toString();
    auto connectionId = m_controller->connectToRemoteHost(
        deviceId, accessCode, serverUrl);

    bool showWindow = params["showWindow"].toBool(true);
    if (showWindow && !connectionId.isEmpty()) {
        m_controller->showRemoteWindowForConnection(connectionId, deviceId);
    }

    QJsonObject data;
    data["connectionId"] = connectionId;
    return makeResult(data);
}

QJsonObject ApiHandler::handleDisconnectFromHost(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }
    m_controller->disconnectFromRemoteHost(connectionId);
    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

QJsonObject ApiHandler::handleDisconnectAll(const QJsonObject&) {
    m_controller->clientManager()->disconnectAll();
    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

// --- Remote Desktop Operations ---

QJsonObject ApiHandler::handleScreenshot(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    auto* shm = m_controller->clientManager()->sharedMemoryManager();
    if (!shm || !shm->isAttached(connectionId)) {
        return makeError(404, QString("No video frame available for: %1").arg(connectionId));
    }

    QVideoFrame videoFrame = shm->readVideoFrame(connectionId);
    if (!videoFrame.isValid()) {
        return makeError(500, "Failed to read video frame");
    }

    QImage image = videoFrame.toImage();
    if (image.isNull()) {
        return makeError(500, "Failed to convert video frame to image");
    }

    int maxWidth = params["maxWidth"].toInt(0);
    int maxHeight = params["maxHeight"].toInt(0);
    if (maxWidth > 0 && image.width() > maxWidth) {
        image = image.scaledToWidth(maxWidth, Qt::SmoothTransformation);
    }
    if (maxHeight > 0 && image.height() > maxHeight) {
        image = image.scaledToHeight(maxHeight, Qt::SmoothTransformation);
    }

    auto format = params["format"].toString("jpeg");
    int quality = params["quality"].toInt(80);
    const char* imgFormat = (format == "png") ? "PNG" : "JPEG";

    QByteArray bytes;
    QBuffer buffer(&bytes);
    buffer.open(QIODevice::WriteOnly);
    if (!image.save(&buffer, imgFormat, quality)) {
        return makeError(500, "Failed to encode image");
    }

    QJsonObject data;
    data["width"] = image.width();
    data["height"] = image.height();
    data["format"] = format;
    data["data"] = QString::fromLatin1(bytes.toBase64());
    return makeResult(data);
}

QJsonObject ApiHandler::handleMouseClick(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    int x = params["x"].toInt();
    int y = params["y"].toInt();
    auto buttonStr = params["button"].toString("left");

    int button = 1; // left
    if (buttonStr == "right") button = 2;
    else if (buttonStr == "middle") button = 4;

    auto* client = m_controller->clientManager();
    client->sendMouseMove(connectionId, x, y);
    client->sendMousePress(connectionId, x, y, button);
    client->sendMouseRelease(connectionId, x, y, button);

    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

QJsonObject ApiHandler::handleMouseDoubleClick(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    int x = params["x"].toInt();
    int y = params["y"].toInt();
    int button = 1;

    auto* client = m_controller->clientManager();
    client->sendMouseMove(connectionId, x, y);
    client->sendMousePress(connectionId, x, y, button);
    client->sendMouseRelease(connectionId, x, y, button);
    client->sendMousePress(connectionId, x, y, button);
    client->sendMouseRelease(connectionId, x, y, button);

    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

QJsonObject ApiHandler::handleMouseMove(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    int x = params["x"].toInt();
    int y = params["y"].toInt();
    m_controller->clientManager()->sendMouseMove(connectionId, x, y);

    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

QJsonObject ApiHandler::handleMouseScroll(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    int x = params["x"].toInt();
    int y = params["y"].toInt();
    int deltaX = params["deltaX"].toInt(0);
    int deltaY = params["deltaY"].toInt(0);
    m_controller->clientManager()->sendMouseWheel(connectionId, x, y, deltaX, deltaY);

    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

QJsonObject ApiHandler::handleKeyboardType(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    auto text = params["text"].toString();
    if (text.isEmpty()) {
        return makeError(400, "Missing 'text'");
    }

    auto* client = m_controller->clientManager();
    // Chromium remoting has clipboard-based text injection via syncClipboard + Ctrl+V,
    // which is the most reliable method for arbitrary text including unicode.
    client->syncClipboard(connectionId, text);

    // Send Ctrl+V to paste
    // Windows scan codes: LCtrl=0x1D, V=0x2F
    int lockStates = 0;
    client->sendKeyPress(connectionId, 0x1D, lockStates);   // Ctrl down
    client->sendKeyPress(connectionId, 0x2F, lockStates);   // V down
    client->sendKeyRelease(connectionId, 0x2F, lockStates); // V up
    client->sendKeyRelease(connectionId, 0x1D, lockStates); // Ctrl up

    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

int ApiHandler::keyNameToScanCode(const QString& keyName) {
    static const QMap<QString, int> kMap = {
        // Modifier keys
        {"ctrl",   0x1D}, {"lctrl",    0x1D}, {"rctrl",  0xE01D},
        {"shift",  0x2A}, {"lshift",   0x2A}, {"rshift", 0x36},
        {"alt",    0x38}, {"lalt",     0x38}, {"ralt",   0xE038},
        {"win",    0xE05B}, {"lwin",   0xE05B}, {"rwin",  0xE05C},
        {"meta",   0xE05B},

        // Common keys
        {"enter",  0x1C}, {"return",   0x1C},
        {"tab",    0x0F},
        {"space",  0x39},
        {"backspace", 0x0E}, {"back",  0x0E},
        {"escape", 0x01}, {"esc",     0x01},
        {"delete", 0xE053}, {"del",   0xE053},
        {"insert", 0xE052}, {"ins",   0xE052},
        {"home",   0xE047},
        {"end",    0xE04F},
        {"pageup", 0xE049}, {"pgup",  0xE049},
        {"pagedown", 0xE051}, {"pgdn", 0xE051},

        // Arrow keys
        {"up",    0xE048}, {"down",  0xE050},
        {"left",  0xE04B}, {"right", 0xE04D},

        // Function keys
        {"f1", 0x3B}, {"f2", 0x3C}, {"f3", 0x3D}, {"f4", 0x3E},
        {"f5", 0x3F}, {"f6", 0x40}, {"f7", 0x41}, {"f8", 0x42},
        {"f9", 0x43}, {"f10", 0x44}, {"f11", 0x57}, {"f12", 0x58},

        // Letters
        {"a", 0x1E}, {"b", 0x30}, {"c", 0x2E}, {"d", 0x20},
        {"e", 0x12}, {"f", 0x21}, {"g", 0x22}, {"h", 0x23},
        {"i", 0x17}, {"j", 0x24}, {"k", 0x25}, {"l", 0x26},
        {"m", 0x32}, {"n", 0x31}, {"o", 0x18}, {"p", 0x19},
        {"q", 0x10}, {"r", 0x13}, {"s", 0x1F}, {"t", 0x14},
        {"u", 0x16}, {"v", 0x2F}, {"w", 0x11}, {"x", 0x2D},
        {"y", 0x15}, {"z", 0x2C},

        // Numbers
        {"0", 0x0B}, {"1", 0x02}, {"2", 0x03}, {"3", 0x04},
        {"4", 0x05}, {"5", 0x06}, {"6", 0x07}, {"7", 0x08},
        {"8", 0x09}, {"9", 0x0A},

        // Punctuation
        {"-", 0x0C}, {"=", 0x0D},
        {"[", 0x1A}, {"]", 0x1B},
        {";", 0x27}, {"'", 0x28},
        {",", 0x33}, {".", 0x34}, {"/", 0x35},
        {"\\", 0x2B}, {"`", 0x29},

        // Special
        {"printscreen", 0xE037}, {"prtsc", 0xE037},
        {"scrolllock", 0x46},
        {"pause", 0xE11D},
        {"capslock", 0x3A},
        {"numlock", 0x45},
    };
    auto it = kMap.find(keyName.toLower().trimmed());
    return (it != kMap.end()) ? it.value() : -1;
}

QJsonObject ApiHandler::handleKeyboardHotkey(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    auto keysArray = params["keys"].toArray();
    if (keysArray.isEmpty()) {
        return makeError(400, "Missing 'keys' array");
    }

    QList<int> scanCodes;
    for (const auto& key : keysArray) {
        int sc = keyNameToScanCode(key.toString());
        if (sc < 0) {
            return makeError(400, QString("Unknown key: '%1'").arg(key.toString()));
        }
        scanCodes.append(sc);
    }

    auto* client = m_controller->clientManager();
    int lockStates = 0;

    for (int sc : scanCodes) {
        client->sendKeyPress(connectionId, sc, lockStates);
    }
    for (int i = scanCodes.size() - 1; i >= 0; --i) {
        client->sendKeyRelease(connectionId, scanCodes[i], lockStates);
    }

    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

QJsonObject ApiHandler::handleMouseDrag(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    int startX = params["startX"].toInt();
    int startY = params["startY"].toInt();
    int endX   = params["endX"].toInt();
    int endY   = params["endY"].toInt();
    auto buttonStr = params["button"].toString("left");

    int button = 1;
    if (buttonStr == "right") button = 2;
    else if (buttonStr == "middle") button = 4;

    auto* client = m_controller->clientManager();
    client->sendMouseMove(connectionId, startX, startY);
    client->sendMousePress(connectionId, startX, startY, button);
    client->sendMouseMove(connectionId, endX, endY);
    client->sendMouseRelease(connectionId, endX, endY, button);

    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

QJsonObject ApiHandler::handleKeyPress(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    auto keyName = params["key"].toString();
    if (keyName.isEmpty()) {
        return makeError(400, "Missing 'key'");
    }

    int sc = keyNameToScanCode(keyName);
    if (sc < 0) {
        return makeError(400, QString("Unknown key: '%1'").arg(keyName));
    }

    m_controller->clientManager()->sendKeyPress(connectionId, sc, 0);

    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

QJsonObject ApiHandler::handleKeyRelease(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    auto keyName = params["key"].toString();
    if (keyName.isEmpty()) {
        return makeError(400, "Missing 'key'");
    }

    int sc = keyNameToScanCode(keyName);
    if (sc < 0) {
        return makeError(400, QString("Unknown key: '%1'").arg(keyName));
    }

    m_controller->clientManager()->sendKeyRelease(connectionId, sc, 0);

    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

QJsonObject ApiHandler::handleGetClipboard(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    QJsonObject data;
    if (m_clipboardCache.contains(connectionId)) {
        data["text"] = m_clipboardCache[connectionId];
    } else {
        data["text"] = "";
        data["note"] = "No clipboard content received yet from remote. "
                       "Clipboard is synced when the remote user copies something.";
    }
    return makeResult(data);
}

QJsonObject ApiHandler::handleSetClipboard(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    auto text = params["text"].toString();
    m_controller->clientManager()->syncClipboard(connectionId, text);

    QJsonObject data;
    data["success"] = true;
    return makeResult(data);
}

QJsonObject ApiHandler::handleGetScreenSize(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty()) {
        return makeError(400, "Missing 'connectionId'");
    }

    auto info = m_controller->clientManager()->getConnection(connectionId);
    if (info.connectionId.isEmpty()) {
        return makeError(404, QString("Connection not found: %1").arg(connectionId));
    }

    QJsonObject data;
    data["width"] = info.width;
    data["height"] = info.height;
    return makeResult(data);
}

// --- OCR / UI 状态 ---

// 辅助函数：将 OcrResult 序列化为 QJsonObject
static QJsonObject ocrResultToJson(const OcrResult& result, const QString& connectionId) {
    QJsonArray blocksArr;
    for (const auto& blk : result.blocks) {
        QJsonObject b;
        b["text"]       = blk.text;
        b["confidence"] = blk.confidence;
        b["bbox"]       = QJsonObject{
            {"x", blk.bbox.x()}, {"y", blk.bbox.y()},
            {"w", blk.bbox.width()}, {"h", blk.bbox.height()}
        };
        b["center"] = QJsonObject{
            {"x", blk.center.x()}, {"y", blk.center.y()}
        };
        blocksArr.append(b);
    }
    QJsonObject data;
    data["connectionId"] = connectionId;
    data["width"]        = result.imageSize.width();
    data["height"]       = result.imageSize.height();
    data["blocks"]       = blocksArr;
    data["frameHash"]    = result.frameHash;
    return data;
}

// 辅助函数：从共享内存读取视频帧并转换为 QImage
static QImage readCurrentFrame(MainController* ctrl, const QString& connectionId, QString& err) {
    auto* shm = ctrl->clientManager()->sharedMemoryManager();
    if (!shm || !shm->isAttached(connectionId)) {
        err = QString("No video frame available for: %1").arg(connectionId);
        return {};
    }
    QVideoFrame vf = shm->readVideoFrame(connectionId);
    if (!vf.isValid()) {
        err = "Failed to read video frame";
        return {};
    }
    QImage img = vf.toImage();
    if (img.isNull()) {
        err = "Failed to convert video frame to QImage";
    }
    return img;
}

QJsonObject ApiHandler::handleGetScreenText(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty())
        return makeError(400, "Missing 'connectionId'");

    if (!OcrEngine::instance().isInitialized())
        return makeError(503, "OCR engine not ready. Check that model files are present in the models/ directory.");

    QString err;
    QImage image = readCurrentFrame(m_controller, connectionId, err);
    if (image.isNull())
        return makeError(404, err);

    // 先查缓存
    QString frameHash = OcrEngine::computeFrameHash(image);
    OcrResult cached;
    if (OcrCache::instance().get(frameHash, cached)) {
        return makeResult(ocrResultToJson(cached, connectionId));
    }

    // 缓存未命中，执行 OCR（同步，约 100~300ms）
    OcrResult result = OcrEngine::instance().recognize(image);
    result.frameHash = frameHash;

    OcrCache::instance().put(frameHash, result);
    return makeResult(ocrResultToJson(result, connectionId));
}

QJsonObject ApiHandler::handleFindElement(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty())
        return makeError(400, "Missing 'connectionId'");

    auto text = params["text"].toString().trimmed();
    if (text.isEmpty())
        return makeError(400, "Missing 'text' to find");

    bool exact      = params["exact"].toBool(false);
    bool ignoreCase = params["ignoreCase"].toBool(true);

    if (!OcrEngine::instance().isInitialized())
        return makeError(503, "OCR engine not ready");

    QString err;
    QImage image = readCurrentFrame(m_controller, connectionId, err);
    if (image.isNull())
        return makeError(404, err);

    // 查缓存或识别
    QString frameHash = OcrEngine::computeFrameHash(image);
    OcrResult result;
    if (!OcrCache::instance().get(frameHash, result)) {
        result = OcrEngine::instance().recognize(image);
        result.frameHash = frameHash;
        OcrCache::instance().put(frameHash, result);
    }

    // 搜索匹配的文本块
    Qt::CaseSensitivity cs = ignoreCase ? Qt::CaseInsensitive : Qt::CaseSensitive;
    QJsonArray matches;
    for (const auto& blk : result.blocks) {
        bool hit = exact ? (blk.text.compare(text, cs) == 0)
                         : blk.text.contains(text, cs);
        if (hit) {
            QJsonObject m;
            m["text"]       = blk.text;
            m["confidence"] = blk.confidence;
            m["bbox"]       = QJsonObject{
                {"x", blk.bbox.x()}, {"y", blk.bbox.y()},
                {"w", blk.bbox.width()}, {"h", blk.bbox.height()}
            };
            m["center"] = QJsonObject{
                {"x", blk.center.x()}, {"y", blk.center.y()}
            };
            matches.append(m);
        }
    }

    QJsonObject data;
    data["connectionId"] = connectionId;
    data["query"]        = text;
    data["found"]        = !matches.isEmpty();
    data["matches"]      = matches;
    data["frameHash"]    = frameHash;
    return makeResult(data);
}

QJsonObject ApiHandler::handleClickText(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty())
        return makeError(400, "Missing 'connectionId'");

    // 1. 先查找文本
    QJsonObject findResult = handleFindElement(params);
    if (findResult.contains("error"))
        return findResult;

    QJsonObject foundData = findResult["result"].toObject();
    QJsonArray matches = foundData["matches"].toArray();
    if (matches.isEmpty()) {
        QJsonObject errData;
        errData["found"]   = false;
        errData["query"]   = params["text"].toString();
        errData["message"] = QString("Text not found on screen: \"%1\"").arg(params["text"].toString());
        return makeResult(errData);
    }

    // 2. 取第一个匹配（最高置信度由 OCR 引擎保证顺序）
    QJsonObject first  = matches[0].toObject();
    QJsonObject center = first["center"].toObject();
    int x = center["x"].toInt();
    int y = center["y"].toInt();

    // 3. 执行鼠标点击
    auto buttonStr = params["button"].toString("left");
    int button = 1;
    if (buttonStr == "right") button = 2;
    else if (buttonStr == "middle") button = 4;

    auto* client = m_controller->clientManager();
    client->sendMouseMove(connectionId, x, y);
    client->sendMousePress(connectionId, x, y, button);
    client->sendMouseRelease(connectionId, x, y, button);

    QJsonObject data;
    data["success"]      = true;
    data["clickedText"]  = first["text"];
    data["x"]            = x;
    data["y"]            = y;
    data["confidence"]   = first["confidence"];
    return makeResult(data);
}

// --- UI 状态 ---

QJsonObject ApiHandler::handleGetUiState(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty())
        return makeError(400, "Missing 'connectionId'");

    UiState state;
    QString err;
    if (!m_uiState.getUiState(connectionId, state, err))
        return makeError(404, err);

    return makeResult(uiStateToJson(state));
}

QJsonObject ApiHandler::handleWaitForText(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty())
        return makeError(400, "Missing 'connectionId'");

    auto text = params["text"].toString().trimmed();
    if (text.isEmpty())
        return makeError(400, "Missing 'text'");

    bool exact      = params["exact"].toBool(false);
    bool ignoreCase = params["ignoreCase"].toBool(true);
    int  timeoutMs  = params["timeoutMs"].toInt(5000);

    OcrTextBlock found;
    QString err;
    bool ok = m_uiState.waitForText(connectionId, text, exact, ignoreCase,
                                    timeoutMs, found, err);

    QJsonObject data;
    data["connectionId"] = connectionId;
    data["query"]        = text;
    data["found"]        = ok;
    if (ok) {
        data["match"] = ocrBlockToJson(found);
    } else if (!err.isEmpty()) {
        return makeError(500, err);
    }
    return makeResult(data);
}

QJsonObject ApiHandler::handleAssertTextPresent(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty())
        return makeError(400, "Missing 'connectionId'");

    auto text = params["text"].toString().trimmed();
    if (text.isEmpty())
        return makeError(400, "Missing 'text'");

    bool exact      = params["exact"].toBool(false);
    bool ignoreCase = params["ignoreCase"].toBool(true);

    OcrTextBlock found;
    QString err;
    bool ok = m_uiState.assertTextPresent(connectionId, text, exact, ignoreCase,
                                          found, err);

    if (!err.isEmpty())
        return makeError(503, err);  // OCR/frame error

    QJsonObject data;
    data["connectionId"] = connectionId;
    data["query"]        = text;
    data["present"]      = ok;
    if (ok) {
        data["match"] = ocrBlockToJson(found);
    }
    return makeResult(data);
}

// --- 验证与自愈 ---

// 辅助：从 JSON Array 解析 conditions 列表
static QList<VerificationCondition> parseConditions(const QJsonArray& arr) {
    QList<VerificationCondition> list;
    for (const auto& v : arr) {
        auto c = VerificationService::conditionFromJson(v.toObject());
        if (!c.type.isEmpty() && !c.value.isEmpty())
            list.append(c);
    }
    return list;
}

QJsonObject ApiHandler::handleVerifyActionResult(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty())
        return makeError(400, "Missing 'connectionId'");

    auto condArr = params["expectations"].toArray();
    if (condArr.isEmpty())
        condArr = params["conditions"].toArray();  // 兼容两种字段名
    if (condArr.isEmpty())
        return makeError(400, "Missing 'expectations' array");

    auto conditions = parseConditions(condArr);
    if (conditions.isEmpty())
        return makeError(400, "No valid conditions in 'expectations'");

    int timeoutMs = params["timeoutMs"].toInt(3000);

    auto vr = m_verification.verifyActionResult(connectionId, conditions, timeoutMs);
    return makeResult(VerificationService::verificationResultToJson(vr));
}

QJsonObject ApiHandler::handleScreenDiffSummary(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty())
        return makeError(400, "Missing 'connectionId'");

    auto fromHash = params["fromHash"].toString();  // empty = no baseline

    QString err;
    auto diff = m_verification.screenDiffSummary(connectionId, fromHash, err);
    if (!err.isEmpty())
        return makeError(500, err);

    return makeResult(VerificationService::screenDiffToJson(diff));
}

QJsonObject ApiHandler::handleAssertScreenState(const QJsonObject& params) {
    auto connectionId = params["connectionId"].toString();
    if (connectionId.isEmpty())
        return makeError(400, "Missing 'connectionId'");

    auto condArr = params["expectations"].toArray();
    if (condArr.isEmpty())
        condArr = params["conditions"].toArray();
    if (condArr.isEmpty())
        return makeError(400, "Missing 'expectations' array");

    auto conditions = parseConditions(condArr);
    if (conditions.isEmpty())
        return makeError(400, "No valid conditions in 'expectations'");

    auto vr = m_verification.assertScreenState(connectionId, conditions);
    return makeResult(VerificationService::verificationResultToJson(vr));
}

// --- Helpers ---

QJsonObject ApiHandler::makeResult(const QJsonObject& data) {
    QJsonObject resp;
    resp["result"] = data;
    return resp;
}

QJsonObject ApiHandler::makeError(int code, const QString& message) {
    QJsonObject resp;
    resp["error"] = QJsonObject{
        {"code", code},
        {"message", message}
    };
    return resp;
}

} // namespace quickdesk
