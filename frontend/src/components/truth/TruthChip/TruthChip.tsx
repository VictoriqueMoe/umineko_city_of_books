import type { Quote } from "../../../types/api";
import styles from "./TruthChip.module.css";

interface TruthChipProps {
    quote: Quote;
    note?: string;
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

export function TruthChip({ quote, note, onRemove }: TruthChipProps) {
    const excerpt = quote.text.length > 100 ? quote.text.slice(0, 100) + "..." : quote.text;

    return (
        <div className={`${styles.chip} ${chipClass(quote)}`}>
            <div className={styles.text}>{excerpt}</div>
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
