import { useState, type ReactNode } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router";
import { useJournal } from "../../hooks/queries/journal";
import {
    useDeleteJournal,
    useFollowJournal,
    useSetJournalPaused,
    useUnfollowJournal,
} from "../../hooks/mutations/journal";
import { useAuth } from "../../hooks/useAuth";
import { useCommentHandlers } from "../../hooks/useCommentHandlers";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import { contentPermissions, isContentOwner, type ContentSubject } from "../../domain/contentPermissions";
import { errorMessage } from "../../utils/errorMessage";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Button } from "../../components/Button/Button";
import { CommentsSection } from "../../components/post/CommentsSection/CommentsSection";
import { ReportButton } from "../../components/ReportButton/ReportButton";
import { renderRich } from "../../components/richText/richText";
import { extractGif } from "../../utils/gif";
import { workLabel } from "../../domain/journal";
import { GifEmbed } from "../../components/GifEmbed/GifEmbed";
import { MediaGallery } from "../../components/post/MediaGallery/MediaGallery";
import { RelativeTimestamp } from "../../components/RelativeTimestamp/RelativeTimestamp";
import styles from "./JournalPage.module.css";

function entryHeading(number: number, title?: string | null): ReactNode {
    if (title && title.trim() !== "") {
        return (
            <>
                Entry {number}: <bdi>{title}</bdi>
            </>
        );
    }
    return `Entry ${number}`;
}

