import { useState } from "react";
import { useAuth } from "../../hooks/useAuth";
import { useEpitaphState } from "./useEpitaphState";
import { EpitaphPanel } from "./EpitaphPanel";
import { PIECES } from "./config";
import styles from "./EpitaphIcon.module.css";

interface EpitaphIconProps {
    profileUserId: string;
    profileSecrets?: string[];
}

export function EpitaphIcon({ profileUserId, profileSecrets }: EpitaphIconProps) {
    const { user } = useAuth();
    const state = useEpitaphState();
    const [open, setOpen] = useState(false);

    const isOwner = !!user && user.id === profileUserId;
    const otherSolved = !isOwner && (profileSecrets?.includes("witchHunter") ?? false);

    if (!isOwner && !otherSolved) {
        return null;
    }
    if (isOwner && state.collectedCount === 0 && !state.solved) {
        return null;
    }

    const solved = isOwner ? state.solved : otherSolved;
    const count = isOwner ? state.collectedCount : PIECES.length;

    return (
        <>
            <button
                type="button"
                className={`${styles.icon}${solved ? ` ${styles.iconSolved}` : ""}`}
                onClick={() => setOpen(true)}
                aria-label={solved ? "The Witch's Epitaph (solved)" : `Epitaph: ${count} of ${PIECES.length} pieces`}
                title={solved ? "The Witch's Epitaph" : `${count} / ${PIECES.length}`}
            >
                {"\u273F"}
                {!solved && isOwner && count < PIECES.length && <span className={styles.badge}>{count}</span>}
            </button>
            <EpitaphPanel isOpen={open} onClose={() => setOpen(false)} />
        </>
    );
}
