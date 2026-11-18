#include "languagemanage.h"

#ifdef Q_OS_WIN32
#include <Windows.h>
#endif

#include <QGuiApplication>
#include <QTranslator>

#include "core/localconfigcenter.h"

static QString kLanguagePath = ":/i18n/";

LanguageManage::LanguageManage(QObject* parent)
    : QObject(parent)
{
}

LanguageManage::~LanguageManage()
{
}

void LanguageManage::init()
{
    installLanguage();
}

QStringList LanguageManage::getSupportLanguages()
{
    QStringList languages;
    languages << "Auto" << "zh_CN" << "en_US" << "it_IT";
    return languages;
}

void LanguageManage::setCurrentLanguage(const QString& language)
{
    core::LocalConfigCenter::instance().setLanguage(language);
    installLanguage();
}

QString LanguageManage::getCurrentLanguage()
{
    return core::LocalConfigCenter::instance().language();
}

QString LanguageManage::getCurrentRealLanguage()
{
    QString language = getCurrentLanguage();
    if (language == "Auto") {
        QLocale::Language systemLanguage = getSystemLanguage();
        if (systemLanguage == QLocale::Chinese) {
            language = "zh_CN";
        } else if (systemLanguage == QLocale::Italian) {
            language = "it_IT";
        } else if (systemLanguage == QLocale::English) {
            language = "en_US";
        }
    }
    return language;
}

QString LanguageManage::getLanguageName(const QString& language)
{
    if (language == "zh_CN") {
        return u8"简体中文";
    }
    if (language == "en_US") {
        return "English";
    }
    if (language == "it_IT") {
        return "Italiano";
    }
    return tr("Auto");
}

void LanguageManage::installLanguage()
{
    auto configLanguage = core::LocalConfigCenter::instance().language();
    QLocale::Language language = getSystemLanguage();
    if (configLanguage == "zh_CN") {
        language = QLocale::Chinese;
    }
    if (configLanguage == "en_US") {
        language = QLocale::English;
    }
    if (configLanguage == "it_IT") {
        language = QLocale::Italian;
    }

    installTranslator(language);
}

QLocale::Language LanguageManage::getSystemLanguage()
{
    QLocale locale;
    QLocale::Language language = locale.language();

#ifdef Q_OS_WIN32
    LANGID languageId = ::GetUserDefaultUILanguage() & 0x03FF;
    if (LANG_CHINESE_SIMPLIFIED == languageId || LANG_CHINESE_TRADITIONAL == languageId) {
        language = QLocale::Chinese;
    }
#endif

    return language;
}

void LanguageManage::installTranslator(const QLocale::Language& language)
{
    static QTranslator zhCnTranslator;
    static QTranslator enUsTranslator;
    static QTranslator itItTranslator;
    static QTranslator* currentTranslator = nullptr;
    QTranslator* translator = nullptr;
    if (language == QLocale::Chinese) {
        if (zhCnTranslator.isEmpty()) {
            (void)zhCnTranslator.load(kLanguagePath + "zh_CN.qm");
        }
        translator = &zhCnTranslator;
    } else if (language == QLocale::Italian) {
        if (itItTranslator.isEmpty()) {
            (void)itItTranslator.load(kLanguagePath + "it_IT.qm");
        }
        translator = &itItTranslator;
    } else {
        if (enUsTranslator.isEmpty()) {
            (void)enUsTranslator.load(kLanguagePath + "en_US.qm");
        }
        translator = &enUsTranslator;
    }

    if (currentTranslator && currentTranslator != translator) {
        qApp->removeTranslator(currentTranslator);
    }
    qApp->installTranslator(translator);
    currentTranslator = translator;
}
