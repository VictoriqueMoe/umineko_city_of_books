import { afterEach, describe, expect, it, vi } from "vitest";
import {
    NO_ROOM_PREF_OVERRIDE,
    readStoredPref,
    removeStoredPref,
    roomInfoExpanded,
    roomInfoExpandedKey,
    roomSidebarCollapsedKey,
    sidebarCollapsed,
    storeRoomInfoExpanded,
    storeSidebarCollapsed,
    writeStoredPref,
} from "./roomPrefs";

function denyStorage(method: "getItem" | "setItem" | "removeItem") {
    return vi.spyOn(Storage.prototype, method).mockImplementation(() => {
        throw new DOMException("The operation is insecure.", "SecurityError");
    });
}

afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
});

describe("pref keys", () => {
    it("scopes the sidebar key to the room", () => {
        // given
        const roomId = "room-1";

        // when
        const key = roomSidebarCollapsedKey(roomId);

        // then
        expect(key).toBe("ut-room-sidebar-collapsed-room-1");
    });

    it("scopes the room info key to the room", () => {
        // given
        const roomId = "room-1";

        // when
        const key = roomInfoExpandedKey(roomId);

        // then
        expect(key).toBe("roomInfoExpanded:room-1");
    });

    it("gives two rooms two different keys for the same preference", () => {
        // given
        const rooms = ["room-1", "room-2"];

        // when
        const keys = rooms.map(roomSidebarCollapsedKey);

        // then
        expect(keys[0]).not.toBe(keys[1]);
    });
});

describe("readStoredPref", () => {
    it("returns what was stored", () => {
        // given
        window.localStorage.setItem("k", "v");

        // when
        const value = readStoredPref("k");

        // then
        expect(value).toBe("v");
    });

    it("returns null for a key that was never written", () => {
        // given
        const key = "never-written";

        // when
        const value = readStoredPref(key);

        // then
        expect(value).toBeNull();
    });

    it("returns null instead of throwing when storage is blocked", () => {
        // given
        denyStorage("getItem");

        // when
        const value = readStoredPref("k");

        // then
        expect(value).toBeNull();
    });
});

describe("writeStoredPref", () => {
    it("reports success and stores the value", () => {
        // given
        const key = "k";

        // when
        const stored = writeStoredPref(key, "v");

        // then
        expect(stored).toBe(true);
        expect(window.localStorage.getItem(key)).toBe("v");
    });

    it("reports failure instead of throwing when the quota is exhausted", () => {
        // given
        const setItem = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
            throw new DOMException("The quota has been exceeded.", "QuotaExceededError");
        });

        // when
        const stored = writeStoredPref("k", "v");

        // then
        expect(stored).toBe(false);
        expect(setItem).toHaveBeenCalledWith("k", "v");
    });
});

describe("removeStoredPref", () => {
    it("reports success and clears the value", () => {
        // given
        window.localStorage.setItem("k", "v");

        // when
        const removed = removeStoredPref("k");

        // then
        expect(removed).toBe(true);
        expect(window.localStorage.getItem("k")).toBeNull();
    });

    it("reports failure instead of throwing when storage is blocked", () => {
        // given
        denyStorage("removeItem");

        // when
        const removed = removeStoredPref("k");

        // then
        expect(removed).toBe(false);
    });
});

describe("sidebarCollapsed", () => {
    it("is expanded with no room selected", () => {
        // given
        const roomId = null;

        // when
        const collapsed = sidebarCollapsed(roomId, NO_ROOM_PREF_OVERRIDE);

        // then
        expect(collapsed).toBe(false);
    });

    it("is expanded when the room has no stored preference", () => {
        // given
        const roomId = "room-1";

        // when
        const collapsed = sidebarCollapsed(roomId, NO_ROOM_PREF_OVERRIDE);

        // then
        expect(collapsed).toBe(false);
    });

    it("is collapsed when the room stored the collapsed marker", () => {
        // given
        window.localStorage.setItem(roomSidebarCollapsedKey("room-1"), "1");

        // when
        const collapsed = sidebarCollapsed("room-1", NO_ROOM_PREF_OVERRIDE);

        // then
        expect(collapsed).toBe(true);
    });

    it("reads only the selected room's own key", () => {
        // given
        window.localStorage.setItem(roomSidebarCollapsedKey("room-1"), "1");

        // when
        const collapsed = sidebarCollapsed("room-2", NO_ROOM_PREF_OVERRIDE);

        // then
        expect(collapsed).toBe(false);
    });

    it("lets an override for the same room win over storage", () => {
        // given
        window.localStorage.setItem(roomSidebarCollapsedKey("room-1"), "1");
        const override = { key: roomSidebarCollapsedKey("room-1"), value: false };

        // when
        const collapsed = sidebarCollapsed("room-1", override);

        // then
        expect(collapsed).toBe(false);
    });

    it("discards an override left behind by a different room", () => {
        // given
        const override = { key: roomSidebarCollapsedKey("room-1"), value: true };

        // when
        const collapsed = sidebarCollapsed("room-2", override);

        // then
        expect(collapsed).toBe(false);
    });

    it("falls back to storage when the override carries no value", () => {
        // given
        window.localStorage.setItem(roomSidebarCollapsedKey("room-1"), "1");
        const override = { key: roomSidebarCollapsedKey("room-1"), value: null };

        // when
        const collapsed = sidebarCollapsed("room-1", override);

        // then
        expect(collapsed).toBe(true);
    });

    it("reads as expanded rather than throwing when storage is blocked", () => {
        // given
        denyStorage("getItem");

        // when
        const collapsed = sidebarCollapsed("room-1", NO_ROOM_PREF_OVERRIDE);

        // then
        expect(collapsed).toBe(false);
    });
});

