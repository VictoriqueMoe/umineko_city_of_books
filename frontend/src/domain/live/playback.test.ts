import { describe, expect, it } from "vitest";
import {
    isOwnStream,
    resolvePlaybackMode,
    resolvePlaybackPlan,
    shouldConnectRoom,
    type PlaybackPlanInput,
    type PlaybackStream,
} from "./playback";

const OWNER_ID = "beatrice";
const STRANGER_ID = "battler";

function stream(overrides: Partial<PlaybackStream> = {}): PlaybackStream {
    return {
        userId: OWNER_ID,
        status: "live",
        defaultMode: "webrtc",
        ...overrides,
    };
}

function input(overrides: Partial<PlaybackPlanInput> = {}): PlaybackPlanInput {
    return {
        stream: stream(),
        viewerId: STRANGER_ID,
        modeOverride: null,
        showOwnPreview: false,
        ...overrides,
    };
}

describe("resolvePlaybackMode", () => {
    it("prefers the viewer's own override over the stream default", () => {
        // given
        const smooth = stream({ defaultMode: "hls", hlsUrl: "https://example.test/live.m3u8" });

        // when
        const mode = resolvePlaybackMode(smooth, "webrtc");

        // then
        expect(mode).toBe("webrtc");
    });

    it("honours an hls default only when the stream actually has an hls url", () => {
        // given
        const withUrl = stream({ defaultMode: "hls", hlsUrl: "https://example.test/live.m3u8" });
        const withoutUrl = stream({ defaultMode: "hls" });

        // when
        const modes = [resolvePlaybackMode(withUrl, null), resolvePlaybackMode(withoutUrl, null)];

        // then
        expect(modes).toEqual(["hls", "webrtc"]);
    });

    it("falls back to webrtc when there is no stream at all", () => {
        // given
        const missing = undefined;

        // when
        const mode = resolvePlaybackMode(missing, null);

        // then
        expect(mode).toBe("webrtc");
    });
});

describe("isOwnStream", () => {
    it("matches the signed-in viewer against the stream owner", () => {
        // given
        const mine = stream({ userId: OWNER_ID });

        // when
        const own = isOwnStream(mine, OWNER_ID);

        // then
        expect(own).toBe(true);
    });

    it("is false for a signed-out viewer even on an owned stream", () => {
        // given
        const mine = stream({ userId: OWNER_ID });

        // when
        const own = isOwnStream(mine, null);

        // then
        expect(own).toBe(false);
    });
});

describe("resolvePlaybackPlan", () => {
    it("gives an ordinary viewer of a webrtc stream both the room and the media", () => {
        // given
        const watching = input({ viewerId: STRANGER_ID });

        // when
        const plan = resolvePlaybackPlan(watching);

        // then
        expect(plan).toMatchObject({
            isLive: true,
            mode: "webrtc",
            isOwnStream: false,
            showsPlayback: true,
            wantsRoom: true,
            wantsMedia: true,
            autoSubscribe: true,
            monitorsWithoutMedia: false,
        });
    });

    it("keeps a viewer on hls out of the livekit room entirely", () => {
        // given
        const smooth = input({
            stream: stream({ defaultMode: "hls", hlsUrl: "https://example.test/live.m3u8" }),
            viewerId: STRANGER_ID,
        });

        // when
        const plan = resolvePlaybackPlan(smooth);

        // then
        expect(plan).toMatchObject({ mode: "hls", showsPlayback: true, wantsMedia: false, wantsRoom: false });
    });

    it("joins the owner to the room without subscribing when their preview is hidden", () => {
        // given
        const owner = input({ viewerId: OWNER_ID, showOwnPreview: false });

        // when
        const plan = resolvePlaybackPlan(owner);

        // then
        expect(plan).toMatchObject({
            isOwnStream: true,
            showsPlayback: false,
            wantsRoom: true,
            wantsMedia: false,
            autoSubscribe: false,
            monitorsWithoutMedia: true,
        });
    });

    it("is the only shape that wants a room while wanting no media", () => {
        // given
        const everyCombination: PlaybackPlanInput[] = [
            input({ viewerId: STRANGER_ID, showOwnPreview: false }),
            input({ viewerId: STRANGER_ID, showOwnPreview: true }),
            input({ viewerId: OWNER_ID, showOwnPreview: false }),
            input({ viewerId: OWNER_ID, showOwnPreview: true }),
            input({ viewerId: OWNER_ID, showOwnPreview: false, modeOverride: "hls" }),
            input({ viewerId: OWNER_ID, showOwnPreview: true, modeOverride: "hls" }),
        ];

        // when
        const monitoring = everyCombination.map(candidate => resolvePlaybackPlan(candidate).monitorsWithoutMedia);

        // then
        expect(monitoring).toEqual([false, false, true, false, true, false]);
    });

    it("still monitors without media when the owner has chosen hls, because a hidden preview beats the mode", () => {
        // given
        const owner = input({
            stream: stream({ defaultMode: "hls", hlsUrl: "https://example.test/live.m3u8" }),
            viewerId: OWNER_ID,
            showOwnPreview: false,
        });

        // when
        const plan = resolvePlaybackPlan(owner);

        // then
        expect(plan).toMatchObject({ mode: "hls", wantsRoom: true, wantsMedia: false, monitorsWithoutMedia: true });
    });

    it("drops the owner's room subscription back to media once they show their preview", () => {
        // given
        const previewing = input({ viewerId: OWNER_ID, showOwnPreview: true });

        // when
        const plan = resolvePlaybackPlan(previewing);

        // then
        expect(plan).toMatchObject({ showsPlayback: true, wantsRoom: true, wantsMedia: true, autoSubscribe: true });
    });

    it("reports a stream that is not live", () => {
        // given
        const offline = input({ stream: stream({ status: "offline" }) });

        // when
        const plan = resolvePlaybackPlan(offline);

        // then
        expect(plan.isLive).toBe(false);
    });
});

describe("shouldConnectRoom", () => {
    it("connects only when there is a stream id, the stream is live and the plan wants a room", () => {
        // given
        const plan = resolvePlaybackPlan(input());

        // when
        const decisions = [shouldConnectRoom("stream-1", plan), shouldConnectRoom(undefined, plan)];

        // then
        expect(decisions).toEqual([true, false]);
    });

    it("refuses to connect an offline stream", () => {
        // given
        const plan = resolvePlaybackPlan(input({ stream: stream({ status: "offline" }) }));

        // when
        const connects = shouldConnectRoom("stream-1", plan);

        // then
        expect(connects).toBe(false);
    });

    it("connects the monitoring owner, who wants a room and no media", () => {
        // given
        const plan = resolvePlaybackPlan(input({ viewerId: OWNER_ID, showOwnPreview: false }));

        // when
        const connects = shouldConnectRoom("stream-1", plan);

        // then
        expect(connects).toBe(true);
        expect(plan.autoSubscribe).toBe(false);
    });
});
