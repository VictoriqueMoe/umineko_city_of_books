import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type {
    ActivityItem,
    ActivityListResponse,
    Art,
    ArtListResponse,
    Fanfic,
    FanficListResponse,
    Gallery,
    Journal,
    JournalListResponse,
    Mystery,
    MysteryListResponse,
    Post,
    PostListResponse,
    Ship,
    ShipListResponse,
} from "../../types/api";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import {
    type BlockedUserItem,
    getBlockedUsers,
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
} from "../endpoints";
import {
    useBlockedUsers,
    useUserActivity,
    useUserArt,
    useUserFanficFavourites,
    useUserFanfics,
    useUserFollowedJournals,
    useUserGalleries,
    useUserJournals,
    useUserMysteries,
    useUserPosts,
    useUserShips,
} from "./user";

vi.mock("../endpoints", () => ({
    getBlockedUsers: vi.fn(),
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
}));

const mockedGetBlockedUsers = vi.mocked(getBlockedUsers);
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

function firstKey(qc: QueryClient): readonly unknown[] {
    return qc.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
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
