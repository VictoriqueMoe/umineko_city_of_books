import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import type { UserProfile } from "../types/api";
import { useProfilePage } from "./useProfilePage";

const mocks = vi.hoisted(() => ({
    useProfile: vi.fn(),
    useTheoryFeed: vi.fn(),
    useFollow: vi.fn(),
    useBlock: vi.fn(),
    useUserPosts: vi.fn(),
    useUserArt: vi.fn(),
    useUserGalleries: vi.fn(),
    useUserShips: vi.fn(),
    useUserMysteries: vi.fn(),
    useUserFanfics: vi.fn(),
    useUserFanficFavourites: vi.fn(),
    useUserJournals: vi.fn(),
    useUserFollowedJournals: vi.fn(),
    useUserActivity: vi.fn(),
    useUserOCs: vi.fn(),
    useFollowers: vi.fn(),
    useFollowing: vi.fn(),
}));

vi.mock("./queries/profile", () => ({ useProfile: mocks.useProfile }));
vi.mock("./queries/theory", () => ({ useTheoryFeed: mocks.useTheoryFeed }));
vi.mock("./useFollow", () => ({ useFollow: mocks.useFollow }));
vi.mock("./useBlock", () => ({ useBlock: mocks.useBlock }));
vi.mock("./queries/user", () => ({
    useUserPosts: mocks.useUserPosts,
    useUserArt: mocks.useUserArt,
    useUserGalleries: mocks.useUserGalleries,
    useUserShips: mocks.useUserShips,
    useUserMysteries: mocks.useUserMysteries,
    useUserFanfics: mocks.useUserFanfics,
    useUserFanficFavourites: mocks.useUserFanficFavourites,
    useUserJournals: mocks.useUserJournals,
    useUserFollowedJournals: mocks.useUserFollowedJournals,
    useUserActivity: mocks.useUserActivity,
    useFollowers: mocks.useFollowers,
    useFollowing: mocks.useFollowing,
}));
vi.mock("./queries/oc", () => ({ useUserOCs: mocks.useUserOCs }));

const profileId = "profile-1";
const viewerId = "viewer-1";

function makeProfile(overrides: Partial<UserProfile> = {}): UserProfile {
    return makeUser({ id: profileId, username: "beatrice", display_name: "Beatrice", ...overrides });
}

function lastTheoryCall() {
    const calls = mocks.useTheoryFeed.mock.calls;
    return calls[calls.length - 1][0] as { authorId?: string; offset: number; limit: number; enabled: boolean };
}

function render(viewer: UserProfile | null = makeUser({ id: viewerId })) {
    return renderHook(() => useProfilePage("beatrice"), { wrapper: providerWrapper({ user: viewer }) });
}

beforeEach(() => {
    mocks.useProfile.mockReturnValue({ profile: makeProfile(), loading: false });
    mocks.useTheoryFeed.mockReturnValue({ theories: [], total: 0, loading: false, refresh: vi.fn() });
    mocks.useFollow.mockReturnValue({ stats: null, loading: false, toggleFollow: vi.fn() });
    mocks.useBlock.mockReturnValue({ status: null, loading: false, toggleBlock: vi.fn() });
    mocks.useUserPosts.mockReturnValue({ posts: [], total: 0, loading: false, refresh: vi.fn() });
    mocks.useUserArt.mockReturnValue({ art: [], total: 0, loading: false });
    mocks.useUserGalleries.mockReturnValue({ galleries: [], loading: false, refresh: vi.fn() });
    mocks.useUserShips.mockReturnValue({ ships: [], total: 0, loading: false });
    mocks.useUserMysteries.mockReturnValue({ mysteries: [], total: 0, loading: false });
    mocks.useUserFanfics.mockReturnValue({ fanfics: [], total: 0, loading: false });
    mocks.useUserFanficFavourites.mockReturnValue({ fanfics: [], total: 0, loading: false });
    mocks.useUserJournals.mockReturnValue({ journals: [], total: 0, loading: false });
    mocks.useUserFollowedJournals.mockReturnValue({ journals: [], total: 0, loading: false });
    mocks.useUserActivity.mockReturnValue({ activity: [], total: 0, loading: false });
    mocks.useUserOCs.mockReturnValue({ ocs: [], total: 0, loading: false });
    mocks.useFollowers.mockReturnValue({ users: [], total: 0, loading: false });
    mocks.useFollowing.mockReturnValue({ users: [], total: 0, loading: false });
});

