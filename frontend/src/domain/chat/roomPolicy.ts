import type { ChatRoom, SiteRole } from "../../types/api";
import { isSiteStaff } from "../permissions";
import { parseServerDate } from "../../utils/time";

export interface RoomPolicyViewer {
    role?: SiteRole;
}

export interface RoomViewerPolicy {
    isHost: boolean;
    isSystem: boolean;
    isSiteMod: boolean;
    canModerateRoom: boolean;
}

export function roomViewerPolicy(
    room: ChatRoom | null | undefined,
    viewer: RoomPolicyViewer | null | undefined,
): RoomViewerPolicy {
    const isHost = room?.viewer_role === "host";
    const isSystem = room?.is_system ?? false;
    const isSiteMod = isSiteStaff(viewer?.role);

    return {
        isHost,
        isSystem,
        isSiteMod,
        canModerateRoom: isHost || isSiteMod,
    };
}

export function isTimeoutActive(timeoutUntil: string | null | undefined, now: number): boolean {
    const until = parseServerDate(timeoutUntil);
    if (!until) {
        return false;
    }

    return until.getTime() > now;
}
