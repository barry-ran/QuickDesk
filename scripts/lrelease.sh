# https://doc.qt.io/qt-5/linguist-manager.html#lrelease
# lrelease -help
if [ -n "$1" ]; then
    export PATH="$1:$PATH"
fi
lrelease ./QuickDesk/res/i18n/en_US.ts ./QuickDesk/res/i18n/zh_CN.ts ./QuickDesk/res/i18n/it_IT.ts
