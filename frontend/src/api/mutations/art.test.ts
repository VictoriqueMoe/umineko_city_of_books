import type { QueryClient } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import {
    useCreateArt,
    useCreateArtComment,
    useCreateGallery,
    useDeleteArt,
    useDeleteArtComment,
    useDeleteGallery,
    useLikeArt,
    useLikeArtComment,
    useSetArtGallery,
    useSetGalleryCover,
    useUnlikeArt,
    useUnlikeArtComment,
    useUpdateArt,
    useUpdateArtComment,
    useUpdateGallery,
    useUploadArtCommentMedia,
} from "./art";

const mocks = vi.hoisted(() => ({
    createArt: vi.fn(),
    createArtComment: vi.fn(),
    createGallery: vi.fn(),
    deleteArt: vi.fn(),
    deleteArtComment: vi.fn(),
    deleteGallery: vi.fn(),
    likeArt: vi.fn(),
    likeArtComment: vi.fn(),
    setArtGallery: vi.fn(),
    setGalleryCover: vi.fn(),
    unlikeArt: vi.fn(),
    unlikeArtComment: vi.fn(),
    updateArt: vi.fn(),
    updateArtComment: vi.fn(),
    updateGallery: vi.fn(),
    uploadArtCommentMedia: vi.fn(),
}));

vi.mock("../endpoints", () => mocks);

const artId = "art-1";
const galleriesKey = ["galleries"];
const galleryKey = ["gallery"];

function client() {
    const qc = createTestQueryClient();

    return { qc, invalidate: vi.spyOn(qc, "invalidateQueries") };
}

async function runMutation<V>(
    use: () => { mutateAsync: (variables: V) => Promise<unknown> },
    variables: V,
    qc: QueryClient,
): Promise<unknown> {
    const { result } = renderHook(use, { wrapper: providerWrapper({ queryClient: qc }) });

    let data: unknown;
    await act(async () => {
        data = await result.current.mutateAsync(variables);
    });

    return data;
}

beforeEach(() => {
    for (const fn of Object.values(mocks)) {
        fn.mockResolvedValue(undefined);
    }
});

describe("art piece mutations", () => {
    const metadata = {
        title: "Golden Witch",
        description: "a tea party",
        corner: "umineko",
        art_type: "digital",
        tags: ["beatrice"],
        is_spoiler: false,
    };

    it("splits the upload into metadata and image file and refreshes every art view", async () => {
        // given
        const { qc, invalidate } = client();
        const imageFile = new File(["png"], "beato.png", { type: "image/png" });
        mocks.createArt.mockResolvedValue({ id: artId });

        // when
        const data = await runMutation(useCreateArt, { metadata, imageFile }, qc);

        // then
        expect(mocks.createArt).toHaveBeenCalledWith(metadata, imageFile);
        expect(data).toEqual({ id: artId });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art"] });
    });

    it("refreshes the gallery views too because a new piece can land inside a gallery", async () => {
        // given
        const { qc, invalidate } = client();
        const imageFile = new File(["png"], "beato.png", { type: "image/png" });

        // when
        await runMutation(useCreateArt, { metadata: { ...metadata, gallery_id: "gallery-1" }, imageFile }, qc);

        // then
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleriesKey });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleryKey });
    });

    it("edits the piece the hook was built for", async () => {
        // given
        const { qc, invalidate } = client();
        const update = { title: "Golden Witch", description: "edited", tags: ["beatrice"], is_spoiler: true };

        // when
        await runMutation(() => useUpdateArt(artId), update, qc);

        // then
        expect(mocks.updateArt).toHaveBeenCalledWith(artId, update);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art"] });
    });

    it("leaves the cache untouched when the edit is rejected", async () => {
        // given
        const { qc, invalidate } = client();
        mocks.updateArt.mockRejectedValue(new Error("not yours"));
        const { result } = renderHook(() => useUpdateArt(artId), { wrapper: providerWrapper({ queryClient: qc }) });

        // when
        await act(async () => {
            await expect(
                result.current.mutateAsync({ title: "x", description: "", tags: [], is_spoiler: false }),
            ).rejects.toThrow("not yours");
        });

        // then
        expect(invalidate).not.toHaveBeenCalled();
    });

    it("deletes the piece it is given rather than one fixed at hook creation", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useDeleteArt, artId, qc);

        // then
        expect(mocks.deleteArt).toHaveBeenCalledWith(artId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleriesKey });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleryKey });
    });

    it("refreshes only the liked piece rather than the whole gallery", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useLikeArt, artId, qc);

        // then
        expect(mocks.likeArt).toHaveBeenCalledWith(artId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art", "detail", artId] });
        expect(invalidate).not.toHaveBeenCalledWith({ queryKey: ["art"] });
    });

    it("refreshes only the unliked piece", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUnlikeArt, artId, qc);

        // then
        expect(mocks.unlikeArt).toHaveBeenCalledWith(artId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art", "detail", artId] });
    });
});

