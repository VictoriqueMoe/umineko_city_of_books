import { useMemo, useRef, useState } from "react";
import { createComment, uploadCommentMedia } from "../../../api/endpoints";
import { useSiteInfo } from "../../../hooks/useSiteInfo";
import { validateFileSize } from "../../../utils/fileValidation";
import { Button } from "../../Button/Button";
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
    const fileInputRef = useRef<HTMLInputElement>(null);

    const previews = useMemo(() => files.map(f => URL.createObjectURL(f)), [files]);

    function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
        if (e.target.files) {
            const newFiles = Array.from(e.target.files);
            const errors: string[] = [];
            const valid: File[] = [];

            for (const file of newFiles) {
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
        }
        e.target.value = "";
    }

    function removeFile(index: number) {
        setFiles(prev => prev.filter((_, i) => i !== index));
    }

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
        } catch {
            void 0;
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
            />

            {files.length > 0 && (
                <div className={styles.previews}>
                    {files.map((file, i) => (
                        <div key={i} className={styles.preview}>
                            {file.type.startsWith("video/") ? (
                                <video className={styles.previewMedia} src={previews[i]} />
                            ) : (
                                <img className={styles.previewMedia} src={previews[i]} alt="" />
                            )}
                            <button className={styles.previewRemove} onClick={() => removeFile(i)}>
                                x
                            </button>
                        </div>
                    ))}
                </div>
            )}

            <div className={styles.bar}>
                <input
                    ref={fileInputRef}
                    type="file"
                    accept="image/*,video/*,.mkv,.avi"
                    multiple
                    onChange={handleFileChange}
                    hidden
                />
                <Button variant="ghost" size="small" onClick={() => fileInputRef.current?.click()}>
                    + Media
                </Button>
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
