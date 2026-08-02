import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Room } from "livekit-client";
import { useAudioPlaybackGuard } from "./useAudioPlaybackGuard";

vi.mock("livekit-client", () => ({
    Room: class {},
    RoomEvent: { AudioPlaybackStatusChanged: "audioPlaybackChanged" },
}));

interface FakeRoom {
    canPlaybackAudio: boolean;
    startAudio: ReturnType<typeof vi.fn>;
    on: ReturnType<typeof vi.fn>;
    off: ReturnType<typeof vi.fn>;
    fire: () => void;
}

function makeRoom(canPlaybackAudio: boolean): FakeRoom {
    const handlers: (() => void)[] = [];

    return {
        canPlaybackAudio,
        startAudio: vi.fn(() => Promise.resolve()),
        on: vi.fn((_event: string, handler: () => void) => {
            handlers.push(handler);
        }),
        off: vi.fn((_event: string, handler: () => void) => {
            const idx = handlers.indexOf(handler);
            if (idx !== -1) {
                handlers.splice(idx, 1);
            }
        }),
        fire: () => {
            for (const handler of [...handlers]) {
                handler();
            }
        },
    };
}

function asRoom(room: FakeRoom): Room {
    return room as unknown as Room;
}

class FakeMediaStream {
    private readonly tracks: { readyState: string }[];

    constructor(tracks: { readyState: string }[]) {
        this.tracks = tracks;
    }

    getAudioTracks(): { readyState: string }[] {
        return this.tracks;
    }
}

function mountAudio(srcObject: unknown): HTMLAudioElement {
    const el = document.createElement("audio");
    Object.defineProperty(el, "srcObject", { configurable: true, writable: true, value: srcObject });
    document.body.appendChild(el);
    return el;
}

beforeEach(() => {
    vi.stubGlobal("MediaStream", FakeMediaStream);
});

describe("useAudioPlaybackGuard", () => {
    it("registers nothing at all while there is no room", () => {
        // given
        const addSpy = vi.spyOn(document, "addEventListener");

        // when
        renderHook(() => useAudioPlaybackGuard(null));

        // then
        expect(addSpy).not.toHaveBeenCalledWith("pause", expect.anything(), true);
    });

    it("listens for blocked playback on the room it is given", () => {
        // given
        const room = makeRoom(true);

        // when
        renderHook(() => useAudioPlaybackGuard(asRoom(room)));

        // then
        expect(room.on).toHaveBeenCalledWith("audioPlaybackChanged", expect.any(Function));
    });

    it("starts audio again once the browser reports playback is blocked", () => {
        // given
        const room = makeRoom(false);
        renderHook(() => useAudioPlaybackGuard(asRoom(room)));

        // when
        room.fire();

        // then
        expect(room.startAudio).toHaveBeenCalledOnce();
    });

    it("leaves audio alone while the browser is happy to play it", () => {
        // given
        const room = makeRoom(true);
        renderHook(() => useAudioPlaybackGuard(asRoom(room)));

        // when
        room.fire();

        // then
        expect(room.startAudio).not.toHaveBeenCalled();
    });

    it("swallows a refusal to start audio rather than raising it", () => {
        // given
        const room = makeRoom(false);
        room.startAudio.mockRejectedValue(new Error("gesture required"));
        renderHook(() => useAudioPlaybackGuard(asRoom(room)));

        // when
        const fire = () => room.fire();

        // then
        expect(fire).not.toThrow();
    });

    it("resumes an audio element that paused while its track was still live", () => {
        // given
        const room = makeRoom(true);
        renderHook(() => useAudioPlaybackGuard(asRoom(room)));
        const el = mountAudio(new FakeMediaStream([{ readyState: "live" }]));
        const play = vi.spyOn(el, "play").mockResolvedValue(undefined);

        // when
        el.dispatchEvent(new Event("pause"));

        // then
        expect(play).toHaveBeenCalledOnce();
    });

    it("leaves an audio element alone when every track of its stream has ended", () => {
        // given
        const room = makeRoom(true);
        renderHook(() => useAudioPlaybackGuard(asRoom(room)));
        const el = mountAudio(new FakeMediaStream([{ readyState: "ended" }]));
        const play = vi.spyOn(el, "play").mockResolvedValue(undefined);

        // when
        el.dispatchEvent(new Event("pause"));

        // then
        expect(play).not.toHaveBeenCalled();
    });

    it("leaves an audio element alone when it is not backed by a media stream", () => {
        // given
        const room = makeRoom(true);
        renderHook(() => useAudioPlaybackGuard(asRoom(room)));
        const el = mountAudio(null);
        const play = vi.spyOn(el, "play").mockResolvedValue(undefined);

        // when
        el.dispatchEvent(new Event("pause"));

        // then
        expect(play).not.toHaveBeenCalled();
    });

    it("ignores a pause raised by anything that is not an audio element", () => {
        // given
        const room = makeRoom(true);
        renderHook(() => useAudioPlaybackGuard(asRoom(room)));
        const video = document.createElement("video");
        document.body.appendChild(video);
        const play = vi.spyOn(video, "play").mockResolvedValue(undefined);

        // when
        video.dispatchEvent(new Event("pause"));

        // then
        expect(play).not.toHaveBeenCalled();
    });

    it("swallows a refusal to resume the paused element", () => {
        // given
        const room = makeRoom(true);
        renderHook(() => useAudioPlaybackGuard(asRoom(room)));
        const el = mountAudio(new FakeMediaStream([{ readyState: "live" }]));
        vi.spyOn(el, "play").mockRejectedValue(new Error("gesture required"));

        // when
        const dispatch = () => el.dispatchEvent(new Event("pause"));

        // then
        expect(dispatch).not.toThrow();
    });

    it("stops listening to the room and the document once it is torn down", () => {
        // given
        const room = makeRoom(true);
        const { unmount } = renderHook(() => useAudioPlaybackGuard(asRoom(room)));
        const el = mountAudio(new FakeMediaStream([{ readyState: "live" }]));
        const play = vi.spyOn(el, "play").mockResolvedValue(undefined);

        // when
        unmount();
        el.dispatchEvent(new Event("pause"));

        // then
        expect(room.off).toHaveBeenCalledWith("audioPlaybackChanged", expect.any(Function));
        expect(play).not.toHaveBeenCalled();
    });

    it("moves its listeners over when the room is replaced", () => {
        // given
        const first = makeRoom(false);
        const second = makeRoom(false);
        const { rerender } = renderHook(({ room }: { room: Room | null }) => useAudioPlaybackGuard(room), {
            initialProps: { room: asRoom(first) },
        });

        // when
        rerender({ room: asRoom(second) });
        first.fire();
        second.fire();

        // then
        expect(first.off).toHaveBeenCalledOnce();
        expect(first.startAudio).not.toHaveBeenCalled();
        expect(second.startAudio).toHaveBeenCalledOnce();
    });
});
