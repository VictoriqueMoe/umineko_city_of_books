import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type * as BusModule from "./bus";
import type { RealtimeEvent, RealtimeEventName } from "./events";

let bus: typeof BusModule;
let windowErrors: unknown[] = [];

const nativeQueueMicrotask = queueMicrotask;

const roomEventNames: Record<"typing" | "read", RealtimeEventName> = {
    typing: "typing",
    read: "chat_read",
};

function captureWindowError(event: ErrorEvent): void {
    event.preventDefault();
    windowErrors.push(event.error);
}

function emulateBrowserErrorReporting(): void {
    vi.stubGlobal("queueMicrotask", (callback: VoidFunction) => {
        nativeQueueMicrotask(() => {
            try {
                callback();
            } catch (error) {
                window.dispatchEvent(new ErrorEvent("error", { error, message: String(error), cancelable: true }));
            }
        });
    });
}

async function flushMicrotasks(): Promise<void> {
    await Promise.resolve();
}

function typingEvent(userId: string): RealtimeEvent {
    return { type: "typing", data: { room_id: "room-1", user_id: userId } };
}

function chatReadEvent(total: number): RealtimeEvent {
    return { type: "chat_read", data: { room_id: "room-1", total } };
}

function liveGamesCountEvent(count: number): RealtimeEvent {
    return { type: "live_games_count", data: { count } };
}

beforeEach(async () => {
    windowErrors = [];
    window.addEventListener("error", captureWindowError);
    emulateBrowserErrorReporting();
    vi.resetModules();
    bus = await import("./bus");
});

afterEach(() => {
    window.removeEventListener("error", captureWindowError);
});

describe("realtime bus delivery", () => {
    it("hands an event to a handler subscribed to its name", () => {
        // given
        const handler = vi.fn();
        bus.subscribe("typing", handler);
        const event = typingEvent("user-1");

        // when
        bus.dispatch(event);

        // then
        expect(handler).toHaveBeenCalledTimes(1);
        expect(handler).toHaveBeenCalledWith(event);
    });

    it("leaves a handler subscribed to a different name alone", () => {
        // given
        const handler = vi.fn();
        bus.subscribe("chat_read", handler);

        // when
        bus.dispatch(typingEvent("user-1"));

        // then
        expect(handler).not.toHaveBeenCalled();
    });

    it("hands every subscribed name to a handler that wanted several", () => {
        // given
        const handler = vi.fn();
        bus.subscribe(["typing", "chat_read", "live_games_count"], handler);
        const typing = typingEvent("user-1");
        const read = chatReadEvent(3);
        const count = liveGamesCountEvent(7);

        // when
        bus.dispatch(typing);
        bus.dispatch(read);
        bus.dispatch(count);

        // then
        expect(handler.mock.calls).toEqual([[typing], [read], [count]]);
    });

    it("subscribes to a name that is only known at runtime", () => {
        // given
        const handler = vi.fn();
        const wanted = roomEventNames.typing;
        bus.subscribe(wanted, handler);
        const event = typingEvent("user-1");

        // when
        bus.dispatch(event);

        // then
        expect(handler).toHaveBeenCalledWith(event);
    });

    it("ignores an event nobody is subscribed to", () => {
        // given
        const handler = vi.fn();
        bus.subscribe("typing", handler);

        // when
        bus.dispatch(liveGamesCountEvent(1));

        // then
        expect(handler).not.toHaveBeenCalled();
        expect(windowErrors).toEqual([]);
    });
});

describe("realtime bus handler isolation", () => {
    it("does not stop the handlers registered after a throwing one", async () => {
        // given
        const failure = new Error("handler is broken");
        const throwing = vi.fn(() => {
            throw failure;
        });
        const second = vi.fn();
        const third = vi.fn();
        bus.subscribe("typing", throwing);
        bus.subscribe("typing", second);
        bus.subscribe("typing", third);
        const event = typingEvent("user-1");

        // when
        bus.dispatch(event);
        await flushMicrotasks();

        // then
        expect(throwing).toHaveBeenCalledTimes(1);
        expect(second).toHaveBeenCalledWith(event);
        expect(third).toHaveBeenCalledWith(event);
        expect(windowErrors).toEqual([failure]);
    });

    it("rethrows a handler failure where a window error listener sees it", async () => {
        // given
        const failure = new Error("handler is broken");
        bus.subscribe("typing", () => {
            throw failure;
        });

        // when
        bus.dispatch(typingEvent("user-1"));

        // then
        expect(windowErrors).toEqual([]);
        await flushMicrotasks();
        expect(windowErrors).toEqual([failure]);
    });

    it("keeps a handler that threw subscribed for the next event", async () => {
        // given
        const failure = new Error("handler is broken");
        const throwing = vi.fn(() => {
            throw failure;
        });
        bus.subscribe("typing", throwing);

        // when
        bus.dispatch(typingEvent("user-1"));
        bus.dispatch(typingEvent("user-2"));
        await flushMicrotasks();

        // then
        expect(throwing).toHaveBeenCalledTimes(2);
        expect(windowErrors).toEqual([failure, failure]);
    });
});

describe("realtime bus subscription lifetime", () => {
    it("stops delivering once the subscriber has unsubscribed", () => {
        // given
        const handler = vi.fn();
        const unsubscribe = bus.subscribe(["typing", "chat_read"], handler);

        // when
        unsubscribe();
        bus.dispatch(typingEvent("user-1"));
        bus.dispatch(chatReadEvent(2));

        // then
        expect(handler).not.toHaveBeenCalled();
    });

    it("ignores a second call to the same unsubscribe", () => {
        // given
        const handler = vi.fn();
        const stale = bus.subscribe("typing", handler);
        stale();
        bus.subscribe("typing", handler);

        // when
        stale();
        bus.dispatch(typingEvent("user-1"));

        // then
        expect(handler).toHaveBeenCalledTimes(1);
    });

    it("does not change who receives the current event when a handler unsubscribes during dispatch", () => {
        // given
        const second = vi.fn();
        const third = vi.fn();
        const unsubscribers: BusModule.RealtimeUnsubscribe[] = [];
        bus.subscribe("typing", () => {
            for (const unsubscribe of unsubscribers) {
                unsubscribe();
            }
        });
        unsubscribers.push(bus.subscribe("typing", second));
        unsubscribers.push(bus.subscribe("typing", third));
        const event = typingEvent("user-1");

        // when
        bus.dispatch(event);

        // then
        expect(second).toHaveBeenCalledWith(event);
        expect(third).toHaveBeenCalledWith(event);

        bus.dispatch(typingEvent("user-2"));
        expect(second).toHaveBeenCalledTimes(1);
        expect(third).toHaveBeenCalledTimes(1);
    });

    it("holds the current event back from a handler that subscribes during dispatch", () => {
        // given
        const latecomer = vi.fn();
        bus.subscribe("typing", () => {
            bus.subscribe("typing", latecomer);
        });

        // when
        bus.dispatch(typingEvent("user-1"));

        // then
        expect(latecomer).not.toHaveBeenCalled();

        bus.dispatch(typingEvent("user-2"));
        expect(latecomer).toHaveBeenCalledTimes(1);
    });
});
