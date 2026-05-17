import { type SubmitEvent, useState } from "react";
import type { JournalWork } from "../../../types/api";
import { Input } from "../../Input/Input";
import { Select } from "../../Select/Select";
import { Button } from "../../Button/Button";
import { JOURNAL_WORKS } from "../../../utils/journalWorks";
import styles from "./JournalForm.module.css";

interface JournalFormProps {
    initialTitle?: string;
    initialWork?: JournalWork;
    submitLabel: string;
    submittingLabel: string;
    onSubmit: (data: { title: string; work: JournalWork }) => Promise<void>;
}

export function JournalForm({
    initialTitle = "",
    initialWork = "general",
    submitLabel,
    submittingLabel,
    onSubmit,
}: JournalFormProps) {
    const [title, setTitle] = useState(initialTitle);
    const [work, setWork] = useState<JournalWork>(initialWork);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    async function handleSubmit(e: SubmitEvent) {
        e.preventDefault();
        if (!title.trim() || submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            await onSubmit({ title: title.trim(), work });
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

            <div className={styles.actions}>
                <Button variant="primary" size="medium" disabled={submitting || !title.trim()}>
                    {submitting ? submittingLabel : submitLabel}
                </Button>
            </div>
        </form>
    );
}
