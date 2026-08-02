import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import {
    useCreateSecretComment,
    useDeleteSecretComment,
    useLikeSecretComment,
    useUnlikeSecretComment,
    useUnlockSecret,
    useUpdateSecretComment,
    useUploadSecretCommentMedia,
} from "./secret";

const mocks = vi.hoisted(() => ({
    createSecretComment: vi.fn(),
    deleteSecretComment: vi.fn(),
    likeSecretComment: vi.fn(),
    unlikeSecretComment: vi.fn(),
    unlockSecret: vi.fn(),
    updateSecretComment: vi.fn(),
    uploadSecretCommentMedia: vi.fn(),
}));

vi.mock("../endpoints", () => mocks);

const secretId = "11111111-1111-1111-1111-111111111111";
const commentId = "22222222-2222-2222-2222-222222222222";

function setup<T>(hook: () => T) {
    const queryClient = createTestQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { result, queryClient, invalidate };
}

beforeEach(() => {
    mocks.createSecretComment.mockResolvedValue({ id: commentId });
    mocks.deleteSecretComment.mockResolvedValue(undefined);
    mocks.likeSecretComment.mockResolvedValue(undefined);
    mocks.unlikeSecretComment.mockResolvedValue(undefined);
    mocks.unlockSecret.mockResolvedValue(undefined);
    mocks.updateSecretComment.mockResolvedValue(undefined);
    mocks.uploadSecretCommentMedia.mockResolvedValue({ id: 7, media_url: "/m/7.png", media_type: "image" });
});

describe("useCreateSecretComment", () => {
    it("posts a top level comment against the secret it was built for", async () => {
        // given
        const { result } = setup(() => useCreateSecretComment(secretId));

        // when
        act(() => {
            result.current.mutate({ body: "the golden witch is watching" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.createSecretComment).toHaveBeenCalledWith(secretId, "the golden witch is watching", undefined);
        expect(result.current.data).toEqual({ id: commentId });
    });

    it("passes the parent comment through when the comment is a reply", async () => {
        // given
        const { result } = setup(() => useCreateSecretComment(secretId));

        // when
        act(() => {
            result.current.mutate({ body: "without love it cannot be seen", parentId: commentId });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.createSecretComment).toHaveBeenCalledWith(secretId, "without love it cannot be seen", commentId);
    });

    it("refreshes the cached secrets once the comment is stored", async () => {
        // given
        const { result, invalidate } = setup(() => useCreateSecretComment(secretId));

        // when
        act(() => {
            result.current.mutate({ body: "a new theory" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["secrets"] });
    });

    it("leaves the cached secrets untouched when the comment cannot be stored", async () => {
        // given
        mocks.createSecretComment.mockRejectedValue(new Error("the door is sealed"));
        const { result, invalidate } = setup(() => useCreateSecretComment(secretId));

        // when
        act(() => {
            result.current.mutate({ body: "a doomed theory" });
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useUpdateSecretComment", () => {
    it("edits the comment by its own id rather than the secret it belongs to", async () => {
        // given
        const { result } = setup(() => useUpdateSecretComment(secretId));

        // when
        act(() => {
            result.current.mutate({ id: commentId, body: "a revised confession" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.updateSecretComment).toHaveBeenCalledWith(commentId, "a revised confession");
    });

    it("refreshes the cached secrets after an edit", async () => {
        // given
        const { result, invalidate } = setup(() => useUpdateSecretComment(secretId));

        // when
        act(() => {
            result.current.mutate({ id: commentId, body: "a revised confession" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["secrets"] });
    });
});

describe("useDeleteSecretComment", () => {
    it("deletes the comment it is given and refreshes the cached secrets", async () => {
        // given
        const { result, invalidate } = setup(() => useDeleteSecretComment(secretId));

        // when
        act(() => {
            result.current.mutate(commentId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.deleteSecretComment).toHaveBeenCalledWith(commentId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["secrets"] });
    });

    it("leaves the cached secrets untouched when the deletion is refused", async () => {
        // given
        mocks.deleteSecretComment.mockRejectedValue(new Error("forbidden"));
        const { result, invalidate } = setup(() => useDeleteSecretComment(secretId));

        // when
        act(() => {
            result.current.mutate(commentId);
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useLikeSecretComment", () => {
    it("likes the comment it is given and refreshes the cached secrets", async () => {
        // given
        const { result, invalidate } = setup(() => useLikeSecretComment(secretId));

        // when
        act(() => {
            result.current.mutate(commentId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.likeSecretComment).toHaveBeenCalledWith(commentId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["secrets"] });
    });
});

describe("useUnlikeSecretComment", () => {
    it("removes the like from the comment it is given and refreshes the cached secrets", async () => {
        // given
        const { result, invalidate } = setup(() => useUnlikeSecretComment(secretId));

        // when
        act(() => {
            result.current.mutate(commentId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.unlikeSecretComment).toHaveBeenCalledWith(commentId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["secrets"] });
    });
});

describe("useUnlockSecret", () => {
    it("sends the secret and the phrase that was typed", async () => {
        // given
        const { result } = setup(() => useUnlockSecret());

        // when
        act(() => {
            result.current.mutate({ id: "beatrice", phrase: "you are the golden witch" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.unlockSecret).toHaveBeenCalledWith("beatrice", "you are the golden witch");
    });

    it("does not refresh any cached queries after unlocking", async () => {
        // given
        const { result, invalidate } = setup(() => useUnlockSecret());

        // when
        act(() => {
            result.current.mutate({ id: "beatrice", phrase: "you are the golden witch" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidate).not.toHaveBeenCalled();
    });

    it("surfaces the failure when the phrase is wrong", async () => {
        // given
        mocks.unlockSecret.mockRejectedValue(new Error("wrong phrase"));
        const { result } = setup(() => useUnlockSecret());

        // when
        act(() => {
            result.current.mutate({ id: "beatrice", phrase: "not the phrase" });
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(result.current.error).toEqual(new Error("wrong phrase"));
    });
});

describe("useUploadSecretCommentMedia", () => {
    it("uploads the file against the comment it is given and returns the stored media", async () => {
        // given
        const { result, invalidate } = setup(() => useUploadSecretCommentMedia(secretId));
        const file = new File(["portrait"], "beatrice.png", { type: "image/png" });

        // when
        act(() => {
            result.current.mutate({ commentId, file });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.uploadSecretCommentMedia).toHaveBeenCalledWith(commentId, file);
        expect(result.current.data).toEqual({ id: 7, media_url: "/m/7.png", media_type: "image" });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["secrets"] });
    });
});