export function JournalPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const { journal, loading, refresh } = useJournal(id ?? "");
    const following = journal?.is_following ?? false;
    const [deleteError, setDeleteError] = useState("");
    usePageTitle(journal?.title ?? "Journal");

    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;

    const followMutation = useFollowJournal();
    const unfollowMutation = useUnfollowJournal();
    const deleteJournalMutation = useDeleteJournal();
    const setPausedMutation = useSetJournalPaused();
    const { createCommentFn, updateFn, deleteFn, likeFn, unlikeFn, uploadMediaFn } = useCommentHandlers(
        "journal",
        id ?? "",
        { enabled: ["create", "update", "delete", "like", "unlike", "uploadMedia"] },
    );

    useScrollToHash(!loading && !!journal, highlightedComment ? `comment-${highlightedComment}` : null);

    function handleFollow() {
        if (!journal || !id) {
            return;
        }

        if (following) {
            unfollowMutation.mutate(id);
            return;
        }

        followMutation.mutate(id);
    }

    function handleTogglePause() {
        if (!journal || !id) {
            return;
        }

        setPausedMutation.mutate({ id, paused: !journal.is_paused });
    }

    async function handleDelete() {
        if (!id || !window.confirm("Delete this journal? This cannot be undone.")) {
            return;
        }

        setDeleteError("");

        try {
            await deleteJournalMutation.mutateAsync(id);
        } catch (e) {
            setDeleteError(errorMessage(e, "Could not delete this journal."));
            return;
        }

        navigate("/journals");
    }

    if (loading) {
        return <div className="loading">Loading journal...</div>;
    }

    if (!journal) {
        return <div className="empty-state">Journal not found.</div>;
    }

    const subject: ContentSubject = { family: "journal", authorId: journal.author.id };
    const isOwner = isContentOwner(user, subject);
    const { canEdit, canDelete } = contentPermissions(user, subject);
    const comments = journal.comments ?? [];
    const canComment = user && !journal.is_archived;
    const entries = journal.entries ?? [];
    const latestEntry = journal.latest_entry;

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate("/journals")}>
                &larr; All Journals
            </span>

            <div className={styles.detail}>
                <div className={styles.header}>
                    <h1 dir="auto" className={styles.title}>
                        {journal.title}
                    </h1>
                    <span className={styles.work}>{workLabel(journal.work)}</span>
                    {journal.is_archived && <span className={styles.archived}>Archived</span>}
                    {!journal.is_archived && journal.is_paused && <span className={styles.archived}>Paused</span>}
                </div>
                <div className={styles.meta}>
                    <ProfileLink user={journal.author} size="small" />
                    <RelativeTimestamp value={journal.created_at} />
                    {journal.updated_at && <span>(edited)</span>}
                    <span className={styles.followerCount}>
                        {"★"} {journal.follower_count} follower{journal.follower_count === 1 ? "" : "s"}
                    </span>
                </div>

                <div className={styles.actions}>
                    {user && !isOwner && (
                        <Button variant={following ? "secondary" : "primary"} size="small" onClick={handleFollow}>
                            {following ? "Following" : "Follow"}
                        </Button>
                    )}
                    {canEdit && (
                        <Link to={`/journals/${journal.id}/edit`}>
                            <Button variant="ghost" size="small">
                                Edit
                            </Button>
                        </Link>
                    )}
                    {isOwner && !journal.is_archived && (
                        <Button variant="ghost" size="small" onClick={handleTogglePause}>
                            {journal.is_paused ? "Resume" : "Pause"}
                        </Button>
                    )}
                    {canDelete && (
                        <Button variant="ghost" size="small" onClick={handleDelete}>
                            Delete
                        </Button>
                    )}
                    {user && !isOwner && <ReportButton targetType="journal" targetId={journal.id} />}
                </div>

                {deleteError && (
                    <div role="alert" className={styles.error}>
                        {deleteError}
                    </div>
                )}

                {journal.is_archived && (
                    <div className={styles.archivedBanner}>
                        This journal was archived after 7 days of inactivity. New comments are disabled.
                        {isOwner && " Post a new entry to reopen it."}
                    </div>
                )}
                {!journal.is_archived && journal.is_paused && (
                    <div className={styles.archivedBanner}>
                        This journal is paused, so it will not be archived while you are away.
                        {isOwner && " Resume it whenever you are ready to continue."}
                    </div>
                )}
            </div>

            {latestEntry ? (
                <div className={styles.latestSpotlight}>
                    <div className={styles.spotlightHeader}>
                        <span className={styles.spotlightTag}>Latest update</span>
                        <RelativeTimestamp value={latestEntry.created_at} className={styles.spotlightWhen} />
                    </div>
                    <h2 dir="auto" className={styles.spotlightTitle}>
                        <Link to={`/journals/${journal.id}/entry/${latestEntry.entry_number}`}>
                            {entryHeading(latestEntry.entry_number, latestEntry.title)}
                        </Link>
                    </h2>
                    {(() => {
                        const gifURL = extractGif(latestEntry.body);
                        if (gifURL) {
                            return <GifEmbed src={gifURL} />;
                        }
                        return (
                            <div dir="auto" className={styles.spotlightBody}>
                                {renderRich(latestEntry.body)}
                            </div>
                        );
                    })()}
                    {latestEntry.media.length > 0 && <MediaGallery media={latestEntry.media} />}
                    <div className={styles.spotlightFooter}>
                        <Link to={`/journals/${journal.id}/entry/${latestEntry.entry_number}`}>
                            <Button variant="primary" size="small">
                                Read full entry &rarr;
                            </Button>
                        </Link>
                        <span className={styles.spotlightWordCount}>{latestEntry.word_count} words</span>
                    </div>
                </div>
            ) : (
                <div className={styles.noEntries}>
                    No entries yet.
                    {isOwner ? " Add the first one below." : " Check back soon."}
                </div>
            )}

            <div className={styles.tocSection}>
                <div className={styles.tocHeader}>
                    <h3 className={styles.tocTitle}>All entries ({entries.length})</h3>
                    {canEdit && (
                        <Link to={`/journals/${journal.id}/entry/new`}>
                            <Button variant="primary" size="small">
                                {journal.is_archived ? "+ New Entry (reopens)" : "+ New Entry"}
                            </Button>
                        </Link>
                    )}
                </div>
                {entries.length === 0 ? (
                    <div className={styles.tocEmpty}>No entries to list yet.</div>
                ) : (
                    <ol className={styles.tocList}>
                        {entries.map(e => (
                            <li key={e.id} className={styles.tocItem}>
                                <Link
                                    to={`/journals/${journal.id}/entry/${e.entry_number}`}
                                    className={styles.tocItemLink}
                                >
                                    <span className={styles.tocItemNumber}>#{e.entry_number}</span>
                                    <span dir="auto" className={styles.tocItemTitle}>
                                        {entryHeading(e.entry_number, e.title)}
                                    </span>
                                    {e.is_draft && <span className={styles.draftBadge}>Draft</span>}
                                    <span className={styles.tocItemMeta}>
                                        {e.word_count} words {"·"} <RelativeTimestamp value={e.created_at} />
                                    </span>
                                </Link>
                            </li>
                        ))}
                    </ol>
                )}
            </div>

            <CommentsSection
                comments={comments}
                targetId={journal.id}
                user={canComment ? user : null}
                onChanged={() => refresh()}
                title="Journal discussion"
                emptyText={
                    journal.is_archived
                        ? null
                        : "No discussion yet. Comments here are about the journal as a whole — entry-specific comments live on each entry's page."
                }
                highlightedId={highlightedComment ?? undefined}
                linkPrefix="/journals"
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
