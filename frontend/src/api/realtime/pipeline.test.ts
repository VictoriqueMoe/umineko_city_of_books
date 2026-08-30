import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FakeWebSocket } from "../../test-utils/ws";
import type * as BusModule from "./bus";
import type * as PipelineModule from "./pipeline";
import type * as SocketModule from "./socket";

const { getAuthToken, isNativeApp } = vi.hoisted(() => ({
    getAuthToken: vi.fn(),
    isNativeApp: vi.fn(),
}));

vi.mock("../authToken", () => ({ getAuthToken }));
vi.mock("../../platform/capabilities", () => ({ isNativeApp, clientPlatform: () => "web" }));

const API_ORIGIN = "https://whentheycry.social";

let bus: typeof BusModule;
let pipeline: typeof PipelineModule;
let socket: typeof SocketModule;

async function loadRealtime(origin = ""): Promise<void> {
    vi.stubEnv("VITE_API_BASE", origin);
    vi.resetModules();
    bus = await import("./bus");
    pipeline = await import("./pipeline");
    socket = await import("./socket");
}

function lastSocket(): FakeWebSocket {
    const instance = FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
    if (!instance) {
        throw new Error("no websocket was opened");
    }

    return instance;
}

function connect(): void {
    pipeline.ensureRealtimePipeline();
    socket.openRealtimeSocket("user-1");
    lastSocket().onopen?.();
}

function receiveRaw(raw: string): void {
    lastSocket().onmessage?.({ data: raw });
}

function receive(message: { type: string; data?: unknown }): void {
    receiveRaw(JSON.stringify(message));
}

beforeEach(async () => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    getAuthToken.mockReturnValue(null);
    isNativeApp.mockReturnValue(false);
    await loadRealtime();
});

afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
});

describe("realtime pipeline delivery", () => {
    it("hands a packet off the socket to a bus subscriber under its declared name", () => {
        // given
        const handler = vi.fn();
        bus.subscribe("live_games_count", handler);
        connect();

        // when
        receive({ type: "live_games_count", data: { count: 3 } });

        // then
        expect(handler).toHaveBeenCalledTimes(1);
        expect(handler).toHaveBeenCalledWith({ type: "live_games_count", data: { count: 3 } });
    });

    it("dispatches nothing when the packet is not json the parser accepts", () => {
        // given
        const handler = vi.fn();
        bus.subscribe("live_games_count", handler);
        connect();

        // when
        receiveRaw('{"type":"live_games_count","data":');

        // then
        expect(handler).not.toHaveBeenCalled();
    });

    it("dispatches nothing when the type is not a declared event", () => {
        // given
        const handler = vi.fn();
        bus.subscribe("live_games_count", handler);
        connect();

        // when
        receive({ type: "totally_made_up", data: { count: 3 } });

        // then
        expect(handler).not.toHaveBeenCalled();
    });

    it("leaves a subscriber to another event alone", () => {
        // given
        const handler = vi.fn();
        bus.subscribe("typing", handler);
        connect();

        // when
        receive({ type: "live_games_count", data: { count: 3 } });

        // then
        expect(handler).not.toHaveBeenCalled();
    });

    it("installs the frame handler once, however many holders ask for the pipeline", () => {
        // given
        const handler = vi.fn();
        bus.subscribe("live_games_count", handler);
        connect();

        // when
        pipeline.ensureRealtimePipeline();
        pipeline.ensureRealtimePipeline();
        receive({ type: "live_games_count", data: { count: 3 } });

        // then
        expect(handler).toHaveBeenCalledTimes(1);
    });
});

describe("realtime pipeline guards", () => {
    it("drops a guarded event whose payload would corrupt state", () => {
        // given an absent banned flag, which the old router coerced into an unban
        const handler = vi.fn();
        bus.subscribe("ban_changed", handler);
        connect();

        // when
        receive({ type: "ban_changed", data: { user_id: "user-9" } });

        // then
        expect(handler).not.toHaveBeenCalled();
    });

    it("delivers a guarded event whose payload is well formed", () => {
        // given
        const handler = vi.fn();
        bus.subscribe("ban_changed", handler);
        connect();
        const data = { user_id: "user-9", banned: true, ban_reason: "spam" };

        // when
        receive({ type: "ban_changed", data });

        // then
        expect(handler).toHaveBeenCalledWith({ type: "ban_changed", data });
    });

    it("delivers an unguarded event exactly as it arrived, which is today's trust level", () => {
        // given
        const handler = vi.fn();
        bus.subscribe("typing", handler);
        connect();

        // when
        receive({ type: "typing", data: { room_id: null } });

        // then
        expect(handler).toHaveBeenCalledWith({ type: "typing", data: { room_id: null } });
    });
});

