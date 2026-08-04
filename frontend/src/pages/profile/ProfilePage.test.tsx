import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeStats, makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { ActivityItem, Fanfic, Gallery, Mystery, OC, Ship, User, UserProfile } from "../../types/api";
import { ProfilePage } from "./ProfilePage";

const mocks = vi.hoisted(() => ({
    navigate: vi.fn(),
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
    toggleFollow: vi.fn(),
    toggleBlock: vi.fn(),
    createGallery: vi.fn(),
    refreshGalleries: vi.fn(),
    refreshPosts: vi.fn(),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocks.navigate };
});

vi.mock("../../api/queries/profile", () => ({ useProfile: mocks.useProfile }));
vi.mock("../../api/queries/theory", () => ({ useTheoryFeed: mocks.useTheoryFeed }));
vi.mock("../../hooks/useFollow", () => ({ useFollow: mocks.useFollow }));
vi.mock("../../hooks/useBlock", () => ({ useBlock: mocks.useBlock }));
vi.mock("../../api/queries/user", () => ({
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
}));
vi.mock("../../api/queries/oc", () => ({ useUserOCs: mocks.useUserOCs }));
vi.mock("../../api/queries/misc", () => ({ useFollowers: mocks.useFollowers, useFollowing: mocks.useFollowing }));
vi.mock("../../api/mutations/art", () => ({
    useCreateGallery: () => ({ mutateAsync: mocks.createGallery, isPending: false }),
}));

vi.mock("./TrophyCase", () => ({ TrophyCase: () => <div data-testid="trophy-case" /> }));
vi.mock("../../features/easterEgg", () => ({ HuntsInProgress: () => null }));
vi.mock("../../components/theory/TheoryCard/TheoryCard", () => ({
    TheoryCard: ({ theory }: { theory: { title: string } }) => <div data-testid="theory-card">{theory.title}</div>,
}));
vi.mock("../../components/post/PostCard/PostCard", () => ({
    PostCard: ({ post }: { post: { body: string } }) => <div data-testid="post-card">{post.body}</div>,
}));
vi.mock("../../components/journal/JournalCard/JournalCard", () => ({
    JournalCard: ({ journal }: { journal: { title: string } }) => <div data-testid="journal-card">{journal.title}</div>,
}));
vi.mock("../../components/art/ArtGrid/ArtGrid", () => ({
    ArtGrid: ({ art }: { art: { id: string }[] }) => <div data-testid="art-grid">{art.length} pieces of art</div>,
}));

const profileId = "profile-1";
const viewerId = "viewer-1";

const author: User = { id: profileId, username: "beatrice", display_name: "Beatrice" };

function makeProfile(overrides: Partial<UserProfile> = {}): UserProfile {
    return makeUser({
        id: profileId,
        username: "beatrice",
        display_name: "Beatrice",
        created_at: "2026-01-15T10:00:00Z",
        gender: "Prefer not to say",
        ...overrides,
    });
}

function makeShip(overrides: Partial<Ship> = {}): Ship {
    return {
        id: "ship-1",
        author,
        title: "Beato and Battler",
        description: "",
        characters: [
            { series: "umineko", character_name: "Beatrice", sort_order: 0 },
            { series: "umineko", character_name: "Battler", sort_order: 1 },
        ],
        vote_score: 3,
        comment_count: 1,
        is_crackship: false,
        created_at: "2026-02-01T00:00:00Z",
        ...overrides,
    };
}

function makeOC(overrides: Partial<OC> = {}): OC {
    return {
        id: "oc-1",
        author,
        name: "Clair Vaux Bernardus",
        description: "",
        series: "umineko",
        gallery: [],
        vote_score: 2,
        favourite_count: 4,
        user_favourited: false,
        comment_count: 1,
        is_crack_oc: false,
        created_at: "2026-02-01T00:00:00Z",
        ...overrides,
    };
}

