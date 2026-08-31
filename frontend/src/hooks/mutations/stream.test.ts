import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../../api/queryKeys";
import { makeStream as makeLiveStream } from "../../test-utils/fixtures";
import { providerWrapper } from "../../test-utils/render";
import type { LiveStream, StreamOwner } from "../../types/api";
import {
    useJoinStreamChat,
    useResetStreamCredentials,
    useStartStream,
    useStopStream,
    useStreamViewerToken,
    useUpdateStreamTitle,
    useUploadStreamThumbnail,
} from "./stream";

const mocks = vi.hoisted(() => ({
    getStreamViewerToken: vi.fn(),
    joinStreamChat: vi.fn(),
    resetStreamCredentials: vi.fn(),
    startStream: vi.fn(),
    stopStream: vi.fn(),
    updateStreamTitle: vi.fn(),
    uploadStreamThumbnail: vi.fn(),
}));

vi.mock("../../api/endpoints/stream", () => mocks);

const streamId = "11111111-1111-1111-1111-111111111111";

function makeStream(overrides: Partial<LiveStream> = {}): LiveStream {
    return makeLiveStream({ id: streamId, ...overrides });
}

function makeOwner(): StreamOwner {
    return { stream: makeStream(), whipUrl: "https://ingress.example/whip", streamKey: "the-golden-key" };
}

interface Harness<T> {
    result: { current: T };
    queryClient: QueryClient;
    invalidated: () => string[];
}

function createRetainingQueryClient(): QueryClient {
    return new QueryClient({
        defaultOptions: {
            queries: { retry: false, gcTime: 5 * 60_000, staleTime: 30_000, refetchOnWindowFocus: false },
            mutations: { retry: false },
        },
    });
}

function setup<T>(hook: () => T): Harness<T> {
    const queryClient = createRetainingQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return {
        result,
        queryClient,
        invalidated: () => invalidate.mock.calls.map(call => JSON.stringify(call[0]?.queryKey)),
    };
}

beforeEach(() => {
    mocks.getStreamViewerToken.mockResolvedValue({ token: "lk-token", url: "wss://livekit.test" });
    mocks.joinStreamChat.mockResolvedValue(undefined);
    mocks.resetStreamCredentials.mockResolvedValue({
        whipUrl: "https://ingress.example/whip",
        streamKey: "a-fresh-key",
        hlsEnabled: true,
    });
    mocks.startStream.mockResolvedValue(makeOwner());
    mocks.stopStream.mockResolvedValue(undefined);
    mocks.updateStreamTitle.mockResolvedValue(makeStream({ title: "Chapter two" }));
    mocks.uploadStreamThumbnail.mockResolvedValue(undefined);
});

