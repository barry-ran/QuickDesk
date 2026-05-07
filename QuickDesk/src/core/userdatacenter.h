#pragma once

#include <QObject>
#include <QString>

#include "base/singleton.h"
#include "core/common/commontype.h"

namespace core {

class UserDataDataBase;
class UserDataCenter : public QObject, public base::Singleton<UserDataCenter> {
    Q_OBJECT
public:
    UserDataCenter(QObject* parent = nullptr);
    ~UserDataCenter();

    bool init();

    // IpRange operations
    bool addIpRange(const IpRange& ipRange, int& ipRangeId);
    bool removeIpRange(int ipRangeId);
    bool allIpRange(QVector<IpRange>& ipRanges);

    // RemoteDevice operations
    bool addOrUpdateRemoteDevice(const RemoteDevice& device);
    bool removeRemoteDevice(const QString& deviceId);
    bool getRemoteDevice(const QString& deviceId, RemoteDevice& device);
    bool getAllRemoteDevices(QVector<RemoteDevice>& devices);
    bool updateDeviceLastConnected(const QString& deviceId);
    bool updateDevicePassword(const QString& deviceId, const QString& encryptedPassword);
    bool cleanOldDevices(int maxCount);

private:
    UserDataDataBase* m_userDataDB = nullptr;
};

}