describe("the *_frame fast path survives the wiring", () => {
    it("delivers a media url inside a frame untouched, because the frame skips normalisation", async () => {
        // given
        await loadRealtime(API_ORIGIN);
        const handler = vi.fn();
        bus.subscribe("game_pong_frame", handler);
        connect();

        // when
        receive({ type: "game_pong_frame", data: { room_id: "room-1", avatar_url: "/uploads/a.png" } });

        // then
        expect(handler).toHaveBeenCalledWith({
            type: "game_pong_frame",
            data: { room_id: "room-1", avatar_url: "/uploads/a.png" },
        });
    });

    it("absolutises the identical payload when it arrives under a non-frame event", async () => {
        // given
        await loadRealtime(API_ORIGIN);
        const handler = vi.fn();
        bus.subscribe("chat_audio", handler);
        connect();

        // when
        receive({ type: "chat_audio", data: { room_id: "room-1", avatar_url: "/uploads/a.png" } });

        // then
        expect(handler).toHaveBeenCalledWith({
            type: "chat_audio",
            data: { room_id: "room-1", avatar_url: `${API_ORIGIN}/uploads/a.png` },
        });
    });
});

describe("an event the contract does not list", () => {
    it("reports the unknown name once so a server event added without events.ts is not silent", async () => {
        // given
        const reported: unknown[] = [];
        const onError = (event: ErrorEvent) => {
            reported.push(event.error);
        };
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
        window.addEventListener("error", onError);
        connect();

        // when
        receive({ type: "an_event_nobody_declared", data: {} });
        receive({ type: "an_event_nobody_declared", data: {} });

        // then
        const calls = consoleError.mock.calls.filter(call => JSON.stringify(call).includes("an_event_nobody_declared"));
        expect(calls).toHaveLength(1);

        window.removeEventListener("error", onError);
    });

    it("still dispatches nothing for it, so an unknown event cannot reach a handler", () => {
        // given
        vi.spyOn(console, "error").mockImplementation(() => {});
        const handler = vi.fn();
        bus.subscribe(["notification"], handler);
        connect();

        // when
        receive({ type: "an_event_nobody_declared", data: {} });

        // then
        expect(handler).not.toHaveBeenCalled();
    });
});

describe("realtime reconnect epoch", () => {
    it("starts at zero before the socket has ever opened", () => {
        // given
        pipeline.ensureRealtimePipeline();

        // then
        expect(pipeline.getRealtimeEpoch()).toBe(0);
    });

    it("counts one for the first connection", () => {
        // when
        connect();

        // then
        expect(pipeline.getRealtimeEpoch()).toBe(1);
    });

    it("does not move while messages are arriving", () => {
        // given
        connect();

        // when
        receive({ type: "live_games_count", data: { count: 1 } });
        receive({ type: "live_games_count", data: { count: 2 } });
        receive({ type: "typing", data: { room_id: "room-1" } });

        // then
        expect(pipeline.getRealtimeEpoch()).toBe(1);
    });

    it("counts one more for a reconnect", () => {
        // given
        vi.useFakeTimers();
        vi.spyOn(Math, "random").mockReturnValue(1);
        connect();

        // when
        lastSocket().onclose?.();
        vi.advanceTimersByTime(1000);
        lastSocket().onopen?.();

        // then
        expect(pipeline.getRealtimeEpoch()).toBe(2);
    });

    it("notifies an epoch subscriber once per connection", () => {
        // given
        const listener = vi.fn();
        pipeline.subscribeRealtimeEpoch(listener);

        // when
        connect();
        receive({ type: "live_games_count", data: { count: 1 } });

        // then
        expect(listener).toHaveBeenCalledTimes(1);
    });

    it("stops notifying an epoch subscriber that has unsubscribed", () => {
        // given
        const listener = vi.fn();
        const unsubscribe = pipeline.subscribeRealtimeEpoch(listener);

        // when
        unsubscribe();
        connect();

        // then
        expect(listener).not.toHaveBeenCalled();
        expect(pipeline.getRealtimeEpoch()).toBe(1);
    });
});
