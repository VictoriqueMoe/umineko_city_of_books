import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WatchPartyType } from "../../../types/api";
import { useSessionMedia } from "./useSessionMedia";

const mocks = vi.hoisted(() => {
    type Handler = (...args: unknown[]) => void;

    class FakeRoom {
        handlers: Record<string, Handler[]> = {};
        localParticipant = {
            isMicrophoneEnabled: false,
            setMicrophoneEnabled: vi.fn((_enabled: boolean) => Promise.resolve()),
            setScreenShareEnabled: vi.fn((_on: boolean, _capture?: unknown, _publish?: unknown) => Promise.resolve()),
        };
        connect = vi.fn(() => Promise.resolve());
        disconnect = vi.fn(() => Promise.resolve());

        constructor() {
            rooms.push(this);
        }

        on = (event: string, handler: Handler) => {
            const list = this.handlers[event] ?? [];
            list.push(handler);
            this.handlers[event] = list;
            return this;
        };

        off = (event: string, handler: Handler) => {
            this.handlers[event] = (this.handlers[event] ?? []).filter(h => h !== handler);
            return this;
        };

        emit(event: string, ...args: unknown[]) {
            for (const handler of this.handlers[event] ?? []) {
                handler(...args);
            }
        }
    }

    const rooms: FakeRoom[] = [];

    return { FakeRoom, rooms, getWatchPartyVoiceToken: vi.fn() };
});

vi.mock("livekit-client", () => ({
    Room: mocks.FakeRoom,
    RoomEvent: {
        Connected: "connected",
        Disconnected: "disconnected",
        ParticipantPermissionsChanged: "participantPermissionsChanged",
    },
    AudioPresets: { musicHighQualityStereo: { maxBitrate: 510_000 } },
}));

vi.mock("../../../api/endpoints", () => ({
    getWatchPartyVoiceToken: mocks.getWatchPartyVoiceToken,
}));

interface SetupOptions {
    type?: WatchPartyType;
    isStarter?: boolean;
}

function setup(options: SetupOptions = {}) {
    return renderHook(() =>
        useSessionMedia({
            roomId: "room-1",
            sessionId: "session-1",
            type: options.type ?? "hyperbeam",
            isStarter: options.isStarter ?? false,
        }),
    );
}

function latestRoom() {
    return mocks.rooms[mocks.rooms.length - 1];
}

async function connectLatest() {
    await waitFor(() => {
        expect(mocks.rooms.length).toBeGreaterThan(0);
    });
    const room = latestRoom();
    await act(async () => {
        room.emit("connected");
    });
    return room;
}

beforeEach(() => {
    mocks.rooms.length = 0;
    mocks.getWatchPartyVoiceToken.mockResolvedValue({ token: "lk-token", url: "wss://livekit.test" });
    vi.spyOn(console, "error").mockImplementation(() => {});
});

describe("useSessionMedia connection", () => {
    it("starts out idle with nothing connected", () => {
        // given
        const { result } = setup();

        // when
        const state = result.current;

        // then
        expect(state.status).toBe("idle");
        expect(state.room).toBeNull();
        expect(state.inVoice).toBe(false);
        expect(state.isSharing).toBe(false);
    });

    it("connects a screen share party as soon as it is mounted", async () => {
        // given
        setup({ type: "screenshare" });

        // when
        await waitFor(() => {
            expect(mocks.getWatchPartyVoiceToken).toHaveBeenCalledWith("room-1", "session-1");
        });

        // then
        await waitFor(() => {
            expect(latestRoom().connect).toHaveBeenCalledWith("wss://livekit.test", "lk-token");
        });
    });

    it("leaves a virtual browser party unconnected until somebody asks for voice", () => {
        // given
        setup({ type: "hyperbeam" });

        // when
        const calls = mocks.getWatchPartyVoiceToken.mock.calls.length;

        // then
        expect(calls).toBe(0);
        expect(mocks.rooms).toHaveLength(0);
    });

    it("reports the room as connected once livekit says so", async () => {
        // given
        const { result } = setup({ type: "screenshare" });

        // when
        const room = await connectLatest();

        // then
        expect(result.current.room).toBe(room);
        expect(result.current.status).toBe("connected");
    });

    it("clears every piece of media state when the room disconnects", async () => {
        // given
        const { result } = setup({ type: "screenshare" });
        const room = await connectLatest();

        // when
        await act(async () => {
            room.emit("disconnected");
        });

        // then
        expect(result.current.room).toBeNull();
        expect(result.current.status).toBe("idle");
        expect(result.current.inVoice).toBe(false);
        expect(result.current.isSharing).toBe(false);
    });

    it("disconnects the room when the party is closed", async () => {
        // given
        const { unmount } = setup({ type: "screenshare" });
        const room = await connectLatest();

        // when
        unmount();

        // then
        expect(room.disconnect).toHaveBeenCalledOnce();
    });

    it("only ever builds one room even when voice is requested twice", async () => {
        // given
        const { result } = setup();

        // when
        await act(async () => {
            await result.current.joinVoice();
        });
        await act(async () => {
            await result.current.joinVoice();
        });

        // then
        expect(mocks.rooms).toHaveLength(1);
        expect(mocks.getWatchPartyVoiceToken).toHaveBeenCalledOnce();
    });
});

