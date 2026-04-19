import { useState } from "react";
import { Modal } from "../../components/Modal/Modal";
import { Input } from "../../components/Input/Input";
import { Button } from "../../components/Button/Button";
import { useEpitaphState } from "./useEpitaphState";
import { EPITAPH_POINTER, JUMBLE, JUMBLE_LENGTH, PIECES } from "./config";
import styles from "./EpitaphPanel.module.css";

interface EpitaphPanelProps {
    isOpen: boolean;
    onClose: () => void;
}

export function EpitaphPanel({ isOpen, onClose }: EpitaphPanelProps) {
    const state = useEpitaphState();
    const [input, setInput] = useState("");
    const [shake, setShake] = useState(false);
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);

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
            setError("Not quite. Read the epitaph again.");
            window.setTimeout(() => setShake(false), 360);
        } else {
            setInput("");
        }
        setSubmitting(false);
    }

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="The Epitaph">
            <div className={styles.body}>
                <div className={styles.tiles}>
                    {Array.from({ length: JUMBLE_LENGTH }, (_, i) => {
                        const tileNumber = i + 1;
                        const piece = PIECES.find(p => p.tile === tileNumber);
                        const found = piece && state.collectedPieces.has(piece.id);
                        return (
                            <span
                                key={tileNumber}
                                className={`${styles.tile}${found ? ` ${styles.tileFilled}` : ` ${styles.tileEmpty}`}`}
                                aria-label={found ? `Tile ${tileNumber}: ${JUMBLE[i]}` : `Tile ${tileNumber}: empty`}
                            >
                                {found ? JUMBLE[i] : "\u00b7"}
                            </span>
                        );
                    })}
                </div>

                {state.solved ? (
                    <div className={styles.success}>
                        Uu~ the Endless Witch has taught you her secret. The Maria theme and the Witch Hunter role are
                        yours.
                    </div>
                ) : (
                    <>
                        <div className={styles.pointer}>{EPITAPH_POINTER}</div>

                        <form onSubmit={handleSubmit} className={shake ? styles.shake : undefined}>
                            <div className={styles.inputRow}>
                                <Input
                                    type="text"
                                    fullWidth
                                    placeholder={
                                        state.allPiecesCollected
                                            ? "Speak the witch's name..."
                                            : `${state.collectedCount} / ${PIECES.length} pieces found`
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

                        {!state.allPiecesCollected && (
                            <div className={styles.hint}>Find all twelve pieces before the witch will hear you.</div>
                        )}
                    </>
                )}
            </div>
        </Modal>
    );
}
