#!/bin/bash

echo
echo
echo "---------------------------------------------------------------"
echo "check ENV"
echo "---------------------------------------------------------------"

if [ -z "$ENV_QT_PATH" ]; then
    ENV_QT_PATH="/Users/kun.ran/Qt/6.8.6"
fi
echo "ENV_QT_PATH: $ENV_QT_PATH"

{
    cd "$(dirname "$0")"
    script_path=$(pwd)
    cd - > /dev/null
} &> /dev/null

old_cd=$(pwd)
cd "$(dirname "$0")"

build_mode=Release
errno=1

echo
echo
echo "---------------------------------------------------------------"
echo "parse arguments"
echo "---------------------------------------------------------------"

while [ $# -gt 0 ]; do
    case "$(echo "$1" | tr '[:upper:]' '[:lower:]')" in
        debug)   build_mode=Debug ;;
        release) build_mode=Release ;;
    esac
    shift
done

echo "[*] arch: arm64"
echo "[*] build mode: $build_mode"
echo

qt_mac_path="$ENV_QT_PATH/macos"
publish_path="$script_path/../publish/$build_mode"
release_path="$script_path/../output/arm64/$build_mode"
src_out_path="$script_path/../../src/out/$build_mode"

echo "[*] Qt macOS path: $qt_mac_path"
echo "[*] publish path: $publish_path"
echo "[*] output path: $release_path"
echo "[*] src/out path: $src_out_path"
echo

export PATH="$qt_mac_path/bin:$PATH"

echo
echo
echo "---------------------------------------------------------------"
echo "begin publish"
echo "---------------------------------------------------------------"

if [ ! -d "$release_path" ]; then
    echo "[!] error: output path does not exist: $release_path"
    echo "[!] please run build_qd_mac.sh $build_mode first"
    cd "$old_cd"
    exit 1
fi

if [ -d "$publish_path" ]; then
    echo "[*] cleaning old publish dir..."
    xattr -rc "$publish_path" 2>/dev/null
    rm -rf "$publish_path"
fi
echo "[*] creating publish dir: $publish_path"
mkdir -p "$publish_path"

echo "[*] copying QuickDesk.app..."
cp -R "$release_path/QuickDesk.app" "$publish_path/"

macos_dir="$publish_path/QuickDesk.app/Contents/MacOS"
frameworks_dir="$publish_path/QuickDesk.app/Contents/Frameworks"

echo "[*] copying host and client..."
thirdparty_path="$script_path/../QuickDesk/3rdparty/quickdesk-remoting/arm64"
echo "[*] 3rdparty path: $thirdparty_path"
mkdir -p "$frameworks_dir"

if [ -d "$src_out_path/quickdesk_host.app" ]; then
    cp -R "$src_out_path/quickdesk_host.app" "$frameworks_dir/"
    echo "[*] copied quickdesk_host.app from src/out"
elif [ -d "$thirdparty_path/quickdesk_host.app" ]; then
    cp -R "$thirdparty_path/quickdesk_host.app" "$frameworks_dir/"
    echo "[*] copied quickdesk_host.app from 3rdparty"
else
    echo "[!] warning: quickdesk_host.app not found"
fi

if [ -f "$src_out_path/quickdesk_client" ]; then
    cp "$src_out_path/quickdesk_client" "$frameworks_dir/"
    echo "[*] copied quickdesk_client from src/out"
elif [ -f "$thirdparty_path/quickdesk_client" ]; then
    cp "$thirdparty_path/quickdesk_client" "$frameworks_dir/"
    echo "[*] copied quickdesk_client from 3rdparty"
else
    echo "[!] warning: quickdesk_client not found"
fi
echo

echo "[*] running macdeployqt..."
macdeployqt "$publish_path/QuickDesk.app" -qmldir="$script_path/../QuickDesk/qml"
if [ $? -ne 0 ]; then
    echo "[!] macdeployqt failed"
    cd "$old_cd"
    exit 1
fi

echo "[*] cleaning unnecessary Qt dependencies..."

plugins_dir="$publish_path/QuickDesk.app/Contents/PlugIns"
frameworks_dir="$publish_path/QuickDesk.app/Contents/Frameworks"

# PlugIns
rm -rf "$plugins_dir/iconengines"
rm -rf "$plugins_dir/virtualkeyboard"
rm -rf "$plugins_dir/printsupport"
rm -rf "$plugins_dir/platforminputcontexts"
rm -rf "$plugins_dir/bearer"
rm -rf "$plugins_dir/qmltooling"
rm -rf "$plugins_dir/generic"

# imageformats - keep only jpeg
if [ -d "$plugins_dir/imageformats" ]; then
    echo "[*] cleaning imageformats..."
    rm -f "$plugins_dir/imageformats/libqgif.dylib"
    rm -f "$plugins_dir/imageformats/libqicns.dylib"
    rm -f "$plugins_dir/imageformats/libqico.dylib"
    rm -f "$plugins_dir/imageformats/libqmacheif.dylib"
    rm -f "$plugins_dir/imageformats/libqmacjp2.dylib"
    rm -f "$plugins_dir/imageformats/libqsvg.dylib"
    rm -f "$plugins_dir/imageformats/libqtga.dylib"
    rm -f "$plugins_dir/imageformats/libqtiff.dylib"
    rm -f "$plugins_dir/imageformats/libqwbmp.dylib"
    rm -f "$plugins_dir/imageformats/libqwebp.dylib"
