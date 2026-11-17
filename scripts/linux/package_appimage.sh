#!/usr/bin/env bash
# Package the QuickDesk Linux client + remoting core into an AppImage.
# Run inside the qd-qt-builder image (Qt 6.8.3 at /opt/Qt), workspace at /ws.
#
#   CORE_DIR=/path/to/Linux-x64 bash scripts/linux/package_appimage.sh
#
# CORE_DIR must contain quickdesk_host, quickdesk_client and icudtl.dat
# (the artifacts produced by the remoting core build).
set -euo pipefail

WS="${WS:-/ws}"
QT_BIN="${QT_BIN:-$WS/output/x64/Release/QuickDesk}"
CORE_DIR="${CORE_DIR:?Set CORE_DIR to the Linux remoting core directory}"
QT_ROOT="${QT_ROOT:-/opt/Qt/6.8.3/gcc_64}"
OUT_DIR="${OUT_DIR:-$WS/output/appimage}"
CACHE="${CACHE:-$WS/scripts/linux/.cache}"
MCP_BIN="${MCP_BIN:-$WS/quickdesk-mcp/target/release/quickdesk-mcp}"
SKILL_HOST_DIR="${SKILL_HOST_DIR:-$WS/quickdesk-skill-host/target/release}"
RAPIDOCR_DIR="${RAPIDOCR_DIR:-$WS/QuickDesk/3rdparty/rapidocr/x64}"

APPDIR="$OUT_DIR/AppDir"
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/lib" "$APPDIR/usr/share/applications" \
         "$APPDIR/usr/share/icons/hicolor/256x256/apps" "$CACHE"

echo "[1/5] Assembling AppDir..."
cp -f "$QT_BIN" "$APPDIR/usr/bin/QuickDesk"
# Core binaries live alongside the Qt executable (ProcessManager publish layout).
for f in quickdesk_host quickdesk_client icudtl.dat; do
    if [ -f "$CORE_DIR/$f" ]; then
        cp -f "$CORE_DIR/$f" "$APPDIR/usr/bin/$f"
        chmod +x "$APPDIR/usr/bin/$f" 2>/dev/null || true
    else
        echo "WARNING: missing core file $CORE_DIR/$f"
    fi
done
# Any shared libraries the core ships (e.g. libquickdesk_core.so).
find "$CORE_DIR" -maxdepth 1 -name '*.so*' -exec cp -f {} "$APPDIR/usr/bin/" \; 2>/dev/null || true

# AI integration: MCP bridge (MainController looks for it next to the executable).
if [ -f "$MCP_BIN" ]; then
    cp -f "$MCP_BIN" "$APPDIR/usr/bin/quickdesk-mcp"
    chmod +x "$APPDIR/usr/bin/quickdesk-mcp"
else
    echo "WARNING: quickdesk-mcp not found at $MCP_BIN"
    echo "         build it with: cargo build --release --manifest-path quickdesk-mcp/Cargo.toml"
fi

# AI host agent: quickdesk-skill-host and its bundled skills
# (MainController looks for the binary and <appDir>/skills).
if [ -f "$SKILL_HOST_DIR/quickdesk-skill-host" ]; then
    cp -f "$SKILL_HOST_DIR/quickdesk-skill-host" "$APPDIR/usr/bin/"
    chmod +x "$APPDIR/usr/bin/quickdesk-skill-host"
    for skill in sys-info file-ops shell-runner; do
        if [ -f "$SKILL_HOST_DIR/$skill" ] && [ -f "$WS/quickdesk-skill-host/skills/$skill/SKILL.md" ]; then
            mkdir -p "$APPDIR/usr/bin/skills/$skill"
            cp -f "$SKILL_HOST_DIR/$skill" "$APPDIR/usr/bin/skills/$skill/$skill"
            cp -f "$WS/quickdesk-skill-host/skills/$skill/SKILL.md" "$APPDIR/usr/bin/skills/$skill/"
            chmod +x "$APPDIR/usr/bin/skills/$skill/$skill"
        else
            echo "WARNING: skill '$skill' not built in $SKILL_HOST_DIR"
        fi
    done
else
    echo "WARNING: quickdesk-skill-host not found at $SKILL_HOST_DIR"
    echo "         build it with: bash scripts/linux/build_skill_host.sh"
fi

