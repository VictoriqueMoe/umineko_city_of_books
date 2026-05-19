import { useState } from "react";
import type { WatchPartyParticipant } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import styles from "./WatchParty.module.css";

interface WatchPartyParticipantsProps {
    participants: WatchPartyParticipant[];
    viewerUserId: string;
    viewerIsOwner: boolean;
    viewerIsStaff: boolean;
    viewerHasControl: boolean;
    ownerUserId: string;
    onTransferControl: (userId: string) => Promise<void>;
}

export function WatchPartyParticipants({
    participants,
    viewerUserId,
    viewerIsOwner,
    viewerIsStaff,
    viewerHasControl,
    ownerUserId,
    onTransferControl,
}: WatchPartyParticipantsProps) {
    const [busyUserId, setBusyUserId] = useState<string | null>(null);

    const handleTransfer = async (rowUserId: string, targetUserId: string) => {
        setBusyUserId(rowUserId);
        try {
            await onTransferControl(targetUserId);
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
                    let actionLabel: string | null = null;
                    let actionTarget: string | null = null;
                    const hasOverride = viewerIsOwner || viewerIsStaff;
                    if (isSelf) {
                        if (hasOverride && !p.has_control) {
                            actionLabel = "Reclaim control";
                            actionTarget = viewerUserId;
                        }
                    } else if (p.has_control && hasOverride) {
                        actionLabel = "Reclaim";
                        actionTarget = viewerUserId;
                    } else if (!p.has_control && (viewerHasControl || hasOverride)) {
                        actionLabel = "Pass control";
                        actionTarget = p.user.id;
                    }
                    return (
                        <li key={p.user.id} className={styles.participantPill}>
                            <ProfileLink user={p.user} size="small" />
                            {isOwner && <span className={styles.ownerPill}>owner</span>}
                            {p.has_control && <span className={styles.controlPill}>control</span>}
                            {actionLabel && actionTarget && (
                                <button
                                    type="button"
                                    className={styles.controlToggle}
                                    onClick={() => handleTransfer(p.user.id, actionTarget)}
                                    disabled={busyUserId === p.user.id}
                                >
                                    {actionLabel}
                                </button>
                            )}
                        </li>
                    );
                })}
            </ul>
        </div>
    );
}
