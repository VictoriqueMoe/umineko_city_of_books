import { useState } from "react";
import type { SiteRole, WatchPartyParticipant } from "../../../types/api";
import { watchPartyControlContext, watchPartyRowControls } from "../../../domain/watchParty/control";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import styles from "./WatchParty.module.css";

type WatchPartyParticipantsLayout = "strip" | "list";

interface WatchPartyParticipantsProps {
    participants: WatchPartyParticipant[];
    viewerUserId: string;
    viewerRole: SiteRole | undefined;
    viewerHasControl: boolean;
    ownerUserId: string;
    onTransferControl: (userId: string) => Promise<void>;
    onKick: (userId: string) => Promise<void>;
    layout?: WatchPartyParticipantsLayout;
}

export function WatchPartyParticipants({
    participants,
    viewerUserId,
    viewerRole,
    viewerHasControl,
    ownerUserId,
    onTransferControl,
    onKick,
    layout = "strip",
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

    const isList = layout === "list";
    const rootClass = isList ? styles.participantList : styles.participantStrip;
    const labelClass = isList ? styles.participantListLabel : styles.participantStripLabel;
    const listClass = isList ? styles.participantListItems : styles.participantStripList;
    const rowClass = isList ? styles.participantRow : styles.participantPill;
    const mainClass = isList ? styles.participantRowMain : styles.participantPillMain;
    const transferClass = isList ? `${styles.controlToggle} ${styles.rowAction}` : styles.controlToggle;
    const kickClass = isList ? `${styles.kickToggle} ${styles.rowAction}` : styles.kickToggle;

    return (
        <div className={rootClass}>
            <span className={labelClass}>
                {participants.length} {participants.length === 1 ? "watcher" : "watchers"}
            </span>
            <ul className={listClass}>
                {participants.map(p => {
                    const isOwner = p.user.id === ownerUserId;
                    const { transferLabel, transferTarget, canKick } = watchPartyRowControls(
                        p,
                        viewer,
                        ownerUserId,
                        context,
                    );

                    return (
                        <li key={p.user.id} className={rowClass}>
                            <span className={mainClass}>
                                <ProfileLink user={p.user} size="small" />
                            </span>
                            {isOwner && <span className={styles.ownerPill}>owner</span>}
                            {p.has_control && <span className={styles.controlPill}>control</span>}
                            {transferLabel && transferTarget && (
                                <button
                                    type="button"
                                    className={transferClass}
                                    onClick={() => runAction(p.user.id, () => onTransferControl(transferTarget))}
                                    disabled={busyUserId === p.user.id}
                                >
                                    {transferLabel}
                                </button>
                            )}
                            {canKick && (
                                <button
                                    type="button"
                                    className={kickClass}
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
