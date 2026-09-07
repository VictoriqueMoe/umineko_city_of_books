import React, { useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState } from "react";
import { Button } from "../../Button/Button";
import { MediaPickerButton, MediaPreviews } from "../../MediaPicker/MediaPicker";
import { MentionTextArea, type MentionTextAreaHandle } from "../../MentionTextArea/MentionTextArea";
import { readChatSendRejection, useSendChatMessage, useSendFirstDMMessage } from "../../../hooks/mutations/chat";
import { useResetOnChange } from "../../../hooks/useResetOnChange";
import { useSiteInfo } from "../../../hooks/useSiteInfo";
import { validateFileSize } from "../../../utils/fileValidation";
import { formatFullDateTime, parseServerDate } from "../../../utils/time";
import { isTimeoutActive } from "../../../domain/chat/roomPolicy";
import type { ChatMessage, ChatRoom, User } from "../../../types/api";
import { GifPicker } from "../GifPicker/GifPicker";
import styles from "./ChatComposer.module.css";

export interface ReplyTarget {
    id: string;
    senderName: string;
    bodyPreview: string;
}

export interface ChatComposerHandle {
    focus: () => void;
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
    extraActions?: React.ReactNode;
    sendOnEnter?: boolean;
    compact?: boolean;
    ref?: React.Ref<ChatComposerHandle>;
}

function formatSendError(err: unknown): string {
    const rejection = readChatSendRejection(err);
    if (rejection?.bannedWord) {
        const suffix = rejection.bannedWord.kicked ? " You have been kicked from this room." : "";
        return `Message blocked by banned-word rule "${rejection.bannedWord.pattern}".${suffix}`;
    }
    if (rejection?.serverMessage) {
        return rejection.serverMessage;
    }
    if (err instanceof Error) {
        return err.message;
    }
    return "Failed to send message";
}

const TYPING_THROTTLE_MS = 2500;

interface PendingAttachment {
    file: File;
    spoiler: boolean;
}

function toPendingAttachments(files: File[]): PendingAttachment[] {
    return files.map(file => ({ file, spoiler: false }));
}

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
    extraActions,
    sendOnEnter = true,
    compact = false,
    ref,
}: ChatComposerProps) {
    const inputRef = useRef<MentionTextAreaHandle>(null);

    useImperativeHandle(
        ref,
        () => ({
            focus: () => {
                inputRef.current?.focus();
            },
        }),
        [],
    );

    const [nowTick, setNowTick] = useState(() => Date.now());
    const [toolbarOpen, setToolbarOpen] = useState(false);
    const timedOut = isTimeoutActive(timeoutUntil, nowTick);
    const siteInfo = useSiteInfo();
    const [body, setBody] = useState("");
    const [attachments, setAttachments] = useState<PendingAttachment[]>([]);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const sendChatMessageMutation = useSendChatMessage(roomId ?? "");
    const sendFirstDMMessageMutation = useSendFirstDMMessage();

    useEffect(() => {
        const reseed = setTimeout(() => setNowTick(Date.now()), 0);

        const parsed = parseServerDate(timeoutUntil);
        const ms = parsed ? parsed.getTime() - Date.now() : 0;
        const expiry = ms > 0 ? setTimeout(() => setNowTick(Date.now()), ms) : null;

        return () => {
            clearTimeout(reseed);
            if (expiry !== null) {
                clearTimeout(expiry);
            }
        };
    }, [timeoutUntil]);
    const [gifPickerOpen, setGifPickerOpen] = useState(false);
    const lastTypingSentRef = useRef(0);

    useResetOnChange(roomId ?? draftRecipientId, () => {
        setBody("");
        setAttachments([]);
        setSubmitting(false);
        setError("");
        setGifPickerOpen(false);
    });

    const replyTargetId = replyingTo?.id ?? null;
    useEffect(() => {
        if (!replyTargetId) {
            return;
        }
        inputRef.current?.focus();
    }, [replyTargetId]);

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

    const files = useMemo(() => attachments.map(a => a.file), [attachments]);
    const spoilers = useMemo(() => attachments.map(a => a.spoiler), [attachments]);

    function removeFile(index: number) {
        setAttachments(prev => prev.filter((_, i) => i !== index));
    }

    function toggleSpoiler(index: number) {
        setAttachments(prev => prev.map((a, i) => (i === index ? { ...a, spoiler: !a.spoiler } : a)));
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
                setAttachments(prev => [...prev, ...toPendingAttachments(valid)]);
            }
        },
        [siteInfo.max_image_size, siteInfo.max_video_size, siteInfo.max_audio_size],
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
                    spoilers,
                });
                onSent(created.message, created.room);
            } else {
                const message = await sendChatMessageMutation.mutateAsync({
                    body: trimmed,
                    reply_to_id: replyingTo?.id,
                    files,
                    spoilers,
                });
                onSent(message);
            }
            setBody("");
            setAttachments([]);
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
        if (!sendOnEnter) {
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
    const showToolbarItems = !compact || toolbarOpen;
    const placeholder = sendOnEnter
        ? "Type a message... (Enter to send, Shift+Enter for newline)"
        : "Type a message...";

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
                        <span className={styles.replyLabel}>
                            Replying to <bdi>{replyingTo.senderName}</bdi>
                        </span>
                        <span dir="auto" className={styles.replyPreview}>
                            {replyingTo.bodyPreview}
                        </span>
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
                    <MediaPreviews
                        files={files}
                        onRemove={removeFile}
                        spoilers={spoilers}
                        onToggleSpoiler={toggleSpoiler}
                        size="small"
                    />
                </div>
            )}
            <div className={styles.textareaWrapper} onKeyDown={handleKeyDown}>
                <MentionTextArea
                    ref={inputRef}
                    placeholder={placeholder}
                    value={body}
                    onChange={handleBodyChange}
                    rows={1}
                    onPasteFiles={handlePasteFiles}
                    mentionPool={mentionPool}
                    showColours
                    colourBarOpen={showToolbarItems}
                />
            </div>
            <div className={styles.actions}>
                {compact && (
                    <Button
                        variant="ghost"
                        size="small"
                        onClick={() => setToolbarOpen(prev => !prev)}
                        aria-expanded={toolbarOpen}
                        aria-label="More options"
                    >
                        {toolbarOpen ? "×" : "+"}
                    </Button>
                )}
                {showToolbarItems && (
                    <>
                        <MediaPickerButton
                            onFiles={valid => setAttachments(prev => [...prev, ...toPendingAttachments(valid)])}
                            onError={setError}
                        />
                        <div className={styles.gifAnchor}>
                            <Button
                                variant="ghost"
                                size="small"
                                onClick={() => setGifPickerOpen(prev => !prev)}
                                disabled={submitting}
                            >
                                + GIF
                            </Button>
                            {gifPickerOpen && (
                                <GifPicker onPick={handleGifPick} onClose={() => setGifPickerOpen(false)} />
                            )}
                        </div>
                        {extraActions}
                    </>
                )}
                <Button
                    variant="primary"
                    size="small"
                    className={styles.send}
                    onClick={handleSubmit}
                    disabled={!canSend}
                >
                    {submitting ? "..." : "Send"}
                </Button>
            </div>
        </div>
    );
}
