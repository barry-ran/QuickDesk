#include "util.h"

#include <QCryptographicHash>
#include <QFile>

namespace base {

QString Util::fileMd5(const QString& filePath)
{
    QFile file(filePath);
    qint64 fileSize = file.size();
    const qint64 bufferSize = 1024 * 10;

    QString md5;
    if (file.open(QIODevice::ReadOnly)) {
        char buffer[bufferSize];
        int bytesRead;
        int readSize = qMin(fileSize, bufferSize);

        QCryptographicHash hash(QCryptographicHash::Md5);
        while (readSize > 0 && (bytesRead = file.read(buffer, readSize)) > 0) {
            fileSize -= bytesRead;
            hash.addData(buffer, bytesRead);
            readSize = qMin(fileSize, bufferSize);
        }

        file.close();
        md5 = QString(hash.result().toHex());
    }

    return md5;
}

}
