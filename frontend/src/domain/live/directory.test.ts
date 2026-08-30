import { describe, expect, it } from "vitest";
import type { LiveStream, LiveStreamListResponse } from "../../types/api";
import { applyStreamPatch, applyTitle, applyViewerCount } from "./directory";

function stream(id: string, overrides: Partial<LiveStream> = {}): LiveStream {
    return {
        id,
        userId: `user-${id}`,
        title: `Stream ${id}`,
        status: "live",
        viewerCount: 0,
        streamerUsername: `streamer-${id}`,
        streamerDisplayName: `Streamer ${id}`,
        streamerAvatarUrl: "",
        defaultMode: "webrtc",
        ...overrides,
    };
}

function list(...streams: LiveStream[]): LiveStreamListResponse {
    return { streams, enabled: true };
}

describe("applyViewerCount", () => {
    it("updates the named stream and leaves the others untouched", () => {
        // given
        const before = list(stream("a", { viewerCount: 3 }), stream("b", { viewerCount: 7 }));

        // when
        const after = applyViewerCount(before, "a", 4);

        // then
        expect(after?.streams.map(entry => entry.viewerCount)).toEqual([4, 7]);
        expect(after?.streams[1]).toBe(before.streams[1]);
    });

    it("keeps the enabled flag and every other field of the patched stream", () => {
        // given
        const before = list(stream("a", { viewerCount: 3, title: "Reading Ep 1" }));

        // when
        const after = applyViewerCount(before, "a", 12);

        // then
        expect(after).toEqual(list(stream("a", { viewerCount: 12, title: "Reading Ep 1" })));
    });

    it("does not mutate the list it was given", () => {
        // given
        const before = list(stream("a", { viewerCount: 3 }));

        // when
        applyViewerCount(before, "a", 99);

        // then
        expect(before.streams[0].viewerCount).toBe(3);
    });

    it("passes an absent cache entry straight through, so nothing is invented", () => {
        // given
        const before = undefined;

        // when
        const after = applyViewerCount(before, "a", 4);

        // then
        expect(after).toBeUndefined();
    });

    it("leaves the list alone when the stream is not in it", () => {
        // given
        const before = list(stream("a", { viewerCount: 3 }));

        // when
        const after = applyViewerCount(before, "missing", 4);

        // then
        expect(after).toEqual(before);
    });
});

describe("applyTitle", () => {
    it("retitles the named stream only", () => {
        // given
        const before = list(stream("a", { title: "Old" }), stream("b", { title: "Other" }));

        // when
        const after = applyTitle(before, "a", "New");

        // then
        expect(after?.streams.map(entry => entry.title)).toEqual(["New", "Other"]);
    });

    it("passes an absent cache entry straight through", () => {
        // given
        const before = undefined;

        // when
        const after = applyTitle(before, "a", "New");

        // then
        expect(after).toBeUndefined();
    });
});

describe("applyStreamPatch", () => {
    it("applies an arbitrary patch to one entry", () => {
        // given
        const before = list(stream("a", { status: "live" }), stream("b"));

        // when
        const after = applyStreamPatch(before, "a", entry => ({ ...entry, status: "offline" }));

        // then
        expect(after?.streams.map(entry => entry.status)).toEqual(["offline", "live"]);
    });
});
