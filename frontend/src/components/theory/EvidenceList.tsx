import type {EvidenceItem} from "../../types/api";
import {useResolveQuotes} from "../../hooks/useResolveQuotes";
import {TruthChip} from "../truth/TruthChip";

interface EvidenceListProps {
    evidence: EvidenceItem[];
}

export function EvidenceList({ evidence }: EvidenceListProps) {
    const quotes = useResolveQuotes(evidence);

    if (evidence.length === 0) {
        return null;
    }

    return (
        <div className="theory-evidence-section">
            <h4 className="section-title">Evidence</h4>
            {evidence.map(ev => {
                const key = ev.audio_id ? `audio:${ev.audio_id}` : `index:${ev.quote_index}`;
                const quote = quotes.get(key);
                if (quote) {
                    return <TruthChip key={ev.id} quote={quote} note={ev.note} />;
                }
                return (
                    <div key={ev.id} className="truth-chip">
                        <div className="truth-chip-text">Loading quote...</div>
                    </div>
                );
            })}
        </div>
    );
}
