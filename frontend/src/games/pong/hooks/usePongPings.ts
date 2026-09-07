import { useRef, useState } from "react";
import { useRealtimeEvent } from "../../../api/realtime/useRealtime";
import { PONG_FRAME_TYPE, PONG_PING_REFRESH_MS, type PongFrame } from "../types";

export type PongPings = [number | null, number | null];

const IDLE: PongPings = [null, null];

function reading(value: number | undefined): number | null {
    if (typeof value !== "number" || value <= 0) {
        return null;
    }

    return value;
}

export function usePongPings(roomId: string | undefined): PongPings {
    const [pings, setPings] = useState<PongPings>(IDLE);
    const shownAtRef = useRef(0);

    useRealtimeEvent(PONG_FRAME_TYPE, event => {
        const frame: PongFrame | null = event.data;

        if (!frame || frame.room_id !== roomId) {
            return;
        }

        const now = Date.now();
        if (now - shownAtRef.current < PONG_PING_REFRESH_MS) {
            return;
        }

        shownAtRef.current = now;
        setPings(current => {
            const next: PongPings = [reading(frame.ping?.[0]), reading(frame.ping?.[1])];
            if (next[0] === current[0] && next[1] === current[1]) {
                return current;
            }

            return next;
        });
    });

    return pings;
}
