import { useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { createPost, uploadPostMedia } from "../../../api/endpoints";
import { useSiteInfo } from "../../../hooks/useSiteInfo";
import { validateFileSize } from "../../../utils/fileValidation";
import { Button } from "../../Button/Button";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import styles from "./PostComposer.module.css";

interface PostComposerProps {
    corner?: string;
}

export function PostComposer({ corner = "general" }: PostComposerProps) {
    const navigate = useNavigate();
    const siteInfo = useSiteInfo();
    const [body, setBody] = useState("");
    const [files, setFiles] = useState<File[]>([]);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const fileInputRef = useRef<HTMLInputElement>(null);

    async function handleSubmit() {
        if (submitting || (!body.trim() && files.length === 0)) {
            return;
        }
        setSubmitting(true);
        setError("");

        try {
            const { id } = await createPost(body.trim(), corner);
            const mediaErrors: string[] = [];
            for (const file of files) {
                try {
                    await uploadPostMedia(id, file);
                } catch (err) {
                    mediaErrors.push(err instanceof Error ? err.message : `Failed to upload ${file.name}`);
                }
            }
            setBody("");
            setFiles([]);
            if (mediaErrors.length > 0) {
                setError(mediaErrors.join(", "));
            } else {
                navigate(`/game-board/${id}`);
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to create post");
        } finally {
            setSubmitting(false);
        }
    }

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

    const previews = useMemo(() => files.map(f => URL.createObjectURL(f)), [files]);

    return (
        <div className={styles.composer}>
            {error && <div className={styles.error}>{error}</div>}
            <MentionTextArea placeholder="What's on your mind?" value={body} onChange={setBody} rows={3} />

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
                    {submitting ? "Posting..." : "Post"}
                </Button>
            </div>
        </div>
    );
}
