import { useEffect, useRef, useState } from "react";
import type { WatchPartyMessage } from "../../../types/api";
import { Button } from "../../Button/Button";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { formatTimeOfDay } from "../../../utils/time";
import styles from "./WatchParty.module.css";

interface WatchPartyChatProps {
    messages: WatchPartyMessage[];
    viewerUserId: string;
    onSend: (body: string) => Promise<void>;
}

export function WatchPartyChat({ messages, viewerUserId, onSend }: WatchPartyChatProps) {
    const [draft, setDraft] = useState("");
    const [busy, setBusy] = useState(false);
    const scrollRef = useRef<HTMLDivElement | null>(null);
    const textareaRef = useRef<HTMLTextAreaElement | null>(null);

    useEffect(() => {
        const el = scrollRef.current;
        if (!el) {
            return;
        }
        el.scrollTop = el.scrollHeight;
    }, [messages.length]);

    const canSend = draft.trim() !== "" && !busy;

    const handleSend = async () => {
        if (!canSend) {
            return;
        }
        const body = draft.trim();
        setBusy(true);
        try {
            await onSend(body);
            setDraft("");
        } finally {
            setBusy(false);
            textareaRef.current?.focus();
        }
    };

    const handleKey = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
        if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
            e.preventDefault();
            handleSend();
        }
    };

    return (
        <aside className={styles.chatPanel}>
            <div className={styles.chatHeader}>Party chat</div>
            <div className={styles.chatScroll} ref={scrollRef}>
                {messages.length === 0 && <div className={styles.chatEmpty}>No messages yet. Say hi.</div>}
                {messages.map(m => {
                    if (m.kind === "system") {
                        return (
                            <div key={m.id} className={styles.chatSystem}>
                                <span className={styles.chatSystemBody}>{m.body}</span>
                                <span className={styles.chatSystemTime}>{formatTimeOfDay(m.created_at)}</span>
                            </div>
                        );
                    }
                    if (!m.sender) {
                        return null;
                    }
                    const isSelf = m.sender.id === viewerUserId;
                    return (
                        <div key={m.id} className={`${styles.chatMsg}${isSelf ? ` ${styles.chatMsgSelf}` : ""}`}>
                            <div className={styles.chatMsgHeader}>
                                <ProfileLink user={m.sender} size="small" />
                                <span className={styles.chatMsgTime}>{formatTimeOfDay(m.created_at)}</span>
                            </div>
                            <div className={styles.chatMsgBody}>{m.body}</div>
                        </div>
                    );
                })}
            </div>
            <div className={styles.chatComposer}>
                <textarea
                    ref={textareaRef}
                    className={styles.chatTextarea}
                    placeholder="Type a message..."
                    value={draft}
                    onChange={e => setDraft(e.target.value)}
                    onKeyDown={handleKey}
                    rows={2}
                    maxLength={2000}
                />
                <Button variant="primary" size="small" onClick={handleSend} disabled={!canSend}>
                    {busy ? "..." : "Send"}
                </Button>
            </div>
        </aside>
    );
}
