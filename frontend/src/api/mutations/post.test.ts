import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import * as endpoints from "../endpoints";
import { queryKeys } from "../queryKeys";
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
    useUploadCommentMedia,
    useUploadPostMedia,
    useUploadPostMediaById,
    useVotePoll,
} from "./post";

vi.mock("../endpoints", () => ({
    createComment: vi.fn(),
    createPost: vi.fn(),
    deleteComment: vi.fn(),
    deletePost: vi.fn(),
    deletePostMedia: vi.fn(),
    likeComment: vi.fn(),
    likePost: vi.fn(),
    resolveSuggestion: vi.fn(),
    unlikeComment: vi.fn(),
    unlikePost: vi.fn(),
    unresolveSuggestion: vi.fn(),
    updateComment: vi.fn(),
    updatePost: vi.fn(),
    uploadCommentMedia: vi.fn(),
    uploadPostMedia: vi.fn(),
    votePoll: vi.fn(),
}));

type TestQueryKey = readonly unknown[];

interface MutationCase {
    name: string;
    useHook: () => { mutateAsync: (variables: never) => Promise<unknown> };
    variables: unknown;
    endpoint: Mock;
    args: unknown[];
    keys: TestQueryKey[];
}

const postId = "p-1";
const file = new File(["gold"], "letter.png", { type: "image/png" });

function setup<T>(useHook: () => T) {
    const queryClient = createTestQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(useHook, { wrapper: providerWrapper({ queryClient }) });

    return { result, invalidate };
}

beforeEach(() => {
    for (const value of Object.values(endpoints)) {
        if (vi.isMockFunction(value)) {
            value.mockReset();
        }
    }
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
            args: ["Without love it cannot be seen", "theories", undefined, "art-3", "art"],
            keys: [queryKeys.post.all],
        },
        {
            name: "useUpdatePost refreshes both its own post and the feed",
            useHook: () => useUpdatePost(postId),
            variables: "an edited body",
            endpoint: vi.mocked(endpoints.updatePost),
            args: [postId, "an edited body"],
            keys: [queryKeys.post.detail(postId), queryKeys.post.all],
        },
        {
            name: "useDeletePost deletes the id it is handed and refreshes the feed",
            useHook: () => useDeletePost(),
            variables: postId,
            endpoint: vi.mocked(endpoints.deletePost),
            args: [postId],
            keys: [queryKeys.post.all],
        },
        {
            name: "useUploadPostMedia attaches the file to the post it was built with",
            useHook: () => useUploadPostMedia(postId),
            variables: file,
            endpoint: vi.mocked(endpoints.uploadPostMedia),
            args: [postId, file],
            keys: [queryKeys.post.detail(postId)],
        },
        {
            name: "useUploadPostMediaById refreshes the post named in its variables",
            useHook: () => useUploadPostMediaById(),
            variables: { id: "p-other", file },
            endpoint: vi.mocked(endpoints.uploadPostMedia),
            args: ["p-other", file],
            keys: [queryKeys.post.detail("p-other")],
        },
        {
            name: "useDeletePostMedia deletes a numbered media item from its own post",
            useHook: () => useDeletePostMedia(postId),
            variables: 7,
            endpoint: vi.mocked(endpoints.deletePostMedia),
            args: [postId, 7],
            keys: [queryKeys.post.detail(postId)],
        },
        {
            name: "useLikePost refreshes only the post that was liked",
            useHook: () => useLikePost(),
            variables: postId,
            endpoint: vi.mocked(endpoints.likePost),
            args: [postId],
            keys: [queryKeys.post.detail(postId)],
        },
        {
            name: "useUnlikePost refreshes only the post that was unliked",
            useHook: () => useUnlikePost(),
            variables: postId,
            endpoint: vi.mocked(endpoints.unlikePost),
            args: [postId],
            keys: [queryKeys.post.detail(postId)],
        },
        {
            name: "useVotePoll sends the chosen option and refreshes the post holding the poll",
            useHook: () => useVotePoll(),
            variables: { postId, optionIdx: 2 },
            endpoint: vi.mocked(endpoints.votePoll),
            args: [postId, 2],
            keys: [queryKeys.post.detail(postId)],
        },
        {
            name: "useResolveSuggestion sends the chosen status and refreshes the feed",
            useHook: () => useResolveSuggestion(),
            variables: { id: postId, status: "rejected" },
            endpoint: vi.mocked(endpoints.resolveSuggestion),
            args: [postId, "rejected"],
            keys: [queryKeys.post.all],
        },
        {
            name: "useUnresolveSuggestion reopens the id it is handed and refreshes the feed",
            useHook: () => useUnresolveSuggestion(),
            variables: postId,
            endpoint: vi.mocked(endpoints.unresolveSuggestion),
            args: [postId],
            keys: [queryKeys.post.all],
        },
        {
            name: "useCreateComment sends the body and the parent under the post it was built with",
            useHook: () => useCreateComment(postId),
            variables: { body: "A fine catch", parentId: "c-parent" },
            endpoint: vi.mocked(endpoints.createComment),
            args: [postId, "A fine catch", "c-parent"],
            keys: [queryKeys.post.detail(postId)],
        },
        {
            name: "useUpdateComment addresses the comment directly but refreshes its post",
            useHook: () => useUpdateComment(postId),
            variables: { commentId: "c-1", body: "Rewritten" },
            endpoint: vi.mocked(endpoints.updateComment),
            args: ["c-1", "Rewritten"],
            keys: [queryKeys.post.detail(postId)],
        },
        {
            name: "useDeleteComment addresses the comment directly but refreshes its post",
            useHook: () => useDeleteComment(postId),
            variables: "c-1",
            endpoint: vi.mocked(endpoints.deleteComment),
            args: ["c-1"],
            keys: [queryKeys.post.detail(postId)],
        },
        {
            name: "useLikeComment likes the comment id it is handed and refreshes its post",
            useHook: () => useLikeComment(postId),
            variables: "c-1",
            endpoint: vi.mocked(endpoints.likeComment),
            args: ["c-1"],
            keys: [queryKeys.post.detail(postId)],
        },
        {
            name: "useUnlikeComment unlikes the comment id it is handed and refreshes its post",
            useHook: () => useUnlikeComment(postId),
            variables: "c-1",
            endpoint: vi.mocked(endpoints.unlikeComment),
            args: ["c-1"],
            keys: [queryKeys.post.detail(postId)],
        },
        {
            name: "useUploadCommentMedia attaches the file to the given comment and refreshes its post",
            useHook: () => useUploadCommentMedia(postId),
            variables: { commentId: "c-1", file },
            endpoint: vi.mocked(endpoints.uploadCommentMedia),
            args: ["c-1", file],
            keys: [queryKeys.post.detail(postId)],
        },
    ];

    it.each(cases)("$name", async ({ useHook, variables, endpoint, args, keys }) => {
        // given the hook, its variables and the endpoint call they should produce, from the table row
        const { result, invalidate } = setup(useHook);

        // when
        await act(async () => {
            await result.current.mutateAsync(variables as never);
        });

        // then
        expect(endpoint).toHaveBeenCalledWith(...args);
        for (const key of keys) {
            expect(invalidate).toHaveBeenCalledWith({ queryKey: key });
        }
        expect(invalidate).toHaveBeenCalledTimes(keys.length);
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
        expect(endpoints.createPost).toHaveBeenCalledWith("Who is the culprit", "general", poll, undefined, undefined);
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
        expect(endpoints.createComment).toHaveBeenCalledWith(postId, "Top level", undefined);
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
