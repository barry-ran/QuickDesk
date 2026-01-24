#include "httprequest.h"

#include <functional>
#include <string>

#include <QNetworkAccessManager>
#include <QNetworkReply>

#include "infra/log/log.h"

namespace infra {

HttpRequest::HttpRequest(QObject* parent)
    : QObject(parent)
    , m_networkAccessManager(new QNetworkAccessManager(this))
{
    m_networkAccessManager->setRedirectPolicy(QNetworkRequest::NoLessSafeRedirectPolicy);
    connect(m_networkAccessManager, &QNetworkAccessManager::finished, this, &HttpRequest::slotHttpFinished);
}

void HttpRequest::sendGetRequest(const QUrl& url, const QList<QPair<QString, QString>>& headers, int timeout, HttpRequestCallback callback)
{
    QNetworkRequest request;
    request.setUrl(url);
    request.setTransferTimeout(timeout);    
    for (auto it = headers.constBegin(); it != headers.constEnd(); ++it) {
        request.setRawHeader(it->first.toUtf8(), it->second.toUtf8());
    }

    configRequest(request);

    QNetworkReply* reply = m_networkAccessManager->get(request);
    auto taskKey = reinterpret_cast<quintptr>(reply);
    LOG_DEBUG("[http] start get:{}", taskKey);
    m_tasks[taskKey] = callback;
}

void HttpRequest::sendPostRequest(const QUrl& url, const QList<QPair<QString, QString>>& headers, const QString& data, int timeout, HttpRequestCallback callback)
{
    QNetworkRequest request;
    request.setUrl(url);
    request.setTransferTimeout(timeout);
    for (auto it = headers.constBegin(); it != headers.constEnd(); ++it) {
        request.setRawHeader(it->first.toUtf8(), it->second.toUtf8());
    }

    configRequest(request);

    QByteArray byte = data.toUtf8();
    request.setHeader(QNetworkRequest::ContentLengthHeader, byte.size());
    QNetworkReply* reply = m_networkAccessManager->post(request, byte);
    auto taskKey = reinterpret_cast<quintptr>(reply);
    LOG_DEBUG("[http] start get:{}", taskKey);
    m_tasks[taskKey] = callback;
}

void HttpRequest::slotHttpFinished(QNetworkReply* reply)
{
    reply->deleteLater();

    auto taskKey = reinterpret_cast<quintptr>(reply);
    if (!m_tasks.contains(taskKey)) {
        LOG_ERROR("[http] no find task key:", taskKey);
        return;
    }

    LOG_DEBUG("[http] end get:{}", taskKey);
    auto callback = m_tasks.take(taskKey);

    if (QNetworkReply::NoError != reply->error()) {
        LOG_ERROR("[http] request failed:{}", static_cast<int>(reply->error()));
    }

    QVariant code = reply->attribute(QNetworkRequest::HttpStatusCodeAttribute);
    QByteArray data = reply->readAll();
    std::string msg = QNetworkReply::NoError == reply->error() ? "" : reply->errorString().toUtf8().toStdString();
    callback(code.toInt(), msg, data.toStdString());
}

void HttpRequest::configRequest(QNetworkRequest &request)
{
    // default ssl config
    QSslConfiguration sslConfig;
    sslConfig.setPeerVerifyMode(QSslSocket::VerifyNone);
    request.setSslConfiguration(sslConfig);

    // 这里不要设置，qt会自动设置"gzip, deflate"并后面自动解压，如果这里手动设置了"gzip, deflate"后面反而需要自己手动解压
    //https://stackoverflow.com/questions/2340548/does-qnetworkmanager-get-accept-compressed-replies-by-default
    //request.setRawHeader(QByteArray("Accept-Encoding"), QByteArray("gzip, deflate"));
}

}
