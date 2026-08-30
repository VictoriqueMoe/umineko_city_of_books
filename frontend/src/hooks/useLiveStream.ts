import { useState } from "react";
import type { Room } from "livekit-client";
import { useStreamDetailSync } from "../api/realtime/sync/useStreamDetailSync";
import { resolvePlaybackPlan, type PlaybackPlan } from "../domain/live/playback";
import type { LiveStream, StreamDefaultMode } from "../types/api";
import { useAuth } from "./useAuth";
import { useLiveKitRoom } from "./useLiveKitRoom";
import { useStreamViewerToken } from "./mutations/stream";
import { useStream } from "./queries/stream";

export interface UseStreamDetailResult {
    stream: LiveStream | null;
    loading: boolean;
    error: string;
}

export interface UseLiveStreamResult {
    stream: LiveStream | null;
    loading: boolean;
    plan: PlaybackPlan;
    room: Room | null;
    error: string | null;
    showOwnPreview: boolean;
    setShowOwnPreview: (show: boolean) => void;
    setMode: (mode: StreamDefaultMode) => void;
}

export function useStreamDetail(streamId: string | undefined): UseStreamDetailResult {
    const detail = useStream(streamId);

    useStreamDetailSync(streamId);

    return detail;
}

export function useLiveStream(streamId: string | undefined): UseLiveStreamResult {
    const { user } = useAuth();
    const { stream, loading } = useStreamDetail(streamId);

    const [modeOverride, setModeOverride] = useState<StreamDefaultMode | null>(null);
    const [showOwnPreview, setShowOwnPreview] = useState(false);

    const plan = resolvePlaybackPlan({
        stream,
        viewerId: user?.id,
        modeOverride,
        showOwnPreview,
    });

    const viewerToken = useStreamViewerToken();

    const { room, error } = useLiveKitRoom({
        streamId,
        isLive: plan.isLive,
        wantsRoom: plan.wantsRoom,
        wantsMedia: plan.wantsMedia,
        requestToken: viewerToken.mutateAsync,
    });

    return {
        stream,
        loading,
        plan,
        room,
        error,
        showOwnPreview,
        setShowOwnPreview,
        setMode: setModeOverride,
    };
}
