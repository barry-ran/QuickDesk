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
release_path="$script_path/../output/x64/$build_mode"
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
    rm -rf "$publish_path"
fi
echo "[*] creating publish dir: $publish_path"
mkdir -p "$publish_path"

echo "[*] copying QuickDesk.app..."
cp -R "$release_path/QuickDesk.app" "$publish_path/"

macos_dir="$publish_path/QuickDesk.app/Contents/MacOS"

echo "[*] copying host and client..."
if [ ! -d "$src_out_path" ]; then
    echo "[!] warning: src/out path does not exist: $src_out_path"
else
    if [ -d "$src_out_path/quickdesk_host.app" ]; then
        cp -R "$src_out_path/quickdesk_host.app" "$macos_dir/"
        echo "[*] copied quickdesk_host.app"
    else
        echo "[!] warning: quickdesk_host.app not found"
    fi

    if [ -f "$src_out_path/quickdesk_client" ]; then
        cp "$src_out_path/quickdesk_client" "$macos_dir/"
        echo "[*] copied quickdesk_client"
    else
        echo "[!] warning: quickdesk_client not found"
    fi
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

# Frameworks
rm -rf "$frameworks_dir/QtVirtualKeyboard.framework"
rm -rf "$frameworks_dir/QtSvg.framework"

echo "[*] cleaning unnecessary files..."
rm -rf "$publish_path/QuickDesk.app/Contents/MacOS/logs"
rm -rf "$publish_path/QuickDesk.app/Contents/MacOS/db"
rm -rf "$publish_path/QuickDesk.app/Contents/translations"

echo
echo
echo "---------------------------------------------------------------"
echo "[*] publish finished!"
echo "---------------------------------------------------------------"
echo "[*] publish dir: $publish_path"
echo

cd "$old_cd"
exit 0
