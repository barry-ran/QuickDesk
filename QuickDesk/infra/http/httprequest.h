#pragma once

#include <QMap>
#include <QObject>

#include "base/singleton.h"

class QNetworkAccessManager;
class QNetworkRequest;
class QNetworkReply;
namespace infra {

typedef std::function<void(int, const std::string&, const std::string&)> HttpRequestCallback;

class HttpRequest : public QObject, public base::Singleton<HttpRequest> {
    Q_OBJECT
public:
    HttpRequest(QObject* parent = nullptr);

    void sendGetRequest(const QUrl& url, const QList<QPair<QString, QString>>& headers, int timeout, HttpRequestCallback callback);
    void sendPostRequest(const QUrl& url, const QList<QPair<QString, QString>>& headers, const QString& data, int timeout, HttpRequestCallback callback);

private slots:
    void slotHttpFinished(QNetworkReply* reply);

private:
    void configRequest(QNetworkRequest& request);

private:
    QNetworkAccessManager* m_networkAccessManager = nullptr;
    QMap<int, HttpRequestCallback> m_tasks;
};

}