function makeMystery(overrides: Partial<Mystery> = {}): Mystery {
    return {
        id: "mystery-1",
        title: "The Sealed Room",
        body: "",
        difficulty: "hard",
        author,
        solved: false,
        paused: false,
        gm_away: false,
        free_for_all: false,
        keep_open_after_solve: false,
        solver_count: 0,
        paused_duration_seconds: 0,
        attempt_count: 2,
        clue_count: 3,
        created_at: "2026-02-01T00:00:00Z",
        ...overrides,
    };
}

function makeFanfic(overrides: Partial<Fanfic> = {}): Fanfic {
    return {
        id: "fanfic-1",
        author,
        title: "Golden Land",
        summary: "",
        series: "umineko",
        rating: "general",
        language: "en",
        status: "complete",
        is_oneshot: false,
        contains_lemons: false,
        genres: [],
        tags: [],
        characters: [],
        is_pairing: false,
        word_count: 12000,
        chapter_count: 1,
        favourite_count: 0,
        view_count: 0,
        comment_count: 0,
        user_favourited: false,
        published_at: "2026-02-01T00:00:00Z",
        created_at: "2026-02-01T00:00:00Z",
        ...overrides,
    };
}

function makeGallery(overrides: Partial<Gallery> = {}): Gallery {
    return {
        id: "gallery-1",
        author,
        name: "Witch Portraits",
        description: "",
        cover_image_url: "",
        cover_thumbnail_url: "",
        art_count: 4,
        created_at: "2026-02-01T00:00:00Z",
        ...overrides,
    };
}

function makeActivity(overrides: Partial<ActivityItem> = {}): ActivityItem {
    return {
        type: "theory",
        theory_id: "theory-1",
        theory_title: "The culprit is on the island",
        body: "A short note.",
        created_at: "2026-02-01T00:00:00Z",
        ...overrides,
    };
}

function statBox(label: string): HTMLElement {
    for (const element of screen.getAllByText(label)) {
        const box = element.closest("div");
        if (element.tagName === "SPAN" && box) {
            return box;
        }
    }

    throw new Error(`there is no ${label} counter on the profile`);
}

function renderProfile(viewer: UserProfile | null = makeUser({ id: viewerId })) {
    const user = userEvent.setup();
    const result = renderWithProviders(<ProfilePage />, {
        user: viewer,
        route: "/user/beatrice",
        path: "/user/:username",
    });

    return { ...result, user };
}

beforeEach(() => {
    mocks.useProfile.mockReturnValue({ profile: makeProfile(), loading: false });
    mocks.useTheoryFeed.mockReturnValue({
        theories: [],
        total: 0,
        loading: false,
        offset: 0,
        limit: 20,
        goNext: vi.fn(),
        goPrev: vi.fn(),
        hasNext: false,
        hasPrev: false,
    });
    mocks.useFollow.mockReturnValue({
        stats: { follower_count: 7, following_count: 3, is_following: false, follows_you: false },
        loading: false,
        toggleFollow: mocks.toggleFollow,
    });
    mocks.useBlock.mockReturnValue({
        status: { blocking: false, blocked_by: false },
        loading: false,
        toggleBlock: mocks.toggleBlock,
    });
    mocks.useUserPosts.mockReturnValue({ posts: [], total: 0, loading: false, refresh: mocks.refreshPosts });
    mocks.useUserArt.mockReturnValue({ art: [], total: 0, loading: false });
    mocks.useUserGalleries.mockReturnValue({ galleries: [], loading: false, refresh: mocks.refreshGalleries });
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
    mocks.createGallery.mockResolvedValue({ id: "gallery-new" });
});

describe("ProfilePage loading and absence", () => {
    it("consults the game board while the profile is being fetched", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: null, loading: true });

        // when
        renderProfile();

        // then
        expect(screen.getByText("Consulting the game board...")).toBeInTheDocument();
    });

    it("says the player is not on the game board when there is no such profile", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: null, loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByText(/Player not found on the game board./)).toBeInTheDocument();
    });

    it("sends the visitor back to the feed from a missing profile", async () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: null, loading: false });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Return to Feed" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/");
    });

    it("looks the profile up by the username in the address", () => {
        // given
        const viewer = makeUser({ id: viewerId });

        // when
        renderProfile(viewer);

        // then
        expect(mocks.useProfile).toHaveBeenCalledWith("beatrice");
    });
});

