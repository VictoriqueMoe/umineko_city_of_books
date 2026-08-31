import { act, renderHook } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { expectInvalidated } from "../../../test-utils/query";
import { makeStream } from "../../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../../test-utils/render";
import type { LiveStream, LiveStreamListResponse } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { dispatch } from "../bus";
import type { RealtimeEvent } from "../events";
import { useStreamDirectorySync } from "./useStreamDirectorySync";

function emit(event: RealtimeEvent): void {
    act(() => {
        dispatch(event);
    });
}

function seedDirectory(queryClient: QueryClient, streams: LiveStream[]): void {
    queryClient.setQueryData<LiveStreamListResponse>(queryKeys.streams.live(), { streams, enabled: true });
}

function readDirectory(queryClient: QueryClient): LiveStreamListResponse | undefined {
    return queryClient.getQueryData<LiveStreamListResponse>(queryKeys.streams.live());
}

let queryClient: QueryClient;

function mount(): { unmount: () => void } {
    return renderHook(() => useStreamDirectorySync(), { wrapper: providerWrapper({ queryClient }) });
}

beforeEach(() => {
    queryClient = createTestQueryClient();
});

describe("useStreamDirectorySync", () => {
    it("invalidates the live directory when a stream goes live", () => {
        // given
        vi.spyOn(queryClient, "invalidateQueries");
        mount();

        // when
        emit({ type: "stream_live", data: makeStream() });

        // then
        expectInvalidated(queryClient, queryKeys.streams.live());
    });

    it("invalidates the live directory when a stream goes offline", () => {
        // given
        vi.spyOn(queryClient, "invalidateQueries");
        mount();

        // when
        emit({ type: "stream_offline", data: { streamId: "stream-1" } });

        // then
        expectInvalidated(queryClient, queryKeys.streams.live());
    });

    it("patches the viewer count of the named stream and leaves its neighbours alone", () => {
        // given
        seedDirectory(queryClient, [makeStream(), makeStream({ id: "stream-2", viewerCount: 9 })]);
        mount();

        // when
        emit({ type: "stream_viewers", data: { streamId: "stream-1", viewerCount: 44 } });

        // then
        expect(readDirectory(queryClient)?.streams.map(s => [s.id, s.viewerCount])).toEqual([
            ["stream-1", 44],
            ["stream-2", 9],
        ]);
    });

    it("patches the title of the named stream", () => {
        // given
        seedDirectory(queryClient, [makeStream()]);
        mount();

        // when
        emit({ type: "stream_title", data: { streamId: "stream-1", title: "Now solving the epitaph" } });

        // then
        expect(readDirectory(queryClient)?.streams[0].title).toBe("Now solving the epitaph");
    });

    it("leaves the list untouched when the event names a stream it does not hold", () => {
        // given
        seedDirectory(queryClient, [makeStream()]);
        mount();

        // when
        emit({ type: "stream_viewers", data: { streamId: "stream-other", viewerCount: 44 } });
        emit({ type: "stream_title", data: { streamId: "stream-other", title: "Not mine" } });

        // then
        expect(readDirectory(queryClient)?.streams).toEqual([makeStream()]);
    });

    it("does not seed a directory list when nothing is cached", () => {
        // given
        mount();

        // when
        emit({ type: "stream_viewers", data: { streamId: "stream-1", viewerCount: 44 } });

        // then
        expect(readDirectory(queryClient)).toBeUndefined();
    });

    it("stops reacting once it is unmounted", () => {
        // given
        seedDirectory(queryClient, [makeStream()]);
        const view = mount();

        // when
        view.unmount();
        emit({ type: "stream_viewers", data: { streamId: "stream-1", viewerCount: 44 } });

        // then
        expect(readDirectory(queryClient)?.streams[0].viewerCount).toBe(3);
    });
});
