import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { providerWrapper } from "../test-utils/render";
import { useFollow } from "./useFollow";

const mocks = vi.hoisted(() => ({
    useFollowStats: vi.fn(),
    follow: vi.fn(),
    unfollow: vi.fn(),
    refresh: vi.fn(),
}));

vi.mock("../api/queries/misc", () => ({
    useFollowStats: mocks.useFollowStats,
}));

vi.mock("../api/mutations/misc", () => ({
    useFollowUser: () => ({ mutateAsync: mocks.follow }),
    useUnfollowUser: () => ({ mutateAsync: mocks.unfollow }),
}));

const userId = "22222222-2222-2222-2222-222222222222";

function statsResult(isFollowing: boolean, loading = false) {
    return {
        stats: { follower_count: 3, following_count: 1, is_following: isFollowing, follows_you: false },
        loading,
        refresh: mocks.refresh,
    };
}

function setup(id: string = userId) {
    return renderHook(() => useFollow(id), { wrapper: providerWrapper() });
}

beforeEach(() => {
    mocks.follow.mockResolvedValue(undefined);
    mocks.unfollow.mockResolvedValue(undefined);
    mocks.refresh.mockResolvedValue(undefined);
    mocks.useFollowStats.mockReturnValue(statsResult(false));
});

describe("useFollow", () => {
    it("asks for the follow stats of the user it was given", () => {
        // given
        mocks.useFollowStats.mockReturnValue(statsResult(true, true));

        // when
        const { result } = setup();

        // then
        expect(mocks.useFollowStats).toHaveBeenCalledWith(userId);
        expect(result.current.stats).toEqual({
            follower_count: 3,
            following_count: 1,
            is_following: true,
            follows_you: false,
        });
        expect(result.current.loading).toBe(true);
    });

    it("follows the user when they are not followed yet", async () => {
        // given
        mocks.useFollowStats.mockReturnValue(statsResult(false));
        const { result } = setup();

        // when
        await act(async () => {
            await result.current.toggleFollow();
        });

        // then
        expect(mocks.follow).toHaveBeenCalledWith(userId);
        expect(mocks.unfollow).not.toHaveBeenCalled();
    });

    it("unfollows the user when they are already followed", async () => {
        // given
        mocks.useFollowStats.mockReturnValue(statsResult(true));
        const { result } = setup();

        // when
        await act(async () => {
            await result.current.toggleFollow();
        });

        // then
        expect(mocks.unfollow).toHaveBeenCalledWith(userId);
        expect(mocks.follow).not.toHaveBeenCalled();
    });

    it("refreshes the stats only after the mutation has settled", async () => {
        // given
        mocks.useFollowStats.mockReturnValue(statsResult(false));
        let release: () => void = () => {};
        mocks.follow.mockImplementation(
            () =>
                new Promise<void>(resolve => {
                    release = resolve;
                }),
        );
        const { result } = setup();

        // when
        let pending: Promise<void> = Promise.resolve();
        act(() => {
            pending = result.current.toggleFollow();
        });

        // then
        expect(mocks.refresh).not.toHaveBeenCalled();
        await act(async () => {
            release();
            await pending;
        });
        expect(mocks.refresh).toHaveBeenCalledOnce();
    });

    it("does nothing when it has no user id to act on", async () => {
        // given
        const { result } = setup("");

        // when
        await act(async () => {
            await result.current.toggleFollow();
        });

        // then
        expect(mocks.follow).not.toHaveBeenCalled();
        expect(mocks.unfollow).not.toHaveBeenCalled();
        expect(mocks.refresh).not.toHaveBeenCalled();
    });

    it("does nothing while the stats are still unknown", async () => {
        // given
        mocks.useFollowStats.mockReturnValue({ stats: null, loading: true, refresh: mocks.refresh });
        const { result } = setup();

        // when
        await act(async () => {
            await result.current.toggleFollow();
        });

        // then
        expect(mocks.follow).not.toHaveBeenCalled();
        expect(mocks.refresh).not.toHaveBeenCalled();
    });

    it("swallows a failed follow and leaves the stats unrefreshed", async () => {
        // given
        mocks.useFollowStats.mockReturnValue(statsResult(false));
        mocks.follow.mockRejectedValue(new Error("the golden land is closed"));
        const { result } = setup();

        // when
        await act(async () => {
            await expect(result.current.toggleFollow()).resolves.toBeUndefined();
        });

        // then
        expect(mocks.follow).toHaveBeenCalledWith(userId);
        expect(mocks.refresh).not.toHaveBeenCalled();
    });
});
