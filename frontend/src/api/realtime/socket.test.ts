import { render } from "@testing-library/react";
import { StrictMode, createElement, useEffect } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FakeWebSocket } from "../../test-utils/ws";
import type * as SocketModule from "./socket";

const { getAuthToken, isNativeApp } = vi.hoisted(() => ({
    getAuthToken: vi.fn(),
    isNativeApp: vi.fn(),
}));

vi.mock("../authToken", () => ({ getAuthToken }));
vi.mock("../../platform/capabilities", () => ({ isNativeApp, clientPlatform: () => "web" }));

let socket: typeof SocketModule;

function sockets(): FakeWebSocket[] {
    return FakeWebSocket.instances;
}

function liveSockets(): FakeWebSocket[] {
    return FakeWebSocket.instances.filter(instance => instance.readyState === FakeWebSocket.OPEN);
}

function lastSocket(): FakeWebSocket {
    const instance = FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
    if (!instance) {
        throw new Error("no websocket was opened");
    }

    return instance;
}

function openSocket(instance: FakeWebSocket = lastSocket()): void {
    instance.onopen?.();
}

function emit(msg: { type: string; data?: unknown }, instance: FakeWebSocket = lastSocket()): void {
    instance.onmessage?.({ data: JSON.stringify(msg) });
}

function setVisibility(state: "visible" | "hidden"): void {
    Object.defineProperty(document, "visibilityState", { configurable: true, get: () => state });
}

let effectRuns = 0;

function Subscriber({ sessionKey }: { sessionKey: string }) {
    useEffect(() => {
        effectRuns += 1;
        socket.openRealtimeSocket(sessionKey);
        return () => {
            socket.closeRealtimeSocket(sessionKey);
        };
    }, [sessionKey]);

    return null;
}

function renderInStrictMode(sessionKey: string) {
    return render(createElement(StrictMode, null, createElement(Subscriber, { sessionKey })));
}

beforeEach(async () => {
    FakeWebSocket.instances = [];
    effectRuns = 0;
    vi.stubGlobal("WebSocket", FakeWebSocket);
    getAuthToken.mockReturnValue(null);
    vi.resetModules();
    socket = await import("./socket");
});

afterEach(() => {
    vi.restoreAllMocks();
    Reflect.deleteProperty(document, "visibilityState");
});

describe("realtime socket refcount", () => {
    it("opens a single socket for the first holder", () => {
        // given
        // when
        socket.openRealtimeSocket("user-1");

        // then
        expect(sockets()).toHaveLength(1);
        expect(liveSockets()).toHaveLength(1);
    });

    it("leaves exactly one live socket after an open, a close and an open", () => {
        // given
        socket.openRealtimeSocket("user-1");

        // when
        socket.closeRealtimeSocket("user-1");
        socket.openRealtimeSocket("user-1");

        // then
        expect(liveSockets()).toHaveLength(1);
        expect(lastSocket().readyState).toBe(FakeWebSocket.OPEN);
    });

    it("keeps the socket open while a second holder still has it", () => {
        // given
        socket.openRealtimeSocket("user-1");
        socket.openRealtimeSocket("user-1");
        const opened = lastSocket();

        // when
        socket.closeRealtimeSocket("user-1");

        // then
        expect(sockets()).toHaveLength(1);
        expect(opened.readyState).toBe(FakeWebSocket.OPEN);
        expect(opened.closeCount).toBe(0);
    });

    it("closes the socket once every holder has let go", () => {
        // given
        socket.openRealtimeSocket("user-1");
        socket.openRealtimeSocket("user-1");
        const opened = lastSocket();

        // when
        socket.closeRealtimeSocket("user-1");
        socket.closeRealtimeSocket("user-1");

        // then
        expect(opened.readyState).toBe(FakeWebSocket.CLOSED);
        expect(opened.closeCount).toBe(1);
        expect(liveSockets()).toHaveLength(0);
    });

    it("reuses the live socket instead of opening a second one for another holder", () => {
        // given
        socket.openRealtimeSocket("user-1");

        // when
        socket.openRealtimeSocket("user-1");

        // then
        expect(sockets()).toHaveLength(1);
    });

    it("ignores a close that never had a matching open", () => {
        // given
        // when
        const close = () => socket.closeRealtimeSocket("user-1");

        // then
        expect(close).not.toThrow();
        expect(sockets()).toHaveLength(0);
    });

    it("does not let unmatched closes push the refcount below zero", () => {
        // given
        socket.closeRealtimeSocket("user-1");
        socket.closeRealtimeSocket("user-1");
        socket.closeRealtimeSocket("user-1");

        // when
        socket.openRealtimeSocket("user-1");
        socket.closeRealtimeSocket("user-1");

        // then
        expect(sockets()).toHaveLength(1);
        expect(lastSocket().readyState).toBe(FakeWebSocket.CLOSED);
    });

    it("opens a brand new socket after a full close rather than reviving the dead one", () => {
        // given
        socket.openRealtimeSocket("user-1");
        const first = lastSocket();
        socket.closeRealtimeSocket("user-1");

        // when
        socket.openRealtimeSocket("user-1");
        const second = lastSocket();

        // then
        expect(sockets()).toHaveLength(2);
        expect(second).not.toBe(first);
        expect(first.readyState).toBe(FakeWebSocket.CLOSED);
        expect(second.readyState).toBe(FakeWebSocket.OPEN);
    });
});

