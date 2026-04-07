import React, { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import type { Post, PostMedia } from "../../../types/api";
import {
    deletePost as apiDeletePost,
    deletePostMedia,
    likePost,
    unlikePost,
    updatePost,
    uploadPostMedia,
} from "../../../api/endpoints";
import { useAuth } from "../../../hooks/useAuth";
import { useNotifications } from "../../../hooks/useNotifications";
import { can } from "../../../utils/permissions";
import { linkify } from "../../../utils/linkify";
import { ReportButton } from "../../ReportButton/ReportButton";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { MediaGallery } from "../MediaGallery/MediaGallery";
import { PollDisplay } from "../PollDisplay/PollDisplay";
import { PostEmbeds } from "../PostEmbeds/PostEmbeds";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import { Button } from "../../Button/Button";
import styles from "./PostCard.module.css";

interface PostCardProps {
    post: Post;
    onDelete?: () => void;
    onEdit?: () => void;
    extraActions?: React.ReactNode;
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

export function PostCard({ post, onDelete, onEdit, extraActions }: PostCardProps) {
    const { user } = useAuth();
    const { addWSListener } = useNotifications();
    const [liked, setLiked] = useState(post.user_liked);
    const [likeCount, setLikeCount] = useState(post.like_count);
    const [editing, setEditing] = useState(false);
    const [displayBody, setDisplayBody] = useState(post.body);
    const [editBody, setEditBody] = useState(post.body);
    const [editMedia, setEditMedia] = useState<PostMedia[]>(post.media);
    const [displayMedia, setDisplayMedia] = useState<PostMedia[]>(post.media);
    const [saving, setSaving] = useState(false);
    const mediaInputRef = useRef<HTMLInputElement>(null);

    const pendingLikeRef = useRef(false);

    useEffect(() => {
        return addWSListener(msg => {
            if (msg.type === "post_like") {
                const data = msg.data as { post_id: string; delta: number };
                if (data.post_id === post.id) {
                    if (pendingLikeRef.current) {
                        pendingLikeRef.current = false;
                        return;
                    }
                    setLikeCount(c => c + data.delta);
                }
            }
        });
    }, [addWSListener, post.id]);

    async function handleLike() {
        if (!user) {
            return;
        }
        pendingLikeRef.current = true;
        if (liked) {
            setLiked(false);
            setLikeCount(c => c - 1);
            await unlikePost(post.id).catch(() => {
                setLiked(true);
                setLikeCount(c => c + 1);
                pendingLikeRef.current = false;
            });
        } else {
            setLiked(true);
            setLikeCount(c => c + 1);
            await likePost(post.id).catch(() => {
                setLiked(false);
                setLikeCount(c => c - 1);
                pendingLikeRef.current = false;
            });
        }
    }

    async function handleDelete() {
        if (!window.confirm("Are you sure you want to delete this post?")) {
            return;
        }
        try {
            await apiDeletePost(post.id);
        } catch {}
        onDelete?.();
    }

    async function handleSaveEdit() {
        if (!editBody.trim() || saving) {
            return;
        }
        setSaving(true);
        try {
            await updatePost(post.id, editBody.trim());
            setDisplayBody(editBody.trim());
            setDisplayMedia([...editMedia]);
            setEditing(false);
            onEdit?.();
        } catch {
        } finally {
            setSaving(false);
        }
    }

    const isOwner = user?.id === post.author.id;
    const canEdit = isOwner || can(user?.role, "edit_any_post");
    const canDelete = isOwner || can(user?.role, "delete_any_post");

    return (
        <div className={styles.card}>
            <div className={styles.header}>
                <ProfileLink user={post.author} size="medium" />
                <span className={styles.time}>
                    {timeAgo(post.created_at)}
                    {post.updated_at && " (edited)"}
                </span>
            </div>

            {editing ? (
                <div className={styles.editArea}>
                    <MentionTextArea value={editBody} onChange={setEditBody} rows={3} />
                    {editMedia.length > 0 && (
                        <div className={styles.editMediaList}>
                            {editMedia.map(m => (
                                <div key={m.id} className={styles.editMediaItem}>
                                    <img src={m.media_url} alt="" className={styles.editMediaThumb} />
                                    <button
                                        type="button"
                                        className={styles.editMediaRemove}
                                        onClick={async () => {
                                            await deletePostMedia(post.id, m.id);
                                            setEditMedia(prev => prev.filter(x => x.id !== m.id));
                                        }}
                                    >
                                        &times;
                                    </button>
                                </div>
                            ))}
                        </div>
                    )}
                    <div className={styles.editActions}>
                        <input
                            ref={mediaInputRef}
                            type="file"
                            accept="image/*,video/*"
                            hidden
                            onChange={async e => {
                                const file = e.target.files?.[0];
                                if (!file) {
                                    return;
                                }
                                e.target.value = "";
                                const result = await uploadPostMedia(post.id, file);
                                setEditMedia(prev => [...prev, result]);
                            }}
                        />
                        <Button variant="ghost" size="small" onClick={() => mediaInputRef.current?.click()}>
                            + Media
                        </Button>
                        <Button variant="ghost" size="small" onClick={() => setEditing(false)}>
                            Cancel
                        </Button>
                        <Button
                            variant="primary"
                            size="small"
                            onClick={handleSaveEdit}
                            disabled={saving || !editBody.trim()}
                        >
                            {saving ? "..." : "Save"}
                        </Button>
                    </div>
                </div>
            ) : (
                <Link to={`/game-board/${post.id}`} className={styles.body}>
                    <p className={styles.text}>{linkify(displayBody)}</p>
                    {post.poll && <PollDisplay poll={post.poll} postId={post.id} onVoted={onEdit} />}
                    <MediaGallery media={displayMedia} />
                    {post.embeds && <PostEmbeds embeds={post.embeds} />}
                </Link>
            )}

            <div className={styles.actions}>
                <Button variant="ghost" size="small" onClick={handleLike} disabled={!user}>
                    {liked ? "\u2665" : "\u2661"} {likeCount > 0 && likeCount}
                </Button>

                <Link to={`/game-board/${post.id}`} style={{ textDecoration: "none" }}>
                    <Button variant="ghost" size="small">
                        {"\uD83D\uDCAC"} {post.comment_count > 0 && post.comment_count}
                    </Button>
                </Link>

                {canEdit && !editing && (
                    <Button variant="ghost" size="small" onClick={() => setEditing(true)}>
                        Edit
                    </Button>
                )}

                {canDelete && (
                    <Button variant="ghost" size="small" onClick={handleDelete}>
                        Delete
                    </Button>
                )}

                <span className={styles.spacer} />

                {post.view_count > 0 && <span className={styles.viewCount}>{post.view_count} views</span>}

                <Button
                    variant="ghost"
                    size="small"
                    onClick={() => navigator.clipboard.writeText(`${window.location.origin}/game-board/${post.id}`)}
                >
                    Copy Link
                </Button>

                {user && !isOwner && <ReportButton targetType="post" targetId={post.id} />}
                {extraActions}
            </div>
        </div>
    );
}
