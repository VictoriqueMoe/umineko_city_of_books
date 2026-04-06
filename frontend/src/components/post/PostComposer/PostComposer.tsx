import { useState } from "react";
import { useNavigate } from "react-router";
import type { CreatePollPayload } from "../../../api/endpoints";
import { createPost, uploadPostMedia } from "../../../api/endpoints";
import { Button } from "../../Button/Button";
import { MediaPickerButton, MediaPreviews } from "../../MediaPicker/MediaPicker";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import { PollCreator } from "../PollCreator/PollCreator";
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
    const [showPoll, setShowPoll] = useState(false);
    const [pollOptions, setPollOptions] = useState<string[]>(["", ""]);
    const [pollDuration, setPollDuration] = useState(86400);

    async function handleSubmit() {
        if (submitting || (!body.trim() && files.length === 0)) {
            return;
        }
        if (showPoll) {
            const validOptions = pollOptions.filter(o => o.trim());
            if (validOptions.length < 2) {
                setError("Poll needs at least 2 non-empty options");
                return;
            }
        }
        setSubmitting(true);
        setError("");

        try {
            let pollPayload: CreatePollPayload | undefined;
            if (showPoll) {
                pollPayload = {
                    options: pollOptions.filter(o => o.trim()).map(label => ({ label: label.trim() })),
                    duration_seconds: pollDuration,
                };
            }
            const { id } = await createPost(body.trim(), corner, pollPayload);
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
            setShowPoll(false);
            setPollOptions(["", ""]);
            setPollDuration(86400);
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

            {showPoll && (
                <PollCreator
                    options={pollOptions}
                    onOptionsChange={setPollOptions}
                    duration={pollDuration}
                    onDurationChange={setPollDuration}
                    onRemove={() => {
                        setShowPoll(false);
                        setPollOptions(["", ""]);
                        setPollDuration(86400);
                    }}
                />
            )}

            <div className={styles.bar}>
                <div className={styles.barLeft}>
                    <MediaPickerButton onFiles={valid => setFiles(prev => [...prev, ...valid])} onError={setError} />
                    {!showPoll && (
                        <Button variant="ghost" size="small" onClick={() => setShowPoll(true)}>
                            + Poll
                        </Button>
                    )}
                </div>
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