describe("ProfilePage header", () => {
    it("introduces the player by their name and handle", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ display_name: "The Golden Witch" }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByRole("heading", { name: /The Golden Witch/ })).toBeInTheDocument();
        expect(screen.getByText("@beatrice")).toBeInTheDocument();
    });

    it("stands in for a missing avatar with the initial of the name", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ avatar_url: "" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByText("B")).toBeInTheDocument();
        expect(screen.queryByAltText("Beatrice")).not.toBeInTheDocument();
    });

    it("shows the avatar the player uploaded", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ avatar_url: "https://cdn.test/beato.png" }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByAltText("Beatrice")).toHaveAttribute("src", "https://cdn.test/beato.png");
    });

    it("flags a banned profile together with the reason", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ banned: true, ban_reason: "Endless witch hunting" }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByText("This user has been banned")).toBeInTheDocument();
        expect(screen.getByText("Reason: Endless witch hunting")).toBeInTheDocument();
    });

    it("leaves the ban banner off an ordinary profile", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ banned: false }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.queryByText("This user has been banned")).not.toBeInTheDocument();
    });

    it("hints at the empty bio when the player wrote none", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ bio: "" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByText("This player has not written a bio yet.")).toBeInTheDocument();
    });

    it("shows the bio the player wrote", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ bio: "Without love it cannot be seen." }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByText("Without love it cannot be seen.")).toBeInTheDocument();
    });

    it("records the day the player joined", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ created_at: "2026-01-15T10:00:00Z" }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByText(/^Joined /)).toBeInTheDocument();
    });

    it("keeps a withheld gender to itself", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ gender: "Prefer not to say" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.queryByText("Prefer not to say")).not.toBeInTheDocument();
    });

    it("shows a gender the player was happy to share", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ gender: "Female" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByText("Female")).toBeInTheDocument();
    });

    it("shows the pronouns as a pair", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ pronoun_subject: "she", pronoun_possessive: "her" }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByText("she/her")).toBeInTheDocument();
    });

    it("works the age out from the date of birth", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ dob: "1995-07-15" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByText(/\(31 years old\)/)).toBeInTheDocument();
    });

    it("uses the singular year for a one year old", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ dob: "2025-01-01" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByText(/\(1 year old\)/)).toBeInTheDocument();
    });

    it("leaves the age off a date of birth that has not happened yet", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ dob: "2030-01-01" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.queryByText(/years old/)).not.toBeInTheDocument();
        expect(screen.getByText(/^Born /)).toBeInTheDocument();
    });

    it("leaves an unreadable date of birth exactly as it stands", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ dob: "sometime" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByText("Born sometime")).toBeInTheDocument();
    });

    it("celebrates the player's favourite character", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ favourite_character: "Battler" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByText("Favourite Character")).toBeInTheDocument();
        expect(screen.getByText("Battler")).toBeInTheDocument();
    });
});