describe("realtime socket session key", () => {
    it("replaces the socket when the session key changes", () => {
        // given
        socket.openRealtimeSocket("user-1");
        const first = lastSocket();

        // when
        socket.openRealtimeSocket("user-2");

        // then
        expect(sockets()).toHaveLength(2);
        expect(first.readyState).toBe(FakeWebSocket.CLOSED);
        expect(liveSockets()).toEqual([lastSocket()]);
    });

    it("ignores a stale close from a session key that has already been replaced", () => {
        // given
        socket.openRealtimeSocket("user-1");
        socket.openRealtimeSocket("user-2");
        const second = lastSocket();

        // when
        socket.closeRealtimeSocket("user-1");

        // then
        expect(second.readyState).toBe(FakeWebSocket.OPEN);
        expect(liveSockets()).toEqual([second]);
    });

    it("closes the new socket when its own holder lets go", () => {
        // given
        socket.openRealtimeSocket("user-1");
        socket.openRealtimeSocket("user-2");
        const second = lastSocket();

        // when
        socket.closeRealtimeSocket("user-1");
        socket.closeRealtimeSocket("user-2");

        // then
        expect(second.readyState).toBe(FakeWebSocket.CLOSED);
        expect(liveSockets()).toHaveLength(0);
    });
});

describe("realtime socket under strict mode", () => {
    it("really does mount, unmount and mount again", () => {
        // given
        // when
        renderInStrictMode("user-1");

        // then
        expect(effectRuns).toBe(2);
    });

    it("still holds a live socket after the double mount", () => {
        // given
        // when
        renderInStrictMode("user-1");

        // then
        expect(effectRuns).toBe(2);
        expect(liveSockets()).toHaveLength(1);
        expect(lastSocket().readyState).toBe(FakeWebSocket.OPEN);
    });

    it("closes every socket it opened once the tree unmounts", () => {
        // given
        const view = renderInStrictMode("user-1");

        // when
        view.unmount();

        // then
        expect(liveSockets()).toHaveLength(0);
        for (const instance of sockets()) {
            expect(instance.readyState).toBe(FakeWebSocket.CLOSED);
        }
    });

    it("holds a live socket for the new user after a signed in user changes", () => {
        // given
        const view = renderInStrictMode("user-1");
        const before = sockets().length;

        // when
        view.rerender(createElement(StrictMode, null, createElement(Subscriber, { sessionKey: "user-2" })));

        // then
        expect(sockets().length).toBeGreaterThan(before);
        expect(liveSockets()).toEqual([lastSocket()]);
    });
});