describe("useSessionMedia voice", () => {
    it("connects and opens the microphone when voice is joined", async () => {
        // given
        const { result } = setup();

        // when
        await act(async () => {
            await result.current.joinVoice();
        });

        // then
        expect(latestRoom().localParticipant.setMicrophoneEnabled).toHaveBeenCalledWith(true);
        expect(result.current.inVoice).toBe(true);
        expect(result.current.status).toBe("connected");
    });

    it("returns to idle when the connection cannot be made", async () => {
        // given
        mocks.getWatchPartyVoiceToken.mockRejectedValue(new Error("no token for you"));
        const { result } = setup();

        // when
        await act(async () => {
            await result.current.joinVoice();
        });

        // then
        expect(result.current.status).toBe("idle");
        expect(result.current.inVoice).toBe(false);
    });

    it("stays out of voice when the microphone is refused", async () => {
        // given
        const { result } = setup({ type: "screenshare" });
        const room = await connectLatest();
        room.localParticipant.setMicrophoneEnabled.mockRejectedValue(new Error("no microphone"));

        // when
        await act(async () => {
            await result.current.joinVoice();
        });

        // then
        expect(result.current.inVoice).toBe(false);
        expect(result.current.status).toBe("connected");
    });

    it("closes the microphone and hangs up when leaving a virtual browser party", async () => {
        // given
        const { result } = setup({ type: "hyperbeam" });
        await act(async () => {
            await result.current.joinVoice();
        });
        const room = latestRoom();

        // when
        await act(async () => {
            await result.current.leaveVoice();
        });

        // then
        expect(room.localParticipant.setMicrophoneEnabled).toHaveBeenLastCalledWith(false);
        expect(room.disconnect).toHaveBeenCalledOnce();
        expect(result.current.inVoice).toBe(false);
    });

    it("keeps the screen share connection alive when only voice is left", async () => {
        // given
        const { result } = setup({ type: "screenshare" });
        const room = await connectLatest();
        await act(async () => {
            await result.current.joinVoice();
        });

        // when
        await act(async () => {
            await result.current.leaveVoice();
        });

        // then
        expect(room.localParticipant.setMicrophoneEnabled).toHaveBeenLastCalledWith(false);
        expect(room.disconnect).not.toHaveBeenCalled();
    });

    it("does nothing when voice is left before anything was connected", async () => {
        // given
        const { result } = setup();

        // when
        await act(async () => {
            await result.current.leaveVoice();
        });

        // then
        expect(mocks.rooms).toHaveLength(0);
    });

    it("reopens the microphone when the server grants publishing rights late", async () => {
        // given
        const { result } = setup();
        await act(async () => {
            await result.current.joinVoice();
        });
        const room = latestRoom();
        room.localParticipant.setMicrophoneEnabled.mockClear();

        // when
        await act(async () => {
            room.emit("participantPermissionsChanged", undefined, room.localParticipant);
        });

        // then
        expect(room.localParticipant.setMicrophoneEnabled).toHaveBeenCalledWith(true);
        expect(result.current.inVoice).toBe(true);
    });

    it("ignores a permissions change for somebody else", async () => {
        // given
        const { result } = setup();
        await act(async () => {
            await result.current.joinVoice();
        });
        const room = latestRoom();
        room.localParticipant.setMicrophoneEnabled.mockClear();

        // when
        await act(async () => {
            room.emit("participantPermissionsChanged", undefined, { identity: "someone-else" });
        });

        // then
        expect(room.localParticipant.setMicrophoneEnabled).not.toHaveBeenCalled();
    });

    it("ignores a permissions change while the microphone is already open", async () => {
        // given
        const { result } = setup();
        await act(async () => {
            await result.current.joinVoice();
        });
        const room = latestRoom();
        room.localParticipant.isMicrophoneEnabled = true;
        room.localParticipant.setMicrophoneEnabled.mockClear();

        // when
        await act(async () => {
            room.emit("participantPermissionsChanged", undefined, room.localParticipant);
        });

        // then
        expect(room.localParticipant.setMicrophoneEnabled).not.toHaveBeenCalled();
    });
});

