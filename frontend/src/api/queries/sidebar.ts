import { useQuery } from "@tanstack/react-query";
import { getHomeActivity, getSidebarActivity, getSidebarLastVisited } from "../endpoints";

export function useHomeActivity() {
    const q = useQuery({ queryKey: ["home", "activity"], queryFn: () => getHomeActivity() });
    return { data: q.data ?? null, loading: q.isPending };
}

export function useSidebarActivity() {
    const q = useQuery({ queryKey: ["sidebar", "activity"], queryFn: () => getSidebarActivity() });
    return { data: q.data ?? null, loading: q.isPending };
}

export function useSidebarLastVisited() {
    const q = useQuery({ queryKey: ["sidebar", "last-visited"], queryFn: () => getSidebarLastVisited() });
    return { data: q.data ?? null, loading: q.isPending, refresh: q.refetch };
}
