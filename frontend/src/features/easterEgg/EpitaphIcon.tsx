import { useState } from "react";
import { useAuth } from "../../hooks/useAuth";
import { useEpitaphState } from "../../hooks/useEpitaphState.ts";
import { EpitaphPanel } from "./EpitaphPanel";
import { PIECES } from "./config";
import styles from "./EpitaphIcon.module.css";

interface EpitaphIconProps {
    profileUserId: string;
}

export function EpitaphIcon({ profileUserId }: EpitaphIconProps) {
    const { user } = useAuth();
    const state = useEpitaphState();
    const [open, setOpen] = useState(false);

    const isOwner = !!user && user.id === profileUserId;

    if (!isOwner) {
        return null;
    }
    if (state.collectedCount === 0 && !open) {
        return null;
    }
    if (state.solved && !open) {
        return null;
    }

    const count = state.collectedCount;
    const showButton = !state.solved && state.collectedCount > 0;

    return (
        <>
            {showButton && (
                <button
                    type="button"
                    className={styles.icon}
                    onClick={() => setOpen(true)}
                    aria-label={`Epitaph: ${count} of ${PIECES.length} pieces`}
                    title={`${count} / ${PIECES.length}`}
                >
                    {"\u273F"}
                    {count < PIECES.length && <span className={styles.badge}>{count}</span>}
                </button>
            )}
            <EpitaphPanel isOpen={open} onClose={() => setOpen(false)} />
        </>
    );
}
