#pragma once

#include <QString>

namespace infra {

class DBBase {
public:
    virtual ~DBBase() = default;
    bool init(const QString& path, const QString& name);

protected:
    virtual bool initTables() = 0;
};

}
