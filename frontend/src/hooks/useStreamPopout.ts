import { useCallback, useEffect, useRef, useState } from "react";
import {
    notifyStreamChatPopoutClosed,
    openStreamChatPopout,
    readStreamChatPopoutClosed,
} from "../platform/streamChatPopout";

export interface StreamChatPopout {
    poppedOut: boolean;
    popOut: () => void;
    bringBack: () => void;
}

export function useStreamChatPopout(streamId: string | undefined): StreamChatPopout {
    const [poppedOut, setPoppedOut] = useState(false);
    const popoutRef = useRef<Window | null>(null);

    const popOut = useCallback(() => {
        if (!streamId) {
            return;
        }

        const opened = openStreamChatPopout(streamId);
        if (!opened) {
            return;
        }

        popoutRef.current = opened;
        setPoppedOut(true);
        opened.focus();
    }, [streamId]);

    const bringBack = useCallback(() => {
        popoutRef.current?.close();
        popoutRef.current = null;
        setPoppedOut(false);
    }, []);

    useEffect(() => {
        if (!poppedOut || !streamId) {
            return;
        }

        const onMessage = (event: MessageEvent) => {
            if (readStreamChatPopoutClosed(event) !== streamId) {
                return;
            }

            popoutRef.current = null;
            setPoppedOut(false);
        };

        window.addEventListener("message", onMessage);

        return () => {
            window.removeEventListener("message", onMessage);
        };
    }, [poppedOut, streamId]);

    return { poppedOut, popOut, bringBack };
}

export function useStreamChatPopoutReporter(streamId: string | undefined): void {
    useEffect(() => {
        const opener = window.opener as Window | null;
        if (!opener || !streamId) {
            return;
        }

        const notify = () => {
            notifyStreamChatPopoutClosed(opener, streamId);
        };

        window.addEventListener("pagehide", notify);

        return () => {
            window.removeEventListener("pagehide", notify);
        };
    }, [streamId]);
}
