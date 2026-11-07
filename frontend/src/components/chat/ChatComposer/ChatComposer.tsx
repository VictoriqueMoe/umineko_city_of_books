import React, { useCallback, useEffect, useImperativeHandle, useRef, useState } from "react";
import { Button } from "../../Button/Button";
import { MediaPickerButton, MediaPreviews } from "../../MediaPicker/MediaPicker";
import { MentionTextArea, type MentionTextAreaHandle } from "../../MentionTextArea/MentionTextArea";
import type { ChatSender, DraftSender } from "../../../hooks/chat/useOptimisticSends";
import { useIsMobile } from "../../../hooks/useIsMobile";
import { useResetOnChange } from "../../../hooks/useResetOnChange";
import { useSiteInfo } from "../../../hooks/useSiteInfo";
import { type StagedAttachment, useStagedMedia } from "../../../hooks/useStagedMedia";
import { useWarnBeforeLeaving } from "../../../hooks/useWarnBeforeLeaving";
import { validateFileSize } from "../../../utils/fileValidation";
import { formatFullDateTime, parseServerDate } from "../../../utils/time";
import type { OutgoingAttachment } from "../../../domain/chat/optimisticSend";
import { isTimeoutActive } from "../../../domain/chat/roomPolicy";
import type { User } from "../../../types/api";
import { GifPicker } from "../GifPicker/GifPicker";
import { renderRichInline } from "../../richText/richText";
import styles from "./ChatComposer.module.css";

export interface ReplyTarget {
    id: string;
    senderName: string;
    body: string;
}

export interface ChatComposerHandle {
    focus: () => void;
}

export interface RoomComposerTarget {
    kind: "room";
    sender: ChatSender;
}

export interface DraftComposerTarget {
    kind: "draft";
    sender: DraftSender;
}

export type ComposerTarget = RoomComposerTarget | DraftComposerTarget;

interface ChatComposerProps {
    target: ComposerTarget | null;
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

const TYPING_THROTTLE_MS = 2500;

function toOutgoingAttachments(items: StagedAttachment[]): OutgoingAttachment[] {
    return items.map(item => ({ file: item.file, spoiler: item.isSpoiler, uploadId: item.uploadId as string }));
}

function targetKey(target: ComposerTarget | null): string | null {
    if (!target) {
        return null;
    }
    if (target.kind === "draft") {
        return `draft:${target.sender.recipientId}`;
    }
    if (target.sender.adoptedFrom) {
        return `draft:${target.sender.adoptedFrom}`;
    }

    return `room:${target.sender.roomId}`;
}

export function ChatComposer({
    target,
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
    const isMobile = useIsMobile();
    const siteInfo = useSiteInfo();
    const [body, setBody] = useState("");
    const media = useStagedMedia();
    const { add: addMedia, reset: resetMedia, files } = media;
    const [error, setError] = useState("");
    useWarnBeforeLeaving(media.busy);

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

    useResetOnChange(targetKey(target), () => {
        setBody("");
        resetMedia();
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

    function clearDraft() {
        setBody("");
        resetMedia();
        if (onCancelReply) {
            onCancelReply();
        }
    }

    function handleGifPick(gif: { id: string; url: string }) {
        setGifPickerOpen(false);
        if (!target) {
            return;
        }

        target.sender.send({ body: gif.url, replyTo: replyingTo ?? null, attachments: [] });
        if (onCancelReply) {
            onCancelReply();
        }
    }

    function handleSubmit() {
        if (!target || !canSend) {
            return;
        }

        target.sender.send({
            body: body.trim(),
            replyTo: replyingTo ?? null,
            attachments: toOutgoingAttachments(media.items),
        });
        clearDraft();
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

    const canSend = media.ready && (body.trim().length > 0 || files.length > 0);
    const showToolbarItems = !compact || toolbarOpen;
    const placeholder =
        sendOnEnter && !isMobile ? "Type a message... (Enter to send, Shift+Enter for newline)" : "Type a message...";

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
                            {renderRichInline(replyingTo.body)}
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
                        onRemove={media.remove}
                        spoilers={media.spoilers}
                        onToggleSpoiler={media.toggleSpoiler}
                        progress={media.progress}
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
                    onSubmitShortcut={handleSubmit}
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
                        <MediaPickerButton onFiles={addMedia} onError={setError} />
                        <div className={styles.gifAnchor}>
                            <Button variant="ghost" size="small" onClick={() => setGifPickerOpen(prev => !prev)}>
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
                    Send
                </Button>
            </div>
        </div>
    );
}
