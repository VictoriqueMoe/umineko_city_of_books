import { useQuery } from "@tanstack/react-query";
import { getOC, listOCs } from "../../api/endpoints/oc";
import { listUserOCs, listUserOCSummaries } from "../../api/endpoints/user";
import { queryKeys } from "../../api/queryKeys";
import { useOwnOCSync } from "../../api/realtime/sync/useOwnOCSync";

export function useOCList(params: {
    sort?: string;
    series?: string;
    custom?: string;
    user_id?: string;
    crack?: boolean;
    limit?: number;
    offset?: number;
}) {
    const q = useQuery({
        queryKey: queryKeys.oc.feed(params),
        queryFn: () => listOCs(params),
    });
    return { ocs: q.data?.ocs ?? [], total: q.data?.total ?? 0, loading: q.isLoading };
}

export function useOC(id: string) {
    const q = useQuery({
        queryKey: queryKeys.oc.detail(id),
        queryFn: () => getOC(id),
        enabled: !!id,
    });
    return { oc: q.data ?? null, loading: q.isLoading, refresh: q.refetch };
}

export function useUserOCs(userId: string) {
    const q = useQuery({
        queryKey: queryKeys.oc.userList(userId),
        queryFn: () => listUserOCs(userId),
        enabled: !!userId,
    });
    return { ocs: q.data?.ocs ?? [], total: q.data?.total ?? 0, loading: q.isLoading };
}

export function useUserOCSummaries(userId: string, currentUserId?: string) {
    const q = useQuery({
        queryKey: queryKeys.oc.userSummaries(userId),
        queryFn: () => listUserOCSummaries(userId),
        enabled: !!userId,
    });

    useOwnOCSync(userId, currentUserId);

    return { summaries: q.data ?? [], loading: q.isLoading };
}
