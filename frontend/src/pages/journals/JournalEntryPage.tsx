import { useNavigate, useParams } from "react-router";
import { useJournal, useJournalEntry } from "../../api/queries/journal";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { can } from "../../utils/permissions";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { CommentsSection } from "../../components/post/CommentsSection/CommentsSection";
import {
    useCreateJournalComment,
    useDeleteJournalComment,
    useDeleteJournalEntry,
    useLikeJournalComment,
    useUnlikeJournalComment,
    useUpdateJournalComment,
    useUploadJournalCommentMedia,
} from "../../api/mutations/journal";
import { linkify } from "../../utils/linkify";
import { renderColours } from "../../utils/colours";
import { relativeTime } from "../../utils/time.ts";
import type { PostComment } from "../../types/api";
import styles from "./JournalEntryPage.module.css";

function entryHeading(number: number, title?: string | null): string {
    if (title && title.trim() !== "") {
        return `Entry ${number}: ${title}`;
    }
    return `Entry ${number}`;
}

export function JournalEntryPage() {
    const { id: journalId, number: numberParam } = useParams<{ id: string; number: string }>();
    const navigate = useNavigate();
    const { user } = useAuth();
    const entryNumber = Number(numberParam);
    const { journal, loading: jLoading } = useJournal(journalId ?? "");
    const { entry, comments, loading: eLoading, refresh } = useJournalEntry(journalId ?? "", entryNumber);
    const loading = jLoading || eLoading;
    usePageTitle(entry ? entryHeading(entry.entry_number, entry.title) : "Entry");

    const createCommentMutation = useCreateJournalComment(journalId ?? "", entry?.id);
    const updateCommentMutation = useUpdateJournalComment(journalId ?? "");
    const deleteCommentMutation = useDeleteJournalComment(journalId ?? "");
    const likeCommentMutation = useLikeJournalComment(journalId ?? "");
    const unlikeCommentMutation = useUnlikeJournalComment(journalId ?? "");
    const uploadMediaMutation = useUploadJournalCommentMedia(journalId ?? "");
    const deleteEntryMutation = useDeleteJournalEntry(journalId ?? "");

    if (loading) {
        return <div className="loading">Loading entry...</div>;
    }
    if (!journal || !entry) {
        return <div className="empty-state">Entry not found.</div>;
    }

    const isOwner = user?.id === journal.author.id;
    const canEdit = isOwner || can(user?.role, "edit_any_journal");
    const canDelete = isOwner || can(user?.role, "delete_any_journal");
    const canComment = user && !journal.is_archived;

    async function handleDeleteEntry() {
        if (!window.confirm("Delete this entry? This cannot be undone.")) {
            return;
        }
        try {
            await deleteEntryMutation.mutateAsync(entry!.id);
            navigate(`/journals/${journalId}`);
        } catch {}
    }

    function navigateTo(target: number) {
        navigate(`/journals/${journalId}/entry/${target}`);
        window.scrollTo({ top: 0, behavior: "smooth" });
    }

    const likeFn = (commentId: string) => likeCommentMutation.mutateAsync(commentId);
    const unlikeFn = (commentId: string) => unlikeCommentMutation.mutateAsync(commentId);
    const deleteFn = (commentId: string) => deleteCommentMutation.mutateAsync(commentId);
    const updateFn = (commentId: string, body: string) =>
        updateCommentMutation.mutateAsync({ id: commentId, body }).then(() => undefined);
    const createCommentFn = (_postId: string, body: string, parentId?: string) =>
        createCommentMutation.mutateAsync({ body, parentId });
    const uploadMediaFn = (commentId: string, file: File) => uploadMediaMutation.mutateAsync({ commentId, file });

    function navButtons() {
        return (
            <div className={styles.nav}>
                <Button
                    variant="secondary"
                    size="small"
                    disabled={!entry!.has_prev}
                    onClick={() => navigateTo(entryNumber - 1)}
                >
                    &larr; Previous
                </Button>
                <span className={styles.navWordCount}>{entry!.word_count} words</span>
                <Button
                    variant="secondary"
                    size="small"
                    disabled={!entry!.has_next}
                    onClick={() => navigateTo(entryNumber + 1)}
                >
                    Next &rarr;
                </Button>
            </div>
        );
    }

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate(`/journals/${journalId}`)}>
                &larr; Back to {journal.title}
            </span>

            <div className={styles.entry}>
                <div className={styles.meta}>
                    <ProfileLink user={journal.author} size="small" />
                    <span>{relativeTime(entry.created_at)}</span>
                    {entry.updated_at && entry.updated_at !== entry.created_at && <span>(edited)</span>}
                </div>
                <h1 className={styles.title}>{entryHeading(entry.entry_number, entry.title)}</h1>

                {navButtons()}

                <div className={styles.body}>{renderColours(entry.body, linkify, `entry-${entry.id}`)}</div>

                <div className={styles.actions}>
                    {canEdit && (
                        <Button
                            variant="ghost"
                            size="small"
                            onClick={() => navigate(`/journals/${journalId}/entry/${entry.entry_number}/edit`)}
                        >
                            Edit entry
                        </Button>
                    )}
                    {canDelete && (
                        <Button variant="ghost" size="small" onClick={handleDeleteEntry}>
                            Delete entry
                        </Button>
                    )}
                </div>

                {navButtons()}
            </div>

            <CommentsSection
                comments={comments as unknown as PostComment[]}
                targetId={entry.id}
                user={canComment ? user : null}
                onChanged={() => refresh()}
                title={`Comments on entry ${entry.entry_number}`}
                emptyText={journal.is_archived ? null : "No comments on this entry yet."}
                linkPrefix={`/journals/${journalId}/entry/${entry.entry_number}`}
                reportType="journal_comment"
                likeFn={likeFn}
                unlikeFn={unlikeFn}
                deleteFn={deleteFn}
                updateFn={updateFn}
                createCommentFn={createCommentFn}
                uploadMediaFn={uploadMediaFn}
            />
        </div>
    );
}
