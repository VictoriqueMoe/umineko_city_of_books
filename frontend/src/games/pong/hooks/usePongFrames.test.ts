import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import type { WSMessageHandler } from "../../../context/notificationContextValue";
import { createTestQueryClient, providerWrapper } from "../../../test-utils/render";
import { PONG_BUFFER, PONG_FRAME_TYPE, type PongFrame } from "../types";
import { usePongFrames } from "./usePongFrames";

interface WSHarness {
    addWSListener: Mock<(handler: WSMessageHandler) => () => void>;
    sendWSMessage: Mock<(msg: object) => void>;
    emit: (type: string, data: unknown) => Promise<void>;
}

function makeWSHarness(): WSHarness {
    const handlers: WSMessageHandler[] = [];
    const addWSListener = vi.fn((handler: WSMessageHandler) => {
        handlers.push(handler);
        return () => {
            const index = handlers.indexOf(handler);
            if (index >= 0) {
                handlers.splice(index, 1);
            }
        };
    });
    const sendWSMessage: Mock<(msg: object) => void> = vi.fn();
    const emit = async (type: string, data: unknown) => {
        await act(async () => {
            for (const handler of [...handlers]) {
                handler({ type, data });
            }
        });
    };
    return { addWSListener, sendWSMessage, emit };
}

function makeFrame(t: number, overrides: Partial<PongFrame> = {}): PongFrame {
    return {
        room_id: "room-1",
        t,
        phase: "rally",
        ball_x: t,
        ball_y: 400,
        ball_vx: 900,
        ball_vy: 0,
        paddle_y: [400, 400],
        scores: [0, 0],
        ack: [0, 0],
        connected: [true, true],
        serve_in_ms: 0,
        events: 0,
        ...overrides,
    };
}

beforeEach(() => {
    vi.spyOn(Date, "now").mockReturnValue(1_000_000);
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe("usePongFrames", () => {
    it("buffers a frame addressed to the room being watched", async () => {
        // given
        const ws = makeWSHarness();
        const { result } = renderHook(() => usePongFrames("room-1"), {
            wrapper: providerWrapper({ notification: ws }),
        });

        // when
        await ws.emit(PONG_FRAME_TYPE, makeFrame(50));

        // then
        expect(result.current.current).toHaveLength(1);
        expect(result.current.current[0].frame.t).toBe(50);
    });

    it("ignores a message of any other type", async () => {
        // given
        const ws = makeWSHarness();
        const { result } = renderHook(() => usePongFrames("room-1"), {
            wrapper: providerWrapper({ notification: ws }),
        });

        // when
        await ws.emit("game_room_action", makeFrame(50));

        // then
        expect(result.current.current).toHaveLength(0);
    });

    it("ignores a frame addressed to another room", async () => {
        // given
        const ws = makeWSHarness();
        const { result } = renderHook(() => usePongFrames("room-1"), {
            wrapper: providerWrapper({ notification: ws }),
        });

        // when
        await ws.emit(PONG_FRAME_TYPE, makeFrame(50, { room_id: "room-2" }));

        // then
        expect(result.current.current).toHaveLength(0);
    });

    it("shrugs off a frame with no payload", async () => {
        // given
        const ws = makeWSHarness();
        const { result } = renderHook(() => usePongFrames("room-1"), {
            wrapper: providerWrapper({ notification: ws }),
        });

        // when
        await ws.emit(PONG_FRAME_TYPE, null);

        // then
        expect(result.current.current).toHaveLength(0);
    });

    it("caps the ring at the buffer size", async () => {
        // given
        const ws = makeWSHarness();
        const { result } = renderHook(() => usePongFrames("room-1"), {
            wrapper: providerWrapper({ notification: ws }),
        });

        // when
        for (let i = 1; i <= PONG_BUFFER + 5; i += 1) {
            await ws.emit(PONG_FRAME_TYPE, makeFrame(i * 50));
        }

        // then
        expect(result.current.current).toHaveLength(PONG_BUFFER);
        expect(result.current.current[result.current.current.length - 1].frame.t).toBe((PONG_BUFFER + 5) * 50);
    });

    it("never writes a frame into react-query", async () => {
        // given
        const ws = makeWSHarness();
        const queryClient = createTestQueryClient();
        const setQueryData = vi.spyOn(queryClient, "setQueryData");
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
        renderHook(() => usePongFrames("room-1"), {
            wrapper: providerWrapper({ notification: ws, queryClient }),
        });

        // when
        await ws.emit(PONG_FRAME_TYPE, makeFrame(50));
        await ws.emit(PONG_FRAME_TYPE, makeFrame(100));

        // then
        expect(setQueryData).not.toHaveBeenCalled();
        expect(invalidateQueries).not.toHaveBeenCalled();
    });

    it("empties the buffer when the watched room changes", async () => {
        // given
        const ws = makeWSHarness();
        const { result, rerender } = renderHook(({ roomId }: { roomId: string }) => usePongFrames(roomId), {
            initialProps: { roomId: "room-1" },
            wrapper: providerWrapper({ notification: ws }),
        });
        await ws.emit(PONG_FRAME_TYPE, makeFrame(50));

        // when
        rerender({ roomId: "room-2" });

        // then
        expect(result.current.current).toHaveLength(0);
    });
});
