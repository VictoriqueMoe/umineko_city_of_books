import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type * as OutboundModule from "../api/realtime/outbound";
import { makeWSHarness, type WSHarness } from "../test-utils/ws";
import { usePresenceReporter } from "./usePresenceReporter";

const IDLE_AFTER_MS = 60_000;

const holder = vi.hoisted(() => ({ ws: null as unknown as WSHarness }));

vi.mock("../api/realtime/pipeline", () => ({
    ensureRealtimePipeline: () => {},
    getRealtimeEpoch: () => holder.ws.getEpoch(),
    subscribeRealtimeEpoch: (listener: () => void) => holder.ws.subscribeEpoch(listener),
}));

vi.mock("../api/realtime/outbound", async importOriginal => {
    const actual = await importOriginal<typeof OutboundModule>();

    return { ...actual, sendRealtime: (command: OutboundModule.RealtimeCommand) => holder.ws.sendRealtime(command) };
});

function setVisibility(state: DocumentVisibilityState): void {
    Object.defineProperty(document, "visibilityState", { configurable: true, get: () => state });
}

function activity(): void {
    window.dispatchEvent(new Event("mousemove"));
}

function visibilityChanged(state: DocumentVisibilityState): void {
    setVisibility(state);
    document.dispatchEvent(new Event("visibilitychange"));
}

function renderReporter(roomId: string | undefined) {
    return renderHook((next: string | undefined) => usePresenceReporter(next), { initialProps: roomId });
}

function viewerState(state: "active" | "idle", roomId = "room-1") {
    return { type: "viewer_state", data: { room_id: roomId, state } };
}

beforeEach(() => {
    holder.ws = makeWSHarness();
});

describe("usePresenceReporter", () => {
    afterEach(() => {
        Reflect.deleteProperty(document, "visibilityState");
    });

    it("announces the viewer as active as soon as the room opens", () => {
        // given
        setVisibility("visible");

        // when
        renderReporter("room-1");

        // then
        expect(holder.ws.sendRealtime).toHaveBeenCalledExactlyOnceWith(viewerState("active"));
    });

    it("announces the viewer as idle when the tab is already hidden", () => {
        // given
        setVisibility("hidden");

        // when
        renderReporter("room-1");

        // then
        expect(holder.ws.sendRealtime).toHaveBeenCalledExactlyOnceWith(viewerState("idle"));
    });

    it("stays quiet while there is no room to report on", () => {
        // given
        setVisibility("visible");

        // when
        renderReporter(undefined);

        // then
        expect(holder.ws.sendRealtime).not.toHaveBeenCalled();
    });

    it("does not repeat a state it has already reported", () => {
        // given
        setVisibility("visible");
        renderReporter("room-1");

        // when
        activity();
        activity();

        // then
        expect(holder.ws.sendRealtime).toHaveBeenCalledOnce();
    });

    it("reports the viewer idle after a minute without interaction", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        renderReporter("room-1");

        // when
        vi.advanceTimersByTime(IDLE_AFTER_MS);

        // then
        expect(holder.ws.sendRealtime).toHaveBeenNthCalledWith(2, viewerState("idle"));
    });

    it("pushes the idle deadline back every time the viewer interacts", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        renderReporter("room-1");

        // when
        vi.advanceTimersByTime(50_000);
        activity();
        vi.advanceTimersByTime(50_000);

        // then
        expect(holder.ws.sendRealtime).toHaveBeenCalledOnce();
        vi.advanceTimersByTime(11_000);
        expect(holder.ws.sendRealtime).toHaveBeenNthCalledWith(2, viewerState("idle"));
    });

    it("ignores interaction that arrives while the tab is hidden", () => {
        // given
        vi.useFakeTimers();
        setVisibility("hidden");
        renderReporter("room-1");

        // when
        activity();
        vi.advanceTimersByTime(IDLE_AFTER_MS * 2);

        // then
        expect(holder.ws.sendRealtime).toHaveBeenCalledExactlyOnceWith(viewerState("idle"));
    });

    it("reports the viewer idle when the tab is hidden and stops the idle countdown", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        renderReporter("room-1");

        // when
        visibilityChanged("hidden");
        vi.advanceTimersByTime(IDLE_AFTER_MS * 2);

        // then
        expect(holder.ws.sendRealtime).toHaveBeenCalledTimes(2);
        expect(holder.ws.sendRealtime).toHaveBeenNthCalledWith(2, viewerState("idle"));
    });

    it("reports the viewer active again when the tab comes back", () => {
        // given
        setVisibility("hidden");
        renderReporter("room-1");

        // when
        visibilityChanged("visible");

        // then
        expect(holder.ws.sendRealtime).toHaveBeenNthCalledWith(2, viewerState("active"));
    });

    it("reports again for the new room when the room changes", () => {
        // given
        setVisibility("visible");
        const { rerender } = renderReporter("room-1");

        // when
        rerender("room-2");

        // then
        expect(holder.ws.sendRealtime).toHaveBeenNthCalledWith(2, viewerState("active", "room-2"));
    });

    it("re-announces the viewer when the websocket reconnects", () => {
        // given
        setVisibility("visible");
        renderReporter("room-1");

        // when
        holder.ws.reconnect();

        // then
        expect(holder.ws.sendRealtime).toHaveBeenNthCalledWith(2, viewerState("active"));
    });

    it("stops listening once the room is closed", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        const { unmount } = renderReporter("room-1");

        // when
        unmount();
        activity();
        visibilityChanged("hidden");
        vi.advanceTimersByTime(IDLE_AFTER_MS * 2);

        // then
        expect(holder.ws.sendRealtime).toHaveBeenCalledOnce();
    });
});
