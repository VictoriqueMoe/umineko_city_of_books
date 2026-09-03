import type { ChatMessage } from "../../../../types/api";
import { useChatRoomAttachments } from "../../../../hooks/queries/chat";
import { previewableURLs } from "../../../../domain/links";
import { RelativeTimestamp } from "../../../RelativeTimestamp/RelativeTimestamp";
import styles from "./LinksTab.module.css";

const MAX_LINKS_PER_MESSAGE = 3;

export interface LinksTabProps {
    roomId: string;
    isActive: boolean;
    onJump: (messageId: string, createdAt?: string) => void;
    onJumped: () => void;
}

interface Row {
    key: string;
    url: string;
    message: ChatMessage;
}

function siteName(url: string): string {
    try {
        return new URL(url).hostname.replace(/^www\./, "");
    } catch {
        return url;
    }
}

function senderName(message: ChatMessage): string {
    if (message.sender_nickname && message.sender_nickname.trim() !== "") {
        return message.sender_nickname;
    }

    return message.sender.display_name || message.sender.username;
}

function flatten(messages: ChatMessage[]): Row[] {
    const rows: Row[] = [];
    for (const message of messages) {
        const urls = previewableURLs(message.body, { limit: MAX_LINKS_PER_MESSAGE, keepInlineMedia: true });
        for (const url of urls) {
            rows.push({ key: `${message.id}-${url}`, url, message });
        }
    }

    return rows;
}

export function LinksTab({ roomId, isActive, onJump, onJumped }: LinksTabProps) {
    const { messages, loading } = useChatRoomAttachments(roomId, "links", isActive);
    const rows = flatten(messages);

    if (loading) {
        return <div className={styles.empty}>Loading...</div>;
    }

    if (rows.length === 0) {
        return <div className={styles.empty}>No links have been shared here yet.</div>;
    }

    return (
        <div className={styles.list}>
            {rows.map(row => (
                <div key={row.key} className={styles.row}>
                    <span className={styles.site}>{siteName(row.url)}</span>
                    <a
                        dir="auto"
                        className={styles.url}
                        href={row.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        title={row.url}
                    >
                        {row.url}
                    </a>
                    <div className={styles.meta}>
                        <span dir="auto" className={styles.sender}>
                            {senderName(row.message)}
                        </span>
                        <RelativeTimestamp value={row.message.created_at} variant="dateTime" className={styles.when} />
                        <button
                            type="button"
                            className={styles.jumpBtn}
                            onClick={() => {
                                onJump(row.message.id, row.message.created_at);
                                onJumped();
                            }}
                        >
                            Jump to message
                        </button>
                    </div>
                </div>
            ))}
        </div>
    );
}
