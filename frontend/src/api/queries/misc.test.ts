import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { FollowStats, TagCount, User } from "../../types/api";
import type { BlockStatus, PublicUser } from "../endpoints";
import * as endpoints from "../endpoints";
import { queryClient } from "../queryClient";
import { queryKeys } from "../queryKeys";
import {
    fetchMutualFollowers,
    fetchSearchUsers,
    useArtCornerCounts,
    useBlockStatus,
    useCornerCounts,
    useFollowers,
    useFollowing,
    useFollowStats,
    useMutualFollowers,
    usePopularTags,
    useRules,
    useSearchUsers,
    useUsersPublic,
} from "./misc";

vi.mock("../endpoints", () => ({
    getArtCornerCounts: vi.fn(),
    getBlockStatus: vi.fn(),
    getCornerCounts: vi.fn(),
    getFollowers: vi.fn(),
    getFollowing: vi.fn(),
    getFollowStats: vi.fn(),
    getMutualFollowers: vi.fn(),
    getPopularTags: vi.fn(),
    getRules: vi.fn(),
    listUsersPublic: vi.fn(),
    searchUsers: vi.fn(),
}));

const getArtCornerCounts = vi.mocked(endpoints.getArtCornerCounts);
const getBlockStatus = vi.mocked(endpoints.getBlockStatus);
const getCornerCounts = vi.mocked(endpoints.getCornerCounts);
const getFollowers = vi.mocked(endpoints.getFollowers);
const getFollowing = vi.mocked(endpoints.getFollowing);
const getFollowStats = vi.mocked(endpoints.getFollowStats);
const getMutualFollowers = vi.mocked(endpoints.getMutualFollowers);
const getPopularTags = vi.mocked(endpoints.getPopularTags);
const getRules = vi.mocked(endpoints.getRules);
const listUsersPublic = vi.mocked(endpoints.listUsersPublic);
const searchUsers = vi.mocked(endpoints.searchUsers);

function makeApiUser(username: string): User {
    return { id: `id-${username}`, username, display_name: username };
}

function makeStatsResponse(overrides: Partial<FollowStats> = {}): FollowStats {
    return { follower_count: 3, following_count: 5, is_following: false, follows_you: true, ...overrides };
}

beforeEach(() => {
    queryClient.clear();
});

describe("fetchMutualFollowers", () => {
    it("fetches the mutuals through the shared query client", async () => {
        // given
        getMutualFollowers.mockResolvedValue([makeApiUser("beatrice")]);

        // when
        const users = await fetchMutualFollowers();

        // then
        expect(users).toEqual([makeApiUser("beatrice")]);
        expect(queryClient.getQueryData(["users", "mutuals"])).toEqual([makeApiUser("beatrice")]);
    });

    it("serves a second call from the cache instead of asking again", async () => {
        // given
        getMutualFollowers.mockResolvedValue([makeApiUser("beatrice")]);
        await fetchMutualFollowers();

        // when
        await fetchMutualFollowers();

        // then
        expect(getMutualFollowers).toHaveBeenCalledTimes(1);
    });
});

describe("fetchSearchUsers", () => {
    it("caches each search term under its own key", async () => {
        // given
        searchUsers.mockResolvedValue([makeApiUser("battler")]);

        // when
        const users = await fetchSearchUsers("bat");

        // then
        expect(searchUsers).toHaveBeenCalledWith("bat");
        expect(users).toEqual([makeApiUser("battler")]);
        expect(queryClient.getQueryData(["users", "search", "bat"])).toEqual([makeApiUser("battler")]);
        expect(queryClient.getQueryData(["users", "search", "ba"])).toBeUndefined();
    });
});

describe("useSearchUsers", () => {
    it("searches for the term and returns the matches", async () => {
        // given
        searchUsers.mockResolvedValue([makeApiUser("battler"), makeApiUser("beatrice")]);
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useSearchUsers("b"), { wrapper: providerWrapper({ queryClient: client }) });

        // then
        await waitFor(() => expect(result.current.users).toHaveLength(2));
        expect(searchUsers).toHaveBeenCalledWith("b");
        expect(client.getQueryData(["users", "search", "b"])).toBeDefined();
    });

    it("does not search while the term is empty", () => {
        // given
        searchUsers.mockResolvedValue([]);

        // when
        const { result } = renderHook(() => useSearchUsers(""), { wrapper: providerWrapper() });

        // then
        expect(searchUsers).not.toHaveBeenCalled();
        expect(result.current.users).toEqual([]);
    });

    it("does not search while it is disabled", () => {
        // given
        searchUsers.mockResolvedValue([]);

        // when
        renderHook(() => useSearchUsers("b", false), { wrapper: providerWrapper() });

        // then
        expect(searchUsers).not.toHaveBeenCalled();
    });
});

