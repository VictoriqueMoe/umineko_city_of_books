import { useQuery } from "@tanstack/react-query";
import { getRules, getSiteInfo, getStaff } from "../../api/endpoints/site";
import { queryKeys } from "../../api/queryKeys";

export function useSiteInfoQuery() {
    const query = useQuery({
        queryKey: queryKeys.auth.siteInfo(),
        queryFn: () => getSiteInfo(),
    });
    return {
        siteInfo: query.data ?? null,
        loading: query.isLoading,
        refresh: query.refetch,
        dataUpdatedAt: query.dataUpdatedAt,
    };
}

export function useStaff() {
    const query = useQuery({
        queryKey: queryKeys.auth.staff(),
        queryFn: () => getStaff(),
    });
    return { staff: query.data ?? [], loading: query.isLoading };
}

export function useRules(page: string) {
    const q = useQuery({
        queryKey: queryKeys.rules(page),
        queryFn: () => getRules(page),
        enabled: !!page,
    });
    return { rules: q.data?.rules ?? "", loading: q.isLoading };
}
