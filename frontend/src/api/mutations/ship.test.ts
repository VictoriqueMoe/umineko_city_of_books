import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { ShipCharacter } from "../../types/api";
import {
    useCreateShip,
    useCreateShipComment,
    useDeleteShip,
    useDeleteShipComment,
    useLikeShipComment,
    useUnlikeShipComment,
    useUpdateShip,
    useUpdateShipComment,
    useUploadShipCommentMedia,
    useUploadShipImageById,
    useVoteShip,
} from "./ship";

const mocks = vi.hoisted(() => ({
    createShip: vi.fn(),
    createShipComment: vi.fn(),
    deleteShip: vi.fn(),
    deleteShipComment: vi.fn(),
    likeShipComment: vi.fn(),
    unlikeShipComment: vi.fn(),
    updateShip: vi.fn(),
    updateShipComment: vi.fn(),
    uploadShipCommentMedia: vi.fn(),
    uploadShipImage: vi.fn(),
    voteShip: vi.fn(),
}));

vi.mock("../endpoints", () => mocks);

const shipId = "33333333-3333-3333-3333-333333333333";
const commentId = "44444444-4444-4444-4444-444444444444";

const characters: ShipCharacter[] = [
    { series: "umineko", character_name: "Battler", sort_order: 0 },
    { series: "umineko", character_name: "Beatrice", sort_order: 1 },
];

function setup<T>(hook: () => T) {
    const queryClient = createTestQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { result, queryClient, invalidate };
}

beforeEach(() => {
    mocks.createShip.mockResolvedValue({ id: shipId });
    mocks.createShipComment.mockResolvedValue({ id: commentId });
    mocks.deleteShip.mockResolvedValue(undefined);
    mocks.deleteShipComment.mockResolvedValue(undefined);
    mocks.likeShipComment.mockResolvedValue(undefined);
    mocks.unlikeShipComment.mockResolvedValue(undefined);
    mocks.updateShip.mockResolvedValue(undefined);
    mocks.updateShipComment.mockResolvedValue(undefined);
    mocks.uploadShipCommentMedia.mockResolvedValue({ id: 9, media_url: "/m/9.png", media_type: "image" });
    mocks.uploadShipImage.mockResolvedValue({ image_url: "/ships/33.png" });
    mocks.voteShip.mockResolvedValue(undefined);
});