describe("ProfilePage social links", () => {
    it("turns a bare handle into a link to the service", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({
                social_twitter: "beato",
                social_github: "beato",
                social_bluesky: "beato",
                social_tumblr: "beato",
            }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getAllByRole("link", { name: "beato" }).map(link => link.getAttribute("href"))).toEqual([
            "https://x.com/beato",
            "https://beato.tumblr.com",
            "https://github.com/beato",
            "https://bsky.app/profile/beato",
        ]);
    });

    it("labels the bluesky handle the player shared", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ social_bluesky: "beato.bsky.social" }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByText("Bluesky")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "beato.bsky.social" })).toHaveAttribute(
            "href",
            "https://bsky.app/profile/beato.bsky.social",
        );
    });

    it("drops the leading at sign from a bluesky handle", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ social_bluesky: "@beato.bsky.social" }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByRole("link", { name: "@beato.bsky.social" })).toHaveAttribute(
            "href",
            "https://bsky.app/profile/beato.bsky.social",
        );
    });

    it("leaves a full bluesky address exactly as the player gave it", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ social_bluesky: "https://bsky.app/profile/beato.bsky.social" }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByRole("link", { name: "https://bsky.app/profile/beato.bsky.social" })).toHaveAttribute(
            "href",
            "https://bsky.app/profile/beato.bsky.social",
        );
    });

    it("sends a bare waifulist handle out to waifulist", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ social_waifulist: "beato" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByRole("link", { name: "beato" })).toHaveAttribute("href", "https://waifulist.moe/beato");
    });

    it("keeps a waifulist path on the waifulist domain the player gave", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ social_waifulist: "waifulist.moe/list/beato" }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByRole("link", { name: "waifulist.moe/list/beato" })).toHaveAttribute(
            "href",
            "https://waifulist.moe/list/beato",
        );
    });

    it("leaves a full address exactly as the player gave it", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ social_twitter: "https://x.com/goldenwitch" }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByRole("link", { name: "https://x.com/goldenwitch" })).toHaveAttribute(
            "href",
            "https://x.com/goldenwitch",
        );
    });

    it("shows the discord tag as plain text because it cannot be linked", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ social_discord: "beato#0001" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByText("beato#0001")).toBeInTheDocument();
        expect(screen.queryByRole("link", { name: "beato#0001" })).not.toBeInTheDocument();
    });

    it("turns a shared email address into a mail link", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ email: "beato@example.com" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByRole("link", { name: "beato@example.com" })).toHaveAttribute(
            "href",
            "mailto:beato@example.com",
        );
    });

    it("adds the missing scheme to a bare website", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ website: "witchs.moe" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.getByRole("link", { name: "witchs.moe" })).toHaveAttribute("href", "https://witchs.moe");
    });

    it("shows no social row when the player shared nothing", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile(), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.queryByText("Twitter / X")).not.toBeInTheDocument();
        expect(screen.queryByText("Website")).not.toBeInTheDocument();
    });
});

