import { useQuery } from "@tanstack/react-query";
import {
    getBlockedUsers,
    getBlockStatus,
    getFollowers,
    getFollowing,
    getFollowStats,
    getMutualFollowers,
    getUserActivity,
    getUserArt,
    getUserFanficFavourites,
    getUserFanfics,
    getUserFollowedJournals,
    getUserGalleries,
    getUserJournals,
    getUserMysteries,
    getUserPosts,
    getUserShips,
    listUsersPublic,
    searchUsers,
} from "../../api/endpoints/user";
import { queryClient } from "../../api/queryClient";
import { queryKeys } from "../../api/queryKeys";

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

export function useUserPosts(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.user.posts(userID, { limit, offset }),
        queryFn: () => getUserPosts(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        posts: query.data?.posts ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
        refresh: query.refetch,
    };
}

export function useUserArt(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.user.art(userID, { limit, offset }),
        queryFn: () => getUserArt(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        art: query.data?.art ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
    };
}

export function useUserGalleries(userID: string) {
    const query = useQuery({
        queryKey: queryKeys.user.galleries(userID),
        queryFn: () => getUserGalleries(userID),
        enabled: !!userID,
    });
    return { galleries: query.data ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useUserShips(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.user.ships(userID, { limit, offset }),
        queryFn: () => getUserShips(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        ships: query.data?.ships ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
    };
}

export function useUserMysteries(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.user.mysteries(userID, { limit, offset }),
        queryFn: () => getUserMysteries(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        mysteries: query.data?.mysteries ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
    };
}

export function useUserFanfics(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.user.fanfics(userID, { limit, offset }),
        queryFn: () => getUserFanfics(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        fanfics: query.data?.fanfics ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
    };
}

export function useUserFanficFavourites(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.user.fanficFavourites(userID, { limit, offset }),
        queryFn: () => getUserFanficFavourites(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        fanfics: query.data?.fanfics ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
    };
}

export function useUserJournals(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.user.journals(userID, { limit, offset }),
        queryFn: () => getUserJournals(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        journals: query.data?.journals ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
    };
}

export function useUserFollowedJournals(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.user.followedJournals(userID, { limit, offset }),
        queryFn: () => getUserFollowedJournals(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        journals: query.data?.journals ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
    };
}

export function useBlockedUsers(userID: string) {
    const query = useQuery({
        queryKey: queryKeys.profile.blockedUsers(userID),
        queryFn: () => getBlockedUsers(),
        enabled: !!userID,
    });
    return { blocked: query.data?.users ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useUserActivity(username: string, limit = 20, offset = 0) {
    const query = useQuery({
        queryKey: queryKeys.user.activity(username, { limit, offset }),
        queryFn: () => getUserActivity(username, limit, offset),
        enabled: !!username,
    });
    return {
        activity: query.data?.items ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
    };
}
