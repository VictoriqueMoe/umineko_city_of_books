import { renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../api/queryKeys";
import type { RealtimeEvent } from "../api/realtime/events";
import { useStreamDirectorySync } from "../api/realtime/sync/useStreamDirectorySync";
import { makeStream } from "../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../test-utils/render";
import { emitSettledRealtimeEvent } from "../test-utils/ws";
import type { LiveStreamListResponse } from "../types/api";
import { useLiveDirectory } from "./useLiveDirectory";

const mocks = vi.hoisted(() => ({
    listLiveStreams: vi.fn(),
    getStream: vi.fn(),
    getMyStream: vi.fn(),
    getStreamCredentials: vi.fn(),
}));

vi.mock("../api/endpoints/stream", () => mocks);

async function emit(event: RealtimeEvent): Promise<void> {
    await emitSettledRealtimeEvent(event);
}

let queryClient: QueryClient;

function mountDirectory() {
    return renderHook(() => useLiveDirectory(), { wrapper: providerWrapper({ queryClient }) });
}

function mountDirectoryBesideTheGlobalSync() {
    return renderHook(
        () => {
            useStreamDirectorySync();

            return useLiveDirectory();
        },
        { wrapper: providerWrapper({ queryClient }) },
    );
}

function readDirectory(): LiveStreamListResponse | undefined {
    return queryClient.getQueryData<LiveStreamListResponse>(queryKeys.streams.live());
}

function liveInvalidations(calls: unknown[][]): number {
    const wanted = JSON.stringify(queryKeys.streams.live());

    return calls.filter(call => JSON.stringify((call[0] as { queryKey?: unknown })?.queryKey) === wanted).length;
}

beforeEach(() => {
    queryClient = createTestQueryClient();
    mocks.listLiveStreams.mockResolvedValue({ streams: [makeStream()], enabled: true });
});

describe("useLiveDirectory", () => {
    it("hands the page the streams the directory is holding", async () => {
        // given
        const view = mountDirectory();

        // when
        await waitFor(() => expect(view.result.current.loading).toBe(false));

        // then
        expect(view.result.current.streams.map(stream => stream.id)).toEqual(["stream-1"]);
        expect(view.result.current.enabled).toBe(true);
    });

    it("says so when live streaming is switched off", async () => {
        // given
        mocks.listLiveStreams.mockResolvedValue({ streams: [], enabled: false });
        const view = mountDirectory();

        // when
        await waitFor(() => expect(view.result.current.loading).toBe(false));

        // then
        expect(view.result.current.enabled).toBe(false);
    });

    it("subscribes to no stream event of its own, leaving presence to the sync hook", async () => {
        // given
        const view = mountDirectory();
        await waitFor(() => expect(view.result.current.loading).toBe(false));
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        await emit({ type: "stream_live", data: makeStream() });
        await emit({ type: "stream_offline", data: { streamId: "stream-1" } });

        // then
        expect(liveInvalidations(invalidateQueries.mock.calls)).toBe(0);
    });

    it("patches nothing itself when a viewer count or a title arrives", async () => {
        // given
        const view = mountDirectory();
        await waitFor(() => expect(view.result.current.loading).toBe(false));

        // when
        await emit({ type: "stream_viewers", data: { streamId: "stream-1", viewerCount: 44 } });
        await emit({ type: "stream_title", data: { streamId: "stream-1", title: "Now solving the epitaph" } });

        // then
        expect(readDirectory()?.streams[0].viewerCount).toBe(3);
        expect(readDirectory()?.streams[0].title).toBe("Reading the epitaph");
    });

    it("refetches the directory when a stream goes live", async () => {
        // given
        const view = mountDirectoryBesideTheGlobalSync();
        await waitFor(() => expect(view.result.current.loading).toBe(false));
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        await emit({ type: "stream_live", data: makeStream() });

        // then
        expect(liveInvalidations(invalidateQueries.mock.calls)).toBe(1);
    });

    it("refetches the directory when a stream goes offline", async () => {
        // given
        const view = mountDirectoryBesideTheGlobalSync();
        await waitFor(() => expect(view.result.current.loading).toBe(false));
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        await emit({ type: "stream_offline", data: { streamId: "stream-1" } });

        // then
        expect(liveInvalidations(invalidateQueries.mock.calls)).toBe(1);
    });

    it("updates a single stream's watcher count in place", async () => {
        // given
        const view = mountDirectoryBesideTheGlobalSync();
        await waitFor(() => expect(view.result.current.loading).toBe(false));
        const setQueryData = vi.spyOn(queryClient, "setQueryData");

        // when
        await emit({ type: "stream_viewers", data: { streamId: "stream-1", viewerCount: 44 } });

        // then
        expect(setQueryData).toHaveBeenCalledTimes(1);
        expect(readDirectory()?.streams[0].viewerCount).toBe(44);
    });

    it("updates a single stream's title in place", async () => {
        // given
        const view = mountDirectoryBesideTheGlobalSync();
        await waitFor(() => expect(view.result.current.loading).toBe(false));
        const setQueryData = vi.spyOn(queryClient, "setQueryData");

        // when
        await emit({ type: "stream_title", data: { streamId: "stream-1", title: "Now solving the epitaph" } });

        // then
        expect(setQueryData).toHaveBeenCalledTimes(1);
        expect(readDirectory()?.streams[0].title).toBe("Now solving the epitaph");
    });

    it("leaves other streams alone when one of them changes title", async () => {
        // given
        mocks.listLiveStreams.mockResolvedValue({
            streams: [makeStream(), makeStream({ id: "stream-2", title: "Higurashi marathon" })],
            enabled: true,
        });
        const view = mountDirectoryBesideTheGlobalSync();
        await waitFor(() => expect(view.result.current.loading).toBe(false));

        // when
        await emit({ type: "stream_title", data: { streamId: "stream-1", title: "Now solving the epitaph" } });

        // then
        expect(readDirectory()?.streams[0].title).toBe("Now solving the epitaph");
        expect(readDirectory()?.streams[1].title).toBe("Higurashi marathon");
    });
});
