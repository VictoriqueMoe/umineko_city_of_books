import type { QueryClient } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import {
    useCreateAnnouncementComment,
    useDeleteAnnouncementComment,
    useLikeAnnouncementComment,
    useUnlikeAnnouncementComment,
    useUpdateAnnouncementComment,
} from "./announcement";

const mocks = vi.hoisted(() => ({
    createAnnouncementComment: vi.fn(),
    deleteAnnouncementComment: vi.fn(),
    likeAnnouncementComment: vi.fn(),
    unlikeAnnouncementComment: vi.fn(),
    updateAnnouncementComment: vi.fn(),
}));

vi.mock("../../api/endpoints/comments", () => mocks);

const announcementId = "announcement-1";

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

describe("useCreateAnnouncementComment", () => {
    it("posts the comment against the announcement the hook was built for", async () => {
        // given
        const { qc, invalidate } = client();
        mocks.createAnnouncementComment.mockResolvedValue({ id: "c1" });

        // when
        const data = await runMutation(() => useCreateAnnouncementComment(announcementId), { body: "hello" }, qc);

        // then
        expect(mocks.createAnnouncementComment).toHaveBeenCalledWith(announcementId, "hello", undefined, undefined);
        expect(data).toEqual({ id: "c1" });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["announcements"] });
    });

    it("passes the parent comment through when the comment is a reply", async () => {
        // given
        const { qc } = client();

        // when
        await runMutation(() => useCreateAnnouncementComment(announcementId), { body: "replying", parentId: "c1" }, qc);

        // then
        expect(mocks.createAnnouncementComment).toHaveBeenCalledWith(announcementId, "replying", "c1", undefined);
    });

    it("sends the staged media with the comment in the same request", async () => {
        // given
        const { qc } = client();
        const media = [{ file: new File(["gif"], "beato.gif", { type: "image/gif" }), isSpoiler: true }];

        // when
        await runMutation(() => useCreateAnnouncementComment(announcementId), { body: "", media }, qc);

        // then
        expect(mocks.createAnnouncementComment).toHaveBeenCalledWith(announcementId, "", undefined, media);
    });

    it("leaves the announcement list alone when the server refuses the comment", async () => {
        // given
        const { qc, invalidate } = client();
        mocks.createAnnouncementComment.mockRejectedValue(new Error("comments are closed"));
        const { result } = renderHook(() => useCreateAnnouncementComment(announcementId), {
            wrapper: providerWrapper({ queryClient: qc }),
        });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync({ body: "hello" })).rejects.toThrow("comments are closed");
        });

        // then
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useUpdateAnnouncementComment", () => {
    it("edits the comment by its own id and ignores the announcement it was built for", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(() => useUpdateAnnouncementComment(announcementId), { id: "c1", body: "edited" }, qc);

        // then
        expect(mocks.updateAnnouncementComment).toHaveBeenCalledWith("c1", "edited");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["announcements"] });
    });
});

describe("useDeleteAnnouncementComment", () => {
    it("deletes the comment and refreshes the announcements", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(() => useDeleteAnnouncementComment(announcementId), "c1", qc);

        // then
        expect(mocks.deleteAnnouncementComment).toHaveBeenCalledWith("c1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["announcements"] });
    });
});

describe("useLikeAnnouncementComment", () => {
    it("likes the comment and refreshes the announcements", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(() => useLikeAnnouncementComment(announcementId), "c1", qc);

        // then
        expect(mocks.likeAnnouncementComment).toHaveBeenCalledWith("c1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["announcements"] });
    });
});

describe("useUnlikeAnnouncementComment", () => {
    it("unlikes the comment and refreshes the announcements", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(() => useUnlikeAnnouncementComment(announcementId), "c1", qc);

        // then
        expect(mocks.unlikeAnnouncementComment).toHaveBeenCalledWith("c1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["announcements"] });
    });
});
