import { useQuery } from "@tanstack/react-query";
import { getMe, getSiteInfo, getStaff } from "../endpoints";

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
    return { siteInfo: query.data ?? null, loading: query.isLoading, refresh: query.refetch };
}

export function useStaff() {
    const query = useQuery({
        queryKey: ["staff"],
        queryFn: () => getStaff(),
    });
    return { staff: query.data ?? [], loading: query.isLoading };
}
