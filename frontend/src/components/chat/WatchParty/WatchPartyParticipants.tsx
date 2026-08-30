import { useState } from "react";
import type { SiteRole, WatchPartyParticipant } from "../../../types/api";
import { watchPartyControlContext, watchPartyRowControls } from "../../../domain/watchParty/control";
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

    const viewer = { userId: viewerUserId, role: viewerRole, hasControl: viewerHasControl };
    const context = watchPartyControlContext(participants, viewer, ownerUserId);

    const runAction = async (rowUserId: string, action: () => Promise<void>) => {
        setBusyUserId(rowUserId);
        try {
            await action();
        } catch {
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
                    const isOwner = p.user.id === ownerUserId;
                    const { transferLabel, transferTarget, canKick } = watchPartyRowControls(
                        p,
                        viewer,
                        ownerUserId,
                        context,
                    );

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