describe("art comment mutations", () => {
    it("posts a top level comment with no parent", async () => {
        // given
        const { qc, invalidate } = client();
        mocks.createArtComment.mockResolvedValue({ id: "c1" });

        // when
        const data = await runMutation(() => useCreateArtComment(artId), { body: "lovely" }, qc);

        // then
        expect(mocks.createArtComment).toHaveBeenCalledWith(artId, "lovely", undefined);
        expect(data).toEqual({ id: "c1" });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art", "detail", artId] });
    });

    it("posts a reply against the parent comment it was given", async () => {
        // given
        const { qc } = client();

        // when
        await runMutation(() => useCreateArtComment(artId), { body: "agreed", parentId: "c1" }, qc);

        // then
        expect(mocks.createArtComment).toHaveBeenCalledWith(artId, "agreed", "c1");
    });

    it("edits a comment by its own id while refreshing the piece it belongs to", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(() => useUpdateArtComment(artId), { commentId: "c1", body: "edited" }, qc);

        // then
        expect(mocks.updateArtComment).toHaveBeenCalledWith("c1", "edited");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art", "detail", artId] });
    });

    it("deletes a comment by its own id", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(() => useDeleteArtComment(artId), "c1", qc);

        // then
        expect(mocks.deleteArtComment).toHaveBeenCalledWith("c1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art", "detail", artId] });
    });

    it("likes a comment", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(() => useLikeArtComment(artId), "c1", qc);

        // then
        expect(mocks.likeArtComment).toHaveBeenCalledWith("c1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art", "detail", artId] });
    });

    it("unlikes a comment", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(() => useUnlikeArtComment(artId), "c1", qc);

        // then
        expect(mocks.unlikeArtComment).toHaveBeenCalledWith("c1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art", "detail", artId] });
    });

    it("attaches media to a comment and returns the stored media", async () => {
        // given
        const { qc, invalidate } = client();
        const file = new File(["gif"], "reaction.gif", { type: "image/gif" });
        mocks.uploadArtCommentMedia.mockResolvedValue({ id: "m1", url: "/uploads/reaction.gif" });

        // when
        const data = await runMutation(() => useUploadArtCommentMedia(artId), { commentId: "c1", file }, qc);

        // then
        expect(mocks.uploadArtCommentMedia).toHaveBeenCalledWith("c1", file);
        expect(data).toEqual({ id: "m1", url: "/uploads/reaction.gif" });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art", "detail", artId] });
    });
});

describe("gallery mutations", () => {
    it("creates a gallery with an empty description when none was typed", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useCreateGallery, { name: "Witches" }, qc);

        // then
        expect(mocks.createGallery).toHaveBeenCalledWith("Witches", "");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleriesKey });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleryKey });
    });

    it("marks the cached gallery list stale so a freshly created gallery shows up", async () => {
        // given
        const { qc } = client();
        qc.setQueryDefaults(["galleries"], { gcTime: Infinity });
        qc.setQueryData(["galleries", "all", ""], []);

        // when
        await runMutation(useCreateGallery, { name: "Witches" }, qc);

        // then
        expect(qc.getQueryState(["galleries", "all", ""])?.isInvalidated).toBe(true);
    });

    it("creates a gallery with the description it was given", async () => {
        // given
        const { qc } = client();

        // when
        await runMutation(useCreateGallery, { name: "Witches", description: "all of them" }, qc);

        // then
        expect(mocks.createGallery).toHaveBeenCalledWith("Witches", "all of them");
    });

    it("renames the gallery the hook was built for and blanks a missing description", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(() => useUpdateGallery("gallery-1"), { name: "Sorcerers" }, qc);

        // then
        expect(mocks.updateGallery).toHaveBeenCalledWith("gallery-1", "Sorcerers", "");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleriesKey });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleryKey });
    });

    it("deletes a gallery by id", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useDeleteGallery, "gallery-1", qc);

        // then
        expect(mocks.deleteGallery).toHaveBeenCalledWith("gallery-1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleriesKey });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleryKey });
    });

    it("marks the cached gallery list stale so a deleted gallery stops being rendered", async () => {
        // given
        const { qc } = client();
        qc.setQueryDefaults(["galleries"], { gcTime: Infinity });
        qc.setQueryData(["galleries", "all", "umineko"], [{ id: "gallery-1" }]);

        // when
        await runMutation(useDeleteGallery, "gallery-1", qc);

        // then
        expect(qc.getQueryState(["galleries", "all", "umineko"])?.isInvalidated).toBe(true);
    });

    it("sets the cover of the gallery the hook was built for to the piece it is given", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(() => useSetGalleryCover("gallery-1"), artId, qc);

        // then
        expect(mocks.setGalleryCover).toHaveBeenCalledWith("gallery-1", artId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleriesKey });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleryKey });
    });

    it("moves a piece into a gallery", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useSetArtGallery, { artId, galleryId: "gallery-1" }, qc);

        // then
        expect(mocks.setArtGallery).toHaveBeenCalledWith(artId, "gallery-1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["art"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleriesKey });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: galleryKey });
    });

    it("marks the cached gallery detail stale when a piece is moved into it", async () => {
        // given
        const { qc } = client();
        qc.setQueryDefaults(["gallery"], { gcTime: Infinity });
        qc.setQueryData(["gallery", "gallery-1", { limit: 24, offset: 0 }], { gallery: { id: "gallery-1" }, art: [] });

        // when
        await runMutation(useSetArtGallery, { artId, galleryId: "gallery-1" }, qc);

        // then
        expect(qc.getQueryState(["gallery", "gallery-1", { limit: 24, offset: 0 }])?.isInvalidated).toBe(true);
    });

    it("takes a piece out of every gallery when the gallery is null", async () => {
        // given
        const { qc } = client();

        // when
        await runMutation(useSetArtGallery, { artId, galleryId: null }, qc);

        // then
        expect(mocks.setArtGallery).toHaveBeenCalledWith(artId, null);
    });
});
