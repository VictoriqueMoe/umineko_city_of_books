import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
    getStreamViewerToken,
    joinStreamChat,
    resetStreamCredentials,
    startStream,
    stopStream,
    updateStreamTitle,
    uploadStreamThumbnail,
} from "../../api/endpoints/stream";
import { queryKeys } from "../../api/queryKeys";
import type { LiveStream, StreamDefaultMode } from "../../types/api";

const DROP_SECRET_ON_SETTLE = {
    gcTime: 0,
} as const;

function refreshOwnerView(qc: QueryClient): void {
    qc.invalidateQueries({ queryKey: queryKeys.streams.mine() });
}

function refreshDirectory(qc: QueryClient): void {
    qc.invalidateQueries({ queryKey: queryKeys.streams.live() });
}

export function useStartStream() {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: (input: { title: string; defaultMode: StreamDefaultMode; bitrate: number }) =>
            startStream(input.title, input.defaultMode, input.bitrate),
        onSuccess: () => {
            refreshOwnerView(qc);
            refreshDirectory(qc);
        },
        ...DROP_SECRET_ON_SETTLE,
    });
}

export function useStopStream() {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: (streamId: string) => stopStream(streamId),
        onSuccess: () => {
            refreshOwnerView(qc);
            refreshDirectory(qc);
        },
    });
}

export function useUpdateStreamTitle() {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: ({ streamId, title }: { streamId: string; title: string }) => updateStreamTitle(streamId, title),
        onSuccess: stream => {
            qc.setQueryData<LiveStream>(queryKeys.streams.byUsername(stream.streamerUsername), stream);
            refreshOwnerView(qc);
            refreshDirectory(qc);
        },
    });
}

export function useResetStreamCredentials() {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: () => resetStreamCredentials(),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.streams.credentials() });
        },
        ...DROP_SECRET_ON_SETTLE,
    });
}

export function useUploadStreamThumbnail() {
    return useMutation({
        mutationFn: ({ streamId, blob }: { streamId: string; blob: Blob }) => uploadStreamThumbnail(streamId, blob),
    });
}

export function useJoinStreamChat() {
    return useMutation({
        mutationFn: (streamId: string) => joinStreamChat(streamId),
    });
}

export function useStreamViewerToken() {
    return useMutation({
        mutationFn: (streamId: string) => getStreamViewerToken(streamId),
        ...DROP_SECRET_ON_SETTLE,
    });
}
