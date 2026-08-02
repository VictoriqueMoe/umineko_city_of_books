import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { UserProfile } from "../../types/api";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { blockUser, createReport, followUser, unblockUser, unfollowUser } from "../endpoints";
import { useBlockUser, useCreateReport, useFollowUser, useUnblockUser, useUnfollowUser } from "./misc";

vi.mock("../endpoints", () => ({
    blockUser: vi.fn(),
    createReport: vi.fn(),
    followUser: vi.fn(),
    unblockUser: vi.fn(),
    unfollowUser: vi.fn(),
}));

const blockUserMock = vi.mocked(blockUser);
const createReportMock = vi.mocked(createReport);
const followUserMock = vi.mocked(followUser);
const unblockUserMock = vi.mocked(unblockUser);
const unfollowUserMock = vi.mocked(unfollowUser);

const targetId = "11111111-1111-1111-1111-111111111111";
const signedInUser = makeUser({ id: "99999999-9999-9999-9999-999999999999" });

function setup<T>(hook: () => T, user: UserProfile | null = null) {
    const queryClient = createTestQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(hook, { wrapper: providerWrapper({ queryClient, user }) });

    return { result, invalidate };
}

beforeEach(() => {
    blockUserMock.mockResolvedValue(undefined);
    createReportMock.mockResolvedValue(undefined);
    followUserMock.mockResolvedValue(undefined);
    unblockUserMock.mockResolvedValue(undefined);
    unfollowUserMock.mockResolvedValue(undefined);
});

describe("useFollowUser", () => {
    it("follows the user it is given and refreshes their follow stats and profile", async () => {
        // given
        const { result, invalidate } = setup(() => useFollowUser(), signedInUser);

        // when
        await act(async () => {
            await result.current.mutateAsync(targetId);
        });

        // then
        expect(followUserMock).toHaveBeenCalledWith(targetId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["follow-stats", targetId] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["users", targetId] });
        expect(invalidate).toHaveBeenCalledTimes(2);
    });

    it("leaves every cache alone when the follow request is rejected", async () => {
        // given
        followUserMock.mockRejectedValue(new Error("the golden land is closed"));
        const { result, invalidate } = setup(() => useFollowUser(), signedInUser);

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync(targetId);
        });

        // then
        await expect(attempt).rejects.toThrow("the golden land is closed");
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useUnfollowUser", () => {
    it("unfollows the user it is given and refreshes their follow stats and profile", async () => {
        // given
        const { result, invalidate } = setup(() => useUnfollowUser(), signedInUser);

        // when
        await act(async () => {
            await result.current.mutateAsync(targetId);
        });

        // then
        expect(unfollowUserMock).toHaveBeenCalledWith(targetId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["follow-stats", targetId] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["users", targetId] });
        expect(invalidate).toHaveBeenCalledTimes(2);
    });

    it("leaves every cache alone when the unfollow request is rejected", async () => {
        // given
        unfollowUserMock.mockRejectedValue(new Error("no such witch"));
        const { result, invalidate } = setup(() => useUnfollowUser(), signedInUser);

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync(targetId);
        });

        // then
        await expect(attempt).rejects.toThrow("no such witch");
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useBlockUser", () => {
    it("blocks the target and refreshes both the block status and the blocked list of the viewer", async () => {
        // given
        const { result, invalidate } = setup(() => useBlockUser(), signedInUser);

        // when
        await act(async () => {
            await result.current.mutateAsync(targetId);
        });

        // then
        expect(blockUserMock).toHaveBeenCalledWith(targetId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["block-status", targetId] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["profile", signedInUser.id, "blocked"] });
        expect(invalidate).toHaveBeenCalledTimes(2);
    });

    it("falls back to an empty owner id in the blocked list key when nobody is signed in", async () => {
        // given
        const { result, invalidate } = setup(() => useBlockUser(), null);

        // when
        await act(async () => {
            await result.current.mutateAsync(targetId);
        });

        // then
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["profile", "", "blocked"] });
    });

    it("leaves every cache alone when the block request is rejected", async () => {
        // given
        blockUserMock.mockRejectedValue(new Error("blocked already"));
        const { result, invalidate } = setup(() => useBlockUser(), signedInUser);

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync(targetId);
        });

        // then
        await expect(attempt).rejects.toThrow("blocked already");
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useUnblockUser", () => {
    it("unblocks the target and refreshes both the block status and the blocked list of the viewer", async () => {
        // given
        const { result, invalidate } = setup(() => useUnblockUser(), signedInUser);

        // when
        await act(async () => {
            await result.current.mutateAsync(targetId);
        });

        // then
        expect(unblockUserMock).toHaveBeenCalledWith(targetId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["block-status", targetId] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["profile", signedInUser.id, "blocked"] });
        expect(invalidate).toHaveBeenCalledTimes(2);
    });

    it("falls back to an empty owner id in the blocked list key when nobody is signed in", async () => {
        // given
        const { result, invalidate } = setup(() => useUnblockUser(), null);

        // when
        await act(async () => {
            await result.current.mutateAsync(targetId);
        });

        // then
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["profile", "", "blocked"] });
    });
});

describe("useCreateReport", () => {
    it("forwards the target, the reason and the context of the report in order", async () => {
        // given
        const { result } = setup(() => useCreateReport(), signedInUser);

        // when
        await act(async () => {
            await result.current.mutateAsync({
                targetType: "post",
                targetId: "p-1",
                reason: "unfavourable behaviour",
                contextId: "room-7",
            });
        });

        // then
        expect(createReportMock).toHaveBeenCalledWith("post", "p-1", "unfavourable behaviour", "room-7");
    });

    it("passes an undefined context when the report has none", async () => {
        // given
        const { result } = setup(() => useCreateReport(), signedInUser);

        // when
        await act(async () => {
            await result.current.mutateAsync({ targetType: "user", targetId: "u-2", reason: "spam" });
        });

        // then
        expect(createReportMock).toHaveBeenCalledWith("user", "u-2", "spam", undefined);
    });

    it("invalidates nothing because a report changes nothing the viewer can see", async () => {
        // given
        const { result, invalidate } = setup(() => useCreateReport(), signedInUser);

        // when
        await act(async () => {
            await result.current.mutateAsync({ targetType: "post", targetId: "p-1", reason: "spam" });
        });

        // then
        expect(invalidate).not.toHaveBeenCalled();
    });

    it("surfaces a rejected report to the caller", async () => {
        // given
        createReportMock.mockRejectedValue(new Error("report limit reached"));
        const { result } = setup(() => useCreateReport(), signedInUser);

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync({ targetType: "post", targetId: "p-1", reason: "spam" });
        });

        // then
        await expect(attempt).rejects.toThrow("report limit reached");
    });
});
