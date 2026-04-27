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
    getShareCount,
    listUsersPublic,
    searchUsers,
} from "../endpoints";
import { queryClient } from "../queryClient";

export function fetchMutualFollowers() {
    return queryClient.fetchQuery({
        queryKey: ["users", "mutuals"],
        queryFn: () => getMutualFollowers(),
    });
}

export function fetchSearchUsers(query: string) {
    return queryClient.fetchQuery({
        queryKey: ["users", "search", query],
        queryFn: () => searchUsers(query),
    });
}

export function useSearchUsers(query: string, enabled = true) {
    const q = useQuery({
        queryKey: ["users", "search", query],
        queryFn: () => searchUsers(query),
        enabled: enabled && !!query,
    });
    return { users: q.data ?? [], loading: q.isPending };
}

export function useMutualFollowers(enabled = true) {
    const q = useQuery({
        queryKey: ["users", "mutuals"],
        queryFn: () => getMutualFollowers(),
        enabled,
    });
    return { mutuals: q.data ?? [], loading: q.isPending };
}

export function useCornerCounts() {
    const q = useQuery({ queryKey: ["posts", "corner-counts"], queryFn: () => getCornerCounts() });
    return { counts: q.data ?? {}, loading: q.isPending };
}

export function useArtCornerCounts() {
    const q = useQuery({ queryKey: ["art", "corner-counts"], queryFn: () => getArtCornerCounts() });
    return { counts: q.data ?? {}, loading: q.isPending };
}

export function usePopularTags(corner?: string) {
    const q = useQuery({
        queryKey: ["art", "popular-tags", corner ?? ""],
        queryFn: () => getPopularTags(corner),
    });
    return { tags: q.data ?? [], loading: q.isPending };
}

export function useFollowStats(userId: string) {
    const q = useQuery({
        queryKey: ["follow-stats", userId],
        queryFn: () => getFollowStats(userId),
        enabled: !!userId,
    });
    return { stats: q.data ?? null, loading: q.isPending, refresh: q.refetch };
}

export function useFollowers(userId: string, limit = 50, offset = 0) {
    const q = useQuery({
        queryKey: ["users", userId, "followers", { limit, offset }],
        queryFn: () => getFollowers(userId, limit, offset),
        enabled: !!userId,
    });
    return {
        users: q.data?.users ?? [],
        total: q.data?.total ?? 0,
        loading: q.isPending,
    };
}

export function useFollowing(userId: string, limit = 50, offset = 0) {
    const q = useQuery({
        queryKey: ["users", userId, "following", { limit, offset }],
        queryFn: () => getFollowing(userId, limit, offset),
        enabled: !!userId,
    });
    return {
        users: q.data?.users ?? [],
        total: q.data?.total ?? 0,
        loading: q.isPending,
    };
}

export function useUsersPublic() {
    const q = useQuery({ queryKey: ["users", "public"], queryFn: () => listUsersPublic() });
    return { users: q.data ?? [], loading: q.isPending };
}

export function useBlockStatus(userId: string) {
    const q = useQuery({
        queryKey: ["block-status", userId],
        queryFn: () => getBlockStatus(userId),
        enabled: !!userId,
    });
    return {
        status: q.data ?? { blocking: false, blocked_by: false },
        loading: q.isPending,
        refresh: q.refetch,
    };
}

export function useRules(page: string) {
    const q = useQuery({
        queryKey: ["rules", page],
        queryFn: () => getRules(page),
        enabled: !!page,
    });
    return { rules: q.data?.rules ?? "", loading: q.isPending };
}

export function useShareCount(contentType: string, contentId: string, enabled = true) {
    const q = useQuery({
        queryKey: ["share-count", contentType, contentId],
        queryFn: () => getShareCount(contentType, contentId),
        enabled: enabled && !!contentId,
    });
    return { count: q.data?.share_count ?? 0, loading: q.isPending };
}
