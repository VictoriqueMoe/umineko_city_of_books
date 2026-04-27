import { useState } from "react";
import type { PostComment } from "../../../types/api";
import { useDeleteComment, useLikeComment, useUnlikeComment, useUpdateComment } from "../../../api/mutations/post";
import { useAuth } from "../../../hooks/useAuth";
import { can } from "../../../utils/permissions";
import { extractGif } from "../../../utils/gif";
import { renderRich } from "../../../utils/richText";
import { shortRelativeTime } from "../../../utils/time";
import { GifEmbed } from "../../GifEmbed/GifEmbed";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { MediaGallery } from "../MediaGallery/MediaGallery";
import { PostEmbeds } from "../PostEmbeds/PostEmbeds";
import { CommentComposer } from "../CommentComposer/CommentComposer";
import { Button } from "../../Button/Button";
import { ReportButton } from "../../ReportButton/ReportButton";
import styles from "./CommentItem.module.css";

type CreateCommentFn = (postId: string, body: string, parentId?: string) => Promise<{ id: string }>;
type UploadMediaFn = (commentId: string, file: File) => Promise<unknown>;

interface CommentItemProps {
    comment: PostComment;
    postId: string;
    onDelete: () => void;
    highlightedId?: string;
    isReply?: boolean;
    replyToName?: string;
    linkPrefix?: string;
    reportType?: string;
    likeFn?: (id: string) => Promise<void>;
    unlikeFn?: (id: string) => Promise<void>;
    deleteFn?: (id: string) => Promise<void>;
    updateFn?: (id: string, body: string) => Promise<void>;
    createCommentFn?: CreateCommentFn;
    uploadMediaFn?: UploadMediaFn;
    viewerBlocked?: boolean;
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

function SingleComment({
    comment,
    postId,
    onDelete,
    highlightedId,
    isReply,
    replyToName,
    linkPrefix = "/game-board",
    reportType = "comment",
    likeFn,
    unlikeFn,
    deleteFn,
    updateFn,
    createCommentFn,
    uploadMediaFn,
    viewerBlocked,
}: CommentItemProps) {
    const highlighted = highlightedId === comment.id;
    const { user } = useAuth();
    const isOwner = user?.id === comment.author.id;
    const canEditComment = isOwner || can(user?.role, "edit_any_comment");
    const canDeleteComment = isOwner || can(user?.role, "delete_any_comment");

    const likeMutation = useLikeComment(postId);
    const unlikeMutation = useUnlikeComment(postId);
    const deleteMutation = useDeleteComment(postId);
    const updateMutation = useUpdateComment(postId);

    const doLike = likeFn || ((id: string) => likeMutation.mutateAsync(id));
    const doUnlike = unlikeFn || ((id: string) => unlikeMutation.mutateAsync(id));
    const doDelete = deleteFn || ((id: string) => deleteMutation.mutateAsync(id));
    const doUpdate =
        updateFn || ((id: string, body: string) => updateMutation.mutateAsync({ commentId: id, body }).then(() => {}));

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
            await doUnlike(comment.id).catch(() => {
                setLiked(true);
                setLikeCount(c => c + 1);
            });
        } else {
            setLiked(true);
            setLikeCount(c => c + 1);
            await doLike(comment.id).catch(() => {
                setLiked(false);
                setLikeCount(c => c - 1);
            });
        }
    }

    async function handleDelete() {
        if (!window.confirm("Are you sure you want to delete this comment?")) {
            return;
        }
        await doDelete(comment.id);
        onDelete();
    }

    async function handleSaveEdit() {
        if (!editBody.trim() || saving) {
            return;
        }
        setSaving(true);
        try {
            await doUpdate(comment.id, editBody.trim());
            setEditing(false);
            onDelete();
        } catch {
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
                    {shortRelativeTime(comment.created_at)}
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
                    {(() => {
                        const gifURL = extractGif(comment.body);
                        if (gifURL) {
                            return <GifEmbed src={gifURL} imgClassName={styles.gifEmbed} />;
                        }
                        return <div className={styles.body}>{renderRich(comment.body)}</div>;
                    })()}
                    <MediaGallery media={comment.media} />
                    {comment.embeds && <PostEmbeds embeds={comment.embeds} />}
                </>
            )}

            <div className={styles.actions}>
                {!viewerBlocked && (
                    <Button variant="ghost" size="small" onClick={handleLike} disabled={!user}>
                        {liked ? "\u2665" : "\u2661"} {likeCount > 0 && likeCount}
                    </Button>
                )}

                {user && !viewerBlocked && (
                    <Button variant="ghost" size="small" onClick={() => setShowReply(!showReply)}>
                        Reply
                    </Button>
                )}

                {canEditComment && !editing && (
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
                            `${window.location.origin}${linkPrefix}/${postId}#comment-${comment.id}`,
                        )
                    }
                >
                    Copy Link
                </Button>

                {user && !isOwner && <ReportButton targetType={reportType} targetId={comment.id} contextId={postId} />}
            </div>

            {showReply && (
                <CommentComposer
                    postId={postId}
                    parentId={comment.id}
                    onCreated={() => {
                        setShowReply(false);
                        onDelete();
                    }}
                    createCommentFn={createCommentFn}
                    uploadMediaFn={uploadMediaFn}
                />
            )}
        </div>
    );
}

export function CommentItem({
    comment,
    postId,
    onDelete,
    highlightedId,
    linkPrefix,
    reportType,
    likeFn,
    unlikeFn,
    deleteFn,
    updateFn,
    createCommentFn,
    uploadMediaFn,
    viewerBlocked,
}: CommentItemProps) {
    const allReplies = flattenReplies(comment);
    const [collapsed, setCollapsed] = useState(false);

    const sharedProps = {
        postId,
        onDelete,
        linkPrefix,
        reportType,
        likeFn,
        unlikeFn,
        deleteFn,
        updateFn,
        createCommentFn,
        uploadMediaFn,
        viewerBlocked,
    };

    return (
        <div>
            <SingleComment comment={comment} highlightedId={highlightedId} {...sharedProps} />

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
                                    highlightedId={highlightedId}
                                    isReply
                                    replyToName={replyToName}
                                    {...sharedProps}
                                />
                            ))}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
