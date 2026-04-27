import { useQuery } from "@tanstack/react-query";
import {
    getBlockedUsers,
    getFollowStats,
    getOnlineStatus,
    getUserActivity,
    getUserArt,
    getUserFanficFavourites,
    getUserFanfics,
    getUserFollowedJournals,
    getUserGalleries,
    getUserJournals,
    getUserMysteries,
    getUserPosts,
    getUserRooms,
    getUserShips,
} from "../endpoints";
import { queryKeys } from "../queryKeys";

const userKey = (userID: string, kind: string, extra: Record<string, unknown> = {}) =>
    ["user", userID, kind, extra] as const;

export function useUserPosts(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: userKey(userID, "posts", { limit, offset }),
        queryFn: () => getUserPosts(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        posts: query.data?.posts ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
        refresh: query.refetch,
    };
}

export function useUserArt(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: userKey(userID, "art", { limit, offset }),
        queryFn: () => getUserArt(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        art: query.data?.art ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
    };
}

export function useUserGalleries(userID: string) {
    const query = useQuery({
        queryKey: userKey(userID, "galleries"),
        queryFn: () => getUserGalleries(userID),
        enabled: !!userID,
    });
    return { galleries: query.data ?? [], loading: query.isPending, refresh: query.refetch };
}

export function useUserShips(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: userKey(userID, "ships", { limit, offset }),
        queryFn: () => getUserShips(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        ships: query.data?.ships ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
    };
}

export function useUserMysteries(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: userKey(userID, "mysteries", { limit, offset }),
        queryFn: () => getUserMysteries(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        mysteries: query.data?.mysteries ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
    };
}

export function useUserFanfics(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: userKey(userID, "fanfics", { limit, offset }),
        queryFn: () => getUserFanfics(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        fanfics: query.data?.fanfics ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
    };
}

export function useUserFanficFavourites(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: userKey(userID, "fanfic-favourites", { limit, offset }),
        queryFn: () => getUserFanficFavourites(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        fanfics: query.data?.fanfics ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
    };
}

export function useUserJournals(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: userKey(userID, "journals", { limit, offset }),
        queryFn: () => getUserJournals(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        journals: query.data?.journals ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
    };
}

export function useUserFollowedJournals(userID: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: userKey(userID, "followed-journals", { limit, offset }),
        queryFn: () => getUserFollowedJournals(userID, limit, offset),
        enabled: !!userID,
    });
    return {
        journals: query.data?.journals ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
    };
}

export function useBlockedUsers(userID: string) {
    const query = useQuery({
        queryKey: queryKeys.profile.blockedUsers(userID),
        queryFn: () => getBlockedUsers(),
        enabled: !!userID,
    });
    return { blocked: query.data?.users ?? [], loading: query.isPending, refresh: query.refetch };
}

export function useFollowStats(userID: string) {
    const query = useQuery({
        queryKey: userKey(userID, "follow-stats"),
        queryFn: () => getFollowStats(userID),
        enabled: !!userID,
    });
    return { stats: query.data ?? null, loading: query.isPending, refresh: query.refetch };
}

export function useUserActivity(username: string, limit = 20, offset = 0) {
    const query = useQuery({
        queryKey: userKey(username, "activity", { limit, offset }),
        queryFn: () => getUserActivity(username, limit, offset),
        enabled: !!username,
    });
    return {
        activity: query.data?.items ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
    };
}

export function useOnlineStatus(userIDs: string[]) {
    const query = useQuery({
        queryKey: ["online-status", userIDs.slice().sort()],
        queryFn: () => getOnlineStatus(userIDs),
        enabled: userIDs.length > 0,
        refetchInterval: 60_000,
    });
    return { statuses: query.data ?? {} };
}

export function useUserRooms() {
    const query = useQuery({
        queryKey: ["user", "chat-rooms"],
        queryFn: () => getUserRooms(),
    });
    return { rooms: query.data?.rooms ?? [], loading: query.isPending };
}
