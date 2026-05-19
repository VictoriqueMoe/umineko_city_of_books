import { useState } from "react";
import type { WatchPartyParticipant } from "../../../types/api";
import type { SiteRole } from "../../../utils/permissions";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import styles from "./WatchParty.module.css";

interface WatchPartyParticipantsProps {
    participants: WatchPartyParticipant[];
    viewerUserId: string;
    viewerRole: SiteRole | undefined;
    viewerHasControl: boolean;
    ownerUserId: string;
    onTransferControl: (userId: string) => Promise<void>;
    onKick: (userId: string) => Promise<void>;
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

function effectiveRank(role: SiteRole | undefined, isOwner: boolean): number {
    const rank = siteRoleRank(role);
    if (isOwner && rank < 1) {
        return 1;
    }
    return rank;
}

export function WatchPartyParticipants({
    participants,
    viewerUserId,
    viewerRole,
    viewerHasControl,
    ownerUserId,
    onTransferControl,
    onKick,
}: WatchPartyParticipantsProps) {
    const [busyUserId, setBusyUserId] = useState<string | null>(null);

    const viewerIsOwner = viewerUserId === ownerUserId;
    const viewerRank = effectiveRank(viewerRole, viewerIsOwner);
    const controller = participants.find(p => p.has_control);
    const controllerRank = controller ? effectiveRank(controller.user.role, controller.user.id === ownerUserId) : 0;
    const canOutrankController = !controller || viewerRank > controllerRank;

    const runAction = async (rowUserId: string, action: () => Promise<void>) => {
        setBusyUserId(rowUserId);
        try {
            await action();
        } finally {
            setBusyUserId(null);
        }
    };

    return (
        <div className={styles.participantStrip}>
            <span className={styles.participantStripLabel}>
                {participants.length} {participants.length === 1 ? "watcher" : "watchers"}
            </span>
            <ul className={styles.participantStripList}>
                {participants.map(p => {
                    const isSelf = p.user.id === viewerUserId;
                    const isOwner = p.user.id === ownerUserId;
                    const targetRank = effectiveRank(p.user.role, isOwner);

                    let transferLabel: string | null = null;
                    let transferTarget: string | null = null;
                    if (isSelf) {
                        if (!p.has_control && canOutrankController) {
                            transferLabel = "Reclaim control";
                            transferTarget = viewerUserId;
                        }
                    } else if (p.has_control) {
                        if (canOutrankController) {
                            transferLabel = "Reclaim";
                            transferTarget = viewerUserId;
                        }
                    } else if (viewerHasControl || canOutrankController) {
                        transferLabel = "Pass control";
                        transferTarget = p.user.id;
                    }

                    const canKick = !isSelf && viewerRank > targetRank;

                    return (
                        <li key={p.user.id} className={styles.participantPill}>
                            <ProfileLink user={p.user} size="small" />
                            {isOwner && <span className={styles.ownerPill}>owner</span>}
                            {p.has_control && <span className={styles.controlPill}>control</span>}
                            {transferLabel && transferTarget && (
                                <button
                                    type="button"
                                    className={styles.controlToggle}
                                    onClick={() => runAction(p.user.id, () => onTransferControl(transferTarget))}
                                    disabled={busyUserId === p.user.id}
                                >
                                    {transferLabel}
                                </button>
                            )}
                            {canKick && (
                                <button
                                    type="button"
                                    className={styles.kickToggle}
                                    onClick={() => runAction(p.user.id, () => onKick(p.user.id))}
                                    disabled={busyUserId === p.user.id}
                                >
                                    Kick
                                </button>
                            )}
                        </li>
                    );
                })}
            </ul>
        </div>
    );
}
