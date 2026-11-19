#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob

packages=(release/*.zip release/*.dmg)
if [[ ${#packages[@]} -eq 0 ]]; then
  echo "No macOS packages found" >&2
  exit 1
fi

verify_app() {
  local app="$1"
  local executable="$app/Contents/MacOS/Darube"
  local engine="$app/Contents/Resources/engine/bin/engine"
  local expected_arch
  expected_arch="$(uname -m)"

  codesign --verify --deep --strict --verbose=2 "$app"
  # Resources/engine is outside the usual nested-code directories, so check
  # its executable signature explicitly as well as the containing bundle seal.
  codesign --verify --strict --verbose=2 "$engine"
  test -x "$executable"
  test -x "$engine"
  lipo -verify_arch "$expected_arch" "$executable"
  lipo -verify_arch "$expected_arch" "$engine"
}

work_dir="$(mktemp -d)"
mount_point=""
cleanup() {
  if [[ -n "$mount_point" ]]; then
    hdiutil detach "$mount_point"
  fi
  rm -rf "$work_dir"
}
trap cleanup EXIT

for package in "${packages[@]}"; do
  echo "Verifying $package"
  if [[ "$package" == *.zip ]]; then
    unpack_dir="$(mktemp -d "$work_dir/zip.XXXXXX")"
    ditto -x -k "$package" "$unpack_dir"
    verify_app "$unpack_dir/Darube.app"
  else
    hdiutil verify "$package"
    mount_point="$work_dir/dmg"
    mkdir -p "$mount_point"
    hdiutil attach "$package" -readonly -nobrowse -mountpoint "$mount_point"
    verify_app "$mount_point/Darube.app"
    hdiutil detach "$mount_point"
    mount_point=""
  fi
done
