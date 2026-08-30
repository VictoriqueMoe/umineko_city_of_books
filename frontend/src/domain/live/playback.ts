import type { StreamDefaultMode } from "../../types/api";

export interface PlaybackStream {
    userId: string;
    status: string;
    defaultMode: StreamDefaultMode;
    hlsUrl?: string;
}

export interface PlaybackPlanInput {
    stream: PlaybackStream | null | undefined;
    viewerId: string | null | undefined;
    modeOverride: StreamDefaultMode | null | undefined;
    showOwnPreview: boolean;
}

export interface PlaybackPlan {
    isLive: boolean;
    mode: StreamDefaultMode;
    isOwnStream: boolean;
    showsPlayback: boolean;
    wantsRoom: boolean;
    wantsMedia: boolean;
    autoSubscribe: boolean;
    monitorsWithoutMedia: boolean;
}

export const LIVE_STATUS = "live";

export function resolvePlaybackMode(
    stream: PlaybackStream | null | undefined,
    modeOverride: StreamDefaultMode | null | undefined,
): StreamDefaultMode {
    return modeOverride ?? (stream?.defaultMode === "hls" && stream?.hlsUrl ? "hls" : "webrtc");
}

export function isOwnStream(stream: PlaybackStream | null | undefined, viewerId: string | null | undefined): boolean {
    return !!viewerId && !!stream && viewerId === stream.userId;
}

export function resolvePlaybackPlan(input: PlaybackPlanInput): PlaybackPlan {
    const { stream, viewerId, modeOverride, showOwnPreview } = input;

    const live = stream?.status === LIVE_STATUS;
    const mode = resolvePlaybackMode(stream, modeOverride);
    const own = isOwnStream(stream, viewerId);

    const showsPlayback = !own || showOwnPreview;
    const wantsMedia = showsPlayback && mode === "webrtc";
    const wantsRoom = wantsMedia || !showsPlayback;

    return {
        isLive: live,
        mode,
        isOwnStream: own,
        showsPlayback,
        wantsRoom,
        wantsMedia,
        autoSubscribe: wantsMedia,
        monitorsWithoutMedia: wantsRoom && !wantsMedia,
    };
}

export function shouldConnectRoom(streamId: string | null | undefined, plan: PlaybackPlan): boolean {
    return !!streamId && plan.isLive && plan.wantsRoom;
}
