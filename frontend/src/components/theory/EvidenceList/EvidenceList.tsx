import type { EvidenceItem } from "../../../types/api";
import type { Series } from "../../../api/endpoints";
import { useResolveQuotes } from "../../../hooks/useResolveQuotes";
import { TruthChip } from "../../truth/TruthChip/TruthChip";
import styles from "./EvidenceList.module.css";

interface EvidenceListProps {
    evidence: EvidenceItem[];
    series?: Series;
}

export function EvidenceList({ evidence, series = "umineko" }: EvidenceListProps) {
    const quotes = useResolveQuotes(evidence, series);

    if (evidence.length === 0) {
        return null;
    }

    return (
        <div className={styles.section}>
            <h4 className={styles.title}>Evidence</h4>
            {evidence.map(ev => {
                const key = ev.audio_id ? `audio:${ev.audio_id}` : `index:${ev.quote_index}`;
                const quote = quotes.get(key);
                if (quote) {
                    return <TruthChip key={ev.id} quote={quote} note={ev.note} lang={ev.lang} />;
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
