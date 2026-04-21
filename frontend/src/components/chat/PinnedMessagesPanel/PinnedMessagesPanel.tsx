import { useCallback, useEffect, useState } from "react";
import type { ChatMessage } from "../../../types/api";
import { getChatRoomPinnedMessages, unpinChatMessage } from "../../../api/endpoints";
import styles from "./PinnedMessagesPanel.module.css";

interface PinnedMessagesPanelProps {
    roomId: string;
    isOpen: boolean;
    onClose: () => void;
    onJump: (messageId: string, createdAt?: string) => void;
    canUnpin: boolean;
    refreshKey?: number;
}

function formatDateTime(iso: string): string {
    if (!iso) {
        return "";
    }
    return new Date(iso).toLocaleString([], {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
    });
}

function senderDisplayName(msg: ChatMessage): string {
    if (msg.sender_nickname && msg.sender_nickname.trim() !== "") {
        return msg.sender_nickname;
    }
    return msg.sender.display_name;
}

function senderAvatarUrl(msg: ChatMessage): string | undefined {
    if (msg.sender_member_avatar_url && msg.sender_member_avatar_url.trim() !== "") {
        return msg.sender_member_avatar_url;
    }
    return msg.sender.avatar_url;
}

export function PinnedMessagesPanel({
    roomId,
    isOpen,
    onClose,
    onJump,
    canUnpin,
    refreshKey,
}: PinnedMessagesPanelProps) {
    const [pins, setPins] = useState<ChatMessage[]>([]);
    const [loading, setLoading] = useState(false);
    const [busyId, setBusyId] = useState<string | null>(null);

    const load = useCallback(async () => {
        setLoading(true);
        try {
            const res = await getChatRoomPinnedMessages(roomId);
            const list = res.messages ?? [];
            list.sort((a, b) => {
                const at = a.pinned_at ? Date.parse(a.pinned_at) : 0;
                const bt = b.pinned_at ? Date.parse(b.pinned_at) : 0;
                return bt - at;
            });
            setPins(list);
        } catch {
            setPins([]);
        } finally {
            setLoading(false);
        }
    }, [roomId]);

    useEffect(() => {
        if (!isOpen) {
            return;
        }
        load();
    }, [isOpen, load, refreshKey]);

    async function handleUnpin(messageId: string) {
        setBusyId(messageId);
        try {
            await unpinChatMessage(messageId);
            setPins(prev => prev.filter(m => m.id !== messageId));
        } catch {
            // leave list unchanged
        } finally {
            setBusyId(null);
        }
    }

    if (!isOpen) {
        return null;
    }

    return (
        <div className={styles.overlay} onClick={onClose}>
            <aside
                className={styles.drawer}
                onClick={e => e.stopPropagation()}
                role="dialog"
                aria-label="Pinned messages"
            >
                <header className={styles.header}>
                    <span className={styles.title}>Pinned messages</span>
                    <button type="button" className={styles.closeBtn} onClick={onClose} aria-label="Close">
                        {"\u2715"}
                    </button>
                </header>
                <div className={styles.body}>
                    {loading && <div className={styles.empty}>Loading...</div>}
                    {!loading && pins.length === 0 && <div className={styles.empty}>No pinned messages yet.</div>}
                    {!loading &&
                        pins.map(m => {
                            const avatar = senderAvatarUrl(m);
                            return (
                                <div key={m.id} className={styles.pinItem}>
                                    <div className={styles.pinMeta}>
                                        {avatar ? (
                                            <img className={styles.pinAvatar} src={avatar} alt="" />
                                        ) : (
                                            <span className={styles.pinAvatarPlaceholder}>
                                                {senderDisplayName(m)[0] ?? "?"}
                                            </span>
                                        )}
                                        <div className={styles.pinMetaText}>
                                            <span className={styles.pinSender}>{senderDisplayName(m)}</span>
                                            <span className={styles.pinTime}>
                                                {m.pinned_at
                                                    ? formatDateTime(m.pinned_at)
                                                    : formatDateTime(m.created_at)}
                                            </span>
                                        </div>
                                    </div>
                                    {m.body && <div className={styles.pinBody}>{m.body}</div>}
                                    {m.media && m.media.length > 0 && (
                                        <div className={styles.pinMediaNote}>
                                            {m.media.length} attachment{m.media.length > 1 ? "s" : ""}
                                        </div>
                                    )}
                                    <div className={styles.pinActions}>
                                        <button
                                            type="button"
                                            className={styles.jumpBtn}
                                            onClick={() => {
                                                onJump(m.id, m.created_at);
                                                onClose();
                                            }}
                                        >
                                            Jump to message
                                        </button>
                                        {canUnpin && (
                                            <button
                                                type="button"
                                                className={styles.unpinBtn}
                                                onClick={() => handleUnpin(m.id)}
                                                disabled={busyId === m.id}
                                            >
                                                {busyId === m.id ? "Unpinning..." : "Unpin"}
                                            </button>
                                        )}
                                    </div>
                                </div>
                            );
                        })}
                </div>
            </aside>
        </div>
    );
}
