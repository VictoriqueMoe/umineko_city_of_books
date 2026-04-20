import { useState } from "react";
import { useAuth } from "../../hooks/useAuth";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { useTheme } from "../../hooks/useTheme";
import { useHuntState } from "../../hooks/useHuntState";
import { Toast } from "../../components/Toast/Toast";
import styles from "./PieceTrigger.module.css";

interface PieceTriggerProps {
    pieceId: string;
    ariaLabel?: string;
}

export function PieceTrigger({ pieceId, ariaLabel }: PieceTriggerProps) {
    const { user } = useAuth();
    const { hasSecret } = useTheme();
    const siteInfo = useSiteInfo();
    const parent = siteInfo.listed_secrets?.find(s => s.pieces.some(p => p.id === pieceId));
    const state = useHuntState(parent?.id ?? "");
    const [justFound, setJustFound] = useState(false);
    const [completed, setCompleted] = useState(false);
    const [collecting, setCollecting] = useState(false);

    if (!user) {
        return null;
    }
    if (!parent || parent.solved) {
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
        const result = await state.collectPiece(pieceId);
        if (result === "new") {
            if (state.collectedCount + 1 === state.totalPieces) {
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
                    Uu~ a piece of the {parent.title}.
                </Toast>
            )}
            {completed && (
                <Toast variant="arcane" duration={12000} onDismiss={() => setCompleted(false)}>
                    Uu~ all {state.totalPieces} pieces of {parent.title} are yours. Read the riddle again, then open the
                    trophy on your profile to whisper the answer.
                </Toast>
            )}
        </>
    );
}
