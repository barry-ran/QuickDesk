#pragma once

#include <QString>

namespace base {

class Util {
public:
    static QString fileMd5(const QString& filePath);
};

}