# OCR: libRapidOcrOnnx.so links ONNX Runtime; models are loaded from
# <appDir>/models (OcrEngine::defaultModelDir).
if [ -f "$RAPIDOCR_DIR/lib/libRapidOcrOnnx.so" ]; then
    cp -f "$RAPIDOCR_DIR/lib/libRapidOcrOnnx.so" "$APPDIR/usr/lib/"
    # Only the SONAME is needed at runtime; the versioned file plus dev symlinks
    # would otherwise be bundled three times.
    cp -f "$RAPIDOCR_DIR/lib/libonnxruntime.so.1.20.1" "$APPDIR/usr/lib/libonnxruntime.so.1"
    mkdir -p "$APPDIR/usr/bin/models"
    cp -f "$RAPIDOCR_DIR"/models/* "$APPDIR/usr/bin/models/"
else
    echo "WARNING: Linux RapidOCR prebuilt not found at $RAPIDOCR_DIR"
    echo "         build it with: bash scripts/linux/build_rapidocr.sh"
fi

cp -f "$WS/scripts/linux/quickdesk.desktop" "$APPDIR/usr/share/applications/quickdesk.desktop"
cp -f "$WS/scripts/linux/quickdesk.desktop" "$APPDIR/quickdesk.desktop"
cp -f "$WS/QuickDesk/res/image/tray/logo.png" "$APPDIR/usr/share/icons/hicolor/256x256/apps/quickdesk.png"
cp -f "$WS/QuickDesk/res/image/tray/logo.png" "$APPDIR/quickdesk.png"

# MIT requires the license text to travel with every distributed copy, and the
# bundled third-party binaries need their notices too.
mkdir -p "$APPDIR/usr/share/doc/quickdesk"
cp -f "$WS/LICENSE" "$APPDIR/usr/share/doc/quickdesk/LICENSE"
cp -f "$WS/THIRD_PARTY_LICENSES" "$APPDIR/usr/share/doc/quickdesk/THIRD_PARTY_LICENSES" 2>/dev/null || true

echo "[2/5] Fetching linuxdeploy..."
LD="$CACHE/linuxdeploy-x86_64.AppImage"
LDQT="$CACHE/linuxdeploy-plugin-qt-x86_64.AppImage"
[ -f "$LD" ]   || curl -fsSL -o "$LD"   "https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage"
[ -f "$LDQT" ] || curl -fsSL -o "$LDQT" "https://github.com/linuxdeploy/linuxdeploy-plugin-qt/releases/download/continuous/linuxdeploy-plugin-qt-x86_64.AppImage"
chmod +x "$LD" "$LDQT"

# Extract instead of relying on FUSE (unavailable in most containers).
# Each AppImage extracts to ./squashfs-root, so use separate directories.
echo "[3/5] Extracting linuxdeploy..."
mkdir -p "$CACHE/ld" "$CACHE/ldqt" "$CACHE/bin"
( cd "$CACHE/ld"   && "$LD"   --appimage-extract >/dev/null )
( cd "$CACHE/ldqt" && "$LDQT" --appimage-extract >/dev/null )
LD_BIN="$CACHE/ld/squashfs-root/AppRun"
# linuxdeploy discovers plugins by name on PATH; wrap the extracted Qt plugin.
cat > "$CACHE/bin/linuxdeploy-plugin-qt" <<EOF
#!/usr/bin/env bash
exec "$CACHE/ldqt/squashfs-root/AppRun" "\$@"
EOF
chmod +x "$CACHE/bin/linuxdeploy-plugin-qt"
export PATH="$CACHE/bin:$CACHE/ld/squashfs-root/usr/bin:$PATH"
export QMAKE="$QT_ROOT/bin/qmake"
export QML_SOURCES_PATHS="$WS/QuickDesk/qml"
export LD_LIBRARY_PATH="$APPDIR/usr/lib:$QT_ROOT/lib:${LD_LIBRARY_PATH:-}"

# QuickDesk stores its local data in SQLite only. linuxdeploy-plugin-qt otherwise
# bundles every SQL driver plugin, and the unused MySQL/PostgreSQL/ODBC/Mimer
# drivers depend on client libraries (libmysqlclient, libpq, libodbc) that are not
# present, which aborts the deploy. Keep only the SQLite driver.
SQLDRIVERS="$QT_ROOT/plugins/sqldrivers"
if [ -d "$SQLDRIVERS" ]; then
    find "$SQLDRIVERS" -maxdepth 1 -name 'libqsql*.so' ! -name 'libqsqlite.so' -delete
fi

echo "[4/5] Running linuxdeploy (Qt plugin)..."
EXTRA_EXE=()
for e in quickdesk_host quickdesk_client; do
    [ -f "$APPDIR/usr/bin/$e" ] && EXTRA_EXE+=(--executable "$APPDIR/usr/bin/$e")
done
# NSS must come from the host: bundling only the ldd-visible subset
# (libnss3/libnssutil3) mixes versions with the dlopened softokn/freebl
# modules and crashes on NSS init.
NSS_EXCLUDES=()
for lib in libnss3.so libnssutil3.so libsmime3.so libssl3.so \
           libnspr4.so libplc4.so libplds4.so; do
    NSS_EXCLUDES+=(--exclude-library "$lib")
done

# linuxdeploy writes to the cwd; run from OUT_DIR so outputs land there.
# All paths below are absolute, so the directory change is safe.
# Deployment only (no --output plugin): the AppImage itself is produced by
# appimagetool directly below, because the bundled output plugin can block on
# appimagetool's appstream validation in containers without a D-Bus session.
( cd "$OUT_DIR" && "$LD_BIN" \
    --appdir "$APPDIR" \
    --executable "$APPDIR/usr/bin/QuickDesk" \
    "${EXTRA_EXE[@]}" \
    "${NSS_EXCLUDES[@]}" \
    --desktop-file "$APPDIR/usr/share/applications/quickdesk.desktop" \
    --icon-file "$APPDIR/usr/share/icons/hicolor/256x256/apps/quickdesk.png" \
    --plugin qt )

echo "[5/5] Building AppImage..."
# linuxdeploy symlinks AppRun to the Qt binary, so a KDE/GNOME Wayland session
# (WAYLAND_DISPLAY set) makes Qt request the wayland platform plugin, which is
# not bundled and aborts at startup ("Could not find the Qt platform plugin").
# Replace AppRun with a wrapper that falls back to xcb (Xwayland) unless the
# caller explicitly chose a platform.
rm -f "$APPDIR/AppRun"
cat > "$APPDIR/AppRun" <<'EOF'
#!/bin/bash
HERE="$(dirname "$(readlink -f "$0")")"
if [ -z "$QT_QPA_PLATFORM" ] || [ "$QT_QPA_PLATFORM" = "wayland" ]; then
    QT_QPA_PLATFORM=xcb
    export QT_QPA_PLATFORM
fi
# The remoting host captures the desktop through X11. A Wayland session marker
# passed down to the host/client makes Chromium take a Wayland capture path
# that yields no frames (video stays 0x0), so drop it whenever we run on X11.
if [ -n "$WAYLAND_DISPLAY" ]; then
    unset WAYLAND_DISPLAY XDG_SESSION_TYPE
fi
# Qt derives QStandardPaths::AppDataLocation from argv[0] at init; keep the
# AppImage path there (as the AppImage runtime does) so the data dir stays
# "QuickDesk-x86_64.AppImage" and settings carry over between launches.
exec -a "${APPIMAGE:-$HERE/usr/bin/QuickDesk}" "$HERE/usr/bin/QuickDesk" "$@"
EOF
chmod +x "$APPDIR/AppRun"
AI_PREFIX="$CACHE/ld/squashfs-root/plugins/linuxdeploy-plugin-appimage/appimagetool-prefix"
( cd "$OUT_DIR" && PATH="$AI_PREFIX/usr/bin:$PATH" ARCH=x86_64 \
    "$AI_PREFIX/usr/bin/appimagetool" --no-appstream "$APPDIR" )
# When run as root (needed to prune the root-owned Qt plugins), hand the outputs
# back to the owner of the mounted workspace so they are not root-owned.
WS_OWNER="$(stat -c '%u:%g' "$WS" 2>/dev/null || true)"
if [ -n "$WS_OWNER" ] && [ "$(id -u)" = "0" ]; then
    chown -R "$WS_OWNER" "$OUT_DIR" "$CACHE" 2>/dev/null || true
fi
ls -lh "$OUT_DIR"/*.AppImage 2>/dev/null || ls -lh ./*.AppImage 2>/dev/null || true