import { useState } from "react";
import type { ChatMessage, PostMedia } from "../../../../types/api";
import { useChatRoomAttachments } from "../../../../hooks/queries/chat";
import { useLoadMoreOnView } from "../useLoadMoreOnView";
import { AudioThumb } from "../../../AudioAttachment/AudioAttachment";
import { SpoilerOverlay } from "../../../SpoilerImage/SpoilerCover";
import { spoilerBlurClass } from "../../../SpoilerImage/spoilerBlur";
import { RelativeTimestamp } from "../../../RelativeTimestamp/RelativeTimestamp";
import styles from "./MediaTab.module.css";

export interface MediaTabProps {
    roomId: string;
    isActive: boolean;
    onJump: (messageId: string, createdAt?: string) => void;
    onJumped: () => void;
    onLightbox?: (src: string) => void;
}

interface Cell {
    key: string;
    media: PostMedia;
    message: ChatMessage;
}

function flatten(messages: ChatMessage[]): Cell[] {
    const cells: Cell[] = [];
    for (const message of messages) {
        for (const media of message.media ?? []) {
            cells.push({ key: `${message.id}-${media.id}`, media, message });
        }
    }

    return cells;
}

function MediaThumb({ media, onLightbox }: { media: PostMedia; onLightbox?: (src: string) => void }) {
    const [revealed, setRevealed] = useState(false);
    const covered = (media.is_spoiler ?? false) && !revealed;

    function handleClick() {
        if (covered) {
            setRevealed(true);
            return;
        }

        onLightbox?.(media.media_url);
    }

    return (
        <button
            type="button"
            className={styles.thumbBtn}
            aria-label={covered ? "Reveal spoiler" : "Open full size"}
            disabled={media.media_type !== "image" && !covered}
            onClick={handleClick}
        >
            {media.media_type === "audio" && (
                <span className={styles.audio}>
                    <AudioThumb />
                    <span dir="auto" className={styles.filename}>
                        {media.filename ?? "Audio"}
                    </span>
                </span>
            )}
            {media.media_type === "video" && (
                <video
                    className={`${styles.thumb} ${spoilerBlurClass(covered)}`}
                    src={media.media_url}
                    poster={media.thumbnail_url || undefined}
                    muted
                    preload="metadata"
                />
            )}
            {media.media_type === "image" && (
                <img
                    className={`${styles.thumb} ${spoilerBlurClass(covered)}`}
                    src={media.thumbnail_url || media.media_url}
                    alt=""
                    loading="lazy"
                />
            )}
            {covered && <SpoilerOverlay compact />}
        </button>
    );
}

export function MediaTab({ roomId, isActive, onJump, onJumped, onLightbox }: MediaTabProps) {
    const { messages, loading, loadingMore, hasMore, loadMore } = useChatRoomAttachments(roomId, "media", isActive);
    const sentinelRef = useLoadMoreOnView({ hasMore, loadingMore, loadMore });
    const cells = flatten(messages);

    if (loading) {
        return <div className={styles.empty}>Loading...</div>;
    }

    if (cells.length === 0) {
        return <div className={styles.empty}>Nothing has been posted here yet.</div>;
    }

    return (
        <div className={styles.grid}>
            {cells.map(cell => (
                <div key={cell.key} className={styles.cell}>
                    <MediaThumb media={cell.media} onLightbox={onLightbox} />
                    <button
                        type="button"
                        className={styles.jumpBtn}
                        aria-label="Jump to message"
                        onClick={() => {
                            onJump(cell.message.id, cell.message.created_at);
                            onJumped();
                        }}
                    >
                        <RelativeTimestamp value={cell.message.created_at} variant="dateTime" />
                    </button>
                </div>
            ))}
            {hasMore && (
                <div ref={sentinelRef} className={styles.sentinel}>
                    {loadingMore ? "Loading..." : ""}
                </div>
            )}
        </div>
    );
}
