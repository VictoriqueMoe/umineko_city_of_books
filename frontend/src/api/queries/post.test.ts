import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Post, PostDetail, PostListResponse } from "../../types/api";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { getPost, listPosts } from "../endpoints";
import { usePost, usePostFeed } from "./post";

vi.mock("../endpoints", () => ({
    getPost: vi.fn(),
    listPosts: vi.fn(),
}));

const mockedGetPost = vi.mocked(getPost);
const mockedListPosts = vi.mocked(listPosts);

const seed = 500000;

let randomSpy: ReturnType<typeof vi.spyOn>;

function makePost(id: string): Post {
    return { id, body: `post ${id}` } as unknown as Post;
}

function makePostList(posts: Post[], total: number): PostListResponse {
    return { posts, total, limit: 20, offset: 0 };
}

function firstKey(qc: QueryClient): readonly unknown[] {
    return qc.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    randomSpy = vi.spyOn(Math, "random").mockReturnValue(0.5);
    mockedListPosts.mockResolvedValue(makePostList([makePost("p-1")], 1));
    mockedGetPost.mockResolvedValue({ id: "p-1", body: "post p-1" } as unknown as PostDetail);
});

afterEach(() => {
    randomSpy.mockRestore();
});

describe("usePostFeed", () => {
    it("keys the feed by its filters, its paging and the seed it generated", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => usePostFeed("everyone", "art", "beato", "new", 2, "open"), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual([
            "post",
            "feed",
            {
                tab: "everyone",
                corner: "art",
                search: "beato",
                sort: "new",
                resolved: "open",
                offset: 20,
                limit: 20,
                seed,
            },
        ]);
    });

    it("turns the page number into an offset of twenty per page", async () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => usePostFeed("everyone", "general", "", "", 3), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(result.current.offset).toBe(40);
        expect(result.current.limit).toBe(20);
        expect(mockedListPosts).toHaveBeenCalledWith(expect.objectContaining({ offset: 40, limit: 20 }));
    });

    it("drops the empty search, sort and resolved filters before asking the server", async () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => usePostFeed("following", "general", "", "", 1, ""), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedListPosts).toHaveBeenCalledWith({
            tab: "following",
            corner: "general",
            search: undefined,
            sort: undefined,
            seed,
            limit: 20,
            offset: 0,
            resolved: undefined,
        });
    });

    it("defaults to the general corner and the first page", async () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => usePostFeed("everyone"), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedListPosts).toHaveBeenCalledWith(expect.objectContaining({ corner: "general", offset: 0 }));
    });

    it("reports empty values while the feed is loading", () => {
        // given
        mockedListPosts.mockReturnValue(new Promise<PostListResponse>(() => {}));

        // when
        const { result } = renderHook(() => usePostFeed("everyone"), { wrapper: providerWrapper() });

        // then
        expect(result.current.posts).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.loading).toBe(true);
        expect(result.current.hasNext).toBe(false);
        expect(result.current.hasPrev).toBe(false);
    });

    it("offers a next page while more posts remain", async () => {
        // given
        mockedListPosts.mockResolvedValue(makePostList([makePost("p-1")], 45));

        // when
        const { result } = renderHook(() => usePostFeed("everyone", "general", "", "", 1), {
            wrapper: providerWrapper(),
        });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.hasNext).toBe(true);
        expect(result.current.hasPrev).toBe(false);
    });

    it("offers no next page once the last page has been reached", async () => {
        // given
        mockedListPosts.mockResolvedValue(makePostList([makePost("p-1")], 45));

        // when
        const { result } = renderHook(() => usePostFeed("everyone", "general", "", "", 3), {
            wrapper: providerWrapper(),
        });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.hasNext).toBe(false);
        expect(result.current.hasPrev).toBe(true);
    });

    it("keeps the same seed across re-renders so the shuffled order is stable", async () => {
        // given
        const { result, rerender } = renderHook(() => usePostFeed("everyone"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        rerender();

        // then
        expect(randomSpy).toHaveBeenCalledOnce();
        expect(mockedListPosts).toHaveBeenCalledOnce();
    });

    it("fetches the feed again when refresh is called", async () => {
        // given
        const { result } = renderHook(() => usePostFeed("everyone"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedListPosts).toHaveBeenCalledTimes(2);
    });
});

describe("usePost", () => {
    it("keys the detail query by the post id", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => usePost("p-7"), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.post).not.toBeNull());

        // then
        expect(firstKey(qc)).toEqual(["post", "detail", "p-7"]);
        expect(mockedGetPost).toHaveBeenCalledWith("p-7");
    });

    it("does not ask the server for a post without an id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => usePost(""), { wrapper });

        // then
        expect(mockedGetPost).not.toHaveBeenCalled();
        expect(result.current.post).toBeNull();
        expect(result.current.loading).toBe(false);
    });

    it("returns the post once it has loaded", async () => {
        // given
        mockedGetPost.mockResolvedValue({ id: "p-2", body: "the witch laughed" } as unknown as PostDetail);

        // when
        const { result } = renderHook(() => usePost("p-2"), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.post).toEqual({ id: "p-2", body: "the witch laughed" });
    });

    it("fetches the post again when refresh is called", async () => {
        // given
        const { result } = renderHook(() => usePost("p-1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedGetPost).toHaveBeenCalledTimes(2);
    });
});
