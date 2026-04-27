import { useQuery } from "@tanstack/react-query";
import { getGMLeaderboard, getMystery, getMysteryLeaderboard, listMysteries } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useMysteryList(params: { sort?: string; solved?: string; limit?: number; offset?: number }) {
    const q = useQuery({
        queryKey: ["mysteries", "list", params],
        queryFn: () => listMysteries(params),
    });
    return {
        mysteries: q.data?.mysteries ?? [],
        total: q.data?.total ?? 0,
        loading: q.isPending,
        refresh: q.refetch,
    };
}

export function useMystery(id: string) {
    const q = useQuery({
        queryKey: queryKeys.mystery.detail(id),
        queryFn: () => getMystery(id),
        enabled: !!id,
    });
    return { mystery: q.data ?? null, loading: q.isPending, refresh: q.refetch };
}

export function useMysteryLeaderboard(limit?: number) {
    const q = useQuery({
        queryKey: ["mysteries", "leaderboard", limit ?? null],
        queryFn: () => getMysteryLeaderboard(limit),
    });
    return { entries: q.data?.entries ?? [], loading: q.isPending };
}

export function useGMLeaderboard(limit?: number) {
    const q = useQuery({
        queryKey: ["mysteries", "gm-leaderboard", limit ?? null],
        queryFn: () => getGMLeaderboard(limit),
    });
    return { entries: q.data?.entries ?? [], loading: q.isPending };
}