describe("useMutualFollowers", () => {
    it("returns the mutuals once they arrive", async () => {
        // given
        getMutualFollowers.mockResolvedValue([makeApiUser("beatrice")]);

        // when
        const { result } = renderHook(() => useMutualFollowers(), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.mutuals).toHaveLength(1));
    });

    it("does not fetch while it is disabled", () => {
        // given
        getMutualFollowers.mockResolvedValue([]);

        // when
        const { result } = renderHook(() => useMutualFollowers(false), { wrapper: providerWrapper() });

        // then
        expect(getMutualFollowers).not.toHaveBeenCalled();
        expect(result.current.mutuals).toEqual([]);
    });
});

describe("useCornerCounts", () => {
    it("returns the post counts under the post corner counts key", async () => {
        // given
        getCornerCounts.mockResolvedValue({ parlour: 4 });
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useCornerCounts(), { wrapper: providerWrapper({ queryClient: client }) });

        // then
        await waitFor(() => expect(result.current.counts).toEqual({ parlour: 4 }));
        expect(client.getQueryData(queryKeys.post.cornerCounts())).toEqual({ parlour: 4 });
    });

    it("falls back to an empty map while loading", () => {
        // given
        getCornerCounts.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useCornerCounts(), { wrapper: providerWrapper() });

        // then
        expect(result.current.counts).toEqual({});
        expect(result.current.loading).toBe(true);
    });
});

describe("useArtCornerCounts", () => {
    it("returns the art counts under the art corner counts key", async () => {
        // given
        getArtCornerCounts.mockResolvedValue({ gallery: 9 });
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useArtCornerCounts(), {
            wrapper: providerWrapper({ queryClient: client }),
        });

        // then
        await waitFor(() => expect(result.current.counts).toEqual({ gallery: 9 }));
        expect(client.getQueryData(["art", "corner-counts"])).toEqual({ gallery: 9 });
    });

    it("falls back to an empty map while loading", () => {
        // given
        getArtCornerCounts.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useArtCornerCounts(), { wrapper: providerWrapper() });

        // then
        expect(result.current.counts).toEqual({});
    });
});

describe("usePopularTags", () => {
    it("keys the tags by the corner it was given", async () => {
        // given
        const tags: TagCount[] = [{ tag: "beatrice", count: 12 }];
        getPopularTags.mockResolvedValue(tags);
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => usePopularTags("gallery"), {
            wrapper: providerWrapper({ queryClient: client }),
        });

        // then
        await waitFor(() => expect(result.current.tags).toEqual(tags));
        expect(getPopularTags).toHaveBeenCalledWith("gallery");
        expect(client.getQueryData(["art", "popular-tags", "gallery"])).toEqual(tags);
    });

    it("uses the empty corner key when no corner is given", async () => {
        // given
        getPopularTags.mockResolvedValue([]);
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => usePopularTags(), { wrapper: providerWrapper({ queryClient: client }) });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(getPopularTags).toHaveBeenCalledWith(undefined);
        expect(client.getQueryData(["art", "popular-tags", ""])).toEqual([]);
    });
});

