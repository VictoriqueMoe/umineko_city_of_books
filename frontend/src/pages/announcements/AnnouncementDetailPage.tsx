import { useLocation, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import { marked } from "marked";
import DOMPurify from "dompurify";
import type { PostComment } from "../../types/api";
import { useAnnouncement } from "../../api/queries/announcement";
import {
    useCreateAnnouncementComment,
    useDeleteAnnouncementComment,
    useLikeAnnouncementComment,
    useUnlikeAnnouncementComment,
    useUpdateAnnouncementComment,
    useUploadAnnouncementCommentMedia,
} from "../../api/mutations/announcement";
import { useAuth } from "../../hooks/useAuth";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { CommentItem } from "../../components/post/CommentItem/CommentItem";
import { CommentComposer } from "../../components/post/CommentComposer/CommentComposer";
import { relativeTime } from "../../utils/notifications";
import styles from "./AnnouncementsPage.module.css";

function renderMarkdown(md: string): string {
    const raw = marked.parse(md, { async: false }) as string;
    return DOMPurify.sanitize(raw);
}

export function AnnouncementDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const { announcement, loading, refresh } = useAnnouncement(id ?? "");
    usePageTitle(announcement?.title ?? "Announcement");
    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;

    const createCommentMutation = useCreateAnnouncementComment(id ?? "");
    const updateCommentMutation = useUpdateAnnouncementComment(id ?? "");
    const deleteCommentMutation = useDeleteAnnouncementComment(id ?? "");
    const likeCommentMutation = useLikeAnnouncementComment(id ?? "");
    const unlikeCommentMutation = useUnlikeAnnouncementComment(id ?? "");
    const uploadMediaMutation = useUploadAnnouncementCommentMedia(id ?? "");

    useScrollToHash(!loading && !!announcement, highlightedComment ? `comment-${highlightedComment}` : null);

    if (loading) {
        return <div className="loading">Loading announcement...</div>;
    }

    if (!announcement) {
        return <div className="empty-state">Announcement not found.</div>;
    }

    const comments = announcement.comments ?? [];

    const likeFn = (commentId: string) => likeCommentMutation.mutateAsync(commentId);
    const unlikeFn = (commentId: string) => unlikeCommentMutation.mutateAsync(commentId);
    const deleteFn = (commentId: string) => deleteCommentMutation.mutateAsync(commentId);
    const updateFn = (commentId: string, body: string) =>
        updateCommentMutation.mutateAsync({ id: commentId, body }).then(() => undefined);
    const createCommentFn = (_postId: string, body: string, parentId?: string) =>
        createCommentMutation.mutateAsync({ body, parentId });
    const uploadMediaFn = (commentId: string, file: File) => uploadMediaMutation.mutateAsync({ commentId, file });

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate("/announcements")}>
                &larr; All Announcements
            </span>

            <div className={styles.detail}>
                <h1 className={styles.detailTitle}>{announcement.title}</h1>
                <div className={styles.detailMeta}>
                    <ProfileLink user={announcement.author} size="small" />
                    <span>{relativeTime(announcement.created_at)}</span>
                    {announcement.updated_at !== announcement.created_at && <span>(edited)</span>}
                </div>
                <div className={styles.body} dangerouslySetInnerHTML={{ __html: renderMarkdown(announcement.body) }} />
            </div>

            <div className={styles.commentsSection}>
                <h3 className={styles.commentsTitle}>Comments {comments.length > 0 && `(${comments.length})`}</h3>
                {comments.map(c => (
                    <CommentItem
                        key={c.id}
                        comment={c as unknown as PostComment}
                        postId={announcement.id}
                        onDelete={() => refresh()}
                        highlightedId={highlightedComment ?? undefined}
                        linkPrefix="/announcements"
                        reportType="announcement_comment"
                        likeFn={likeFn}
                        unlikeFn={unlikeFn}
                        deleteFn={deleteFn}
                        updateFn={updateFn}
                        createCommentFn={createCommentFn}
                        uploadMediaFn={uploadMediaFn}
                    />
                ))}
                {comments.length === 0 && <p className="empty-state">No comments yet.</p>}
                {user && id && (
                    <CommentComposer
                        postId={id}
                        onCreated={() => refresh()}
                        createCommentFn={createCommentFn}
                        uploadMediaFn={uploadMediaFn}
                    />
                )}
            </div>
        </div>
    );
}
