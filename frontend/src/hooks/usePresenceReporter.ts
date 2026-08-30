import { useEffect, useRef } from "react";
import { REALTIME_COMMANDS, sendRealtime, type ViewerState } from "../api/realtime/outbound";
import { useRealtimeStatus } from "../api/realtime/useRealtime";

const IDLE_AFTER_MS = 60_000;

export function usePresenceReporter(roomId: string | undefined): void {
    const lastSentRef = useRef<ViewerState | null>(null);
    const idleTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const epoch = useRealtimeStatus();

    useEffect(() => {
        if (!roomId) {
            return;
        }
        lastSentRef.current = null;

        const report = (state: ViewerState) => {
            if (lastSentRef.current === state) {
                return;
            }
            lastSentRef.current = state;
            sendRealtime({ type: REALTIME_COMMANDS.VIEWER_STATE, data: { room_id: roomId, state } });
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
    }, [roomId, epoch]);
}
