import type { ChatRoomMember, User } from "../types/api";
import { isSiteStaff } from "./permissions";

export function effectiveMemberUser(member: ChatRoomMember): User {
    return {
        ...member.user,
        display_name:
            member.nickname && member.nickname.trim() !== ""
                ? member.nickname
                : member.user.display_name && member.user.display_name.trim() !== ""
                  ? member.user.display_name
                  : member.user.username,
        avatar_url:
            member.member_avatar_url && member.member_avatar_url.trim() !== ""
                ? member.member_avatar_url
                : member.user.avatar_url,
    };
}

export interface MemberModContext {
    selfId: string;
    isSystem: boolean;
    isSiteMod: boolean;
    canModerateRoom: boolean;
}

export interface MemberModPermissions {
    isSelf: boolean;
    timeoutIsActive: boolean;
    canKick: boolean;
    canEditNickname: boolean;
    canTimeout: boolean;
    canClearTimeout: boolean;
    canActOnMember: boolean;
}

export function memberModPermissions(member: ChatRoomMember, ctx: MemberModContext): MemberModPermissions {
    const { selfId, isSystem, isSiteMod, canModerateRoom } = ctx;

    const isSelf = member.user.id === selfId;
    const targetIsSiteMod = isSiteStaff(member.user.role);
    const targetIsHost = member.role === "host";
    const timeoutIsActive = Boolean(member.timeout_until);

    const canKick = canModerateRoom && !isSystem && !isSelf && !targetIsHost && !targetIsSiteMod;
    const canEditNickname = isSiteMod && !targetIsSiteMod && !isSelf && !isSystem;

    let canTimeout = false;
    if (canModerateRoom && !isSystem && !isSelf && !targetIsSiteMod) {
        if (isSiteMod) {
            canTimeout = true;
        } else {
            canTimeout = !targetIsHost;
        }
    }

    if (canTimeout && timeoutIsActive && member.timeout_set_by_staff && !isSiteMod) {
        canTimeout = false;
    }

    const canClearTimeout =
        canModerateRoom && !isSystem && timeoutIsActive && (isSiteMod || !member.timeout_set_by_staff);

    const canActOnMember = canKick || canEditNickname || canTimeout || canClearTimeout;

    return {
        isSelf,
        timeoutIsActive,
        canKick,
        canEditNickname,
        canTimeout,
        canClearTimeout,
        canActOnMember,
    };
}
