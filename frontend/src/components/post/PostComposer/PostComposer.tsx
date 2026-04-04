import { useState } from "react";
import { useNavigate } from "react-router";
import { createPost, uploadPostMedia } from "../../../api/endpoints";
import { Button } from "../../Button/Button";
import { MediaPickerButton, MediaPreviews } from "../../MediaPicker/MediaPicker";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import styles from "./PostComposer.module.css";

interface PostComposerProps {
    corner?: string;
}

export function PostComposer({ corner = "general" }: PostComposerProps) {
    const navigate = useNavigate();
    const [body, setBody] = useState("");
    const [files, setFiles] = useState<File[]>([]);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

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

    function removeFile(index: number) {
        setFiles(prev => prev.filter((_, i) => i !== index));
    }

    return (
        <div className={styles.composer}>
            {error && <div className={styles.error}>{error}</div>}
            <MentionTextArea placeholder="What's on your mind?" value={body} onChange={setBody} rows={3} />

            <MediaPreviews files={files} onRemove={removeFile} />

            <div className={styles.bar}>
                <MediaPickerButton onFiles={valid => setFiles(prev => [...prev, ...valid])} onError={setError} />
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
