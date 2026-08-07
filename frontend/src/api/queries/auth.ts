import { useQuery } from "@tanstack/react-query";
import { getChatbotOptIn, getMe, getSiteInfo, getStaff } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useMe() {
    const query = useQuery({
        queryKey: ["auth", "me"],
        queryFn: () => getMe(),
    });
    return { me: query.data ?? null, loading: query.isLoading, refresh: query.refetch };
}

export function useSiteInfoQuery() {
    const query = useQuery({
        queryKey: ["site-info"],
        queryFn: () => getSiteInfo(),
    });
    return {
        siteInfo: query.data ?? null,
        loading: query.isLoading,
        refresh: query.refetch,
        dataUpdatedAt: query.dataUpdatedAt,
    };
}

export function useChatbotOptIn(enabled: boolean) {
    const query = useQuery({
        queryKey: queryKeys.preferences.chatbotOptIn(),
        queryFn: () => getChatbotOptIn(),
        enabled,
    });
    return { optedIn: query.data?.opted_in ?? false, loading: enabled && query.isLoading };
}

export function useStaff() {
    const query = useQuery({
        queryKey: ["staff"],
        queryFn: () => getStaff(),
    });
    return { staff: query.data ?? [], loading: query.isLoading };
}
