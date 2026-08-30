import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type {
    ActivityItem,
    ActivityListResponse,
    Art,
    ArtListResponse,
    BlockedUserItem,
    BlockStatus,
    Fanfic,
    FanficListResponse,
    FollowStats,
    Gallery,
    Journal,
    JournalListResponse,
    Mystery,
    MysteryListResponse,
    Post,
    PostListResponse,
    PublicUser,
    Ship,
    ShipListResponse,
    User,
} from "../../types/api";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
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
import {
    fetchMutualFollowers,
    fetchSearchUsers,
    useBlockedUsers,
    useBlockStatus,
    useFollowers,
    useFollowing,
    useFollowStats,
    useMutualFollowers,
    useSearchUsers,
    useUserActivity,
    useUserArt,
    useUserFanficFavourites,
    useUserFanfics,
    useUserFollowedJournals,
    useUserGalleries,
    useUserJournals,
    useUserMysteries,
    useUserPosts,
    useUsersPublic,
    useUserShips,
} from "./user";

vi.mock("../../api/endpoints/user", () => ({
    getBlockedUsers: vi.fn(),
    getBlockStatus: vi.fn(),
    getFollowers: vi.fn(),
    getFollowing: vi.fn(),
    getFollowStats: vi.fn(),
    getMutualFollowers: vi.fn(),
    getUserActivity: vi.fn(),
    getUserArt: vi.fn(),
    getUserFanficFavourites: vi.fn(),
    getUserFanfics: vi.fn(),
    getUserFollowedJournals: vi.fn(),
    getUserGalleries: vi.fn(),
    getUserJournals: vi.fn(),
    getUserMysteries: vi.fn(),
    getUserPosts: vi.fn(),
    getUserShips: vi.fn(),
    listUsersPublic: vi.fn(),
    searchUsers: vi.fn(),
}));

const mockedGetBlockedUsers = vi.mocked(getBlockedUsers);
const mockedGetBlockStatus = vi.mocked(getBlockStatus);
const mockedGetFollowers = vi.mocked(getFollowers);
const mockedGetFollowing = vi.mocked(getFollowing);
const mockedGetFollowStats = vi.mocked(getFollowStats);
const mockedGetMutualFollowers = vi.mocked(getMutualFollowers);
const mockedGetUserActivity = vi.mocked(getUserActivity);
const mockedGetUserArt = vi.mocked(getUserArt);
const mockedGetUserFanficFavourites = vi.mocked(getUserFanficFavourites);
const mockedGetUserFanfics = vi.mocked(getUserFanfics);
const mockedGetUserFollowedJournals = vi.mocked(getUserFollowedJournals);
const mockedGetUserGalleries = vi.mocked(getUserGalleries);
const mockedGetUserJournals = vi.mocked(getUserJournals);
const mockedGetUserMysteries = vi.mocked(getUserMysteries);
const mockedGetUserPosts = vi.mocked(getUserPosts);
const mockedGetUserShips = vi.mocked(getUserShips);
const mockedListUsersPublic = vi.mocked(listUsersPublic);
const mockedSearchUsers = vi.mocked(searchUsers);

const userId = "11111111-1111-1111-1111-111111111111";

function entity<T>(id: string): T {
    return { id } as unknown as T;
}

function makeActivityItem(title: string): ActivityItem {
    return { type: "response", theory_id: "t-1", theory_title: title, body: "", created_at: "2026-01-01T00:00:00Z" };
}

function makeBlocked(id: string): BlockedUserItem {
    return { id, username: "erika", display_name: "Erika", avatar_url: "", blocked_at: "2026-01-01T00:00:00Z" };
}

function makeApiUser(username: string): User {
    return { id: `id-${username}`, username, display_name: username };
}

function makeStatsResponse(overrides: Partial<FollowStats> = {}): FollowStats {
    return { follower_count: 3, following_count: 5, is_following: false, follows_you: true, ...overrides };
}

