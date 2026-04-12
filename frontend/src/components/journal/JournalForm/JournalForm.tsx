import { useState } from "react";
import type { JournalWork } from "../../../types/api";
import { Input } from "../../Input/Input";
import { Select } from "../../Select/Select";
import { Button } from "../../Button/Button";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import { JOURNAL_WORKS } from "../../../utils/journalWorks";
import styles from "./JournalForm.module.css";

interface JournalFormProps {
    initialTitle?: string;
    initialBody?: string;
    initialWork?: JournalWork;
    submitLabel: string;
    submittingLabel: string;
    onSubmit: (data: { title: string; body: string; work: JournalWork }) => Promise<void>;
}

export function JournalForm({
    initialTitle = "",
    initialBody = "",
    initialWork = "general",
    submitLabel,
    submittingLabel,
    onSubmit,
}: JournalFormProps) {
    const [title, setTitle] = useState(initialTitle);
    const [body, setBody] = useState(initialBody);
    const [work, setWork] = useState<JournalWork>(initialWork);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    async function handleSubmit(e: React.FormEvent) {
        e.preventDefault();
        if (!title.trim() || !body.trim() || submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            await onSubmit({ title: title.trim(), body: body.trim(), work });
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to save");
            setSubmitting(false);
        }
    }

    return (
        <form className={styles.form} onSubmit={handleSubmit}>
            {error && <div className={styles.error}>{error}</div>}

            <div className={styles.field}>
                <label className={styles.label}>Title</label>
                <Input
                    type="text"
                    value={title}
                    onChange={e => setTitle(e.target.value)}
                    placeholder="e.g. My first Umineko read-through"
                    maxLength={200}
                />
            </div>

            <div className={styles.field}>
                <label className={styles.label}>Work</label>
                <Select value={work} onChange={e => setWork((e.target as HTMLSelectElement).value as JournalWork)}>
                    {JOURNAL_WORKS.map(w => (
                        <option key={w.id} value={w.id}>
                            {w.label}
                        </option>
                    ))}
                </Select>
            </div>

            <div className={styles.field}>
                <label className={styles.label}>Intro</label>
                <MentionTextArea
                    value={body}
                    onChange={setBody}
                    rows={8}
                    placeholder="Introduce your read-through. What are you reading, what are you hoping for? You can post updates as comments below once you've created the journal."
                />
            </div>

            <div className={styles.actions}>
                <Button variant="primary" size="medium" disabled={submitting || !title.trim() || !body.trim()}>
                    {submitting ? submittingLabel : submitLabel}
                </Button>
            </div>
        </form>
    );
}
