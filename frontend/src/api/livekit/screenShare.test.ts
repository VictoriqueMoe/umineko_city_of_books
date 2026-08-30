import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Room } from "livekit-client";
import { SCREEN_SHARE_PRESETS, setScreenShareEnabled } from "./screenShare";

const mocks = vi.hoisted(() => ({
    setScreenShareEnabled: vi.fn<(...args: unknown[]) => Promise<void>>(() => Promise.resolve()),
}));

vi.mock("livekit-client", () => ({
    AudioPresets: { musicHighQualityStereo: { maxBitrate: 510_000 } },
}));

function fakeRoom(): Room {
    return { localParticipant: { setScreenShareEnabled: mocks.setScreenShareEnabled } } as unknown as Room;
}

beforeEach(() => {
    mocks.setScreenShareEnabled.mockClear();
});

describe("SCREEN_SHARE_PRESETS", () => {
    it("keeps gaming and screenshare as one table, each preset whole", () => {
        // then
        expect(SCREEN_SHARE_PRESETS).toEqual({
            gaming: {
                contentHint: "motion",
                resolution: { width: 1920, height: 1080, frameRate: 60 },
                videoCodec: "vp9",
                degradationPreference: "maintain-framerate",
                maxBitrate: 6_000_000,
            },
            screenshare: {
                contentHint: "detail",
                resolution: { width: 1920, height: 1080, frameRate: 15 },
                videoCodec: "vp9",
                degradationPreference: "maintain-resolution",
                maxBitrate: 2_500_000,
            },
        });
    });
});

describe("setScreenShareEnabled", () => {
    it("publishes a gaming share with stereo music audio and framerate held over resolution", async () => {
        // given
        const room = fakeRoom();

        // when
        await setScreenShareEnabled(room, true, "gaming");

        // then
        expect(mocks.setScreenShareEnabled).toHaveBeenCalledExactlyOnceWith(
            true,
            {
                audio: { echoCancellation: false, noiseSuppression: false, autoGainControl: false },
                contentHint: "motion",
                resolution: { width: 1920, height: 1080, frameRate: 60 },
            },
            {
                audioPreset: { maxBitrate: 510_000 },
                dtx: false,
                red: false,
                forceStereo: true,
                videoCodec: "vp9",
                degradationPreference: "maintain-framerate",
                screenShareEncoding: { maxBitrate: 6_000_000, maxFramerate: 60 },
            },
        );
    });

    it("publishes a screenshare with resolution held over framerate, so text stays readable", async () => {
        // given
        const room = fakeRoom();

        // when
        await setScreenShareEnabled(room, true, "screenshare");

        // then
        const [, capture, publish] = mocks.setScreenShareEnabled.mock.calls[0];
        expect(capture).toMatchObject({ contentHint: "detail", resolution: { frameRate: 15 } });
        expect(publish).toMatchObject({
            degradationPreference: "maintain-resolution",
            screenShareEncoding: { maxBitrate: 2_500_000, maxFramerate: 15 },
        });
    });

    it("stops the share when told to, carrying the same preset it was started with", async () => {
        // given
        const room = fakeRoom();

        // when
        await setScreenShareEnabled(room, false, "gaming");

        // then
        const [enabled, , publish] = mocks.setScreenShareEnabled.mock.calls[0];
        expect(enabled).toBe(false);
        expect(publish).toMatchObject({ screenShareEncoding: { maxBitrate: 6_000_000 } });
    });
});
