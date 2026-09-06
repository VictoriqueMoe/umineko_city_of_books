import { useCallback, useState } from "react";
import { useNavigate } from "react-router";
import type { CreatePollPayload } from "../../../types/api";
import { useCreatePost, useUploadPostMediaById } from "../../../hooks/mutations/post";
import { useStagedMedia } from "../../../hooks/useStagedMedia";
import { useSiteInfo } from "../../../hooks/useSiteInfo";
import { validateFileSize } from "../../../utils/fileValidation";
import { Button } from "../../Button/Button";
import { GifPicker } from "../../chat/GifPicker/GifPicker";
import { MediaPickerButton, MediaPreviews } from "../../MediaPicker/MediaPicker";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import { PollCreator } from "../PollCreator/PollCreator";
import styles from "./PostComposer.module.css";

interface PostComposerProps {
    corner?: string;
}

export function PostComposer({ corner = "general" }: PostComposerProps) {
    const navigate = useNavigate();
    const siteInfo = useSiteInfo();
    const [body, setBody] = useState("");
    const media = useStagedMedia();
    const addMedia = media.add;
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const [showPoll, setShowPoll] = useState(false);
    const [pollOptions, setPollOptions] = useState<string[]>(["", ""]);
    const [pollDuration, setPollDuration] = useState(86400);
    const [gifPickerOpen, setGifPickerOpen] = useState(false);
    const createPostMutation = useCreatePost();
    const uploadMediaMutation = useUploadPostMediaById();

    async function handleGifPick(gif: { url: string }) {
        setGifPickerOpen(false);
        if (submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            const { id } = await createPostMutation.mutateAsync({ body: gif.url, corner });
            navigate(`/game-board/${id}`);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to send GIF");
        } finally {
            setSubmitting(false);
        }
    }

    async function handleSubmit() {
        if (submitting || (!body.trim() && media.files.length === 0)) {
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
            const { id } = await createPostMutation.mutateAsync({ body: body.trim(), corner, poll: pollPayload });
            const mediaErrors: string[] = [];
            const failedFiles: File[] = [];
            for (const item of media.staged) {
                try {
                    await uploadMediaMutation.mutateAsync({ id, file: item.file, isSpoiler: item.isSpoiler });
                } catch (err) {
                    mediaErrors.push(err instanceof Error ? err.message : `Failed to upload ${item.file.name}`);
                    failedFiles.push(item.file);
                }
            }
            setBody("");
            media.keepOnly(failedFiles);
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

    const handlePasteFiles = useCallback(
        (pasted: File[]) => {
            const errors: string[] = [];
            const valid: File[] = [];
            for (const file of pasted) {
                const err = validateFileSize(
                    file,
                    siteInfo.max_image_size,
                    siteInfo.max_video_size,
                    siteInfo.max_audio_size,
                );
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
                addMedia(valid);
            }
        },
        [addMedia, siteInfo.max_image_size, siteInfo.max_video_size, siteInfo.max_audio_size],
    );

    return (
        <div className={styles.composer}>
            {error && <div className={styles.error}>{error}</div>}
            <MentionTextArea
                placeholder="What's on your mind?"
                value={body}
                onChange={setBody}
                rows={3}
                onPasteFiles={handlePasteFiles}
                showColours
            />

            <MediaPreviews
                files={media.files}
                onRemove={media.remove}
                onReorder={media.reorder}
                spoilers={media.spoilers}
                onToggleSpoiler={media.toggleSpoiler}
            />

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
                    <MediaPickerButton onFiles={media.add} onError={setError} />
                    <div className={styles.gifAnchor}>
                        <Button
                            variant="ghost"
                            size="small"
                            onClick={() => setGifPickerOpen(prev => !prev)}
                            disabled={submitting}
                        >
                            + GIF
                        </Button>
                        {gifPickerOpen && <GifPicker onPick={handleGifPick} onClose={() => setGifPickerOpen(false)} />}
                    </div>
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
                    disabled={submitting || (!body.trim() && media.files.length === 0)}
                >
                    {submitting ? "Posting..." : "Post"}
                </Button>
            </div>
        </div>
    );
}
