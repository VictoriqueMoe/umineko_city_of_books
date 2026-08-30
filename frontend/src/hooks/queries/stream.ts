import { useQuery } from "@tanstack/react-query";
import { getMyStream, getStream, getStreamCredentials, listLiveStreams } from "../../api/endpoints/stream";
import { queryKeys } from "../../api/queryKeys";
import { errorMessage } from "../../utils/errorMessage";
import type { LiveStream, StreamCredentials, StreamOwner } from "../../types/api";

const SECRET_CACHE = {
    staleTime: 0,
    gcTime: 0,
} as const;

export function useLiveStreams() {
    const query = useQuery({
        queryKey: queryKeys.streams.live(),
        queryFn: listLiveStreams,
    });

    return {
        streams: query.data?.streams ?? [],
        enabled: query.data?.enabled ?? false,
        loading: query.isLoading,
    };
}

export function useLiveStreamsCount() {
    const { streams } = useLiveStreams();

    return { count: streams.length };
}

interface UseStreamResult {
    stream: LiveStream | null;
    loading: boolean;
    error: string;
}

export function useStream(streamId: string | undefined): UseStreamResult {
    const query = useQuery({
        queryKey: queryKeys.streams.detail(streamId),
        queryFn: () => getStream(streamId as string),
        enabled: !!streamId,
    });

    return {
        stream: query.data ?? null,
        loading: query.isLoading,
        error: query.error ? errorMessage(query.error, "Could not load the stream.") : "",
    };
}

interface UseMyStreamResult {
    owner: StreamOwner | null;
    loading: boolean;
    error: string;
}

export function useMyStream(enabled = true): UseMyStreamResult {
    const query = useQuery({
        queryKey: queryKeys.streams.mine(),
        queryFn: getMyStream,
        enabled,
        ...SECRET_CACHE,
    });

    return {
        owner: query.data ?? null,
        loading: query.isLoading,
        error: query.error ? errorMessage(query.error, "Could not load your stream.") : "",
    };
}

interface UseStreamCredentialsResult {
    credentials: StreamCredentials | null;
    loading: boolean;
    error: string;
}

export function useStreamCredentials(enabled = true): UseStreamCredentialsResult {
    const query = useQuery({
        queryKey: queryKeys.streams.credentials(),
        queryFn: getStreamCredentials,
        enabled,
        ...SECRET_CACHE,
    });

    return {
        credentials: query.data ?? null,
        loading: query.isLoading,
        error: query.error ? errorMessage(query.error, "Could not load your stream key.") : "",
    };
}
