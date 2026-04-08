import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { createFanficChapter, getFanficChapter, updateFanficChapter } from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { RichTextEditor } from "../../components/RichTextEditor/RichTextEditor";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import styles from "./FanficPages.module.css";

export function ChapterEditorPage() {
    const { id: fanficId, number: numParam } = useParams<{ id: string; number: string }>();
    const navigate = useNavigate();

    const isEdit = numParam !== "new" && numParam !== undefined;
    const chapterNumber = isEdit ? Number(numParam) : 0;

    const [chapterId, setChapterId] = useState("");
    const [title, setTitle] = useState("");
    const [body, setBody] = useState("");
    const [loading, setLoading] = useState(isEdit);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        if (!isEdit || !fanficId || isNaN(chapterNumber)) {
            return;
        }
        getFanficChapter(fanficId, chapterNumber)
            .then(ch => {
                setChapterId(ch.id);
                setTitle(ch.title);
                setBody(ch.body);
                setLoading(false);
            })
            .catch(() => setLoading(false));
    }, [isEdit, fanficId, chapterNumber]);

    async function handleSubmit() {
        if (!body.trim() || submitting || !fanficId) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            if (isEdit) {
                await updateFanficChapter(chapterId, title.trim(), body);
                navigate(`/fanfiction/${fanficId}/chapter/${chapterNumber}`);
            } else {
                await createFanficChapter(fanficId, title.trim(), body);
                navigate(`/fanfiction/${fanficId}`);
            }
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to save chapter");
        } finally {
            setSubmitting(false);
        }
    }

    if (loading) {
        return <div className="loading">Loading...</div>;
    }

    if (isEdit && !chapterId) {
        return <div className="empty-state">Chapter not found.</div>;
    }

    const backPath = isEdit ? `/fanfiction/${fanficId}/chapter/${chapterNumber}` : `/fanfiction/${fanficId}`;

    return (
        <div className={styles.formPage}>
            <span className={styles.back} onClick={() => navigate(backPath)}>
                &larr; {isEdit ? "Back to Chapter" : "Back to Fanfic"}
            </span>
            <h1 className={styles.formHeading}>{isEdit ? `Edit Chapter ${chapterNumber}` : "Add Chapter"}</h1>

            <div className={styles.formRow}>
                <label className={styles.formLabel}>Chapter Title (optional)</label>
                <Input
                    type="text"
                    value={title}
                    onChange={e => setTitle(e.target.value)}
                    placeholder="Chapter title..."
                    fullWidth
                />
            </div>

            <div className={styles.formRow}>
                <label className={styles.formLabel}>Content</label>
                <RichTextEditor content={body} onChange={setBody} placeholder="Write your chapter here..." />
            </div>

            {error && <ErrorBanner message={error} />}

            <div className={styles.formActions}>
                <Button variant="ghost" onClick={() => navigate(backPath)}>
                    Cancel
                </Button>
                <Button variant="primary" onClick={handleSubmit} disabled={submitting || !body.trim()}>
                    {submitting ? "Saving..." : "Save Chapter"}
                </Button>
            </div>
        </div>
    );
}
