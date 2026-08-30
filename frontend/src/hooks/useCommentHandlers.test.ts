import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { type CommentFamily, type CommentOperation, useCommentHandlers } from "./useCommentHandlers";

const mocks = vi.hoisted(() => {
    function operation(result: unknown) {
        const mutateAsync = vi.fn((): Promise<unknown> => Promise.resolve(result));

        return { mutateAsync, hook: vi.fn(() => ({ mutateAsync })) };
    }

    function family(name: string) {
        return {
            create: operation({ id: `${name}-comment` }),
            update: operation(undefined),
            delete: operation(undefined),
            like: operation(undefined),
            unlike: operation(undefined),
            uploadMedia: operation({ id: `${name}-media` }),
        };
    }

    return {
        announcement: family("announcement"),
        art: family("art"),
        fanfic: family("fanfic"),
        journal: family("journal"),
        mystery: family("mystery"),
        oc: family("oc"),
        post: family("post"),
        secret: family("secret"),
        ship: family("ship"),
    };
});

vi.mock("./mutations/announcement", () => ({
    useCreateAnnouncementComment: mocks.announcement.create.hook,
    useUpdateAnnouncementComment: mocks.announcement.update.hook,
    useDeleteAnnouncementComment: mocks.announcement.delete.hook,
    useLikeAnnouncementComment: mocks.announcement.like.hook,
    useUnlikeAnnouncementComment: mocks.announcement.unlike.hook,
    useUploadAnnouncementCommentMedia: mocks.announcement.uploadMedia.hook,
}));

vi.mock("./mutations/art", () => ({
    useCreateArtComment: mocks.art.create.hook,
    useUpdateArtComment: mocks.art.update.hook,
    useDeleteArtComment: mocks.art.delete.hook,
    useLikeArtComment: mocks.art.like.hook,
    useUnlikeArtComment: mocks.art.unlike.hook,
    useUploadArtCommentMedia: mocks.art.uploadMedia.hook,
}));

vi.mock("./mutations/fanfic", () => ({
    useCreateFanficComment: mocks.fanfic.create.hook,
    useUpdateFanficComment: mocks.fanfic.update.hook,
    useDeleteFanficComment: mocks.fanfic.delete.hook,
    useLikeFanficComment: mocks.fanfic.like.hook,
    useUnlikeFanficComment: mocks.fanfic.unlike.hook,
    useUploadFanficCommentMedia: mocks.fanfic.uploadMedia.hook,
}));

vi.mock("./mutations/journal", () => ({
    useCreateJournalComment: mocks.journal.create.hook,
    useUpdateJournalComment: mocks.journal.update.hook,
    useDeleteJournalComment: mocks.journal.delete.hook,
    useLikeJournalComment: mocks.journal.like.hook,
    useUnlikeJournalComment: mocks.journal.unlike.hook,
    useUploadJournalCommentMedia: mocks.journal.uploadMedia.hook,
}));

vi.mock("./mutations/mystery", () => ({
    useCreateMysteryComment: mocks.mystery.create.hook,
    useUpdateMysteryComment: mocks.mystery.update.hook,
    useDeleteMysteryComment: mocks.mystery.delete.hook,
    useLikeMysteryComment: mocks.mystery.like.hook,
    useUnlikeMysteryComment: mocks.mystery.unlike.hook,
    useUploadMysteryCommentMedia: mocks.mystery.uploadMedia.hook,
}));

vi.mock("./mutations/oc", () => ({
    useCreateOCComment: mocks.oc.create.hook,
    useUpdateOCComment: mocks.oc.update.hook,
    useDeleteOCComment: mocks.oc.delete.hook,
    useLikeOCComment: mocks.oc.like.hook,
    useUnlikeOCComment: mocks.oc.unlike.hook,
    useUploadOCCommentMedia: mocks.oc.uploadMedia.hook,
}));

vi.mock("./mutations/post", () => ({
    useCreateComment: mocks.post.create.hook,
    useUpdateComment: mocks.post.update.hook,
    useDeleteComment: mocks.post.delete.hook,
    useLikeComment: mocks.post.like.hook,
    useUnlikeComment: mocks.post.unlike.hook,
    useUploadCommentMedia: mocks.post.uploadMedia.hook,
}));

