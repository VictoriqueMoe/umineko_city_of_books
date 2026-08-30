import type { SiteRole, WatchPartyParticipant } from "../../types/api";

export interface WatchPartyControlViewer {
    userId: string;
    role: SiteRole | undefined;
    hasControl: boolean;
}

export interface WatchPartyControlContext {
    viewerRank: number;
    canOutrankController: boolean;
}

export interface WatchPartyRowControls {
    transferLabel: string | null;
    transferTarget: string | null;
    canKick: boolean;
}

function siteRoleRank(role: SiteRole | undefined): number {
    switch (role) {
        case "super_admin": {
            return 4;
        }
        case "admin": {
            return 3;
        }
        case "moderator": {
            return 2;
        }
        default: {
            return 0;
        }
    }
}

export function effectiveRank(role: SiteRole | undefined, isOwner: boolean): number {
    const rank = siteRoleRank(role);
    if (isOwner && rank < 1) {
        return 1;
    }
    return rank;
}

export function watchPartyControlContext(
    participants: WatchPartyParticipant[],
    viewer: WatchPartyControlViewer,
    ownerUserId: string,
): WatchPartyControlContext {
    const viewerRank = effectiveRank(viewer.role, viewer.userId === ownerUserId);

    const controller = participants.find(p => p.has_control);
    const controllerRank = controller ? effectiveRank(controller.user.role, controller.user.id === ownerUserId) : 0;

    return { viewerRank, canOutrankController: !controller || viewerRank > controllerRank };
}

export function watchPartyRowControls(
    participant: WatchPartyParticipant,
    viewer: WatchPartyControlViewer,
    ownerUserId: string,
    context: WatchPartyControlContext,
): WatchPartyRowControls {
    const isSelf = participant.user.id === viewer.userId;
    const isOwner = participant.user.id === ownerUserId;
    const targetRank = effectiveRank(participant.user.role, isOwner);

    let transferLabel: string | null = null;
    let transferTarget: string | null = null;
    if (isSelf) {
        if (!participant.has_control && context.canOutrankController) {
            transferLabel = "Reclaim control";
            transferTarget = viewer.userId;
        }
    } else if (participant.has_control) {
        if (context.canOutrankController) {
            transferLabel = "Reclaim";
            transferTarget = viewer.userId;
        }
    } else if (viewer.hasControl || context.canOutrankController) {
        transferLabel = "Pass control";
        transferTarget = participant.user.id;
    }

    return { transferLabel, transferTarget, canKick: !isSelf && context.viewerRank > targetRank };
}
