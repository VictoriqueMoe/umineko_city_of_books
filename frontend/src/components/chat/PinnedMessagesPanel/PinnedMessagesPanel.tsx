import { useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { ChatMessage } from "../../../types/api";
import { useChatRoomPinnedMessages } from "../../../api/queries/chat";
import { useUnpinChatMessage } from "../../../api/mutations/chat";
import { queryKeys } from "../../../api/queryKeys";
import { parseServerDate } from "../../../utils/time";
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
    const d = parseServerDate(iso);
    if (!d) {
        return "";
    }
    return d.toLocaleString([], {
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
    const queryClient = useQueryClient();
    const [busyId, setBusyId] = useState<string | null>(null);
    const pinnedQuery = useChatRoomPinnedMessages(roomId, isOpen);
    const unpinMutation = useUnpinChatMessage(roomId);
    const loading = pinnedQuery.loading;
    const pins = useMemo(() => {
        const list = pinnedQuery.messages.slice();
        list.sort((a, b) => {
            const at = a.pinned_at ? Date.parse(a.pinned_at) : 0;
            const bt = b.pinned_at ? Date.parse(b.pinned_at) : 0;
            return bt - at;
        });
        return list;
    }, [pinnedQuery.messages]);

    useEffect(() => {
        if (refreshKey === undefined) {
            return;
        }
        if (!isOpen || !roomId) {
            return;
        }
        void pinnedQuery.refresh();
    }, [refreshKey, isOpen, roomId, pinnedQuery]);

    async function handleUnpin(messageId: string) {
        setBusyId(messageId);
        try {
            await unpinMutation.mutateAsync(messageId);
            queryClient.setQueryData<{ messages: ChatMessage[] }>(queryKeys.chat.pinned(roomId), prev =>
                prev ? { ...prev, messages: prev.messages.filter(m => m.id !== messageId) } : prev,
            );
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
