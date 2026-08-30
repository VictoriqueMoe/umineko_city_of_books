import type { LiveStream } from "../types/api";
import { useLiveStreams } from "./queries/stream";

export interface UseLiveDirectoryResult {
    streams: LiveStream[];
    enabled: boolean;
    loading: boolean;
}

export function useLiveDirectory(): UseLiveDirectoryResult {
    const { streams, enabled, loading } = useLiveStreams();

    return { streams, enabled, loading };
}
