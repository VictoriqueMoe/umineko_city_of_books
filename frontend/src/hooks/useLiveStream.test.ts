import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import type { Room } from "livekit-client";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as livekit from "../api/livekit/connect";
import type { ConnectRoomOptions } from "../api/livekit/connect";
import { queryKeys } from "../api/queryKeys";
import { expectInvalidated } from "../test-utils/query";
import { makeStream as makeLiveStream, makeUser } from "../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../test-utils/render";
import { emitRealtimeEvent } from "../test-utils/ws";
import type { LiveStream, UserProfile } from "../types/api";
import { useLiveStream } from "./useLiveStream";

const mocks = vi.hoisted(() => ({
    getStream: vi.fn(),
    getStreamViewerToken: vi.fn(),
}));

vi.mock("../api/endpoints/stream", () => ({
    listLiveStreams: vi.fn(),
    getStream: mocks.getStream,
    getMyStream: vi.fn(),
    getStreamCredentials: vi.fn(),
    getStreamViewerToken: mocks.getStreamViewerToken,
    joinStreamChat: vi.fn(),
    resetStreamCredentials: vi.fn(),
    startStream: vi.fn(),
    stopStream: vi.fn(),
    updateStreamTitle: vi.fn(),
    uploadStreamThumbnail: vi.fn(),
}));

vi.mock("../api/livekit/connect", () => ({
    connectRoom: vi.fn(),
    disconnectRoom: vi.fn(),
}));

const { getStream, getStreamViewerToken } = mocks;
const connectRoom = vi.mocked(livekit.connectRoom);
const disconnectRoom = vi.mocked(livekit.disconnectRoom);

const STREAMER_ID = "streamer-1";

function makeStream(overrides: Partial<LiveStream> = {}): LiveStream {
    return makeLiveStream({
        userId: STREAMER_ID,
        title: "Reading Episode 4",
        startedAt: "2026-02-01T12:00:00Z",
        streamerUsername: "beatrice",
        ...overrides,
    });
}

function makeRoom(name: string): Room {
    return { name } as unknown as Room;
}

function lastOptions(): ConnectRoomOptions {
    const { calls } = connectRoom.mock;
    if (calls.length === 0) {
        throw new Error("connectRoom was never called");
    }

    return calls[calls.length - 1][0];
}

let queryClient: QueryClient;

function mount(options: { user?: UserProfile | null; streamId?: string } = {}) {
    return renderHook(() => useLiveStream(options.streamId ?? "stream-1"), {
        wrapper: providerWrapper({ queryClient, user: options.user ?? null }),
    });
}

beforeEach(() => {
    queryClient = createTestQueryClient();
    connectRoom.mockReset();
    disconnectRoom.mockReset();
    connectRoom.mockResolvedValue(makeRoom("room-1"));
    getStream.mockResolvedValue(makeStream());
    getStreamViewerToken.mockResolvedValue({ token: "tok", url: "wss://livekit.test" });
});

describe("useLiveStream lookup", () => {
    it("asks the server for the stream named in the address", async () => {
        // given
        mount({ streamId: "stream-77" });

        // when
        await waitFor(() => {
            expect(getStream).toHaveBeenCalledWith("stream-77");
        });

        // then
        expect(getStream).toHaveBeenCalledTimes(1);
    });

    it("waits while the stream is being looked up", () => {
        // given
        getStream.mockReturnValue(new Promise<LiveStream>(() => {}));

        // when
        const view = mount();

        // then
        expect(view.result.current.loading).toBe(true);
        expect(view.result.current.stream).toBeNull();
    });

    it("hands the stream over once it arrives", async () => {
        // given
        const view = mount();

        // when
        await waitFor(() => {
            expect(view.result.current.stream).not.toBeNull();
        });

        // then
        expect(view.result.current.stream?.title).toBe("Reading Episode 4");
        expect(view.result.current.plan.isLive).toBe(true);
    });
});