describe("useProfilePage tabs", () => {
    it("looks the profile up by the username it was given", () => {
        // when
        render();

        // then
        expect(mocks.useProfile).toHaveBeenCalledWith("beatrice");
    });

    it("opens on posts when the player named no default tab", () => {
        // when
        const { result } = render();

        // then
        expect(result.current.activeTab).toBe("posts");
        expect(mocks.useUserPosts).toHaveBeenLastCalledWith(profileId, 20, 0);
    });

    it("opens on the tab the player chose as their default", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ private: { default_profile_tab: "mysteries" } }),
            loading: false,
        });

        // when
        const { result } = render();

        // then
        expect(result.current.activeTab).toBe("mysteries");
        expect(mocks.useUserMysteries).toHaveBeenLastCalledWith(profileId, 20, 0);
    });

    it("lets an explicit choice override the player's default tab", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ private: { default_profile_tab: "mysteries" } }),
            loading: false,
        });
        const { result } = render();

        // when
        act(() => result.current.setActiveTab("ships"));

        // then
        expect(result.current.activeTab).toBe("ships");
        expect(mocks.useUserShips).toHaveBeenLastCalledWith(profileId, 20, 0);
        expect(mocks.useUserMysteries).toHaveBeenLastCalledWith("", 20, 0);
    });

    it("asks nothing of the tabs the viewer is not looking at", () => {
        // when
        render();

        // then
        expect(mocks.useUserArt).toHaveBeenLastCalledWith("", 24, 0);
        expect(mocks.useUserGalleries).toHaveBeenLastCalledWith("");
        expect(mocks.useUserShips).toHaveBeenLastCalledWith("", 20, 0);
        expect(mocks.useUserOCs).toHaveBeenLastCalledWith("");
        expect(mocks.useUserMysteries).toHaveBeenLastCalledWith("", 20, 0);
        expect(mocks.useUserFanfics).toHaveBeenLastCalledWith("", 20, 0);
        expect(mocks.useUserFanficFavourites).toHaveBeenLastCalledWith("", 20, 0);
        expect(mocks.useUserJournals).toHaveBeenLastCalledWith("", 20, 0);
        expect(mocks.useUserFollowedJournals).toHaveBeenLastCalledWith("", 20, 0);
        expect(mocks.useUserActivity).toHaveBeenLastCalledWith("", 20, 0);
        expect(mocks.useFollowers).toHaveBeenLastCalledWith("");
        expect(mocks.useFollowing).toHaveBeenLastCalledWith("");
    });

    it("keeps the theories query switched off until the theories tab is opened", () => {
        // when
        const { result } = render();

        // then
        expect(lastTheoryCall().enabled).toBe(false);

        // when
        act(() => result.current.setActiveTab("theories"));

        // then
        expect(lastTheoryCall().enabled).toBe(true);
        expect(lastTheoryCall().authorId).toBe(profileId);
    });

    it("leaves the theories query switched off while there is no profile to ask about", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: null, loading: false });
        const { result } = render();

        // when
        act(() => result.current.setActiveTab("theories"));

        // then
        expect(lastTheoryCall().enabled).toBe(false);
    });

    it("asks for the activity by username rather than by id", () => {
        // given
        const { result } = render();

        // when
        act(() => result.current.setActiveTab("activity"));

        // then
        expect(mocks.useUserActivity).toHaveBeenLastCalledWith("beatrice", 20, 0);
    });

    it("reads the follow list from the side the viewer is looking at", () => {
        // given
        mocks.useFollowers.mockReturnValue({ users: [makeUser({ id: "f-1" })], total: 1, loading: false });
        mocks.useFollowing.mockReturnValue({ users: [], total: 0, loading: true });
        const { result } = render();

        // when
        act(() => result.current.setActiveTab("followers"));

        // then
        expect(mocks.useFollowers).toHaveBeenLastCalledWith(profileId);
        expect(mocks.useFollowing).toHaveBeenLastCalledWith("");
        expect(result.current.followList.items).toHaveLength(1);
        expect(result.current.followList.loading).toBe(false);
    });
});

describe("useProfilePage paging", () => {
    it("gives every paged tab its own place in the list", () => {
        // given
        const { result } = render();

        // when
        act(() => result.current.posts.page?.goNext());
        act(() => result.current.setActiveTab("ships"));

        // then
        expect(mocks.useUserPosts).toHaveBeenLastCalledWith("", 20, 20);
        expect(mocks.useUserShips).toHaveBeenLastCalledWith(profileId, 20, 0);
    });

    it("walks a tab forward and back one window at a time", () => {
        // given
        const { result } = render();

        // when
        act(() => result.current.posts.page?.goNext());
        act(() => result.current.posts.page?.goNext());
        act(() => result.current.posts.page?.goPrev());

        // then
        expect(mocks.useUserPosts).toHaveBeenLastCalledWith(profileId, 20, 20);
    });

    it("pages the art tab in windows of twenty four", () => {
        // given
        const { result } = render();
        act(() => result.current.setActiveTab("art"));

        // when
        act(() => result.current.art.page?.goNext());

        // then
        expect(mocks.useUserArt).toHaveBeenLastCalledWith(profileId, 24, 24);
    });

    it("offers no pager at all to the lists the server never pages", () => {
        // given
        const { result } = render();

        // then
        expect(result.current.galleries.page).toBeNull();
        expect(result.current.ocs.page).toBeNull();
        expect(result.current.followList.page).toBeNull();
    });

    it("counts the galleries it was given, because the server sends no total", () => {
        // given
        mocks.useUserGalleries.mockReturnValue({
            galleries: [{ id: "g-1" }, { id: "g-2" }],
            loading: false,
            refresh: vi.fn(),
        });

        // when
        const { result } = render();

        // then
        expect(result.current.galleries.total).toBe(2);
    });
});

describe("useProfilePage viewer", () => {
    it("offers account management to staff looking at somebody else", () => {
        // when
        const { result } = render(makeUser({ id: viewerId, role: "moderator" }));

        // then
        expect(result.current.canManageUser).toBe(true);
    });

    it("keeps account management off a staff member's own profile", () => {
        // when
        const { result } = render(makeUser({ id: profileId, role: "admin" }));

        // then
        expect(result.current.canManageUser).toBe(false);
    });

    it("keeps account management away from an ordinary member", () => {
        // when
        const { result } = render(makeUser({ id: viewerId }));

        // then
        expect(result.current.canManageUser).toBe(false);
    });
});
