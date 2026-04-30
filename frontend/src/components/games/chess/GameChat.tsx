import { useEffect, useMemo, useRef, useState } from "react";
import * as api from "../../../api/endpoints.ts";
import type { SpectatorChatResponse, SpectatorMessage, WSMessage } from "../../../types/api.ts";
import { useAuth } from "../../../hooks/useAuth.ts";
import { useNotifications } from "../../../hooks/useNotifications.ts";
import { formatTimeOfDay } from "../../../utils/time.ts";
import { Button } from "../../Button/Button.tsx";
import styles from "./SpectatorChat.module.css";

export type GameChatVariant = "spectator" | "player";

interface GameChatProps {
    roomId: string;
    variant: GameChatVariant;
    watcherCount?: number;
}

interface VariantConfig {
    title: string;
    rightMeta: string;
    emptyText: string;
    placeholder: string;
    fetch: (roomId: string) => Promise<SpectatorChatResponse>;
    post: (roomId: string, body: string) => Promise<SpectatorMessage>;
    wsType: string;
}

function configFor(variant: GameChatVariant, watcherCount: number): VariantConfig {
    if (variant === "player") {
        return {
            title: "Player chat",
            rightMeta: "Private",
            emptyText: "Only you and your opponent can see this chat.",
            placeholder: "Message your opponent...",
            fetch: api.getPlayerChat,
            post: api.postPlayerChat,
            wsType: "player_chat_message",
        };
    }
    return {
        title: "Spectator chat",
        rightMeta: `${watcherCount} watching`,
        emptyText: "No messages yet. Say hello.",
        placeholder: "Chat with other watchers...",
        fetch: api.getSpectatorChat,
        post: api.postSpectatorChat,
        wsType: "spectator_chat_message",
    };
}

export function GameChat({ roomId, variant, watcherCount = 0 }: GameChatProps) {
    const cfg = useMemo(() => configFor(variant, watcherCount), [variant, watcherCount]);
    const { user } = useAuth();
    const { addWSListener, wsEpoch } = useNotifications();
    const [messages, setMessages] = useState<SpectatorMessage[]>([]);
    const [body, setBody] = useState("");
    const [sending, setSending] = useState(false);
    const [error, setError] = useState("");
    const scrollRef = useRef<HTMLDivElement>(null);

    const fetchChat = cfg.fetch;
    useEffect(() => {
        let cancelled = false;
        fetchChat(roomId)
            .then(resp => {
                if (cancelled) {
                    return;
                }
                setMessages(resp.messages ?? []);
            })
            .catch(() => {});
        return () => {
            cancelled = true;
        };
    }, [roomId, wsEpoch, fetchChat]);

    const wsType = cfg.wsType;
    useEffect(() => {
        return addWSListener((msg: WSMessage) => {
            if (msg.type !== wsType) {
                return;
            }
            const data = msg.data as { room_id?: string; message?: SpectatorMessage };
            if (data.room_id !== roomId || !data.message) {
                return;
            }
            setMessages(prev => {
                if (prev.some(m => m.id === data.message!.id)) {
                    return prev;
                }
                return [...prev, data.message as SpectatorMessage];
            });
        });
    }, [addWSListener, roomId, wsType]);

    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
        }
    }, [messages.length]);

    async function handleSend() {
        const trimmed = body.trim();
        if (!trimmed || sending) {
            return;
        }
        setSending(true);
        setError("");
        try {
            const sent = await cfg.post(roomId, trimmed);
            setMessages(prev => {
                if (prev.some(m => m.id === sent.id)) {
                    return prev;
                }
                return [...prev, sent];
            });
            setBody("");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to send");
        } finally {
            setSending(false);
        }
    }

    function handleKey(e: React.KeyboardEvent<HTMLInputElement>) {
        if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            void handleSend();
        }
    }

    return (
        <div className={styles.panel}>
            <div className={styles.header}>
                <span>{cfg.title}</span>
                <span>{cfg.rightMeta}</span>
            </div>
            <div className={styles.messages} ref={scrollRef}>
                {messages.length === 0 ? (
                    <p className={styles.empty}>{cfg.emptyText}</p>
                ) : (
                    messages.map(m => (
                        <div key={m.id} className={styles.message}>
                            <div className={styles.messageHeader}>
                                <span className={styles.author}>{m.user.display_name}</span>
                                <span className={styles.timestamp}>{formatTimeOfDay(m.created_at)}</span>
                            </div>
                            <span className={styles.body}>{m.body}</span>
                        </div>
                    ))
                )}
            </div>
            {error && <div className={styles.empty}>{error}</div>}
            {user ? (
                <div className={styles.inputRow}>
                    <input
                        className={styles.input}
                        placeholder={cfg.placeholder}
                        value={body}
                        onChange={e => setBody(e.target.value)}
                        onKeyDown={handleKey}
                        maxLength={500}
                        disabled={sending}
                    />
                    <Button variant="primary" size="small" onClick={handleSend} disabled={sending || !body.trim()}>
                        Send
                    </Button>
                </div>
            ) : (
                <div className={styles.disabled}>Sign in to join the chat.</div>
            )}
        </div>
    );
}
