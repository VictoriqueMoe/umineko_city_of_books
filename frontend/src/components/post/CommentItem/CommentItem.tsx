import {useState} from "react";
import type {PostComment} from "../../../types/api";
import {deleteComment as apiDeleteComment, likeComment, unlikeComment, updateComment} from "../../../api/endpoints";
import {useAuth} from "../../../hooks/useAuth";
import {can} from "../../../utils/permissions";
import {linkify} from "../../../utils/linkify";
import {ProfileLink} from "../../ProfileLink/ProfileLink";
import {MediaGallery} from "../MediaGallery/MediaGallery";
import {PostEmbeds} from "../PostEmbeds/PostEmbeds";
import {CommentComposer} from "../CommentComposer/CommentComposer";
import {Button} from "../../Button/Button";
import styles from "./CommentItem.module.css";

interface CommentItemProps {
    comment: PostComment;
    postId: string;
    onDelete: () => void;
    highlighted?: boolean;
    isReply?: boolean;
    replyToName?: string;
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
    return `${days}d`;
}

function flattenReplies(comment: PostComment): { reply: PostComment; replyToName: string }[] {
    const result: { reply: PostComment; replyToName: string }[] = [];

    function walk(c: PostComment, parentName: string) {
        for (const reply of c.replies ?? []) {
            result.push({ reply, replyToName: parentName });
            walk(reply, reply.author.display_name);
        }
    }

    walk(comment, comment.author.display_name);
    return result;
}

function SingleComment({ comment, postId, onDelete, highlighted, isReply, replyToName }: CommentItemProps) {
    const { user } = useAuth();
    const isOwner = user?.id === comment.author.id;
    const canDeleteComment = isOwner || can(user?.role, "delete_any_comment");

    const [liked, setLiked] = useState(comment.user_liked);
    const [likeCount, setLikeCount] = useState(comment.like_count);
    const [showReply, setShowReply] = useState(false);
    const [editing, setEditing] = useState(false);
    const [editBody, setEditBody] = useState(comment.body);
    const [saving, setSaving] = useState(false);

    async function handleLike() {
        if (!user) {
            return;
        }
        if (liked) {
            setLiked(false);
            setLikeCount(c => c - 1);
            await unlikeComment(comment.id).catch(() => {
                setLiked(true);
                setLikeCount(c => c + 1);
            });
        } else {
            setLiked(true);
            setLikeCount(c => c + 1);
            await likeComment(comment.id).catch(() => {
                setLiked(false);
                setLikeCount(c => c - 1);
            });
        }
    }

    async function handleDelete() {
        if (!window.confirm("Are you sure you want to delete this comment?")) {
            return;
        }
        await apiDeleteComment(comment.id);
        onDelete();
    }

    async function handleSaveEdit() {
        if (!editBody.trim() || saving) {
            return;
        }
        setSaving(true);
        try {
            await updateComment(comment.id, editBody.trim());
            setEditing(false);
            onDelete();
        } catch {
            void 0;
        } finally {
            setSaving(false);
        }
    }

    return (
        <div
            id={`comment-${comment.id}`}
            className={`${styles.comment}${highlighted ? ` ${styles.highlighted}` : ""}${isReply ? ` ${styles.reply}` : ""}`}
        >
            <div className={styles.header}>
                <ProfileLink user={comment.author} size="small" />
                {replyToName && <span className={styles.replyTo}>@{replyToName}</span>}
                <span className={styles.time}>
                    {timeAgo(comment.created_at)}
                    {comment.updated_at && " (edited)"}
                </span>
            </div>

            {editing ? (
                <div className={styles.editArea}>
                    <textarea
                        className={styles.editTextarea}
                        value={editBody}
                        onChange={e => setEditBody(e.target.value)}
                        rows={2}
                    />
                    <div className={styles.editActions}>
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
                    <p className={styles.body}>{linkify(comment.body)}</p>
                    <MediaGallery media={comment.media} />
                    {comment.embeds && <PostEmbeds embeds={comment.embeds} />}
                </>
            )}

            <div className={styles.actions}>
                <Button variant="ghost" size="small" onClick={handleLike} disabled={!user}>
                    {liked ? "\u2665" : "\u2661"} {likeCount > 0 && likeCount}
                </Button>

                {user && (
                    <Button variant="ghost" size="small" onClick={() => setShowReply(!showReply)}>
                        Reply
                    </Button>
                )}

                {isOwner && !editing && (
                    <Button variant="ghost" size="small" onClick={() => setEditing(true)}>
                        Edit
                    </Button>
                )}

                {canDeleteComment && (
                    <Button variant="ghost" size="small" onClick={handleDelete}>
                        Delete
                    </Button>
                )}

                <Button
                    variant="ghost"
                    size="small"
                    className={styles.copyLink}
                    onClick={() =>
                        navigator.clipboard.writeText(
                            `${window.location.origin}/game-board/${postId}#comment-${comment.id}`,
                        )
                    }
                >
                    Copy Link
                </Button>
            </div>

            {showReply && (
                <CommentComposer
                    postId={postId}
                    parentId={comment.id}
                    onCreated={() => {
                        setShowReply(false);
                        onDelete();
                    }}
                />
            )}
        </div>
    );
}

export function CommentItem({ comment, postId, onDelete, highlighted }: CommentItemProps) {
    const allReplies = flattenReplies(comment);
    const [collapsed, setCollapsed] = useState(false);

    return (
        <div>
            <SingleComment
                comment={comment}
                postId={postId}
                onDelete={onDelete}
                highlighted={highlighted === true || undefined}
            />

            {allReplies.length > 0 && (
                <div className={styles.threadContainer}>
                    <button className={styles.collapseBtn} onClick={() => setCollapsed(!collapsed)}>
                        {collapsed
                            ? `Show ${allReplies.length} ${allReplies.length === 1 ? "reply" : "replies"}`
                            : `Hide ${allReplies.length} ${allReplies.length === 1 ? "reply" : "replies"}`}
                    </button>

                    {!collapsed && (
                        <div className={styles.thread}>
                            {allReplies.map(({ reply, replyToName }) => (
                                <SingleComment
                                    key={reply.id}
                                    comment={reply}
                                    postId={postId}
                                    onDelete={onDelete}
                                    highlighted={highlighted === true || undefined}
                                    isReply
                                    replyToName={replyToName}
                                />
                            ))}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
