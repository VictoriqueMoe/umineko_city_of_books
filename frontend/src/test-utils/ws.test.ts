import { renderHook } from "@testing-library/react";
import { useSyncExternalStore } from "react";
import { describe, expect, it, vi } from "vitest";
import { subscribe } from "../api/realtime/bus";
import { emitRealtimeEvent, emitSettledRealtimeEvent, makeWSHarness } from "./ws";

describe("makeWSHarness delivery", () => {
    it("hands an emitted event to a handler subscribed to that name", () => {
        // given
        const ws = makeWSHarness();
        const handler = vi.fn();
        ws.subscribe("typing", handler);

        // when
        ws.emit({ type: "typing", data: { room_id: "room-1", user_id: "user-1" } });

        // then
        expect(handler).toHaveBeenCalledWith({ type: "typing", data: { room_id: "room-1", user_id: "user-1" } });
    });

    it("leaves a handler subscribed to another name alone", () => {
        // given
        const ws = makeWSHarness();
        const handler = vi.fn();
        ws.subscribe("typing", handler);

        // when
        ws.emit({ type: "chat_message", data: {} });

        // then
        expect(handler).not.toHaveBeenCalled();
    });

    it("delivers every name a multi-name handler asked for", () => {
        // given
        const ws = makeWSHarness();
        const handler = vi.fn();
        ws.subscribe(["stream_live", "stream_offline"], handler);

        // when
        ws.emit({ type: "stream_live", data: {} });
        ws.emit({ type: "stream_offline", data: { streamId: "s1" } });

        // then
        expect(handler).toHaveBeenCalledTimes(2);
    });

    it("does not change who receives the current event when a handler unsubscribes during delivery", () => {
        // given
        const ws = makeWSHarness();
        const second = vi.fn();
        const stopSecond = ws.subscribe("typing", () => {
            stopSecond();
        });
        ws.subscribe("typing", second);

        // when
        ws.emit({ type: "typing", data: {} });

        // then
        expect(second).toHaveBeenCalledOnce();
    });
});

describe("makeWSHarness subscription lifetime", () => {
    it("stops delivering once the returned unsubscribe has run", () => {
        // given
        const ws = makeWSHarness();
        const handler = vi.fn();
        const stop = ws.subscribe("typing", handler);

        // when
        stop();
        ws.emit({ type: "typing", data: {} });

        // then
        expect(handler).not.toHaveBeenCalled();
    });

    it("records every unsubscribe on one spy, so a test can assert the consumer let go", () => {
        // given
        const ws = makeWSHarness();
        const stop = ws.subscribe("typing", vi.fn());

        // when
        stop();

        // then
        expect(ws.unsubscribe).toHaveBeenCalledOnce();
    });

    it("keeps a resubscription alive when a stale unsubscribe runs a second time", () => {
        // given
        const ws = makeWSHarness();
        const handler = vi.fn();
        const stop = ws.subscribe("typing", handler);
        stop();
        ws.subscribe("typing", handler);

        // when
        stop();
        ws.emit({ type: "typing", data: {} });

        // then
        expect(handler).toHaveBeenCalledOnce();
    });
});

describe("makeWSHarness reconnects", () => {
    it("starts the epoch at zero and raises it on every reconnect", () => {
        // given
        const ws = makeWSHarness();

        // when
        ws.reconnect();
        ws.reconnect();

        // then
        expect(ws.getEpoch()).toBe(2);
    });

    it("re-renders a consumer reading the epoch through useSyncExternalStore", () => {
        // given
        const ws = makeWSHarness();
        const { result } = renderHook(() => useSyncExternalStore(ws.subscribeEpoch, ws.getEpoch));
        expect(result.current).toBe(0);

        // when
        ws.reconnect();

        // then
        expect(result.current).toBe(1);
    });

    it("stops telling a listener that has gone away", () => {
        // given
        const ws = makeWSHarness();
        const listener = vi.fn();
        const stop = ws.subscribeEpoch(listener);

        // when
        stop();
        ws.reconnect();

        // then
        expect(listener).not.toHaveBeenCalled();
    });
});

describe("emitRealtimeEvent", () => {
    it("delivers onto the real bus, for a consumer that may not import it", () => {
        // given
        const handler = vi.fn();
        const stop = subscribe("secret_closed", handler);

        // when
        emitRealtimeEvent({ type: "secret_closed", data: { secret_id: "secret-1" } });
        stop();

        // then
        expect(handler).toHaveBeenCalledWith({ type: "secret_closed", data: { secret_id: "secret-1" } });
    });

    it("leaves a handler subscribed to another name alone", () => {
        // given
        const handler = vi.fn();
        const stop = subscribe("secret_closed", handler);

        // when
        emitRealtimeEvent({ type: "secret_solved", data: {} });
        stop();

        // then
        expect(handler).not.toHaveBeenCalled();
    });
});

describe("emitSettledRealtimeEvent", () => {
    it("lets the query client's notifications land before it returns", async () => {
        // given
        const handler = vi.fn();
        const stop = subscribe("secret_closed", handler);
        let notified = false;

        // when
        await emitSettledRealtimeEvent({ type: "secret_closed", data: { secret_id: "secret-1" } });
        setTimeout(() => {
            notified = true;
        }, 0);
        await emitSettledRealtimeEvent({ type: "secret_solved", data: {} });
        stop();

        // then
        expect(handler).toHaveBeenCalledTimes(1);
        expect(notified).toBe(true);
    });
});

describe("makeWSHarness outbound", () => {
    it("records the commands a consumer sends", () => {
        // given
        const ws = makeWSHarness();

        // when
        ws.sendRealtime({ type: "join_room", data: { room_id: "room-1" } });

        // then
        expect(ws.sendRealtime).toHaveBeenCalledWith({ type: "join_room", data: { room_id: "room-1" } });
    });
});