describe("useCreateShip", () => {
    it("sends the whole ship payload untouched and returns the new id", async () => {
        // given
        const { result } = setup(() => useCreateShip());

        // when
        act(() => {
            result.current.mutate({ title: "Battler and Beatrice", description: "the endless game", characters });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.createShip).toHaveBeenCalledWith({
            title: "Battler and Beatrice",
            description: "the endless game",
            characters,
        });
        expect(result.current.data).toEqual({ id: shipId });
    });

    it("refreshes the cached ships once the ship exists", async () => {
        // given
        const { result, invalidate } = setup(() => useCreateShip());

        // when
        act(() => {
            result.current.mutate({ title: "Battler and Beatrice", description: "the endless game", characters });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["ship"] });
    });

    it("leaves the cached ships untouched when creation fails", async () => {
        // given
        mocks.createShip.mockRejectedValue(new Error("that pairing is forbidden"));
        const { result, invalidate } = setup(() => useCreateShip());

        // when
        act(() => {
            result.current.mutate({ title: "", description: "", characters: [] });
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useUpdateShip", () => {
    it("edits the ship it was built for and refreshes the cached ships", async () => {
        // given
        const { result, invalidate } = setup(() => useUpdateShip(shipId));

        // when
        act(() => {
            result.current.mutate({ title: "a new title", description: "a new description", characters });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.updateShip).toHaveBeenCalledWith(shipId, {
            title: "a new title",
            description: "a new description",
            characters,
        });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["ship"] });
    });
});

describe("useDeleteShip", () => {
    it("deletes the ship it is given and refreshes the cached ships", async () => {
        // given
        const { result, invalidate } = setup(() => useDeleteShip());

        // when
        act(() => {
            result.current.mutate(shipId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.deleteShip).toHaveBeenCalledWith(shipId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["ship"] });
    });

    it("leaves the cached ships untouched when the deletion is refused", async () => {
        // given
        mocks.deleteShip.mockRejectedValue(new Error("forbidden"));
        const { result, invalidate } = setup(() => useDeleteShip());

        // when
        act(() => {
            result.current.mutate(shipId);
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useUploadShipImageById", () => {
    it("uploads the file against the ship id in the payload and returns the image url", async () => {
        // given
        const { result, invalidate } = setup(() => useUploadShipImageById());
        const file = new File(["cover"], "ship.png", { type: "image/png" });

        // when
        act(() => {
            result.current.mutate({ id: shipId, file });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.uploadShipImage).toHaveBeenCalledWith(shipId, file);
        expect(result.current.data).toEqual({ image_url: "/ships/33.png" });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["ship"] });
    });
});

describe("useVoteShip", () => {
    it("votes on the ship it was built for with the value it is given", async () => {
        // given
        const { result, invalidate } = setup(() => useVoteShip(shipId));

        // when
        act(() => {
            result.current.mutate(1);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.voteShip).toHaveBeenCalledWith(shipId, 1);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["ship"] });
    });

    it("passes a retracted vote of zero straight through", async () => {
        // given
        const { result } = setup(() => useVoteShip(shipId));

        // when
        act(() => {
            result.current.mutate(0);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.voteShip).toHaveBeenCalledWith(shipId, 0);
    });
});

describe("useCreateShipComment", () => {
    it("posts a top level comment against the ship it was built for", async () => {
        // given
        const { result, invalidate } = setup(() => useCreateShipComment(shipId));

        // when
        act(() => {
            result.current.mutate({ body: "this pairing is my favourite" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.createShipComment).toHaveBeenCalledWith(shipId, "this pairing is my favourite", undefined);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["ship"] });
    });

    it("passes the parent comment through when the comment is a reply", async () => {
        // given
        const { result } = setup(() => useCreateShipComment(shipId));

        // when
        act(() => {
            result.current.mutate({ body: "agreed", parentId: commentId });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.createShipComment).toHaveBeenCalledWith(shipId, "agreed", commentId);
    });
});

describe("useUpdateShipComment", () => {
    it("edits the comment by its own id rather than the ship it belongs to", async () => {
        // given
        const { result, invalidate } = setup(() => useUpdateShipComment(shipId));

        // when
        act(() => {
            result.current.mutate({ id: commentId, body: "a tidier opinion" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.updateShipComment).toHaveBeenCalledWith(commentId, "a tidier opinion");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["ship"] });
    });
});

describe("useDeleteShipComment", () => {
    it("deletes the comment it is given and refreshes the cached ships", async () => {
        // given
        const { result, invalidate } = setup(() => useDeleteShipComment(shipId));

        // when
        act(() => {
            result.current.mutate(commentId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.deleteShipComment).toHaveBeenCalledWith(commentId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["ship"] });
    });
});

describe("useLikeShipComment", () => {
    it("likes the comment it is given and refreshes the cached ships", async () => {
        // given
        const { result, invalidate } = setup(() => useLikeShipComment(shipId));

        // when
        act(() => {
            result.current.mutate(commentId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.likeShipComment).toHaveBeenCalledWith(commentId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["ship"] });
    });
});

describe("useUnlikeShipComment", () => {
    it("removes the like from the comment it is given and refreshes the cached ships", async () => {
        // given
        const { result, invalidate } = setup(() => useUnlikeShipComment(shipId));

        // when
        act(() => {
            result.current.mutate(commentId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.unlikeShipComment).toHaveBeenCalledWith(commentId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["ship"] });
    });
});

describe("useUploadShipCommentMedia", () => {
    it("uploads the file against the comment it is given and returns the stored media", async () => {
        // given
        const { result, invalidate } = setup(() => useUploadShipCommentMedia(shipId));
        const file = new File(["fanart"], "art.png", { type: "image/png" });

        // when
        act(() => {
            result.current.mutate({ commentId, file });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.uploadShipCommentMedia).toHaveBeenCalledWith(commentId, file);
        expect(result.current.data).toEqual({ id: 9, media_url: "/m/9.png", media_type: "image" });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["ship"] });
    });

    it("leaves the cached ships untouched when the upload is rejected", async () => {
        // given
        mocks.uploadShipCommentMedia.mockRejectedValue(new Error("file too large"));
        const { result, invalidate } = setup(() => useUploadShipCommentMedia(shipId));
        const file = new File(["fanart"], "art.png", { type: "image/png" });

        // when
        act(() => {
            result.current.mutate({ commentId, file });
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(invalidate).not.toHaveBeenCalled();
    });
});
