import type { ChatMessage, PostMedia } from "../../../../types/api";
import { useChatRoomAttachments } from "../../../../hooks/queries/chat";
import { AudioThumb } from "../../../AudioAttachment/AudioAttachment";
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

export function MediaTab({ roomId, isActive, onJump, onJumped, onLightbox }: MediaTabProps) {
    const { messages, loading } = useChatRoomAttachments(roomId, "media", isActive);
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
                    <button
                        type="button"
                        className={styles.thumbBtn}
                        aria-label="Open full size"
                        disabled={cell.media.media_type !== "image"}
                        onClick={() => onLightbox?.(cell.media.media_url)}
                    >
                        {cell.media.media_type === "audio" && (
                            <span className={styles.audio}>
                                <AudioThumb />
                                <span dir="auto" className={styles.filename}>
                                    {cell.media.filename ?? "Audio"}
                                </span>
                            </span>
                        )}
                        {cell.media.media_type === "video" && (
                            <video
                                className={styles.thumb}
                                src={cell.media.media_url}
                                poster={cell.media.thumbnail_url || undefined}
                                muted
                                preload="metadata"
                            />
                        )}
                        {cell.media.media_type === "image" && (
                            <img
                                className={styles.thumb}
                                src={cell.media.thumbnail_url || cell.media.media_url}
                                alt=""
                                loading="lazy"
                            />
                        )}
                    </button>
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
        </div>
    );
}
