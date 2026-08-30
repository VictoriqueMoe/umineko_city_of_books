import { useQuery } from "@tanstack/react-query";
import { getOverlayConnection } from "../../api/endpoints/overlay";
import { queryKeys } from "../../api/queryKeys";
import { errorMessage } from "../../utils/errorMessage";
import type { OverlayConnection } from "../../types/api";

const SECRET_CACHE = {
    staleTime: 0,
    gcTime: 0,
} as const;

interface UseOverlayConnectionResult {
    connection: OverlayConnection | null;
    loading: boolean;
    error: string;
}

export function useOverlayConnection(enabled = true): UseOverlayConnectionResult {
    const query = useQuery({
        queryKey: queryKeys.overlay.connection(),
        queryFn: getOverlayConnection,
        enabled,
        ...SECRET_CACHE,
    });

    return {
        connection: query.data ?? null,
        loading: query.isLoading,
        error: query.error ? errorMessage(query.error, "Could not load your overlay connection.") : "",
    };
}
