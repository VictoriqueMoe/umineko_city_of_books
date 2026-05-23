import { useState } from "react";
import type { CSSProperties } from "react";
import { CHARACTERS } from "../../../games/minesweeper/characters";
import type { MinesweeperState } from "../../../types/api";
import type { CharacterId } from "../../../games/minesweeper/types";
import styles from "./MinesweeperCharacterSelect.module.css";

interface MinesweeperCharacterSelectProps {
    state: MinesweeperState;
    mySlot: number | null;
    isSpectator: boolean;
    submitting: boolean;
    onSelect: (character: CharacterId) => Promise<void>;
}

export function MinesweeperCharacterSelect({
    state,
    mySlot,
    isSpectator,
    submitting,
    onSelect,
}: MinesweeperCharacterSelectProps) {
    const [pending, setPending] = useState<CharacterId | null>(null);
    const opponentSlot = mySlot === null ? null : 1 - mySlot;
    const myPick = mySlot !== null ? state.characters[mySlot] : "";
    const oppPick = opponentSlot !== null ? state.characters[opponentSlot] : "";

    async function pick(character: CharacterId) {
        if (submitting || isSpectator || mySlot === null) {
            return;
        }
        setPending(character);
        try {
            await onSelect(character);
        } finally {
            setPending(null);
        }
    }

    let status = "Choose a witch to fight as.";
    if (isSpectator) {
        status = "Spectating the council.";
    } else if (myPick && oppPick) {
        status = "Both have committed. The board awakens.";
    } else if (myPick) {
        status = "Your witch is chosen. Awaiting your opponent.";
    }

    return (
        <div className={styles.altar}>
            <div className={styles.headingRow}>
                <span className={styles.flourish} aria-hidden="true">
                    ✦
                </span>
                <h3 className={styles.title}>Witches&apos; Council</h3>
                <span className={styles.flourish} aria-hidden="true">
                    ✦
                </span>
            </div>
            <p className={styles.hint}>{status}</p>
            <div className={styles.lineup}>
                {CHARACTERS.map((c, i) => {
                    const isMine = myPick === c.id;
                    const isOpp = oppPick === c.id;
                    const isPending = pending === c.id;
                    const classes = [styles.figure];
                    if (isMine) {
                        classes.push(styles.chosen);
                    }
                    if (isOpp) {
                        classes.push(styles.opposed);
                    }
                    if (isPending) {
                        classes.push(styles.pending);
                    }
                    return (
                        <button
                            type="button"
                            key={c.id}
                            className={classes.join(" ")}
                            disabled={isSpectator || submitting || pending !== null || isOpp}
                            onClick={() => void pick(c.id)}
                            aria-pressed={isMine}
                            style={{ "--i": i } as CSSProperties}
                        >
                            <span className={styles.halo} aria-hidden="true" />
                            <span className={`${styles.corner} ${styles.cornerTL}`} aria-hidden="true" />
                            <span className={`${styles.corner} ${styles.cornerTR}`} aria-hidden="true" />
                            <span className={`${styles.corner} ${styles.cornerBL}`} aria-hidden="true" />
                            <span className={`${styles.corner} ${styles.cornerBR}`} aria-hidden="true" />
                            <div className={styles.portraitWrap}>
                                <img src={c.image} alt={c.name} className={styles.portrait} />
                                {isOpp && <span className={styles.veil} aria-hidden="true" />}
                            </div>
                            <div className={styles.nameplate}>
                                <span className={styles.name}>{c.name}</span>
                                {isMine && <span className={styles.tagChosen}>· Chosen ·</span>}
                                {isOpp && <span className={styles.tagOpposed}>Opposed</span>}
                            </div>
                        </button>
                    );
                })}
            </div>
        </div>
    );
}
