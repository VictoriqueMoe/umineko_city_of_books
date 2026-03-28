import type { Quote } from "../../types/api";

interface TruthCardProps {
    quote: Quote;
    onClick?: () => void;
    selected?: boolean;
}

function cardClass(quote: Quote): string {
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

export function TruthCard({ quote, onClick, selected }: TruthCardProps) {
    return (
        <div
            className={`truth-card ${cardClass(quote)}${selected ? " selected" : ""}`}
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
            <div className="truth-text" dangerouslySetInnerHTML={{ __html: quote.textHtml }} />
            <div className="truth-meta">
                <span className="truth-speaker">{quote.character}</span>
                <span className="truth-episode">Episode {quote.episode}</span>
            </div>
        </div>
    );
}
