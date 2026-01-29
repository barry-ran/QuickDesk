// Copyright 2026 QuickDesk Authors
// Process and Server status enums

#ifndef QUICKDESK_COMMON_PROCESSSTATUS_H
#define QUICKDESK_COMMON_PROCESSSTATUS_H

#include <QObject>

namespace quickdesk {

/**
 * @brief Process status enum
 */
class ProcessStatus : public QObject {
    Q_OBJECT
    
public:
    enum Status {
        NotStarted,     // 未启动
        Starting,       // 启动中
        Running,        // 已启动
        Failed,         // 启动失败
        Restarting      // 重启中
    };
    Q_ENUM(Status)
    
private:
    ProcessStatus() = delete;
};

/**
 * @brief Server connection status enum
 */
class ServerStatus : public QObject {
    Q_OBJECT
    
public:
    enum Status {
        Disconnected,   // 未连接
        Connecting,     // 连接中
        Connected,      // 已连接
        Failed,         // 连接失败
        Reconnecting    // 重连中
    };
    Q_ENUM(Status)
    
private:
    ServerStatus() = delete;
};

} // namespace quickdesk

#endif // QUICKDESK_COMMON_PROCESSSTATUS_H
