#!/bin/bash

{
    cd "$(dirname "$0")"
    script_path=$(pwd)
    cd - > /dev/null
} &> /dev/null

old_cd=$(pwd)
cd "$(dirname "$0")"

build_mode=Release

while [ $# -gt 0 ]; do
    case "$(echo "$1" | tr '[:upper:]' '[:lower:]')" in
        debug)   build_mode=Debug ;;
        release) build_mode=Release ;;
    esac
    shift
done

echo
echo
echo "---------------------------------------------------------------"
echo "install dmgbuild"
echo "---------------------------------------------------------------"

python3 -m pip install -r "$script_path/package/requirements.txt"
if [ $? -ne 0 ]; then
    echo "[!] pip install failed"
    cd "$old_cd"
    exit 1
fi

echo
echo
echo "---------------------------------------------------------------"
echo "create DMG"
echo "---------------------------------------------------------------"

python3 "$script_path/package/package_dmg.py" "$build_mode"
if [ $? -ne 0 ]; then
    echo "[!] create DMG failed"
    cd "$old_cd"
    exit 1
fi

echo
echo
echo "---------------------------------------------------------------"
echo "[*] DMG package finished!"
echo "---------------------------------------------------------------"

cd "$old_cd"
exit 0
