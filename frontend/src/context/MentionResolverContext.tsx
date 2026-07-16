import { type PropsWithChildren, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { resolveUsernames } from "../api/endpoints";
import { MentionResolverContext, type MentionResolverContextValue } from "./mentionResolverContextValue";

const BATCH_DELAY_MS = 50;

export function MentionResolverProvider({ children }: PropsWithChildren) {
    const [known, setKnown] = useState<ReadonlyMap<string, boolean>>(new Map());
    const requestedRef = useRef<Set<string>>(new Set());
    const queueRef = useRef<Set<string>>(new Set());
    const timerRef = useRef<number | null>(null);

    const flush = useCallback(() => {
        timerRef.current = null;
        const batch = Array.from(queueRef.current);
        queueRef.current.clear();
        if (batch.length === 0) {
            return;
        }

        resolveUsernames(batch)
            .then(res => {
                const existing = new Set(res.usernames.map(u => u.toLowerCase()));
                setKnown(prev => {
                    const next = new Map(prev);
                    for (const name of batch) {
                        next.set(name, existing.has(name));
                    }
                    return next;
                });
            })
            .catch(() => {
                for (const name of batch) {
                    requestedRef.current.delete(name);
                }
            });
    }, []);

    const request = useCallback(
        (username: string) => {
            const key = username.toLowerCase();
            if (requestedRef.current.has(key)) {
                return;
            }

            requestedRef.current.add(key);
            queueRef.current.add(key);
            if (timerRef.current === null) {
                timerRef.current = window.setTimeout(flush, BATCH_DELAY_MS);
            }
        },
        [flush],
    );

    useEffect(() => {
        return () => {
            if (timerRef.current !== null) {
                window.clearTimeout(timerRef.current);
            }
        };
    }, []);

    const value = useMemo<MentionResolverContextValue>(
        () => ({
            isKnown: (username: string) => known.get(username.toLowerCase()),
            request,
        }),
        [known, request],
    );

    return <MentionResolverContext.Provider value={value}>{children}</MentionResolverContext.Provider>;
}
