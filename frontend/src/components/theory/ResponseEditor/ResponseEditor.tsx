import React, { useState } from "react";
import { createResponse, type Series } from "../../../api/endpoints";
import { useEvidence } from "../../../hooks/useEvidence";
import { getSeriesConfig } from "../../../utils/seriesConfig";
import { Button } from "../../Button/Button";
import { Input } from "../../Input/Input";
import { TextArea } from "../../TextArea/TextArea";
import { TruthPicker } from "../../truth/TruthPicker/TruthPicker";
import { TruthChip } from "../../truth/TruthChip/TruthChip";
import styles from "./ResponseEditor.module.css";

interface ResponseEditorProps {
    theoryId: string;
    parentId?: string;
    inheritedSide?: "with_love" | "without_love";
    onCreated: () => void;
    series?: Series;
}

export function ResponseEditor({
    theoryId,
    parentId,
    inheritedSide,
    onCreated,
    series = "umineko",
}: ResponseEditorProps) {
    const cfg = getSeriesConfig(series);
    const [side, setSide] = useState<"with_love" | "without_love" | null>(inheritedSide ?? null);
    const [body, setBody] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const ev = useEvidence(undefined, series);
    const isReply = parentId !== undefined;

    async function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault();
        if (!side || !body.trim() || submitting) {
            return;
        }

        setError("");
        setSubmitting(true);
        try {
            await createResponse(theoryId, { parent_id: parentId, side, body: body.trim(), evidence: ev.toInput() });
            setBody("");
            if (!isReply) {
                setSide(null);
            }
            ev.clear();
            onCreated();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to submit response.");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <>
            <form className={styles.editor} onSubmit={handleSubmit}>
                <h4 className={styles.title}>{isReply ? "Reply" : "Add your response"}</h4>

                {error && <div className={styles.error}>{error}</div>}

                {!isReply && (
                    <div className={styles.sideSelector}>
                        <button
                            type="button"
                            className={`${styles.sideBtn} ${styles.sideBtnWithLove}${side === "with_love" ? ` ${styles.sideBtnWithLoveActive}` : ""}`}
                            onClick={() => setSide("with_love")}
                        >
                            <span className={styles.sideBtnTitle}>{cfg.withLoveTitle}</span>
                            <span className={styles.sideBtnSubtitle}>{cfg.withLoveSubtitle}</span>
                        </button>
                        <button
                            type="button"
                            className={`${styles.sideBtn} ${styles.sideBtnWithoutLove}${side === "without_love" ? ` ${styles.sideBtnWithoutLoveActive}` : ""}`}
                            onClick={() => setSide("without_love")}
                        >
                            <span className={styles.sideBtnTitle}>{cfg.withoutLoveTitle}</span>
                            <span className={styles.sideBtnSubtitle}>{cfg.withoutLoveSubtitle}</span>
                        </button>
                    </div>
                )}

                <TextArea
                    placeholder={isReply ? "Write your reply..." : "State your argument..."}
                    value={body}
                    onChange={e => setBody(e.target.value)}
                />

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
                    <Button variant="primary" type="submit" disabled={!side || !body.trim() || submitting}>
                        {submitting ? "Submitting..." : isReply ? "Reply" : "Submit Response"}
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
