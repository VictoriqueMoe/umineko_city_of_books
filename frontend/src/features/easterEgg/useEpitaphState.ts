import { useCallback, useMemo } from "react";
import { useTheme } from "../../hooks/useTheme";
import { useAuth } from "../../hooks/useAuth";
import { submitAnswer, submitPiece } from "./api";
import { FINAL_SECRET_ID, PIECES } from "./config";

interface EpitaphState {
    collectedPieces: Set<string>;
    collectedCount: number;
    allPiecesCollected: boolean;
    solved: boolean;
    collectPiece: (pieceId: string) => Promise<"new" | "already" | "error">;
    attemptAnswer: (phrase: string) => Promise<boolean>;
}

export function useEpitaphState(): EpitaphState {
    const { hasSecret, addSecret } = useTheme();
    const { user } = useAuth();

    const collectedPieces = useMemo(() => {
        const set = new Set<string>();
        for (let i = 0; i < PIECES.length; i++) {
            const piece = PIECES[i];
            if (hasSecret(piece.id)) {
                set.add(piece.id);
            }
        }
        return set;
    }, [hasSecret]);

    const collectedCount = collectedPieces.size;
    const allPiecesCollected = collectedCount === PIECES.length;
    const solved = hasSecret(FINAL_SECRET_ID);

    const collectPiece = useCallback(
        async (pieceId: string): Promise<"new" | "already" | "error"> => {
            if (!user) {
                return "error";
            }
            if (hasSecret(pieceId)) {
                return "already";
            }
            try {
                await submitPiece(pieceId);
                addSecret(pieceId);
                return "new";
            } catch {
                return "error";
            }
        },
        [user, hasSecret, addSecret],
    );

    const attemptAnswer = useCallback(
        async (phrase: string): Promise<boolean> => {
            const ok = await submitAnswer(phrase);
            if (ok) {
                addSecret(FINAL_SECRET_ID);
            }
            return ok;
        },
        [addSecret],
    );

    return {
        collectedPieces,
        collectedCount,
        allPiecesCollected,
        solved,
        collectPiece,
        attemptAnswer,
    };
}
