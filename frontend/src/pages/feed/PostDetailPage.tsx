import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { PostDetail } from "../../types/api";
import { getPost } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { PostCard } from "../../components/post/PostCard/PostCard";
import { CommentItem } from "../../components/post/CommentItem/CommentItem";
import { CommentComposer } from "../../components/post/CommentComposer/CommentComposer";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import styles from "./PostDetailPage.module.css";

export function PostDetailPage() {
    usePageTitle("Post");
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const [post, setPost] = useState<PostDetail | null>(null);
    const [loading, setLoading] = useState(true);
    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;

    const fetchPost = useCallback(() => {
        if (!id) {
            return;
        }
        getPost(id)
            .then(setPost)
            .catch(() => setPost(null))
            .finally(() => setLoading(false));
    }, [id]);

    useEffect(() => {
        fetchPost();
    }, [fetchPost]);

    useEffect(() => {
        if (!post || loading || !highlightedComment) {
            return;
        }

        requestAnimationFrame(() => {
            const el = document.getElementById(`comment-${highlightedComment}`);
            if (el) {
                el.scrollIntoView({ behavior: "smooth", block: "center" });
            }
        });
    }, [post, loading, highlightedComment]);

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

            <div className={styles.comments}>
                <h3 className={styles.commentsTitle}>
                    Comments {post.comments.length > 0 && `(${post.comments.length})`}
                </h3>
                {post.comments.map(c => (
                    <CommentItem
                        key={c.id}
                        comment={c}
                        postId={post.id}
                        onDelete={fetchPost}
                        highlightedId={highlightedComment ?? undefined}
                        viewerBlocked={post.viewer_blocked}
                    />
                ))}
                {post.comments.length === 0 && <p className={styles.noComments}>No comments yet.</p>}
                {post.viewer_blocked && <p className={styles.blockedNotice}>You cannot interact with this post.</p>}
                {user && !post.viewer_blocked && <CommentComposer postId={post.id} onCreated={fetchPost} />}
            </div>
        </div>
    );
}
