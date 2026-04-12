import { useCallback, useEffect, useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router";
import type { JournalDetail, PostComment } from "../../types/api";
import {
    createJournalComment,
    deleteJournal,
    deleteJournalComment,
    followJournal,
    getJournal,
    likeJournalComment,
    unfollowJournal,
    unlikeJournalComment,
    updateJournalComment,
    uploadJournalCommentMedia,
} from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import { can } from "../../utils/permissions";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Button } from "../../components/Button/Button";
import { CommentItem } from "../../components/post/CommentItem/CommentItem";
import { CommentComposer } from "../../components/post/CommentComposer/CommentComposer";
import { ReportButton } from "../../components/ReportButton/ReportButton";
import { linkify } from "../../utils/linkify";
import { relativeTime } from "../../utils/notifications";
import { workLabel } from "../../utils/journalWorks";
import styles from "./JournalPage.module.css";

export function JournalPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const [journal, setJournal] = useState<JournalDetail | null>(null);
    const [loading, setLoading] = useState(true);
    const [following, setFollowing] = useState(false);
    usePageTitle(journal?.title ?? "Journal");

    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;

    const fetchJournal = useCallback(() => {
        if (!id) {
            return;
        }
        getJournal(id)
            .then(j => {
                setJournal(j);
                setFollowing(j.is_following);
            })
            .catch(() => setJournal(null))
            .finally(() => setLoading(false));
    }, [id]);

    useEffect(() => {
        fetchJournal();
    }, [fetchJournal]);

    useScrollToHash(!loading && !!journal, highlightedComment ? `comment-${highlightedComment}` : null);

    async function handleFollow() {
        if (!journal || !id) {
            return;
        }
        const wasFollowing = following;
        setFollowing(!wasFollowing);
        try {
            if (wasFollowing) {
                await unfollowJournal(id);
            } else {
                await followJournal(id);
            }
        } catch {
            setFollowing(wasFollowing);
        }
    }

    async function handleDelete() {
        if (!id || !window.confirm("Delete this journal? This cannot be undone.")) {
            return;
        }
        try {
            await deleteJournal(id);
            navigate("/journals");
        } catch {}
    }

    if (loading) {
        return <div className="loading">Loading journal...</div>;
    }

    if (!journal) {
        return <div className="empty-state">Journal not found.</div>;
    }

    const isOwner = user?.id === journal.author.id;
    const canEdit = isOwner || can(user?.role, "edit_any_journal");
    const canDelete = isOwner || can(user?.role, "delete_any_journal");
    const comments = journal.comments ?? [];
    const canComment = user && !journal.is_archived;

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate("/journals")}>
                &larr; All Journals
            </span>

            <div className={styles.detail}>
                <div className={styles.header}>
                    <h1 className={styles.title}>{journal.title}</h1>
                    <span className={styles.work}>{workLabel(journal.work)}</span>
                    {journal.is_archived && <span className={styles.archived}>Archived</span>}
                </div>
                <div className={styles.meta}>
                    <ProfileLink user={journal.author} size="small" />
                    <span>{relativeTime(journal.created_at)}</span>
                    {journal.updated_at && <span>(edited)</span>}
                    <span className={styles.followerCount}>
                        {"\u2605"} {journal.follower_count} follower{journal.follower_count === 1 ? "" : "s"}
                    </span>
                </div>

                <p className={styles.body}>{linkify(journal.body)}</p>

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
                    {canDelete && (
                        <Button variant="ghost" size="small" onClick={handleDelete}>
                            Delete
                        </Button>
                    )}
                    {user && !isOwner && <ReportButton targetType="journal" targetId={journal.id} />}
                </div>

                {journal.is_archived && (
                    <div className={styles.archivedBanner}>
                        This journal was archived after 7 days of inactivity. New comments are disabled.
                    </div>
                )}
            </div>

            <div className={styles.commentsSection}>
                <h3 className={styles.commentsTitle}>
                    Updates &amp; Discussion {comments.length > 0 && `(${comments.length})`}
                </h3>
                {comments.map(c => (
                    <CommentItem
                        key={c.id}
                        comment={c as unknown as PostComment}
                        postId={journal.id}
                        onDelete={fetchJournal}
                        highlightedId={highlightedComment ?? undefined}
                        linkPrefix="/journals"
                        reportType="journal_comment"
                        likeFn={likeJournalComment}
                        unlikeFn={unlikeJournalComment}
                        deleteFn={deleteJournalComment}
                        updateFn={updateJournalComment}
                        createCommentFn={createJournalComment}
                        uploadMediaFn={uploadJournalCommentMedia}
                    />
                ))}
                {comments.length === 0 && !journal.is_archived && (
                    <p className="empty-state">No entries yet. {isOwner && "Post the first update!"}</p>
                )}
                {canComment && (
                    <CommentComposer
                        postId={journal.id}
                        onCreated={fetchJournal}
                        createCommentFn={createJournalComment}
                        uploadMediaFn={uploadJournalCommentMedia}
                    />
                )}
            </div>
        </div>
    );
}
