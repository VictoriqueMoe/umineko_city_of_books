import { act, renderHook } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { expectInvalidated } from "../../../test-utils/query";
import { makeStream } from "../../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../../test-utils/render";
import type { LiveStream } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { dispatch } from "../bus";
import type { RealtimeEvent } from "../events";
import { useStreamDetailSync } from "./useStreamDetailSync";

function emit(event: RealtimeEvent): void {
    act(() => {
        dispatch(event);
    });
}

function readDetail(queryClient: QueryClient, streamId: string): LiveStream | undefined {
    return queryClient.getQueryData<LiveStream>(queryKeys.streams.detail(streamId));
}

let queryClient: QueryClient;

function mount(streamId: string | undefined) {
    return renderHook(({ id }: { id: string | undefined }) => useStreamDetailSync(id), {
        wrapper: providerWrapper({ queryClient }),
        initialProps: { id: streamId },
    });
}

beforeEach(() => {
    queryClient = createTestQueryClient();
});

describe("useStreamDetailSync", () => {
    it("invalidates its own stream when that stream goes live", () => {
        // given
        vi.spyOn(queryClient, "invalidateQueries");
        mount("stream-1");

        // when
        emit({ type: "stream_live", data: makeStream({ id: "stream-1" }) });

        // then
        expectInvalidated(queryClient, queryKeys.streams.detail("stream-1"));
    });

    it("invalidates its own stream when that stream goes offline", () => {
        // given
        vi.spyOn(queryClient, "invalidateQueries");
        mount("stream-1");

        // when
        emit({ type: "stream_offline", data: { streamId: "stream-1" } });

        // then
        expectInvalidated(queryClient, queryKeys.streams.detail("stream-1"));
    });

    it("ignores another stream going live or offline", () => {
        // given
        vi.spyOn(queryClient, "invalidateQueries");
        mount("stream-1");

        // when
        emit({ type: "stream_live", data: makeStream({ id: "stream-other" }) });
        emit({ type: "stream_offline", data: { streamId: "stream-other" } });

        // then
        expect(queryClient.invalidateQueries).not.toHaveBeenCalled();
    });

    it("patches the cached title of its own stream", () => {
        // given
        queryClient.setQueryData<LiveStream>(queryKeys.streams.detail("stream-1"), makeStream());
        mount("stream-1");

        // when
        emit({ type: "stream_title", data: { streamId: "stream-1", title: "Now solving the epitaph" } });

        // then
        expect(readDetail(queryClient, "stream-1")?.title).toBe("Now solving the epitaph");
    });

    it("ignores a title for another stream", () => {
        // given
        queryClient.setQueryData<LiveStream>(queryKeys.streams.detail("stream-1"), makeStream());
        mount("stream-1");

        // when
        emit({ type: "stream_title", data: { streamId: "stream-other", title: "Not mine" } });

        // then
        expect(readDetail(queryClient, "stream-1")?.title).toBe("Reading the epitaph");
    });

    it("does not seed a detail entry for a stream it has never fetched", () => {
        // given
        mount("stream-1");

        // when
        emit({ type: "stream_title", data: { streamId: "stream-1", title: "Now solving the epitaph" } });

        // then
        expect(readDetail(queryClient, "stream-1")).toBeUndefined();
    });

    it("reacts to nothing while it has no stream id", () => {
        // given
        vi.spyOn(queryClient, "invalidateQueries");
        mount(undefined);

        // when
        emit({ type: "stream_live", data: makeStream() });
        emit({ type: "stream_offline", data: { streamId: "stream-1" } });

        // then
        expect(queryClient.invalidateQueries).not.toHaveBeenCalled();
    });

    it("follows the stream id from the latest render", () => {
        // given
        vi.spyOn(queryClient, "invalidateQueries");
        const view = mount("stream-1");

        // when
        view.rerender({ id: "stream-2" });
        emit({ type: "stream_offline", data: { streamId: "stream-2" } });

        // then
        expectInvalidated(queryClient, queryKeys.streams.detail("stream-2"));
    });

    it("stops reacting once it is unmounted", () => {
        // given
        vi.spyOn(queryClient, "invalidateQueries");
        const view = mount("stream-1");

        // when
        view.unmount();
        emit({ type: "stream_offline", data: { streamId: "stream-1" } });

        // then
        expect(queryClient.invalidateQueries).not.toHaveBeenCalled();
    });
});
