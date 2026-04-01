import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import type { Post } from "../../../types/api";
import { deletePost as apiDeletePost, likePost, unlikePost } from "../../../api/endpoints";
import { useAuth } from "../../../hooks/useAuth";
import { useNotifications } from "../../../hooks/useNotifications";
import { can } from "../../../utils/permissions";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { MediaGallery } from "../MediaGallery/MediaGallery";
import { Button } from "../../Button/Button";
import styles from "./PostCard.module.css";

interface PostCardProps {
    post: Post;
    onDelete?: () => void;
}

function timeAgo(dateStr: string): string {
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) {
        return "just now";
    }
    if (mins < 60) {
        return `${mins}m`;
    }
    const hours = Math.floor(mins / 60);
    if (hours < 24) {
        return `${hours}h`;
    }
    const days = Math.floor(hours / 24);
    if (days < 30) {
        return `${days}d`;
    }
    return new Date(dateStr).toLocaleDateString();
}

export function PostCard({ post, onDelete }: PostCardProps) {
    const navigate = useNavigate();
    const { user } = useAuth();
    const { addWSListener } = useNotifications();
    const [liked, setLiked] = useState(post.user_liked);
    const [likeCount, setLikeCount] = useState(post.like_count);

    useEffect(() => {
        return addWSListener(msg => {
            if (msg.type === "post_like") {
                const data = msg.data as { post_id: string; delta: number };
                if (data.post_id === post.id) {
                    setLikeCount(c => c + data.delta);
                }
            }
        });
    }, [addWSListener, post.id]);

    async function handleLike() {
        if (!user) {
            return;
        }
        if (liked) {
            setLiked(false);
            setLikeCount(c => c - 1);
            await unlikePost(post.id).catch(() => {
                setLiked(true);
                setLikeCount(c => c + 1);
            });
        } else {
            setLiked(true);
            setLikeCount(c => c + 1);
            await likePost(post.id).catch(() => {
                setLiked(false);
                setLikeCount(c => c - 1);
            });
        }
    }

    async function handleDelete() {
        try {
            await apiDeletePost(post.id);
        } catch {
            void 0;
        }
        onDelete?.();
    }

    const isOwner = user?.id === post.author.id;
    const canDelete = isOwner || can(user?.role, "delete_any_post");

    return (
        <div className={styles.card}>
            <div className={styles.header}>
                <ProfileLink user={post.author} size="medium" />
                <span className={styles.time}>{timeAgo(post.created_at)}</span>
            </div>

            <div className={styles.body} onClick={() => navigate(`/game-board/${post.id}`)}>
                <p className={styles.text}>{post.body}</p>
                <MediaGallery media={post.media} />
            </div>

            <div className={styles.actions}>
                <Button variant="ghost" size="small" onClick={handleLike} disabled={!user}>
                    {liked ? "\u2665" : "\u2661"} {likeCount > 0 && likeCount}
                </Button>

                <Button variant="ghost" size="small" onClick={() => navigate(`/game-board/${post.id}`)}>
                    {"\uD83D\uDCAC"} {post.comment_count > 0 && post.comment_count}
                </Button>

                {post.view_count > 0 && <span className={styles.viewCount}>{post.view_count} views</span>}

                {canDelete && (
                    <Button variant="ghost" size="small" onClick={handleDelete}>
                        Delete
                    </Button>
                )}

                <Button
                    variant="ghost"
                    size="small"
                    onClick={() => navigator.clipboard.writeText(`${window.location.origin}/game-board/${post.id}`)}
                >
                    Copy Link
                </Button>
            </div>
        </div>
    );
}
