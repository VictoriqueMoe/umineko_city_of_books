import type { PostComment, UserProfile } from "../../../types/api";
import { CommentItem } from "../CommentItem/CommentItem";
import { CommentComposer } from "../CommentComposer/CommentComposer";
import styles from "./CommentsSection.module.css";

type CreateCommentFn = (targetId: string, body: string, parentId?: string) => Promise<{ id: string }>;
type UploadMediaFn = (commentId: string, file: File) => Promise<unknown>;

interface CommentsSectionProps {
    comments: PostComment[] | null | undefined;
    targetId: string;
    user: UserProfile | null | undefined;
    onChanged: () => void;
    title?: string;
    emptyText?: string | null;
    blockedText?: string;
    viewerBlocked?: boolean;
    highlightedId?: string;
    linkPrefix?: string;
    reportType?: string;
    showComposer?: boolean;
    composerPosition?: "top" | "bottom";
    likeFn?: (id: string) => Promise<void>;
    unlikeFn?: (id: string) => Promise<void>;
    deleteFn?: (id: string) => Promise<void>;
    updateFn?: (id: string, body: string) => Promise<void>;
    createCommentFn?: CreateCommentFn;
    uploadMediaFn?: UploadMediaFn;
}

export function CommentsSection({
    comments,
    targetId,
    user,
    onChanged,
    title = "Comments",
    emptyText = "No comments yet.",
    blockedText = "You cannot interact with this post.",
    viewerBlocked,
    highlightedId,
    linkPrefix,
    reportType,
    showComposer = true,
    composerPosition = "bottom",
    likeFn,
    unlikeFn,
    deleteFn,
    updateFn,
    createCommentFn,
    uploadMediaFn,
}: CommentsSectionProps) {
    const list = comments ?? [];
    const count = list.length;
    const composer = showComposer && user && !viewerBlocked && (
        <CommentComposer
            postId={targetId}
            onCreated={onChanged}
            createCommentFn={createCommentFn}
            uploadMediaFn={uploadMediaFn}
        />
    );

    return (
        <div className={styles.section}>
            <h3 className={styles.title}>
                {title} {count > 0 && `(${count})`}
            </h3>
            {composerPosition === "top" && composer}
            {list.map(c => (
                <CommentItem
                    key={c.id}
                    comment={c}
                    postId={targetId}
                    onDelete={onChanged}
                    highlightedId={highlightedId}
                    linkPrefix={linkPrefix}
                    reportType={reportType}
                    likeFn={likeFn}
                    unlikeFn={unlikeFn}
                    deleteFn={deleteFn}
                    updateFn={updateFn}
                    createCommentFn={createCommentFn}
                    uploadMediaFn={uploadMediaFn}
                    viewerBlocked={viewerBlocked}
                />
            ))}
            {count === 0 && emptyText && <p className={styles.empty}>{emptyText}</p>}
            {viewerBlocked && <p className={styles.empty}>{blockedText}</p>}
            {composerPosition === "bottom" && composer}
        </div>
    );
}
