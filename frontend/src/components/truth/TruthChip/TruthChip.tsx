import { useState } from "react";
import type { Quote } from "../../../types/api";
import styles from "./TruthChip.module.css";

interface TruthChipProps {
    quote: Quote;
    note?: string;
    lang?: string;
    onRemove?: () => void;
}

function chipClass(quote: Quote): string {
    if (quote.hasRedTruth) {
        return styles.red;
    }
    if (quote.hasBlueTruth) {
        return styles.blue;
    }
    if (quote.hasGoldTruth) {
        return styles.gold;
    }
    if (quote.hasPurpleTruth) {
        return styles.purple;
    }
    return "";
}

export function TruthChip({ quote, note, lang, onRemove }: TruthChipProps) {
    const [expanded, setExpanded] = useState(false);
    const rawText = lang === "jp" && quote.textJp ? quote.textJp : quote.text;
    const isTruncated = rawText.length > 100;
    const displayText = isTruncated && !expanded ? rawText.slice(0, 100) + "..." : rawText;

    return (
        <div className={`${styles.chip} ${chipClass(quote)}`}>
            <div className={styles.text}>
                {displayText}
                {isTruncated && (
                    <button
                        type="button"
                        className={styles.expandToggle}
                        onClick={e => {
                            e.stopPropagation();
                            setExpanded(!expanded);
                        }}
                    >
                        {expanded ? "show less" : "show more"}
                    </button>
                )}
            </div>
            <div className={styles.meta}>
                <span className={styles.speaker}>{quote.character}</span>
                <span>EP{quote.episode}</span>
            </div>
            {note && <div className={styles.note}>{note}</div>}
            {onRemove && (
                <button className={styles.remove} onClick={onRemove}>
                    {"\u2715"}
                </button>
            )}
        </div>
    );
}
