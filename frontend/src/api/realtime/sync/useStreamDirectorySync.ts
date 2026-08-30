import { useQueryClient } from "@tanstack/react-query";
import { applyTitle, applyViewerCount } from "../../../domain/live/directory";
import type { LiveStreamListResponse } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { REALTIME_EVENTS } from "../events";
import { useRealtimeEvent } from "../useRealtime";

const DIRECTORY_PRESENCE_EVENTS = [REALTIME_EVENTS.STREAM_LIVE, REALTIME_EVENTS.STREAM_OFFLINE] as const;

export function useStreamDirectorySync(): void {
    const queryClient = useQueryClient();

    useRealtimeEvent(DIRECTORY_PRESENCE_EVENTS, () => {
        queryClient.invalidateQueries({ queryKey: queryKeys.streams.live() });
    });

    useRealtimeEvent(REALTIME_EVENTS.STREAM_VIEWERS, event => {
        const { streamId, viewerCount } = event.data;

        queryClient.setQueryData<LiveStreamListResponse>(queryKeys.streams.live(), prev =>
            applyViewerCount(prev, streamId, viewerCount),
        );
    });

    useRealtimeEvent(REALTIME_EVENTS.STREAM_TITLE, event => {
        const { streamId, title } = event.data;

        queryClient.setQueryData<LiveStreamListResponse>(queryKeys.streams.live(), prev =>
            applyTitle(prev, streamId, title),
        );
    });
}
