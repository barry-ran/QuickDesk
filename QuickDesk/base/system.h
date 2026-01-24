#pragma once

#include <QString>

namespace base {

class System {
public:
    // cpuid不是每个电脑都能获取到，同一批电脑重复概率大，英特尔明确不保证唯一
    static QString getCpuProcessorId();
    // bios serial不是每个电脑都能获取到，同一批电脑重复概率大
    static QString getBiosSerialNumber();
    // 硬盘id唯一，但是换硬盘比较常见
    static QString getDiskDriveSerialNumber();
    // 主板uuid，保证唯一，获取不到的几率较小 // https://blog.csdn.net/cunhan4654/article/details/108221980
    // 实测重复概率极大
    static QString getMotherboardUUID();
};

}
