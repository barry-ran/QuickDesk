// Copyright 2026 QuickDesk Authors
#ifndef QUICKDESK_MANAGER_CLOUDDEVICEMANAGER_H
#define QUICKDESK_MANAGER_CLOUDDEVICEMANAGER_H

#include <functional>

#include <QObject>
#include <QString>
#include <QHash>
#include <QVariantList>
#include <QWebSocket>

namespace quickdesk {

class ServerManager;
class AuthManager;
class HostManager;

/**
 * CloudDeviceManager — user-scoped device, favorite, connection-history
 * surface against /v1/me/* and the realtime events WebSocket
 * /v1/realtime/events (§2.2 / §2.7 / §2.8).
 *
 * State management:
 *   - The local snapshot (m_myDevices, m_myFavorites) is replaced
 *     wholesale on the WS `snapshot` frame (§2.8 bootstrap or
 *     `snapshot_required`). Incremental events (`device.*`, `favorite.*`)
 *     patch in place — no fetchMyDevices on every event (avoids
 *     thundering-herd refetch storms).
 *   - server_rev is tracked so reconnects send `{type:"resume", since_rev}`.
 *
 * Access code upload (§2.23 syncAccessCode):
 *   - URL is `/v1/devices/:id/access-code` (NO `/me/` prefix).
 *   - Auth header is `Authorization: Bearer <device_secret>`, NOT the
 *     user access_token. device_secret is delivered runtime by the
 *     Chromium host through native-messaging (HostManager); Qt holds it
 *     in memory only (§2.22 — never persisted).
 *   - Failure cases: silently retried on signaling reconnect; no UI alert.
 */
class CloudDeviceManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(QVariantList myDevices READ myDevices NOTIFY myDevicesChanged)
    Q_PROPERTY(QVariantList myFavorites READ myFavorites NOTIFY myFavoritesChanged)
    Q_PROPERTY(QVariantList connectionLogs READ connectionLogs NOTIFY connectionLogsChanged)

public:
    explicit CloudDeviceManager(ServerManager* serverManager,
                                  AuthManager* authManager,
                                  HostManager* hostManager,
                                  QObject* parent = nullptr);
    ~CloudDeviceManager() override;

    // My Devices
    Q_INVOKABLE void fetchMyDevices();
    Q_INVOKABLE void autoBindDevice(const QString& deviceId);
    Q_INVOKABLE void unbindDevice(const QString& deviceId);
    Q_INVOKABLE void setDeviceRemark(const QString& deviceId, const QString& remark);

    // §2.23: access_code upload. Uses device_secret (Bearer) + X-API-Key.
    // Host delivers device_secret through native-messaging at hostReady;
    // CloudDeviceManager keeps the value in memory only (HostManager owns it).
    Q_INVOKABLE void syncAccessCode(const QString& deviceId, const QString& accessCode);

    // §2.6: verify access_code and mint a one-shot signaling token for
    // the client process. POST /v1/devices/:id/access-code:verify;
    // on HTTP 200 invokes |onSuccess(signalToken)|, on any failure
    // invokes |onError(httpStatus, code, detail)|. Both callbacks are
    // posted back to this object's thread so callers may update Qt UI
    // state without extra marshalling.
    //
    // Auth: this endpoint accepts X-API-Key OR an Origin-whitelisted
    // caller (§2.2 H1). Qt always attaches X-API-Key via
    // AuthManager::publicHeaders so user login state is irrelevant.
    void verifyAccessCode(
        const QString& deviceId,
        const QString& accessCode,
        std::function<void(const QString& signalToken)> onSuccess,
        std::function<void(int httpStatus,
                            const QString& code,
                            const QString& detail)> onError);

    Q_INVOKABLE QString getDeviceAccessCode(const QString& deviceId) const;
    Q_INVOKABLE QString getDeviceDisplayName(const QString& deviceId) const;

    void refreshDeviceAccessCodeNow(
        const QString& deviceId,
        std::function<void(const QString& accessCode)> onSuccess,
        std::function<void(int httpStatus,
                            const QString& code,
                            const QString& detail)> onError);

    // Connection record
    Q_INVOKABLE void recordConnection(const QString& deviceId, int duration,
                                       const QString& status, const QString& errorMsg = QString());
    Q_INVOKABLE void fetchConnectionLogs();

    // My Favorites
    Q_INVOKABLE void fetchFavorites();
    Q_INVOKABLE void addFavorite(const QString& deviceId, const QString& name, const QString& password);
    Q_INVOKABLE void updateFavorite(const QString& deviceId, const QString& name, const QString& password);
    Q_INVOKABLE void removeFavorite(const QString& deviceId);

    // Realtime events WebSocket (§2.8)
    void startSync();
    void stopSync();

    QVariantList myDevices() const;
    QVariantList myFavorites() const;
    QVariantList connectionLogs() const;

signals:
    void myDevicesChanged();
    void myFavoritesChanged();
    void connectionLogsChanged();
    void syncMessage(const QJsonObject& msg);
    // syncConnected fires when the realtime events WS has finished its
    // first-frame auth_ok handshake AND completed snapshot/replay
    // bootstrap. Subscribers (MainController.autoBindDevice) can now
    // safely act; subsequent device.* events are newer than the local
    // baseline and are applied by server_rev.
    void syncConnected();
    void syncDisconnected();

private slots:
    void onSyncTextMessageReceived(const QString& message);
    void onSyncDisconnected();
    void onSyncConnected();

private:
    QString httpBaseUrl() const;

    // Headers for device-level (Bearer device_secret) requests.
    QList<QPair<QString, QString>> deviceSecretHeaders() const;

    void handleEventFrame(const QJsonObject& msg);
    void handleSnapshotFrame(const QJsonObject& msg);
    void applyDeviceEvent(const QString& type, const QJsonObject& data);
    void applyFavoriteEvent(const QString& type, const QJsonObject& data);
    void refreshDeviceAccessCode(const QString& deviceId, qint64 eventRev);
    void sendAuthFrame();
    bool shouldApplyWireFrame(const QString& type, qint64 rev);
    void setDeviceRev(const QString& deviceId, qint64 rev);
    qint64 deviceRev(const QString& deviceId) const;

    ServerManager* m_serverManager;
    AuthManager*   m_authManager;
    HostManager*   m_hostManager;

    QWebSocket* m_syncSocket = nullptr;
    bool m_syncAuthOk = false;
    bool m_syncBootstrapComplete = false;
    qint64 m_serverRev = 0;
    qint64 m_lastAppliedRev = 0;
    int m_reconnectAttempt = 0;

    QVariantList m_myDevices;
    QVariantList m_myFavorites;
    QVariantList m_connectionLogs;
    QHash<QString, qint64> m_deviceRevs;

    // De-dup guard for syncAccessCode (R28): three MainController
    // listeners (hostReady / accessCodeChanged / deviceSecretReady) can
    // each fire within the same host-ready burst, each asking us to
    // upload the *same* code. Remember what we last pushed for this
    // device so we don't PUT 3× in a row — the second/third call is a
    // no-op which also suppresses the duplicate
    // `device.access_code.changed` realtime event.
    QString m_lastSyncedDeviceId;
    QString m_lastSyncedAccessCode;
};

} // namespace quickdesk

#endif // QUICKDESK_MANAGER_CLOUDDEVICEMANAGER_H
