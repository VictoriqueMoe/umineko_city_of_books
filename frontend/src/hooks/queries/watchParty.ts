import { useQuery } from "@tanstack/react-query";
import { listWatchParties } from "../../api/endpoints/watchParty";
import { queryKeys } from "../../api/queryKeys";
import { errorMessage } from "../../utils/errorMessage";
import type { WatchPartySession } from "../../types/api";

interface UseWatchPartiesResult {
    sessions: WatchPartySession[];
    enabled: boolean;
    screenShareEnabled: boolean;
    loading: boolean;
    loaded: boolean;
    error: string;
    refresh: () => Promise<void>;
}

export function useWatchParties(roomId: string | null | undefined): UseWatchPartiesResult {
    const query = useQuery({
        queryKey: queryKeys.watchParty.list(roomId),
        queryFn: () => listWatchParties(roomId as string),
        enabled: !!roomId,
    });

    return {
        sessions: query.data?.sessions ?? [],
        enabled: query.data?.enabled ?? false,
        screenShareEnabled: query.data?.screen_share_enabled ?? false,
        loading: query.isLoading,
        loaded: !!roomId && query.isSuccess,
        error: query.error ? errorMessage(query.error, "Failed to load watch parties") : "",
        refresh: async () => {
            await query.refetch();
        },
    };
}
