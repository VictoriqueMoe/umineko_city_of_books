import { useCallback, useEffect, useRef, type RefObject } from "react";
import { sendRealtime } from "../../../api/realtime/outbound";
import { useRealtimeStatus } from "../../../api/realtime/useRealtime";
import {
    PONG_INPUT_EPSILON,
    PONG_INPUT_INTERVAL_MS,
    PONG_INPUT_KEEPALIVE_MS,
    PONG_INPUT_TYPE,
    PONG_PADDLE_MAX_SPEED,
} from "../types";

export interface UsePongInputOptions {
    roomId: string | undefined;
    enabled: boolean;
    courtHeight: number;
    paddleHeight: number;
    initialY?: number;
}

export interface PongInputHandle {
    targetRef: RefObject<number>;
    setTargetY: (y: number) => void;
    seedTargetY: (y: number) => void;
    setTargetFromPointer: (clientY: number, rect: { top: number; height: number }) => void;
}

const UP_KEYS = new Set(["ArrowUp", "w", "W"]);
const DOWN_KEYS = new Set(["ArrowDown", "s", "S"]);
const EDITABLE_TAGS = new Set(["INPUT", "TEXTAREA", "SELECT"]);

function clamp(value: number, min: number, max: number): number {
    if (value < min) {
        return min;
    }
    if (value > max) {
        return max;
    }
    return value;
}

function isEditableTarget(target: EventTarget | null): boolean {
    if (!(target instanceof HTMLElement)) {
        return false;
    }
    if (target.isContentEditable) {
        return true;
    }

    return EDITABLE_TAGS.has(target.tagName);
}

export function usePongInput(options: UsePongInputOptions): PongInputHandle {
    const { roomId, enabled, courtHeight, paddleHeight, initialY } = options;
    const epoch = useRealtimeStatus();

    const limitsRef = useRef({ courtHeight, paddleHeight });
    const targetRef = useRef(initialY ?? courtHeight / 2);
    const lastSentTargetRef = useRef<number | null>(null);
    const lastSentAtRef = useRef(0);
    const seqRef = useRef(0);
    const seededRef = useRef(false);
    const keysRef = useRef({ up: false, down: false });

    useEffect(() => {
        limitsRef.current = { courtHeight, paddleHeight };
    }, [courtHeight, paddleHeight]);

    const setTargetY = useCallback((y: number) => {
        const { courtHeight: height, paddleHeight: paddle } = limitsRef.current;
        seededRef.current = true;
        targetRef.current = clamp(y, paddle / 2, height - paddle / 2);
    }, []);

    const seedTargetY = useCallback(
        (y: number) => {
            if (seededRef.current) {
                return;
            }
            setTargetY(y);
        },
        [setTargetY],
    );

    const setTargetFromPointer = useCallback(
        (clientY: number, rect: { top: number; height: number }) => {
            if (rect.height <= 0) {
                return;
            }
            setTargetY(((clientY - rect.top) / rect.height) * limitsRef.current.courtHeight);
        },
        [setTargetY],
    );

    useEffect(() => {
        if (!enabled) {
            keysRef.current = { up: false, down: false };
            return;
        }

        const onKeyDown = (event: KeyboardEvent) => {
            if (isEditableTarget(event.target)) {
                return;
            }

            if (UP_KEYS.has(event.key)) {
                keysRef.current.up = true;
                event.preventDefault();
                return;
            }
            if (DOWN_KEYS.has(event.key)) {
                keysRef.current.down = true;
                event.preventDefault();
            }
        };

        const onKeyUp = (event: KeyboardEvent) => {
            if (isEditableTarget(event.target)) {
                return;
            }

            if (UP_KEYS.has(event.key)) {
                keysRef.current.up = false;
                return;
            }
            if (DOWN_KEYS.has(event.key)) {
                keysRef.current.down = false;
            }
        };

        const releaseKeys = () => {
            keysRef.current = { up: false, down: false };
        };

        const onVisibilityChange = () => {
            if (document.hidden) {
                releaseKeys();
            }
        };

        const onFocusIn = (event: FocusEvent) => {
            if (isEditableTarget(event.target)) {
                releaseKeys();
            }
        };

        window.addEventListener("keydown", onKeyDown);
        window.addEventListener("keyup", onKeyUp);
        window.addEventListener("blur", releaseKeys);
        document.addEventListener("visibilitychange", onVisibilityChange);
        document.addEventListener("focusin", onFocusIn);

        return () => {
            window.removeEventListener("keydown", onKeyDown);
            window.removeEventListener("keyup", onKeyUp);
            window.removeEventListener("blur", releaseKeys);
            document.removeEventListener("visibilitychange", onVisibilityChange);
            document.removeEventListener("focusin", onFocusIn);
            keysRef.current = { up: false, down: false };
        };
    }, [enabled]);

    useEffect(() => {
        if (!enabled || !roomId) {
            return;
        }

        lastSentTargetRef.current = null;
        lastSentAtRef.current = Number.NEGATIVE_INFINITY;

        let handle = 0;
        let previous = 0;

        const step = (now: number) => {
            handle = window.requestAnimationFrame(step);

            const elapsed = previous === 0 ? 0 : Math.min((now - previous) / 1000, 0.1);
            previous = now;

            const keys = keysRef.current;
            if (keys.up !== keys.down) {
                const direction = keys.up ? -1 : 1;
                setTargetY(targetRef.current + direction * PONG_PADDLE_MAX_SPEED * elapsed);
            }

            if (!seededRef.current) {
                return;
            }

            const target = targetRef.current;
            const sinceSent = now - lastSentAtRef.current;
            if (sinceSent < PONG_INPUT_INTERVAL_MS) {
                return;
            }

            const lastSent = lastSentTargetRef.current;
            const moved = lastSent === null || Math.abs(target - lastSent) >= PONG_INPUT_EPSILON;
            if (!moved && sinceSent < PONG_INPUT_KEEPALIVE_MS) {
                return;
            }

            seqRef.current += 1;
            lastSentTargetRef.current = target;
            lastSentAtRef.current = now;
            sendRealtime({
                type: PONG_INPUT_TYPE,
                data: { room_id: roomId, payload: { y: Math.round(target), seq: seqRef.current } },
            });
        };

        handle = window.requestAnimationFrame(step);

        return () => {
            window.cancelAnimationFrame(handle);
        };
    }, [enabled, epoch, roomId, setTargetY]);

    return { targetRef, setTargetY, seedTargetY, setTargetFromPointer };
}
