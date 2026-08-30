import React, { useEffect, useMemo, useRef, useState } from "react";
import { usePostPlayerChat, usePostSpectatorChat } from "../../../hooks/mutations/gameRoom";
import { usePlayerChat, useSpectatorChat } from "../../../hooks/queries/gameRoom";
import { useAuth } from "../../../hooks/useAuth";
import { useGameChatMessages, type GameChatEventName } from "../../../hooks/useGameChatMessages";
import { Button } from "../../Button/Button";
import { RelativeTimestamp } from "../../RelativeTimestamp/RelativeTimestamp";
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
    wsType: GameChatEventName;
}

function configFor(variant: GameChatVariant, watcherCount: number): VariantConfig {
    if (variant === "player") {
        return {
            title: "Player chat",
            rightMeta: "Private",
            emptyText: "Only you and your opponent can see this chat.",
            placeholder: "Message your opponent...",
            wsType: "player_chat_message",
        };
    }
    return {
        title: "Spectator chat",
        rightMeta: `${watcherCount} watching`,
        emptyText: "No messages yet. Say hello.",
        placeholder: "Chat with other watchers...",
        wsType: "spectator_chat_message",
    };
}

export function GameChat({ roomId, variant, watcherCount = 0 }: GameChatProps) {
    const isPlayer = variant === "player";
    const spectatorHistory = useSpectatorChat(roomId, !isPlayer);
    const playerHistory = usePlayerChat(roomId, isPlayer);
    const postSpectatorMessage = usePostSpectatorChat(roomId);
    const postPlayerMessage = usePostPlayerChat(roomId);
    const history = isPlayer ? playerHistory : spectatorHistory;
    const post = isPlayer ? postPlayerMessage : postSpectatorMessage;

    const cfg = useMemo(() => configFor(variant, watcherCount), [variant, watcherCount]);
    const { user } = useAuth();
    const messages = useGameChatMessages(roomId, cfg.wsType, history);
    const [body, setBody] = useState("");
    const [error, setError] = useState("");
    const scrollRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
        }
    }, [messages.length]);

    async function handleSend() {
        const trimmed = body.trim();
        if (!trimmed || post.isPending) {
            return;
        }
        setError("");

        try {
            await post.mutateAsync(trimmed);
            setBody("");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to send");
        }
    }

    function handleKey(e: React.KeyboardEvent<HTMLInputElement>) {
        if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            handleSend();
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
                                <RelativeTimestamp
                                    value={m.created_at}
                                    variant="timeOnly"
                                    className={styles.timestamp}
                                />
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
                        dir="auto"
                        className={styles.input}
                        placeholder={cfg.placeholder}
                        value={body}
                        onChange={e => setBody(e.target.value)}
                        onKeyDown={handleKey}
                        maxLength={500}
                        disabled={post.isPending}
                    />
                    <Button
                        variant="primary"
                        size="small"
                        onClick={handleSend}
                        disabled={post.isPending || !body.trim()}
                    >
                        Send
                    </Button>
                </div>
            ) : (
                <div className={styles.disabled}>Sign in to join the chat.</div>
            )}
        </div>
    );
}
