import { useState } from "react";
import type { ChatMessage, ReactionGroup, User } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { RolePill } from "../../RolePill/RolePill";
import { linkify } from "../../../utils/linkify";
import { EmojiPicker } from "../EmojiPicker/EmojiPicker";
import styles from "./MessageBubble.module.css";

interface MessageBubbleProps {
    message: ChatMessage;
    isOwn: boolean;
    onLightbox?: (src: string) => void;
    onReply?: (msg: ChatMessage) => void;
    onReactionToggle?: (msg: ChatMessage, emoji: string) => void;
    onPinToggle?: (msg: ChatMessage) => void;
    onDelete?: (msg: ChatMessage) => void;
    canPin?: boolean;
    canModerate?: boolean;
    highlighted?: boolean;
    seenLabel?: string | null;
    senderIsStaff?: boolean;
}

function formatTime(dateStr: string): string {
    if (!dateStr) {
        return "";
    }
    return new Date(dateStr).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
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
    canPin,
    canModerate,
    highlighted,
    seenLabel,
    senderIsStaff,
}: MessageBubbleProps) {
    const [pickerOpen, setPickerOpen] = useState(false);
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
                <div className={styles.systemMessageText}>{linkify(message.body)}</div>
                <div className={styles.systemMessageTime}>{formatTime(message.created_at)}</div>
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
                {!isOwn && (
                    <div className={styles.messageSender}>
                        {effectiveSender.display_name}
                        <RolePill role={effectiveSender.role ?? ""} userId={effectiveSender.id} />
                    </div>
                )}
                {message.body.trim() && <div className={styles.messageText}>{linkify(message.body)}</div>}
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
                        {message.reactions.map(r => (
                            <button
                                key={r.emoji}
                                type="button"
                                className={`${styles.reactionChip} ${r.viewer_reacted ? styles.reactionChipMine : ""}`}
                                onClick={() => onReactionToggle?.(message, r.emoji)}
                                title={reactionTooltip(r)}
                            >
                                <span className={styles.reactionEmoji}>{r.emoji}</span>
                                <span className={styles.reactionCount}>{r.count}</span>
                            </button>
                        ))}
                    </div>
                )}
                <div className={styles.messageTime}>
                    {formatTime(message.created_at)}
                    {seenLabel && <span className={styles.seenLabel}> · {seenLabel}</span>}
                </div>
            </div>
            <div className={styles.actions}>
                {onReactionToggle && (
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
