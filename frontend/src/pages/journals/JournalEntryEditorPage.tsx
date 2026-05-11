import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useJournal, useJournalEntry } from "../../api/queries/journal";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { can } from "../../utils/permissions";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { MentionTextArea } from "../../components/MentionTextArea/MentionTextArea";
import { useCreateJournalEntry, useUpdateJournalEntry } from "../../api/mutations/journal";
import styles from "./JournalEntryEditorPage.module.css";

export function JournalEntryEditorPage() {
    const { id: journalId, number: numberParam } = useParams<{ id: string; number?: string }>();
    const isEdit = numberParam !== undefined && numberParam !== "new";
    const entryNumber = isEdit ? Number(numberParam) : 0;
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const { journal, loading: jLoading } = useJournal(journalId ?? "");
    const { entry, loading: eLoading } = useJournalEntry(journalId ?? "", isEdit ? entryNumber : 0);

    const createMutation = useCreateJournalEntry(journalId ?? "");
    const updateMutation = useUpdateJournalEntry(journalId ?? "");

    const [titleDraft, setTitleDraft] = useState<string | null>(null);
    const [bodyDraft, setBodyDraft] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    const title = titleDraft ?? (isEdit && entry ? (entry.title ?? "") : "");
    const body = bodyDraft ?? (isEdit && entry ? entry.body : "");
    const setTitle = setTitleDraft;
    const setBody = setBodyDraft;

    usePageTitle(isEdit ? "Edit Entry" : "New Entry");

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    if (authLoading || jLoading || (isEdit && eLoading)) {
        return <div className="loading">Loading...</div>;
    }
    if (!user || !journal) {
        return <div className="empty-state">Journal not found.</div>;
    }
    if (isEdit && !entry) {
        return <div className="empty-state">Entry not found.</div>;
    }

    const isOwner = user.id === journal.author.id;
    if (!isOwner && !can(user.role, "edit_any_journal")) {
        return <div className="empty-state">You can't edit this journal.</div>;
    }

    async function handleSubmit(e: React.FormEvent) {
        e.preventDefault();
        if (!body.trim() || submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            if (isEdit && entry) {
                await updateMutation.mutateAsync({
                    id: entry.id,
                    payload: { title: title.trim(), body: body.trim() },
                });
                navigate(`/journals/${journalId}/entry/${entry.entry_number}`);
            } else {
                const result = await createMutation.mutateAsync({ title: title.trim(), body: body.trim() });
                navigate(`/journals/${journalId}/entry/${result.entry_number}`);
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to save entry");
            setSubmitting(false);
        }
    }

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate(`/journals/${journalId}`)}>
                &larr; Back to {journal.title}
            </span>
            <h2 className={styles.heading}>{isEdit ? "Edit entry" : "New entry"}</h2>

            <form className={styles.form} onSubmit={handleSubmit}>
                {error && <div className={styles.error}>{error}</div>}

                <div className={styles.field}>
                    <label className={styles.label}>Title (optional)</label>
                    <Input
                        type="text"
                        value={title}
                        onChange={e => setTitle(e.target.value)}
                        placeholder="Leave empty to use 'Entry N'"
                        maxLength={200}
                    />
                </div>

                <div className={styles.field}>
                    <label className={styles.label}>Body</label>
                    <MentionTextArea
                        value={body}
                        onChange={setBody}
                        rows={16}
                        placeholder="Write your entry. Use the colour tags above for blue/red/gold/purple truths."
                        showColours
                    />
                </div>

                <div className={styles.actions}>
                    <Button variant="primary" size="medium" disabled={submitting || !body.trim()}>
                        {submitting ? "Saving..." : isEdit ? "Save entry" : "Publish entry"}
                    </Button>
                </div>
            </form>
        </div>
    );
}
