import { type SubmitEvent, useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useJournal, useJournalEntry } from "../../api/queries/journal";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { can } from "../../utils/permissions";
import { validateFileSize } from "../../utils/fileValidation";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { MentionTextArea } from "../../components/MentionTextArea/MentionTextArea";
import { MediaPickerButton, MediaPreviews } from "../../components/MediaPicker/MediaPicker";
import { GifPicker } from "../../components/chat/GifPicker/GifPicker";
import {
    useCreateJournalEntry,
    useDeleteJournalEntryMedia,
    useUpdateJournalEntry,
    useUploadJournalEntryMedia,
} from "../../api/mutations/journal";
import styles from "./JournalEntryEditorPage.module.css";

export function JournalEntryEditorPage() {
    const { id: journalId, number: numberParam } = useParams<{ id: string; number?: string }>();
    const isEdit = numberParam !== undefined && numberParam !== "new";
    const entryNumber = isEdit ? Number(numberParam) : 0;
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const siteInfo = useSiteInfo();
    const { journal, loading: jLoading } = useJournal(journalId ?? "");
    const { entry, loading: eLoading } = useJournalEntry(journalId ?? "", isEdit ? entryNumber : 0);

    const createMutation = useCreateJournalEntry(journalId ?? "");
    const updateMutation = useUpdateJournalEntry(journalId ?? "");
    const uploadMediaMutation = useUploadJournalEntryMedia(journalId ?? "");
    const deleteMediaMutation = useDeleteJournalEntryMedia(journalId ?? "");

    const [titleDraft, setTitleDraft] = useState<string | null>(null);
    const [bodyDraft, setBodyDraft] = useState<string | null>(null);
    const [files, setFiles] = useState<File[]>([]);
    const [pendingDeletions, setPendingDeletions] = useState<number[]>([]);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const [gifPickerOpen, setGifPickerOpen] = useState(false);

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

    const handlePasteFiles = useCallback(
        (pasted: File[]) => {
            const errors: string[] = [];
            const valid: File[] = [];
            for (const file of pasted) {
                const err = validateFileSize(file, siteInfo.max_image_size, siteInfo.max_video_size);
                if (err) {
                    errors.push(err);
                } else {
                    valid.push(file);
                }
            }
            if (errors.length > 0) {
                setError(errors.join(" "));
            }
            if (valid.length > 0) {
                setFiles(prev => [...prev, ...valid]);
            }
        },
        [siteInfo.max_image_size, siteInfo.max_video_size],
    );

    function removeFile(index: number) {
        setFiles(prev => prev.filter((_, i) => i !== index));
    }

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

    async function uploadAllTo(entryId: string) {
        for (const file of files) {
            try {
                await uploadMediaMutation.mutateAsync({ entryId, file });
            } catch (err) {
                setError(err instanceof Error ? err.message : "Failed to upload media");
            }
        }
    }

    async function deletePendingFrom(entryId: string) {
        for (const mediaId of pendingDeletions) {
            try {
                await deleteMediaMutation.mutateAsync({ entryId, mediaId });
            } catch (err) {
                setError(err instanceof Error ? err.message : "Failed to remove attachment");
            }
        }
    }

    function togglePendingDeletion(mediaId: number) {
        setPendingDeletions(prev => (prev.includes(mediaId) ? prev.filter(id => id !== mediaId) : [...prev, mediaId]));
    }

    async function save(asDraft: boolean) {
        if ((!body.trim() && files.length === 0) || submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            if (isEdit && entry) {
                await updateMutation.mutateAsync({
                    id: entry.id,
                    payload: { title: title.trim(), body: body.trim(), is_draft: asDraft },
                });
                await deletePendingFrom(entry.id);
                await uploadAllTo(entry.id);
                if (asDraft) {
                    navigate(`/journals/${journalId}`);
                } else {
                    navigate(`/journals/${journalId}/entry/${entry.entry_number}`);
                }
            } else {
                const result = await createMutation.mutateAsync({
                    title: title.trim(),
                    body: body.trim(),
                    is_draft: asDraft,
                });
                await uploadAllTo(result.id);
                if (asDraft) {
                    navigate(`/journals/${journalId}`);
                } else {
                    navigate(`/journals/${journalId}/entry/${result.entry_number}`);
                }
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to save entry");
            setSubmitting(false);
        }
    }

    const wasDraft = isEdit && entry ? entry.is_draft : false;
    const canSaveAsDraft = !isEdit || (isEdit && entry !== null && entry !== undefined && entry.is_draft);

    async function handleSubmit(e: SubmitEvent) {
        e.preventDefault();
        await save(false);
    }

    async function handleGifPick(gif: { url: string }) {
        setGifPickerOpen(false);
        if (submitting) {
            return;
        }
        const next = body.trim() === "" ? gif.url : `${body}\n${gif.url}`;
        setBody(next);
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
                        onPasteFiles={handlePasteFiles}
                        showColours
                    />
                </div>

                {isEdit && entry && entry.media.length > 0 && (
                    <div className={styles.existingMedia}>
                        {entry.media.map(m => {
                            const markedForRemoval = pendingDeletions.includes(m.id);
                            const itemClass = `${styles.existingMediaItem}${markedForRemoval ? ` ${styles.existingMediaItemPending}` : ""}`;
                            return (
                                <div key={m.id} className={itemClass}>
                                    {m.media_type === "video" ? (
                                        <video src={m.media_url} className={styles.existingMediaThumb} />
                                    ) : (
                                        <img src={m.media_url} className={styles.existingMediaThumb} alt="" />
                                    )}
                                    <button
                                        type="button"
                                        className={styles.existingMediaRemove}
                                        onClick={() => togglePendingDeletion(m.id)}
                                        aria-label={markedForRemoval ? "Undo remove" : "Remove attachment"}
                                        title={markedForRemoval ? "Undo remove" : "Remove on save"}
                                    >
                                        {markedForRemoval ? "↺" : "×"}
                                    </button>
                                </div>
                            );
                        })}
                    </div>
                )}

                <MediaPreviews files={files} onRemove={removeFile} />

                <div className={styles.toolbar}>
                    <MediaPickerButton onFiles={valid => setFiles(prev => [...prev, ...valid])} onError={setError} />
                    <div className={styles.gifAnchor}>
                        <Button
                            type="button"
                            variant="ghost"
                            size="small"
                            onClick={() => setGifPickerOpen(prev => !prev)}
                            disabled={submitting}
                        >
                            + GIF
                        </Button>
                        {gifPickerOpen && <GifPicker onPick={handleGifPick} onClose={() => setGifPickerOpen(false)} />}
                    </div>
                </div>

                {isEdit && wasDraft && (
                    <p className={styles.draftHint}>
                        This entry is a draft. Save keeps it private; Publish notifies your followers.
                    </p>
                )}

                <div className={styles.actions}>
                    {canSaveAsDraft && (
                        <Button
                            type="button"
                            variant="ghost"
                            size="medium"
                            onClick={() => void save(true)}
                            disabled={submitting || (!body.trim() && files.length === 0)}
                        >
                            {submitting ? "Saving..." : "Save as draft"}
                        </Button>
                    )}
                    <Button
                        variant="primary"
                        size="medium"
                        disabled={submitting || (!body.trim() && files.length === 0)}
                    >
                        {submitting
                            ? "Saving..."
                            : isEdit
                              ? wasDraft
                                  ? "Publish entry"
                                  : "Save entry"
                              : "Publish entry"}
                    </Button>
                </div>
            </form>
        </div>
    );
}
