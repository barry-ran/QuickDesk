#pragma once

#include <QByteArray>
#include <QPair>
#include <QString>
#include <QVariant>

#include "core/common/commontype.h"
#include "infra/db/dbbase.h"

namespace core {

class UserDataDataBase : public infra::DBBase {
public:
    virtual ~UserDataDataBase() = default;

    bool init();

    bool addIpRange(const IpRange& ipRange, int& ipRangeId);
    bool removeIpRange(int ipRangeId);
    bool allIpRange(QVector<IpRange>& ipRanges);

    // Remote devices operations
    bool addOrUpdateRemoteDevice(const RemoteDevice& device);
    bool removeRemoteDevice(const QString& deviceId);
    bool getRemoteDevice(const QString& deviceId, RemoteDevice& device);
    bool getAllRemoteDevices(QVector<RemoteDevice>& devices);
    bool updateDeviceLastConnected(const QString& deviceId);
    bool updateDevicePassword(const QString& deviceId, const QString& encryptedPassword);
    bool cleanOldDevices(int maxCount);

private:
    bool initTables() override;
    bool upgradeTables();
    int getLastId(const QString& tableName);
};

}