describe("realtime socket url", () => {
    it("opens a socket for a signed in user", () => {
        // given
        const wsOrigin = window.location.origin.replace(/^http/, "ws");

        // when
        socket.openRealtimeSocket("user-1");

        // then
        expect(sockets()).toHaveLength(1);
        expect(lastSocket().url).toBe(`${wsOrigin}/api/v1/ws`);
    });

    it("carries the native session token on the socket url", () => {
        // given the app has no cookie to fall back on
        getAuthToken.mockReturnValue("tok en&1");
        isNativeApp.mockReturnValue(true);

        // when
        socket.openRealtimeSocket("user-1");

        // then
        expect(lastSocket().url).toContain("?token=tok%20en%261");
    });

    it("keeps the session token out of the socket url in a browser", () => {
        // given a same-origin browser session, where the cookie already authenticates
        getAuthToken.mockReturnValue("tok en&1");
        isNativeApp.mockReturnValue(false);

        // when
        socket.openRealtimeSocket("user-1");

        // then the token must never reach proxy or edge access logs
        expect(lastSocket().url).not.toContain("token=");
    });
});

describe("realtime socket send", () => {
    it("sends a message down an open socket", () => {
        // given
        socket.openRealtimeSocket("user-1");
        openSocket();

        // when
        socket.sendRaw(JSON.stringify({ type: "typing", data: { room_id: "room-1" } }));

        // then
        expect(lastSocket().sent).toContain(JSON.stringify({ type: "typing", data: { room_id: "room-1" } }));
    });

    it("drops a message when the socket is not open", () => {
        // given
        socket.openRealtimeSocket("user-1");
        openSocket();
        const opened = lastSocket();
        opened.readyState = FakeWebSocket.CLOSED;

        // when
        socket.sendRaw(JSON.stringify({ type: "typing", data: {} }));

        // then
        expect(opened.sent).toHaveLength(0);
    });
});

describe("realtime socket keepalive", () => {
    it("pings the server while the connection is idle", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        socket.openRealtimeSocket("user-1");
        openSocket();

        // when
        vi.advanceTimersByTime(20_000);

        // then
        expect(lastSocket().sent).toEqual([JSON.stringify({ type: "ping", data: {} })]);
    });

    it("probes a connection that has gone quiet for too long rather than closing it", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        socket.openRealtimeSocket("user-1");
        openSocket();
        const opened = lastSocket();

        // when
        vi.advanceTimersByTime(100_000);

        // then
        expect(opened.sent).toHaveLength(5);
        expect(opened.closeCount).toBe(0);
    });

    it("closes a quiet connection once the probe goes unanswered", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        socket.openRealtimeSocket("user-1");
        openSocket();
        const opened = lastSocket();

        // when
        vi.advanceTimersByTime(100_000);
        vi.advanceTimersByTime(5_000);

        // then
        expect(opened.closeCount).toBe(1);
    });

    it("keeps a quiet connection that answers the probe", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        socket.openRealtimeSocket("user-1");
        openSocket();
        const opened = lastSocket();

        // when
        vi.advanceTimersByTime(100_000);
        vi.advanceTimersByTime(1_000);
        emit({ type: "pong", data: {} }, opened);
        vi.advanceTimersByTime(10_000);

        // then
        expect(opened.closeCount).toBe(0);
    });

    it("keeps the connection alive while messages are still arriving", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        socket.openRealtimeSocket("user-1");
        openSocket();
        const opened = lastSocket();

        // when
        vi.advanceTimersByTime(80_000);
        emit({ type: "chat_message", data: {} }, opened);
        vi.advanceTimersByTime(40_000);

        // then
        expect(opened.closeCount).toBe(0);
    });
});

