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
clean_output=false
errno=1

echo
echo
echo "---------------------------------------------------------------"
echo "parse arguments"
echo "---------------------------------------------------------------"

while [ $# -gt 0 ]; do
    case "$(echo "$1" | tr '[:upper:]' '[:lower:]')" in
        debug)          build_mode=Debug ;;
        release)        build_mode=Release ;;
        minsizerel)     build_mode=MinSizeRel ;;
        relwithdebinfo) build_mode=RelWithDebInfo ;;
        clean)          clean_output=true ;;
    esac
    shift
done

echo "[*] build mode: $build_mode"
echo "[*] clean output: $clean_output"
echo

qt_cmake_path="$ENV_QT_PATH/macos"
echo "Qt cmake path: $qt_cmake_path"

echo
echo
echo "---------------------------------------------------------------"
echo "begin CMake build"
echo "---------------------------------------------------------------"

output_path="$script_path/../output"
if [ "$clean_output" = "true" ]; then
    if [ -d "$output_path" ]; then
        echo "[*] cleaning output dir: $output_path"
        rm -rf "$output_path"
    fi
else
    echo "[*] output dir: $output_path"
fi

temp_path="$script_path/../build-temp"
if [ "$clean_output" = "true" ]; then
    if [ -d "$temp_path" ]; then
        echo "[*] cleaning temp dir: $temp_path"
        rm -rf "$temp_path"
    fi
else
    echo "[*] temp dir (incremental build): $temp_path"
fi

if [ ! -d "$temp_path" ]; then
    mkdir -p "$temp_path"
fi
cd "$temp_path"

cmake_params="-DCMAKE_PREFIX_PATH=$qt_cmake_path -DCMAKE_BUILD_TYPE=$build_mode -DCMAKE_OSX_ARCHITECTURES=arm64"
echo "[*] CMake params: $cmake_params"
echo

cmake $cmake_params ../
if [ $? -ne 0 ]; then
    echo "[!] CMake configure failed"
    cd "$old_cd"
    exit 1
fi

echo
echo "[*] building..."
cmake --build . --config "$build_mode" -j$(sysctl -n hw.ncpu)
if [ $? -ne 0 ]; then
    echo "[!] CMake build failed"
    cd "$old_cd"
    exit 1
fi

echo
echo
echo "---------------------------------------------------------------"
echo "[*] build finished!"
echo "---------------------------------------------------------------"

cd "$old_cd"
exit 0