function firstKey(qc: QueryClient): readonly unknown[] {
    return qc.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    queryClient.clear();
    mockedGetUserPosts.mockResolvedValue({ posts: [entity<Post>("p-1")], total: 1, limit: 20, offset: 0 });
    mockedGetUserArt.mockResolvedValue({ art: [entity<Art>("a-1")], total: 1, limit: 24, offset: 0 });
    mockedGetUserGalleries.mockResolvedValue([entity<Gallery>("g-1")]);
    mockedGetUserShips.mockResolvedValue({ ships: [entity<Ship>("s-1")], total: 1, limit: 20, offset: 0 });
    mockedGetUserMysteries.mockResolvedValue({
        mysteries: [entity<Mystery>("m-1")],
        total: 1,
        limit: 20,
        offset: 0,
    });
    mockedGetUserFanfics.mockResolvedValue({ fanfics: [entity<Fanfic>("f-1")], total: 1, limit: 20, offset: 0 });
    mockedGetUserFanficFavourites.mockResolvedValue({
        fanfics: [entity<Fanfic>("f-2")],
        total: 1,
        limit: 20,
        offset: 0,
    });
    mockedGetUserJournals.mockResolvedValue({ journals: [entity<Journal>("j-1")], total: 1, limit: 20, offset: 0 });
    mockedGetUserFollowedJournals.mockResolvedValue({
        journals: [entity<Journal>("j-2")],
        total: 1,
        limit: 20,
        offset: 0,
    });
    mockedGetBlockedUsers.mockResolvedValue({ users: [makeBlocked("u-1")] });
    mockedGetUserActivity.mockResolvedValue({ items: [makeActivityItem("a theory")], total: 1, limit: 20, offset: 0 });
});

describe("useUserPosts", () => {
    it("keys the query by the owner, the kind and the paging", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserPosts(userId, 10, 20), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["user", userId, "posts", { limit: 10, offset: 20 }]);
        expect(mockedGetUserPosts).toHaveBeenCalledWith(userId, 10, 20);
    });

    it("returns the posts and the total once the response arrives", async () => {
        // given
        mockedGetUserPosts.mockResolvedValue({
            posts: [entity<Post>("p-1"), entity<Post>("p-2")],
            total: 8,
            limit: 20,
            offset: 0,
        });

        // when
        const { result } = renderHook(() => useUserPosts(userId, 20, 0), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.posts).toHaveLength(2);
        expect(result.current.total).toBe(8);
    });

    it("does not ask the server without an owner id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserPosts("", 20, 0), { wrapper });

        // then
        expect(mockedGetUserPosts).not.toHaveBeenCalled();
        expect(result.current.posts).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.loading).toBe(false);
    });

    it("falls back to empty values when the response carries no posts", async () => {
        // given
        mockedGetUserPosts.mockResolvedValue({} as unknown as PostListResponse);

        // when
        const { result } = renderHook(() => useUserPosts(userId, 20, 0), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.posts).toEqual([]);
        expect(result.current.total).toBe(0);
    });

    it("fetches the posts again when refresh is called", async () => {
        // given
        const { result } = renderHook(() => useUserPosts(userId, 20, 0), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedGetUserPosts).toHaveBeenCalledTimes(2);
    });
});

describe("useUserArt", () => {
    it("keys the query by the owner, the kind and the paging", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserArt(userId, 24, 0), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["user", userId, "art", { limit: 24, offset: 0 }]);
        expect(mockedGetUserArt).toHaveBeenCalledWith(userId, 24, 0);
        expect(result.current.art).toHaveLength(1);
    });

    it("does not ask the server without an owner id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserArt("", 24, 0), { wrapper });

        // then
        expect(mockedGetUserArt).not.toHaveBeenCalled();
        expect(result.current.art).toEqual([]);
        expect(result.current.total).toBe(0);
    });

    it("falls back to empty values when the response carries no art", async () => {
        // given
        mockedGetUserArt.mockResolvedValue({} as unknown as ArtListResponse);

        // when
        const { result } = renderHook(() => useUserArt(userId, 24, 0), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.art).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useUserGalleries", () => {
    it("keys the query by the owner with no paging of its own", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserGalleries(userId), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["user", userId, "galleries", {}]);
        expect(mockedGetUserGalleries).toHaveBeenCalledWith(userId);
    });

    it("returns the galleries straight from the response", async () => {
        // given
        mockedGetUserGalleries.mockResolvedValue([entity<Gallery>("g-1"), entity<Gallery>("g-2")]);

        // when
        const { result } = renderHook(() => useUserGalleries(userId), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.galleries).toHaveLength(2);
    });

    it("does not ask the server without an owner id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserGalleries(""), { wrapper });

        // then
        expect(mockedGetUserGalleries).not.toHaveBeenCalled();
        expect(result.current.galleries).toEqual([]);
    });

    it("fetches the galleries again when refresh is called", async () => {
        // given
        const { result } = renderHook(() => useUserGalleries(userId), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedGetUserGalleries).toHaveBeenCalledTimes(2);
    });
});

