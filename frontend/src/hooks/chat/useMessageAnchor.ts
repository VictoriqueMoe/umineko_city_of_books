import { useCallback, useEffect, useRef, useState } from "react";
import { useLocation } from "react-router";
import type { ChatMessage } from "../../types/api";

const HASH_PREFIX = "#msg-";
const HIGHLIGHT_MS = 3000;
const SETTLE_MS = 300;
const RESETTLE_MS = 600;
const NOT_FOUND = "Couldn't locate that message.";

function messageElement(messageId: string): HTMLElement | null {
    return document.getElementById(`chat-msg-${messageId}`);
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

    const targetMsgId = location.hash.startsWith(HASH_PREFIX) ? location.hash.slice(HASH_PREFIX.length) : null;
    const targetMsgCreatedAt = new URLSearchParams(location.search).get("at") ?? undefined;
    const pendingTargetMsgId = targetMsgId && handledHash !== targetMsgId ? targetMsgId : null;

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
                    }
                });
            }

            return;
        }

        const t = setTimeout(() => {
            const el = messageElement(pendingTargetMsgId);
            if (el) {
                el.scrollIntoView({ behavior: "smooth", block: "center" });
                setHighlightedMsgId(pendingTargetMsgId);
                setHandledHash(pendingTargetMsgId);
            }
        }, SETTLE_MS);

        return () => clearTimeout(t);
    }, [pendingTargetMsgId, targetMsgCreatedAt, messages, loadUntilMessage]);

    useEffect(() => {
        if (!highlightedMsgId) {
            return;
        }

        const t = setTimeout(() => setHighlightedMsgId(null), HIGHLIGHT_MS);

        return () => clearTimeout(t);
    }, [highlightedMsgId]);

    const handleJumpToMessage = useCallback(
        async (messageId: string, targetCreatedAt?: string) => {
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
        [messages, loadUntilMessage, onError],
    );

    return { highlightedMsgId, handleJumpToMessage };
}