describe("useStartStream", () => {
    it("starts the stream with the title, the playback mode and the bitrate", async () => {
        // given
        const { result } = setup(() => useStartStream());

        // when
        act(() => {
            result.current.mutate({ title: "Reading the epitaph", defaultMode: "hls", bitrate: 6000 });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.startStream).toHaveBeenCalledWith("Reading the epitaph", "hls", 6000);
    });

    it("refreshes the directory itself so no parent has to pass an onChanged callback", async () => {
        // given
        const { result, invalidated } = setup(() => useStartStream());

        // when
        act(() => {
            result.current.mutate({ title: "Reading the epitaph", defaultMode: "webrtc", bitrate: 0 });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidated()).toContain(JSON.stringify(queryKeys.streams.live()));
        expect(invalidated()).toContain(JSON.stringify(queryKeys.streams.mine()));
    });

    it("never writes the returned stream key into the query cache", async () => {
        // given
        const { result, queryClient } = setup(() => useStartStream());

        // when
        act(() => {
            result.current.mutate({ title: "Reading the epitaph", defaultMode: "webrtc", bitrate: 0 });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(queryClient.getQueryData(queryKeys.streams.mine())).toBeUndefined();
    });
});

describe("useStopStream", () => {
    it("stops the stream and refreshes the directory, the owner view and the stream itself", async () => {
        // given
        const { result, invalidated } = setup(() => useStopStream());

        // when
        act(() => {
            result.current.mutate(streamId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.stopStream).toHaveBeenCalledWith(streamId);
        expect(invalidated()).toContain(JSON.stringify(queryKeys.streams.live()));
        expect(invalidated()).toContain(JSON.stringify(queryKeys.streams.mine()));
        expect(invalidated()).toContain(JSON.stringify(queryKeys.streams.detail(streamId)));
    });
});

describe("useUpdateStreamTitle", () => {
    it("writes the returned stream straight into the cached detail", async () => {
        // given
        const { result, queryClient } = setup(() => useUpdateStreamTitle());

        // when
        act(() => {
            result.current.mutate({ streamId, title: "Chapter two" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.updateStreamTitle).toHaveBeenCalledWith(streamId, "Chapter two");
        expect(queryClient.getQueryData<LiveStream>(queryKeys.streams.detail(streamId))?.title).toBe("Chapter two");
    });

    it("refreshes the directory and the owner view so both carry the new title", async () => {
        // given
        const { result, invalidated } = setup(() => useUpdateStreamTitle());

        // when
        act(() => {
            result.current.mutate({ streamId, title: "Chapter two" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidated()).toContain(JSON.stringify(queryKeys.streams.live()));
        expect(invalidated()).toContain(JSON.stringify(queryKeys.streams.mine()));
    });
});

describe("useResetStreamCredentials", () => {
    it("refreshes the credentials query rather than writing the new key into the cache", async () => {
        // given
        const { result, queryClient, invalidated } = setup(() => useResetStreamCredentials());

        // when
        act(() => {
            result.current.mutate();
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidated()).toContain(JSON.stringify(queryKeys.streams.credentials()));
        expect(queryClient.getQueryData(queryKeys.streams.credentials())).toBeUndefined();
    });

    it("leaves the public directory alone, a new key changes nothing anyone else can see", async () => {
        // given
        const { result, invalidated } = setup(() => useResetStreamCredentials());

        // when
        act(() => {
            result.current.mutate();
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidated()).not.toContain(JSON.stringify(queryKeys.streams.live()));
    });
});

describe("useUploadStreamThumbnail", () => {
    it("uploads the captured frame against the stream it came from", async () => {
        // given
        const { result } = setup(() => useUploadStreamThumbnail());
        const blob = new Blob(["frame"], { type: "image/webp" });

        // when
        act(() => {
            result.current.mutate({ streamId, blob });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.uploadStreamThumbnail).toHaveBeenCalledWith(streamId, blob);
    });

    it("invalidates nothing, it fires every fifty seconds and only the streamer's own tab would refetch", async () => {
        // given
        const { result, invalidated } = setup(() => useUploadStreamThumbnail());

        // when
        act(() => {
            result.current.mutate({ streamId, blob: new Blob(["frame"]) });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidated()).toEqual([]);
    });

    it("reports the failure rather than swallowing it", async () => {
        // given
        mocks.uploadStreamThumbnail.mockRejectedValue(new Error("thumbnail rejected"));
        const { result } = setup(() => useUploadStreamThumbnail());

        // when
        act(() => {
            result.current.mutate({ streamId, blob: new Blob(["frame"]) });
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(result.current.error?.message).toBe("thumbnail rejected");
    });
});

describe("useJoinStreamChat", () => {
    it("joins the chat room of the stream", async () => {
        // given
        const { result } = setup(() => useJoinStreamChat());

        // when
        act(() => {
            result.current.mutate(streamId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.joinStreamChat).toHaveBeenCalledWith(streamId);
    });

    it("reports a refused join so the panel can say so", async () => {
        // given
        mocks.joinStreamChat.mockRejectedValue(new Error("chat is closed"));
        const { result } = setup(() => useJoinStreamChat());

        // when
        act(() => {
            result.current.mutate(streamId);
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(result.current.error?.message).toBe("chat is closed");
    });
});

describe("useStreamViewerToken", () => {
    it("returns the viewer token and never caches it under a query key", async () => {
        // given
        const { result, queryClient } = setup(() => useStreamViewerToken());

        // when
        act(() => {
            result.current.mutate(streamId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(result.current.data).toEqual({ token: "lk-token", url: "wss://livekit.test" });
        expect(queryClient.getQueryCache().getAll()).toEqual([]);
    });
});
