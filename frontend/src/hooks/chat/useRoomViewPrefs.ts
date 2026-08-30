import { useCallback, useState } from "react";
import {
    NO_ROOM_PREF_OVERRIDE,
    roomInfoExpanded,
    roomInfoExpandedKey,
    roomSidebarCollapsedKey,
    sidebarCollapsed as resolveSidebarCollapsed,
    storeRoomInfoExpanded,
    storeSidebarCollapsed,
    type RoomPrefOverride,
} from "../../domain/chat/roomPrefs";

const WIDE_VIEWPORT = "(min-width: 769px)";

export interface RoomViewPrefs {
    sidebarCollapsed: boolean;
    toggleSidebar: () => void;
    descExpanded: boolean;
    toggleDescExpanded: () => void;
}

function resolveDescExpanded(roomId: string | undefined, override: RoomPrefOverride): boolean {
    if (typeof window === "undefined") {
        return true;
    }

    return roomInfoExpanded(roomId, override, window.matchMedia(WIDE_VIEWPORT).matches);
}

export function useRoomViewPrefs(roomId: string | undefined): RoomViewPrefs {
    const [sidebarOverride, setSidebarOverride] = useState<RoomPrefOverride>(NO_ROOM_PREF_OVERRIDE);
    const [descOverride, setDescOverride] = useState<RoomPrefOverride>(NO_ROOM_PREF_OVERRIDE);

    const sidebarCollapsed = resolveSidebarCollapsed(roomId, sidebarOverride);
    const descExpanded = resolveDescExpanded(roomId, descOverride);

    const toggleSidebar = useCallback(() => {
        if (!roomId) {
            return;
        }

        const next = !sidebarCollapsed;
        storeSidebarCollapsed(roomId, next);
        setSidebarOverride({ key: roomSidebarCollapsedKey(roomId), value: next });
    }, [roomId, sidebarCollapsed]);

    const toggleDescExpanded = useCallback(() => {
        const next = !descExpanded;
        storeRoomInfoExpanded(roomId, next);
        setDescOverride({ key: roomId ? roomInfoExpandedKey(roomId) : null, value: next });
    }, [roomId, descExpanded]);

    return { sidebarCollapsed, toggleSidebar, descExpanded, toggleDescExpanded };
}
