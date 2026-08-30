import { beforeEach, describe, expect, it, type Mock, vi } from "vitest";
import type { Room } from "livekit-client";
import { connectRoom, disconnectRoom } from "./connect";

interface FakeRoom {
    handlers: Record<string, (...args: unknown[]) => void>;
    on: Mock;
    connect: Mock;
    disconnect: Mock;
    localParticipant: { identity: string };
}

const mocks = vi.hoisted(() => ({
    rooms: [] as FakeRoom[],
    connectImpl: (): Promise<void> => Promise.resolve(),
    disconnectImpl: (): Promise<void> => Promise.resolve(),
}));

vi.mock("livekit-client", () => {
    class Room {
        handlers: Record<string, (...args: unknown[]) => void> = {};
        connect = vi.fn(() => mocks.connectImpl());
        disconnect = vi.fn(() => mocks.disconnectImpl());
        localParticipant = { identity: "me" };
        on = vi.fn((event: string, handler: (...args: unknown[]) => void) => {
            this.handlers[event] = handler;
            return this;
        });

        constructor() {
            mocks.rooms.push(this as unknown as FakeRoom);
        }
    }

    return {
        Room,
        RoomEvent: {
            Connected: "connected",
            Disconnected: "disconnected",
            ParticipantConnected: "participantConnected",
            ParticipantDisconnected: "participantDisconnected",
            ParticipantPermissionsChanged: "participantPermissionsChanged",
        },
    };
});

function lastRoom(): FakeRoom {
    return mocks.rooms[mocks.rooms.length - 1];
}

beforeEach(() => {
    mocks.rooms.length = 0;
    mocks.connectImpl = () => Promise.resolve();
    mocks.disconnectImpl = () => Promise.resolve();
});

describe("connectRoom", () => {
    it("connects to the given url with the given token and answers with the room", async () => {
        // when
        const room = await connectRoom({ url: "wss://livekit.test", token: "token-123" });

        // then
        expect(mocks.rooms).toHaveLength(1);
        expect(room).toBe(lastRoom() as unknown as Room);
        expect(lastRoom().connect).toHaveBeenCalledWith("wss://livekit.test", "token-123", undefined);
    });

    it("passes autoSubscribe through, so a streamer can hold a slot without downloading their own video", async () => {
        // when
        await connectRoom({ url: "wss://livekit.test", token: "token-123", autoSubscribe: false });

        // then
        expect(lastRoom().connect).toHaveBeenCalledWith("wss://livekit.test", "token-123", { autoSubscribe: false });
    });

    it("leaves the connect options unset when the caller says nothing about subscribing", async () => {
        // when
        await connectRoom({ url: "wss://livekit.test", token: "token-123" });

        // then
        expect(lastRoom().connect.mock.calls[0][2]).toBeUndefined();
    });

    it("binds only the handlers the caller asked for", async () => {
        // when
        await connectRoom({ url: "wss://livekit.test", token: "t", on: { onConnected: vi.fn() } });

        // then
        expect(Object.keys(lastRoom().handlers)).toEqual(["connected"]);
    });

    it("binds the handlers before connecting, so a Connected raised during connect is not missed", async () => {
        // given
        const onConnected = vi.fn();
        mocks.connectImpl = () => {
            lastRoom().handlers.connected();
            return Promise.resolve();
        };

        // when
        const room = await connectRoom({ url: "wss://livekit.test", token: "t", on: { onConnected } });

        // then
        expect(onConnected).toHaveBeenCalledExactlyOnceWith(room);
    });

    it("reports Connected, Disconnected and the two participant events with the room", async () => {
        // given
        const on = {
            onConnected: vi.fn(),
            onDisconnected: vi.fn(),
            onParticipantConnected: vi.fn(),
            onParticipantDisconnected: vi.fn(),
        };
        const room = await connectRoom({ url: "wss://livekit.test", token: "t", on });

        // when
        lastRoom().handlers.connected();
        lastRoom().handlers.disconnected();
        lastRoom().handlers.participantConnected();
        lastRoom().handlers.participantDisconnected();

        // then
        expect(on.onConnected).toHaveBeenCalledExactlyOnceWith(room);
        expect(on.onDisconnected).toHaveBeenCalledExactlyOnceWith(room);
        expect(on.onParticipantConnected).toHaveBeenCalledExactlyOnceWith(room);
        expect(on.onParticipantDisconnected).toHaveBeenCalledExactlyOnceWith(room);
    });

    it("reports a permissions change only when it is about the local participant", async () => {
        // given
        const onLocalPermissionsChanged = vi.fn();
        const room = await connectRoom({ url: "wss://livekit.test", token: "t", on: { onLocalPermissionsChanged } });

        // when
        lastRoom().handlers.participantPermissionsChanged(undefined, lastRoom().localParticipant);

        // then
        expect(onLocalPermissionsChanged).toHaveBeenCalledExactlyOnceWith(room);
    });

    it("ignores a permissions change about somebody else", async () => {
        // given
        const onLocalPermissionsChanged = vi.fn();
        await connectRoom({ url: "wss://livekit.test", token: "t", on: { onLocalPermissionsChanged } });

        // when
        lastRoom().handlers.participantPermissionsChanged(undefined, { identity: "someone-else" });

        // then
        expect(onLocalPermissionsChanged).not.toHaveBeenCalled();
    });

    it("never builds a room when the caller aborted before the transport finished loading", async () => {
        // given
        const controller = new AbortController();
        controller.abort();

        // when
        const room = await connectRoom({ url: "wss://livekit.test", token: "t", signal: controller.signal });

        // then
        expect(room).toBeNull();
        expect(mocks.rooms).toHaveLength(0);
    });

    it("hangs up and answers null when the caller aborted while the connect was in flight", async () => {
        // given
        const controller = new AbortController();
        mocks.connectImpl = () => {
            controller.abort();
            return Promise.resolve();
        };

        // when
        const room = await connectRoom({ url: "wss://livekit.test", token: "t", signal: controller.signal });

        // then
        expect(room).toBeNull();
        expect(lastRoom().disconnect).toHaveBeenCalledOnce();
    });

    it("lets a connect failure reach the caller instead of answering null", async () => {
        // given
        mocks.connectImpl = () => Promise.reject(new Error("livekit refused the token"));

        // when
        const attempt = connectRoom({ url: "wss://livekit.test", token: "bad" });

        // then
        await expect(attempt).rejects.toThrow("livekit refused the token");
    });
});

describe("disconnectRoom", () => {
    it("hangs up the room it is given", async () => {
        // given
        const room = await connectRoom({ url: "wss://livekit.test", token: "t" });

        // when
        disconnectRoom(room);

        // then
        expect(lastRoom().disconnect).toHaveBeenCalledOnce();
    });

    it("does nothing when there is no room, so teardown never has to check first", () => {
        // then
        expect(() => disconnectRoom(null)).not.toThrow();
        expect(() => disconnectRoom(undefined)).not.toThrow();
    });

    it("catches a hang-up that fails, because a teardown has nowhere to report it", async () => {
        // given
        const room = await connectRoom({ url: "wss://livekit.test", token: "t" });
        mocks.disconnectImpl = () => Promise.reject(new Error("socket already gone"));

        // when
        disconnectRoom(room);
        await Promise.resolve();

        // then
        expect(lastRoom().disconnect).toHaveBeenCalledOnce();
    });
});
