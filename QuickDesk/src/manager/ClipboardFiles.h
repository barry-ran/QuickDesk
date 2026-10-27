// Copyright 2026 QuickDesk Authors
#ifndef QUICKDESK_MANAGER_CLIPBOARDFILES_H
#define QUICKDESK_MANAGER_CLIPBOARDFILES_H

#include <QClipboard>
#include <QFileInfo>
#include <QGuiApplication>
#include <QMimeData>
#include <QUrl>

namespace quickdesk {

// Call only following an explicit user action for a confirmed download.
inline bool copyLocalFileToClipboard(const QString& path)
{
    const QFileInfo file(path);
    auto* clipboard = QGuiApplication::clipboard();
    if (path.isEmpty() || !file.exists() || !file.isFile() || !file.isReadable() || !clipboard)
        return false;
    auto* mime = new QMimeData;
    mime->setUrls({QUrl::fromLocalFile(file.canonicalFilePath())});
    clipboard->setMimeData(mime);
    return true;
}

} // namespace quickdesk
#endif
