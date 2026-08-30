import type { Room } from "livekit-client";

export type ScreenShareMode = "gaming" | "screenshare";

export interface ScreenSharePreset {
    contentHint: "motion" | "detail";
    resolution: { width: number; height: number; frameRate: number };
    videoCodec: "vp9";
    degradationPreference: "maintain-framerate" | "maintain-resolution";
    maxBitrate: number;
}

export const SCREEN_SHARE_PRESETS: Record<ScreenShareMode, ScreenSharePreset> = {
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
};

export async function setScreenShareEnabled(room: Room, enabled: boolean, mode: ScreenShareMode): Promise<void> {
    const { AudioPresets } = await import("livekit-client");

    const preset = SCREEN_SHARE_PRESETS[mode];

    await room.localParticipant.setScreenShareEnabled(
        enabled,
        {
            audio: {
                echoCancellation: false,
                noiseSuppression: false,
                autoGainControl: false,
            },
            contentHint: preset.contentHint,
            resolution: preset.resolution,
        },
        {
            audioPreset: AudioPresets.musicHighQualityStereo,
            dtx: false,
            red: false,
            forceStereo: true,
            videoCodec: preset.videoCodec,
            degradationPreference: preset.degradationPreference,
            screenShareEncoding: {
                maxBitrate: preset.maxBitrate,
                maxFramerate: preset.resolution.frameRate,
            },
        },
    );
}
