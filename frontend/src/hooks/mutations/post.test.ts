import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import * as endpoints from "../../api/endpoints/post";
import * as comments from "../../api/endpoints/comments";
import { ApiError } from "../../api/client";
import { queryKeys } from "../../api/queryKeys";
import {
    useCreateComment,
    useCreatePost,
    useDeleteComment,
    useDeletePost,
    useDeletePostMedia,
    useLikeComment,
    useLikePost,
    useResolveSuggestion,
    useUnlikeComment,
    useUnlikePost,
    useUnresolveSuggestion,
    useUpdateComment,
    useUpdatePost,
    useUploadPostMedia,
    useVotePoll,
} from "./post";

vi.mock("../../api/endpoints/post", () => ({
    createPost: vi.fn(),
    deletePost: vi.fn(),
    deletePostMedia: vi.fn(),
    likePost: vi.fn(),
    resolveSuggestion: vi.fn(),
    unlikePost: vi.fn(),
    unresolveSuggestion: vi.fn(),
    updatePost: vi.fn(),
    uploadPostMedia: vi.fn(),
    votePoll: vi.fn(),
}));

vi.mock("../../api/endpoints/comments", () => ({
    createComment: vi.fn(),
    deleteComment: vi.fn(),
    likeComment: vi.fn(),
    unlikeComment: vi.fn(),
    updateComment: vi.fn(),
}));

interface MutationCase {
    name: string;
    useHook: () => { mutateAsync: (variables: never) => Promise<unknown> };
    variables: unknown;
    endpoint: Mock;
    args: unknown[];
    invalidations: unknown[];
}

const postId = "p-1";
const file = new File(["gold"], "letter.png", { type: "image/png" });
const postViews = [{ queryKey: queryKeys.post.all }, { predicate: expect.any(Function) }];
const detailOnly = [{ queryKey: queryKeys.post.detail(postId) }];

function setup<T>(useHook: () => T) {
    const queryClient = createTestQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const remove = vi.spyOn(queryClient, "removeQueries");
    const { result } = renderHook(useHook, { wrapper: providerWrapper({ queryClient }) });

    return { result, invalidate, remove };
}

beforeEach(() => {
    for (const value of [...Object.values(endpoints), ...Object.values(comments)]) {
        if (vi.isMockFunction(value)) {
            value.mockReset();
        }
    }
    vi.mocked(endpoints.createPost).mockResolvedValue({ id: postId });
});

