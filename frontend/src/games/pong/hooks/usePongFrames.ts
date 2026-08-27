import { useEffect, useRef, type RefObject } from "react";
import { useNotifications } from "../../../hooks/useNotifications";
import type { WSMessage } from "../../../types/api";
import { pushFrame, type BufferedFrame } from "../interpolate";
import { PONG_FRAME_TYPE, type PongFrame } from "../types";

export function usePongFrames(roomId: string | undefined): RefObject<BufferedFrame[]> {
    const { addWSListener } = useNotifications();
    const bufferRef = useRef<BufferedFrame[]>([]);

    useEffect(() => {
        bufferRef.current = [];

        return addWSListener((msg: WSMessage) => {
            if (msg.type !== PONG_FRAME_TYPE) {
                return;
            }
            const frame = msg.data as PongFrame | null;
            if (!frame || frame.room_id !== roomId) {
                return;
            }
            bufferRef.current = pushFrame(bufferRef.current, frame, Date.now());
        });
    }, [addWSListener, roomId]);

    return bufferRef;
}
