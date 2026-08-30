import { apiDelete, apiFetch, apiPost, buildQueryString } from "../client";
import type {
    ActivityListResponse,
    ArtListResponse,
    BlockedUserItem,
    BlockStatus,
    FanficListResponse,
    FollowStats,
    Gallery,
    JournalListResponse,
    MysteryListResponse,
    OCListResponse,
    OCSummary,
    PostListResponse,
    PublicUser,
    ShipListResponse,
    User,
    UserProfile,
} from "../../types/api";

export async function getUserProfile(username: string): Promise<UserProfile> {
    return apiFetch<UserProfile>(`/users/${encodeURIComponent(username)}`);
}

export async function getUserActivity(
    username: string,
    limit?: number,
    offset?: number,
): Promise<ActivityListResponse> {
    const qs = buildQueryString({ limit: limit ?? 20, offset });
    return apiFetch<ActivityListResponse>(`/users/${username}/activity${qs}`);
}

export async function searchUsers(query: string): Promise<User[]> {
    return apiFetch<User[]>(`/users/search?q=${encodeURIComponent(query)}`);
}

export async function resolveUsernames(usernames: string[]): Promise<{ usernames: string[] }> {
    return apiFetch<{ usernames: string[] }>(`/users/resolve?usernames=${encodeURIComponent(usernames.join(","))}`);
}

export async function getMutualFollowers(): Promise<User[]> {
    return apiFetch<User[]>("/users/mutuals");
}

export async function getUserPosts(userId: string, limit: number = 20, offset: number = 0): Promise<PostListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<PostListResponse>(`/users/${userId}/posts${qs}`);
}

export async function followUser(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/users/${id}/follow`, undefined);
}

export async function unfollowUser(id: string): Promise<void> {
    await apiDelete(`/users/${id}/follow`);
}

export async function getFollowStats(id: string): Promise<FollowStats> {
    return apiFetch<FollowStats>(`/users/${id}/follow-stats`);
}

export async function getFollowers(
    id: string,
    limit: number = 50,
    offset: number = 0,
): Promise<{ users: User[]; total: number }> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<{ users: User[]; total: number }>(`/users/${id}/followers${qs}`);
}

export async function getFollowing(
    id: string,
    limit: number = 50,
    offset: number = 0,
): Promise<{ users: User[]; total: number }> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<{ users: User[]; total: number }>(`/users/${id}/following${qs}`);
}

export async function listUsersPublic(): Promise<PublicUser[]> {
    return apiFetch<PublicUser[]>("/users");
}

export async function getUserGalleries(userId: string): Promise<Gallery[]> {
    return apiFetch<Gallery[]>(`/users/${userId}/galleries`);
}

export async function getUserArt(userId: string, limit: number = 24, offset: number = 0): Promise<ArtListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<ArtListResponse>(`/users/${userId}/art${qs}`);
}

export async function blockUser(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/users/${id}/block`, undefined);
}

export async function unblockUser(id: string): Promise<void> {
    await apiDelete(`/users/${id}/block`);
}

export async function getBlockStatus(id: string): Promise<BlockStatus> {
    return apiFetch<BlockStatus>(`/users/${id}/block-status`);
}

export async function getBlockedUsers(): Promise<{ users: BlockedUserItem[] }> {
    return apiFetch<{ users: BlockedUserItem[] }>("/blocked-users");
}

export async function getUserShips(userId: string, limit = 20, offset = 0): Promise<ShipListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<ShipListResponse>(`/users/${userId}/ships${qs}`);
}

export async function getUserMysteries(userId: string, limit = 20, offset = 0): Promise<MysteryListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<MysteryListResponse>(`/users/${userId}/mysteries${qs}`);
}

export async function getUserFanfics(userId: string, limit = 20, offset = 0): Promise<FanficListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<FanficListResponse>(`/users/${userId}/fanfics${qs}`);
}

export async function getUserFanficFavourites(userId: string, limit = 20, offset = 0): Promise<FanficListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<FanficListResponse>(`/users/${userId}/fanfic-favourites${qs}`);
}

export async function getUserJournals(userId: string, limit = 20, offset = 0): Promise<JournalListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<JournalListResponse>(`/users/${userId}/journals${qs}`);
}

export async function getUserFollowedJournals(userId: string, limit = 20, offset = 0): Promise<JournalListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<JournalListResponse>(`/users/${userId}/journal-follows${qs}`);
}

export async function listUserOCs(userId: string, limit = 20, offset = 0): Promise<OCListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<OCListResponse>(`/users/${userId}/ocs${qs}`);
}

export async function listUserOCSummaries(userId: string): Promise<OCSummary[]> {
    return apiFetch<OCSummary[]>(`/users/${userId}/oc-summaries`);
}