describe("post mutations", () => {
    const cases: MutationCase[] = [
        {
            name: "useCreatePost sends the body, the corner and the shared content in order",
            useHook: () => useCreatePost(),
            variables: {
                body: "Without love it cannot be seen",
                corner: "theories",
                sharedContentId: "art-3",
                sharedContentType: "art",
            },
            endpoint: vi.mocked(endpoints.createPost),
            args: ["Without love it cannot be seen", "theories", undefined, "art-3", "art", undefined],
            invalidations: postViews,
        },
        {
            name: "useUpdatePost refreshes both its own post and the feed",
            useHook: () => useUpdatePost(postId),
            variables: "an edited body",
            endpoint: vi.mocked(endpoints.updatePost),
            args: [postId, "an edited body"],
            invalidations: postViews,
        },
        {
            name: "useUploadPostMedia attaches the file to the post it was built with",
            useHook: () => useUploadPostMedia(postId),
            variables: { file },
            endpoint: vi.mocked(endpoints.uploadPostMedia),
            args: [postId, file, false],
            invalidations: postViews,
        },
        {
            name: "useDeletePostMedia deletes a numbered media item from its own post",
            useHook: () => useDeletePostMedia(postId),
            variables: 7,
            endpoint: vi.mocked(endpoints.deletePostMedia),
            args: [postId, 7],
            invalidations: postViews,
        },
        {
            name: "useLikePost refreshes every view of posts so the feed shows the like",
            useHook: () => useLikePost(),
            variables: postId,
            endpoint: vi.mocked(endpoints.likePost),
            args: [postId],
            invalidations: postViews,
        },
        {
            name: "useUnlikePost refreshes every view of posts so the feed drops the like",
            useHook: () => useUnlikePost(),
            variables: postId,
            endpoint: vi.mocked(endpoints.unlikePost),
            args: [postId],
            invalidations: postViews,
        },
        {
            name: "useVotePoll sends the chosen option and refreshes the post holding the poll",
            useHook: () => useVotePoll(),
            variables: { postId, optionIdx: 2 },
            endpoint: vi.mocked(endpoints.votePoll),
            args: [postId, 2],
            invalidations: postViews,
        },
        {
            name: "useResolveSuggestion sends the chosen status and refreshes the feed",
            useHook: () => useResolveSuggestion(),
            variables: { id: postId, status: "rejected" },
            endpoint: vi.mocked(endpoints.resolveSuggestion),
            args: [postId, "rejected"],
            invalidations: postViews,
        },
        {
            name: "useUnresolveSuggestion reopens the id it is handed and refreshes the feed",
            useHook: () => useUnresolveSuggestion(),
            variables: postId,
            endpoint: vi.mocked(endpoints.unresolveSuggestion),
            args: [postId],
            invalidations: postViews,
        },
        {
            name: "useCreateComment sends the body and the parent under the post it was built with",
            useHook: () => useCreateComment(postId),
            variables: { body: "A fine catch", parentId: "c-parent" },
            endpoint: vi.mocked(comments.createComment),
            args: [postId, "A fine catch", "c-parent", undefined],
            invalidations: postViews,
        },
        {
            name: "useCreateComment sends the staged media in the same request",
            useHook: () => useCreateComment(postId),
            variables: { body: "", media: [{ file, isSpoiler: true }] },
            endpoint: vi.mocked(comments.createComment),
            args: [postId, "", undefined, [{ file, isSpoiler: true }]],
            invalidations: postViews,
        },
        {
            name: "useUpdateComment addresses the comment directly but refreshes its post",
            useHook: () => useUpdateComment(postId),
            variables: { commentId: "c-1", body: "Rewritten" },
            endpoint: vi.mocked(comments.updateComment),
            args: ["c-1", "Rewritten"],
            invalidations: detailOnly,
        },
        {
            name: "useDeleteComment addresses the comment directly and refreshes every view of posts",
            useHook: () => useDeleteComment(postId),
            variables: "c-1",
            endpoint: vi.mocked(comments.deleteComment),
            args: ["c-1"],
            invalidations: postViews,
        },
        {
            name: "useLikeComment likes the comment id it is handed and refreshes its post",
            useHook: () => useLikeComment(postId),
            variables: "c-1",
            endpoint: vi.mocked(comments.likeComment),
            args: ["c-1"],
            invalidations: detailOnly,
        },
        {
            name: "useUnlikeComment unlikes the comment id it is handed and refreshes its post",
            useHook: () => useUnlikeComment(postId),
            variables: "c-1",
            endpoint: vi.mocked(comments.unlikeComment),
            args: ["c-1"],
            invalidations: detailOnly,
        },
    ];

    it.each(cases)("$name", async ({ useHook, variables, endpoint, args, invalidations }) => {
        // given the hook, its variables and the endpoint call they should produce, from the table row
        const { result, invalidate } = setup(useHook);

        // when
        await act(async () => {
            await result.current.mutateAsync(variables as never);
        });

        // then
        expect(endpoint).toHaveBeenCalledWith(...args);
        expect(invalidate.mock.calls).toEqual(invalidations.map(call => [call]));
    });
});

