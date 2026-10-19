import { useCallback, useState } from "react";
import { useCreateComment } from "../../../hooks/mutations/post";
import type { CreateCommentHandler } from "../../../hooks/useCommentHandlers";
import { useResetOnChange } from "../../../hooks/useResetOnChange";
import { useSiteInfo } from "../../../hooks/useSiteInfo";
import { validateFileSize } from "../../../utils/fileValidation";
import { Button } from "../../Button/Button";
import { GifPicker } from "../../chat/GifPicker/GifPicker";
import { useStagedMedia } from "../../../hooks/useStagedMedia";
import { MediaPickerButton, MediaPreviews } from "../../MediaPicker/MediaPicker";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import styles from "./CommentComposer.module.css";

interface CommentComposerProps {
    postId: string;
    parentId?: string;
    onCreated: () => void;
    createCommentFn?: CreateCommentHandler;
}

export function CommentComposer({ postId, parentId, onCreated, createCommentFn }: CommentComposerProps) {
    const siteInfo = useSiteInfo();
    const [body, setBody] = useState("");
    const media = useStagedMedia();
    const addMedia = media.add;
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const [gifPickerOpen, setGifPickerOpen] = useState(false);

    useResetOnChange(postId, () => {
        setBody("");
        setSubmitting(false);
        setError("");
        setGifPickerOpen(false);
        media.reset();
    });

    const createCommentMutation = useCreateComment(postId);
    const defaultCreate: CreateCommentHandler = (_postId, b, parent, staged) =>
        createCommentMutation.mutateAsync({ body: b, parentId: parent, media: staged });

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

    async function handleGifPick(gif: { url: string }) {
        setGifPickerOpen(false);
        if (submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            const doCreate = createCommentFn || defaultCreate;
            await doCreate(postId, gif.url, parentId);
            onCreated();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to send GIF");
        } finally {
            setSubmitting(false);
        }
    }

    async function handleSubmit() {
        if ((!body.trim() && media.files.length === 0) || submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            const doCreate = createCommentFn || defaultCreate;
            await doCreate(postId, body.trim(), parentId, media.staged);

            setBody("");
            media.reset();
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
                onSubmitShortcut={handleSubmit}
                showColours
            />

            <MediaPreviews
                files={media.files}
                onRemove={media.remove}
                onReorder={media.reorder}
                spoilers={media.spoilers}
                onToggleSpoiler={media.toggleSpoiler}
                size="small"
            />

            <div className={styles.bar}>
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
                <Button
                    variant="primary"
                    size="small"
                    onClick={handleSubmit}
                    disabled={submitting || (!body.trim() && media.files.length === 0)}
                >
                    {submitting ? "..." : parentId ? "Reply" : "Comment"}
                </Button>
            </div>
        </div>
    );
}
