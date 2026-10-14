import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import type * as BusModule from "../api/realtime/bus";
import { makeWSHarness, type RealtimeTestNames, type WSHarness } from "../test-utils/ws";
import { usePostCommentEvents, usePostLikeEvents } from "./usePostEvents";

const holder = vi.hoisted(() => ({
    ws: null as unknown as WSHarness,
    watchPost: vi.fn<(postId: string) => () => void>(),
    releases: new Map<string, Mock<() => void>>(),
}));

vi.mock("../api/realtime/bus", async importOriginal => {
    const actual = await importOriginal<typeof BusModule>();

    return {
        ...actual,
        subscribe: (names: RealtimeTestNames, handler: BusModule.RealtimeEventHandler) =>
            holder.ws.subscribe(names, handler),
    };
});

vi.mock("../api/realtime/postWatch", () => ({ watchPost: holder.watchPost }));

function releaseFor(postId: string): Mock<() => void> {
    const release = holder.releases.get(postId);
    if (!release) {
        throw new Error(`post ${postId} was never watched`);
    }

    return release;
}

beforeEach(() => {
    holder.ws = makeWSHarness();
    holder.releases.clear();
    holder.watchPost.mockReset();
    holder.watchPost.mockImplementation(postId => {
        const release = vi.fn<() => void>();
        holder.releases.set(postId, release);

        return release;
    });
});

const watchingHooks: { name: string; useWatching: (postId: string, callback: () => void) => void }[] = [
    { name: "usePostLikeEvents", useWatching: usePostLikeEvents },
    { name: "usePostCommentEvents", useWatching: usePostCommentEvents },
];

for (const entry of watchingHooks) {
    describe(`${entry.name} post watching`, () => {
        it("asks the server to watch the post while mounted", () => {
            // when
            renderHook(() => entry.useWatching("post-1", vi.fn()));

            // then
            expect(holder.watchPost).toHaveBeenCalledExactlyOnceWith("post-1");
            expect(releaseFor("post-1")).not.toHaveBeenCalled();
        });

        it("lets go of the post when it unmounts", () => {
            // given
            const view = renderHook(() => entry.useWatching("post-1", vi.fn()));

            // when
            view.unmount();

            // then
            expect(releaseFor("post-1")).toHaveBeenCalledOnce();
        });

        it("swaps the watched post when the id changes", () => {
            // given
            const view = renderHook(({ postId }) => entry.useWatching(postId, vi.fn()), {
                initialProps: { postId: "post-1" },
            });

            // when
            view.rerender({ postId: "post-2" });

            // then
            expect(releaseFor("post-1")).toHaveBeenCalledOnce();
            expect(holder.watchPost).toHaveBeenLastCalledWith("post-2");
            expect(releaseFor("post-2")).not.toHaveBeenCalled();
        });

        it("watches nothing for an empty post id", () => {
            // when
            renderHook(() => entry.useWatching("", vi.fn()));

            // then
            expect(holder.watchPost).not.toHaveBeenCalled();
        });
    });
}

describe("usePostLikeEvents", () => {
    it("listens for likes alone", () => {
        // when
        renderHook(() => usePostLikeEvents("post-1", vi.fn()));

        // then
        expect(holder.ws.subscribe).toHaveBeenCalledTimes(1);
        expect(holder.ws.subscribe.mock.calls[0][0]).toBe("post_like");
    });

    it("hands on the change to the count", () => {
        // given
        const onLike = vi.fn();
        renderHook(() => usePostLikeEvents("post-1", onLike));

        // when
        holder.ws.emit({ type: "post_like", data: { post_id: "post-1", delta: -1 } });

        // then
        expect(onLike).toHaveBeenCalledWith(-1);
    });

    it("ignores a like meant for a different post", () => {
        // given
        const onLike = vi.fn();
        renderHook(() => usePostLikeEvents("post-1", onLike));

        // when
        holder.ws.emit({ type: "post_like", data: { post_id: "post-2", delta: 1 } });

        // then
        expect(onLike).not.toHaveBeenCalled();
    });

    it("drops its subscription when it goes away", () => {
        // given
        const view = renderHook(() => usePostLikeEvents("post-1", vi.fn()));

        // when
        view.unmount();

        // then
        expect(holder.ws.unsubscribe).toHaveBeenCalledTimes(1);
    });
});

describe("usePostCommentEvents", () => {
    it("listens for comments alone", () => {
        // when
        renderHook(() => usePostCommentEvents("post-1", vi.fn()));

        // then
        expect(holder.ws.subscribe).toHaveBeenCalledTimes(1);
        expect(holder.ws.subscribe.mock.calls[0][0]).toBe("post_comment");
    });

    it("calls back when a comment lands on the post being read", () => {
        // given
        const onComment = vi.fn();
        renderHook(() => usePostCommentEvents("post-1", onComment));

        // when
        holder.ws.emit({ type: "post_comment", data: { post_id: "post-1", comment_id: "c1" } });

        // then
        expect(onComment).toHaveBeenCalledOnce();
    });

    it("ignores a comment on a different post", () => {
        // given
        const onComment = vi.fn();
        renderHook(() => usePostCommentEvents("post-1", onComment));

        // when
        holder.ws.emit({ type: "post_comment", data: { post_id: "post-2", comment_id: "c1" } });

        // then
        expect(onComment).not.toHaveBeenCalled();
    });

    it("drops its subscription when it goes away", () => {
        // given
        const view = renderHook(() => usePostCommentEvents("post-1", vi.fn()));

        // when
        view.unmount();

        // then
        expect(holder.ws.unsubscribe).toHaveBeenCalledTimes(1);
    });
});
