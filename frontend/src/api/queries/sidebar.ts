import { useQuery } from "@tanstack/react-query";
import { getHomeActivity, getSidebarActivity, getSidebarLastVisited } from "../endpoints";
import { useAuth } from "../../hooks/useAuth";

export function useHomeActivity() {
    const q = useQuery({ queryKey: ["home", "activity"], queryFn: () => getHomeActivity() });
    return { data: q.data ?? null, loading: q.isLoading };
}

export function useSidebarActivity() {
    const q = useQuery({ queryKey: ["sidebar", "activity"], queryFn: () => getSidebarActivity() });
    return { data: q.data ?? null, loading: q.isLoading };
}

export function useSidebarLastVisited() {
    const { user } = useAuth();
    const q = useQuery({
        queryKey: ["sidebar", "last-visited"],
        queryFn: () => getSidebarLastVisited(),
        enabled: !!user,
    });
    return { data: q.data ?? null, loading: q.isLoading, refresh: q.refetch };
}
