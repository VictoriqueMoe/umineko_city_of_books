import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type * as BusModule from "../../../api/realtime/bus";
import type { RealtimeEvent } from "../../../api/realtime/events";
import { isSimulationFrame } from "../../../api/realtime/parse";
import { closeRealtimeSocket, openRealtimeSocket } from "../../../api/realtime/socket";
import { createTestQueryClient, providerWrapper } from "../../../test-utils/render";
import { FakeWebSocket, makeWSHarness, type RealtimeTestNames, type WSHarness } from "../../../test-utils/ws";
import { PONG_BUFFER, PONG_FRAME_TYPE, type PongFrame } from "../types";
import { usePongFrames } from "./usePongFrames";

const holder = vi.hoisted(() => ({
    ws: null as unknown as WSHarness,
    absolutizeMedia: vi.fn((data: unknown) => data),
}));

vi.mock("../../../api/realtime/bus", async importOriginal => {
    const actual = await importOriginal<typeof BusModule>();

    return {
        ...actual,
        subscribe: (names: RealtimeTestNames, handler: BusModule.RealtimeEventHandler) =>
            holder.ws.subscribe(names, handler),
        dispatch: (event: RealtimeEvent) => {
            holder.ws.emit(event);
        },
    };
});

vi.mock("../../../api/client", async importOriginal => {
    const actual = await importOriginal<Record<string, unknown>>();

    return { ...actual, absolutizeMedia: (data: unknown) => holder.absolutizeMedia(data) };
});

const SESSION_KEY = "pong-frames-test";

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

function lastSocket(): FakeWebSocket {
    const socket = FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
    if (!socket) {
        throw new Error("no websocket was opened");
    }

    return socket;
}

function receiveOffTheWire(message: { type: string; data: unknown }): void {
    lastSocket().onmessage?.({ data: JSON.stringify(message) });
}

beforeEach(() => {
    holder.ws = makeWSHarness();
    holder.absolutizeMedia.mockReset();
    holder.absolutizeMedia.mockImplementation((data: unknown) => data);
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    vi.spyOn(Date, "now").mockReturnValue(1_000_000);
});

afterEach(() => {
    closeRealtimeSocket(SESSION_KEY);
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
});

describe("usePongFrames", () => {
    it("buffers a frame addressed to the room being watched", () => {
        // given
        const { result } = renderHook(() => usePongFrames("room-1"));

        // when
        holder.ws.emit({ type: PONG_FRAME_TYPE, data: makeFrame(50) });

        // then
        expect(result.current.current).toHaveLength(1);
        expect(result.current.current[0].frame.t).toBe(50);
    });

    it("ignores a message of any other type", () => {
        // given
        const { result } = renderHook(() => usePongFrames("room-1"));

        // when
        holder.ws.emit({ type: "game_room_action", data: makeFrame(50) });

        // then
        expect(result.current.current).toHaveLength(0);
    });

    it("asks the bus for the frame name alone, so the type filter is the subscription", () => {
        // when
        renderHook(() => usePongFrames("room-1"));

        // then
        expect(holder.ws.subscribe).toHaveBeenCalledWith(PONG_FRAME_TYPE, expect.any(Function));
        expect(holder.ws.subscribe).toHaveBeenCalledTimes(1);
    });

    it("ignores a frame addressed to another room", () => {
        // given
        const { result } = renderHook(() => usePongFrames("room-1"));

        // when
        holder.ws.emit({ type: PONG_FRAME_TYPE, data: makeFrame(50, { room_id: "room-2" }) });

        // then
        expect(result.current.current).toHaveLength(0);
    });

    it("shrugs off a frame with no payload", () => {
        // given
        const { result } = renderHook(() => usePongFrames("room-1"));

        // when
        holder.ws.emit({ type: PONG_FRAME_TYPE, data: null });

        // then
        expect(result.current.current).toHaveLength(0);
    });

    it("caps the ring at the buffer size", () => {
        // given
        const { result } = renderHook(() => usePongFrames("room-1"));

        // when
        for (let i = 1; i <= PONG_BUFFER + 5; i += 1) {
            holder.ws.emit({ type: PONG_FRAME_TYPE, data: makeFrame(i * 50) });
        }

        // then
        expect(result.current.current).toHaveLength(PONG_BUFFER);
        expect(result.current.current[result.current.current.length - 1].frame.t).toBe((PONG_BUFFER + 5) * 50);
    });

    it("never writes a frame into react-query", () => {
        // given
        const queryClient = createTestQueryClient();
        const setQueryData = vi.spyOn(queryClient, "setQueryData");
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
        renderHook(() => usePongFrames("room-1"), { wrapper: providerWrapper({ queryClient }) });

        // when
        holder.ws.emit({ type: PONG_FRAME_TYPE, data: makeFrame(50) });
        holder.ws.emit({ type: PONG_FRAME_TYPE, data: makeFrame(100) });

        // then
        expect(setQueryData).not.toHaveBeenCalled();
        expect(invalidateQueries).not.toHaveBeenCalled();
    });

    it("empties the buffer when the watched room changes", () => {
        // given
        const { result, rerender } = renderHook(({ roomId }: { roomId: string }) => usePongFrames(roomId), {
            initialProps: { roomId: "room-1" },
        });
        holder.ws.emit({ type: PONG_FRAME_TYPE, data: makeFrame(50) });

        // when
        rerender({ roomId: "room-2" });

        // then
        expect(result.current.current).toHaveLength(0);
    });

    it("lets go of the subscription when the watcher unmounts", () => {
        // given
        const view = renderHook(() => usePongFrames("room-1"));

        // when
        view.unmount();
        holder.ws.emit({ type: PONG_FRAME_TYPE, data: makeFrame(50) });

        // then
        expect(holder.ws.unsubscribe).toHaveBeenCalledTimes(1);
        expect(view.result.current.current).toHaveLength(0);
    });
});

describe("usePongFrames rides the *_frame fast path", () => {
    it("subscribes under a name the parser treats as a simulation frame", () => {
        // then
        expect(isSimulationFrame(PONG_FRAME_TYPE)).toBe(true);
    });

    it("takes a frame off the wire without walking it for media", () => {
        // given
        const { result } = renderHook(() => usePongFrames("room-1"));
        openRealtimeSocket(SESSION_KEY);

        // when
        receiveOffTheWire({ type: PONG_FRAME_TYPE, data: makeFrame(50) });

        // then
        expect(result.current.current).toHaveLength(1);
        expect(result.current.current[0].frame.t).toBe(50);
        expect(holder.absolutizeMedia).not.toHaveBeenCalled();
    });

    it("walks a non-frame event off the same socket, which is the route the frame skipped", () => {
        // given
        renderHook(() => usePongFrames("room-1"));
        openRealtimeSocket(SESSION_KEY);

        // when
        receiveOffTheWire({ type: "game_room_action", data: { room_id: "room-1", avatar_url: "/uploads/a.png" } });

        // then
        expect(holder.absolutizeMedia).toHaveBeenCalledTimes(1);
    });
});
