#pragma once

#include <QObject>

class ConfigViewModel : public QObject {
    Q_OBJECT
    Q_PROPERTY(int darkTheme READ darkTheme WRITE setDarkTheme NOTIFY darkThemeChanged)
    Q_PROPERTY(QString language READ language WRITE setLanguage NOTIFY languageChanged)
    Q_PROPERTY(int passwordRefreshInterval READ passwordRefreshInterval WRITE setPasswordRefreshInterval NOTIFY passwordRefreshIntervalChanged)

public:
    ConfigViewModel(QObject* parent = nullptr);
    virtual ~ConfigViewModel();

    int darkTheme();
    void setDarkTheme(int value);
    
    QString language();
    void setLanguage(const QString& value);
    
    int passwordRefreshInterval();
    void setPasswordRefreshInterval(int value);

signals:
    void darkThemeChanged(int value);
    void languageChanged(const QString& value);
    void passwordRefreshIntervalChanged(int value);
};