describe("ProfilePage viewer gates", () => {
    it("offers a signed out visitor nothing to press", () => {
        // given
        const viewer = null;

        // when
        renderProfile(viewer);

        // then
        expect(screen.queryByRole("button", { name: "Follow" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Message" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Block" })).not.toBeInTheDocument();
    });

    it("does not invite the owner to follow themselves", () => {
        // given
        const viewer = makeUser({ id: profileId });

        // when
        renderProfile(viewer);

        // then
        expect(screen.queryByRole("button", { name: "Follow" })).not.toBeInTheDocument();
    });

    it("invites a member to follow the player", async () => {
        // given
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Follow" }));

        // then
        expect(mocks.toggleFollow).toHaveBeenCalledOnce();
    });

    it("offers to unfollow a player the member already follows", () => {
        // given
        mocks.useFollow.mockReturnValue({
            stats: { follower_count: 7, following_count: 3, is_following: true, follows_you: false },
            loading: false,
            toggleFollow: mocks.toggleFollow,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByRole("button", { name: "Unfollow" })).toBeInTheDocument();
    });

    it("mentions when the player follows the viewer back", () => {
        // given
        mocks.useFollow.mockReturnValue({
            stats: { follower_count: 7, following_count: 3, is_following: false, follows_you: true },
            loading: false,
            toggleFollow: mocks.toggleFollow,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByText("Follows you")).toBeInTheDocument();
    });

    it("opens a direct message with the player from the profile", async () => {
        // given
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Message" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/chat", { state: { dmUserId: profileId } });
    });

    it("hides the message shortcut when the player takes no direct messages", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ dms_enabled: false }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.queryByRole("button", { name: "Message" })).not.toBeInTheDocument();
    });

    it("hides the message shortcut while the viewer is blocking the player", () => {
        // given
        mocks.useBlock.mockReturnValue({
            status: { blocking: true, blocked_by: false },
            loading: false,
            toggleBlock: mocks.toggleBlock,
        });

        // when
        renderProfile();

        // then
        expect(screen.queryByRole("button", { name: "Message" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Unblock" })).toBeInTheDocument();
    });

    it("warns the viewer that this player has blocked them", () => {
        // given
        mocks.useBlock.mockReturnValue({
            status: { blocking: false, blocked_by: true },
            loading: false,
            toggleBlock: mocks.toggleBlock,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByText("This user has blocked you.")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Message" })).not.toBeInTheDocument();
    });

    it("lets a member block an ordinary player", async () => {
        // given
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Block" }));

        // then
        expect(mocks.toggleBlock).toHaveBeenCalledOnce();
    });

    it("refuses to offer a block against a member of staff", () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: makeProfile({ role: "moderator" }), loading: false });

        // when
        renderProfile();

        // then
        expect(screen.queryByRole("button", { name: "Block" })).not.toBeInTheDocument();
    });

    it("offers staff a way through to the account management screen", async () => {
        // given
        const { user } = renderProfile(makeUser({ id: viewerId, role: "moderator" }));

        // when
        await user.click(screen.getByRole("button", { name: "Manage account" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith(`/admin/users/${profileId}`);
    });

    it("keeps account management away from an ordinary member", () => {
        // given
        const viewer = makeUser({ id: viewerId });

        // when
        renderProfile(viewer);

        // then
        expect(screen.queryByRole("button", { name: "Manage account" })).not.toBeInTheDocument();
    });

    it("keeps account management off a staff member's own profile", () => {
        // given
        const viewer = makeUser({ id: profileId, role: "admin" });

        // when
        renderProfile(viewer);

        // then
        expect(screen.queryByRole("button", { name: "Manage account" })).not.toBeInTheDocument();
    });
});

describe("ProfilePage stats", () => {
    it("counts the theories, responses and votes the player earned", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ stats: makeStats({ theory_count: 4, response_count: 9, votes_received: 21 }) }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByText("Votes Received")).toBeInTheDocument();
        expect(screen.getByText("4")).toBeInTheDocument();
        expect(screen.getByText("9")).toBeInTheDocument();
        expect(screen.getByText("21")).toBeInTheDocument();
    });

    it("counts followers and following once the follow stats arrive", () => {
        // given
        mocks.useFollow.mockReturnValue({
            stats: { follower_count: 7, following_count: 3, is_following: false, follows_you: false },
            loading: false,
            toggleFollow: mocks.toggleFollow,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByText("Followers")).toBeInTheDocument();
        expect(screen.getByText("Following")).toBeInTheDocument();
    });

    it("leaves the follower counters out until the follow stats arrive", () => {
        // given
        mocks.useFollow.mockReturnValue({ stats: null, loading: true, toggleFollow: mocks.toggleFollow });

        // when
        renderProfile();

        // then
        expect(screen.queryByText("Followers")).not.toBeInTheDocument();
        expect(screen.queryByText("Following")).not.toBeInTheDocument();
    });

    it("jumps to the ships tab from the ships counter", async () => {
        // given
        const { user } = renderProfile();

        // when
        await user.click(statBox("Ships"));

        // then
        expect(screen.getByText("No ships declared yet.")).toBeInTheDocument();
    });

    it("jumps to the followers tab from the followers counter", async () => {
        // given
        const { user } = renderProfile();

        // when
        await user.click(statBox("Followers"));

        // then
        expect(screen.getByText("No followers yet.")).toBeInTheDocument();
    });
});

describe("ProfilePage tabs", () => {
    it("opens on posts and says when the player has written none", () => {
        // given
        mocks.useUserPosts.mockReturnValue({ posts: [], total: 0, loading: false, refresh: mocks.refreshPosts });

        // when
        renderProfile();

        // then
        expect(screen.getByText("No posts yet.")).toBeInTheDocument();
        expect(mocks.useUserPosts).toHaveBeenCalledWith(profileId, 20, 0);
    });

    it("opens on the tab the player chose as their default", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ private: { default_profile_tab: "mysteries" } }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(screen.getByText("No mysteries declared yet.")).toBeInTheDocument();
        expect(mocks.useUserMysteries).toHaveBeenCalledWith(profileId, 20, 0);
    });

    it("asks nothing of the tabs the viewer is not looking at", () => {
        // given
        const viewer = makeUser({ id: viewerId });

        // when
        renderProfile(viewer);

        // then
        expect(mocks.useUserShips).toHaveBeenCalledWith("", 20, 0);
        expect(mocks.useUserOCs).toHaveBeenCalledWith("");
        expect(mocks.useUserActivity).toHaveBeenCalledWith("", 20, 0);
    });

    it("fetches the player's ships once the ships tab is opened", async () => {
        // given
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Ships" }));

        // then
        expect(mocks.useUserShips).toHaveBeenLastCalledWith(profileId, 20, 0);
    });

    it("describes each ship with its pairing and score", async () => {
        // given
        mocks.useUserShips.mockReturnValue({ ships: [makeShip()], total: 1, loading: false });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Ships" }));

        // then
        expect(screen.getByText("Beato and Battler")).toBeInTheDocument();
        expect(screen.getByText("Beatrice × Battler")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Beato and Battler/ })).toHaveAttribute("href", "/ships/ship-1");
    });

    it("waits while the ships of the tab are being fetched", async () => {
        // given
        mocks.useUserShips.mockReturnValue({ ships: [], total: 0, loading: true });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Ships" }));

        // then
        expect(screen.getByText("Loading ships...")).toBeInTheDocument();
        expect(screen.queryByText("No ships declared yet.")).not.toBeInTheDocument();
    });

    it("names the series of a custom original character", async () => {
        // given
        mocks.useUserOCs.mockReturnValue({
            ocs: [makeOC({ series: "custom", custom_series_name: "Rose Guns Days" })],
            total: 1,
            loading: false,
        });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "OCs" }));

        // then
        expect(screen.getByText("Clair Vaux Bernardus")).toBeInTheDocument();
        expect(screen.getByText("Rose Guns Days")).toBeInTheDocument();
    });

    it("marks an unsolved mystery as still open", async () => {
        // given
        mocks.useUserMysteries.mockReturnValue({ mysteries: [makeMystery()], total: 1, loading: false });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Mysteries" }));

        // then
        expect(screen.getByText("The Sealed Room")).toBeInTheDocument();
        expect(screen.getByText("Open")).toBeInTheDocument();
    });

    it("names the winner of a solved mystery", async () => {
        // given
        mocks.useUserMysteries.mockReturnValue({
            mysteries: [makeMystery({ solved: true, winner: { id: "w", username: "ange", display_name: "Ange" } })],
            total: 1,
            loading: false,
        });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Mysteries" }));

        // then
        expect(screen.getByText("Solved")).toBeInTheDocument();
        expect(screen.getByText(/Winner: Ange/)).toBeInTheDocument();
    });

    it("counts the words and chapters of each fanfiction", async () => {
        // given
        mocks.useUserFanfics.mockReturnValue({ fanfics: [makeFanfic()], total: 1, loading: false });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Fanfictions" }));

        // then
        expect(screen.getByText("Golden Land")).toBeInTheDocument();
        expect(screen.getByText(/12,000 words/)).toBeInTheDocument();
        expect(screen.getByText(/1 chapter$/)).toBeInTheDocument();
    });

    it("keeps saved fanfictions apart from the ones the player wrote", async () => {
        // given
        mocks.useUserFanficFavourites.mockReturnValue({
            fanfics: [makeFanfic({ id: "fanfic-2", title: "Rokkenjima Nights" })],
            total: 1,
            loading: false,
        });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Saved Fics" }));

        // then
        expect(screen.getByText("Rokkenjima Nights")).toBeInTheDocument();
        expect(mocks.useUserFanficFavourites).toHaveBeenLastCalledWith(profileId, 20, 0);
    });

    it("lists the reading journals through their cards", async () => {
        // given
        mocks.useUserJournals.mockReturnValue({
            journals: [{ id: "journal-1", title: "First read" }],
            total: 1,
            loading: false,
        });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Journals" }));

        // then
        expect(screen.getByTestId("journal-card")).toHaveTextContent("First read");
    });

    it("says when the player follows no journals", async () => {
        // given
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Following Journals" }));

        // then
        expect(screen.getByText("Not following any journals yet.")).toBeInTheDocument();
    });

    it("hands the theories tab to the theory cards", async () => {
        // given
        mocks.useTheoryFeed.mockReturnValue({
            theories: [{ id: "theory-1", title: "The witch did it" }],
            total: 1,
            loading: false,
            offset: 0,
            limit: 20,
            goNext: vi.fn(),
            goPrev: vi.fn(),
            hasNext: false,
            hasPrev: false,
        });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Theories" }));

        // then
        expect(screen.getByTestId("theory-card")).toHaveTextContent("The witch did it");
    });

    it("shows the player's art in a grid", async () => {
        // given
        mocks.useUserArt.mockReturnValue({ art: [{ id: "art-1" }, { id: "art-2" }], total: 2, loading: false });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Art" }));

        // then
        expect(screen.getByTestId("art-grid")).toHaveTextContent("2 pieces of art");
    });
});

