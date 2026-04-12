import type { ChatMessage } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { linkify } from "../../../utils/linkify";
import styles from "./MessageBubble.module.css";

interface MessageBubbleProps {
    message: ChatMessage;
    isOwn: boolean;
    onLightbox?: (src: string) => void;
    onReply?: (msg: ChatMessage) => void;
    highlighted?: boolean;
    seenLabel?: string | null;
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

export function MessageBubble({ message, isOwn, onLightbox, onReply, highlighted, seenLabel }: MessageBubbleProps) {
    const classes = [styles.messageBubble];
    if (isOwn) {
        classes.push(styles.ownMessage);
    }
    if (highlighted) {
        classes.push(styles.messageHighlighted);
    }

    return (
        <div id={`chat-msg-${message.id}`} className={classes.join(" ")}>
            <ProfileLink user={message.sender} size="small" showName={false} />
            <div className={styles.messageContent}>
                {message.reply_to && (
                    <div className={styles.replyPreview} onClick={() => jumpToMessage(message.reply_to!.id)}>
                        <span className={styles.replyArrow}>{"\u21B5"}</span>
                        <span className={styles.replySender}>{message.reply_to.sender_name}</span>
                        <span className={styles.replyText}>{message.reply_to.body_preview}</span>
                    </div>
                )}
                {!isOwn && <div className={styles.messageSender}>{message.sender.display_name}</div>}
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
                <div className={styles.messageTime}>
                    {formatTime(message.created_at)}
                    {seenLabel && <span className={styles.seenLabel}> · {seenLabel}</span>}
                </div>
            </div>
            {onReply && (
                <button
                    type="button"
                    className={styles.replyBtn}
                    onClick={() => onReply(message)}
                    aria-label="Reply"
                    title="Reply"
                >
                    {"\u21A9"}
                </button>
            )}
        </div>
    );
}
