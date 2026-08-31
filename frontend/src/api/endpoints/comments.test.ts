import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./comments";
import { deleteMock, lastFormData, postFormDataMock, postMock, putMock, resetTransports } from "./testHarness";

type CommentCreate = (parentId: string, body: string, replyToId?: string) => Promise<{ id: string }>;

interface CommentFamily {
    family: string;
    parentPath: string;
    commentPath: string;
    likeBody: undefined | Record<string, never>;
    create: CommentCreate | null;
    update: (id: string, body: string) => Promise<void>;
    remove: (id: string) => Promise<void>;
    like: (id: string) => Promise<void>;
    unlike: (id: string) => Promise<void>;
    uploadMedia: (commentId: string, file: File) => Promise<unknown>;
}

vi.mock("../../platform/capabilities", () => ({
    isNativeApp: () => false,
    clientPlatform: () => "web",
}));

vi.mock("../client", async importOriginal => {
    const actual = await importOriginal<typeof import("../client")>();
    return {
        ...actual,
        apiFetch: vi.fn(),
        apiFetchText: vi.fn(),
        apiPost: vi.fn(),
        apiPut: vi.fn(),
        apiPatch: vi.fn(),
        apiDelete: vi.fn(),
        apiDeleteWithBody: vi.fn(),
        apiPostFormData: vi.fn(),
    };
});

beforeEach(resetTransports);

const families: CommentFamily[] = [
    {
        family: "posts",
        parentPath: "/posts",
        commentPath: "/comments",
        likeBody: undefined,
        create: api.createComment,
        update: api.updateComment,
        remove: api.deleteComment,
        like: api.likeComment,
        unlike: api.unlikeComment,
        uploadMedia: api.uploadCommentMedia,
    },
    {
        family: "art",
        parentPath: "/art",
        commentPath: "/art-comments",
        likeBody: undefined,
        create: api.createArtComment,
        update: api.updateArtComment,
        remove: api.deleteArtComment,
        like: api.likeArtComment,
        unlike: api.unlikeArtComment,
        uploadMedia: api.uploadArtCommentMedia,
    },
    {
        family: "mysteries",
        parentPath: "/mysteries",
        commentPath: "/mystery-comments",
        likeBody: {},
        create: api.createMysteryComment,
        update: api.updateMysteryComment,
        remove: api.deleteMysteryComment,
        like: api.likeMysteryComment,
        unlike: api.unlikeMysteryComment,
        uploadMedia: api.uploadMysteryCommentMedia,
    },
    {
        family: "secrets",
        parentPath: "/secrets",
        commentPath: "/secret-comments",
        likeBody: {},
        create: api.createSecretComment,
        update: api.updateSecretComment,
        remove: api.deleteSecretComment,
        like: api.likeSecretComment,
        unlike: api.unlikeSecretComment,
        uploadMedia: api.uploadSecretCommentMedia,
    },
    {
        family: "fanfics",
        parentPath: "/fanfics",
        commentPath: "/fanfic-comments",
        likeBody: {},
        create: api.createFanficComment,
        update: api.updateFanficComment,
        remove: api.deleteFanficComment,
        like: api.likeFanficComment,
        unlike: api.unlikeFanficComment,
        uploadMedia: api.uploadFanficCommentMedia,
    },
    {
        family: "announcements",
        parentPath: "/announcements",
        commentPath: "/announcement-comments",
        likeBody: {},
        create: api.createAnnouncementComment,
        update: api.updateAnnouncementComment,
        remove: api.deleteAnnouncementComment,
        like: api.likeAnnouncementComment,
        unlike: api.unlikeAnnouncementComment,
        uploadMedia: api.uploadAnnouncementCommentMedia,
    },
    {
        family: "journals",
        parentPath: "/journals",
        commentPath: "/journal-comments",
        likeBody: {},
        create: null,
        update: api.updateJournalComment,
        remove: api.deleteJournalComment,
        like: api.likeJournalComment,
        unlike: api.unlikeJournalComment,
        uploadMedia: api.uploadJournalCommentMedia,
    },
    {
        family: "ships",
        parentPath: "/ships",
        commentPath: "/ship-comments",
        likeBody: {},
        create: api.createShipComment,
        update: api.updateShipComment,
        remove: api.deleteShipComment,
        like: api.likeShipComment,
        unlike: api.unlikeShipComment,
        uploadMedia: api.uploadShipCommentMedia,
    },
    {
        family: "ocs",
        parentPath: "/ocs",
        commentPath: "/oc-comments",
        likeBody: {},
        create: api.createOCComment,
        update: api.updateOCComment,
        remove: api.deleteOCComment,
        like: api.likeOCComment,
        unlike: api.unlikeOCComment,
        uploadMedia: api.uploadOCCommentMedia,
    },
];

const creatable = families.filter((entry): entry is CommentFamily & { create: CommentCreate } => entry.create !== null);

describe("the comment family table", () => {
    it("covers all nine comment families", () => {
        // then
        expect(families).toHaveLength(9);
        expect(families.map(entry => entry.family)).toEqual([
            "posts",
            "art",
            "mysteries",
            "secrets",
            "fanfics",
            "announcements",
            "journals",
            "ships",
            "ocs",
        ]);
    });

    it("sends no like body at all for posts and art, and an empty object for the other seven", () => {
        // then
        expect(families.filter(entry => entry.likeBody === undefined).map(entry => entry.family)).toEqual([
            "posts",
            "art",
        ]);
        expect(families.filter(entry => entry.likeBody !== undefined).map(entry => entry.family)).toEqual([
            "mysteries",
            "secrets",
            "fanfics",
            "announcements",
            "journals",
            "ships",
            "ocs",
        ]);
    });

    it("binds a create endpoint for every family except journals", () => {
        // then
        expect(families.filter(entry => entry.create === null).map(entry => entry.family)).toEqual(["journals"]);
    });
});

describe("the comment endpoint factory", () => {
    it.each(families)(
        "$family comments create, update, delete, like, unlike and upload against their own paths",
        async family => {
            // given
            const file = new File(["x"], "comment.png", { type: "image/png" });

            // when
            if (family.create) {
                await family.create("parent-1", "lovely");
            }
            await family.update("c-1", "on second thoughts");
            await family.remove("c-1");
            await family.like("c-1");
            await family.unlike("c-1");
            await family.uploadMedia("c-1", file);

            // then
            if (family.create) {
                expect(postMock.mock.calls[0]).toStrictEqual([
                    `${family.parentPath}/parent-1/comments`,
                    { body: "lovely", parent_id: undefined },
                ]);
            }
            expect(putMock.mock.lastCall).toStrictEqual([`${family.commentPath}/c-1`, { body: "on second thoughts" }]);
            expect(postMock.mock.lastCall).toStrictEqual([`${family.commentPath}/c-1/like`, family.likeBody]);
            expect(deleteMock.mock.calls).toEqual([[`${family.commentPath}/c-1`], [`${family.commentPath}/c-1/like`]]);
            expect(postFormDataMock.mock.calls[0][0]).toBe(`${family.commentPath}/c-1/media`);
            expect(lastFormData().get("media")).toBe(file);
        },
    );

    it.each(creatable)("$family comments carry the parent id of a reply", async family => {
        // when
        await family.create("parent-1", "quite so", "reply-to-1");

        // then
        expect(postMock.mock.lastCall).toStrictEqual([
            `${family.parentPath}/parent-1/comments`,
            { body: "quite so", parent_id: "reply-to-1" },
        ]);
    });
});