vi.mock("./mutations/secret", () => ({
    useCreateSecretComment: mocks.secret.create.hook,
    useUpdateSecretComment: mocks.secret.update.hook,
    useDeleteSecretComment: mocks.secret.delete.hook,
    useLikeSecretComment: mocks.secret.like.hook,
    useUnlikeSecretComment: mocks.secret.unlike.hook,
    useUploadSecretCommentMedia: mocks.secret.uploadMedia.hook,
}));

vi.mock("./mutations/ship", () => ({
    useCreateShipComment: mocks.ship.create.hook,
    useUpdateShipComment: mocks.ship.update.hook,
    useDeleteShipComment: mocks.ship.delete.hook,
    useLikeShipComment: mocks.ship.like.hook,
    useUnlikeShipComment: mocks.ship.unlike.hook,
    useUploadShipCommentMedia: mocks.ship.uploadMedia.hook,
}));

const everyFamily: CommentFamily[] = [
    "announcement",
    "art",
    "fanfic",
    "journal",
    "mystery",
    "oc",
    "post",
    "secret",
    "ship",
];

const everyOperation: readonly CommentOperation[] = ["create", "update", "delete", "like", "unlike", "uploadMedia"];

function operationsOf(family: CommentFamily) {
    const set = mocks[family];

    return [set.create, set.update, set.delete, set.like, set.unlike, set.uploadMedia];
}

describe("the family registry", () => {
    it.each(everyFamily)("routes %s comments to that family's own mutation module", async family => {
        // given
        const { result } = renderHook(() => useCommentHandlers(family, "target-1", { enabled: everyOperation }));

        // when
        await act(async () => {
            await result.current.createCommentFn("target-1", "the golden truth");
            await result.current.updateFn("comment-1", "without love it cannot be seen");
            await result.current.deleteFn("comment-1");
            await result.current.likeFn("comment-1");
            await result.current.unlikeFn("comment-1");
            await result.current.uploadMediaFn("comment-1", new File(["x"], "witch.png"));
        });

        // then
        for (const own of operationsOf(family)) {
            expect(own.mutateAsync).toHaveBeenCalledTimes(1);
        }
        for (const other of everyFamily.filter(name => name !== family)) {
            for (const stranger of operationsOf(other)) {
                expect(stranger.hook).not.toHaveBeenCalled();
            }
        }
    });

    it("binds every operation of a family to the same target", () => {
        // given / when
        renderHook(() => useCommentHandlers("secret", "secret-1", { enabled: everyOperation }));

        // then
        expect(mocks.secret.create.hook).toHaveBeenCalledWith("secret-1", undefined);
        for (const rest of [mocks.secret.update, mocks.secret.delete, mocks.secret.like, mocks.secret.unlike]) {
            expect(rest.hook).toHaveBeenCalledWith("secret-1");
        }
        expect(mocks.secret.uploadMedia.hook).toHaveBeenCalledWith("secret-1");
    });
});

describe("hook order", () => {
    it("builds the whole mutation set even when no operation is enabled", () => {
        // given / when
        renderHook(() => useCommentHandlers("ship", "ship-1", { enabled: [] }));

        // then
        for (const op of operationsOf("ship")) {
            expect(op.hook).toHaveBeenCalledTimes(1);
        }
    });

    it("calls the same number of hooks after the enabled set changes between renders", () => {
        // given
        const { rerender } = renderHook(
            ({ enabled }: { enabled: CommentOperation[] }) => useCommentHandlers("ship", "ship-1", { enabled }),
            { initialProps: { enabled: [] as CommentOperation[] } },
        );
        const before = operationsOf("ship").map(op => op.hook.mock.calls.length);

        // when
        rerender({ enabled: [...everyOperation] });

        // then
        const after = operationsOf("ship").map(op => op.hook.mock.calls.length);
        expect(new Set(before).size).toBe(1);
        expect(new Set(after).size).toBe(1);
        expect(after[0]).toBeGreaterThan(before[0]);
    });
});

