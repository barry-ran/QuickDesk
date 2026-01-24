#include "dbbase.h"

#include <QDir>
#include <QSqlDatabase>
#include <QSqlError>
#include <QSqlQuery>

#include "infra/log/log.h"

namespace infra {

bool DBBase::init(const QString& path, const QString& name)
{
    QDir dbDir(path);
    if (!dbDir.mkpath(path)) {
        return false;
    }

    QSqlDatabase db = QSqlDatabase::addDatabase("QSQLITE", name);
    db.setDatabaseName(path + QDir::separator() + name);
    if (!db.open()) {
        LOG_ERROR("{} open failed:{}", name.toStdString(), db.lastError().text().toStdString());
        return false;
    }

    return initTables();
}

}
