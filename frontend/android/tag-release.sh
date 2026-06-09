#!/bin/bash

set -e

RESET="\033[0m"
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"

BUILD_FILE="app/build.gradle"
ACTIONS_URL="https://github.com/VictoriqueMoe/umineko_city_of_books/actions"

if [ ! -f "$BUILD_FILE" ]; then
    echo -e "${RED}Error: $BUILD_FILE not found!${RESET}"
    echo -e "${YELLOW}Make sure you're running this script from the frontend/android/ directory.${RESET}"
    exit 1
fi

CURRENT_VERSION=$(grep -oP 'versionName\s+"\K[^"]+' "$BUILD_FILE")
CURRENT_CODE=$(grep -oP 'versionCode\s+\K\d+' "$BUILD_FILE")

echo -e "${BLUE}Current version: ${GREEN}${CURRENT_VERSION}${BLUE} (code: ${CURRENT_CODE})${RESET}"
echo ""
read -p "$(echo -e "${YELLOW}Enter new version (x.x.x format): ${RESET}")" NEW_VERSION

if [ -z "$NEW_VERSION" ]; then
    echo -e "${RED}Error: Version cannot be empty!${RESET}"
    exit 1
fi

if ! echo "$NEW_VERSION" | grep -qP '^\d+\.\d+\.\d+$'; then
    echo -e "${RED}Error: Version must be in x.x.x format${RESET}"
    exit 1
fi

NEW_CODE=$((CURRENT_CODE + 1))

echo ""
echo -e "${GREEN}┌─────────────────────────────────────┐${RESET}"
echo -e "${GREEN}│ ${BLUE}Version: ${CURRENT_VERSION} → ${NEW_VERSION}${RESET}"
echo -e "${GREEN}│ ${BLUE}Code:    ${CURRENT_CODE} → ${NEW_CODE}${RESET}"
echo -e "${GREEN}│ ${BLUE}Tag:     android-v${NEW_VERSION}${RESET}"
echo -e "${GREEN}└─────────────────────────────────────┘${RESET}"
echo ""
read -p "$(echo -e "${YELLOW}Proceed? (y/n): ${RESET}")" CONFIRM

if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "yes" ]; then
    echo -e "${YELLOW}Cancelled.${RESET}"
    exit 0
fi

echo -e "\n${BLUE}[1/4] Updating version in ${BUILD_FILE}...${RESET}"
sed -i "s/versionCode ${CURRENT_CODE}/versionCode ${NEW_CODE}/" "$BUILD_FILE"
sed -i "s/versionName \"${CURRENT_VERSION}\"/versionName \"${NEW_VERSION}\"/" "$BUILD_FILE"
echo -e "${GREEN}✓ Version updated${RESET}"

echo -e "\n${BLUE}[2/4] Committing...${RESET}"
git add "$BUILD_FILE"
git commit -m "android: bump version to ${NEW_VERSION}"
echo -e "${GREEN}✓ Committed${RESET}"

echo -e "\n${BLUE}[3/4] Tagging android-v${NEW_VERSION}...${RESET}"
git tag "android-v${NEW_VERSION}"
echo -e "${GREEN}✓ Tagged${RESET}"

echo -e "\n${BLUE}[4/4] Pushing commit and tag...${RESET}"
git push
git push origin "android-v${NEW_VERSION}"
echo -e "${GREEN}✓ Pushed${RESET}"

echo ""
echo -e "${GREEN}╔════════════════════════════════════════╗${RESET}"
echo -e "${GREEN}║   Release android-v${NEW_VERSION} triggered!${RESET}"
echo -e "${GREEN}╚════════════════════════════════════════╝${RESET}"
echo -e "${BLUE}Check: ${ACTIONS_URL}${RESET}"
