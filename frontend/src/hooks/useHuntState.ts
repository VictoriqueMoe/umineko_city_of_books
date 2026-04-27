import { useCallback, useMemo } from "react";
import type { SiteInfoSecret } from "../api/endpoints";
import { useUnlockSecret } from "../api/mutations/secret";
import { useTheme } from "./useTheme.ts";
import { useAuth } from "./useAuth.ts";
import { useSiteInfo } from "./useSiteInfo";

export interface HuntState {
    secret: SiteInfoSecret | null;
    collectedPieces: Set<string>;
    collectedCount: number;
    totalPieces: number;
    allPiecesCollected: boolean;
    solved: boolean;
    closed: boolean;
    collectPiece: (pieceId: string) => Promise<"new" | "already" | "error" | "closed">;
    attemptAnswer: (phrase: string) => Promise<boolean>;
}

function piecePhrase(pieceId: string, letter: string): string {
    return `${pieceId}_${letter.toLowerCase()}`;
}

export function useHuntState(secretId: string): HuntState {
    const { hasSecret, addSecret } = useTheme();
    const { user } = useAuth();
    const siteInfo = useSiteInfo();
    const unlockSecretMutation = useUnlockSecret();
    const unlockMutate = unlockSecretMutation.mutateAsync;

    const secret = useMemo(
        () => siteInfo.listed_secrets?.find(s => s.id === secretId) ?? null,
        [siteInfo.listed_secrets, secretId],
    );

    const collectedPieces = useMemo(() => {
        const set = new Set<string>();
        if (!secret) {
            return set;
        }
        for (const piece of secret.pieces) {
            if (hasSecret(piece.id)) {
                set.add(piece.id);
            }
        }
        return set;
    }, [hasSecret, secret]);

    const collectedCount = collectedPieces.size;
    const totalPieces = secret?.pieces.length ?? 0;
    const allPiecesCollected = totalPieces > 0 && collectedCount === totalPieces;
    const solved = hasSecret(secretId);
    const closed = !solved && (secret?.solved ?? false);

    const collectPiece = useCallback(
        async (pieceId: string): Promise<"new" | "already" | "error" | "closed"> => {
            if (!user || !secret) {
                return "error";
            }
            if (secret.solved) {
                return "closed";
            }
            if (hasSecret(pieceId)) {
                return "already";
            }
            const piece = secret.pieces.find(p => p.id === pieceId);
            if (!piece || !piece.letter) {
                return "error";
            }
            try {
                await unlockMutate({ id: pieceId, phrase: piecePhrase(pieceId, piece.letter) });
                addSecret(pieceId);
                return "new";
            } catch {
                return "error";
            }
        },
        [user, hasSecret, addSecret, secret, unlockMutate],
    );

    const attemptAnswer = useCallback(
        async (phrase: string): Promise<boolean> => {
            if (!secret || secret.solved) {
                return false;
            }
            try {
                await unlockMutate({ id: secretId, phrase });
                addSecret(secretId);
                return true;
            } catch {
                return false;
            }
        },
        [addSecret, secret, secretId, unlockMutate],
    );

    return {
        secret,
        collectedPieces,
        collectedCount,
        totalPieces,
        allPiecesCollected,
        solved,
        closed,
        collectPiece,
        attemptAnswer,
    };
}
