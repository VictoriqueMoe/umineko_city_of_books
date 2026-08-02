import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import {
    useCreateFanfic,
    useCreateFanficChapter,
    useCreateFanficComment,
    useDeleteFanfic,
    useDeleteFanficChapter,
    useDeleteFanficComment,
    useDeleteFanficCover,
    useFavouriteFanfic,
    useLikeFanficComment,
    useUnfavouriteFanfic,
    useUnlikeFanficComment,
    useUpdateFanfic,
    useUpdateFanficChapter,
    useUpdateFanficComment,
    useUploadFanficCommentMedia,
    useUploadFanficCover,
    useUploadFanficCoverFor,
} from "./fanfic";

const mocks = vi.hoisted(() => ({
    createFanfic: vi.fn(),
    createFanficChapter: vi.fn(),
    createFanficComment: vi.fn(),
    deleteFanfic: vi.fn(),
    deleteFanficChapter: vi.fn(),
    deleteFanficComment: vi.fn(),
    deleteFanficCover: vi.fn(),
    favouriteFanfic: vi.fn(),
    likeFanficComment: vi.fn(),
    unfavouriteFanfic: vi.fn(),
    unlikeFanficComment: vi.fn(),
    updateFanfic: vi.fn(),
    updateFanficChapter: vi.fn(),
    updateFanficComment: vi.fn(),
    uploadFanficCommentMedia: vi.fn(),
    uploadFanficCover: vi.fn(),
}));

vi.mock("../endpoints", () => mocks);

const fanficKey = ["fanfic"];
const fanficId = "11111111-1111-1111-1111-111111111111";
const chapterId = "22222222-2222-2222-2222-222222222222";
const commentId = "33333333-3333-3333-3333-333333333333";

const draft = {
    title: "the golden truth",
    summary: "a tale of the sixth game",
    series: "umineko",
    rating: "teen",
    language: "english",
    is_oneshot: false,
    contains_lemons: false,
    genres: ["mystery"],
    tags: ["beatrice"],
    characters: [{ series: "umineko", character_name: "Beatrice", sort_order: 0 }],
    is_pairing: false,
};

function harness() {
    const queryClient = createTestQueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

    return { invalidateQueries, queryClient, wrapper: providerWrapper({ queryClient }) };
}

function makeFile() {
    return new File(["gold"], "cover.png", { type: "image/png" });
}

beforeEach(() => {
    for (const fn of Object.values(mocks)) {
        fn.mockResolvedValue(undefined);
    }
});

