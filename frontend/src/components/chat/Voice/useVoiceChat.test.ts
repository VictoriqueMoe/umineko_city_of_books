import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock, vi } from "vitest";
import type { WSMessageHandler } from "../../../context/notificationContextValue";
import { providerWrapper } from "../../../test-utils/render";
import type { WSMessage } from "../../../types/api";
import { useVoiceChat } from "./useVoiceChat";

interface FakeRoom {
    handlers: Record<string, () => void>;
    on: Mock;
    connect: Mock;
    disconnect: Mock;
    localParticipant: { setMicrophoneEnabled: Mock };
}

const mocks = vi.hoisted(() => ({
    getVoiceToken: vi.fn(),
    playVoiceJoinSound: vi.fn(),
    playVoiceLeaveSound: vi.fn(),
    rooms: [] as FakeRoom[],
}));

vi.mock("../../../api/endpoints", () => ({ getVoiceToken: mocks.getVoiceToken }));

vi.mock("../../../utils/sound", () => ({
    playVoiceJoinSound: mocks.playVoiceJoinSound,
    playVoiceLeaveSound: mocks.playVoiceLeaveSound,
}));

vi.mock("livekit-client", () => {
    class Room {
        handlers: Record<string, () => void> = {};
        connect = vi.fn(() => Promise.resolve());
        disconnect = vi.fn(() => Promise.resolve());
        localParticipant = { setMicrophoneEnabled: vi.fn(() => Promise.resolve()) };
        on = vi.fn((event: string, handler: () => void) => {
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
            Disconnected: "disconnected",
            ParticipantConnected: "participantConnected",
            ParticipantDisconnected: "participantDisconnected",
        },
    };
});

let wsListeners: WSMessageHandler[] = [];

function addWSListener(handler: WSMessageHandler): () => void {
    wsListeners.push(handler);

    return () => {
        wsListeners = wsListeners.filter(l => l !== handler);
    };
}

function emit(msg: WSMessage): void {
    act(() => {
        for (const listener of wsListeners.slice()) {
            listener(msg);
        }
    });
}

function renderVoice(roomId = "room-1", initialParticipants: string[] = []) {
    return renderHook(() => useVoiceChat(roomId, initialParticipants), {
        wrapper: providerWrapper({ notification: { addWSListener } }),
    });
}

async function joinCall(result: { current: ReturnType<typeof useVoiceChat> }) {
    await act(async () => {
        result.current.join();
    });
    await waitFor(() => expect(result.current.status).toBe("connected"));

    return mocks.rooms[mocks.rooms.length - 1];
}

beforeEach(() => {
    wsListeners = [];
    mocks.rooms.length = 0;
    mocks.getVoiceToken.mockResolvedValue({ token: "golden-token", url: "wss://voice.example" });
});

describe("useVoiceChat", () => {
    it("starts idle with nobody connected and the seeded participants showing", () => {
        // given
        const initialParticipants = ["battler", "beatrice"];

        // when
        const { result } = renderVoice("room-1", initialParticipants);

        // then
        expect(result.current.status).toBe("idle");
        expect(result.current.room).toBeNull();
        expect(result.current.participantIds).toEqual(["battler", "beatrice"]);
        expect(result.current.presenceCount).toBe(2);
    });

    it("replaces the seeded participants once a presence update arrives for this room", () => {
        // given
        const { result } = renderVoice("room-1", ["battler"]);

        // when
        emit({ type: "voice_presence", data: { room_id: "room-1", participants: ["beatrice", "ronove"], count: 2 } });

        // then
        expect(result.current.participantIds).toEqual(["beatrice", "ronove"]);
        expect(result.current.presenceCount).toBe(2);
    });

    it("ignores a presence update aimed at a different room", () => {
        // given
        const { result } = renderVoice("room-1", ["battler"]);

        // when
        emit({ type: "voice_presence", data: { room_id: "room-2", participants: ["beatrice"], count: 1 } });

        // then
        expect(result.current.participantIds).toEqual(["battler"]);
    });

    it("ignores websocket traffic that is not a voice presence update", () => {
        // given
        const { result } = renderVoice("room-1", ["battler"]);

        // when
        emit({ type: "chat_message", data: { room_id: "room-1", participants: [], count: 0 } });

        // then
        expect(result.current.participantIds).toEqual(["battler"]);
    });

    it("treats a presence update with no participant list as an empty call", () => {
        // given
        const { result } = renderVoice("room-1", ["battler"]);

        // when
        emit({ type: "voice_presence", data: { room_id: "room-1", count: 0 } });

        // then
        expect(result.current.participantIds).toEqual([]);
        expect(result.current.presenceCount).toBe(0);
    });

    it("fetches a token for the room and connects to the url it came back with", async () => {
        // given
        const { result } = renderVoice("room-1");

        // when
        const room = await joinCall(result);

        // then
        expect(mocks.getVoiceToken).toHaveBeenCalledWith("room-1");
        expect(room.connect).toHaveBeenCalledWith("wss://voice.example", "golden-token");
        expect(result.current.room).not.toBeNull();
    });

    it("opens the microphone and plays the join sound once connected", async () => {
        // given
        const { result } = renderVoice("room-1");

        // when
        const room = await joinCall(result);

        // then
        expect(room.localParticipant.setMicrophoneEnabled).toHaveBeenCalledWith(true);
        expect(mocks.playVoiceJoinSound).toHaveBeenCalledTimes(1);
    });

    it("reports connecting while the token request is still in flight", async () => {
        // given
        let release: (value: { token: string; url: string }) => void = () => {};
        mocks.getVoiceToken.mockReturnValue(
            new Promise(resolve => {
                release = resolve;
            }),
        );
        const { result } = renderVoice("room-1");

        // when
        act(() => {
            result.current.join();
        });

        // then
        expect(result.current.status).toBe("connecting");
        await act(async () => {
            release({ token: "golden-token", url: "wss://voice.example" });
        });
        await waitFor(() => expect(result.current.status).toBe("connected"));
    });

    it("ignores a second join while a call is already up", async () => {
        // given
        const { result } = renderVoice("room-1");
        await joinCall(result);

        // when
        await act(async () => {
            result.current.join();
        });

        // then
        expect(mocks.getVoiceToken).toHaveBeenCalledTimes(1);
        expect(mocks.rooms).toHaveLength(1);
    });

    it("ignores a second join while the first one is still connecting", async () => {
        // given
        let release: (value: { token: string; url: string }) => void = () => {};
        mocks.getVoiceToken.mockReturnValue(
            new Promise(resolve => {
                release = resolve;
            }),
        );
        const { result } = renderVoice("room-1");

        // when
        act(() => {
            result.current.join();
            result.current.join();
        });

        // then
        expect(mocks.getVoiceToken).toHaveBeenCalledTimes(1);
        await act(async () => {
            release({ token: "golden-token", url: "wss://voice.example" });
        });
        await waitFor(() => expect(result.current.status).toBe("connected"));
        expect(mocks.rooms).toHaveLength(1);
    });

    it("falls back to idle when the token request fails", async () => {
        // given
        mocks.getVoiceToken.mockRejectedValue(new Error("no voice for you"));
        const { result } = renderVoice("room-1");

        // when
        await act(async () => {
            result.current.join();
        });

        // then
        await waitFor(() => expect(result.current.status).toBe("idle"));
        expect(result.current.room).toBeNull();
        expect(mocks.playVoiceJoinSound).not.toHaveBeenCalled();
    });

    it("lets the viewer join again after a failed attempt", async () => {
        // given
        mocks.getVoiceToken.mockRejectedValueOnce(new Error("no voice for you"));
        const { result } = renderVoice("room-1");
        await act(async () => {
            result.current.join();
        });

        // when
        await joinCall(result);

        // then
        expect(mocks.getVoiceToken).toHaveBeenCalledTimes(2);
    });

    it("disconnects and plays the leave sound when the viewer leaves", async () => {
        // given
        const { result } = renderVoice("room-1");
        const room = await joinCall(result);

        // when
        act(() => {
            result.current.leave();
        });

        // then
        expect(room.disconnect).toHaveBeenCalledTimes(1);
        expect(mocks.playVoiceLeaveSound).toHaveBeenCalledTimes(1);
        expect(result.current.status).toBe("idle");
        expect(result.current.room).toBeNull();
    });

    it("stays quiet when leaving a call that was never joined", () => {
        // given
        const { result } = renderVoice("room-1");

        // when
        act(() => {
            result.current.leave();
        });

        // then
        expect(mocks.playVoiceLeaveSound).not.toHaveBeenCalled();
        expect(result.current.status).toBe("idle");
    });

    it("returns to idle when livekit reports the room disconnected", async () => {
        // given
        const { result } = renderVoice("room-1");
        const room = await joinCall(result);

        // when
        act(() => {
            room.handlers.disconnected();
        });

        // then
        expect(result.current.status).toBe("idle");
        expect(result.current.room).toBeNull();
    });

    it("plays a sound as other people arrive in and drop out of the call", async () => {
        // given
        const { result } = renderVoice("room-1");
        const room = await joinCall(result);

        // when
        act(() => {
            room.handlers.participantConnected();
            room.handlers.participantDisconnected();
        });

        // then
        expect(mocks.playVoiceJoinSound).toHaveBeenCalledTimes(2);
        expect(mocks.playVoiceLeaveSound).toHaveBeenCalledTimes(1);
    });

    it("hangs up the call when the component using it goes away", async () => {
        // given
        const { result, unmount } = renderVoice("room-1");
        const room = await joinCall(result);

        // when
        unmount();

        // then
        expect(room.disconnect).toHaveBeenCalledTimes(1);
    });
});
