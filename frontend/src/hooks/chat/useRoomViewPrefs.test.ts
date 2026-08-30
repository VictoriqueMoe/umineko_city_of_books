import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useRoomViewPrefs } from "./useRoomViewPrefs";

function renderPrefs(roomId = "room-1") {
    return renderHook(() => useRoomViewPrefs(roomId));
}

describe("useRoomViewPrefs", () => {
    it("remembers that the viewer collapsed the sidebar", () => {
        // given
        const { result } = renderPrefs();

        // when
        act(() => {
            result.current.toggleSidebar();
        });

        // then
        expect(result.current.sidebarCollapsed).toBe(true);
        expect(localStorage.getItem("ut-room-sidebar-collapsed-room-1")).toBe("1");
    });

    it("opens the sidebar again and forgets the preference", () => {
        // given
        localStorage.setItem("ut-room-sidebar-collapsed-room-1", "1");
        const { result } = renderPrefs();

        // when
        act(() => {
            result.current.toggleSidebar();
        });

        // then
        expect(result.current.sidebarCollapsed).toBe(false);
        expect(localStorage.getItem("ut-room-sidebar-collapsed-room-1")).toBeNull();
    });

    it("remembers that the viewer expanded the room description", () => {
        // given
        const { result } = renderPrefs();

        // when
        act(() => {
            result.current.toggleDescExpanded();
        });

        // then
        expect(result.current.descExpanded).toBe(true);
        expect(localStorage.getItem("roomInfoExpanded:room-1")).toBe("true");
    });

    it("honours a stored description preference over the screen size", () => {
        // given
        localStorage.setItem("roomInfoExpanded:room-1", "true");

        // when
        const { result } = renderPrefs();

        // then
        expect(result.current.descExpanded).toBe(true);
    });
});
