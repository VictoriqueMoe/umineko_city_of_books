import { useEffect, useRef } from "react";

const IDLE_AFTER_MS = 60_000;

interface Options {
    roomId: string | undefined;
    sendWSMessage: (msg: object) => void;
}

export function usePresenceReporter({ roomId, sendWSMessage }: Options): void {
    const lastSentRef = useRef<"active" | "idle" | null>(null);
    const idleTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    useEffect(() => {
        if (!roomId) {
            return;
        }

        const report = (state: "active" | "idle") => {
            if (lastSentRef.current === state) {
                return;
            }
            lastSentRef.current = state;
            sendWSMessage({ type: "viewer_state", data: { room_id: roomId, state } });
        };

        const clearIdleTimer = () => {
            if (idleTimerRef.current) {
                clearTimeout(idleTimerRef.current);
                idleTimerRef.current = null;
            }
        };

        const armIdleTimer = () => {
            clearIdleTimer();
            idleTimerRef.current = setTimeout(() => report("idle"), IDLE_AFTER_MS);
        };

        const onActivity = () => {
            if (document.visibilityState !== "visible") {
                return;
            }
            report("active");
            armIdleTimer();
        };

        const onVisibilityChange = () => {
            if (document.visibilityState === "visible") {
                report("active");
                armIdleTimer();
            } else {
                clearIdleTimer();
                report("idle");
            }
        };

        if (document.visibilityState === "visible") {
            report("active");
            armIdleTimer();
        } else {
            report("idle");
        }

        window.addEventListener("mousemove", onActivity);
        window.addEventListener("keydown", onActivity);
        window.addEventListener("click", onActivity);
        window.addEventListener("scroll", onActivity, true);
        window.addEventListener("touchstart", onActivity);
        document.addEventListener("visibilitychange", onVisibilityChange);

        return () => {
            clearIdleTimer();
            lastSentRef.current = null;
            window.removeEventListener("mousemove", onActivity);
            window.removeEventListener("keydown", onActivity);
            window.removeEventListener("click", onActivity);
            window.removeEventListener("scroll", onActivity, true);
            window.removeEventListener("touchstart", onActivity);
            document.removeEventListener("visibilitychange", onVisibilityChange);
        };
    }, [roomId, sendWSMessage]);
}