describe("useSessionMedia screen share", () => {
    it("refuses to share for anybody who did not start the party", async () => {
        // given
        const { result } = setup({ type: "screenshare", isStarter: false });
        await connectLatest();

        // when
        await act(async () => {
            await result.current.shareScreen(true, "gaming");
        });

        // then
        expect(latestRoom().localParticipant.setScreenShareEnabled).not.toHaveBeenCalled();
        expect(result.current.isSharing).toBe(false);
    });

    it("shares with the gaming preset for smooth motion", async () => {
        // given
        const { result } = setup({ type: "screenshare", isStarter: true });
        const room = await connectLatest();

        // when
        await act(async () => {
            await result.current.shareScreen(true, "gaming");
        });

        // then
        const [on, capture, publish] = room.localParticipant.setScreenShareEnabled.mock.calls[0];
        expect(on).toBe(true);
        expect(capture).toMatchObject({
            contentHint: "motion",
            resolution: { width: 1920, height: 1080, frameRate: 60 },
        });
        expect(publish).toMatchObject({
            videoCodec: "vp9",
            degradationPreference: "maintain-framerate",
            forceStereo: true,
            dtx: false,
            red: false,
            screenShareEncoding: { maxBitrate: 6_000_000, maxFramerate: 60 },
        });
        expect(result.current.isSharing).toBe(true);
    });

    it("shares with the screenshare preset for legible text", async () => {
        // given
        const { result } = setup({ type: "screenshare", isStarter: true });
        const room = await connectLatest();

        // when
        await act(async () => {
            await result.current.shareScreen(true, "screenshare");
        });

        // then
        const [, capture, publish] = room.localParticipant.setScreenShareEnabled.mock.calls[0];
        expect(capture).toMatchObject({
            contentHint: "detail",
            resolution: { width: 1920, height: 1080, frameRate: 15 },
        });
        expect(publish).toMatchObject({
            degradationPreference: "maintain-resolution",
            screenShareEncoding: { maxBitrate: 2_500_000, maxFramerate: 15 },
        });
    });

    it("turns the browser audio processing off so the shared sound is untouched", async () => {
        // given
        const { result } = setup({ type: "screenshare", isStarter: true });
        const room = await connectLatest();

        // when
        await act(async () => {
            await result.current.shareScreen(true, "gaming");
        });

        // then
        const [, capture] = room.localParticipant.setScreenShareEnabled.mock.calls[0];
        expect(capture).toMatchObject({
            audio: { echoCancellation: false, noiseSuppression: false, autoGainControl: false },
        });
    });

    it("records that sharing has stopped when it is switched off", async () => {
        // given
        const { result } = setup({ type: "screenshare", isStarter: true });
        await connectLatest();
        await act(async () => {
            await result.current.shareScreen(true, "gaming");
        });

        // when
        await act(async () => {
            await result.current.shareScreen(false, "gaming");
        });

        // then
        expect(result.current.isSharing).toBe(false);
    });
});

describe("useSessionMedia reload", () => {
    it("drops the old connection and builds a fresh one", async () => {
        // given
        const { result } = setup({ type: "screenshare" });
        const first = await connectLatest();

        // when
        await act(async () => {
            await result.current.reload();
        });

        // then
        expect(first.disconnect).toHaveBeenCalledOnce();
        expect(mocks.rooms).toHaveLength(2);
    });

    it("reopens the microphone on the new connection when voice was in use", async () => {
        // given
        const { result } = setup({ type: "screenshare" });
        await connectLatest();
        await act(async () => {
            await result.current.joinVoice();
        });

        // when
        await act(async () => {
            await result.current.reload();
        });

        // then
        expect(latestRoom().localParticipant.setMicrophoneEnabled).toHaveBeenCalledWith(true);
    });

    it("leaves the microphone shut when voice was never joined", async () => {
        // given
        const { result } = setup({ type: "screenshare" });
        await connectLatest();

        // when
        await act(async () => {
            await result.current.reload();
        });

        // then
        expect(latestRoom().localParticipant.setMicrophoneEnabled).not.toHaveBeenCalled();
    });
});
