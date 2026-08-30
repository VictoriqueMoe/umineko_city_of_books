import { useQueryClient } from "@tanstack/react-query";
import type { LiveStream } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { REALTIME_EVENTS } from "../events";
import { useRealtimeEvent } from "../useRealtime";

export function useStreamDetailSync(streamId: string | undefined): void {
    const queryClient = useQueryClient();

    useRealtimeEvent(REALTIME_EVENTS.STREAM_LIVE, event => {
        if (!streamId || event.data.id !== streamId) {
            return;
        }

        queryClient.invalidateQueries({ queryKey: queryKeys.streams.detail(streamId) });
    });

    useRealtimeEvent(REALTIME_EVENTS.STREAM_OFFLINE, event => {
        if (!streamId || event.data.streamId !== streamId) {
            return;
        }

        queryClient.invalidateQueries({ queryKey: queryKeys.streams.detail(streamId) });
    });

    useRealtimeEvent(REALTIME_EVENTS.STREAM_TITLE, event => {
        const { title } = event.data;

        if (!streamId || event.data.streamId !== streamId) {
            return;
        }

        queryClient.setQueryData<LiveStream>(queryKeys.streams.detail(streamId), prev =>
            prev ? { ...prev, title } : prev,
        );
    });
}
