import { useCallback, useEffect, useRef, useState } from "react";

const TYPING_EXPIRY_MS = 5000;

function sameIds(a: string[], b: string[]): boolean {
    if (a.length !== b.length) {
        return false;
    }
    for (let i = 0; i < a.length; i++) {
        if (a[i] !== b[i]) {
            return false;
        }
    }

    return true;
}

interface TypingState {
    scope: string | undefined;
    ids: string[];
}

export function useTypingIndicator(scope?: string) {
    const [state, setState] = useState<TypingState>({ scope, ids: [] });
    const expiryRef = useRef<Map<string, number>>(new Map());
    const expiryScopeRef = useRef<string | undefined>(scope);
    const tickRef = useRef<ReturnType<typeof setInterval> | null>(null);
    const typingUserIds = state.scope === scope ? state.ids : [];

    const ensureScope = useCallback(() => {
        if (expiryScopeRef.current !== scope) {
            expiryRef.current.clear();
            expiryScopeRef.current = scope;
        }
    }, [scope]);

    const noteTyping = useCallback(
        (userId: string) => {
            ensureScope();
            expiryRef.current.set(userId, Date.now() + TYPING_EXPIRY_MS);

            const ids = Array.from(expiryRef.current.keys());
            setState(prev => (prev.scope === scope && sameIds(prev.ids, ids) ? prev : { scope, ids }));
        },
        [ensureScope, scope],
    );

    const clearUser = useCallback(
        (userId: string) => {
            ensureScope();
            expiryRef.current.delete(userId);

            const ids = Array.from(expiryRef.current.keys());
            setState(prev => (prev.scope === scope && sameIds(prev.ids, ids) ? prev : { scope, ids }));
        },
        [ensureScope, scope],
    );

    const reset = useCallback(() => {
        expiryRef.current.clear();
        expiryScopeRef.current = scope;
        setState({ scope, ids: [] });
    }, [scope]);

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
                setState({ scope: expiryScopeRef.current, ids: Array.from(expiryRef.current.keys()) });
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
