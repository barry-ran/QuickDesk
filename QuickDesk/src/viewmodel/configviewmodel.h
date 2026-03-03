#pragma once

#include <QObject>

class ConfigViewModel : public QObject {
    Q_OBJECT
    Q_PROPERTY(int darkTheme READ darkTheme WRITE setDarkTheme NOTIFY darkThemeChanged)
    Q_PROPERTY(QString language READ language WRITE setLanguage NOTIFY languageChanged)
    Q_PROPERTY(int accessCodeRefreshInterval READ accessCodeRefreshInterval WRITE setAccessCodeRefreshInterval NOTIFY accessCodeRefreshIntervalChanged)
    Q_PROPERTY(QString preferredVideoCodec READ preferredVideoCodec WRITE setPreferredVideoCodec NOTIFY preferredVideoCodecChanged)
    Q_PROPERTY(bool autoStart READ autoStart WRITE setAutoStart NOTIFY autoStartChanged)

public:
    ConfigViewModel(QObject* parent = nullptr);
    virtual ~ConfigViewModel();

    int darkTheme();
    void setDarkTheme(int value);
    
    QString language();
    void setLanguage(const QString& value);
    
    int accessCodeRefreshInterval();
    void setAccessCodeRefreshInterval(int value);
    
    QString preferredVideoCodec();
    void setPreferredVideoCodec(const QString& value);

    bool autoStart();
    void setAutoStart(bool value);

signals:
    void darkThemeChanged(int value);
    void languageChanged(const QString& value);
    void accessCodeRefreshIntervalChanged(int value);
    void preferredVideoCodecChanged(const QString& value);
    void autoStartChanged(bool value);
};
