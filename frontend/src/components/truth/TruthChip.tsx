import type {Quote} from "../../types/api";

interface TruthChipProps {
    quote: Quote;
    note?: string;
    onRemove?: () => void;
}

function chipClass(quote: Quote): string {
    if (quote.hasRedTruth) {
        return "red";
    }
    if (quote.hasBlueTruth) {
        return "blue";
    }
    if (quote.hasGoldTruth) {
        return "gold";
    }
    if (quote.hasPurpleTruth) {
        return "purple";
    }
    return "";
}

export function TruthChip({ quote, note, onRemove }: TruthChipProps) {
    const excerpt = quote.text.length > 100 ? quote.text.slice(0, 100) + "..." : quote.text;

    return (
        <div className={`truth-chip ${chipClass(quote)}`}>
            <div className="truth-chip-text">{excerpt}</div>
            <div className="truth-chip-meta">
                <span className="truth-chip-speaker">{quote.character}</span>
                <span className="truth-chip-episode">EP{quote.episode}</span>
            </div>
            {note && <div className="truth-chip-note">{note}</div>}
            {onRemove && (
                <button className="truth-chip-remove" onClick={onRemove}>
                    {"\u2715"}
                </button>
            )}
        </div>
    );
}
