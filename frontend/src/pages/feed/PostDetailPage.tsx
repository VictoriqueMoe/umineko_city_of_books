import { useLocation, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import { usePost } from "../../api/queries/post";
import { useAuth } from "../../hooks/useAuth";
import { PostCard } from "../../components/post/PostCard/PostCard";
import { CommentsSection } from "../../components/post/CommentsSection/CommentsSection";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import styles from "./PostDetailPage.module.css";

export function PostDetailPage() {
    usePageTitle("Post");
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const { post, loading, refresh } = usePost(id ?? "");
    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;

    const fetchPost = () => {
        refresh();
    };

    useScrollToHash(!loading && !!post, highlightedComment ? `comment-${highlightedComment}` : null);

    if (loading) {
        return <div className="loading">Loading post...</div>;
    }

    if (!post) {
        return <div className="empty-state">Post not found.</div>;
    }

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate(-1)}>
                &larr; Back to Game Board
            </span>
            <PostCard post={post} onDelete={() => navigate("/game-board")} onEdit={fetchPost} />

            {post.liked_by && post.liked_by.length > 0 && (
                <div className={styles.likedBy}>
                    <h3 className={styles.commentsTitle}>Liked by ({post.liked_by.length})</h3>
                    <div className={styles.likedByList}>
                        {post.liked_by.map(u => (
                            <ProfileLink key={u.id} user={u} size="small" />
                        ))}
                    </div>
                </div>
            )}

            <CommentsSection
                comments={post.comments}
                targetId={post.id}
                user={user}
                onChanged={fetchPost}
                blockedText="You cannot interact with this post."
                viewerBlocked={post.viewer_blocked}
                highlightedId={highlightedComment ?? undefined}
            />
        </div>
    );
}
