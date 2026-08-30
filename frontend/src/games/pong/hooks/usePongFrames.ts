import { useEffect, useRef, type RefObject } from "react";
import { useRealtimeEvent } from "../../../api/realtime/useRealtime";
import { pushFrame, type BufferedFrame } from "../interpolate";
import { PONG_FRAME_TYPE, type PongFrame } from "../types";

export function usePongFrames(roomId: string | undefined): RefObject<BufferedFrame[]> {
    const bufferRef = useRef<BufferedFrame[]>([]);

    useEffect(() => {
        bufferRef.current = [];
    }, [roomId]);

    useRealtimeEvent(PONG_FRAME_TYPE, event => {
        const frame: PongFrame | null = event.data;

        if (!frame || frame.room_id !== roomId) {
            return;
        }

        bufferRef.current = pushFrame(bufferRef.current, frame, Date.now());
    });

    return bufferRef;
}
