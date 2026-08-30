import { type ReactNode, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { useJournal, useJournalEntry } from "../../hooks/queries/journal";
import { useAuth } from "../../hooks/useAuth";
import { useCommentHandlers } from "../../hooks/useCommentHandlers";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import { contentPermissions, type ContentSubject } from "../../domain/contentPermissions";
import { errorMessage } from "../../utils/errorMessage";
import { Button } from "../../components/Button/Button";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { CommentsSection } from "../../components/post/CommentsSection/CommentsSection";
import { useDeleteJournalEntry } from "../../hooks/mutations/journal";
import { renderRich } from "../../components/richText/richText";
import { extractGif } from "../../utils/gif";
import { GifEmbed } from "../../components/GifEmbed/GifEmbed";
import { MediaGallery } from "../../components/post/MediaGallery/MediaGallery";
import { RelativeTimestamp } from "../../components/RelativeTimestamp/RelativeTimestamp";
import styles from "./JournalEntryPage.module.css";

function entryHeading(number: number, title?: string | null): string {
    if (title && title.trim() !== "") {
        return `Entry ${number}: ${title}`;
    }
    return `Entry ${number}`;
}

function entryHeadingNode(number: number, title?: string | null): ReactNode {
    if (title && title.trim() !== "") {
        return (
            <>
                Entry {number}: <bdi>{title}</bdi>
            </>
        );
    }
    return `Entry ${number}`;
}

export function JournalEntryPage() {
    const { id: journalId, number: numberParam } = useParams<{ id: string; number: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const entryNumber = Number(numberParam);
    const { journal, loading: jLoading } = useJournal(journalId ?? "");
    const { entry, comments, loading: eLoading, refresh } = useJournalEntry(journalId ?? "", entryNumber);
    const loading = jLoading || eLoading;
    usePageTitle(entry ? entryHeading(entry.entry_number, entry.title) : "Entry");

    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;
    useScrollToHash(!loading && !!entry, highlightedComment ? `comment-${highlightedComment}` : null);

    const { createCommentFn, updateFn, deleteFn, likeFn, unlikeFn, uploadMediaFn } = useCommentHandlers(
        "journal",
        journalId ?? "",
        { enabled: ["create", "update", "delete", "like", "unlike", "uploadMedia"], entryId: entry?.id },
    );
    const deleteEntryMutation = useDeleteJournalEntry(journalId ?? "");
    const [deleteError, setDeleteError] = useState("");

    if (loading) {
        return <div className="loading">Loading entry...</div>;
    }
    if (!journal || !entry) {
        return <div className="empty-state">Entry not found.</div>;
    }

    const subject: ContentSubject = { family: "journal_entry", authorId: journal.author.id };
    const { canEdit, canDelete } = contentPermissions(user, subject);
    const canComment = user && !journal.is_archived;

    async function handleDeleteEntry() {
        if (!window.confirm("Delete this entry? This cannot be undone.")) {
            return;
        }
        setDeleteError("");
        try {
            await deleteEntryMutation.mutateAsync(entry!.id);
            navigate(`/journals/${journalId}`);
        } catch (thrown) {
            setDeleteError(errorMessage(thrown, "Failed to delete this entry"));
        }
    }

    function navigateTo(target: number) {
        navigate(`/journals/${journalId}/entry/${target}`);
        window.scrollTo({ top: 0, behavior: "smooth" });
    }

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
                &larr; Back to <bdi>{journal.title}</bdi>
            </span>

            <div className={styles.entry}>
                <div className={styles.meta}>
                    <ProfileLink user={journal.author} size="small" />
                    <RelativeTimestamp value={entry.created_at} />
                    {entry.updated_at && entry.updated_at !== entry.created_at && <span>(edited)</span>}
                </div>
                <h1 dir="auto" className={styles.title}>
                    {entryHeadingNode(entry.entry_number, entry.title)}
                    {entry.is_draft && <span className={styles.draftBadge}>Draft</span>}
                </h1>

                {entry.is_draft && (
                    <div className={styles.draftBanner}>
                        Only you can see this draft. Open it in the editor to publish — that's what notifies your
                        followers.
                    </div>
                )}

                {navButtons()}

                {(() => {
                    const gifURL = extractGif(entry.body);
                    if (gifURL) {
                        return <GifEmbed src={gifURL} />;
                    }
                    return (
                        <div dir="auto" className={styles.body}>
                            {renderRich(entry.body)}
                        </div>
                    );
                })()}

                <MediaGallery media={entry.media} />

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

                {deleteError && <ErrorBanner message={deleteError} />}

                {navButtons()}
            </div>

            <CommentsSection
                comments={comments}
                targetId={entry.id}
                user={canComment ? user : null}
                onChanged={() => refresh()}
                title={`Comments on entry ${entry.entry_number}`}
                emptyText={journal.is_archived ? null : "No comments on this entry yet."}
                highlightedId={highlightedComment ?? undefined}
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
