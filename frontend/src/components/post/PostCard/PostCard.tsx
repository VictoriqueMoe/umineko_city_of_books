import React, { useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router";
import type { Post, PostMedia } from "../../../types/api";
import {
    useDeletePost,
    useDeletePostMedia,
    useLikePost,
    useUnlikePost,
    useUpdatePost,
    useUploadPostMedia,
} from "../../../api/mutations/post";
import { useAuth } from "../../../hooks/useAuth";
import { useNotifications } from "../../../hooks/useNotifications";
import { can } from "../../../utils/permissions";
import { extractGif } from "../../../utils/gif";
import { renderRich } from "../../../utils/richText";
import { shortRelativeTime } from "../../../utils/time";
import { GifEmbed } from "../../GifEmbed/GifEmbed";
import { ReportButton } from "../../ReportButton/ReportButton";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { MediaGallery } from "../MediaGallery/MediaGallery";
import { PollDisplay } from "../PollDisplay/PollDisplay";
import { PostEmbeds } from "../PostEmbeds/PostEmbeds";
import { SharedContentCard } from "../SharedContentCard/SharedContentCard";
import { ShareDialog } from "../ShareDialog/ShareDialog";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import { Button } from "../../Button/Button";
import { CommentComposer } from "../CommentComposer/CommentComposer";
import styles from "./PostCard.module.css";

interface PostCardProps {
    post: Post;
    onDelete?: () => void;
    onEdit?: () => void;
    extraActions?: React.ReactNode;
}

export function PostCard({ post, onDelete, onEdit, extraActions }: PostCardProps) {
    const navigate = useNavigate();
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
    const [shareOpen, setShareOpen] = useState(false);
    const [replyOpen, setReplyOpen] = useState(false);
    const mediaInputRef = useRef<HTMLInputElement>(null);

    const pendingLikeRef = useRef(false);
    const likeMutation = useLikePost();
    const unlikeMutation = useUnlikePost();
    const deleteMutation = useDeletePost();
    const updateMutation = useUpdatePost(post.id);
    const uploadMediaMutation = useUploadPostMedia(post.id);
    const deleteMediaMutation = useDeletePostMedia(post.id);

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
            await unlikeMutation.mutateAsync(post.id).catch(() => {
                setLiked(true);
                setLikeCount(c => c + 1);
                pendingLikeRef.current = false;
            });
        } else {
            setLiked(true);
            setLikeCount(c => c + 1);
            await likeMutation.mutateAsync(post.id).catch(() => {
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
            await deleteMutation.mutateAsync(post.id);
        } catch {
            // ignore
        }
        onDelete?.();
    }

    async function handleSaveEdit() {
        if (!editBody.trim() || saving) {
            return;
        }
        setSaving(true);
        try {
            await updateMutation.mutateAsync(editBody.trim());
            setDisplayBody(editBody.trim());
            setDisplayMedia([...editMedia]);
            setEditing(false);
            onEdit?.();
        } catch {
            // ignore
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
                    {shortRelativeTime(post.created_at)}
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
                                            await deleteMediaMutation.mutateAsync(m.id);
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
                                const result = await uploadMediaMutation.mutateAsync(file);
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
                <>
                    <div
                        className={styles.body}
                        onClick={e => {
                            if ((e.target as HTMLElement).closest("a")) {
                                return;
                            }
                            navigate(`/game-board/${post.id}`);
                        }}
                        onMouseDown={e => {
                            if (e.button === 1 && !(e.target as HTMLElement).closest("a")) {
                                e.preventDefault();
                            }
                        }}
                        onAuxClick={e => {
                            if (e.button === 1 && !(e.target as HTMLElement).closest("a")) {
                                e.preventDefault();
                                window.open(`/game-board/${post.id}`, "_blank");
                            }
                        }}
                    >
                        {(() => {
                            const gifURL = extractGif(displayBody);
                            if (gifURL) {
                                return <GifEmbed src={gifURL} imgClassName={styles.gifEmbed} />;
                            }
                            return <div className={styles.text}>{renderRich(displayBody)}</div>;
                        })()}
                        <MediaGallery media={displayMedia} />
                        {post.embeds && <PostEmbeds embeds={post.embeds} />}
                        {post.shared_content && <SharedContentCard content={post.shared_content} />}
                    </div>
                    {post.poll && <PollDisplay poll={post.poll} postId={post.id} onVoted={onEdit} />}
                </>
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

                {user && (
                    <Button variant="ghost" size="small" onClick={() => setReplyOpen(prev => !prev)}>
                        {"\u21B6"} Reply
                    </Button>
                )}

                {user && (
                    <Button variant="ghost" size="small" onClick={() => setShareOpen(true)}>
                        Share {post.share_count > 0 && post.share_count}
                    </Button>
                )}

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

            {user && replyOpen && (
                <div className={styles.quickReply}>
                    <CommentComposer
                        postId={post.id}
                        onCreated={() => {
                            setReplyOpen(false);
                            onEdit?.();
                        }}
                    />
                </div>
            )}

            {shareOpen && (
                <ShareDialog
                    isOpen={shareOpen}
                    onClose={() => setShareOpen(false)}
                    contentId={post.id}
                    contentType="post"
                    contentTitle={post.body.slice(0, 50)}
                    onShared={onEdit}
                />
            )}
        </div>
    );
}
