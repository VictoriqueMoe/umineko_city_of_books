import { useEffect, useRef, useState } from "react";
import * as api from "../../api/endpoints";
import type { SpectatorMessage, WSMessage } from "../../types/api";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import { Button } from "../Button/Button";
import styles from "./SpectatorChat.module.css";

interface SpectatorChatProps {
    roomId: string;
    watcherCount: number;
}

export function SpectatorChat({ roomId, watcherCount }: SpectatorChatProps) {
    const { user } = useAuth();
    const { addWSListener, wsEpoch } = useNotifications();
    const [messages, setMessages] = useState<SpectatorMessage[]>([]);
    const [body, setBody] = useState("");
    const [sending, setSending] = useState(false);
    const [error, setError] = useState("");
    const scrollRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        let cancelled = false;
        api.getSpectatorChat(roomId)
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
    }, [roomId, wsEpoch]);

    useEffect(() => {
        return addWSListener((msg: WSMessage) => {
            if (msg.type !== "spectator_chat_message") {
                return;
            }
            const data = msg.data as { room_id?: string; message?: SpectatorMessage };
            if (data.room_id !== roomId || !data.message) {
                return;
            }
            setMessages(prev => [...prev, data.message as SpectatorMessage]);
        });
    }, [addWSListener, roomId]);

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
            await api.postSpectatorChat(roomId, trimmed);
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
                <span>Spectator chat</span>
                <span>{watcherCount} watching</span>
            </div>
            <div className={styles.messages} ref={scrollRef}>
                {messages.length === 0 ? (
                    <p className={styles.empty}>No messages yet. Say hello.</p>
                ) : (
                    messages.map(m => (
                        <div key={m.id} className={styles.message}>
                            <div className={styles.messageHeader}>
                                <span className={styles.author}>{m.user.display_name}</span>
                                <span className={styles.timestamp}>{new Date(m.created_at).toLocaleTimeString()}</span>
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
                        placeholder="Chat with other watchers..."
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
