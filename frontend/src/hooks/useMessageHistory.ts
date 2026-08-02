import {
    type Dispatch,
    type SetStateAction,
    useCallback,
    useEffect,
    useLayoutEffect,
    useMemo,
    useRef,
    useState,
} from "react";
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

export function useMessageHistory(roomId: string | undefined, maxMessages?: number) {
    const [state, setState] = useState<RoomState>({ roomId, messages: [], hasMore: false });
    const [loadingMore, setLoadingMore] = useState(false);
    const loadingMoreRef = useRef(false);
    const containerElRef = useRef<HTMLDivElement | null>(null);
    const contentElRef = useRef<HTMLDivElement | null>(null);
    const endRef = useRef<HTMLDivElement>(null);
    const observerRef = useRef<ResizeObserver | null>(null);
    const suppressScrollToBottom = useRef(false);
    const isAtBottomRef = useRef(true);
    const currentRoomIdRef = useRef<string | undefined>(roomId);
    const messagesRef = useRef<{ roomId: string | undefined; messages: ChatMessage[] }>({ roomId, messages: [] });
    useEffect(() => {
        currentRoomIdRef.current = roomId;
    }, [roomId]);
    const messages = useMemo<ChatMessage[]>(() => (state.roomId === roomId ? state.messages : []), [state, roomId]);
    const hasMore = state.roomId === roomId ? state.hasMore : false;

    const computeIsAtBottom = useCallback(() => {
        const container = containerElRef.current;
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
            const container = containerElRef.current;
            if (container) {
                container.scrollTo({ top: container.scrollHeight, behavior: "smooth" });
            }
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
        const container = containerElRef.current;
        if (container) {
            container.scrollTop = container.scrollHeight;
        }
    }, []);

    const snapToBottomIfPinned = useCallback(() => {
        const container = containerElRef.current;
        if (!container || suppressScrollToBottom.current || !isAtBottomRef.current) {
            return;
        }
        container.scrollTop = container.scrollHeight;
    }, []);

    const ensureObserver = useCallback(() => {
        if (observerRef.current || typeof ResizeObserver === "undefined") {
            return observerRef.current;
        }
        observerRef.current = new ResizeObserver(() => {
            snapToBottomIfPinned();
        });
        return observerRef.current;
    }, [snapToBottomIfPinned]);

    const containerRef = useCallback(
        (node: HTMLDivElement | null) => {
            const observer = node ? ensureObserver() : observerRef.current;
            if (containerElRef.current && observer) {
                observer.unobserve(containerElRef.current);
            }
            containerElRef.current = node;
            if (node && observer) {
                observer.observe(node);
            }
        },
        [ensureObserver],
    );

    const contentRef = useCallback(
        (node: HTMLDivElement | null) => {
            const observer = node ? ensureObserver() : observerRef.current;
            if (contentElRef.current && observer) {
                observer.unobserve(contentElRef.current);
            }
            contentElRef.current = node;
            if (node && observer) {
                observer.observe(node);
            }
        },
        [ensureObserver],
    );

    useEffect(() => {
        return () => {
            if (observerRef.current) {
                observerRef.current.disconnect();
                observerRef.current = null;
            }
        };
    }, []);

    useLayoutEffect(() => {
        snapToBottomIfPinned();
    }, [messages, snapToBottomIfPinned]);

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
                messagesRef.current = { roomId, messages: res.messages };
                setState({
                    roomId,
                    messages: res.messages,
                    hasMore: res.messages.length < res.total,
                });
                setLoadingMore(false);
                setTimeout(() => {
                    const container = containerElRef.current;
                    if (container) {
                        container.scrollTop = container.scrollHeight;
                    }
                }, 50);
            })
            .catch(() => {
                if (cancelled || currentRoomIdRef.current !== roomId) {
                    return;
                }
                messagesRef.current = { roomId, messages: [] };
                setState({ roomId, messages: [], hasMore: false });
            });

        return () => {
            cancelled = true;
        };
    }, [roomId]);

    const setMessages: Dispatch<SetStateAction<ChatMessage[]>> = useCallback(
        updater => {
            const sameRoom = messagesRef.current.roomId === currentRoomIdRef.current;
            const base = sameRoom ? messagesRef.current.messages : [];
            const next = typeof updater === "function" ? updater(base) : updater;

            const canTrim =
                maxMessages !== undefined &&
                next.length > maxMessages &&
                isAtBottomRef.current &&
                !suppressScrollToBottom.current;
            const applied = canTrim ? next.slice(next.length - maxMessages) : next;
            messagesRef.current = { roomId: currentRoomIdRef.current, messages: applied };

            setState(prev => ({
                roomId: currentRoomIdRef.current,
                messages: applied,
                hasMore: canTrim || (prev.roomId === currentRoomIdRef.current && prev.hasMore),
            }));
        },
        [maxMessages],
    );

    const seedMessages = useCallback((seedRoomId: string, seed: ChatMessage[]) => {
        currentRoomIdRef.current = seedRoomId;
        messagesRef.current = { roomId: seedRoomId, messages: seed };
        setState({ roomId: seedRoomId, messages: seed, hasMore: false });
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
        isAtBottomRef.current = false;
        try {
            const container = containerElRef.current;
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
        const container = containerElRef.current;
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
                    setMessages(prev => {
                        const existing = new Set(prev.map(m => m.id));
                        const merged = prev.slice();
                        for (const msg of res.messages) {
                            if (!existing.has(msg.id)) {
                                merged.push(msg);
                                existing.add(msg.id);
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
                    if (res.messages.some(m => m.id === messageId)) {
                        return true;
                    }
                }
                while (pages < maxPages) {
                    const current = messagesRef.current.roomId === roomId ? messagesRef.current.messages : [];
                    if (current.some(m => m.id === messageId)) {
                        return true;
                    }
                    if (current.length === 0) {
                        return false;
                    }

                    const oldest = current[0];
                    const res = await fetchRoomMessagesBefore(roomId, `${oldest.created_at}|${oldest.id}`, PAGE_SIZE);
                    if (res.messages.length === 0) {
                        setHasMore(false);
                        return false;
                    }

                    setMessages(prev => {
                        const existing = new Set(prev.map(m => m.id));
                        const olderUnique: ChatMessage[] = [];
                        for (const msg of res.messages) {
                            if (!existing.has(msg.id)) {
                                olderUnique.push(msg);
                                existing.add(msg.id);
                            }
                        }
                        return [...olderUnique, ...prev];
                    });
                    if (res.messages.some(m => m.id === messageId)) {
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

    const resync = useCallback(async () => {
        const rid = currentRoomIdRef.current;
        if (!rid) {
            return;
        }

        try {
            const res = await fetchRoomMessages(rid, PAGE_SIZE);
            if (currentRoomIdRef.current !== rid) {
                return;
            }

            setMessages(prev => {
                const existing = new Set(prev.map(m => m.id));
                const fresh: ChatMessage[] = [];
                for (let i = 0; i < res.messages.length; i++) {
                    const message = res.messages[i];
                    if (!existing.has(message.id)) {
                        fresh.push(message);
                    }
                }

                if (fresh.length === 0) {
                    return prev;
                }

                return [...prev, ...fresh];
            });
        } catch {}
    }, [setMessages]);

    return {
        messages,
        setMessages,
        seedMessages,
        hasMore,
        loadingMore,
        containerRef,
        contentRef,
        endRef,
        scrollToBottom,
        scrollToBottomInstant,
        handleScroll,
        addMessage,
        loadUntilMessage,
        resync,
    };
}