describe("useUserShips", () => {
    it("keys the query by the owner, the kind and the paging", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserShips(userId, 20, 40), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["user", userId, "ships", { limit: 20, offset: 40 }]);
        expect(mockedGetUserShips).toHaveBeenCalledWith(userId, 20, 40);
        expect(result.current.ships).toHaveLength(1);
        expect(result.current.total).toBe(1);
    });

    it("does not ask the server without an owner id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserShips("", 20, 0), { wrapper });

        // then
        expect(mockedGetUserShips).not.toHaveBeenCalled();
        expect(result.current.ships).toEqual([]);
    });

    it("falls back to empty values when the response carries no ships", async () => {
        // given
        mockedGetUserShips.mockResolvedValue({} as unknown as ShipListResponse);

        // when
        const { result } = renderHook(() => useUserShips(userId, 20, 0), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.ships).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useUserMysteries", () => {
    it("keys the query by the owner, the kind and the paging", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserMysteries(userId, 20, 0), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["user", userId, "mysteries", { limit: 20, offset: 0 }]);
        expect(mockedGetUserMysteries).toHaveBeenCalledWith(userId, 20, 0);
        expect(result.current.mysteries).toHaveLength(1);
    });

    it("does not ask the server without an owner id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserMysteries("", 20, 0), { wrapper });

        // then
        expect(mockedGetUserMysteries).not.toHaveBeenCalled();
        expect(result.current.mysteries).toEqual([]);
    });

    it("falls back to empty values when the response carries no mysteries", async () => {
        // given
        mockedGetUserMysteries.mockResolvedValue({} as unknown as MysteryListResponse);

        // when
        const { result } = renderHook(() => useUserMysteries(userId, 20, 0), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.mysteries).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useUserFanfics", () => {
    it("keys the query by the owner, the kind and the paging", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserFanfics(userId, 20, 0), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["user", userId, "fanfics", { limit: 20, offset: 0 }]);
        expect(mockedGetUserFanfics).toHaveBeenCalledWith(userId, 20, 0);
        expect(result.current.fanfics).toHaveLength(1);
    });

    it("does not ask the server without an owner id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserFanfics("", 20, 0), { wrapper });

        // then
        expect(mockedGetUserFanfics).not.toHaveBeenCalled();
        expect(result.current.fanfics).toEqual([]);
    });
});

describe("useUserFanficFavourites", () => {
    it("keys the favourites apart from the authored fanfics", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserFanficFavourites(userId, 20, 0), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["user", userId, "fanfic-favourites", { limit: 20, offset: 0 }]);
        expect(mockedGetUserFanficFavourites).toHaveBeenCalledWith(userId, 20, 0);
        expect(mockedGetUserFanfics).not.toHaveBeenCalled();
    });

    it("does not ask the server without an owner id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserFanficFavourites("", 20, 0), { wrapper });

        // then
        expect(mockedGetUserFanficFavourites).not.toHaveBeenCalled();
        expect(result.current.fanfics).toEqual([]);
    });

    it("falls back to empty values when the response carries no favourites", async () => {
        // given
        mockedGetUserFanficFavourites.mockResolvedValue({} as unknown as FanficListResponse);

        // when
        const { result } = renderHook(() => useUserFanficFavourites(userId, 20, 0), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.fanfics).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useUserJournals", () => {
    it("keys the query by the owner, the kind and the paging", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserJournals(userId, 20, 0), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["user", userId, "journals", { limit: 20, offset: 0 }]);
        expect(mockedGetUserJournals).toHaveBeenCalledWith(userId, 20, 0);
        expect(result.current.journals).toHaveLength(1);
    });

    it("does not ask the server without an owner id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserJournals("", 20, 0), { wrapper });

        // then
        expect(mockedGetUserJournals).not.toHaveBeenCalled();
        expect(result.current.journals).toEqual([]);
    });
});

