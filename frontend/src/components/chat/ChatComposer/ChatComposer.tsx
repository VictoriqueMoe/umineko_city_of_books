import React, { useCallback, useEffect, useRef, useState } from "react";
import { Button } from "../../Button/Button";
import { MediaPickerButton, MediaPreviews } from "../../MediaPicker/MediaPicker";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import { useSendChatMessage, useSendFirstDMMessage } from "../../../api/mutations/chat";
import { ApiError } from "../../../api/client";
import { useSiteInfo } from "../../../hooks/useSiteInfo";
import { validateFileSize } from "../../../utils/fileValidation";
import { formatFullDateTime, parseServerDate } from "../../../utils/time";
import type { ChatMessage, ChatRoom, User } from "../../../types/api";
import { GifPicker } from "../GifPicker/GifPicker";
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
    onTyping?: () => void;
    onEditLast?: () => void;
    timeoutUntil?: string;
}

function formatSendError(err: unknown): string {
    if (err instanceof ApiError) {
        const body = err.body as { code?: string; pattern?: string; action?: string; error?: string } | null;
        if (body?.code === "banned_word" && body.pattern) {
            const suffix = body.action === "kick" ? " You have been kicked from this room." : "";
            return `Message blocked by banned-word rule "${body.pattern}".${suffix}`;
        }
        if (body?.error) {
            return body.error;
        }
    }
    if (err instanceof Error) {
        return err.message;
    }
    return "Failed to send message";
}

function isTimeoutActive(until?: string): boolean {
    const d = parseServerDate(until);
    if (!d) {
        return false;
    }
    return d.getTime() > Date.now();
}

const TYPING_THROTTLE_MS = 2500;

export function ChatComposer({
    roomId,
    draftRecipientId,
    onSent,
    mentionPool,
    replyingTo,
    onCancelReply,
    onTyping,
    onEditLast,
    timeoutUntil,
}: ChatComposerProps) {
    const [, setTimeoutTick] = useState(0);
    const timedOut = isTimeoutActive(timeoutUntil);
    const siteInfo = useSiteInfo();
    const [body, setBody] = useState("");
    const [files, setFiles] = useState<File[]>([]);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const sendChatMessageMutation = useSendChatMessage(roomId ?? "");
    const sendFirstDMMessageMutation = useSendFirstDMMessage();

    useEffect(() => {
        if (!timeoutUntil) {
            return;
        }
        const parsed = parseServerDate(timeoutUntil);
        if (!parsed) {
            return;
        }
        const ms = parsed.getTime() - Date.now();
        if (ms <= 0) {
            return;
        }
        const timer = setTimeout(() => setTimeoutTick(t => t + 1), ms);
        return () => clearTimeout(timer);
    }, [timeoutUntil]);
    const [gifPickerOpen, setGifPickerOpen] = useState(false);
    const lastTypingSentRef = useRef(0);

    const handleBodyChange = useCallback(
        (value: string) => {
            setBody(value);
            if (onTyping && value.length > 0) {
                const now = Date.now();
                if (now - lastTypingSentRef.current >= TYPING_THROTTLE_MS) {
                    lastTypingSentRef.current = now;
                    onTyping();
                }
            }
        },
        [onTyping],
    );

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

    async function sendBody(content: string): Promise<ChatMessage | null> {
        if (draftRecipientId && !roomId) {
            const created = await sendFirstDMMessageMutation.mutateAsync({
                recipientId: draftRecipientId,
                body: content,
            });
            onSent(created.message, created.room);
            return created.message;
        }
        if (!roomId) {
            return null;
        }
        const message = await sendChatMessageMutation.mutateAsync({
            body: content,
            reply_to_id: replyingTo?.id,
        });
        onSent(message);
        return message;
    }

    async function handleGifPick(gif: { id: string; url: string }) {
        setGifPickerOpen(false);
        if (submitting) {
            return;
        }
        if (!roomId && !draftRecipientId) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            await sendBody(gif.url);
            if (onCancelReply) {
                onCancelReply();
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to send GIF");
        } finally {
            setSubmitting(false);
        }
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
                const created = await sendFirstDMMessageMutation.mutateAsync({
                    recipientId: draftRecipientId,
                    body: trimmed,
                    files,
                });
                onSent(created.message, created.room);
            } else {
                const message = await sendChatMessageMutation.mutateAsync({
                    body: trimmed,
                    reply_to_id: replyingTo?.id,
                    files,
                });
                onSent(message);
            }
            setBody("");
            setFiles([]);
            if (onCancelReply) {
                onCancelReply();
            }
        } catch (err) {
            setError(formatSendError(err));
        } finally {
            setSubmitting(false);
        }
    }

    function handleKeyDown(e: React.KeyboardEvent<HTMLDivElement>) {
        if (e.defaultPrevented) {
            return;
        }
        if (e.key === "ArrowUp" && !e.shiftKey && !e.nativeEvent.isComposing) {
            if (body === "" && files.length === 0 && !replyingTo && onEditLast) {
                e.preventDefault();
                onEditLast();
            }
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

    if (timedOut) {
        const until = formatFullDateTime(timeoutUntil);
        return (
            <div className={styles.composer}>
                <div className={styles.timeoutBanner}>You are timed out until {until}.</div>
            </div>
        );
    }

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
            <div className={styles.textareaWrapper} onKeyDown={handleKeyDown}>
                <MentionTextArea
                    placeholder="Type a message... (Enter to send, Shift+Enter for newline)"
                    value={body}
                    onChange={handleBodyChange}
                    rows={1}
                    onPasteFiles={handlePasteFiles}
                    mentionPool={mentionPool}
                    showColours
                />
            </div>
            <div className={styles.actions}>
                <MediaPickerButton onFiles={valid => setFiles(prev => [...prev, ...valid])} onError={setError} />
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
                <span className={styles.spacer} />
                <Button variant="primary" size="small" onClick={handleSubmit} disabled={!canSend}>
                    {submitting ? "..." : "Send"}
                </Button>
            </div>
        </div>
    );
}