describe("useFollowStats", () => {
    it("fetches the stats for the given user", async () => {
        // given
        getFollowStats.mockResolvedValue(makeStatsResponse());
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useFollowStats("u1"), {
            wrapper: providerWrapper({ queryClient: client }),
        });

        // then
        await waitFor(() => expect(result.current.stats).toEqual(makeStatsResponse()));
        expect(getFollowStats).toHaveBeenCalledWith("u1");
        expect(client.getQueryData(["follow-stats", "u1"])).toBeDefined();
    });

    it("does not fetch when there is no user id", () => {
        // given
        getFollowStats.mockResolvedValue(makeStatsResponse());

        // when
        const { result } = renderHook(() => useFollowStats(""), { wrapper: providerWrapper() });

        // then
        expect(getFollowStats).not.toHaveBeenCalled();
        expect(result.current.stats).toBeNull();
    });

    it("refetches the stats when refresh is called", async () => {
        // given
        getFollowStats.mockResolvedValue(makeStatsResponse());
        const { result } = renderHook(() => useFollowStats("u1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.stats).not.toBeNull());

        // when
        await result.current.refresh();

        // then
        expect(getFollowStats).toHaveBeenCalledTimes(2);
    });
});

describe("useFollowers", () => {
    it("asks for the default window of fifty", async () => {
        // given
        getFollowers.mockResolvedValue({ users: [makeApiUser("battler")], total: 1 });
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useFollowers("u1"), { wrapper: providerWrapper({ queryClient: client }) });

        // then
        await waitFor(() => expect(result.current.users).toHaveLength(1));
        expect(getFollowers).toHaveBeenCalledWith("u1", 50, 0);
        expect(client.getQueryData(["users", "u1", "followers", { limit: 50, offset: 0 }])).toBeDefined();
    });

    it("forwards an explicit window", async () => {
        // given
        getFollowers.mockResolvedValue({ users: [], total: 80 });

        // when
        const { result } = renderHook(() => useFollowers("u1", 10, 20), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.total).toBe(80));
        expect(getFollowers).toHaveBeenCalledWith("u1", 10, 20);
    });

    it("does not fetch when there is no user id", () => {
        // given
        getFollowers.mockResolvedValue({ users: [], total: 0 });

        // when
        const { result } = renderHook(() => useFollowers(""), { wrapper: providerWrapper() });

        // then
        expect(getFollowers).not.toHaveBeenCalled();
        expect(result.current.users).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useFollowing", () => {
    it("asks for the default window of fifty", async () => {
        // given
        getFollowing.mockResolvedValue({ users: [makeApiUser("beatrice")], total: 1 });
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useFollowing("u1"), { wrapper: providerWrapper({ queryClient: client }) });

        // then
        await waitFor(() => expect(result.current.users).toHaveLength(1));
        expect(getFollowing).toHaveBeenCalledWith("u1", 50, 0);
        expect(client.getQueryData(["users", "u1", "following", { limit: 50, offset: 0 }])).toBeDefined();
    });

    it("does not fetch when there is no user id", () => {
        // given
        getFollowing.mockResolvedValue({ users: [], total: 0 });

        // when
        const { result } = renderHook(() => useFollowing(""), { wrapper: providerWrapper() });

        // then
        expect(getFollowing).not.toHaveBeenCalled();
        expect(result.current.users).toEqual([]);
    });
});

describe("useUsersPublic", () => {
    it("returns the public member list", async () => {
        // given
        const users = [{ ...makeApiUser("beatrice"), online: true }] as PublicUser[];
        listUsersPublic.mockResolvedValue(users);

        // when
        const { result } = renderHook(() => useUsersPublic(), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.users).toEqual(users));
    });

    it("falls back to an empty list while loading", () => {
        // given
        listUsersPublic.mockReturnValue(new Promise(() => {}));

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
        getBlockStatus.mockResolvedValue(status);
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useBlockStatus("u1"), {
            wrapper: providerWrapper({ queryClient: client }),
        });

        // then
        await waitFor(() => expect(result.current.status).toEqual(status));
        expect(getBlockStatus).toHaveBeenCalledWith("u1");
        expect(client.getQueryData(["block-status", "u1"])).toEqual(status);
    });

    it("assumes nobody is blocked before the answer arrives", () => {
        // given
        getBlockStatus.mockReturnValue(new Promise(() => {}));

        // when
        const { result } = renderHook(() => useBlockStatus("u1"), { wrapper: providerWrapper() });

        // then
        expect(result.current.status).toEqual({ blocking: false, blocked_by: false });
    });

    it("does not fetch when there is no user id", () => {
        // given
        getBlockStatus.mockResolvedValue({ blocking: false, blocked_by: false });

        // when
        renderHook(() => useBlockStatus(""), { wrapper: providerWrapper() });

        // then
        expect(getBlockStatus).not.toHaveBeenCalled();
    });
});

describe("useRules", () => {
    it("returns the rules body for the requested page", async () => {
        // given
        getRules.mockResolvedValue({ page: "chat", rules: "Be kind to the furniture" });
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useRules("chat"), { wrapper: providerWrapper({ queryClient: client }) });

        // then
        await waitFor(() => expect(result.current.rules).toBe("Be kind to the furniture"));
        expect(getRules).toHaveBeenCalledWith("chat");
        expect(client.getQueryData(["rules", "chat"])).toBeDefined();
    });

    it("does not fetch when no page is named", () => {
        // given
        getRules.mockResolvedValue({ page: "chat", rules: "Be kind to the furniture" });

        // when
        const { result } = renderHook(() => useRules(""), { wrapper: providerWrapper() });

        // then
        expect(getRules).not.toHaveBeenCalled();
        expect(result.current.rules).toBe("");
    });
});
