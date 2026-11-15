#!/usr/bin/env bash
# Build the Linux RapidOCR prebuilt tree consumed by cmake/rapidocr.cmake.
#
#   bash scripts/linux/build_rapidocr.sh
#
# Output: QuickDesk/3rdparty/rapidocr/<arch>/{include,lib,models}
#   lib/libRapidOcrOnnx.so    OCR engine (statically linked OpenCV)
#   lib/libonnxruntime.so.*   ONNX Runtime, bundled into the AppImage
#   models/                   PP-OCRv4 det/rec + cls + keys
#
# Upstream publishes Windows/macOS prebuilts only, so Linux builds one from
# RapidOcrOnnx source. Building the minimal static OpenCV is the slow part;
# it is cached in scripts/linux/.cache/rapidocr.
#
# Requirements: cmake, g++, git, curl, unzip, python3. Runs unchanged inside
# the qt-builder image (scripts/linux/qt-builder.Dockerfile).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ARCH="${ARCH:-x64}"
CACHE="${CACHE:-$ROOT_DIR/scripts/linux/.cache/rapidocr}"
DEST="${DEST:-$ROOT_DIR/QuickDesk/3rdparty/rapidocr/$ARCH}"
JOBS="${JOBS:-$(nproc)}"

OCR_REPO="https://github.com/barry-ran/RapidOcrOnnx.git"
OCR_COMMIT="313dd3e81a0a014516e5393a149740ab91f9e456"
ORT_VERSION="1.20.1"
OPENCV_VERSION="4.10.0"
MODELS_ZIP="https://github.com/barry-ran/3rdparty-prebuilt/releases/download/1.0.0/rapidocr-windows-x64-shared-release-mt.zip"

SRC="$CACHE/RapidOcrOnnx"
ORT="$CACHE/onnxruntime"
OPENCV="$CACHE/opencv-static"
BUILD="$CACHE/build"

for tool in cmake g++ git curl unzip python3; do
    command -v "$tool" >/dev/null || { echo "missing build tool: $tool" >&2; exit 1; }
done
mkdir -p "$CACHE"

echo "[1/4] Sources"
if [ ! -d "$SRC/.git" ]; then
    git clone "$OCR_REPO" "$SRC"
fi
git -C "$SRC" checkout -q "$OCR_COMMIT"

if [ ! -f "$ORT/lib/libonnxruntime.so" ]; then
    curl -fsSL -o "$CACHE/onnxruntime.tgz" \
        "https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/onnxruntime-linux-x64-${ORT_VERSION}.tgz"
    rm -rf "$ORT" && mkdir -p "$ORT"
    tar xzf "$CACHE/onnxruntime.tgz" -C "$ORT" --strip-components=1
fi

echo "[2/4] Static OpenCV (cached at $OPENCV)"
if [ ! -f "$OPENCV/lib/libopencv_core.a" ]; then
    if [ ! -d "$CACHE/opencv-src" ]; then
        curl -fsSL -o "$CACHE/opencv.tar.gz" \
            "https://github.com/opencv/opencv/archive/refs/tags/${OPENCV_VERSION}.tar.gz"
        tar xzf "$CACHE/opencv.tar.gz" -C "$CACHE"
        mv "$CACHE/opencv-${OPENCV_VERSION}" "$CACHE/opencv-src"
    fi
    cmake -S "$CACHE/opencv-src" -B "$CACHE/opencv-build" \
        -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="$OPENCV" \
        -DBUILD_SHARED_LIBS=OFF -DBUILD_LIST=core,imgproc,imgcodecs \
        -DBUILD_TESTS=OFF -DBUILD_PERF_TESTS=OFF -DBUILD_EXAMPLES=OFF \
        -DBUILD_opencv_apps=OFF -DBUILD_JAVA=OFF -DBUILD_opencv_python3=OFF -DBUILD_DOCS=OFF \
        -DWITH_IPP=OFF -DWITH_OPENCL=OFF -DWITH_TBB=OFF -DWITH_OPENMP=OFF \
        -DWITH_LAPACK=OFF -DWITH_EIGEN=OFF -DWITH_FFMPEG=OFF -DWITH_GSTREAMER=OFF \
        -DWITH_PROTOBUF=OFF -DWITH_V4L=OFF -DWITH_GTK=OFF -DWITH_QT=OFF \
        -DWITH_TIFF=OFF -DWITH_WEBP=OFF -DWITH_OPENJPEG=OFF -DWITH_OPENEXR=OFF \
        -DWITH_1394=OFF -DWITH_ADE=OFF -DWITH_JPEG=ON -DWITH_PNG=ON \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON
    cmake --build "$CACHE/opencv-build" -j"$JOBS"
    cmake --install "$CACHE/opencv-build"
fi

echo "[3/4] libRapidOcrOnnx.so"
mkdir -p "$SRC/onnxruntime-static/linux" "$SRC/opencv-static"
ln -sfn "$OPENCV" "$SRC/opencv-static/linux"
cat > "$SRC/onnxruntime-static/linux/OnnxRuntimeConfig.cmake" <<EOF
set(OnnxRuntime_FOUND TRUE)
set(OnnxRuntime_VERSION ${ORT_VERSION})
set(OnnxRuntime_INCLUDE_DIRS ${ORT}/include)
set(OnnxRuntime_LIBS ${ORT}/lib/libonnxruntime.so)
EOF
rm -rf "$BUILD"
cmake -S "$SRC" -B "$BUILD" \
    -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="$CACHE/install" \
    -DOCR_OUTPUT=CLIB -DOCR_BENCHMARK=OFF
cmake --build "$BUILD" -j"$JOBS"
cmake --install "$BUILD"

echo "[4/4] Assembling $DEST"
rm -rf "$DEST"
mkdir -p "$DEST/lib" "$DEST/models"
cp -r "$CACHE/install/include" "$DEST/include"
cp "$CACHE/install/lib/libRapidOcrOnnx.so" "$DEST/lib/"
cp -P "$ORT/lib/libonnxruntime.so" "$ORT/lib/libonnxruntime.so.1" \
      "$ORT/lib/libonnxruntime.so.$ORT_VERSION" "$DEST/lib/"

# PP-OCRv4 set (OcrEngine prefers v4 and falls back to v3 only if v4 is absent).
if [ ! -f "$CACHE/models/ch_PP-OCRv4_det_infer.onnx" ]; then
    curl -fsSL -o "$CACHE/models.zip" "$MODELS_ZIP"
    rm -rf "$CACHE/models" && mkdir -p "$CACHE/models"
    ( cd "$CACHE/models" && unzip -j -q "$CACHE/models.zip" "*/models/*" )
fi
for m in ch_PP-OCRv4_det_infer.onnx ch_PP-OCRv4_rec_infer.onnx \
         ch_ppocr_mobile_v2.0_cls_infer.onnx ppocr_keys_v1.txt; do
    cp "$CACHE/models/$m" "$DEST/models/"
done

echo "OK: $DEST ($(du -sh "$DEST" | cut -f1))"