describe("ProfilePage activity tab", () => {
    it("says when the player has done nothing yet", async () => {
        // given
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Activity" }));

        // then
        expect(screen.getByText("No activity yet.")).toBeInTheDocument();
        expect(mocks.useUserActivity).toHaveBeenLastCalledWith("beatrice", 20, 0);
    });

    it("walks the activity on to the next page", async () => {
        // given
        mocks.useUserActivity.mockReturnValue({ activity: [makeActivity()], total: 45, loading: false });
        const { user } = renderProfile();
        await user.click(screen.getByRole("button", { name: "Activity" }));

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(mocks.useUserActivity).toHaveBeenLastCalledWith("beatrice", 20, 20);
    });

    it("leaves the activity pager out while everything fits on one page", async () => {
        // given
        mocks.useUserActivity.mockReturnValue({ activity: [makeActivity()], total: 1, loading: false });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Activity" }));

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("labels a theory the player created", async () => {
        // given
        mocks.useUserActivity.mockReturnValue({ activity: [makeActivity()], total: 1, loading: false });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Activity" }));

        // then
        expect(screen.getByText("Created theory")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /The culprit is on the island/ })).toHaveAttribute(
            "href",
            "/theory/theory-1",
        );
    });

    it("labels which side a response took", async () => {
        // given
        mocks.useUserActivity.mockReturnValue({
            activity: [makeActivity({ type: "response", side: "without_love" })],
            total: 1,
            loading: false,
        });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Activity" }));

        // then
        expect(screen.getByText("Responded without love")).toBeInTheDocument();
    });

    it("shortens a long piece of activity", async () => {
        // given
        const body = "a".repeat(250);
        mocks.useUserActivity.mockReturnValue({ activity: [makeActivity({ body })], total: 1, loading: false });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Activity" }));

        // then
        expect(screen.getByText(`${"a".repeat(200)}...`)).toBeInTheDocument();
    });
});

