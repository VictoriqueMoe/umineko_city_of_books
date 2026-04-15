import { useCallback, useEffect, useRef, useState } from "react";

const TYPING_EXPIRY_MS = 5000;

export function useTypingIndicator() {
    const [typingUserIds, setTypingUserIds] = useState<string[]>([]);
    const expiryRef = useRef<Map<string, number>>(new Map());
    const tickRef = useRef<ReturnType<typeof setInterval> | null>(null);

    const noteTyping = useCallback((userId: string) => {
        expiryRef.current.set(userId, Date.now() + TYPING_EXPIRY_MS);
        setTypingUserIds(Array.from(expiryRef.current.keys()));
    }, []);

    const clearUser = useCallback((userId: string) => {
        expiryRef.current.delete(userId);
        setTypingUserIds(Array.from(expiryRef.current.keys()));
    }, []);

    const reset = useCallback(() => {
        expiryRef.current.clear();
        setTypingUserIds([]);
    }, []);

    useEffect(() => {
        tickRef.current = setInterval(() => {
            const now = Date.now();
            let changed = false;
            for (const [uid, exp] of expiryRef.current) {
                if (exp <= now) {
                    expiryRef.current.delete(uid);
                    changed = true;
                }
            }
            if (changed) {
                setTypingUserIds(Array.from(expiryRef.current.keys()));
            }
        }, 1000);
        return () => {
            if (tickRef.current) {
                clearInterval(tickRef.current);
            }
        };
    }, []);

    return { typingUserIds, noteTyping, clearUser, reset };
}