describe("useUserFollowedJournals", () => {
    it("keys the followed journals apart from the authored ones", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserFollowedJournals(userId, 20, 0), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["user", userId, "followed-journals", { limit: 20, offset: 0 }]);
        expect(mockedGetUserFollowedJournals).toHaveBeenCalledWith(userId, 20, 0);
        expect(mockedGetUserJournals).not.toHaveBeenCalled();
    });

    it("falls back to empty values when the response carries no journals", async () => {
        // given
        mockedGetUserFollowedJournals.mockResolvedValue({} as unknown as JournalListResponse);

        // when
        const { result } = renderHook(() => useUserFollowedJournals(userId, 20, 0), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.journals).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useBlockedUsers", () => {
    it("keys the block list under the profile of the viewer", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useBlockedUsers(userId), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["profile", userId, "blocked"]);
    });

    it("asks the server for the block list of whoever is signed in", async () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useBlockedUsers(userId), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedGetBlockedUsers).toHaveBeenCalledWith();
        expect(result.current.blocked).toHaveLength(1);
    });

    it("does not ask the server when nobody is signed in", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useBlockedUsers(""), { wrapper });

        // then
        expect(mockedGetBlockedUsers).not.toHaveBeenCalled();
        expect(result.current.blocked).toEqual([]);
        expect(result.current.loading).toBe(false);
    });

    it("fetches the block list again when refresh is called", async () => {
        // given
        const { result } = renderHook(() => useBlockedUsers(userId), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedGetBlockedUsers).toHaveBeenCalledTimes(2);
    });
});