describe("useLiveStream playback plan", () => {
    it("subscribes to the media of somebody else's stream", async () => {
        // given
        mount({ user: makeUser({ id: "viewer-1" }) });

        // when
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalled();
        });

        // then
        expect(lastOptions().autoSubscribe).toBe(true);
    });

    it("joins the room without media while a streamer's own preview is hidden", async () => {
        // given
        mount({ user: makeUser({ id: STREAMER_ID }) });

        // when
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalled();
        });

        // then
        expect(lastOptions().autoSubscribe).toBe(false);
    });

    it("opens no livekit room while the smooth feed is playing", async () => {
        // given
        getStream.mockResolvedValue(makeStream({ defaultMode: "hls", hlsUrl: "https://edge/s.m3u8" }));
        const view = mount({ user: makeUser({ id: "viewer-1" }) });

        // when
        await waitFor(() => {
            expect(view.result.current.plan.mode).toBe("hls");
        });

        // then
        expect(connectRoom).not.toHaveBeenCalled();
        expect(getStreamViewerToken).not.toHaveBeenCalled();
    });

    it("falls back to the low latency room when the stream prefers hls but has no url", async () => {
        // given
        getStream.mockResolvedValue(makeStream({ defaultMode: "hls", hlsUrl: undefined }));
        const view = mount({ user: makeUser({ id: "viewer-1" }) });

        // when
        await waitFor(() => {
            expect(view.result.current.stream).not.toBeNull();
        });

        // then
        expect(view.result.current.plan.mode).toBe("webrtc");
    });

    it("opens no room while the stream is offline", async () => {
        // given
        getStream.mockResolvedValue(makeStream({ status: "offline" }));
        const view = mount({ user: makeUser({ id: "viewer-1" }) });

        // when
        await waitFor(() => {
            expect(view.result.current.stream).not.toBeNull();
        });

        // then
        expect(view.result.current.plan.isLive).toBe(false);
        expect(connectRoom).not.toHaveBeenCalled();
    });

    it("tears the room down and builds a new one when the streamer toggles their own preview", async () => {
        // given
        const room = makeRoom("room-1");
        const view = mount({ user: makeUser({ id: STREAMER_ID }) });
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalledTimes(1);
        });
        expect(lastOptions().autoSubscribe).toBe(false);
        act(() => {
            lastOptions().on?.onConnected?.(room);
        });

        // when
        act(() => {
            view.result.current.setShowOwnPreview(true);
        });

        // then
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalledTimes(2);
        });
        expect(disconnectRoom).toHaveBeenCalledWith(room);
        expect(lastOptions().autoSubscribe).toBe(true);
    });

    it("tears the room down when the viewer switches to the smooth feed", async () => {
        // given
        getStream.mockResolvedValue(makeStream({ hlsUrl: "https://edge/s.m3u8" }));
        const room = makeRoom("room-1");
        const view = mount({ user: makeUser({ id: "viewer-1" }) });
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalledTimes(1);
        });
        act(() => {
            lastOptions().on?.onConnected?.(room);
        });

        // when
        act(() => {
            view.result.current.setMode("hls");
        });

        // then
        expect(disconnectRoom).toHaveBeenCalledWith(room);
        expect(connectRoom).toHaveBeenCalledTimes(1);
    });
});

describe("useLiveStream live updates", () => {
    it("refetches the stream when it goes offline", async () => {
        // given
        mount({ user: makeUser({ id: "viewer-1" }) });
        await waitFor(() => {
            expect(getStream).toHaveBeenCalled();
        });
        vi.spyOn(queryClient, "invalidateQueries");

        // when
        emitRealtimeEvent({ type: "stream_offline", data: { streamId: "stream-1" } });

        // then
        expectInvalidated(queryClient, queryKeys.streams.detail("stream-1"));
    });

    it("refetches the stream when it comes back live", async () => {
        // given
        mount({ user: makeUser({ id: "viewer-1" }) });
        await waitFor(() => {
            expect(getStream).toHaveBeenCalled();
        });
        vi.spyOn(queryClient, "invalidateQueries");

        // when
        emitRealtimeEvent({ type: "stream_live", data: makeStream({ id: "stream-1" }) });

        // then
        expectInvalidated(queryClient, queryKeys.streams.detail("stream-1"));
    });

    it("renames the stream in place when the streamer changes the title", async () => {
        // given
        const view = mount({ user: makeUser({ id: "viewer-1" }) });
        await waitFor(() => {
            expect(view.result.current.stream).not.toBeNull();
        });

        // when
        emitRealtimeEvent({ type: "stream_title", data: { streamId: "stream-1", title: "Now solving the epitaph" } });

        // then
        await waitFor(() => {
            expect(view.result.current.stream?.title).toBe("Now solving the epitaph");
        });
    });

    it("ignores another stream going offline", async () => {
        // given
        mount({ user: makeUser({ id: "viewer-1" }) });
        await waitFor(() => {
            expect(getStream).toHaveBeenCalled();
        });
        vi.spyOn(queryClient, "invalidateQueries");

        // when
        emitRealtimeEvent({ type: "stream_offline", data: { streamId: "another-stream" } });

        // then
        expect(queryClient.invalidateQueries).not.toHaveBeenCalled();
    });

    it("leaves the title alone when another stream is renamed", async () => {
        // given
        const view = mount({ user: makeUser({ id: "viewer-1" }) });
        await waitFor(() => {
            expect(view.result.current.stream).not.toBeNull();
        });

        // when
        emitRealtimeEvent({ type: "stream_title", data: { streamId: "another-stream", title: "Something else" } });

        // then
        expect(view.result.current.stream?.title).toBe("Reading Episode 4");
    });

    it("stops listening once the viewer leaves the page", async () => {
        // given
        const view = mount({ user: makeUser({ id: "viewer-1" }) });
        await waitFor(() => {
            expect(getStream).toHaveBeenCalled();
        });
        vi.spyOn(queryClient, "invalidateQueries");

        // when
        view.unmount();
        emitRealtimeEvent({ type: "stream_offline", data: { streamId: "stream-1" } });

        // then
        expect(queryClient.invalidateQueries).not.toHaveBeenCalled();
    });
});
