import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { usePresenceReporter } from "./usePresenceReporter";

const IDLE_AFTER_MS = 60_000;

interface ReporterProps {
    roomId: string | undefined;
    sendWSMessage: (msg: object) => void;
    wsEpoch: number;
}

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

function renderReporter(props: ReporterProps) {
    return renderHook((next: ReporterProps) => usePresenceReporter(next), { initialProps: props });
}

function viewerState(state: "active" | "idle", roomId = "room-1") {
    return { type: "viewer_state", data: { room_id: roomId, state } };
}

describe("usePresenceReporter", () => {
    afterEach(() => {
        Reflect.deleteProperty(document, "visibilityState");
    });

    it("announces the viewer as active as soon as the room opens", () => {
        // given
        setVisibility("visible");
        const sendWSMessage = vi.fn();

        // when
        renderReporter({ roomId: "room-1", sendWSMessage, wsEpoch: 0 });

        // then
        expect(sendWSMessage).toHaveBeenCalledExactlyOnceWith(viewerState("active"));
    });

    it("announces the viewer as idle when the tab is already hidden", () => {
        // given
        setVisibility("hidden");
        const sendWSMessage = vi.fn();

        // when
        renderReporter({ roomId: "room-1", sendWSMessage, wsEpoch: 0 });

        // then
        expect(sendWSMessage).toHaveBeenCalledExactlyOnceWith(viewerState("idle"));
    });

    it("stays quiet while there is no room to report on", () => {
        // given
        setVisibility("visible");
        const sendWSMessage = vi.fn();

        // when
        renderReporter({ roomId: undefined, sendWSMessage, wsEpoch: 0 });

        // then
        expect(sendWSMessage).not.toHaveBeenCalled();
    });

    it("does not repeat a state it has already reported", () => {
        // given
        setVisibility("visible");
        const sendWSMessage = vi.fn();
        renderReporter({ roomId: "room-1", sendWSMessage, wsEpoch: 0 });

        // when
        activity();
        activity();

        // then
        expect(sendWSMessage).toHaveBeenCalledOnce();
    });

    it("reports the viewer idle after a minute without interaction", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        const sendWSMessage = vi.fn();
        renderReporter({ roomId: "room-1", sendWSMessage, wsEpoch: 0 });

        // when
        vi.advanceTimersByTime(IDLE_AFTER_MS);

        // then
        expect(sendWSMessage).toHaveBeenNthCalledWith(2, viewerState("idle"));
    });

    it("pushes the idle deadline back every time the viewer interacts", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        const sendWSMessage = vi.fn();
        renderReporter({ roomId: "room-1", sendWSMessage, wsEpoch: 0 });

        // when
        vi.advanceTimersByTime(50_000);
        activity();
        vi.advanceTimersByTime(50_000);

        // then
        expect(sendWSMessage).toHaveBeenCalledOnce();
        vi.advanceTimersByTime(11_000);
        expect(sendWSMessage).toHaveBeenNthCalledWith(2, viewerState("idle"));
    });

    it("ignores interaction that arrives while the tab is hidden", () => {
        // given
        vi.useFakeTimers();
        setVisibility("hidden");
        const sendWSMessage = vi.fn();
        renderReporter({ roomId: "room-1", sendWSMessage, wsEpoch: 0 });

        // when
        activity();
        vi.advanceTimersByTime(IDLE_AFTER_MS * 2);

        // then
        expect(sendWSMessage).toHaveBeenCalledExactlyOnceWith(viewerState("idle"));
    });

    it("reports the viewer idle when the tab is hidden and stops the idle countdown", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        const sendWSMessage = vi.fn();
        renderReporter({ roomId: "room-1", sendWSMessage, wsEpoch: 0 });

        // when
        visibilityChanged("hidden");
        vi.advanceTimersByTime(IDLE_AFTER_MS * 2);

        // then
        expect(sendWSMessage).toHaveBeenCalledTimes(2);
        expect(sendWSMessage).toHaveBeenNthCalledWith(2, viewerState("idle"));
    });

    it("reports the viewer active again when the tab comes back", () => {
        // given
        setVisibility("hidden");
        const sendWSMessage = vi.fn();
        renderReporter({ roomId: "room-1", sendWSMessage, wsEpoch: 0 });

        // when
        visibilityChanged("visible");

        // then
        expect(sendWSMessage).toHaveBeenNthCalledWith(2, viewerState("active"));
    });

    it("reports again for the new room when the room changes", () => {
        // given
        setVisibility("visible");
        const sendWSMessage = vi.fn();
        const { rerender } = renderReporter({ roomId: "room-1", sendWSMessage, wsEpoch: 0 });

        // when
        rerender({ roomId: "room-2", sendWSMessage, wsEpoch: 0 });

        // then
        expect(sendWSMessage).toHaveBeenNthCalledWith(2, viewerState("active", "room-2"));
    });

    it("re-announces the viewer when the websocket reconnects", () => {
        // given
        setVisibility("visible");
        const sendWSMessage = vi.fn();
        const { rerender } = renderReporter({ roomId: "room-1", sendWSMessage, wsEpoch: 0 });

        // when
        rerender({ roomId: "room-1", sendWSMessage, wsEpoch: 1 });

        // then
        expect(sendWSMessage).toHaveBeenNthCalledWith(2, viewerState("active"));
    });

    it("stops listening once the room is closed", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        const sendWSMessage = vi.fn();
        const { unmount } = renderReporter({ roomId: "room-1", sendWSMessage, wsEpoch: 0 });

        // when
        unmount();
        activity();
        visibilityChanged("hidden");
        vi.advanceTimersByTime(IDLE_AFTER_MS * 2);

        // then
        expect(sendWSMessage).toHaveBeenCalledOnce();
    });
});
