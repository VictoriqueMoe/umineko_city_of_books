#!/usr/bin/env bash
set -euo pipefail

GRADLE_FILE="app/build.gradle"

current_name=$(grep -oP 'versionName "\K[^"]+' "$GRADLE_FILE")
current_code=$(grep -oP 'versionCode \K\d+' "$GRADLE_FILE")

echo "Current version: $current_name (code $current_code)"
read -rp "New version (semver, e.g. 1.0.1): " new_name

if ! [[ "$new_name" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "Invalid version: $new_name (expected x.y.z)"
	exit 1
fi

new_code=$((current_code + 1))

sed -i "s/versionName \"$current_name\"/versionName \"$new_name\"/" "$GRADLE_FILE"
sed -i "s/versionCode $current_code/versionCode $new_code/" "$GRADLE_FILE"

git add "$GRADLE_FILE"
git commit -m "android: bump version to $new_name"
git tag "android-v$new_name"
git push
git push origin "android-v$new_name"

echo "Tagged android-v$new_name (versionCode $new_code). The Android Release workflow will build and publish the APK."
