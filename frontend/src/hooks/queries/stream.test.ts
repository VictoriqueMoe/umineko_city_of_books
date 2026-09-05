import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as endpoints from "../../api/endpoints/stream";
import { queryKeys } from "../../api/queryKeys";
import { makeStream as makeLiveStream } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { LiveStream, LiveStreamListResponse, StreamOwner } from "../../types/api";
import { useLiveStreams, useLiveStreamsCount, useMyStream, useStreamByUsername, useStreamCredentials } from "./stream";

vi.mock("../../api/endpoints/stream", () => ({
    listLiveStreams: vi.fn(),
    getStreamByUsername: vi.fn(),
    getMyStream: vi.fn(),
    getStreamCredentials: vi.fn(),
}));

const listLiveStreams = vi.mocked(endpoints.listLiveStreams);
const getStreamByUsername = vi.mocked(endpoints.getStreamByUsername);
const getMyStream = vi.mocked(endpoints.getMyStream);
const getStreamCredentials = vi.mocked(endpoints.getStreamCredentials);

function createRetainingQueryClient(): QueryClient {
    return new QueryClient({
        defaultOptions: {
            queries: { retry: false, gcTime: 5 * 60_000, staleTime: 30_000, refetchOnWindowFocus: false },
            mutations: { retry: false },
        },
    });
}

function makeStream(id: string): LiveStream {
    return makeLiveStream({ id, userId: `user-${id}` });
}

function makeList(streams: LiveStream[], enabled = true): LiveStreamListResponse {
    return { streams, enabled };
}

function makeOwner(id: string): StreamOwner {
    return {
        stream: makeStream(id),
        whipUrl: "https://ingress.example/whip",
        streamKey: "the-golden-key",
    };
}

let queryClient: QueryClient;

beforeEach(() => {
    queryClient = createTestQueryClient();
    listLiveStreams.mockResolvedValue(makeList([makeStream("stream-1"), makeStream("stream-2")]));
    getStreamByUsername.mockResolvedValue(makeStream("stream-1"));
    getMyStream.mockResolvedValue(makeOwner("stream-1"));
    getStreamCredentials.mockResolvedValue({
        whipUrl: "https://ingress.example/whip",
        streamKey: "the-golden-key",
        hlsEnabled: true,
    });
});

describe("useLiveStreams", () => {
    it("reports the streams and the enabled flag once the list arrives", async () => {
        // given
        const { result } = renderHook(() => useLiveStreams(), { wrapper: providerWrapper({ queryClient }) });

        // then
        expect(result.current.loading).toBe(true);

        await waitFor(() => {
            expect(result.current.streams).toHaveLength(2);
        });
        expect(result.current.enabled).toBe(true);
    });
});

