#include "userdatacenter.h"

#include "core/db/userdatadatabase.h"

namespace core {

UserDataCenter::UserDataCenter(QObject* parent)
    : QObject(parent)
{
    m_userDataDB = new UserDataDataBase();
}

UserDataCenter::~UserDataCenter()
{
    delete m_userDataDB;
    m_userDataDB = nullptr;
}

bool UserDataCenter::init()
{
    return m_userDataDB->init();
}

bool UserDataCenter::addIpRange(const IpRange& ipRange, int& ipRangeId)
{
    return m_userDataDB->addIpRange(ipRange, ipRangeId);
}

bool UserDataCenter::removeIpRange(int ipRangeId)
{
    return m_userDataDB->removeIpRange(ipRangeId);
}

bool UserDataCenter::allIpRange(QVector<IpRange>& ipRanges)
{
    return m_userDataDB->allIpRange(ipRanges);
}

bool UserDataCenter::addOrUpdateRemoteDevice(const RemoteDevice& device)
{
    return m_userDataDB->addOrUpdateRemoteDevice(device);
}

bool UserDataCenter::removeRemoteDevice(const QString& deviceId)
{
    return m_userDataDB->removeRemoteDevice(deviceId);
}

bool UserDataCenter::getRemoteDevice(const QString& deviceId, RemoteDevice& device)
{
    return m_userDataDB->getRemoteDevice(deviceId, device);
}

bool UserDataCenter::getAllRemoteDevices(QVector<RemoteDevice>& devices)
{
    return m_userDataDB->getAllRemoteDevices(devices);
}

bool UserDataCenter::updateDeviceLastConnected(const QString& deviceId)
{
    return m_userDataDB->updateDeviceLastConnected(deviceId);
}

bool UserDataCenter::updateDevicePassword(const QString& deviceId, const QString& encryptedPassword)
{
    return m_userDataDB->updateDevicePassword(deviceId, encryptedPassword);
}

bool UserDataCenter::cleanOldDevices(int maxCount)
{
    return m_userDataDB->cleanOldDevices(maxCount);
}

}
