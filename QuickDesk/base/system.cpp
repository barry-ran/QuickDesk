#include "system.h"

#include <QProcess>

namespace base {

// https://blog.csdn.net/lxj362343/article/details/108619997
QString execWMIC(const QString& args)
{
    QProcess p;
    p.start("wmic", args.split(" "));
    p.waitForStarted();
    p.waitForFinished();
    QString result = QString::fromLocal8Bit(p.readAllStandardOutput());

    QStringList list = args.split(" ");
    result = result.remove(list.last(), Qt::CaseInsensitive);
    result = result.replace("\r", "");
    result = result.replace("\n", "");
    result = result.simplified();

    return result;
}

QString System::getCpuProcessorId()
{
    return execWMIC("cpu get processorid");
}

QString System::getBiosSerialNumber()
{
    return execWMIC("bios get serialnumber");
}

QString System::getDiskDriveSerialNumber()
{
    return execWMIC("diskdrive where index=0 get serialnumber");
}

QString System::getMotherboardUUID()
{
    // csproduct: Computer system product information from SMBIOS.
    return execWMIC("csproduct get uuid");
}

}
