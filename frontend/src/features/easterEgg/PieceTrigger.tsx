import { useState } from "react";
import { useAuth } from "../../hooks/useAuth";
import { useTheme } from "../../hooks/useTheme";
import { Toast } from "../../components/Toast/Toast";
import { useEpitaphState } from "../../hooks/useEpitaphState.ts";
import { PIECE_BY_ID, PIECES } from "./config";
import styles from "./PieceTrigger.module.css";

interface PieceTriggerProps {
    pieceId: string;
    ariaLabel?: string;
}

export function PieceTrigger({ pieceId, ariaLabel }: PieceTriggerProps) {
    const { user } = useAuth();
    const { hasSecret } = useTheme();
    const { collectPiece, collectedCount } = useEpitaphState();
    const [justFound, setJustFound] = useState(false);
    const [completed, setCompleted] = useState(false);
    const [collecting, setCollecting] = useState(false);

    if (!user) {
        return null;
    }
    if (!PIECE_BY_ID.has(pieceId)) {
        return null;
    }
    const alreadyCollected = hasSecret(pieceId);
    if (alreadyCollected && !justFound && !completed) {
        return null;
    }

    async function handleClick() {
        if (collecting) {
            return;
        }
        setCollecting(true);
        const result = await collectPiece(pieceId);
        if (result === "new") {
            if (collectedCount + 1 === PIECES.length) {
                setCompleted(true);
            } else {
                setJustFound(true);
            }
        }
        setCollecting(false);
    }

    return (
        <>
            {!alreadyCollected && (
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
            )}
            {justFound && (
                <Toast variant="arcane" duration={4200} onDismiss={() => setJustFound(false)}>
                    Uu~ a piece of the epitaph.
                </Toast>
            )}
            {completed && (
                <Toast variant="arcane" duration={12000} onDismiss={() => setCompleted(false)}>
                    Uu~ all twelve pieces are yours. Read mama&apos;s writing again, follow her count, then open the
                    rose on your own profile and whisper the witch&apos;s truth.
                </Toast>
            )}
        </>
    );
}