describe("realtime socket reconnect", () => {
    it("reconnects with a growing backoff after the connection drops", () => {
        // given
        vi.useFakeTimers();
        vi.spyOn(Math, "random").mockReturnValue(1);
        socket.openRealtimeSocket("user-1");
        openSocket();

        // when
        lastSocket().onclose?.();

        // then
        expect(sockets()).toHaveLength(1);
        vi.advanceTimersByTime(1000);
        expect(sockets()).toHaveLength(2);

        lastSocket().onclose?.();
        vi.advanceTimersByTime(1000);
        expect(sockets()).toHaveLength(2);
        vi.advanceTimersByTime(1000);
        expect(sockets()).toHaveLength(3);
    });

    it("spreads the reconnect across the backoff window instead of retrying in lockstep", () => {
        // given
        vi.useFakeTimers();
        vi.spyOn(Math, "random").mockReturnValue(0);
        socket.openRealtimeSocket("user-1");
        openSocket();

        // when
        lastSocket().onclose?.();
        vi.advanceTimersByTime(0);

        // then
        expect(sockets()).toHaveLength(2);
    });

    it("never waits longer than the backoff cap", () => {
        // given
        vi.useFakeTimers();
        vi.spyOn(Math, "random").mockReturnValue(0.999);
        socket.openRealtimeSocket("user-1");
        openSocket();

        // when
        lastSocket().onclose?.();
        vi.advanceTimersByTime(1000);

        // then
        expect(sockets()).toHaveLength(2);
    });

    it("ignores a late close from a socket that has already been replaced", () => {
        // given
        vi.useFakeTimers();
        socket.openRealtimeSocket("user-1");
        openSocket();
        const stale = lastSocket();
        socket.openRealtimeSocket("user-2");
        const live = lastSocket();
        openSocket(live);

        // when
        stale.onclose?.();
        vi.advanceTimersByTime(5000);

        // then
        expect(sockets()).toHaveLength(2);
        socket.sendRaw(JSON.stringify({ type: "typing", data: {} }));
        expect(live.sent).toContain(JSON.stringify({ type: "typing", data: {} }));
    });

    it("never reconnects once the provider has gone away", () => {
        // given
        vi.useFakeTimers();
        socket.openRealtimeSocket("user-1");
        openSocket();
        const opened = lastSocket();

        // when
        socket.closeRealtimeSocket("user-1");
        opened.onclose?.();
        vi.advanceTimersByTime(30_000);

        // then
        expect(sockets()).toHaveLength(1);
    });

    it("closes a socket that errored so the reconnect can take over", () => {
        // given
        socket.openRealtimeSocket("user-1");
        openSocket();
        const opened = lastSocket();

        // when
        opened.onerror?.();

        // then
        expect(opened.closeCount).toBe(1);
    });
});

describe("realtime socket visibility", () => {
    it("pings the server when the tab becomes visible again", () => {
        // given
        setVisibility("visible");
        socket.openRealtimeSocket("user-1");
        openSocket();

        // when
        document.dispatchEvent(new Event("visibilitychange"));

        // then
        expect(lastSocket().sent).toEqual([JSON.stringify({ type: "ping", data: {} })]);
    });

    it("probes a stale socket when the tab becomes visible again rather than closing it", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        socket.openRealtimeSocket("user-1");
        const opened = lastSocket();

        // when
        document.dispatchEvent(new Event("visibilitychange"));

        // then
        expect(opened.sent).toHaveLength(1);
        expect(opened.closeCount).toBe(0);
    });

    it("closes the socket when the visibility probe goes unanswered", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        socket.openRealtimeSocket("user-1");
        const opened = lastSocket();

        // when
        document.dispatchEvent(new Event("visibilitychange"));
        vi.advanceTimersByTime(5_000);

        // then
        expect(opened.closeCount).toBe(1);
    });

    it("keeps the socket when the visibility probe is answered", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        socket.openRealtimeSocket("user-1");
        const opened = lastSocket();

        // when
        document.dispatchEvent(new Event("visibilitychange"));
        vi.advanceTimersByTime(1_000);
        emit({ type: "pong", data: {} }, opened);
        vi.advanceTimersByTime(10_000);

        // then
        expect(opened.closeCount).toBe(0);
    });

    it("does nothing when the tab is being hidden", () => {
        // given
        setVisibility("hidden");
        socket.openRealtimeSocket("user-1");
        openSocket();
        const opened = lastSocket();

        // when
        document.dispatchEvent(new Event("visibilitychange"));

        // then
        expect(opened.sent).toHaveLength(0);
        expect(opened.closeCount).toBe(0);
    });
});
