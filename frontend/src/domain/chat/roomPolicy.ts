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

export type RoomKind = "pair" | "group";

export type RoomIdentitySource = "pair_key" | "row_id";

export type RoomDisplayNameSource = "other_member" | "owned";

export type RoomRoleModel = "none" | "host_and_member";

export type RoomModerationAuthority = "site_staff" | "host_and_site_staff";

export type RoomBlockGate = "symmetric" | "from_host" | "none";

export type RoomReadReceipts = "pairwise" | "none";

export type RoomBannedWordAction = "delete" | "delete_and_kick";

export type RoomBotTrigger = "every_message" | "mention_or_reply";

export type RoomBotPromptScope = "whole_history" | "reply_chain";

export interface RoomCapabilities extends RoomViewerPolicy {
    kind: RoomKind;
    isMember: boolean;

    identity: RoomIdentitySource;
    displayName: RoomDisplayNameSource;
    roles: RoomRoleModel;
    moderation: RoomModerationAuthority;

    canDeleteOthersMessages: boolean;
    canPinMessages: boolean;
    canEditSettings: boolean;
    canInvite: boolean;
    canSelfJoin: boolean;
    canStartWatchParty: boolean;
    canTimeOutMembers: boolean;
    canDestroyRoom: boolean;

    isDiscoverable: boolean;
    hasPublicPreview: boolean;
    isAudited: boolean;
    archivesWhenStale: boolean;
    leaveMeansHide: boolean;
    destroysWhenEmpty: boolean;
    countsTowardsUnreadBadge: boolean;
    newThreadGatedByDmsEnabled: boolean;

    blockGatesSending: RoomBlockGate;
    readReceipts: RoomReadReceipts;
    bannedWordAction: RoomBannedWordAction;
    botTrigger: RoomBotTrigger;
    botPromptScope: RoomBotPromptScope;
    botCooldown: boolean;
}

const hostBlockedSystemKinds = ["live_stream", "watch_party"];

function viewerFacts(room: ChatRoom | null | undefined, viewer: RoomPolicyViewer | null | undefined) {
    return {
        isHost: room?.viewer_role === "host",
        isSystem: room?.is_system ?? false,
        isSiteMod: isSiteStaff(viewer?.role),
    };
}

function blockGate(isPair: boolean, isSystem: boolean, systemKind: string | undefined): RoomBlockGate {
    if (isPair) {
        return "symmetric";
    }

    if (!isSystem || hostBlockedSystemKinds.includes(systemKind ?? "")) {
        return "from_host";
    }

    return "none";
}

export function roomViewerPolicy(
    room: ChatRoom | null | undefined,
    viewer: RoomPolicyViewer | null | undefined,
): RoomViewerPolicy {
    const { isHost, isSystem, isSiteMod } = viewerFacts(room, viewer);

    return {
        isHost,
        isSystem,
        isSiteMod,
        canModerateRoom: isHost || isSiteMod,
    };
}

export function roomCapabilities(
    room: ChatRoom | null | undefined,
    viewer: RoomPolicyViewer | null | undefined,
): RoomCapabilities {
    const { isHost, isSystem, isSiteMod } = viewerFacts(room, viewer);

    const isPair = room?.type === "dm";
    const isMember = room?.is_member ?? false;
    const isPublic = room?.is_public ?? false;
    const isArchived = Boolean(room?.archived_at);

    const canModerateRoom = isPair ? isSiteMod : isHost || isSiteMod;
    const isOpenGroup = !isPair && !isSystem;
    const canManageGroup = isOpenGroup && canModerateRoom;

    return {
        kind: isPair ? "pair" : "group",
        isHost,
        isSystem,
        isSiteMod,
        isMember,

        identity: isPair ? "pair_key" : "row_id",
        displayName: isPair ? "other_member" : "owned",
        roles: isPair ? "none" : "host_and_member",
        moderation: isPair ? "site_staff" : "host_and_site_staff",

        canModerateRoom,
        canDeleteOthersMessages: canModerateRoom,
        canPinMessages: isPair ? isMember || isSiteMod : canModerateRoom,
        canEditSettings: canManageGroup,
        canInvite: canManageGroup,
        canSelfJoin: isOpenGroup && isPublic,
        canStartWatchParty: isOpenGroup && isMember,
        canTimeOutMembers: !isSystem && canModerateRoom,
        canDestroyRoom: canManageGroup,

        isDiscoverable: isOpenGroup && isPublic && !isArchived,
        hasPublicPreview: isOpenGroup && isPublic,
        isAudited: isOpenGroup && isPublic,
        archivesWhenStale: isOpenGroup,
        leaveMeansHide: isPair,
        destroysWhenEmpty: isPair,
        countsTowardsUnreadBadge: isPair,
        newThreadGatedByDmsEnabled: isPair,

        blockGatesSending: blockGate(isPair, isSystem, room?.system_kind),
        readReceipts: isPair ? "pairwise" : "none",
        bannedWordAction: isPair ? "delete" : "delete_and_kick",
        botTrigger: isPair ? "every_message" : "mention_or_reply",
        botPromptScope: isPair ? "whole_history" : "reply_chain",
        botCooldown: true,
    };
}

export function isTimeoutActive(timeoutUntil: string | null | undefined, now: number): boolean {
    const until = parseServerDate(timeoutUntil);
    if (!until) {
        return false;
    }

    return until.getTime() > now;
}
