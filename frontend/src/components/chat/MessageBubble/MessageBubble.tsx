import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { ChatMessage, ReactionGroup, User } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { RolePill } from "../../RolePill/RolePill";
import { renderRich } from "../../../utils/richText";
import { GifEmbed } from "../../GifEmbed/GifEmbed";
import { EmojiPicker } from "../EmojiPicker/EmojiPicker";
import { YouTubeEmbed } from "../YouTubeEmbed/YouTubeEmbed";
import { formatFullDateTime, formatMessageTime } from "../../../utils/time";
import { extractYouTubeIDs } from "../../../utils/youtube";
import styles from "./MessageBubble.module.css";

interface MessageBubbleProps {
    message: ChatMessage;
    isOwn: boolean;
    onLightbox?: (src: string) => void;
    onReply?: (msg: ChatMessage) => void;
    onReactionToggle?: (msg: ChatMessage, emoji: string) => void;
    onPinToggle?: (msg: ChatMessage) => void;
    onDelete?: (msg: ChatMessage) => void;
    onEdit?: (msg: ChatMessage, newBody: string) => Promise<void>;
    onEditStart?: (msg: ChatMessage) => void;
    onEditCancel?: () => void;
    editing?: boolean;
    canPin?: boolean;
    canModerate?: boolean;
    canReact?: boolean;
    canEdit?: boolean;
    highlighted?: boolean;
    notifiesViewer?: boolean;
    seenLabel?: string | null;
    senderIsStaff?: boolean;
}

const GIPHY_URL_RE = /^https:\/\/(media[0-9]*|i)\.giphy\.com\/[^\s]+\.(gif|webp|mp4)(\?[^\s]*)?$/i;

function extractGif(body: string): string | null {
    const trimmed = body.trim();
    if (GIPHY_URL_RE.test(trimmed)) {
        return trimmed;
    }
    return null;
}

function jumpToMessage(id: string) {
    const el = document.getElementById(`chat-msg-${id}`);
    if (el) {
        el.scrollIntoView({ behavior: "smooth", block: "center" });
    }
}

function reactionTooltip(r: ReactionGroup): string {
    const names = r.display_names ?? [];
    if (names.length === 0) {
        return r.viewer_reacted ? "Click to remove your reaction" : "Click to react";
    }
    return names.join("\n");
}

function applySenderOverrides(message: ChatMessage): User {
    const override: User = { ...message.sender };
    if (message.sender_nickname) {
        override.display_name = message.sender_nickname;
    }
    if (!override.display_name || override.display_name.trim() === "") {
        override.display_name = override.username;
    }
    if (message.sender_member_avatar_url) {
        override.avatar_url = message.sender_member_avatar_url;
    }
    return override;
}

