#include "RemoteDeviceManager.h"
#include "core/userdatacenter.h"
#include "infra/log/log.h"

#include <QDateTime>
#include <QVariantMap>

namespace quickdesk {

constexpr char RemoteDeviceManager::ENCRYPTION_KEY[];

RemoteDeviceManager::RemoteDeviceManager(QObject *parent)
    : QObject(parent)
    , m_dataCenter(core::UserDataCenter::instance())
{
}

bool RemoteDeviceManager::init()
{
    // Load all devices
    if (!m_dataCenter.getAllRemoteDevices(m_devices)) {
        LOG_ERROR("[RemoteDeviceManager] Failed to load devices");
        return false;
    }

    LOG_INFO("[RemoteDeviceManager] Initialized with {} devices", m_devices.size());
    
    // Emit signal to notify QML that device list is ready
    emit deviceListChanged();
    
    return true;
}

bool RemoteDeviceManager::saveDevice(const QString& deviceId, const QString& password, const QString& deviceName)
{
    if (deviceId.isEmpty() || password.isEmpty()) {
        LOG_WARN("[RemoteDeviceManager] Cannot save device with empty ID or password");
        return false;
    }

    core::RemoteDevice device;
    device.deviceId = deviceId;
    device.deviceName = deviceName.isEmpty() ? deviceId : deviceName;
    device.accessPassword = encryptPassword(password);
    device.isFavorite = false;
    device.lastConnectedTime = QDateTime::currentMSecsSinceEpoch();
    device.createdTime = QDateTime::currentMSecsSinceEpoch();

    if (!m_dataCenter.addOrUpdateRemoteDevice(device)) {
        LOG_ERROR("[RemoteDeviceManager] Failed to save device: {}", deviceId.toStdString());
        return false;
    }

    // Reload device list
    m_devices.clear();
    if (!m_dataCenter.getAllRemoteDevices(m_devices)) {
        LOG_ERROR("[RemoteDeviceManager] Failed to reload devices");
        return false;
    }

    // Clean old devices if exceeded limit
    cleanOldDevices();

    emit deviceListChanged();
    emit deviceAdded(deviceId);

    LOG_INFO("[RemoteDeviceManager] Saved device: {}", deviceId.toStdString());
    return true;
}

bool RemoteDeviceManager::removeDevice(const QString& deviceId)
{
    if (deviceId.isEmpty()) {
        return false;
    }

    if (!m_dataCenter.removeRemoteDevice(deviceId)) {
        LOG_ERROR("[RemoteDeviceManager] Failed to remove device: {}", deviceId.toStdString());
        return false;
    }

    // Reload device list
    m_devices.clear();
    if (!m_dataCenter.getAllRemoteDevices(m_devices)) {
        LOG_ERROR("[RemoteDeviceManager] Failed to reload devices");
        return false;
    }

    emit deviceListChanged();
    emit deviceRemoved(deviceId);

    LOG_INFO("[RemoteDeviceManager] Removed device: {}", deviceId.toStdString());
    return true;
}

QString RemoteDeviceManager::getDevicePassword(const QString& deviceId)
{
    core::RemoteDevice device;
    if (m_dataCenter.getRemoteDevice(deviceId, device)) {
        return decryptPassword(device.accessPassword);
    }
    return QString();
}

QVariantList RemoteDeviceManager::deviceList() const
{
    QVariantList list;
    for (const auto& device : m_devices) {
        list.append(deviceToVariant(device));
    }
    return list;
}

void RemoteDeviceManager::updateDeviceConnected(const QString& deviceId)
{
    if (deviceId.isEmpty()) {
        return;
    }

    m_dataCenter.updateDeviceLastConnected(deviceId);

    // Reload device list to reflect new order
    m_devices.clear();
    if (m_dataCenter.getAllRemoteDevices(m_devices)) {
        emit deviceListChanged();
    }
}

bool RemoteDeviceManager::updateDevicePassword(const QString& deviceId, const QString& newPassword)
{
    if (deviceId.isEmpty() || newPassword.isEmpty()) {
        return false;
    }

    // Check if device exists in recent connections
    core::RemoteDevice device;
    if (!m_dataCenter.getRemoteDevice(deviceId, device)) {
        return false;  // Device not in recent connections, nothing to update
    }

    QString encrypted = encryptPassword(newPassword);
    if (!m_dataCenter.updateDevicePassword(deviceId, encrypted)) {
        LOG_ERROR("[RemoteDeviceManager] Failed to update password for device: {}", deviceId.toStdString());
        return false;
    }

    // Reload device list
    m_devices.clear();
    if (m_dataCenter.getAllRemoteDevices(m_devices)) {
        emit deviceListChanged();
    }

    LOG_INFO("[RemoteDeviceManager] Updated password for device: {}", deviceId.toStdString());
    return true;
}

QString RemoteDeviceManager::encryptPassword(const QString& password) const
{
    // Simple XOR encryption with Base64 encoding
    QByteArray passwordBytes = password.toUtf8();
    QByteArray key = QByteArray::fromRawData(ENCRYPTION_KEY, strlen(ENCRYPTION_KEY));
    QByteArray encrypted;

    for (int i = 0; i < passwordBytes.size(); ++i) {
        encrypted.append(passwordBytes[i] ^ key[i % key.size()]);
    }

    return QString::fromLatin1(encrypted.toBase64());
}

QString RemoteDeviceManager::decryptPassword(const QString& encryptedPassword) const
{
    // Decrypt XOR + Base64
    QByteArray encrypted = QByteArray::fromBase64(encryptedPassword.toLatin1());
    QByteArray key = QByteArray::fromRawData(ENCRYPTION_KEY, strlen(ENCRYPTION_KEY));
    QByteArray decrypted;

    for (int i = 0; i < encrypted.size(); ++i) {
        decrypted.append(encrypted[i] ^ key[i % key.size()]);
    }

    return QString::fromUtf8(decrypted);
}

void RemoteDeviceManager::cleanOldDevices()
{
    // Count non-favorite devices
    int nonFavoriteCount = 0;
    for (const auto& device : m_devices) {
        if (!device.isFavorite) {
            nonFavoriteCount++;
        }
    }

    // Clean if exceeded limit
    if (nonFavoriteCount > MAX_DEVICE_COUNT) {
        m_dataCenter.cleanOldDevices(MAX_DEVICE_COUNT);
        
        // Reload device list
        m_devices.clear();
        if (m_dataCenter.getAllRemoteDevices(m_devices)) {
            LOG_INFO("[RemoteDeviceManager] Cleaned old devices, now have {} devices", m_devices.size());
        }
    }
}

QVariantMap RemoteDeviceManager::deviceToVariant(const core::RemoteDevice& device) const
{
    QVariantMap map;
    map["deviceId"] = device.deviceId;
    map["deviceName"] = device.deviceName;
    map["isFavorite"] = device.isFavorite;
    map["connectionCount"] = device.connectionCount;
    map["lastConnectedTime"] = device.lastConnectedTime;
    return map;
}

} // namespace quickdesk