describe("useUserActivity", () => {
    it("keys the activity by the username and the paging", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserActivity("beatrice", 5, 10), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["user", "beatrice", "activity", { limit: 5, offset: 10 }]);
        expect(mockedGetUserActivity).toHaveBeenCalledWith("beatrice", 5, 10);
    });

    it("asks for twenty entries from the start by default", async () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserActivity("beatrice"), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedGetUserActivity).toHaveBeenCalledWith("beatrice", 20, 0);
    });

    it("returns the activity items and the total once they arrive", async () => {
        // given
        mockedGetUserActivity.mockResolvedValue({
            items: [makeActivityItem("first"), makeActivityItem("second")],
            total: 30,
            limit: 20,
            offset: 0,
        });

        // when
        const { result } = renderHook(() => useUserActivity("beatrice"), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.activity).toHaveLength(2);
        expect(result.current.total).toBe(30);
    });

    it("does not ask the server without a username", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserActivity(""), { wrapper });

        // then
        expect(mockedGetUserActivity).not.toHaveBeenCalled();
        expect(result.current.activity).toEqual([]);
        expect(result.current.total).toBe(0);
    });

    it("falls back to empty values when the response carries no items", async () => {
        // given
        mockedGetUserActivity.mockResolvedValue({} as unknown as ActivityListResponse);

        // when
        const { result } = renderHook(() => useUserActivity("beatrice"), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.activity).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("fetchMutualFollowers", () => {
    it("fetches the mutuals through the shared query client", async () => {
        // given
        mockedGetMutualFollowers.mockResolvedValue([makeApiUser("beatrice")]);

        // when
        const users = await fetchMutualFollowers();

        // then
        expect(users).toEqual([makeApiUser("beatrice")]);
        expect(queryClient.getQueryData(["users", "mutuals"])).toEqual([makeApiUser("beatrice")]);
    });

    it("serves a second call from the cache instead of asking again", async () => {
        // given
        mockedGetMutualFollowers.mockResolvedValue([makeApiUser("beatrice")]);
        await fetchMutualFollowers();

        // when
        await fetchMutualFollowers();

        // then
        expect(mockedGetMutualFollowers).toHaveBeenCalledTimes(1);
    });
});

describe("fetchSearchUsers", () => {
    it("caches each search term under its own key", async () => {
        // given
        mockedSearchUsers.mockResolvedValue([makeApiUser("battler")]);

        // when
        const users = await fetchSearchUsers("bat");

        // then
        expect(mockedSearchUsers).toHaveBeenCalledWith("bat");
        expect(users).toEqual([makeApiUser("battler")]);
        expect(queryClient.getQueryData(["users", "search", "bat"])).toEqual([makeApiUser("battler")]);
        expect(queryClient.getQueryData(["users", "search", "ba"])).toBeUndefined();
    });
});

describe("useSearchUsers", () => {
    it("searches for the term and returns the matches", async () => {
        // given
        mockedSearchUsers.mockResolvedValue([makeApiUser("battler"), makeApiUser("beatrice")]);
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useSearchUsers("b"), { wrapper: providerWrapper({ queryClient: client }) });

        // then
        await waitFor(() => expect(result.current.users).toHaveLength(2));
        expect(mockedSearchUsers).toHaveBeenCalledWith("b");
        expect(client.getQueryData(["users", "search", "b"])).toBeDefined();
    });

    it("does not search while the term is empty", () => {
        // given
        mockedSearchUsers.mockResolvedValue([]);

        // when
        const { result } = renderHook(() => useSearchUsers(""), { wrapper: providerWrapper() });

        // then
        expect(mockedSearchUsers).not.toHaveBeenCalled();
        expect(result.current.users).toEqual([]);
    });

    it("does not search while it is disabled", () => {
        // given
        mockedSearchUsers.mockResolvedValue([]);

        // when
        renderHook(() => useSearchUsers("b", false), { wrapper: providerWrapper() });

        // then
        expect(mockedSearchUsers).not.toHaveBeenCalled();
    });
});

describe("useMutualFollowers", () => {
    it("returns the mutuals once they arrive", async () => {
        // given
        mockedGetMutualFollowers.mockResolvedValue([makeApiUser("beatrice")]);

        // when
        const { result } = renderHook(() => useMutualFollowers(), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.mutuals).toHaveLength(1));
    });

    it("does not fetch while it is disabled", () => {
        // given
        mockedGetMutualFollowers.mockResolvedValue([]);

        // when
        const { result } = renderHook(() => useMutualFollowers(false), { wrapper: providerWrapper() });

        // then
        expect(mockedGetMutualFollowers).not.toHaveBeenCalled();
        expect(result.current.mutuals).toEqual([]);
    });
});