describe("storeSidebarCollapsed", () => {
    it("writes the marker when collapsing", () => {
        // given
        const roomId = "room-1";

        // when
        const stored = storeSidebarCollapsed(roomId, true);

        // then
        expect(stored).toBe(true);
        expect(window.localStorage.getItem(roomSidebarCollapsedKey(roomId))).toBe("1");
    });

    it("removes the marker when expanding rather than writing a false", () => {
        // given
        window.localStorage.setItem(roomSidebarCollapsedKey("room-1"), "1");

        // when
        const stored = storeSidebarCollapsed("room-1", false);

        // then
        expect(stored).toBe(true);
        expect(window.localStorage.getItem(roomSidebarCollapsedKey("room-1"))).toBeNull();
    });

    it("reports failure when a private-mode store refuses the write", () => {
        // given
        denyStorage("setItem");

        // when
        const stored = storeSidebarCollapsed("room-1", true);

        // then
        expect(stored).toBe(false);
    });

    it("reports failure when a private-mode store refuses the removal", () => {
        // given
        denyStorage("removeItem");

        // when
        const stored = storeSidebarCollapsed("room-1", false);

        // then
        expect(stored).toBe(false);
    });
});

describe("roomInfoExpanded", () => {
    it("follows the viewport when the room has no stored preference", () => {
        // given
        const roomId = "room-1";

        // when
        const wide = roomInfoExpanded(roomId, NO_ROOM_PREF_OVERRIDE, true);
        const narrow = roomInfoExpanded(roomId, NO_ROOM_PREF_OVERRIDE, false);

        // then
        expect(wide).toBe(true);
        expect(narrow).toBe(false);
    });

    it("lets a stored collapse survive a wide viewport", () => {
        // given
        window.localStorage.setItem(roomInfoExpandedKey("room-1"), "false");

        // when
        const expanded = roomInfoExpanded("room-1", NO_ROOM_PREF_OVERRIDE, true);

        // then
        expect(expanded).toBe(false);
    });

    it("lets a stored expansion survive a narrow viewport", () => {
        // given
        window.localStorage.setItem(roomInfoExpandedKey("room-1"), "true");

        // when
        const expanded = roomInfoExpanded("room-1", NO_ROOM_PREF_OVERRIDE, false);

        // then
        expect(expanded).toBe(true);
    });

    it("lets an override for the same room win over storage", () => {
        // given
        window.localStorage.setItem(roomInfoExpandedKey("room-1"), "false");
        const override = { key: roomInfoExpandedKey("room-1"), value: true };

        // when
        const expanded = roomInfoExpanded("room-1", override, false);

        // then
        expect(expanded).toBe(true);
    });

    it("discards an override left behind by a different room", () => {
        // given
        const override = { key: roomInfoExpandedKey("room-1"), value: false };

        // when
        const expanded = roomInfoExpanded("room-2", override, true);

        // then
        expect(expanded).toBe(true);
    });

    it("keeps a keyless override with no room selected, since it has nowhere to persist", () => {
        // given
        const override = { key: null, value: false };

        // when
        const expanded = roomInfoExpanded(null, override, true);

        // then
        expect(expanded).toBe(false);
    });

    it("follows the viewport rather than throwing when storage is blocked", () => {
        // given
        denyStorage("getItem");

        // when
        const expanded = roomInfoExpanded("room-1", NO_ROOM_PREF_OVERRIDE, true);

        // then
        expect(expanded).toBe(true);
    });
});

describe("storeRoomInfoExpanded", () => {
    it("writes the flag as a word rather than a marker", () => {
        // given
        const roomId = "room-1";

        // when
        const stored = storeRoomInfoExpanded(roomId, false);

        // then
        expect(stored).toBe(true);
        expect(window.localStorage.getItem(roomInfoExpandedKey(roomId))).toBe("false");
    });

    it("reports no persistence with no room selected", () => {
        // given
        const roomId = null;

        // when
        const stored = storeRoomInfoExpanded(roomId, true);

        // then
        expect(stored).toBe(false);
    });

    it("reports failure when a private-mode store refuses the write", () => {
        // given
        denyStorage("setItem");

        // when
        const stored = storeRoomInfoExpanded("room-1", true);

        // then
        expect(stored).toBe(false);
    });
});
