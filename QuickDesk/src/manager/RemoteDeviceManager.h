#pragma once

#include <QObject>
#include <QString>
#include <QVector>
#include <QVariantList>
#include "core/common/commontype.h"

namespace core {
class UserDataCenter;
}

namespace quickdesk {

/**
 * @brief Manager for remote device history and credentials
 * 
 * This class handles:
 * - Storing and retrieving remote device connection history
 * - Encrypting/decrypting access passwords
 * - Managing device list (add, remove, update)
 * - Automatic cleanup of old devices (keeps max 50)
 */
class RemoteDeviceManager : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QVariantList deviceList READ deviceList NOTIFY deviceListChanged)

public:
    explicit RemoteDeviceManager(QObject *parent = nullptr);
    ~RemoteDeviceManager() override = default;

    // Initialize database
    bool init();

    // Add or update a device after successful connection
    Q_INVOKABLE bool saveDevice(const QString& deviceId, const QString& password, const QString& deviceName = QString());

    // Remove a device from history
    Q_INVOKABLE bool removeDevice(const QString& deviceId);

    // Get device password (decrypted)
    Q_INVOKABLE QString getDevicePassword(const QString& deviceId);

    // Get all devices for UI display (sorted by last connected time)
    QVariantList deviceList() const;

    // Update device last connected time
    Q_INVOKABLE void updateDeviceConnected(const QString& deviceId);

    // Update device password (for access code sync from cloud)
    Q_INVOKABLE bool updateDevicePassword(const QString& deviceId, const QString& newPassword);

signals:
    void deviceListChanged();
    void deviceAdded(const QString& deviceId);
    void deviceRemoved(const QString& deviceId);

private:
    // Encrypt password using simple XOR + Base64 (Qt built-in)
    QString encryptPassword(const QString& password) const;
    
    // Decrypt password
    QString decryptPassword(const QString& encryptedPassword) const;

    // Clean old devices (keep max 50 non-favorite devices)
    void cleanOldDevices();

    // Convert RemoteDevice to QVariantMap for QML
    QVariantMap deviceToVariant(const core::RemoteDevice& device) const;

    core::UserDataCenter& m_dataCenter;
    QVector<core::RemoteDevice> m_devices;
    
    static constexpr int MAX_DEVICE_COUNT = 50;
    // Simple XOR key for password encryption
    static constexpr char ENCRYPTION_KEY[] = "QuickDeskRemote2024";
};

} // namespace quickdesk
