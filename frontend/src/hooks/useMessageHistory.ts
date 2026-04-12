import { useCallback, useEffect, useRef, useState } from "react";
import type { ChatMessage } from "../types/api";
import { getRoomMessages, getRoomMessagesBefore } from "../api/endpoints";

const PAGE_SIZE = 50;

export function useMessageHistory(roomId: string | undefined) {
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [hasMore, setHasMore] = useState(false);
    const [loadingMore, setLoadingMore] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);
    const endRef = useRef<HTMLDivElement>(null);
    const suppressScrollToBottom = useRef(false);

    const scrollToBottom = useCallback(() => {
        if (suppressScrollToBottom.current) {
            return;
        }
        endRef.current?.scrollIntoView({ behavior: "smooth" });
    }, []);

    const scrollToBottomInstant = useCallback(() => {
        if (suppressScrollToBottom.current) {
            return;
        }
        endRef.current?.scrollIntoView();
    }, []);

    useEffect(() => {
        if (!roomId) {
            setMessages([]);
            setHasMore(false);
            return;
        }
        let cancelled = false;
        setMessages([]);
        setHasMore(false);
        suppressScrollToBottom.current = false;

        getRoomMessages(roomId, PAGE_SIZE)
            .then(res => {
                if (cancelled) {
                    return;
                }
                setMessages(res.messages);
                setHasMore(res.messages.length < res.total);
                setTimeout(() => endRef.current?.scrollIntoView(), 50);
            })
            .catch(() => {
                if (!cancelled) {
                    setMessages([]);
                }
            });

        return () => {
            cancelled = true;
        };
    }, [roomId]);

    const loadOlder = useCallback(async () => {
        if (!roomId || loadingMore || !hasMore) {
            return;
        }
        const current = messages;
        if (current.length === 0) {
            return;
        }
        const oldest = current[0].created_at;
        setLoadingMore(true);
        suppressScrollToBottom.current = true;
        try {
            const container = containerRef.current;
            const prevScrollHeight = container ? container.scrollHeight : 0;
            const res = await getRoomMessagesBefore(roomId, oldest, PAGE_SIZE);
            if (res.messages.length === 0) {
                setHasMore(false);
            } else {
                if (res.messages.length < PAGE_SIZE) {
                    setHasMore(false);
                }
                setMessages(prev => [...res.messages, ...prev]);
                if (container) {
                    requestAnimationFrame(() => {
                        container.scrollTop = container.scrollHeight - prevScrollHeight;
                    });
                }
            }
        } catch {
            setHasMore(false);
        } finally {
            setLoadingMore(false);
            setTimeout(() => {
                suppressScrollToBottom.current = false;
            }, 200);
        }
    }, [roomId, loadingMore, hasMore, messages]);

    const handleScroll = useCallback(() => {
        const container = containerRef.current;
        if (!container || loadingMore || !hasMore) {
            return;
        }
        if (container.scrollTop < 100) {
            loadOlder();
        }
    }, [loadOlder, loadingMore, hasMore]);

    const addMessage = useCallback((message: ChatMessage) => {
        setMessages(prev => {
            const idx = prev.findIndex(m => m.id === message.id);
            if (idx !== -1) {
                const next = prev.slice();
                next[idx] = message;
                return next;
            }
            return [...prev, message];
        });
    }, []);

    return {
        messages,
        setMessages,
        hasMore,
        loadingMore,
        containerRef,
        endRef,
        scrollToBottom,
        scrollToBottomInstant,
        handleScroll,
        addMessage,
    };
}