describe("useLiveStreamsCount", () => {
    it("counts nothing until the list arrives", () => {
        // given
        const { result } = renderHook(() => useLiveStreamsCount(), { wrapper: providerWrapper({ queryClient }) });

        // then
        expect(result.current.count).toBe(0);
    });

    it("counts the streams the directory query already holds", async () => {
        // given
        const { result } = renderHook(() => useLiveStreamsCount(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => {
            expect(result.current.count).toBe(2);
        });
    });

    it("sees a directory the counter never fetched itself", async () => {
        // given the directory query is already in the cache
        queryClient.setQueryData<LiveStreamListResponse>(queryKeys.streams.live(), makeList([makeStream("stream-9")]));

        // when
        const { result } = renderHook(() => useLiveStreamsCount(), { wrapper: providerWrapper({ queryClient }) });

        // then
        expect(result.current.count).toBe(1);
    });

    it("shares one request with the directory query rather than doubling it", async () => {
        // given both the counter and the directory are mounted at once
        const { result } = renderHook(() => ({ directory: useLiveStreams(), counter: useLiveStreamsCount() }), {
            wrapper: providerWrapper({ queryClient }),
        });

        // then
        await waitFor(() => {
            expect(result.current.counter.count).toBe(2);
        });
        expect(result.current.directory.streams).toHaveLength(2);
        expect(listLiveStreams).toHaveBeenCalledTimes(1);
    });

    it("refetches once per invalidation, not once per observer", async () => {
        // given both the counter and the directory are mounted at once
        const { result } = renderHook(() => ({ directory: useLiveStreams(), counter: useLiveStreamsCount() }), {
            wrapper: providerWrapper({ queryClient }),
        });
        await waitFor(() => {
            expect(result.current.counter.count).toBe(2);
        });

        // when the stream_live invalidation the sync hook owns lands
        listLiveStreams.mockResolvedValue(makeList([makeStream("stream-1")]));
        await act(async () => {
            await queryClient.invalidateQueries({ queryKey: queryKeys.streams.live() });
        });

        // then
        expect(listLiveStreams).toHaveBeenCalledTimes(2);
        await waitFor(() => {
            expect(result.current.counter.count).toBe(1);
        });
    });
});

describe("useStreamByUsername", () => {
    it("reads a streamer's current stream by their name", async () => {
        // given
        const { result } = renderHook(() => useStreamByUsername("beatrice"), {
            wrapper: providerWrapper({ queryClient }),
        });

        // then
        await waitFor(() => {
            expect(result.current.stream?.id).toBe("stream-1");
        });
        expect(getStreamByUsername).toHaveBeenCalledWith("beatrice");
        expect(queryClient.getQueryData(queryKeys.streams.byUsername("beatrice"))).toBeDefined();
    });

    it("asks for nothing while the route has no username", () => {
        // given
        const { result } = renderHook(() => useStreamByUsername(undefined), {
            wrapper: providerWrapper({ queryClient }),
        });

        // then
        expect(getStreamByUsername).not.toHaveBeenCalled();
        expect(result.current.stream).toBeNull();
        expect(result.current.loading).toBe(false);
    });

    it("drops the stream rather than leaving stale live data on screen", async () => {
        // given a streamer who was live and has now stopped, so the lookup 404s
        const { result } = renderHook(() => useStreamByUsername("beatrice"), {
            wrapper: providerWrapper({ queryClient }),
        });
        await waitFor(() => {
            expect(result.current.stream).not.toBeNull();
        });

        // when the refetch fails
        getStreamByUsername.mockRejectedValue(new Error("stream not found"));
        await act(async () => {
            await queryClient.refetchQueries({ queryKey: queryKeys.streams.byUsername("beatrice") });
        });

        // then nothing stale survives
        await waitFor(() => {
            expect(result.current.stream).toBeNull();
        });
        expect(result.current.error).toBe("stream not found");
    });
});

describe("useMyStream", () => {
    it("reports the owner view with its ingress credentials", async () => {
        // given
        const { result } = renderHook(() => useMyStream(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => {
            expect(result.current.owner?.streamKey).toBe("the-golden-key");
        });
        expect(result.current.error).toBe("");
    });

    it("reports no owner rather than an error when the member is not streaming", async () => {
        // given
        getMyStream.mockResolvedValue(null);

        // when
        const { result } = renderHook(() => useMyStream(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => {
            expect(result.current.loading).toBe(false);
        });
        expect(result.current.owner).toBeNull();
        expect(result.current.error).toBe("");
    });

    it("surfaces the failure the panel used to swallow", async () => {
        // given
        getMyStream.mockRejectedValue(new Error("could not reach the ingress"));

        // when
        const { result } = renderHook(() => useMyStream(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => {
            expect(result.current.error).toBe("could not reach the ingress");
        });
    });

    it("asks for nothing while it is disabled", () => {
        // given
        renderHook(() => useMyStream(false), { wrapper: providerWrapper({ queryClient }) });

        // then
        expect(getMyStream).not.toHaveBeenCalled();
    });

    it("drops the stream key out of the cache the moment the last reader unmounts", async () => {
        // given a client that would otherwise hold every query for five minutes
        const retaining = createRetainingQueryClient();
        const { result, unmount } = renderHook(() => ({ mine: useMyStream(), directory: useLiveStreams() }), {
            wrapper: providerWrapper({ queryClient: retaining }),
        });
        await waitFor(() => {
            expect(result.current.mine.owner?.streamKey).toBe("the-golden-key");
            expect(result.current.directory.streams).toHaveLength(2);
        });

        // when
        unmount();

        // then
        await waitFor(() => {
            expect(retaining.getQueryData(queryKeys.streams.mine())).toBeUndefined();
        });
        expect(retaining.getQueryData(queryKeys.streams.live())).toBeDefined();
    });

    it("refetches the owner view on every mount rather than trusting a cached copy", async () => {
        // given
        const retaining = createRetainingQueryClient();
        const first = renderHook(() => useMyStream(), { wrapper: providerWrapper({ queryClient: retaining }) });
        await waitFor(() => {
            expect(first.result.current.owner).not.toBeNull();
        });
        first.unmount();

        // when
        const second = renderHook(() => useMyStream(), { wrapper: providerWrapper({ queryClient: retaining }) });

        // then
        await waitFor(() => {
            expect(second.result.current.owner).not.toBeNull();
        });
        expect(getMyStream).toHaveBeenCalledTimes(2);
    });
});

describe("useStreamCredentials", () => {
    it("reports the ingress url, key and the smooth playback flag", async () => {
        // given
        const { result } = renderHook(() => useStreamCredentials(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => {
            expect(result.current.credentials?.hlsEnabled).toBe(true);
        });
        expect(result.current.credentials?.streamKey).toBe("the-golden-key");
    });

    it("surfaces the failure instead of leaving the panel blank", async () => {
        // given
        getStreamCredentials.mockRejectedValue(new Error("no key for you"));

        // when
        const { result } = renderHook(() => useStreamCredentials(), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => {
            expect(result.current.error).toBe("no key for you");
        });
        expect(result.current.credentials).toBeNull();
    });

    it("drops the credentials out of the cache the moment the last reader unmounts", async () => {
        // given a client that would otherwise hold every query for five minutes
        const retaining = createRetainingQueryClient();
        const { result, unmount } = renderHook(() => useStreamCredentials(), {
            wrapper: providerWrapper({ queryClient: retaining }),
        });
        await waitFor(() => {
            expect(result.current.credentials).not.toBeNull();
        });

        // when
        unmount();

        // then
        await waitFor(() => {
            expect(retaining.getQueryData(queryKeys.streams.credentials())).toBeUndefined();
        });
    });
});
