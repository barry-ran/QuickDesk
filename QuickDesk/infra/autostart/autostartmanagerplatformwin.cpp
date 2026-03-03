#include "autostartmanagerplatform.h"

#include <Windows.h>
#include <ShlObj.h>
#include <objbase.h>
#include <shobjidl.h>
#include <propvarutil.h>
#include <propkey.h>

#include <QCoreApplication>
#include <QDir>
#include <QFile>
#include <QStandardPaths>

#pragma comment(lib, "ole32.lib")
#pragma comment(lib, "propsys.lib")

namespace infra {

namespace {

QString getStartupFolderPath()
{
    wchar_t path[MAX_PATH];
    if (SUCCEEDED(SHGetFolderPathW(nullptr, CSIDL_STARTUP, nullptr, 0, path))) {
        return QString::fromWCharArray(path);
    }
    return QString();
}

QString getStartupLinkPath()
{
    QString startupFolder = getStartupFolderPath();
    if (startupFolder.isEmpty()) {
        return QString();
    }
    QString appName = QCoreApplication::applicationName();
    return startupFolder + QDir::separator() + appName + ".lnk";
}

bool createShortcut(const QString& linkPath, const QString& targetPath,
                    const QString& args, const QString& workingDir)
{
    // Qt 主线程调用了 OleInitialize(nullptr) 等同于调用了 CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
    // qtbase\src\plugins\platforms\windows\qwindowscontext.cpp
    // 这里如果是在主线程执行理论上会返回 S_FALSE, 后续会执行对应的 CoUninitialize 逻辑 确保引用计数 在此方法中正确清零

    // msdn 推荐使用 CoInitializeEx 方法
    // 使用 COINIT_APARTMENTTHREADED (STA) 模式，适用于 IShellLink 等 Shell 组件
    HRESULT hrInit = CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
    // S_OK: 首次初始化成功（如非主线程）
    // S_FALSE: 已初始化（如主线程 Qt 已初始化），引用计数增加，仍需要 CoUninitialize
    // FAILED: 初始化失败，不需要 CoUninitialize
    bool needUninitialize = (hrInit == S_OK || hrInit == S_FALSE);

    IShellLinkW* psl = nullptr;
    HRESULT hr = CoCreateInstance(CLSID_ShellLink, nullptr, CLSCTX_INPROC_SERVER,
                                  IID_IShellLinkW, reinterpret_cast<LPVOID*>(&psl));
    if (FAILED(hr)) {
        if (needUninitialize) {
            CoUninitialize();
        }
        return false;
    }

    psl->SetPath(reinterpret_cast<LPCWSTR>(targetPath.utf16()));
    psl->SetArguments(reinterpret_cast<LPCWSTR>(args.utf16()));
    psl->SetWorkingDirectory(reinterpret_cast<LPCWSTR>(workingDir.utf16()));

    IPersistFile* ppf = nullptr;
    hr = psl->QueryInterface(IID_IPersistFile, reinterpret_cast<void**>(&ppf));
    if (FAILED(hr)) {
        psl->Release();
        if (needUninitialize) {
            CoUninitialize();
        }
        return false;
    }

    hr = ppf->Save(reinterpret_cast<LPCWSTR>(linkPath.utf16()), TRUE);
    ppf->Release();
    psl->Release();
    if (needUninitialize) {
        CoUninitialize();
    }

    return SUCCEEDED(hr);
}

} // namespace

bool AutoStartManagerPlatform::isAutoStartEnabled()
{
    QString linkPath = getStartupLinkPath();
    if (linkPath.isEmpty()) {
        return false;
    }
    return QFile::exists(linkPath);
}

bool AutoStartManagerPlatform::enableAutoStart(const QString& args)
{
    QString linkPath = getStartupLinkPath();
    if (linkPath.isEmpty()) {
        return false;
    }

    QString exePath = QCoreApplication::applicationFilePath();
    QString workingDir = QCoreApplication::applicationDirPath();

    return createShortcut(linkPath, exePath, args, workingDir);
}

bool AutoStartManagerPlatform::disableAutoStart()
{
    QString linkPath = getStartupLinkPath();
    if (linkPath.isEmpty()) {
        return false;
    }

    if (!QFile::exists(linkPath)) {
        return true;
    }

    return QFile::remove(linkPath);
}

}
