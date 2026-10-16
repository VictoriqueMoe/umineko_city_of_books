import { useCallback, useEffect, useRef, useState } from "react";
import { useLocation } from "react-router";
import type { ChatMessage } from "../../types/api";

const HASH_PREFIX = "#msg-";
const HIGHLIGHT_MS = 3000;
const SETTLE_MS = 300;
const RESETTLE_MS = 600;
const ANCHOR_HOLD_MS = 3000;
const ANCHOR_VISIBLE_RATIO = 0.5;
const USER_SCROLL_INPUTS = ["wheel", "touchmove", "keydown", "pointerdown"] as const;
const USER_SCROLL_LISTENER: AddEventListenerOptions = { capture: true, passive: true };
const NOT_FOUND = "Couldn't locate that message.";

function messageElement(messageId: string): HTMLElement | null {
    return document.getElementById(`chat-msg-${messageId}`);
}

function holdInView(el: HTMLElement, release: () => void): () => void {
    const observer = new IntersectionObserver(
        entries => {
            for (const entry of entries) {
                if (entry.intersectionRatio < ANCHOR_VISIBLE_RATIO) {
                    el.scrollIntoView({ behavior: "auto", block: "center" });
                }
            }
        },
        { threshold: ANCHOR_VISIBLE_RATIO },
    );
    observer.observe(el);

    const giveUp = setTimeout(release, ANCHOR_HOLD_MS);
    for (const type of USER_SCROLL_INPUTS) {
        document.addEventListener(type, release, USER_SCROLL_LISTENER);
    }

    return () => {
        observer.disconnect();
        clearTimeout(giveUp);
        for (const type of USER_SCROLL_INPUTS) {
            document.removeEventListener(type, release, USER_SCROLL_LISTENER);
        }
    };
}

export interface MessageAnchorOptions {
    messages: readonly ChatMessage[];
    loadUntilMessage: (messageId: string, targetCreatedAt?: string) => Promise<boolean>;
    onError: (message: string) => void;
}

export interface MessageAnchor {
    highlightedMsgId: string | null;
    handleJumpToMessage: (messageId: string, targetCreatedAt?: string) => Promise<void>;
}

export function useMessageAnchor(options: MessageAnchorOptions): MessageAnchor {
    const { messages, loadUntilMessage, onError } = options;
    const location = useLocation();

    const [highlightedMsgId, setHighlightedMsgId] = useState<string | null>(null);
    const [handledHash, setHandledHash] = useState<string | null>(null);
    const hashLoadRef = useRef<string | null>(null);
    const releaseHoldRef = useRef<(() => void) | null>(null);

    const targetMsgId = location.hash.startsWith(HASH_PREFIX) ? location.hash.slice(HASH_PREFIX.length) : null;
    const targetMsgCreatedAt = new URLSearchParams(location.search).get("at") ?? undefined;
    const pendingTargetMsgId = targetMsgId && handledHash !== targetMsgId ? targetMsgId : null;

    const releaseAnchor = useCallback(() => {
        releaseHoldRef.current?.();
        releaseHoldRef.current = null;
    }, []);

    useEffect(() => {
        return releaseAnchor;
    }, [releaseAnchor]);

    useEffect(() => {
        if (!pendingTargetMsgId || messages.length === 0) {
            return;
        }

        if (!messages.some(m => m.id === pendingTargetMsgId)) {
            if (hashLoadRef.current !== pendingTargetMsgId) {
                hashLoadRef.current = pendingTargetMsgId;
                loadUntilMessage(pendingTargetMsgId, targetMsgCreatedAt).then(found => {
                    if (!found) {
                        setHandledHash(pendingTargetMsgId);
                        onError(NOT_FOUND);
                    }
                });
            }

            return;
        }

        const msgId = pendingTargetMsgId;
        let awaitElement: MutationObserver | null = null;

        const land = (el: HTMLElement) => {
            releaseAnchor();
            el.scrollIntoView({ behavior: "auto", block: "center" });
            releaseHoldRef.current = holdInView(el, releaseAnchor);
            setHighlightedMsgId(msgId);
            setHandledHash(msgId);
        };

        const t = setTimeout(() => {
            const el = messageElement(msgId);
            if (el) {
                land(el);
                return;
            }

            awaitElement = new MutationObserver(() => {
                const mounted = messageElement(msgId);
                if (!mounted) {
                    return;
                }

                awaitElement?.disconnect();
                land(mounted);
            });
            awaitElement.observe(document.body, { childList: true, subtree: true });
        }, SETTLE_MS);

        return () => {
            clearTimeout(t);
            awaitElement?.disconnect();
        };
    }, [pendingTargetMsgId, targetMsgCreatedAt, messages, loadUntilMessage, onError, releaseAnchor]);

    useEffect(() => {
        if (!highlightedMsgId) {
            return;
        }

        const t = setTimeout(() => setHighlightedMsgId(null), HIGHLIGHT_MS);

        return () => clearTimeout(t);
    }, [highlightedMsgId]);

    const handleJumpToMessage = useCallback(
        async (messageId: string, targetCreatedAt?: string) => {
            releaseAnchor();

            const scrollToEl = (smooth: boolean) => {
                const el = messageElement(messageId);
                if (el) {
                    el.scrollIntoView({ behavior: smooth ? "smooth" : "auto", block: "center" });
                    setHighlightedMsgId(messageId);
                }
            };

            if (messages.some(m => m.id === messageId)) {
                scrollToEl(true);
                return;
            }

            const found = await loadUntilMessage(messageId, targetCreatedAt);
            if (!found) {
                onError(NOT_FOUND);
                return;
            }

            requestAnimationFrame(() => scrollToEl(false));
            setTimeout(() => scrollToEl(false), SETTLE_MS);
            setTimeout(() => scrollToEl(true), RESETTLE_MS);
        },
        [messages, loadUntilMessage, onError, releaseAnchor],
    );

    return { highlightedMsgId, handleJumpToMessage };
}