export function MessageBubble({
    message,
    isOwn,
    onLightbox,
    onReply,
    onReactionToggle,
    onPinToggle,
    onDelete,
    onEdit,
    onEditStart,
    onEditCancel,
    editing = false,
    canPin,
    canModerate,
    canReact = true,
    canEdit = true,
    highlighted,
    notifiesViewer,
    seenLabel,
    senderIsStaff,
}: MessageBubbleProps) {
    const [pickerOpen, setPickerOpen] = useState(false);

    const [reactorsPopover, setReactorsPopover] = useState<string | null>(null);
    const longPressTimerRef = useRef<number | null>(null);
    const longPressedRef = useRef(false);

    useEffect(() => {
        if (!reactorsPopover) {
            return;
        }
        function handleClickOutside(e: Event) {
            const target = e.target as HTMLElement | null;
            if (!target) {
                return;
            }
            if (!target.closest(`[data-reactor-popover="${message.id}"]`)) {
                setReactorsPopover(null);
            }
        }
        document.addEventListener("mousedown", handleClickOutside);
        document.addEventListener("touchstart", handleClickOutside);
        return () => {
            document.removeEventListener("mousedown", handleClickOutside);
            document.removeEventListener("touchstart", handleClickOutside);
        };
    }, [reactorsPopover, message.id]);

    const popoverRef = useRef<HTMLDivElement | null>(null);
    useLayoutEffect(() => {
        if (!reactorsPopover) {
            return;
        }
        const popover = popoverRef.current;
        if (!popover) {
            return;
        }
        const anchor = popover.parentElement;
        if (!anchor) {
            return;
        }
        const chipRect = anchor.getBoundingClientRect();
        popover.style.position = "fixed";
        popover.style.right = "auto";
        popover.style.bottom = `${Math.round(window.innerHeight - chipRect.top + 8)}px`;
        const popWidth = popover.offsetWidth;
        const margin = 8;
        const chipCenter = chipRect.left + chipRect.width / 2;
        const rawLeft = chipCenter - popWidth / 2;
        const clampedLeft = Math.max(margin, Math.min(rawLeft, window.innerWidth - popWidth - margin));
        popover.style.left = `${Math.round(clampedLeft)}px`;
    }, [reactorsPopover]);

    function clearLongPressTimer() {
        if (longPressTimerRef.current !== null) {
            window.clearTimeout(longPressTimerRef.current);
            longPressTimerRef.current = null;
        }
    }

    function handleReactionPointerDown(emoji: string) {
        longPressedRef.current = false;
        clearLongPressTimer();
        longPressTimerRef.current = window.setTimeout(() => {
            longPressedRef.current = true;
            setReactorsPopover(emoji);
        }, 450);
    }

    function handleReactionPointerEnd() {
        clearLongPressTimer();
    }

    function handleReactionClick(r: ReactionGroup) {
        if (longPressedRef.current) {
            longPressedRef.current = false;
            return;
        }
        if (canReact) {
            onReactionToggle?.(message, r.emoji);
        }
    }

    function startEdit() {
        onEditStart?.(message);
    }
    const isSystemMessage = message.is_system;
    const classes = [styles.messageBubble];
    if (isOwn && !isSystemMessage) {
        classes.push(styles.ownMessage);
    }
    if (isSystemMessage) {
        classes.push(styles.systemMessage);
    }
    if (highlighted) {
        classes.push(styles.messageHighlighted);
    }
    if (notifiesViewer && !isOwn) {
        classes.push(styles.notifiesViewer);
    }
    if (message.pinned) {
        classes.push(styles.messagePinned);
    }

    const effectiveSender = applySenderOverrides(message);

    function handlePick(emoji: string) {
        setPickerOpen(false);
        onReactionToggle?.(message, emoji);
    }

    if (isSystemMessage) {
        return (
            <div id={`chat-msg-${message.id}`} className={classes.join(" ")}>
                <div className={styles.systemMessageText}>{renderRich(message.body)}</div>
                <div className={styles.systemMessageTime} title={formatFullDateTime(message.created_at)}>
                    {formatMessageTime(message.created_at)}
                </div>
            </div>
        );
    }

    return (
        <div id={`chat-msg-${message.id}`} className={classes.join(" ")}>
            <ProfileLink user={effectiveSender} size="small" showName={false} />
            <div className={styles.messageContent}>
                {message.pinned && (
                    <div className={styles.pinnedIndicator} title="Pinned">
                        {"\u{1F4CC}"} <span>Pinned</span>
                    </div>
                )}
                {message.reply_to && (
                    <div className={styles.replyPreview} onClick={() => jumpToMessage(message.reply_to!.id)}>
                        <span className={styles.replyArrow}>{"\u21B5"}</span>
                        <span className={styles.replySender}>{message.reply_to.sender_name}</span>
                        <span className={styles.replyText}>{message.reply_to.body_preview}</span>
                    </div>
                )}
                <div className={`${styles.messageSender} ${isOwn ? styles.messageSenderOwn : ""}`}>
                    {effectiveSender.display_name}
                    <RolePill role={effectiveSender.role ?? ""} userId={effectiveSender.id} />
                </div>
                {editing ? (
                    <EditRow
                        key={message.id}
                        initialBody={message.body}
                        onCommit={async next => {
                            if (!onEdit) {
                                return;
                            }
                            await onEdit(message, next);
                            onEditCancel?.();
                        }}
                        onCancel={() => onEditCancel?.()}
                    />
                ) : (
                    (() => {
                        const gifURL = extractGif(message.body);
                        if (gifURL) {
                            return (
                                <GifEmbed
                                    src={gifURL}
                                    imgClassName={styles.gifEmbed}
                                    onClick={() => onLightbox?.(gifURL)}
                                />
                            );
                        }
                        const youtubeIds = extractYouTubeIDs(message.body);
                        return (
                            <>
                                {message.body.trim() && (
                                    <div className={styles.messageText}>{renderRich(message.body)}</div>
                                )}
                                {youtubeIds.length > 0 && <YouTubeEmbed videoIds={youtubeIds} />}
                            </>
                        );
                    })()
                )}
                {message.media && message.media.length > 0 && (
                    <div className={styles.messageMedia}>
                        {message.media.map(m =>
                            m.media_type === "video" ? (
                                <video
                                    key={m.id}
                                    className={styles.messageMediaItem}
                                    src={m.media_url}
                                    controls
                                    poster={m.thumbnail_url || undefined}
                                />
                            ) : (
                                <img
                                    key={m.id}
                                    className={styles.messageMediaItem}
                                    src={m.media_url}
                                    alt=""
                                    onClick={() => onLightbox?.(m.media_url)}
                                />
                            ),
                        )}
                    </div>
                )}
                {message.reactions && message.reactions.length > 0 && (
                    <div className={styles.reactionRow}>
                        {message.reactions.map(r => {
                            const names = r.display_names ?? [];
                            const isOpen = reactorsPopover === r.emoji;
                            return (
                                <span key={r.emoji} className={styles.reactionAnchor} data-reactor-popover={message.id}>
                                    <button
                                        type="button"
                                        className={`${styles.reactionChip} ${r.viewer_reacted ? styles.reactionChipMine : ""}`}
                                        onClick={() => handleReactionClick(r)}
                                        onPointerDown={() => handleReactionPointerDown(r.emoji)}
                                        onPointerUp={handleReactionPointerEnd}
                                        onPointerLeave={handleReactionPointerEnd}
                                        onPointerCancel={handleReactionPointerEnd}
                                        onContextMenu={e => {
                                            e.preventDefault();
                                            setReactorsPopover(r.emoji);
                                        }}
                                        disabled={!canReact && !names.length}
                                        title={canReact ? reactionTooltip(r) : "You are timed out"}
                                    >
                                        <span className={styles.reactionEmoji}>{r.emoji}</span>
                                        <span className={styles.reactionCount}>{r.count}</span>
                                    </button>
                                    {isOpen && (
                                        <div
                                            ref={popoverRef}
                                            className={styles.reactorPopover}
                                            role="dialog"
                                            aria-label="Reactors"
                                        >
                                            <div className={styles.reactorHeader}>
                                                <span className={styles.reactorHeaderEmoji}>{r.emoji}</span>
                                                <span>{r.count} reacted</span>
                                            </div>
                                            {names.length > 0 ? (
                                                <ul className={styles.reactorList}>
                                                    {names.map(n => (
                                                        <li key={n}>{n}</li>
                                                    ))}
                                                </ul>
                                            ) : (
                                                <div className={styles.reactorEmpty}>No reactor names available.</div>
                                            )}
                                        </div>
                                    )}
                                </span>
                            );
                        })}
                    </div>
                )}
                <div className={styles.messageTime} title={formatFullDateTime(message.created_at)}>
                    {formatMessageTime(message.created_at)}
                    {message.edited_at && (
                        <span className={styles.editedLabel} title={`Edited ${formatFullDateTime(message.edited_at)}`}>
                            {" "}
                            (edited)
                        </span>
                    )}
                    {seenLabel && <span className={styles.seenLabel}> · {seenLabel}</span>}
                </div>
            </div>
            <div className={styles.actions}>
                {onReactionToggle && canReact && (
                    <div className={styles.reactAnchor}>
                        <button
                            type="button"
                            className={styles.actionBtn}
                            onClick={() => setPickerOpen(prev => !prev)}
                            aria-label="React"
                            title="React"
                        >
                            {"\u{1F642}+"}
                        </button>
                        {pickerOpen && <EmojiPicker onPick={handlePick} onClose={() => setPickerOpen(false)} />}
                    </div>
                )}
                {onReply && (
                    <button
                        type="button"
                        className={styles.actionBtn}
                        onClick={() => onReply(message)}
                        aria-label="Reply"
                        title="Reply"
                    >
                        {"\u21A9"}
                    </button>
                )}
                {canPin && onPinToggle && (
                    <button
                        type="button"
                        className={styles.actionBtn}
                        onClick={() => onPinToggle(message)}
                        aria-label={message.pinned ? "Unpin message" : "Pin message"}
                        title={message.pinned ? "Unpin message" : "Pin message"}
                    >
                        {message.pinned ? "\u{1F4CC}\u2715" : "\u{1F4CC}"}
                    </button>
                )}
                {isOwn && canEdit && onEdit && !editing && (
                    <button
                        type="button"
                        className={styles.actionBtn}
                        onClick={startEdit}
                        aria-label="Edit message"
                        title="Edit message"
                    >
                        {"\u270E"}
                    </button>
                )}
                {(isOwn || (canModerate && !senderIsStaff)) && onDelete && (
                    <button
                        type="button"
                        className={styles.actionBtn}
                        onClick={() => {
                            if (window.confirm("Delete this message?")) {
                                onDelete(message);
                            }
                        }}
                        aria-label="Delete message"
                        title="Delete message"
                    >
                        {"\u{1F5D1}"}
                    </button>
                )}
            </div>
        </div>
    );
}

