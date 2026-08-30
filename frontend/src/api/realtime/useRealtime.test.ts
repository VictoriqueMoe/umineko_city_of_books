import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { FakeWebSocket } from "../../test-utils/ws";
import type * as BusModule from "./bus";
import type { RealtimeEventName } from "./events";
import type * as SocketModule from "./socket";
import type * as UseRealtimeModule from "./useRealtime";

const { getAuthToken, isNativeApp, subscribeCalls } = vi.hoisted(() => ({
    getAuthToken: vi.fn(),
    isNativeApp: vi.fn(),
    subscribeCalls: { count: 0 },
}));

vi.mock("../authToken", () => ({ getAuthToken }));
vi.mock("../../platform/capabilities", () => ({ isNativeApp, clientPlatform: () => "web" }));

vi.mock("./bus", async importOriginal => {
    const actual = await importOriginal<typeof BusModule>();

    function countingSubscribe<K extends RealtimeEventName>(
        names: K | readonly K[],
        handler: BusModule.RealtimeEventHandler<K>,
    ): BusModule.RealtimeUnsubscribe {
        subscribeCalls.count += 1;

        return actual.subscribe(names, handler);
    }

    return { ...actual, subscribe: countingSubscribe };
});

let hooks: typeof UseRealtimeModule;
let socket: typeof SocketModule;

function lastSocket(): FakeWebSocket {
    const instance = FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
    if (!instance) {
        throw new Error("no websocket was opened");
    }

    return instance;
}

function connect(): void {
    act(() => {
        socket.openRealtimeSocket("user-1");
        lastSocket().onopen?.();
    });
}

function reconnect(): void {
    act(() => {
        socket.closeRealtimeSocket("user-1");
        socket.openRealtimeSocket("user-1");
        lastSocket().onopen?.();
    });
}

function receive(message: { type: string; data?: unknown }): void {
    act(() => {
        lastSocket().onmessage?.({ data: JSON.stringify(message) });
    });
}

beforeEach(async () => {
    FakeWebSocket.instances = [];
    subscribeCalls.count = 0;
    vi.stubGlobal("WebSocket", FakeWebSocket);
    getAuthToken.mockReturnValue(null);
    isNativeApp.mockReturnValue(false);
    vi.resetModules();
    socket = await import("./socket");
    hooks = await import("./useRealtime");
});

describe("useRealtimeEvent", () => {
    it("delivers an event straight off the socket, having opened the pipeline itself", () => {
        // given
        const handler = vi.fn();
        renderHook(() => {
            hooks.useRealtimeEvent("live_games_count", handler);
        });
        connect();

        // when
        receive({ type: "live_games_count", data: { count: 3 } });

        // then
        expect(handler).toHaveBeenCalledTimes(1);
        expect(handler).toHaveBeenCalledWith({ type: "live_games_count", data: { count: 3 } });
    });

    it("hands the event to the handler from the latest render, not the one that subscribed", () => {
        // given a handler closing over a value that changes on every render
        const seen: number[] = [];
        const { rerender } = renderHook(
            ({ value }: { value: number }) => {
                hooks.useRealtimeEvent("live_games_count", () => {
                    seen.push(value);
                });
            },
            { initialProps: { value: 1 } },
        );
        connect();

        // when
        rerender({ value: 2 });
        rerender({ value: 3 });
        receive({ type: "live_games_count", data: { count: 7 } });

        // then
        expect(seen).toEqual([3]);
    });

    it("does not resubscribe when a re-render brings a new array literal and a new handler closure", () => {
        // given
        const handler = vi.fn();
        const { rerender } = renderHook(
            ({ value }: { value: number }) => {
                hooks.useRealtimeEvent(["live_games_count", "typing"], () => {
                    handler(value);
                });
            },
            { initialProps: { value: 1 } },
        );
        connect();

        // when
        rerender({ value: 2 });
        rerender({ value: 3 });
        rerender({ value: 4 });
        receive({ type: "typing", data: { room_id: "room-1" } });

        // then
        expect(subscribeCalls.count).toBe(1);
        expect(handler).toHaveBeenCalledWith(4);
    });

    it("resubscribes when the wanted names really change", () => {
        // given
        const handler = vi.fn();
        const { rerender } = renderHook(
            ({ name }: { name: RealtimeEventName }) => {
                hooks.useRealtimeEvent(name, handler);
            },
            { initialProps: { name: "live_games_count" as RealtimeEventName } },
        );
        connect();

        // when
        rerender({ name: "typing" });
        receive({ type: "live_games_count", data: { count: 3 } });
        receive({ type: "typing", data: { room_id: "room-1" } });

        // then
        expect(subscribeCalls.count).toBe(2);
        expect(handler).toHaveBeenCalledTimes(1);
        expect(handler).toHaveBeenCalledWith({ type: "typing", data: { room_id: "room-1" } });
    });

    it("stops delivering once the consumer unmounts", () => {
        // given
        const handler = vi.fn();
        const view = renderHook(() => {
            hooks.useRealtimeEvent("live_games_count", handler);
        });
        connect();

        // when
        view.unmount();
        receive({ type: "live_games_count", data: { count: 3 } });

        // then
        expect(handler).not.toHaveBeenCalled();
    });
});

describe("useRealtimeStatus", () => {
    it("reports zero until the socket has connected", () => {
        // given
        const { result } = renderHook(() => hooks.useRealtimeStatus());

        // then
        expect(result.current).toBe(0);
    });

    it("moves once when the socket connects", () => {
        // given
        const { result } = renderHook(() => hooks.useRealtimeStatus());

        // when
        connect();

        // then
        expect(result.current).toBe(1);
    });

    it("keeps the same value across messages, so a consumer effect keyed on it does not refire", () => {
        // given
        const seen: number[] = [];
        renderHook(() => {
            seen.push(hooks.useRealtimeStatus());
        });
        connect();

        // when
        receive({ type: "live_games_count", data: { count: 1 } });
        receive({ type: "live_games_count", data: { count: 2 } });
        receive({ type: "typing", data: { room_id: "room-1" } });

        // then
        expect(seen).toEqual([0, 1]);
    });

    it("moves exactly once more when the socket connects again", () => {
        // given
        const { result } = renderHook(() => hooks.useRealtimeStatus());
        connect();

        // when
        reconnect();

        // then
        expect(result.current).toBe(2);
    });
});
