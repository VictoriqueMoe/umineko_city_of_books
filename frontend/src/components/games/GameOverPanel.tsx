import type { PropsWithChildren } from "react";
import type { ResultTone } from "./gameRoomHelpers";
import styles from "./GameOverPanel.module.css";

const TONE_CLASSES: Record<ResultTone, string> = {
    win: styles.resultWin,
    loss: styles.resultLoss,
    draw: styles.resultDraw,
    neutral: "",
};

type GameOverPanelProps = PropsWithChildren<{
    isOver: boolean;
    showChildren: boolean;
    resultText: string;
    resultTone: ResultTone;
    reasonText?: string;
}>;

export function GameOverPanel({
    isOver,
    showChildren,
    resultText,
    resultTone,
    reasonText,
    children,
}: GameOverPanelProps) {
    if (!isOver && !showChildren) {
        return null;
    }
    const toneClass = TONE_CLASSES[resultTone];
    return (
        <div className={styles.gameOver}>
            {isOver && (
                <div className={styles.result}>
                    <span className={toneClass}>{resultText}</span>
                    {reasonText && <span className={styles.resultReason}> {reasonText}</span>}
                </div>
            )}
            {showChildren && children}
        </div>
    );
}
