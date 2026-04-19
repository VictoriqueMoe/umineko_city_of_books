import { unlockSecret } from "../../api/endpoints";
import { FINAL_SECRET_ID, PIECE_BY_ID } from "./config";

export async function submitPiece(pieceId: string): Promise<void> {
    const piece = PIECE_BY_ID.get(pieceId);
    if (!piece) {
        throw new Error(`unknown piece: ${pieceId}`);
    }
    await unlockSecret(piece.id, piece.phrase);
}

export async function submitAnswer(phrase: string): Promise<boolean> {
    try {
        await unlockSecret(FINAL_SECRET_ID, phrase);
        return true;
    } catch {
        return false;
    }
}