describe("useFollowStats", () => {
    it("fetches the stats for the given user", async () => {
        // given
        mockedGetFollowStats.mockResolvedValue(makeStatsResponse());
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useFollowStats("u1"), {
            wrapper: providerWrapper({ queryClient: client }),
        });

        // then
        await waitFor(() => expect(result.current.stats).toEqual(makeStatsResponse()));
        expect(mockedGetFollowStats).toHaveBeenCalledWith("u1");
        expect(client.getQueryData(["follow-stats", "u1"])).toBeDefined();
    });

    it("does not fetch when there is no user id", () => {
        // given
        mockedGetFollowStats.mockResolvedValue(makeStatsResponse());

        // when
        const { result } = renderHook(() => useFollowStats(""), { wrapper: providerWrapper() });

        // then
        expect(mockedGetFollowStats).not.toHaveBeenCalled();
        expect(result.current.stats).toBeNull();
    });

    it("refetches the stats when refresh is called", async () => {
        // given
        mockedGetFollowStats.mockResolvedValue(makeStatsResponse());
        const { result } = renderHook(() => useFollowStats("u1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.stats).not.toBeNull());

        // when
        await result.current.refresh();

        // then
        expect(mockedGetFollowStats).toHaveBeenCalledTimes(2);
    });
});

describe("useFollowers", () => {
    it("asks for the default window of fifty", async () => {
        // given
        mockedGetFollowers.mockResolvedValue({ users: [makeApiUser("battler")], total: 1 });
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useFollowers("u1"), { wrapper: providerWrapper({ queryClient: client }) });

        // then
        await waitFor(() => expect(result.current.users).toHaveLength(1));
        expect(mockedGetFollowers).toHaveBeenCalledWith("u1", 50, 0);
        expect(client.getQueryData(["users", "u1", "followers", { limit: 50, offset: 0 }])).toBeDefined();
    });

    it("forwards an explicit window", async () => {
        // given
        mockedGetFollowers.mockResolvedValue({ users: [], total: 80 });

        // when
        const { result } = renderHook(() => useFollowers("u1", 10, 20), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.total).toBe(80));
        expect(mockedGetFollowers).toHaveBeenCalledWith("u1", 10, 20);
    });

    it("does not fetch when there is no user id", () => {
        // given
        mockedGetFollowers.mockResolvedValue({ users: [], total: 0 });

        // when
        const { result } = renderHook(() => useFollowers(""), { wrapper: providerWrapper() });

        // then
        expect(mockedGetFollowers).not.toHaveBeenCalled();
        expect(result.current.users).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useFollowing", () => {
    it("asks for the default window of fifty", async () => {
        // given
        mockedGetFollowing.mockResolvedValue({ users: [makeApiUser("beatrice")], total: 1 });
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useFollowing("u1"), { wrapper: providerWrapper({ queryClient: client }) });

        // then
        await waitFor(() => expect(result.current.users).toHaveLength(1));
        expect(mockedGetFollowing).toHaveBeenCalledWith("u1", 50, 0);
        expect(client.getQueryData(["users", "u1", "following", { limit: 50, offset: 0 }])).toBeDefined();
    });

    it("does not fetch when there is no user id", () => {
        // given
        mockedGetFollowing.mockResolvedValue({ users: [], total: 0 });

        // when
        const { result } = renderHook(() => useFollowing(""), { wrapper: providerWrapper() });

        // then
        expect(mockedGetFollowing).not.toHaveBeenCalled();
        expect(result.current.users).toEqual([]);
    });
});

describe("useUsersPublic", () => {
    it("returns the public member list", async () => {
        // given
        const users = [{ ...makeApiUser("beatrice"), online: true }] as PublicUser[];
        mockedListUsersPublic.mockResolvedValue(users);

        // when
        const { result } = renderHook(() => useUsersPublic(), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.users).toEqual(users));
    });

    it("falls back to an empty list while loading", () => {
        // given
        mockedListUsersPublic.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useUsersPublic(), { wrapper: providerWrapper() });

        // then
        expect(result.current.users).toEqual([]);
        expect(result.current.loading).toBe(true);
    });
});

describe("useBlockStatus", () => {
    it("returns the block status for the given user", async () => {
        // given
        const status: BlockStatus = { blocking: true, blocked_by: false };
        mockedGetBlockStatus.mockResolvedValue(status);
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useBlockStatus("u1"), {
            wrapper: providerWrapper({ queryClient: client }),
        });

        // then
        await waitFor(() => expect(result.current.status).toEqual(status));
        expect(mockedGetBlockStatus).toHaveBeenCalledWith("u1");
        expect(client.getQueryData(["block-status", "u1"])).toEqual(status);
    });

    it("assumes nobody is blocked before the answer arrives", () => {
        // given
        mockedGetBlockStatus.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useBlockStatus("u1"), { wrapper: providerWrapper() });

        // then
        expect(result.current.status).toEqual({ blocking: false, blocked_by: false });
    });

    it("does not fetch when there is no user id", () => {
        // given
        mockedGetBlockStatus.mockResolvedValue({ blocking: false, blocked_by: false });

        // when
        renderHook(() => useBlockStatus(""), { wrapper: providerWrapper() });

        // then
        expect(mockedGetBlockStatus).not.toHaveBeenCalled();
    });
});
