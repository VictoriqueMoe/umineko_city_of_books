import { useState } from "react";
import type { SiteInfoSecret } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { useHuntState } from "../../hooks/useHuntState";
import { HuntPanel } from "./HuntPanel";
import styles from "./EpitaphIcon.module.css";

interface HuntIconProps {
    profileUserId: string;
    secret: SiteInfoSecret;
}

export function HuntIcon({ profileUserId, secret }: HuntIconProps) {
    const { user } = useAuth();
    const state = useHuntState(secret.id);
    const [open, setOpen] = useState(false);

    const isOwner = !!user && user.id === profileUserId;

    if (!isOwner) {
        return null;
    }
    if (state.solved) {
        return null;
    }
    if (state.collectedCount === 0) {
        return null;
    }

    const icon = secret.icon || "\u2605";
    const count = state.collectedCount;
    const total = state.totalPieces;

    return (
        <>
            <button
                type="button"
                className={styles.icon}
                onClick={() => setOpen(true)}
                aria-label={`${secret.title}: ${count} of ${total} pieces`}
                title={`${secret.title} - ${count} / ${total}`}
            >
                {icon}
                {count < total && <span className={styles.badge}>{count}</span>}
            </button>
            <HuntPanel secretId={secret.id} isOpen={open} onClose={() => setOpen(false)} />
        </>
    );
}
