import { useEffect } from "react";
import type { CharacterDef } from "../../../games/minesweeper/types";
import { MinesweeperLightningCanvas } from "./MinesweeperLightningCanvas";
import styles from "./MinesweeperVsIntro.module.css";

interface MinesweeperVsIntroProps {
    myCharacter: CharacterDef | undefined;
    opponentCharacter: CharacterDef | undefined;
    onDone: () => void;
    durationMs?: number;
}

export function MinesweeperVsIntro({ myCharacter, opponentCharacter, onDone, durationMs }: MinesweeperVsIntroProps) {
    const duration = durationMs ?? 2400;

    useEffect(() => {
        const timer = setTimeout(() => {
            onDone();
        }, duration);
        return () => clearTimeout(timer);
    }, [duration, onDone]);

    return (
        <div className={styles.overlay}>
            <MinesweeperLightningCanvas active />
            <div className={styles.row}>
                <div className={`${styles.side} ${styles.left}`}>
                    {myCharacter && <img src={myCharacter.image} alt={myCharacter.name} className={styles.portrait} />}
                    <span className={styles.name}>{myCharacter?.name ?? ""}</span>
                </div>
                <div className={styles.vs}>VS</div>
                <div className={`${styles.side} ${styles.right}`}>
                    {opponentCharacter && (
                        <img src={opponentCharacter.image} alt={opponentCharacter.name} className={styles.portrait} />
                    )}
                    <span className={styles.name}>{opponentCharacter?.name ?? ""}</span>
                </div>
            </div>
        </div>
    );
}
