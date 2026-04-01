import { useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { createPost, uploadPostMedia } from "../../../api/endpoints";
import { Button } from "../../Button/Button";
import { TextArea } from "../../TextArea/TextArea";
import styles from "./PostComposer.module.css";

export function PostComposer() {
    const navigate = useNavigate();
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
            const { id } = await createPost(body.trim());
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
            setFiles(prev => [...prev, ...Array.from(e.target.files!)]);
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
            <TextArea
                placeholder="What's on your mind?"
                value={body}
                onChange={e => setBody(e.target.value)}
                rows={3}
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
                    {submitting ? "Posting..." : "Post"}
                </Button>
            </div>
        </div>
    );
}