fi

# sqldrivers - keep only sqlite
if [ -d "$plugins_dir/sqldrivers" ]; then
    echo "[*] cleaning sqldrivers (keep sqlite)..."
    for f in "$plugins_dir/sqldrivers/"*.dylib; do
        if [[ "$(basename "$f")" != *sqlite* ]]; then
            rm -f "$f"
        fi
    done
fi

# Frameworks - remove unnecessary styles and components
rm -rf "$frameworks_dir/QtVirtualKeyboard.framework"
rm -rf "$frameworks_dir/QtVirtualKeyboardSettings.framework"
rm -rf "$frameworks_dir/QtSvg.framework"
rm -rf "$frameworks_dir/QtQuickControls2FluentWinUI3StyleImpl.framework"
rm -rf "$frameworks_dir/QtQuickControls2Fusion.framework"
rm -rf "$frameworks_dir/QtQuickControls2FusionStyleImpl.framework"
rm -rf "$frameworks_dir/QtQuickControls2IOSStyleImpl.framework"
rm -rf "$frameworks_dir/QtQuickControls2Imagine.framework"
rm -rf "$frameworks_dir/QtQuickControls2ImagineStyleImpl.framework"
rm -rf "$frameworks_dir/QtQuickControls2Material.framework"
rm -rf "$frameworks_dir/QtQuickControls2MaterialStyleImpl.framework"
rm -rf "$frameworks_dir/QtQuickControls2Universal.framework"
rm -rf "$frameworks_dir/QtQuickControls2UniversalStyleImpl.framework"

# PlugIns/quick - remove unnecessary style and virtual keyboard plugins
echo "[*] cleaning unnecessary quick plugins..."
rm -f "$plugins_dir/quick/libqtquickcontrols2fluentwinui3styleimplplugin.dylib"
rm -f "$plugins_dir/quick/libqtquickcontrols2fluentwinui3styleplugin.dylib"
rm -f "$plugins_dir/quick/libqtquickcontrols2fusionstyleimplplugin.dylib"
rm -f "$plugins_dir/quick/libqtquickcontrols2fusionstyleplugin.dylib"
rm -f "$plugins_dir/quick/libqtquickcontrols2imaginestyleimplplugin.dylib"
rm -f "$plugins_dir/quick/libqtquickcontrols2imaginestyleplugin.dylib"
rm -f "$plugins_dir/quick/libqtquickcontrols2iosstyleimplplugin.dylib"
rm -f "$plugins_dir/quick/libqtquickcontrols2iosstyleplugin.dylib"
rm -f "$plugins_dir/quick/libqtquickcontrols2materialstyleimplplugin.dylib"
rm -f "$plugins_dir/quick/libqtquickcontrols2materialstyleplugin.dylib"
rm -f "$plugins_dir/quick/libqtquickcontrols2universalstyleimplplugin.dylib"
rm -f "$plugins_dir/quick/libqtquickcontrols2universalstyleplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkbbuiltinstylesplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkbcomponentsplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkbhangulplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkblayoutsplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkbopenwnnplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkbpinyinplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkbplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkbpluginsplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkbsettingsplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkbstylesplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkbtcimeplugin.dylib"
rm -f "$plugins_dir/quick/libqtvkbthaiplugin.dylib"

echo "[*] cleaning unnecessary files..."
rm -rf "$publish_path/QuickDesk.app/Contents/MacOS/logs"
rm -rf "$publish_path/QuickDesk.app/Contents/MacOS/db"
rm -rf "$publish_path/QuickDesk.app/Contents/translations"

# Clean stale files that would break code signing
find "$frameworks_dir/quickdesk_host.app" -name "*.log" -delete 2>/dev/null

echo "[*] ad-hoc code signing (inside-out)..."
# Sign nested components first, then the outer bundle.
# Ad-hoc signing gives each binary a stable code identity (CDHash)
# so that macOS TCC can recognize the app in permission lists.
if [ -d "$frameworks_dir/quickdesk_host.app" ]; then
    codesign --force --sign - "$frameworks_dir/quickdesk_host.app"
fi
if [ -f "$frameworks_dir/quickdesk_client" ]; then
    codesign --force --sign - "$frameworks_dir/quickdesk_client"
fi
find "$frameworks_dir" -name "*.framework" -maxdepth 1 -exec codesign --force --sign - {} \;
find "$frameworks_dir" -name "*.dylib" -maxdepth 1 -exec codesign --force --sign - {} \;
find "$plugins_dir" -name "*.dylib" -exec codesign --force --sign - {} \;
codesign --force --sign - "$publish_path/QuickDesk.app"
echo "[*] code signing done"

echo
echo
echo "---------------------------------------------------------------"
echo "[*] publish finished!"
echo "---------------------------------------------------------------"
echo "[*] publish dir: $publish_path"
echo

cd "$old_cd"
exit 0
