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
resources_dir="$publish_path/QuickDesk.app/Contents/Resources"
mkdir -p "$resources_dir"

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

# Copy MCP bridge
echo "[*] copying quickdesk-mcp..."
mcp_output="$script_path/../output/arm64/$build_mode/quickdesk-mcp"
if [ -f "$mcp_output" ]; then
    cp "$mcp_output" "$frameworks_dir/"
    echo "[*] copied quickdesk-mcp from output"
else
    echo "[!] warning: quickdesk-mcp not found (run build_mcp_mac.sh first)"
fi
echo

# Copy skill-host and built-in skills
echo "[*] copying quickdesk-skill-host..."
skill_host_output="$script_path/../output/arm64/$build_mode/quickdesk-skill-host"
if [ -f "$skill_host_output" ]; then
    cp "$skill_host_output" "$frameworks_dir/"
    echo "[*] copied quickdesk-skill-host from output"
else
    echo "[!] warning: quickdesk-skill-host not found (run build_skill_host_mac.sh first)"
fi

echo "[*] copying built-in skills..."
# Apple's bundle convention requires Contents/Frameworks/ to contain only
# frameworks / dylibs / executables. The skills directory holds mixed
# content (Mach-O binaries + SKILL.md), which prevents codesign from
# sealing the bundle ("code has no resources but signature indicates they
# must be present") and triggers the "app is damaged" Gatekeeper error.
# Place it under Contents/Resources/ instead, which is the documented
# location for arbitrary application resources.
skills_output="$script_path/../output/arm64/$build_mode/skills"
if [ -d "$skills_output" ]; then
    mkdir -p "$resources_dir/skills"
    cp -R "$skills_output/"* "$resources_dir/skills/"
    echo "[*] copied skills directory to Contents/Resources/skills"
else
    echo "[!] warning: skills directory not found (run build_skill_host_mac.sh first)"
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
resources_dir="$publish_path/QuickDesk.app/Contents/Resources"

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

# macdeployqt creates symlinks under Contents/Resources/qml/QtQuick/Controls/<Style>/
# that point at Contents/PlugIns/quick/libqtquickcontrols2<style>*.dylib so the QML
# engine can pick up styles by name. We strip those style plugins above to slim the
# bundle, which leaves the symlinks dangling. `codesign --deep --strict` then trips
# over them with "No such file or directory" and the whole bundle is rejected as
# damaged. Remove every broken symlink before signing.
echo "[*] removing dangling symlinks left by Qt cleanup..."
removed_links=0
while IFS= read -r -d '' link; do
    if [ ! -e "$link" ]; then
        rm -f "$link"
        removed_links=$((removed_links + 1))
    fi
done < <(find "$publish_path/QuickDesk.app" -type l -print0 2>/dev/null)
echo "[*] removed $removed_links dangling symlink(s)"

echo "[*] ad-hoc code signing (inside-out)..."
# Strict inside-out order is required so that codesign can seal the outer
# bundle. Any unsigned Mach-O found while sealing the parent will produce
# "code object is not signed at all" and the bundle ends up with
# "Sealed Resources=none", which Gatekeeper reports as "app is damaged".
#
# Order:
#   1. Mach-O resources under Contents/Resources/ (skills/*)
#   2. Plugin dylibs under Contents/PlugIns/
#   3. Loose dylibs and frameworks under Contents/Frameworks/
#   4. Helper executables and nested .app bundles under Contents/Frameworks/
#   5. The outer .app bundle (with --deep --strict self-verification)

sign_mach_o_tree() {
    local root="$1"
    [ -d "$root" ] || return 0
    while IFS= read -r -d '' bin; do
        case "$(file -b "$bin" 2>/dev/null)" in
            Mach-O*)
                codesign --force --sign - --timestamp=none "$bin" || return 1
                ;;
        esac
    done < <(find "$root" -type f -print0)
}

# 1) Mach-O binaries shipped as resources (built-in skills).
#    New crates under Contents/Resources/skills/ are picked up automatically.
sign_mach_o_tree "$resources_dir/skills" || { echo "[!] sign skills failed"; cd "$old_cd"; exit 1; }

# 2) Qt plugin dylibs.
find "$plugins_dir" -name "*.dylib" -exec codesign --force --sign - --timestamp=none {} \;

# 3) Frameworks (.framework) and loose dylibs at the top level of Frameworks/.
find "$frameworks_dir" -maxdepth 1 -name "*.dylib" -exec codesign --force --sign - --timestamp=none {} \;
find "$frameworks_dir" -maxdepth 1 -name "*.framework" -exec codesign --force --sign - --timestamp=none {} \;

# 4) Helper executables and nested app bundles in Frameworks/.
if [ -d "$frameworks_dir/quickdesk_host.app" ]; then
    codesign --force --sign - --timestamp=none --deep "$frameworks_dir/quickdesk_host.app"
fi
for helper in quickdesk_client quickdesk-mcp quickdesk-skill-host; do
    if [ -f "$frameworks_dir/$helper" ]; then
        codesign --force --sign - --timestamp=none "$frameworks_dir/$helper"
    fi
done

# 5) Outer bundle. --deep here is a safety net only — every nested binary
#    has already been signed individually above so codesign just seals
#    Contents/_CodeSignature/CodeResources.
codesign --force --sign - --timestamp=none --deep "$publish_path/QuickDesk.app"

echo "[*] verifying signature (this MUST succeed or the dmg will be 'damaged')..."
if ! codesign --verify --deep --strict --verbose=2 "$publish_path/QuickDesk.app"; then
    echo "[!] code signature verification failed"
    cd "$old_cd"
    exit 1
fi
codesign -dv --verbose=2 "$publish_path/QuickDesk.app" 2>&1 | grep -E 'Sealed Resources|Identifier|TeamIdentifier|Signature' || true
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
