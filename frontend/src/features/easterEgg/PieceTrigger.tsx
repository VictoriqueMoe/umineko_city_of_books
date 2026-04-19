import { useState } from "react";
import { useAuth } from "../../hooks/useAuth";
import { useTheme } from "../../hooks/useTheme";
import { Toast } from "../../components/Toast/Toast";
import { useEpitaphState } from "./useEpitaphState";
import { PIECE_BY_ID } from "./config";
import styles from "./PieceTrigger.module.css";

interface PieceTriggerProps {
    pieceId: string;
    ariaLabel?: string;
}

export function PieceTrigger({ pieceId, ariaLabel }: PieceTriggerProps) {
    const { user } = useAuth();
    const { hasSecret } = useTheme();
    const { collectPiece } = useEpitaphState();
    const [justFound, setJustFound] = useState(false);
    const [collecting, setCollecting] = useState(false);

    if (!user) {
        return null;
    }
    if (!PIECE_BY_ID.has(pieceId)) {
        return null;
    }
    if (hasSecret(pieceId)) {
        return null;
    }

    async function handleClick() {
        if (collecting) {
            return;
        }
        setCollecting(true);
        const result = await collectPiece(pieceId);
        if (result === "new") {
            setJustFound(true);
        }
        setCollecting(false);
    }

    return (
        <>
            <button
                type="button"
                className={styles.trigger}
                onClick={handleClick}
                disabled={collecting}
                aria-label={ariaLabel ?? "A curious sparkle"}
                title=""
            >
                {"\u2726"}
            </button>
            {justFound && (
                <Toast variant="arcane" duration={4200} onDismiss={() => setJustFound(false)}>
                    Uu~ a piece of the epitaph.
                </Toast>
            )}
        </>
    );
}
