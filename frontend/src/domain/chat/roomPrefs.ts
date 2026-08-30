export interface RoomPrefOverride {
    key: string | null;
    value: boolean | null;
}

export const NO_ROOM_PREF_OVERRIDE: RoomPrefOverride = { key: null, value: null };

export function roomSidebarCollapsedKey(roomId: string): string {
    return `ut-room-sidebar-collapsed-${roomId}`;
}

export function roomInfoExpandedKey(roomId: string): string {
    return `roomInfoExpanded:${roomId}`;
}

export function readStoredPref(key: string): string | null {
    try {
        return window.localStorage.getItem(key);
    } catch {
        return null;
    }
}

export function writeStoredPref(key: string, value: string): boolean {
    try {
        window.localStorage.setItem(key, value);
        return true;
    } catch {
        return false;
    }
}

export function removeStoredPref(key: string): boolean {
    try {
        window.localStorage.removeItem(key);
        return true;
    } catch {
        return false;
    }
}

function resolvePrefFlag(
    key: string | null,
    override: RoomPrefOverride,
    trueValue: string,
    fallback: boolean,
): boolean {
    if (override.key === key && override.value !== null) {
        return override.value;
    }

    if (key === null) {
        return fallback;
    }

    const stored = readStoredPref(key);
    if (stored === null) {
        return fallback;
    }

    return stored === trueValue;
}

export function sidebarCollapsed(roomId: string | null | undefined, override: RoomPrefOverride): boolean {
    if (!roomId) {
        return false;
    }

    return resolvePrefFlag(roomSidebarCollapsedKey(roomId), override, "1", false);
}

export function storeSidebarCollapsed(roomId: string, collapsed: boolean): boolean {
    const key = roomSidebarCollapsedKey(roomId);
    if (collapsed) {
        return writeStoredPref(key, "1");
    }

    return removeStoredPref(key);
}

export function roomInfoExpanded(
    roomId: string | null | undefined,
    override: RoomPrefOverride,
    wideViewport: boolean,
): boolean {
    const key = roomId ? roomInfoExpandedKey(roomId) : null;

    return resolvePrefFlag(key, override, "true", wideViewport);
}

export function storeRoomInfoExpanded(roomId: string | null | undefined, expanded: boolean): boolean {
    if (!roomId) {
        return false;
    }

    return writeStoredPref(roomInfoExpandedKey(roomId), expanded ? "true" : "false");
}