describe("useCreateFanfic", () => {
    it("sends the draft untouched and refreshes every fanfic query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.createFanfic.mockResolvedValue({ id: fanficId });
        const { result } = renderHook(() => useCreateFanfic(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(draft);
        });

        // then
        expect(mocks.createFanfic).toHaveBeenCalledWith(draft);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });

    it("hands the new fanfic id back so the caller can navigate to it", async () => {
        // given
        const { wrapper } = harness();
        mocks.createFanfic.mockResolvedValue({ id: fanficId });
        const { result } = renderHook(() => useCreateFanfic(), { wrapper });

        // when
        let created: { id: string } | undefined;
        await act(async () => {
            created = await result.current.mutateAsync({ ...draft, body: "a oneshot body" });
        });

        // then
        expect(created).toEqual({ id: fanficId });
        expect(mocks.createFanfic).toHaveBeenCalledWith({ ...draft, body: "a oneshot body" });
    });

    it("leaves the fanfic cache alone when the creation is rejected", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.createFanfic.mockRejectedValue(new Error("title is required"));
        const { result } = renderHook(() => useCreateFanfic(), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync(draft)).rejects.toThrow("title is required");
        });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useUpdateFanfic", () => {
    it("edits the fanfic the hook was built for and refreshes every fanfic query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const payload = { ...draft, status: "complete" };
        const { result } = renderHook(() => useUpdateFanfic(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(payload);
        });

        // then
        expect(mocks.updateFanfic).toHaveBeenCalledWith(fanficId, payload);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useDeleteFanfic", () => {
    it("deletes the fanfic it was handed and refreshes every fanfic query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useDeleteFanfic(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(fanficId);
        });

        // then
        expect(mocks.deleteFanfic).toHaveBeenCalledWith(fanficId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useUploadFanficCover", () => {
    it("uploads the file against the fanfic the hook was built for", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const file = makeFile();
        mocks.uploadFanficCover.mockResolvedValue({ image_url: "/media/cover.png" });
        const { result } = renderHook(() => useUploadFanficCover(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(file);
        });

        // then
        expect(mocks.uploadFanficCover).toHaveBeenCalledWith(fanficId, file);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useUploadFanficCoverFor", () => {
    it("takes the fanfic id at call time instead of at hook time", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const file = makeFile();
        const otherId = "44444444-4444-4444-4444-444444444444";
        const { result } = renderHook(() => useUploadFanficCoverFor(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ id: otherId, file });
        });

        // then
        expect(mocks.uploadFanficCover).toHaveBeenCalledWith(otherId, file);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useDeleteFanficCover", () => {
    it("deletes the cover of the fanfic the hook was built for", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useDeleteFanficCover(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync();
        });

        // then
        expect(mocks.deleteFanficCover).toHaveBeenCalledWith(fanficId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useCreateFanficChapter", () => {
    it("spreads the title and the body into positional arguments after the fanfic id", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.createFanficChapter.mockResolvedValue({ id: chapterId });
        const { result } = renderHook(() => useCreateFanficChapter(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ title: "first twilight", body: "six chosen by the key" });
        });

        // then
        expect(mocks.createFanficChapter).toHaveBeenCalledWith(fanficId, "first twilight", "six chosen by the key");
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useUpdateFanficChapter", () => {
    it("edits the chapter by its own id and ignores the fanfic id it was built with", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUpdateFanficChapter(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ chapterId, title: "second twilight", body: "the two who are close" });
        });

        // then
        expect(mocks.updateFanficChapter).toHaveBeenCalledWith(chapterId, "second twilight", "the two who are close");
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useDeleteFanficChapter", () => {
    it("deletes the chapter it was handed and refreshes every fanfic query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useDeleteFanficChapter(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(chapterId);
        });

        // then
        expect(mocks.deleteFanficChapter).toHaveBeenCalledWith(chapterId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useFavouriteFanfic", () => {
    it("favourites the fanfic it was handed and refreshes every fanfic query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useFavouriteFanfic(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(fanficId);
        });

        // then
        expect(mocks.favouriteFanfic).toHaveBeenCalledWith(fanficId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useUnfavouriteFanfic", () => {
    it("unfavourites the fanfic it was handed and refreshes every fanfic query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUnfavouriteFanfic(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(fanficId);
        });

        // then
        expect(mocks.unfavouriteFanfic).toHaveBeenCalledWith(fanficId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });

    it("leaves the fanfic cache alone when the unfavourite fails", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.unfavouriteFanfic.mockRejectedValue(new Error("not favourited"));
        const { result } = renderHook(() => useUnfavouriteFanfic(), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync(fanficId)).rejects.toThrow("not favourited");
        });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useCreateFanficComment", () => {
    it("posts a top level comment with no parent id", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.createFanficComment.mockResolvedValue({ id: commentId });
        const { result } = renderHook(() => useCreateFanficComment(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "without love it cannot be seen" });
        });

        // then
        expect(mocks.createFanficComment).toHaveBeenCalledWith(fanficId, "without love it cannot be seen", undefined);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });

    it("threads the comment under its parent when one is given", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useCreateFanficComment(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "a reply", parentId: commentId });
        });

        // then
        expect(mocks.createFanficComment).toHaveBeenCalledWith(fanficId, "a reply", commentId);
    });
});

describe("useUpdateFanficComment", () => {
    it("sends the comment id and the new body", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUpdateFanficComment(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ id: commentId, body: "edited" });
        });

        // then
        expect(mocks.updateFanficComment).toHaveBeenCalledWith(commentId, "edited");
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useDeleteFanficComment", () => {
    it("deletes the comment it was handed and refreshes every fanfic query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useDeleteFanficComment(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(commentId);
        });

        // then
        expect(mocks.deleteFanficComment).toHaveBeenCalledWith(commentId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useLikeFanficComment", () => {
    it("likes the comment it was handed and refreshes every fanfic query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useLikeFanficComment(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(commentId);
        });

        // then
        expect(mocks.likeFanficComment).toHaveBeenCalledWith(commentId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useUnlikeFanficComment", () => {
    it("unlikes the comment it was handed and refreshes every fanfic query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUnlikeFanficComment(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(commentId);
        });

        // then
        expect(mocks.unlikeFanficComment).toHaveBeenCalledWith(commentId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});

describe("useUploadFanficCommentMedia", () => {
    it("uploads the file against the comment it was given", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const file = makeFile();
        const { result } = renderHook(() => useUploadFanficCommentMedia(fanficId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ commentId, file });
        });

        // then
        expect(mocks.uploadFanficCommentMedia).toHaveBeenCalledWith(commentId, file);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: fanficKey });
    });
});
