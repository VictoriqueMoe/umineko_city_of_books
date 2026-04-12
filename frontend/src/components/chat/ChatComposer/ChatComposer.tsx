import React, { useCallback, useState } from "react";
import { Button } from "../../Button/Button";
import { MediaPickerButton, MediaPreviews } from "../../MediaPicker/MediaPicker";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import { sendChatMessage, sendFirstDMMessage, uploadChatMessageMedia } from "../../../api/endpoints";
import { useSiteInfo } from "../../../hooks/useSiteInfo";
import { validateFileSize } from "../../../utils/fileValidation";
import type { ChatMessage, ChatRoom, PostMedia, User } from "../../../types/api";
import styles from "./ChatComposer.module.css";

export interface ReplyTarget {
    id: string;
    senderName: string;
    bodyPreview: string;
}

interface ChatComposerProps {
    roomId: string | null;
    draftRecipientId: string | null;
    onSent: (message: ChatMessage, room?: ChatRoom) => void;
    mentionPool?: User[];
    replyingTo?: ReplyTarget | null;
    onCancelReply?: () => void;
}

export function ChatComposer({
    roomId,
    draftRecipientId,
    onSent,
    mentionPool,
    replyingTo,
    onCancelReply,
}: ChatComposerProps) {
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
            for (let i = 0; i < pasted.length; i++) {
                const err = validateFileSize(pasted[i], siteInfo.max_image_size, siteInfo.max_video_size);
                if (err) {
                    errors.push(err);
                } else {
                    valid.push(pasted[i]);
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

    async function uploadAttached(messageId: string): Promise<PostMedia[]> {
        const uploaded: PostMedia[] = [];
        for (let i = 0; i < files.length; i++) {
            try {
                const m = await uploadChatMessageMedia(messageId, files[i]);
                uploaded.push(m);
            } catch (err) {
                setError(err instanceof Error ? err.message : "Failed to upload media");
            }
        }
        return uploaded;
    }

    async function handleSubmit() {
        const trimmed = body.trim();
        if ((!trimmed && files.length === 0) || submitting) {
            return;
        }
        if (!roomId && !draftRecipientId) {
            return;
        }

        setSubmitting(true);
        setError("");
        try {
            if (draftRecipientId && !roomId) {
                const created = await sendFirstDMMessage(draftRecipientId, trimmed || " ");
                let message = created.message;
                if (files.length > 0) {
                    const uploaded = await uploadAttached(created.message.id);
                    message = { ...message, media: uploaded };
                }
                onSent(message, created.room);
            } else {
                let message = await sendChatMessage(roomId!, {
                    body: trimmed || " ",
                    reply_to_id: replyingTo?.id,
                });
                if (files.length > 0) {
                    const uploaded = await uploadAttached(message.id);
                    message = { ...message, media: uploaded };
                }
                onSent(message);
            }
            setBody("");
            setFiles([]);
            if (onCancelReply) {
                onCancelReply();
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to send message");
        } finally {
            setSubmitting(false);
        }
    }

    function handleKeyDown(e: React.KeyboardEvent<HTMLDivElement>) {
        if (e.defaultPrevented) {
            return;
        }
        if (e.key !== "Enter") {
            return;
        }
        if (e.shiftKey) {
            return;
        }
        if (e.nativeEvent.isComposing) {
            return;
        }
        e.preventDefault();
        handleSubmit();
    }

    const canSend = !submitting && (body.trim().length > 0 || files.length > 0);

    return (
        <div className={styles.composer}>
            {error && <div className={styles.error}>{error}</div>}
            {replyingTo && (
                <div className={styles.replyBar}>
                    <div className={styles.replyContent}>
                        <span className={styles.replyLabel}>Replying to {replyingTo.senderName}</span>
                        <span className={styles.replyPreview}>{replyingTo.bodyPreview}</span>
                    </div>
                    {onCancelReply && (
                        <button className={styles.replyCancel} onClick={onCancelReply} aria-label="Cancel reply">
                            ✕
                        </button>
                    )}
                </div>
            )}
            {files.length > 0 && (
                <div className={styles.previews}>
                    <MediaPreviews files={files} onRemove={removeFile} size="small" />
                </div>
            )}
            <div className={styles.row} onKeyDown={handleKeyDown}>
                <div className={styles.textareaWrapper}>
                    <MentionTextArea
                        placeholder="Type a message... (Enter to send, Shift+Enter for newline)"
                        value={body}
                        onChange={setBody}
                        rows={1}
                        onPasteFiles={handlePasteFiles}
                        mentionPool={mentionPool}
                    />
                </div>
                <MediaPickerButton onFiles={valid => setFiles(prev => [...prev, ...valid])} onError={setError} />
                <Button variant="primary" size="small" onClick={handleSubmit} disabled={!canSend}>
                    {submitting ? "..." : "Send"}
                </Button>
            </div>
        </div>
    );
}
