import React, { useState } from "react";
import { useEvidence } from "../../../hooks/useEvidence";
import { Button } from "../../Button/Button";
import { Input } from "../../Input/Input";
import { TextArea } from "../../TextArea/TextArea";
import { TruthPicker } from "../../truth/TruthPicker/TruthPicker";
import { TruthChip } from "../../truth/TruthChip/TruthChip";
import { Select } from "../../Select/Select";
import type { EvidenceInput, EvidenceItem } from "../../../types/api";
import type { Series } from "../../../api/endpoints";
import { getSeriesConfig } from "../../../utils/seriesConfig";
import styles from "./TheoryForm.module.css";

interface TheoryFormProps {
    initialTitle?: string;
    initialBody?: string;
    initialEpisode?: number;
    initialEvidence?: EvidenceItem[];
    submitLabel: string;
    submittingLabel: string;
    series?: Series;
    onSubmit: (data: { title: string; body: string; episode: number; evidence: EvidenceInput[] }) => Promise<void>;
}

export function TheoryForm({
    initialTitle = "",
    initialBody = "",
    initialEpisode = 0,
    initialEvidence,
    submitLabel,
    submittingLabel,
    series = "umineko",
    onSubmit,
}: TheoryFormProps) {
    const cfg = getSeriesConfig(series);
    const [title, setTitle] = useState(initialTitle);
    const [body, setBody] = useState(initialBody);
    const [episode, setEpisode] = useState(initialEpisode);
    const [submitting, setSubmitting] = useState(false);
    const ev = useEvidence(initialEvidence, series);

    async function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault();
        if (!title.trim() || !body.trim() || submitting) {
            return;
        }

        setSubmitting(true);
        try {
            await onSubmit({
                title: title.trim(),
                body: body.trim(),
                episode,
                evidence: ev.toInput(),
            });
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <>
            <form onSubmit={handleSubmit}>
                <Input
                    type="text"
                    fullWidth
                    placeholder="Theory title..."
                    value={title}
                    onChange={e => setTitle(e.target.value)}
                    maxLength={200}
                />

                <TextArea placeholder="State your theory..." value={body} onChange={e => setBody(e.target.value)} />

                <Select value={episode} onChange={e => setEpisode(Number((e.target as HTMLSelectElement).value))}>
                    <option value={0}>General (no specific episode)</option>
                    {Array.from({ length: cfg.episodeCount }, (_, i) => i + 1).map(ep => (
                        <option key={ep} value={ep}>
                            Episode {ep}
                        </option>
                    ))}
                </Select>

                {ev.evidence.length > 0 && (
                    <div className={styles.evidenceSection}>
                        {ev.evidence.map((item, i) => (
                            <div key={item.quote.audioId} className={styles.evidenceItem}>
                                <TruthChip quote={item.quote} lang={item.lang} onRemove={() => ev.removeAt(i)} />
                                <Input
                                    type="text"
                                    fullWidth
                                    placeholder="Why is this relevant?"
                                    value={item.note}
                                    onChange={e => ev.updateNote(i, e.target.value)}
                                />
                            </div>
                        ))}
                    </div>
                )}

                <div className={styles.actions}>
                    <Button variant="ghost" type="button" onClick={ev.openPicker}>
                        + Attach Evidence
                    </Button>
                    <Button variant="primary" type="submit" disabled={!title.trim() || !body.trim() || submitting}>
                        {submitting ? submittingLabel : submitLabel}
                    </Button>
                </div>
            </form>

            <TruthPicker
                isOpen={ev.pickerOpen}
                onClose={ev.closePicker}
                onSelect={ev.addQuote}
                selectedKeys={ev.selectedKeys}
                series={series}
            />
        </>
    );
}