describe("the enabled set", () => {
    it("hands back only the operations the caller asked for", () => {
        // given / when
        const { result } = renderHook(() => useCommentHandlers("ship", "ship-1", { enabled: ["create"] }));

        // then
        expect(typeof result.current.createCommentFn).toBe("function");
        expect(result.current.updateFn).toBeUndefined();
        expect(result.current.deleteFn).toBeUndefined();
        expect(result.current.likeFn).toBeUndefined();
        expect(result.current.unlikeFn).toBeUndefined();
        expect(result.current.uploadMediaFn).toBeUndefined();
    });

    it("hands back every operation when the caller asks for all six", () => {
        // given / when
        const { result } = renderHook(() => useCommentHandlers("post", "post-1", { enabled: everyOperation }));

        // then
        for (const handler of Object.values(result.current)) {
            expect(typeof handler).toBe("function");
        }
    });

    it("hands back nothing at all when the enabled set is empty", () => {
        // given / when
        const { result } = renderHook(() => useCommentHandlers("oc", "oc-1", { enabled: [] }));

        // then
        for (const handler of Object.values(result.current)) {
            expect(handler).toBeUndefined();
        }
    });
});

describe("createCommentFn", () => {
    it("ignores the target id it is handed and sends the body and the parent id", async () => {
        // given
        const { result } = renderHook(() => useCommentHandlers("mystery", "mystery-1", { enabled: ["create"] }));

        // when
        let created: { id: string } | undefined;
        await act(async () => {
            created = await result.current.createCommentFn("ignored", "the golden truth", "comment-1");
        });

        // then
        expect(mocks.mystery.create.mutateAsync).toHaveBeenCalledWith({
            body: "the golden truth",
            parentId: "comment-1",
        });
        expect(created).toEqual({ id: "mystery-comment" });
    });

    it("forwards the journal entry id so an entry comment stays on its entry", () => {
        // given / when
        renderHook(() => useCommentHandlers("journal", "journal-1", { enabled: ["create"], entryId: "entry-7" }));

        // then
        expect(mocks.journal.create.hook).toHaveBeenCalledWith("journal-1", "entry-7");
    });
});

describe("updateFn", () => {
    it("sends the comment id under both key spellings the mutation modules use", async () => {
        // given
        const { result } = renderHook(() => useCommentHandlers("post", "post-1", { enabled: ["update"] }));

        // when
        let resolved: unknown = "unset";
        await act(async () => {
            resolved = await result.current.updateFn("comment-1", "without love it cannot be seen");
        });

        // then
        expect(mocks.post.update.mutateAsync).toHaveBeenCalledWith({
            id: "comment-1",
            commentId: "comment-1",
            body: "without love it cannot be seen",
        });
        expect(resolved).toBeUndefined();
    });
});

describe("deleteFn, likeFn and unlikeFn", () => {
    it("sends the bare comment id", async () => {
        // given
        const { result } = renderHook(() =>
            useCommentHandlers("art", "art-1", { enabled: ["delete", "like", "unlike"] }),
        );

        // when
        await act(async () => {
            await result.current.deleteFn("comment-1");
            await result.current.likeFn("comment-2");
            await result.current.unlikeFn("comment-3");
        });

        // then
        expect(mocks.art.delete.mutateAsync).toHaveBeenCalledWith("comment-1");
        expect(mocks.art.like.mutateAsync).toHaveBeenCalledWith("comment-2");
        expect(mocks.art.unlike.mutateAsync).toHaveBeenCalledWith("comment-3");
    });
});

describe("uploadMediaFn", () => {
    it("sends the comment id and the file", async () => {
        // given
        const file = new File(["x"], "witch.png");
        const { result } = renderHook(() => useCommentHandlers("fanfic", "fanfic-1", { enabled: ["uploadMedia"] }));

        // when
        let uploaded: unknown;
        await act(async () => {
            uploaded = await result.current.uploadMediaFn("comment-1", file);
        });

        // then
        expect(mocks.fanfic.uploadMedia.mutateAsync).toHaveBeenCalledWith({ commentId: "comment-1", file });
        expect(uploaded).toEqual({ id: "fanfic-media" });
    });
});
