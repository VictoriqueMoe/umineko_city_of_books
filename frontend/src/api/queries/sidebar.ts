import { useQuery } from "@tanstack/react-query";
import { getHomeActivity, getSidebarActivity, getSidebarLastVisited } from "../endpoints";
import { useAuth } from "../../hooks/useAuth";
import { queryKeys } from "../queryKeys";

export function useHomeActivity() {
    const q = useQuery({ queryKey: queryKeys.sidebar.homeActivity(), queryFn: () => getHomeActivity() });
    return { data: q.data ?? null, loading: q.isLoading };
}

export function useSidebarActivity() {
    const q = useQuery({ queryKey: queryKeys.sidebar.activity(), queryFn: () => getSidebarActivity() });
    return { data: q.data ?? null, loading: q.isLoading };
}

export function useSidebarLastVisited() {
    const { user } = useAuth();
    const q = useQuery({
        queryKey: queryKeys.sidebar.lastVisited(),
        queryFn: () => getSidebarLastVisited(),
        enabled: !!user,
    });
    return { data: q.data ?? null, loading: q.isLoading, refresh: q.refetch };
}