describe("useCreatePost", () => {
    it("falls back to the general corner when the composer did not choose one", async () => {
        // given
        const { result } = setup(() => useCreatePost());

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "Happy Halloween" });
        });

        // then
        expect(endpoints.createPost).toHaveBeenCalledWith(
            "Happy Halloween",
            "general",
            undefined,
            undefined,
            undefined,
            undefined,
        );
    });

    it("passes a poll through untouched", async () => {
        // given
        const poll = { options: [{ label: "Beatrice" }, { label: "Battler" }], duration_seconds: 86400 };
        const { result } = setup(() => useCreatePost());

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "Who is the culprit", poll });
        });

        // then
        expect(endpoints.createPost).toHaveBeenCalledWith(
            "Who is the culprit",
            "general",
            poll,
            undefined,
            undefined,
            undefined,
        );
    });

    it("hands the id of the freshly created post back to the caller", async () => {
        // given
        vi.mocked(endpoints.createPost).mockResolvedValue({ id: "p-new" });
        const { result } = setup(() => useCreatePost());

        // when
        let created: { id: string } | undefined;
        await act(async () => {
            created = await result.current.mutateAsync({ body: "A new letter" });
        });

        // then
        expect(created).toEqual({ id: "p-new" });
    });

    it("sends every attachment in display order with its spoiler flag in the create request itself", async () => {
        // given
        const second = new File(["silver"], "epitaph.png", { type: "image/png" });
        const media = [
            { file, isSpoiler: true },
            { file: second, isSpoiler: false },
        ];
        const { result } = setup(() => useCreatePost());

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "Look at these", media });
        });

        // then
        expect(endpoints.createPost).toHaveBeenCalledWith(
            "Look at these",
            "general",
            undefined,
            undefined,
            undefined,
            media,
        );
        expect(endpoints.uploadPostMedia).not.toHaveBeenCalled();
    });

    it("refreshes the post lists once, only after the post and its media are saved", async () => {
        // given
        let finishCreate: () => void = () => {};
        vi.mocked(endpoints.createPost).mockImplementation(
            () =>
                new Promise(resolve => {
                    finishCreate = () => resolve({ id: postId });
                }),
        );
        const { result, invalidate } = setup(() => useCreatePost());

        // when the request carrying the image is still in flight
        let pending: Promise<{ id: string }> = Promise.resolve({ id: "" });
        act(() => {
            pending = result.current.mutateAsync({ body: "With an image", media: [{ file, isSpoiler: false }] });
        });
        await waitFor(() => expect(endpoints.createPost).toHaveBeenCalled());

        // then nothing has refreshed the feed yet
        expect(invalidate).not.toHaveBeenCalled();

        // when the request finishes
        await act(async () => {
            finishCreate();
            await pending;
        });

        // then every post view refreshes exactly once
        expect(invalidate.mock.calls).toEqual(postViews.map(call => [call]));
    });

    it("rejects the whole post when its media is refused", async () => {
        // given
        vi.mocked(endpoints.createPost).mockRejectedValue(new Error("the media room is sealed"));
        const { result, invalidate } = setup(() => useCreatePost());

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync({ body: "Two images", media: [{ file, isSpoiler: false }] });
        });

        // then
        await expect(attempt).rejects.toThrow("the media room is sealed");
        expect(invalidate).not.toHaveBeenCalled();
    });

    it("leaves the feed cache alone when the post is rejected", async () => {
        // given
        vi.mocked(endpoints.createPost).mockRejectedValue(new Error("body is empty"));
        const { result, invalidate } = setup(() => useCreatePost());

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync({ body: "" });
        });

        // then
        await expect(attempt).rejects.toThrow("body is empty");
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useCreateComment", () => {
    it("sends an undefined parent when the comment is not a reply", async () => {
        // given
        const { result } = setup(() => useCreateComment(postId));

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "Top level" });
        });

        // then
        expect(comments.createComment).toHaveBeenCalledWith(postId, "Top level", undefined, undefined);
    });
});

describe("useResolveSuggestion", () => {
    it("leaves the status undefined so the endpoint can apply its own default", async () => {
        // given
        const { result } = setup(() => useResolveSuggestion());

        // when
        await act(async () => {
            await result.current.mutateAsync({ id: postId });
        });

        // then
        expect(endpoints.resolveSuggestion).toHaveBeenCalledWith(postId, undefined);
    });
});

describe("useDeletePost", () => {
    it("deletes the id it is handed, drops that post's detail cache and refreshes everything else", async () => {
        // given
        const { result, invalidate, remove } = setup(() => useDeletePost());

        // when
        await act(async () => {
            await result.current.mutateAsync(postId);
        });

        // then
        expect(endpoints.deletePost).toHaveBeenCalledWith(postId);
        expect(remove).toHaveBeenCalledWith({ queryKey: queryKeys.post.detail(postId) });
        expect(invalidate.mock.calls).toEqual(postViews.map(call => [call]));
    });

    it("treats a 404 as an already-deleted post rather than a failure", async () => {
        // given
        vi.mocked(endpoints.deletePost).mockRejectedValue(new ApiError(404, "post not found", null));
        const { result, remove } = setup(() => useDeletePost());

        // when
        await act(async () => {
            await result.current.mutateAsync(postId);
        });

        // then
        expect(remove).toHaveBeenCalledWith({ queryKey: queryKeys.post.detail(postId) });
    });

    it("leaves the feed cache alone when the deletion is rejected", async () => {
        // given
        vi.mocked(endpoints.deletePost).mockRejectedValue(new Error("not the author"));
        const { result, invalidate } = setup(() => useDeletePost());

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync(postId);
        });

        // then
        await expect(attempt).rejects.toThrow("not the author");
        expect(invalidate).not.toHaveBeenCalled();
    });
});
