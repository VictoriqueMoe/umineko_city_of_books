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
import {
    beforeCursor,
    emptyMessageList,
    MAX_MESSAGE_ID,
    messageListReducer,
    type MessageListAction,
    type MessageListState,
} from "../domain/chat/messageStore";
import { fetchRoomMessages, fetchRoomMessagesBefore } from "./queries/chat";

const PAGE_SIZE = 50;
const AT_BOTTOM_THRESHOLD = 80;
const HOLD_HEADROOM = 150;

export interface ScrollToBottomOptions {
    force?: boolean;
}

export function useMessageHistory(roomId: string | undefined, maxMessages?: number) {
    const [state, setState] = useState<MessageListState>(() => emptyMessageList(roomId));
    const [loadingMore, setLoadingMore] = useState(false);
    const loadingMoreRef = useRef(false);
    const containerElRef = useRef<HTMLDivElement | null>(null);
    const contentElRef = useRef<HTMLDivElement | null>(null);
    const endRef = useRef<HTMLDivElement>(null);
    const observerRef = useRef<ResizeObserver | null>(null);
    const suppressScrollToBottom = useRef(false);
    const pointerHold = useRef(false);
    const isAtBottomRef = useRef(true);
    const currentRoomIdRef = useRef<string | undefined>(roomId);
    const stateRef = useRef<MessageListState>(state);
    useEffect(() => {
        currentRoomIdRef.current = roomId;
    }, [roomId]);
    const messages = useMemo<ChatMessage[]>(() => (state.roomId === roomId ? state.messages : []), [state, roomId]);
    const hasMore = state.roomId === roomId ? state.hasMore : false;

    const dispatch = useCallback((action: MessageListAction) => {
        const next = messageListReducer(stateRef.current, action);
        stateRef.current = next;
        setState(next);
    }, []);

    const trimLimit = useCallback(() => {
        if (!isAtBottomRef.current || suppressScrollToBottom.current) {
            return undefined;
        }

        if (pointerHold.current && maxMessages !== undefined) {
            return maxMessages + HOLD_HEADROOM;
        }

        return maxMessages;
    }, [maxMessages]);

    const heldMessages = useCallback((forRoomId: string | undefined) => {
        return stateRef.current.roomId === forRoomId ? stateRef.current.messages : [];
    }, []);

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
        if (!opts?.force && (pointerHold.current || !isAtBottomRef.current)) {
            return;
        }
        pointerHold.current = false;
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
        if (!opts?.force && (pointerHold.current || !isAtBottomRef.current)) {
            return;
        }
        pointerHold.current = false;
        isAtBottomRef.current = true;
        const container = containerElRef.current;
        if (container) {
            container.scrollTop = container.scrollHeight;
        }
    }, []);

    const snapToBottomIfPinned = useCallback(() => {
        const container = containerElRef.current;
        if (!container || suppressScrollToBottom.current || pointerHold.current || !isAtBottomRef.current) {
            return;
        }
        container.scrollTop = container.scrollHeight;
    }, []);

    const holdAutoScroll = useCallback((event: PointerEvent) => {
        if (event.pointerType === "mouse") {
            pointerHold.current = true;
        }
    }, []);

    const releaseAutoScroll = useCallback(() => {
        if (!pointerHold.current) {
            return;
        }

        pointerHold.current = false;
        snapToBottomIfPinned();
    }, [snapToBottomIfPinned]);

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
            const previous = containerElRef.current;
            if (previous) {
                if (observer) {
                    observer.unobserve(previous);
                }
                previous.removeEventListener("pointerenter", holdAutoScroll);
                previous.removeEventListener("pointermove", holdAutoScroll);
                previous.removeEventListener("pointerleave", releaseAutoScroll);
            }

            containerElRef.current = node;
            pointerHold.current = false;

            if (node) {
                if (observer) {
                    observer.observe(node);
                }
                node.addEventListener("pointerenter", holdAutoScroll, { passive: true });
                node.addEventListener("pointermove", holdAutoScroll, { passive: true });
                node.addEventListener("pointerleave", releaseAutoScroll, { passive: true });
            }
        },
        [ensureObserver, holdAutoScroll, releaseAutoScroll],
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
        pointerHold.current = false;
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
                dispatch({ type: "historyLoaded", roomId, messages: res.messages, total: res.total });
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
                dispatch({ type: "historyFailed", roomId });
            });

        return () => {
            cancelled = true;
        };
    }, [roomId, dispatch]);

    const setMessages: Dispatch<SetStateAction<ChatMessage[]>> = useCallback(
        updater => {
            const patch = typeof updater === "function" ? updater : () => updater;

            dispatch({
                type: "messagesPatched",
                roomId: currentRoomIdRef.current,
                patch,
                limit: trimLimit(),
            });
        },
        [dispatch, trimLimit],
    );

    const seedMessages = useCallback(
        (seedRoomId: string, seed: ChatMessage[]) => {
            currentRoomIdRef.current = seedRoomId;
            dispatch({ type: "seeded", roomId: seedRoomId, messages: seed });
        },
        [dispatch],
    );

    const setHasMore = useCallback(
        (value: boolean) => {
            dispatch({ type: "hasMoreChanged", hasMore: value });
        },
        [dispatch],
    );

    const loadOlder = useCallback(async () => {
        if (!roomId || loadingMoreRef.current || !hasMore) {
            return;
        }
        const current = messages;
        if (current.length === 0) {
            return;
        }
        const oldest = current[0];
        const cursor = beforeCursor(oldest);
        loadingMoreRef.current = true;
        setLoadingMore(true);
        suppressScrollToBottom.current = true;
        isAtBottomRef.current = false;
        try {
            const container = containerElRef.current;
            const prevScrollHeight = container ? container.scrollHeight : 0;
            const res = await fetchRoomMessagesBefore(roomId, cursor, PAGE_SIZE);
            if (res.messages.length === 0) {
                setHasMore(false);
            } else {
                dispatch({
                    type: "olderLoaded",
                    roomId: currentRoomIdRef.current,
                    messages: res.messages,
                    limit: trimLimit(),
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
    }, [roomId, hasMore, messages, setHasMore, dispatch, trimLimit]);

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
                    const cursor = beforeCursor({ created_at: targetCreatedAt, id: MAX_MESSAGE_ID });
                    const res = await fetchRoomMessagesBefore(roomId, cursor, PAGE_SIZE);
                    dispatch({
                        type: "gapLoaded",
                        roomId: currentRoomIdRef.current,
                        messages: res.messages,
                        limit: trimLimit(),
                    });
                    if (res.messages.some(m => m.id === messageId)) {
                        return true;
                    }
                }
                while (pages < maxPages) {
                    const current = heldMessages(roomId);
                    if (current.some(m => m.id === messageId)) {
                        return true;
                    }
                    if (current.length === 0) {
                        return false;
                    }

                    const oldest = current[0];
                    const res = await fetchRoomMessagesBefore(roomId, beforeCursor(oldest), PAGE_SIZE);
                    if (res.messages.length === 0) {
                        setHasMore(false);
                        return false;
                    }

                    dispatch({
                        type: "olderLoaded",
                        roomId: currentRoomIdRef.current,
                        messages: res.messages,
                        limit: trimLimit(),
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
        [roomId, dispatch, trimLimit, heldMessages, setHasMore],
    );

    const addMessage = useCallback(
        (message: ChatMessage) => {
            dispatch({
                type: "messageUpserted",
                roomId: currentRoomIdRef.current,
                message,
                limit: trimLimit(),
            });
        },
        [dispatch, trimLimit],
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

            dispatch({
                type: "resynced",
                roomId: currentRoomIdRef.current,
                messages: res.messages,
                limit: trimLimit(),
            });
        } catch {}
    }, [dispatch, trimLimit]);

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
