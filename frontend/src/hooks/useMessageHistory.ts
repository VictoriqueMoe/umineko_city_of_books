import { type Dispatch, type SetStateAction, useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ChatMessage } from "../types/api";
import { fetchRoomMessages, fetchRoomMessagesBefore } from "../api/queries/chat";

const PAGE_SIZE = 50;
const AT_BOTTOM_THRESHOLD = 80;

export interface ScrollToBottomOptions {
    force?: boolean;
}

interface RoomState {
    roomId: string | undefined;
    messages: ChatMessage[];
    hasMore: boolean;
}

export function useMessageHistory(roomId: string | undefined) {
    const [state, setState] = useState<RoomState>({ roomId, messages: [], hasMore: false });
    const [loadingMore, setLoadingMore] = useState(false);
    const loadingMoreRef = useRef(false);
    const containerRef = useRef<HTMLDivElement>(null);
    const endRef = useRef<HTMLDivElement>(null);
    const suppressScrollToBottom = useRef(false);
    const isAtBottomRef = useRef(true);
    const currentRoomIdRef = useRef<string | undefined>(roomId);
    useEffect(() => {
        currentRoomIdRef.current = roomId;
    }, [roomId]);
    const messages = useMemo<ChatMessage[]>(() => (state.roomId === roomId ? state.messages : []), [state, roomId]);
    const hasMore = state.roomId === roomId ? state.hasMore : false;

    const computeIsAtBottom = useCallback(() => {
        const container = containerRef.current;
        if (!container) {
            return true;
        }
        return container.scrollHeight - container.scrollTop - container.clientHeight < AT_BOTTOM_THRESHOLD;
    }, []);

    const scrollToBottom = useCallback((opts?: ScrollToBottomOptions) => {
        if (suppressScrollToBottom.current) {
            return;
        }
        if (!opts?.force && !isAtBottomRef.current) {
            return;
        }
        isAtBottomRef.current = true;
        requestAnimationFrame(() => {
            endRef.current?.scrollIntoView({ behavior: "smooth" });
        });
    }, []);

    const scrollToBottomInstant = useCallback((opts?: ScrollToBottomOptions) => {
        if (suppressScrollToBottom.current) {
            return;
        }
        if (!opts?.force && !isAtBottomRef.current) {
            return;
        }
        isAtBottomRef.current = true;
        requestAnimationFrame(() => {
            endRef.current?.scrollIntoView();
        });
    }, []);

    useEffect(() => {
        loadingMoreRef.current = false;
        suppressScrollToBottom.current = false;
        isAtBottomRef.current = true;
        if (!roomId) {
            return;
        }
        let cancelled = false;
        fetchRoomMessages(roomId, PAGE_SIZE)
            .then(res => {
                if (cancelled || currentRoomIdRef.current !== roomId) {
                    return;
                }
                setState({
                    roomId,
                    messages: res.messages,
                    hasMore: res.messages.length < res.total,
                });
                setLoadingMore(false);
                setTimeout(() => endRef.current?.scrollIntoView(), 50);
            })
            .catch(() => {
                if (cancelled || currentRoomIdRef.current !== roomId) {
                    return;
                }
                setState({ roomId, messages: [], hasMore: false });
            });

        return () => {
            cancelled = true;
        };
    }, [roomId]);

    const setMessages: Dispatch<SetStateAction<ChatMessage[]>> = useCallback(updater => {
        setState(prev => {
            const base = prev.roomId === currentRoomIdRef.current ? prev.messages : [];
            const next = typeof updater === "function" ? updater(base) : updater;
            return {
                roomId: currentRoomIdRef.current,
                messages: next,
                hasMore: prev.roomId === currentRoomIdRef.current ? prev.hasMore : false,
            };
        });
    }, []);

    const setHasMore = useCallback((value: boolean) => {
        setState(prev => ({ ...prev, hasMore: value }));
    }, []);

    const loadOlder = useCallback(async () => {
        if (!roomId || loadingMoreRef.current || !hasMore) {
            return;
        }
        const current = messages;
        if (current.length === 0) {
            return;
        }
        const oldest = current[0];
        const beforeCursor = `${oldest.created_at}|${oldest.id}`;
        loadingMoreRef.current = true;
        setLoadingMore(true);
        suppressScrollToBottom.current = true;
        try {
            const container = containerRef.current;
            const prevScrollHeight = container ? container.scrollHeight : 0;
            const res = await fetchRoomMessagesBefore(roomId, beforeCursor, PAGE_SIZE);
            if (res.messages.length === 0) {
                setHasMore(false);
            } else {
                setMessages(prev => {
                    const existing = new Set(prev.map(message => message.id));
                    const olderUnique: ChatMessage[] = [];
                    for (let i = 0; i < res.messages.length; i++) {
                        const message = res.messages[i];
                        if (!existing.has(message.id)) {
                            olderUnique.push(message);
                            existing.add(message.id);
                        }
                    }
                    return [...olderUnique, ...prev];
                });
                if (container) {
                    requestAnimationFrame(() => {
                        container.scrollTop = container.scrollHeight - prevScrollHeight;
                    });
                }
            }
        } catch {
        } finally {
            loadingMoreRef.current = false;
            setLoadingMore(false);
            setTimeout(() => {
                suppressScrollToBottom.current = false;
            }, 200);
        }
    }, [roomId, hasMore, messages, setHasMore, setMessages]);

    const handleScroll = useCallback(() => {
        const container = containerRef.current;
        if (!container) {
            return;
        }
        isAtBottomRef.current = computeIsAtBottom();
        if (loadingMore || !hasMore) {
            return;
        }
        if (container.scrollTop < 100) {
            loadOlder();
        }
    }, [loadOlder, loadingMore, hasMore, computeIsAtBottom]);

    const loadUntilMessage = useCallback(
        async (messageId: string, targetCreatedAt?: string, maxPages = 20): Promise<boolean> => {
            if (!roomId) {
                return false;
            }
            let pages = 0;
            suppressScrollToBottom.current = true;
            try {
                if (targetCreatedAt) {
                    const cursor = `${targetCreatedAt}|ffffffff-ffff-ffff-ffff-ffffffffffff`;
                    const res = await fetchRoomMessagesBefore(roomId, cursor, PAGE_SIZE);
                    let foundInBatch = false;
                    setMessages(prev => {
                        const existing = new Set(prev.map(m => m.id));
                        const merged = prev.slice();
                        for (const msg of res.messages) {
                            if (!existing.has(msg.id)) {
                                merged.push(msg);
                                existing.add(msg.id);
                            }
                            if (msg.id === messageId) {
                                foundInBatch = true;
                            }
                        }
                        merged.sort((a, b) => {
                            const ta = Date.parse(a.created_at);
                            const tb = Date.parse(b.created_at);
                            if (ta !== tb) {
                                return ta - tb;
                            }
                            return a.id.localeCompare(b.id);
                        });
                        return merged;
                    });
                    if (foundInBatch) {
                        return true;
                    }
                }
                while (pages < maxPages) {
                    let found = false;
                    let oldestCursor = "";
                    let keepGoing = true;
                    setMessages(prev => {
                        found = prev.some(m => m.id === messageId);
                        if (prev.length === 0) {
                            keepGoing = false;
                        } else {
                            const oldest = prev[0];
                            oldestCursor = `${oldest.created_at}|${oldest.id}`;
                        }
                        return prev;
                    });
                    if (found) {
                        return true;
                    }
                    if (!keepGoing) {
                        return false;
                    }
                    const res = await fetchRoomMessagesBefore(roomId, oldestCursor, PAGE_SIZE);
                    if (res.messages.length === 0) {
                        setHasMore(false);
                        return false;
                    }
                    let foundInBatch = false;
                    setMessages(prev => {
                        const existing = new Set(prev.map(m => m.id));
                        const olderUnique: ChatMessage[] = [];
                        for (const msg of res.messages) {
                            if (!existing.has(msg.id)) {
                                olderUnique.push(msg);
                                existing.add(msg.id);
                            }
                            if (msg.id === messageId) {
                                foundInBatch = true;
                            }
                        }
                        return [...olderUnique, ...prev];
                    });
                    if (foundInBatch) {
                        return true;
                    }
                    pages++;
                }
                return false;
            } finally {
                setTimeout(() => {
                    suppressScrollToBottom.current = false;
                }, 200);
            }
        },
        [roomId, setMessages, setHasMore],
    );

    const addMessage = useCallback(
        (message: ChatMessage) => {
            setMessages(prev => {
                const idx = prev.findIndex(m => m.id === message.id);
                if (idx !== -1) {
                    const next = prev.slice();
                    next[idx] = message;
                    return next;
                }
                return [...prev, message];
            });
        },
        [setMessages],
    );

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
        loadUntilMessage,
    };
}
