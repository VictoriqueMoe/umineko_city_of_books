import { useQuery } from "@tanstack/react-query";
import { getMe, getSiteInfo } from "../endpoints";

export function useMe() {
    const query = useQuery({
        queryKey: ["auth", "me"],
        queryFn: () => getMe(),
    });
    return { me: query.data ?? null, loading: query.isPending, refresh: query.refetch };
}

export function useSiteInfoQuery() {
    const query = useQuery({
        queryKey: ["site-info"],
        queryFn: () => getSiteInfo(),
    });
    return { siteInfo: query.data ?? null, loading: query.isPending, refresh: query.refetch };
}
