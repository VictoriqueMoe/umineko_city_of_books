import { useQuery } from "@tanstack/react-query";
import {
    getArtCornerCounts,
    getBlockStatus,
    getCornerCounts,
    getFollowers,
    getFollowing,
    getFollowStats,
    getMutualFollowers,
    getPopularTags,
    getRules,
    listUsersPublic,
    searchUsers,
} from "../endpoints";
import { queryClient } from "../queryClient";
import { queryKeys } from "../queryKeys";

export function fetchMutualFollowers() {
    return queryClient.fetchQuery({
        queryKey: queryKeys.users.mutuals(),
        queryFn: () => getMutualFollowers(),
    });
}

export function fetchSearchUsers(query: string) {
    return queryClient.fetchQuery({
        queryKey: queryKeys.users.search(query),
        queryFn: () => searchUsers(query),
    });
}

export function useSearchUsers(query: string, enabled = true) {
    const q = useQuery({
        queryKey: queryKeys.users.search(query),
        queryFn: () => searchUsers(query),
        enabled: enabled && !!query,
    });
    return { users: q.data ?? [], loading: q.isLoading };
}

export function useMutualFollowers(enabled = true) {
    const q = useQuery({
        queryKey: queryKeys.users.mutuals(),
        queryFn: () => getMutualFollowers(),
        enabled,
    });
    return { mutuals: q.data ?? [], loading: q.isLoading };
}

export function useCornerCounts() {
    const q = useQuery({ queryKey: queryKeys.post.cornerCounts(), queryFn: () => getCornerCounts() });
    return { counts: q.data ?? {}, loading: q.isLoading };
}

export function useArtCornerCounts() {
    const q = useQuery({ queryKey: queryKeys.art.cornerCounts(), queryFn: () => getArtCornerCounts() });
    return { counts: q.data ?? {}, loading: q.isLoading };
}

export function usePopularTags(corner?: string) {
    const q = useQuery({
        queryKey: queryKeys.art.popularTags(corner ?? ""),
        queryFn: () => getPopularTags(corner),
    });
    return { tags: q.data ?? [], loading: q.isLoading };
}

export function useFollowStats(userId: string) {
    const q = useQuery({
        queryKey: queryKeys.followStats(userId),
        queryFn: () => getFollowStats(userId),
        enabled: !!userId,
    });
    return { stats: q.data ?? null, loading: q.isLoading, refresh: q.refetch };
}

export function useFollowers(userId: string, limit = 50, offset = 0) {
    const q = useQuery({
        queryKey: queryKeys.users.followers(userId, { limit, offset }),
        queryFn: () => getFollowers(userId, limit, offset),
        enabled: !!userId,
    });
    return {
        users: q.data?.users ?? [],
        total: q.data?.total ?? 0,
        loading: q.isLoading,
    };
}

export function useFollowing(userId: string, limit = 50, offset = 0) {
    const q = useQuery({
        queryKey: queryKeys.users.following(userId, { limit, offset }),
        queryFn: () => getFollowing(userId, limit, offset),
        enabled: !!userId,
    });
    return {
        users: q.data?.users ?? [],
        total: q.data?.total ?? 0,
        loading: q.isLoading,
    };
}

export function useUsersPublic() {
    const q = useQuery({ queryKey: queryKeys.users.publicList(), queryFn: () => listUsersPublic() });
    return { users: q.data ?? [], loading: q.isLoading };
}

export function useBlockStatus(userId: string) {
    const q = useQuery({
        queryKey: queryKeys.blockStatus(userId),
        queryFn: () => getBlockStatus(userId),
        enabled: !!userId,
    });
    return {
        status: q.data ?? { blocking: false, blocked_by: false },
        loading: q.isLoading,
        refresh: q.refetch,
    };
}

export function useRules(page: string) {
    const q = useQuery({
        queryKey: queryKeys.rules(page),
        queryFn: () => getRules(page),
        enabled: !!page,
    });
    return { rules: q.data?.rules ?? "", loading: q.isLoading };
}
