import { useIsFetching, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { queryKeys } from "../api/queryKeys";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";
import {
    BITRATE_STORAGE_KEY,
    DEFAULT_CALCULATOR_FPS,
    DEFAULT_RESOLUTION_INDEX,
    isBitrateValid,
    parseBitrate,
    recommendedBitrateForResolution,
    type BitrateRecommendation,
} from "../domain/live/bitrate";
import { LIVE_STATUS } from "../domain/live/playback";
import type { LiveStream, StreamCredentials, StreamDefaultMode, StreamOwner } from "../types/api";
import { errorMessage } from "../utils/errorMessage";
import { nonFatal } from "../utils/nonFatal";
import { useResetStreamCredentials, useStartStream, useStopStream, useUpdateStreamTitle } from "./mutations/stream";
import { useMyStream, useStreamCredentials } from "./queries/stream";

const RESET_CONFIRMATION = "Reset your stream key? You'll need to paste the new key into OBS before your next stream.";

export interface GoLiveCalculator {
    resolutionIndex: number;
    setResolutionIndex: (index: number) => void;
    fps: number;
    setFps: (fps: number) => void;
    recommendation: BitrateRecommendation;
    applyTypical: () => void;
}

export interface GoLiveTitleEdit {
    editing: boolean;
    draft: string;
    setDraft: (draft: string) => void;
    open: () => void;
    cancel: () => void;
    save: () => void;
    saving: boolean;
    canSave: boolean;
}

export interface UseGoLiveResult {
    owner: StreamOwner | null;
    ownerError: string;
    credentials: StreamCredentials | null;
    credentialsError: string;
    smoothAvailable: boolean;
    title: string;
    setTitle: (title: string) => void;
    defaultMode: StreamDefaultMode;
    setDefaultMode: (mode: StreamDefaultMode) => void;
    bitrate: string;
    setBitrate: (bitrate: string) => void;
    bitrateValid: boolean;
    calculator: GoLiveCalculator;
    titleEdit: GoLiveTitleEdit;
    start: () => void;
    stop: () => void;
    resetCredentials: () => void;
    canStart: boolean;
    busy: boolean;
    starting: boolean;
    stopping: boolean;
    resetting: boolean;
    error: string;
    copied: string | null;
    copy: (label: string, value: string) => void;
}

export function useGoLive(): UseGoLiveResult {
    const queryClient = useQueryClient();

    const { owner, error: ownerError } = useMyStream();
    const { credentials, error: credentialsError } = useStreamCredentials();

    const startStream = useStartStream();
    const stopStream = useStopStream();
    const updateTitle = useUpdateStreamTitle();
    const resetStreamCredentials = useResetStreamCredentials();

    const [title, setTitle] = useState("");
    const [defaultMode, setDefaultMode] = useState<StreamDefaultMode>("webrtc");
    const [resolutionIndex, setResolutionIndex] = useState(DEFAULT_RESOLUTION_INDEX);
    const [fps, setFps] = useState(DEFAULT_CALCULATOR_FPS);
    const [bitrate, setBitrate] = useState(() => localStorage.getItem(BITRATE_STORAGE_KEY) ?? "");
    const [error, setError] = useState("");
    const [copied, setCopied] = useState<string | null>(null);
    const [editingTitle, setEditingTitle] = useState(false);
    const [titleDraft, setTitleDraft] = useState("");

    const refreshingOwner = useIsFetching({ queryKey: queryKeys.streams.mine() }) > 0;
    const refreshingCredentials = useIsFetching({ queryKey: queryKeys.streams.credentials() }) > 0;

    const ownerStreamId = owner?.stream.id;
    const smoothAvailable = credentials?.hlsEnabled === true;
    const bitrateValid = isBitrateValid(bitrate);
    const starting = startStream.isPending;
    const stopping = stopStream.isPending;
    const busy = starting || stopping || refreshingOwner;
    const savingTitle = updateTitle.isPending;
    const trimmedDraft = titleDraft.trim();

    function patchOwnStream(patch: (stream: LiveStream) => LiveStream): void {
        queryClient.setQueryData<StreamOwner | null>(queryKeys.streams.mine(), prev =>
            prev ? { ...prev, stream: patch(prev.stream) } : prev,
        );
    }

    useRealtimeEvent(REALTIME_EVENTS.STREAM_LIVE, event => {
        if (event.data.id !== ownerStreamId) {
            return;
        }

        patchOwnStream(stream => ({ ...stream, status: LIVE_STATUS }));
    });

    useRealtimeEvent(REALTIME_EVENTS.STREAM_TITLE, event => {
        if (event.data.streamId !== ownerStreamId) {
            return;
        }

        patchOwnStream(stream => ({ ...stream, title: event.data.title }));
    });

    useRealtimeEvent(REALTIME_EVENTS.STREAM_OFFLINE, event => {
        if (event.data.streamId !== ownerStreamId) {
            return;
        }

        queryClient.setQueryData<StreamOwner | null>(queryKeys.streams.mine(), null);
        setTitle("");
    });

    async function runStart(): Promise<void> {
        const trimmed = title.trim();

        if (!trimmed) {
            return;
        }

        if (smoothAvailable && !bitrateValid) {
            return;
        }

        setError("");

        try {
            const kbps = smoothAvailable ? Math.round(parseBitrate(bitrate)) : 0;
            const mode: StreamDefaultMode = smoothAvailable ? defaultMode : "webrtc";

            if (smoothAvailable) {
                localStorage.setItem(BITRATE_STORAGE_KEY, String(kbps));
            }

            await startStream.mutateAsync({ title: trimmed, defaultMode: mode, bitrate: kbps });
            await queryClient.invalidateQueries({ queryKey: queryKeys.streams.credentials() });
        } catch (err) {
            setError(errorMessage(err, "Could not start the stream."));
        }
    }

    async function runStop(): Promise<void> {
        if (!owner) {
            return;
        }

        setError("");

        try {
            await stopStream.mutateAsync(owner.stream.id);
            setTitle("");
        } catch (err) {
            setError(errorMessage(err, "Could not stop the stream."));
        }
    }

    async function runSaveTitle(): Promise<void> {
        if (!owner) {
            return;
        }

        if (!trimmedDraft || trimmedDraft === owner.stream.title) {
            setEditingTitle(false);
            return;
        }

        setError("");

        try {
            await updateTitle.mutateAsync({ streamId: owner.stream.id, title: trimmedDraft });
            setEditingTitle(false);
        } catch (err) {
            setError(errorMessage(err, "Could not update the title."));
        }
    }

    async function runResetCredentials(): Promise<void> {
        if (owner) {
            return;
        }

        if (!window.confirm(RESET_CONFIRMATION)) {
            return;
        }

        setError("");

        try {
            await resetStreamCredentials.mutateAsync();
        } catch (err) {
            setError(errorMessage(err, "Could not reset your stream key."));
        }
    }

    const recommendation = recommendedBitrateForResolution(resolutionIndex, fps);

    return {
        owner,
        ownerError,
        credentials,
        credentialsError,
        smoothAvailable,
        title,
        setTitle,
        defaultMode,
        setDefaultMode,
        bitrate,
        setBitrate,
        bitrateValid,
        calculator: {
            resolutionIndex,
            setResolutionIndex,
            fps,
            setFps,
            recommendation,
            applyTypical: () => setBitrate(String(recommendation.typical)),
        },
        titleEdit: {
            editing: editingTitle,
            draft: titleDraft,
            setDraft: setTitleDraft,
            open: () => {
                setTitleDraft(owner?.stream.title ?? "");
                setEditingTitle(true);
            },
            cancel: () => setEditingTitle(false),
            save: () => {
                runSaveTitle().catch(nonFatal);
            },
            saving: savingTitle,
            canSave: !savingTitle && trimmedDraft !== "" && trimmedDraft !== owner?.stream.title,
        },
        start: () => {
            runStart().catch(nonFatal);
        },
        stop: () => {
            runStop().catch(nonFatal);
        },
        resetCredentials: () => {
            runResetCredentials().catch(nonFatal);
        },
        canStart: !busy && title.trim() !== "" && (!smoothAvailable || bitrateValid),
        busy,
        starting,
        stopping,
        resetting: resetStreamCredentials.isPending || refreshingCredentials,
        error,
        copied,
        copy: (label: string, value: string) => {
            navigator.clipboard
                .writeText(value)
                .then(() => setCopied(label))
                .catch(nonFatal);
        },
    };
}