interface EditRowProps {
    initialBody: string;
    onCommit: (next: string) => Promise<void>;
    onCancel: () => void;
}

function EditRow({ initialBody, onCommit, onCancel }: EditRowProps) {
    const [draft, setDraft] = useState(initialBody);
    const [saving, setSaving] = useState(false);

    async function commit() {
        const next = draft.trim();
        if (next === "" || next === initialBody.trim()) {
            onCancel();
            return;
        }
        setSaving(true);
        try {
            await onCommit(next);
        } finally {
            setSaving(false);
        }
    }

    return (
        <div className={styles.editRow}>
            <textarea
                className={styles.editTextarea}
                value={draft}
                onChange={e => setDraft(e.target.value)}
                onKeyDown={e => {
                    if (e.key === "Enter" && !e.shiftKey) {
                        e.preventDefault();
                        void commit();
                        return;
                    }
                    if (e.key === "Escape") {
                        e.preventDefault();
                        onCancel();
                    }
                }}
                disabled={saving}
                autoFocus
                rows={Math.min(8, Math.max(2, draft.split("\n").length))}
            />
            <div className={styles.editActions}>
                <button type="button" className={styles.editBtn} onClick={onCancel} disabled={saving}>
                    Cancel
                </button>
                <button
                    type="button"
                    className={`${styles.editBtn} ${styles.editBtnPrimary}`}
                    onClick={() => void commit()}
                    disabled={saving || draft.trim() === ""}
                >
                    {saving ? "Saving..." : "Save"}
                </button>
            </div>
            <div className={styles.editHint}>Enter to save · Esc to cancel</div>
        </div>
    );
}
