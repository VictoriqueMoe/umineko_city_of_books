import { useState } from "react";
import { Modal } from "../../components/Modal/Modal";
import { Input } from "../../components/Input/Input";
import { Button } from "../../components/Button/Button";
import { useHuntState } from "../../hooks/useHuntState";
import styles from "./HuntPanel.module.css";

interface HuntPanelProps {
    secretId: string;
    isOpen: boolean;
    onClose: () => void;
}

export function HuntPanel({ secretId, isOpen, onClose }: HuntPanelProps) {
    const state = useHuntState(secretId);
    const [input, setInput] = useState("");
    const [shake, setShake] = useState(false);
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);

    const secret = state.secret;

    async function handleSubmit(e: React.FormEvent) {
        e.preventDefault();
        if (!state.allPiecesCollected || submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        const ok = await state.attemptAnswer(input.trim().toLowerCase());
        if (!ok) {
            setShake(true);
            setError("Not quite. Read the riddle again.");
            window.setTimeout(() => setShake(false), 360);
        } else {
            setInput("");
        }
        setSubmitting(false);
    }

    if (!secret) {
        return null;
    }

    const tiles = Array.from({ length: secret.pieces.length }, (_, i) => {
        const tileNumber = i + 1;
        const piece = secret.pieces.find(p => p.tile === tileNumber);
        const letter = piece?.letter ?? "";
        const found = piece && state.collectedPieces.has(piece.id);
        return { tileNumber, letter, found: Boolean(found) };
    });

    return (
        <Modal isOpen={isOpen} onClose={onClose} title={secret.title}>
            <div className={styles.body}>
                {tiles.length > 0 && (
                    <div className={styles.tiles}>
                        {tiles.map(t => (
                            <span
                                key={t.tileNumber}
                                className={`${styles.tile}${t.found ? ` ${styles.tileFilled}` : ` ${styles.tileEmpty}`}`}
                                aria-label={
                                    t.found ? `Tile ${t.tileNumber}: ${t.letter}` : `Tile ${t.tileNumber}: empty`
                                }
                            >
                                {t.found ? t.letter : "\u00b7"}
                            </span>
                        ))}
                    </div>
                )}

                {state.solved ? (
                    <div className={styles.success}>
                        {secret.solved_message || "You solved the hunt. The reward has been added to your profile."}
                    </div>
                ) : state.closed ? (
                    <div className={styles.success}>
                        Someone else whispered the answer before you. The hunt is closed. Your {state.collectedCount} /{" "}
                        {state.totalPieces} pieces stay with you as a keepsake.
                    </div>
                ) : (
                    <>
                        {secret.pointer && <div className={styles.pointer}>{secret.pointer}</div>}

                        <form onSubmit={handleSubmit} className={shake ? styles.shake : undefined}>
                            <div className={styles.inputRow}>
                                <Input
                                    type="text"
                                    fullWidth
                                    placeholder={
                                        state.allPiecesCollected
                                            ? (secret.ready_placeholder ?? "Whisper the answer...")
                                            : `${state.collectedCount} / ${state.totalPieces} pieces found`
                                    }
                                    value={input}
                                    onChange={e => setInput(e.target.value)}
                                    disabled={!state.allPiecesCollected || submitting}
                                />
                                <Button
                                    type="submit"
                                    variant="primary"
                                    disabled={!state.allPiecesCollected || submitting || !input.trim()}
                                >
                                    Declare
                                </Button>
                            </div>
                            <div className={styles.error}>{error}</div>
                        </form>

                        {!state.allPiecesCollected && secret.pending_hint && (
                            <div className={styles.hint}>{secret.pending_hint}</div>
                        )}
                    </>
                )}
            </div>
        </Modal>
    );
}