describe("ProfilePage follow lists", () => {
    it("lists the players following this profile", async () => {
        // given
        mocks.useFollowers.mockReturnValue({
            users: [{ id: "u-1", username: "ange", display_name: "Ange" }],
            total: 1,
            loading: false,
        });
        const { user } = renderProfile();

        // when
        await user.click(statBox("Followers"));

        // then
        expect(screen.getByRole("link", { name: /Ange/ })).toHaveAttribute("href", "/user/ange");
        expect(mocks.useFollowers).toHaveBeenLastCalledWith(profileId);
    });

    it("says the player follows nobody yet", async () => {
        // given
        const { user } = renderProfile();

        // when
        await user.click(statBox("Following"));

        // then
        expect(screen.getByText("Not following anyone yet.")).toBeInTheDocument();
        expect(mocks.useFollowing).toHaveBeenLastCalledWith(profileId);
    });
});

describe("ProfilePage galleries tab", () => {
    async function openGalleries(user: ReturnType<typeof userEvent.setup>) {
        await user.click(screen.getByRole("button", { name: "Galleries" }));
    }

    it("shows each gallery with how much art it holds", async () => {
        // given
        mocks.useUserGalleries.mockReturnValue({
            galleries: [makeGallery()],
            loading: false,
            refresh: mocks.refreshGalleries,
        });
        const { user } = renderProfile();

        // when
        await openGalleries(user);

        // then
        expect(screen.getByText("Witch Portraits")).toBeInTheDocument();
        expect(screen.getByText("4 pieces")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Witch Portraits/ })).toHaveAttribute(
            "href",
            "/gallery/view/gallery-1",
        );
    });

    it("keeps gallery creation away from a visitor", async () => {
        // given
        const { user } = renderProfile();

        // when
        await openGalleries(user);

        // then
        expect(screen.queryByRole("button", { name: "Create Gallery" })).not.toBeInTheDocument();
        expect(screen.getByText("No galleries yet.")).toBeInTheDocument();
    });

    it("lets the owner start a new gallery", async () => {
        // given
        const { user } = renderProfile(makeUser({ id: profileId }));

        // when
        await openGalleries(user);

        // then
        expect(screen.getByRole("button", { name: "Create Gallery" })).toBeInTheDocument();
    });

    it("refuses to create a gallery without a name", async () => {
        // given
        const { user } = renderProfile(makeUser({ id: profileId }));
        await openGalleries(user);

        // when
        await user.click(screen.getByRole("button", { name: "Create Gallery" }));

        // then
        expect(screen.getByRole("button", { name: "Create" })).toBeDisabled();
    });

    it("creates the gallery with a trimmed name and description", async () => {
        // given
        const { user } = renderProfile(makeUser({ id: profileId }));
        await openGalleries(user);
        await user.click(screen.getByRole("button", { name: "Create Gallery" }));

        // when
        await user.type(screen.getByPlaceholderText("Gallery name"), "  Witch Portraits  ");
        await user.type(screen.getByPlaceholderText("Description (optional)"), "  Beato only  ");
        await user.click(screen.getByRole("button", { name: "Create" }));

        // then
        expect(mocks.createGallery).toHaveBeenCalledWith({ name: "Witch Portraits", description: "Beato only" });
        expect(mocks.refreshGalleries).toHaveBeenCalledOnce();
    });

    it("keeps the form open when the gallery could not be created", async () => {
        // given
        mocks.createGallery.mockRejectedValue(new Error("Gallery limit reached."));
        const { user } = renderProfile(makeUser({ id: profileId }));
        await openGalleries(user);
        await user.click(screen.getByRole("button", { name: "Create Gallery" }));
        await user.type(screen.getByPlaceholderText("Gallery name"), "Witch Portraits");

        // when
        await user.click(screen.getByRole("button", { name: "Create" }));

        // then
        expect(screen.getByPlaceholderText("Gallery name")).toHaveValue("Witch Portraits");
        expect(mocks.refreshGalleries).not.toHaveBeenCalled();
    });

    it("lets the owner abandon the new gallery", async () => {
        // given
        const { user } = renderProfile(makeUser({ id: profileId }));
        await openGalleries(user);
        await user.click(screen.getByRole("button", { name: "Create Gallery" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByPlaceholderText("Gallery name")).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Create Gallery" })).toBeInTheDocument();
    });
});

describe("ProfilePage pagination", () => {
    it("leaves the pager out while everything fits on one page", () => {
        // given
        mocks.useUserPosts.mockReturnValue({
            posts: [{ id: "post-1", body: "hello" }],
            total: 1,
            loading: false,
            refresh: mocks.refreshPosts,
        });

        // when
        renderProfile();

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("walks the posts on to the next page", async () => {
        // given
        mocks.useUserPosts.mockReturnValue({
            posts: [{ id: "post-1", body: "hello" }],
            total: 45,
            loading: false,
            refresh: mocks.refreshPosts,
        });
        const { user } = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(mocks.useUserPosts).toHaveBeenLastCalledWith(profileId, 20, 20);
    });

    it("walks the posts back to the previous page", async () => {
        // given
        mocks.useUserPosts.mockReturnValue({
            posts: [{ id: "post-1", body: "hello" }],
            total: 45,
            loading: false,
            refresh: mocks.refreshPosts,
        });
        const { user } = renderProfile();
        await user.click(screen.getByRole("button", { name: "Next" }));

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(mocks.useUserPosts).toHaveBeenLastCalledWith(profileId, 20, 0);
    });
});
