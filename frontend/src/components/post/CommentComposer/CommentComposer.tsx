import { useCallback, useState } from "react";
import { createComment, uploadCommentMedia } from "../../../api/endpoints";
import { useSiteInfo } from "../../../hooks/useSiteInfo";
import { validateFileSize } from "../../../utils/fileValidation";
import { Button } from "../../Button/Button";
import { MediaPickerButton, MediaPreviews } from "../../MediaPicker/MediaPicker";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import styles from "./CommentComposer.module.css";

type CreateCommentFn = (postId: string, body: string, parentId?: string) => Promise<{ id: string }>;
type UploadMediaFn = (commentId: string, file: File) => Promise<unknown>;

interface CommentComposerProps {
    postId: string;
    parentId?: string;
    onCreated: () => void;
    createCommentFn?: CreateCommentFn;
    uploadMediaFn?: UploadMediaFn;
}

export function CommentComposer({ postId, parentId, onCreated, createCommentFn, uploadMediaFn }: CommentComposerProps) {
    const siteInfo = useSiteInfo();
    const [body, setBody] = useState("");
    const [files, setFiles] = useState<File[]>([]);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    function removeFile(index: number) {
        setFiles(prev => prev.filter((_, i) => i !== index));
    }

    const handlePasteFiles = useCallback(
        (pasted: File[]) => {
            const errors: string[] = [];
            const valid: File[] = [];
            for (const file of pasted) {
                const err = validateFileSize(file, siteInfo.max_image_size, siteInfo.max_video_size);
                if (err) {
                    errors.push(err);
                } else {
                    valid.push(file);
                }
            }
            if (errors.length > 0) {
                setError(errors.join(" "));
            }
            if (valid.length > 0) {
                setFiles(prev => [...prev, ...valid]);
            }
        },
        [siteInfo.max_image_size, siteInfo.max_video_size],
    );

    async function handleSubmit() {
        if ((!body.trim() && files.length === 0) || submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            const doCreate = createCommentFn || createComment;
            const doUpload = uploadMediaFn || uploadCommentMedia;
            const { id } = await doCreate(postId, body.trim(), parentId);
            for (const file of files) {
                try {
                    await doUpload(id, file);
                } catch (err) {
                    setError(err instanceof Error ? err.message : "Failed to upload media");
                }
            }
            setBody("");
            setFiles([]);
            onCreated();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to post comment");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <div className={styles.composer}>
            {error && <div className={styles.error}>{error}</div>}
            <MentionTextArea
                placeholder={parentId ? "Write a reply..." : "Write a comment..."}
                value={body}
                onChange={setBody}
                rows={2}
                onPasteFiles={handlePasteFiles}
            />

            <MediaPreviews files={files} onRemove={removeFile} size="small" />

            <div className={styles.bar}>
                <MediaPickerButton onFiles={valid => setFiles(prev => [...prev, ...valid])} onError={setError} />
                <Button
                    variant="primary"
                    size="small"
                    onClick={handleSubmit}
                    disabled={submitting || (!body.trim() && files.length === 0)}
                >
                    {submitting ? "..." : parentId ? "Reply" : "Comment"}
                </Button>
            </div>
        </div>
    );
}
