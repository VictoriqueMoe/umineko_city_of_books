import type {PostComment} from "../../../types/api";
import {deleteComment as apiDeleteComment} from "../../../api/endpoints";
import {useAuth} from "../../../hooks/useAuth";
import {can} from "../../../utils/permissions";
import {ProfileLink} from "../../ProfileLink/ProfileLink";
import {MediaGallery} from "../MediaGallery/MediaGallery";
import {Button} from "../../Button/Button";
import styles from "./CommentItem.module.css";

interface CommentItemProps {
    comment: PostComment;
    postId: string;
    onDelete: () => void;
    highlighted?: boolean;
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

export function CommentItem({ comment, postId, onDelete, highlighted }: CommentItemProps) {
    const { user } = useAuth();
    const isOwner = user?.id === comment.author.id;
    const canDelete = isOwner || can(user?.role, "delete_any_comment");

    async function handleDelete() {
        await apiDeleteComment(comment.id);
        onDelete();
    }

    return (
        <div id={`comment-${comment.id}`} className={`${styles.comment}${highlighted ? ` ${styles.highlighted}` : ""}`}>
            <div className={styles.header}>
                <ProfileLink user={comment.author} size="small" />
                <span className={styles.time}>{timeAgo(comment.created_at)}</span>
            </div>
            <p className={styles.body}>{comment.body}</p>
            <MediaGallery media={comment.media} />
            <div className={styles.actions}>
                {canDelete && (
                    <Button variant="ghost" size="small" onClick={handleDelete}>
                        Delete
                    </Button>
                )}
                <Button
                    variant="ghost"
                    size="small"
                    onClick={() =>
                        navigator.clipboard.writeText(
                            `${window.location.origin}/game-board/${postId}#comment-${comment.id}`,
                        )
                    }
                >
                    Copy Link
                </Button>
            </div>
        </div>
    );
}
