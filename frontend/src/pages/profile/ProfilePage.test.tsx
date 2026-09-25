import { screen, type Matcher } from "@testing-library/react";
import userEvent, { type UserEvent } from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { makeGallery, makeStats, makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type {
    ActivityItem,
    BlockStatus,
    Fanfic,
    FollowStats,
    Mystery,
    OC,
    SessionUser,
    Ship,
    SiteInfo,
    User,
    UserProfile,
} from "../../types/api";
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

vi.mock("../../hooks/queries/profile", () => ({ useProfile: mocks.useProfile }));
vi.mock("../../hooks/queries/theory", () => ({ useTheoryFeed: mocks.useTheoryFeed }));
vi.mock("../../hooks/useFollow", () => ({ useFollow: mocks.useFollow }));
vi.mock("../../hooks/useBlock", () => ({ useBlock: mocks.useBlock }));
vi.mock("../../hooks/queries/user", () => ({
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
vi.mock("../../hooks/queries/oc", () => ({ useUserOCs: mocks.useUserOCs }));
vi.mock("../../hooks/mutations/art", () => ({
    useCreateGallery: () => ({ mutateAsync: mocks.createGallery, isPending: false }),
}));

vi.mock("./TrophyCase", () => ({ TrophyCase: () => <div data-testid="trophy-case" /> }));
vi.mock("../../components/easterEgg", () => ({ HuntsInProgress: () => null }));
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

interface ProfileCase {
    name: string;
    viewer?: SessionUser | null;
    siteInfo?: Partial<SiteInfo>;
    profile?: Partial<UserProfile>;
    follow?: Partial<FollowStats> | null;
    block?: Partial<BlockStatus>;
    serves?: [Mock, object];
    stat?: string;
    press?: string[];
    shown?: Matcher[];
    hidden?: Matcher[];
    buttons?: string[];
    noButtons?: string[];
    reads?: [Matcher, string][];
    links?: [RegExp, string][];
    socialLinks?: [string, string][];
    cards?: [string, string][];
    fetches?: [Mock, ...unknown[]];
    once?: Mock[];
    calls?: [Mock, ...unknown[]];
}

const profileId = "profile-1";
const viewerId = "viewer-1";
const emptyBio = "This player has not written a bio yet.";
const guide = "How to talk to me";

const author: User = { id: profileId, username: "beatrice", display_name: "Beatrice" };
const member = makeUser({ id: viewerId });
const owner = makeUser({ id: profileId });
const staff = makeUser({ id: viewerId, role: "moderator" });
const followStats: FollowStats = { follower_count: 7, following_count: 3, is_following: false, follows_you: false };

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

function renderProfile(viewer: SessionUser | null = member, siteInfo?: Partial<SiteInfo>) {
    const user = userEvent.setup();
    renderWithProviders(<ProfilePage />, { user: viewer, siteInfo, route: "/user/beatrice", path: "/user/:username" });

    return user;
}

async function openGalleryForm(user: UserEvent) {
    await user.click(screen.getByRole("button", { name: "Galleries" }));
    await user.click(screen.getByRole("button", { name: "Create Gallery" }));
}

async function checkProfileCase(testCase: ProfileCase) {
    // given
    mocks.useProfile.mockReturnValue({ profile: makeProfile(testCase.profile), loading: false });
    mocks.useFollow.mockReturnValue({
        stats: testCase.follow === null ? null : { ...followStats, ...testCase.follow },
        loading: testCase.follow === null,
        toggleFollow: mocks.toggleFollow,
    });
    mocks.useBlock.mockReturnValue({
        status: { blocking: false, blocked_by: false, ...testCase.block },
        loading: false,
        toggleBlock: mocks.toggleBlock,
    });
    if (testCase.serves) {
        const [hook, value] = testCase.serves;
        hook.mockReturnValue(value);
    }

    // when
    const user = renderProfile(testCase.viewer, testCase.siteInfo);
    if (testCase.stat) {
        await user.click(statBox(testCase.stat));
    }
    for (const name of testCase.press ?? []) {
        await user.click(screen.getByRole("button", { name }));
    }

    // then
    for (const text of testCase.shown ?? []) {
        expect(screen.getByText(text)).toBeInTheDocument();
    }

    for (const text of testCase.hidden ?? []) {
        expect(screen.queryByText(text)).not.toBeInTheDocument();
    }

    for (const name of testCase.buttons ?? []) {
        expect(screen.getByRole("button", { name })).toBeInTheDocument();
    }

    for (const name of testCase.noButtons ?? []) {
        expect(screen.queryByRole("button", { name })).not.toBeInTheDocument();
    }

    for (const [text, content] of testCase.reads ?? []) {
        expect(screen.getByText(text)).toHaveTextContent(content);
    }

    for (const [name, href] of testCase.links ?? []) {
        expect(screen.getByRole("link", { name })).toHaveAttribute("href", href);
    }

    if (testCase.socialLinks) {
        const links = screen
            .queryAllByRole("link")
            .filter(link => /^https?:\/\//.test(link.getAttribute("href") ?? ""))
            .map(link => [link.textContent, link.getAttribute("href")]);
        expect(links).toEqual(testCase.socialLinks);
    }

    for (const [testId, content] of testCase.cards ?? []) {
        expect(screen.getByTestId(testId)).toHaveTextContent(content);
    }

    if (testCase.fetches) {
        const [hook, ...args] = testCase.fetches;
        expect(hook).toHaveBeenLastCalledWith(...args);
    }

    for (const spy of testCase.once ?? []) {
        expect(spy).toHaveBeenCalledOnce();
    }

    if (testCase.calls) {
        const [spy, ...args] = testCase.calls;
        expect(spy).toHaveBeenCalledExactlyOnceWith(...args);
    }
}

beforeEach(() => {
    mocks.useProfile.mockReturnValue({ profile: makeProfile(), loading: false });
    mocks.useTheoryFeed.mockReturnValue({ theories: [], total: 0, loading: false });
    mocks.useFollow.mockReturnValue({ stats: followStats, loading: false, toggleFollow: mocks.toggleFollow });
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

    it("says the player is not on the game board and sends the visitor back to the feed", async () => {
        // given
        mocks.useProfile.mockReturnValue({ profile: null, loading: false });
        const user = renderProfile();

        // when
        await user.click(screen.getByRole("button", { name: "Return to Feed" }));

        // then
        expect(screen.getByText(/Player not found on the game board./)).toBeInTheDocument();
        expect(mocks.navigate).toHaveBeenCalledWith("/");
    });
});

describe("ProfilePage header", () => {
    it("looks the player up by the username in the address and introduces them by name and handle", () => {
        // given
        mocks.useProfile.mockReturnValue({
            profile: makeProfile({ display_name: "The Golden Witch" }),
            loading: false,
        });

        // when
        renderProfile();

        // then
        expect(mocks.useProfile).toHaveBeenCalledWith("beatrice");
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

    const headerCases: ProfileCase[] = [
        {
            name: "flags a banned profile together with the reason",
            profile: { banned: true, ban_reason: "Endless witch hunting" },
            shown: ["This user has been banned"],
            reads: [[/Reason:/, "Reason: Endless witch hunting"]],
        },
        {
            name: "leaves the ban banner off an ordinary profile",
            profile: { banned: false },
            hidden: ["This user has been banned"],
        },
        {
            name: "shows the bio the player wrote",
            profile: { bio: "Without love it cannot be seen." },
            shown: ["Without love it cannot be seen."],
        },
        {
            name: "does not offer the member empty-bio line to a character with no bio",
            profile: { bio: "", is_bot: true },
            shown: [guide],
            hidden: [emptyBio],
        },
        {
            name: "keeps a withheld gender to itself",
            profile: { gender: "Prefer not to say" },
            hidden: ["Prefer not to say"],
        },
        {
            name: "shows a gender the player was happy to share",
            profile: { gender: "Female" },
            shown: ["Female"],
        },
        {
            name: "shows the pronouns as a pair",
            profile: { pronoun_subject: "she", pronoun_possessive: "her" },
            shown: ["she/her"],
        },
        {
            name: "leaves an unreadable date of birth exactly as it stands",
            profile: { dob: "sometime" },
            shown: ["Born sometime"],
        },
        {
            name: "celebrates the player's favourite character",
            profile: { favourite_character: "Battler" },
            shown: ["Favourite Character", "Battler"],
        },
    ];

    it.each(headerCases)("$name", checkProfileCase);
});

describe("ProfilePage social links", () => {
    const socialCases: ProfileCase[] = [
        {
            name: "turns a bare handle into a link to the service",
            profile: {
                social_twitter: "beato",
                social_github: "beato",
                social_bluesky: "beato",
                social_tumblr: "beato",
            },
            socialLinks: [
                ["beato", "https://x.com/beato"],
                ["beato", "https://beato.tumblr.com"],
                ["beato", "https://github.com/beato"],
                ["beato", "https://bsky.app/profile/beato"],
            ],
        },
        {
            name: "labels the bluesky handle the player shared",
            profile: { social_bluesky: "beato.bsky.social" },
            shown: ["Bluesky"],
            socialLinks: [["beato.bsky.social", "https://bsky.app/profile/beato.bsky.social"]],
        },
        {
            name: "shows the discord tag as plain text because it cannot be linked",
            profile: { social_discord: "beato#0001" },
            shown: ["beato#0001"],
            socialLinks: [],
        },
        {
            name: "shows no social row when the player shared nothing",
            hidden: ["Twitter / X", "Website"],
            socialLinks: [],
        },
    ];

    it.each(socialCases)("$name", checkProfileCase);
});

describe("ProfilePage viewer gates", () => {
    const gateCases: ProfileCase[] = [
        {
            name: "offers a signed out visitor nothing to press",
            viewer: null,
            noButtons: ["Follow", "Message", "Block"],
        },
        {
            name: "does not invite the owner to follow themselves",
            viewer: owner,
            noButtons: ["Follow"],
        },
        {
            name: "invites a member to follow the player",
            press: ["Follow"],
            once: [mocks.toggleFollow],
        },
        {
            name: "offers to unfollow a player the member already follows",
            follow: { is_following: true },
            buttons: ["Unfollow"],
        },
        {
            name: "mentions when the player follows the viewer back",
            follow: { follows_you: true },
            shown: ["Follows you"],
        },
        {
            name: "opens a direct message with the player from the profile",
            press: ["Message"],
            calls: [mocks.navigate, "/chat", { state: { dmUserId: profileId } }],
        },
        {
            name: "hides the message shortcut when the player takes no direct messages",
            profile: { dms_enabled: false },
            noButtons: ["Message"],
        },
        {
            name: "hides the message shortcut while the viewer is blocking the player",
            block: { blocking: true },
            buttons: ["Unblock"],
            noButtons: ["Message"],
        },
        {
            name: "warns the viewer that this player has blocked them",
            block: { blocked_by: true },
            shown: ["This user has blocked you."],
            noButtons: ["Message"],
        },
        {
            name: "lets a member block an ordinary player",
            press: ["Block"],
            once: [mocks.toggleBlock],
        },
        {
            name: "refuses to offer a block against a member of staff",
            profile: { role: "moderator" },
            noButtons: ["Block"],
        },
        {
            name: "offers staff a way through to the account management screen",
            viewer: staff,
            press: ["Manage account"],
            calls: [mocks.navigate, `/admin/users/${profileId}`],
        },
        {
            name: "keeps account management away from an ordinary member",
            noButtons: ["Manage account"],
        },
        {
            name: "keeps account management off a staff member's own profile",
            viewer: makeUser({ id: profileId, role: "admin" }),
            noButtons: ["Manage account"],
        },
    ];

    it.each(gateCases)("$name", checkProfileCase);
});

describe("ProfilePage stats", () => {
    const statCases: ProfileCase[] = [
        {
            name: "counts the theories, responses and votes the player earned",
            profile: { stats: makeStats({ theory_count: 4, response_count: 9, votes_received: 21 }) },
            shown: ["Votes Received", "4", "9", "21"],
        },
        {
            name: "counts followers and following once the follow stats arrive",
            shown: ["Followers", "Following"],
        },
        {
            name: "leaves the follower counters out until the follow stats arrive",
            follow: null,
            hidden: ["Followers", "Following"],
        },
        {
            name: "jumps to the ships tab from the ships counter",
            stat: "Ships",
            shown: ["No ships declared yet."],
        },
        {
            name: "jumps to the followers tab from the followers counter",
            stat: "Followers",
            shown: ["No followers yet."],
        },
    ];

    it.each(statCases)("$name", checkProfileCase);
});

describe("ProfilePage tabs", () => {
    const tabCases: ProfileCase[] = [
        {
            name: "opens on posts and says when the player has written none",
            shown: ["No posts yet."],
            fetches: [mocks.useUserPosts, profileId, 20, 0],
        },
        {
            name: "opens on the tab the player chose as their default",
            profile: { private: { default_profile_tab: "mysteries" } },
            shown: ["No mysteries declared yet."],
            fetches: [mocks.useUserMysteries, profileId, 20, 0],
        },
        {
            name: "fetches the player's ships once the tab is opened and describes each with its pairing and score",
            serves: [mocks.useUserShips, { ships: [makeShip()], total: 1, loading: false }],
            press: ["Ships"],
            shown: ["Beato and Battler", "Beatrice × Battler"],
            links: [[/Beato and Battler/, "/ships/ship-1"]],
            fetches: [mocks.useUserShips, profileId, 20, 0],
        },
        {
            name: "waits while the ships of the tab are being fetched",
            serves: [mocks.useUserShips, { ships: [], total: 0, loading: true }],
            press: ["Ships"],
            shown: ["Loading ships..."],
            hidden: ["No ships declared yet."],
        },
        {
            name: "names the series of a custom original character",
            serves: [
                mocks.useUserOCs,
                { ocs: [makeOC({ series: "custom", custom_series_name: "Rose Guns Days" })], total: 1, loading: false },
            ],
            press: ["OCs"],
            shown: ["Clair Vaux Bernardus", "Rose Guns Days"],
        },
        {
            name: "marks an unsolved mystery as still open",
            serves: [mocks.useUserMysteries, { mysteries: [makeMystery()], total: 1, loading: false }],
            press: ["Mysteries"],
            shown: ["The Sealed Room", "Open"],
        },
        {
            name: "names the winner of a solved mystery",
            serves: [
                mocks.useUserMysteries,
                {
                    mysteries: [
                        makeMystery({ solved: true, winner: { id: "w", username: "ange", display_name: "Ange" } }),
                    ],
                    total: 1,
                    loading: false,
                },
            ],
            press: ["Mysteries"],
            shown: ["Solved"],
            reads: [[/Winner:/, "Winner: Ange"]],
        },
        {
            name: "counts the words and chapters of each fanfiction",
            serves: [mocks.useUserFanfics, { fanfics: [makeFanfic()], total: 1, loading: false }],
            press: ["Fanfictions"],
            shown: ["Golden Land", /12,000 words/, /1 chapter$/],
        },
        {
            name: "keeps saved fanfictions apart from the ones the player wrote",
            serves: [
                mocks.useUserFanficFavourites,
                { fanfics: [makeFanfic({ id: "fanfic-2", title: "Rokkenjima Nights" })], total: 1, loading: false },
            ],
            press: ["Saved Fics"],
            shown: ["Rokkenjima Nights"],
            fetches: [mocks.useUserFanficFavourites, profileId, 20, 0],
        },
        {
            name: "lists the reading journals through their cards",
            serves: [
                mocks.useUserJournals,
                { journals: [{ id: "journal-1", title: "First read" }], total: 1, loading: false },
            ],
            press: ["Journals"],
            cards: [["journal-card", "First read"]],
        },
        {
            name: "says when the player follows no journals",
            press: ["Following Journals"],
            shown: ["Not following any journals yet."],
        },
        {
            name: "hands the theories tab to the theory cards",
            serves: [
                mocks.useTheoryFeed,
                { theories: [{ id: "theory-1", title: "The witch did it" }], total: 1, loading: false },
            ],
            press: ["Theories"],
            cards: [["theory-card", "The witch did it"]],
        },
        {
            name: "shows the player's art in a grid",
            serves: [mocks.useUserArt, { art: [{ id: "art-1" }, { id: "art-2" }], total: 2, loading: false }],
            press: ["Art"],
            cards: [["art-grid", "2 pieces of art"]],
        },
    ];

    it.each(tabCases)("$name", checkProfileCase);
});

describe("ProfilePage activity tab", () => {
    const activityCases: ProfileCase[] = [
        {
            name: "says when the player has done nothing yet",
            press: ["Activity"],
            shown: ["No activity yet."],
            fetches: [mocks.useUserActivity, "beatrice", 20, 0],
        },
        {
            name: "walks the activity on to the next page",
            serves: [mocks.useUserActivity, { activity: [makeActivity()], total: 45, loading: false }],
            press: ["Activity", "Next"],
            fetches: [mocks.useUserActivity, "beatrice", 20, 20],
        },
        {
            name: "labels a theory the player created and leaves the pager out while it fits on one page",
            serves: [mocks.useUserActivity, { activity: [makeActivity()], total: 1, loading: false }],
            press: ["Activity"],
            shown: ["Created theory"],
            links: [[/The culprit is on the island/, "/theory/theory-1"]],
            noButtons: ["Next"],
        },
        {
            name: "labels which side a response took",
            serves: [
                mocks.useUserActivity,
                { activity: [makeActivity({ type: "response", side: "without_love" })], total: 1, loading: false },
            ],
            press: ["Activity"],
            shown: ["Responded without love"],
        },
        {
            name: "shortens a long piece of activity",
            serves: [
                mocks.useUserActivity,
                { activity: [makeActivity({ body: "a".repeat(250) })], total: 1, loading: false },
            ],
            press: ["Activity"],
            shown: [`${"a".repeat(200)}...`],
        },
    ];

    it.each(activityCases)("$name", checkProfileCase);
});

describe("ProfilePage follow lists", () => {
    const followListCases: ProfileCase[] = [
        {
            name: "lists the players following this profile",
            serves: [
                mocks.useFollowers,
                { users: [{ id: "u-1", username: "ange", display_name: "Ange" }], total: 1, loading: false },
            ],
            stat: "Followers",
            links: [[/Ange/, "/user/ange"]],
            fetches: [mocks.useFollowers, profileId],
        },
        {
            name: "says the player follows nobody yet",
            stat: "Following",
            shown: ["Not following anyone yet."],
            fetches: [mocks.useFollowing, profileId],
        },
    ];

    it.each(followListCases)("$name", checkProfileCase);
});

describe("ProfilePage galleries tab", () => {
    const galleryCases: ProfileCase[] = [
        {
            name: "shows each gallery with how much art it holds",
            serves: [
                mocks.useUserGalleries,
                {
                    galleries: [makeGallery({ author, name: "Witch Portraits", art_count: 4 })],
                    loading: false,
                    refresh: mocks.refreshGalleries,
                },
            ],
            press: ["Galleries"],
            shown: ["Witch Portraits", "4 pieces"],
            links: [[/Witch Portraits/, "/gallery/view/gallery-1"]],
        },
        {
            name: "keeps gallery creation away from a visitor",
            press: ["Galleries"],
            shown: ["No galleries yet."],
            noButtons: ["Create Gallery"],
        },
        {
            name: "lets the owner start a new gallery",
            viewer: owner,
            press: ["Galleries"],
            buttons: ["Create Gallery"],
        },
    ];

    it.each(galleryCases)("$name", checkProfileCase);

    it("refuses to create a gallery without a name", async () => {
        // given
        const user = renderProfile(owner);

        // when
        await openGalleryForm(user);

        // then
        expect(screen.getByRole("button", { name: "Create" })).toBeDisabled();
    });

    it("creates the gallery with a trimmed name and description", async () => {
        // given
        const user = renderProfile(owner);
        await openGalleryForm(user);

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
        const user = renderProfile(owner);
        await openGalleryForm(user);
        await user.type(screen.getByPlaceholderText("Gallery name"), "Witch Portraits");

        // when
        await user.click(screen.getByRole("button", { name: "Create" }));

        // then
        expect(screen.getByPlaceholderText("Gallery name")).toHaveValue("Witch Portraits");
        expect(mocks.refreshGalleries).not.toHaveBeenCalled();
    });

    it("lets the owner abandon the new gallery", async () => {
        // given
        const user = renderProfile(owner);
        await openGalleryForm(user);

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByPlaceholderText("Gallery name")).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Create Gallery" })).toBeInTheDocument();
    });
});

describe("ProfilePage pagination", () => {
    function onePostOf(total: number): [Mock, object] {
        return [
            mocks.useUserPosts,
            { posts: [{ id: "post-1", body: "hello" }], total, loading: false, refresh: mocks.refreshPosts },
        ];
    }

    const pagerCases: ProfileCase[] = [
        {
            name: "leaves the pager out while everything fits on one page",
            serves: onePostOf(1),
            noButtons: ["Next"],
        },
        {
            name: "walks the posts on to the next page",
            serves: onePostOf(45),
            press: ["Next"],
            fetches: [mocks.useUserPosts, profileId, 20, 20],
        },
        {
            name: "walks the posts back to the previous page",
            serves: onePostOf(45),
            press: ["Next", "Previous"],
            fetches: [mocks.useUserPosts, profileId, 20, 0],
        },
    ];

    it.each(pagerCases)("$name", checkProfileCase);
});

describe("ProfilePage character guide", () => {
    const guideCases: ProfileCase[] = [
        {
            name: "leaves an ordinary member profile alone",
            profile: { is_bot: false },
            shown: [emptyBio],
            hidden: [guide],
        },
        {
            name: "leaves a profile alone when the server sends no bot flag at all",
            shown: [emptyBio],
            hidden: [guide],
        },
        {
            name: "keeps a character's own bio above the guide",
            profile: { is_bot: true, bio: "The Golden Witch, endless and cruel." },
            shown: ["The Golden Witch, endless and cruel.", guide],
        },
        {
            name: "renders the guide in full under a very long bio",
            profile: { is_bot: true, bio: "Beatrice repeats herself endlessly. ".repeat(200) },
            shown: [guide, /read back over the last 20 messages/, /back through up to 25 messages/],
        },
        {
            name: "takes both memory limits from the live settings",
            siteInfo: { chatbot_context_messages: 42, chatbot_max_reply_chain: 99 },
            profile: { is_bot: true },
            shown: [/read back over the last 42 messages/, /back through up to 99 messages/],
            hidden: [/20 messages/, /25 messages/],
        },
    ];

    it.each(guideCases)("$name", checkProfileCase);
});
