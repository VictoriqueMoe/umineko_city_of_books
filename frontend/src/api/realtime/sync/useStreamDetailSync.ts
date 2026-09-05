import { useQueryClient } from "@tanstack/react-query";
import type { LiveStream } from "../../../types/api";
import { REALTIME_EVENTS } from "../events";
import { useRealtimeEvent } from "../useRealtime";

export interface StreamDetailSyncTarget {
    streamId: string | undefined;
    username?: string;
    queryKey: readonly unknown[];
}

function sameUsername(a: string | undefined, b: string | undefined): boolean {
    return !!a && !!b && a.toLowerCase() === b.toLowerCase();
}

export function useStreamDetailSync(target: StreamDetailSyncTarget): void {
    const queryClient = useQueryClient();
    const { streamId, username, queryKey } = target;

    useRealtimeEvent(REALTIME_EVENTS.STREAM_LIVE, event => {
        const isThisStream =
            (!!streamId && event.data.id === streamId) || sameUsername(username, event.data.streamerUsername);
        if (!isThisStream) {
            return;
        }

        queryClient.invalidateQueries({ queryKey });
    });

    useRealtimeEvent(REALTIME_EVENTS.STREAM_OFFLINE, event => {
        if (!streamId || event.data.streamId !== streamId) {
            return;
        }

        queryClient.invalidateQueries({ queryKey });
    });

    useRealtimeEvent(REALTIME_EVENTS.STREAM_TITLE, event => {
        const { title } = event.data;

        if (!streamId || event.data.streamId !== streamId) {
            return;
        }

        queryClient.setQueryData<LiveStream>(queryKey, prev => (prev ? { ...prev, title } : prev));
    });
}
