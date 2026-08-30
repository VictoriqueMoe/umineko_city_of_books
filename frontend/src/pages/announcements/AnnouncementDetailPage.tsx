import { useLocation, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import { marked } from "marked";
import DOMPurify from "dompurify";
import { useAnnouncement } from "../../hooks/queries/announcement";
import { useAuth } from "../../hooks/useAuth";
import { useCommentHandlers } from "../../hooks/useCommentHandlers";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { CommentsSection } from "../../components/post/CommentsSection/CommentsSection";
import { RelativeTimestamp } from "../../components/RelativeTimestamp/RelativeTimestamp";
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

    const { createCommentFn, updateFn, deleteFn, likeFn, unlikeFn, uploadMediaFn } = useCommentHandlers(
        "announcement",
        id ?? "",
        { enabled: ["create", "update", "delete", "like", "unlike", "uploadMedia"] },
    );

    useScrollToHash(!loading && !!announcement, highlightedComment ? `comment-${highlightedComment}` : null);

    if (loading) {
        return <div className="loading">Loading announcement...</div>;
    }

    if (!announcement) {
        return <div className="empty-state">Announcement not found.</div>;
    }

    const comments = announcement.comments ?? [];

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate("/announcements")}>
                &larr; All Announcements
            </span>

            <div className={styles.detail}>
                <h1 dir="auto" className={styles.detailTitle}>
                    {announcement.title}
                </h1>
                <div className={styles.detailMeta}>
                    <ProfileLink user={announcement.author} size="small" />
                    <RelativeTimestamp value={announcement.created_at} />
                    {announcement.updated_at !== announcement.created_at && <span>(edited)</span>}
                </div>
                <div
                    dir="auto"
                    className={styles.body}
                    dangerouslySetInnerHTML={{ __html: renderMarkdown(announcement.body) }}
                />
            </div>

            <CommentsSection
                comments={comments}
                targetId={announcement.id}
                user={user}
                onChanged={() => refresh()}
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
        </div>
    );
}
