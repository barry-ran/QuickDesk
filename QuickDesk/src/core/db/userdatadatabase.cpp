#include "userdatadatabase.h"

#include <QDataStream>
#include <QDateTime>
#include <QSqlDatabase>
#include <QSqlError>
#include <QSqlQuery>

#include "infra/env/applicationcontext.h"
#include "infra/log/log.h"

namespace core {

constexpr char kDeviceDb[] = "userdata.db";
constexpr char kIpRangeListTable[] = "ip_range_list";
constexpr char kRemoteDevicesTable[] = "remote_devices";

bool UserDataDataBase::init()
{
    return DBBase::init(infra::ApplicationContext::instance().dbPath(), kDeviceDb);
}

bool UserDataDataBase::initTables()
{
    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    // 注意：表名和字段名不能用占位符+bindValue的方式指定，sql为了防止sql注入设计的
    QString sql = QString(R"(
        CREATE TABLE IF NOT EXISTS %1 (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            start_ip TEXT NOT NULL,
            stop_ip TEXT NOT NULL,
            port TEXT NOT NULL,
            stop_port TEXT NOT NULL
            )
        )")
              .arg(kIpRangeListTable);
    if (!query.prepare(sql) || !query.exec()) {
        LOG_ERROR("[database] create table {} failed:{}", kIpRangeListTable, query.lastError().text().toStdString());
        return false;
    }

    // Create remote_devices table
    sql = QString(R"(
        CREATE TABLE IF NOT EXISTS %1 (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            device_id TEXT NOT NULL UNIQUE,
            device_name TEXT,
            access_password TEXT,
            is_favorite INTEGER DEFAULT 0,
            connection_count INTEGER DEFAULT 0,
            last_connected_time INTEGER,
            created_time INTEGER NOT NULL
            )
        )")
              .arg(kRemoteDevicesTable);
    if (!query.prepare(sql) || !query.exec()) {
        LOG_ERROR("[database] create table {} failed:{}", kRemoteDevicesTable, query.lastError().text().toStdString());
        return false;
    }

    // Create index on last_connected_time for better sorting performance
    sql = QString("CREATE INDEX IF NOT EXISTS idx_last_connected ON %1(last_connected_time DESC)").arg(kRemoteDevicesTable);
    if (!query.prepare(sql) || !query.exec()) {
        LOG_ERROR("[database] create index on {} failed:{}", kRemoteDevicesTable, query.lastError().text().toStdString());
        return false;
    }

    if (!upgradeTables()) {
        return false;
    }

    return true;
}

bool UserDataDataBase::upgradeTables()
{
    return true;
}

int UserDataDataBase::getLastId(const QString& tableName)
{
    QString sql = QString("SELECT seq FROM sqlite_sequence WHERE name='%1'").arg(tableName);
    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    query.prepare(sql);
    if (!query.exec()) {
        LOG_ERROR("[database] select table {} failed:{}", tableName.toStdString(), query.lastError().text().toStdString());
        return 0;
    }

    int lastId = 0;
    while (query.next()) {
        lastId = query.value(0).toInt();
        return lastId;
    }

    return lastId;
}

bool UserDataDataBase::addIpRange(const IpRange& ipRange, int& ipRangeId)
{
    QString sql = QString("INSERT INTO %1 (start_ip, stop_ip, port, stop_port) VALUES(:start_ip, :stop_ip, :port, :stop_port)").arg(kIpRangeListTable);
    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    query.prepare(sql);
    query.bindValue(":start_ip", ipRange.startIp);
    query.bindValue(":stop_ip", ipRange.stopIp);
    query.bindValue(":port", ipRange.startPort);
    query.bindValue(":stop_port", ipRange.stopPort);
    if (!query.exec()) {
        LOG_ERROR("[database] insert into table {} failed:{}", kIpRangeListTable, query.lastError().text().toStdString());
        return false;
    }
    query.finish();

    ipRangeId = getLastId(kIpRangeListTable);
    return true;
}

bool UserDataDataBase::removeIpRange(int ipRangeId)
{
    QString sql = QString("DELETE FROM %1 WHERE id=:id").arg(kIpRangeListTable);
    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    query.prepare(sql);
    query.bindValue(":id", ipRangeId);
    if (!query.exec()) {
        LOG_ERROR("[database] delete table {} failed:{}", kIpRangeListTable, query.lastError().text().toStdString());
        return false;
    }

    return true;
}

bool UserDataDataBase::allIpRange(QVector<IpRange>& ipRanges)
{
    QString sql = QString("SELECT id, start_ip, stop_ip, port, stop_port FROM %1").arg(kIpRangeListTable);
    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    query.prepare(sql);
    if (!query.exec()) {
        LOG_ERROR("[database] select table {} failed:{}", kIpRangeListTable, query.lastError().text().toStdString());
        return false;
    }

    while (query.next()) {
        int id = query.value(0).toInt();
        QString startIp = query.value(1).toString();
        QString stopIp = query.value(2).toString();
        QString startPort = query.value(3).toString();
        QString stopPort = query.value(4).toString();
        // 兼容旧版本
        if (stopPort.isEmpty()) {
            stopPort = startPort;
        }
        ipRanges.push_back({ id, startIp, stopIp, startPort, stopPort });
    }

    return true;
}

bool UserDataDataBase::addOrUpdateRemoteDevice(const RemoteDevice& device)
{
    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    
    // Check if device exists
    QString checkSql = QString("SELECT id FROM %1 WHERE device_id = :device_id").arg(kRemoteDevicesTable);
    query.prepare(checkSql);
    query.bindValue(":device_id", device.deviceId);
    
    if (!query.exec()) {
        LOG_ERROR("[database] check device existence failed: {}", query.lastError().text().toStdString());
        return false;
    }
    
    bool exists = query.next();
    query.finish();
    
    if (exists) {
        // Update existing device
        QString updateSql = QString(R"(
            UPDATE %1 SET 
                device_name = :device_name,
                access_password = :access_password,
                is_favorite = :is_favorite,
                connection_count = connection_count + 1,
                last_connected_time = :last_connected_time
            WHERE device_id = :device_id
        )").arg(kRemoteDevicesTable);
        
        query.prepare(updateSql);
        query.bindValue(":device_name", device.deviceName);
        query.bindValue(":access_password", device.accessPassword);
        query.bindValue(":is_favorite", device.isFavorite ? 1 : 0);
        query.bindValue(":last_connected_time", device.lastConnectedTime);
        query.bindValue(":device_id", device.deviceId);
        
        if (!query.exec()) {
            LOG_ERROR("[database] update device failed: {}", query.lastError().text().toStdString());
            return false;
        }
    } else {
        // Insert new device
        QString insertSql = QString(R"(
            INSERT INTO %1 (device_id, device_name, access_password, is_favorite, connection_count, last_connected_time, created_time)
            VALUES (:device_id, :device_name, :access_password, :is_favorite, 1, :last_connected_time, :created_time)
        )").arg(kRemoteDevicesTable);
        
        query.prepare(insertSql);
        query.bindValue(":device_id", device.deviceId);
        query.bindValue(":device_name", device.deviceName);
        query.bindValue(":access_password", device.accessPassword);
        query.bindValue(":is_favorite", device.isFavorite ? 1 : 0);
        query.bindValue(":last_connected_time", device.lastConnectedTime);
        query.bindValue(":created_time", device.createdTime);
        
        if (!query.exec()) {
            LOG_ERROR("[database] insert device failed: {}", query.lastError().text().toStdString());
            return false;
        }
    }
    
    return true;
}

bool UserDataDataBase::removeRemoteDevice(const QString& deviceId)
{
    QString sql = QString("DELETE FROM %1 WHERE device_id = :device_id").arg(kRemoteDevicesTable);
    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    query.prepare(sql);
    query.bindValue(":device_id", deviceId);
    
    if (!query.exec()) {
        LOG_ERROR("[database] delete device failed: {}", query.lastError().text().toStdString());
        return false;
    }
    
    return true;
}

bool UserDataDataBase::getRemoteDevice(const QString& deviceId, RemoteDevice& device)
{
    QString sql = QString(R"(
        SELECT id, device_id, device_name, access_password, is_favorite, connection_count, last_connected_time, created_time
        FROM %1 WHERE device_id = :device_id
    )").arg(kRemoteDevicesTable);
    
    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    query.prepare(sql);
    query.bindValue(":device_id", deviceId);
    
    if (!query.exec()) {
        LOG_ERROR("[database] query device failed: {}", query.lastError().text().toStdString());
        return false;
    }
    
    if (query.next()) {
        device.id = query.value(0).toInt();
        device.deviceId = query.value(1).toString();
        device.deviceName = query.value(2).toString();
        device.accessPassword = query.value(3).toString();
        device.isFavorite = query.value(4).toInt() == 1;
        device.connectionCount = query.value(5).toInt();
        device.lastConnectedTime = query.value(6).toLongLong();
        device.createdTime = query.value(7).toLongLong();
        return true;
    }
    
    return false;
}

bool UserDataDataBase::getAllRemoteDevices(QVector<RemoteDevice>& devices)
{
    // Order by: favorites first, then by last connected time
    QString sql = QString(R"(
        SELECT id, device_id, device_name, access_password, is_favorite, connection_count, last_connected_time, created_time
        FROM %1
        ORDER BY is_favorite DESC, last_connected_time DESC
    )").arg(kRemoteDevicesTable);
    
    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    query.prepare(sql);
    
    if (!query.exec()) {
        LOG_ERROR("[database] query all devices failed: {}", query.lastError().text().toStdString());
        return false;
    }
    
    while (query.next()) {
        RemoteDevice device;
        device.id = query.value(0).toInt();
        device.deviceId = query.value(1).toString();
        device.deviceName = query.value(2).toString();
        device.accessPassword = query.value(3).toString();
        device.isFavorite = query.value(4).toInt() == 1;
        device.connectionCount = query.value(5).toInt();
        device.lastConnectedTime = query.value(6).toLongLong();
        device.createdTime = query.value(7).toLongLong();
        devices.push_back(device);
    }
    
    return true;
}

bool UserDataDataBase::updateDeviceLastConnected(const QString& deviceId)
{
    QString sql = QString(R"(
        UPDATE %1 SET 
            connection_count = connection_count + 1,
            last_connected_time = :last_connected_time
        WHERE device_id = :device_id
    )").arg(kRemoteDevicesTable);
    
    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    query.prepare(sql);
    query.bindValue(":last_connected_time", QDateTime::currentMSecsSinceEpoch());
    query.bindValue(":device_id", deviceId);
    
    if (!query.exec()) {
        LOG_ERROR("[database] update device last connected failed: {}", query.lastError().text().toStdString());
        return false;
    }
    
    return true;
}

bool UserDataDataBase::updateDevicePassword(const QString& deviceId, const QString& encryptedPassword)
{
    QString sql = QString(R"(
        UPDATE %1 SET access_password = :access_password
        WHERE device_id = :device_id
    )").arg(kRemoteDevicesTable);

    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    query.prepare(sql);
    query.bindValue(":access_password", encryptedPassword);
    query.bindValue(":device_id", deviceId);

    if (!query.exec()) {
        LOG_ERROR("[database] update device password failed: {}", query.lastError().text().toStdString());
        return false;
    }

    return query.numRowsAffected() > 0;
}

bool UserDataDataBase::cleanOldDevices(int maxCount)
{
    // Keep only the most recent maxCount devices (excluding favorites)
    QString sql = QString(R"(
        DELETE FROM %1
        WHERE is_favorite = 0
        AND id NOT IN (
            SELECT id FROM %1
            WHERE is_favorite = 0
            ORDER BY last_connected_time DESC
            LIMIT :max_count
        )
    )").arg(kRemoteDevicesTable);
    
    QSqlQuery query(QSqlDatabase::database(kDeviceDb));
    query.prepare(sql);
    query.bindValue(":max_count", maxCount);
    
    if (!query.exec()) {
        LOG_ERROR("[database] clean old devices failed: {}", query.lastError().text().toStdString());
        return false;
    }
    
    int deletedCount = query.numRowsAffected();
    if (deletedCount > 0) {
        LOG_INFO("[database] cleaned {} old devices", deletedCount);
    }
    
    return true;
}
} // namespace core
