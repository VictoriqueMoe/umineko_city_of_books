import { useCallback, useEffect, useRef } from "react";
import { useMarkChatRoomRead } from "../mutations/chat";

export const MARK_READ_DEBOUNCE_MS = 500;

export function useDebouncedMarkChatRoomRead(delayMs: number = MARK_READ_DEBOUNCE_MS): (roomId: string) => void {
    const { mutate } = useMarkChatRoomRead();
    const mutateRef = useRef(mutate);
    const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const pendingRoomIdRef = useRef<string | null>(null);

    useEffect(() => {
        mutateRef.current = mutate;
    }, [mutate]);

    useEffect(() => {
        return () => {
            if (timerRef.current !== null) {
                clearTimeout(timerRef.current);
                timerRef.current = null;
            }
            pendingRoomIdRef.current = null;
        };
    }, []);

    return useCallback(
        (roomId: string) => {
            if (document.visibilityState !== "visible" || !document.hasFocus()) {
                return;
            }

            pendingRoomIdRef.current = roomId;
            if (timerRef.current !== null) {
                clearTimeout(timerRef.current);
            }

            timerRef.current = setTimeout(() => {
                timerRef.current = null;
                const pending = pendingRoomIdRef.current;
                pendingRoomIdRef.current = null;
                if (pending === null) {
                    return;
                }

                mutateRef.current(pending);
            }, delayMs);
        },
        [delayMs],
    );
}
