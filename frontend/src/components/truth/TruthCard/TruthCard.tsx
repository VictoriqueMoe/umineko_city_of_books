import type {Quote} from "../../../types/api";
import styles from "./TruthCard.module.css";

interface TruthCardProps {
    quote: Quote;
    onClick?: () => void;
    selected?: boolean;
}

function cardClass(quote: Quote): string {
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

export function TruthCard({ quote, onClick, selected }: TruthCardProps) {
    return (
        <div
            className={`${styles.card} ${cardClass(quote)}${selected ? ` ${styles.selected}` : ""}`}
            onClick={onClick}
            role={onClick ? "button" : undefined}
            tabIndex={onClick ? 0 : undefined}
            onKeyDown={e => {
                if (onClick && (e.key === "Enter" || e.key === " ")) {
                    e.preventDefault();
                    onClick();
                }
            }}
        >
            <div className={styles.text} dangerouslySetInnerHTML={{ __html: quote.textHtml }} />
            <div className={styles.meta}>
                <span className={styles.speaker}>{quote.character}</span>
                <span className={styles.episode}>Episode {quote.episode}</span>
            </div>
        </div>
    );
}
