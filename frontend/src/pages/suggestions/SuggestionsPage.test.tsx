import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Post, UserProfile } from "../../types/api";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { SuggestionsPage } from "./SuggestionsPage";

const mocks = vi.hoisted(() => ({
    usePostFeed: vi.fn(),
    resolve: vi.fn(),
    unresolve: vi.fn(),
    refresh: vi.fn(),
}));

vi.mock("../../api/queries/post", () => ({ usePostFeed: mocks.usePostFeed }));

vi.mock("../../api/mutations/post", () => ({
    useResolveSuggestion: () => ({ mutateAsync: mocks.resolve }),
    useUnresolveSuggestion: () => ({ mutateAsync: mocks.unresolve }),
}));

vi.mock("../../components/post/PostCard/PostCard", () => ({
    PostCard: ({ post, extraActions }: { post: Post; extraActions?: ReactNode }) => (
        <div data-testid="post-card">
            <span>{post.body}</span>
            {extraActions}
        </div>
    ),
}));

vi.mock("../../components/post/PostComposer/PostComposer", () => ({
    PostComposer: ({ corner }: { corner: string }) => <div data-testid="post-composer">{corner}</div>,
}));

vi.mock("../../components/RulesBox/RulesBox", () => ({
    RulesBox: ({ page }: { page: string }) => <div data-testid="rules-box">{page}</div>,
}));

function makePost(overrides: Partial<Post> = {}): Post {
    return {
        id: "post-1",
        author: { id: "user-1", username: "battler", display_name: "Battler" },
        body: "Add a dark mode for the game board",
        media: [],
        share_count: 0,
        like_count: 0,
        comment_count: 0,
        view_count: 0,
        user_liked: false,
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

interface FeedState {
    posts?: Post[];
    total?: number;
    loading?: boolean;
    offset?: number;
    hasNext?: boolean;
    hasPrev?: boolean;
}

function feedState(state: FeedState = {}) {
    return {
        posts: state.posts ?? [],
        total: state.total ?? 0,
        loading: state.loading ?? false,
        offset: state.offset ?? 0,
        limit: 20,
        hasNext: state.hasNext ?? false,
        hasPrev: state.hasPrev ?? false,
        refresh: mocks.refresh,
    };
}

interface SetupOptions {
    user?: UserProfile | null;
    feed?: FeedState;
}

function setup(options: SetupOptions = {}) {
    mocks.usePostFeed.mockReturnValue(feedState(options.feed));
    const user = userEvent.setup();
    const result = renderWithProviders(<SuggestionsPage />, { user: options.user ?? null });

    return { user, ...result };
}

beforeEach(() => {
    mocks.usePostFeed.mockReturnValue(feedState());
    mocks.resolve.mockResolvedValue(undefined);
    mocks.unresolve.mockResolvedValue(undefined);
});

describe("SuggestionsPage feed request", () => {
    it("asks for the newest open suggestions on the first page", () => {
        // given
        const options = {};

        // when
        setup(options);

        // then
        expect(mocks.usePostFeed).toHaveBeenLastCalledWith("everyone", "suggestions", undefined, "new", 1, "open");
    });

    it("drops the status filter when every suggestion is wanted", async () => {
        // given
        const { user } = setup();

        // when
        await user.selectOptions(screen.getByRole("combobox"), "");

        // then
        expect(mocks.usePostFeed).toHaveBeenLastCalledWith("everyone", "suggestions", undefined, "new", 1, undefined);
    });

    it("asks for the archived suggestions when that status is chosen", async () => {
        // given
        const { user } = setup();

        // when
        await user.selectOptions(screen.getByRole("combobox"), "archived");

        // then
        expect(mocks.usePostFeed).toHaveBeenLastCalledWith("everyone", "suggestions", undefined, "new", 1, "archived");
    });

    it("walks forward through the suggestions", async () => {
        // given
        const { user } = setup({ feed: { posts: [makePost()], total: 45, hasNext: true } });

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(mocks.usePostFeed).toHaveBeenLastCalledWith("everyone", "suggestions", undefined, "new", 2, "open");
    });

    it("returns to the first page when the status filter changes", async () => {
        // given
        const { user } = setup({ feed: { posts: [makePost()], total: 45, hasNext: true } });
        await user.click(screen.getByRole("button", { name: "Next" }));

        // when
        await user.selectOptions(screen.getByRole("combobox"), "done");

        // then
        expect(mocks.usePostFeed).toHaveBeenLastCalledWith("everyone", "suggestions", undefined, "new", 1, "done");
    });
});

describe("SuggestionsPage states", () => {
    it("explains what the corner is for", () => {
        // given
        const options = {};

        // when
        setup(options);

        // then
        expect(screen.getByRole("heading", { name: "Site Improvements" })).toBeInTheDocument();
        expect(screen.getByTestId("rules-box")).toHaveTextContent("suggestions");
    });

    it("waits quietly while the suggestions load", () => {
        // given
        const feed = { loading: true };

        // when
        setup({ feed });

        // then
        expect(screen.getByText("Loading suggestions...")).toBeInTheDocument();
        expect(screen.queryByTestId("post-card")).not.toBeInTheDocument();
    });

    it("invites the first idea when the corner is empty", () => {
        // given
        const feed = { posts: [] };

        // when
        setup({ feed });

        // then
        expect(screen.getByText("No suggestions yet. Be the first to share your ideas!")).toBeInTheDocument();
    });

    it("shows a card for every suggestion", () => {
        // given
        const feed = { posts: [makePost(), makePost({ id: "post-2", body: "Bigger avatars" })], total: 2 };

        // when
        setup({ feed });

        // then
        expect(screen.getAllByTestId("post-card")).toHaveLength(2);
    });

    it("hides the composer from a signed out visitor", () => {
        // given
        const signedOut = null;

        // when
        setup({ user: signedOut });

        // then
        expect(screen.queryByTestId("post-composer")).not.toBeInTheDocument();
    });

    it("offers the composer to a signed in member", () => {
        // given
        const member = makeUser();

        // when
        setup({ user: member });

        // then
        expect(screen.getByTestId("post-composer")).toHaveTextContent("suggestions");
    });

    it("badges a suggestion that has been done", () => {
        // given
        const feed = { posts: [makePost({ resolved_status: "done" })], total: 1 };

        // when
        setup({ feed });

        // then
        expect(screen.getByText("Done", { selector: "div" })).toBeInTheDocument();
    });

    it("badges a suggestion that has been archived", () => {
        // given
        const feed = { posts: [makePost({ resolved_status: "archived" })], total: 1 };

        // when
        setup({ feed });

        // then
        expect(screen.getByText("Archived", { selector: "div" })).toBeInTheDocument();
    });
});

describe("SuggestionsPage resolving", () => {
    it("keeps the resolve controls away from an ordinary member", () => {
        // given
        const member = makeUser();

        // when
        setup({ user: member, feed: { posts: [makePost()], total: 1 } });

        // then
        expect(screen.queryByRole("button", { name: "Mark as Done" })).not.toBeInTheDocument();
    });

    it("keeps the resolve controls away from a moderator", () => {
        // given
        const moderator = makeUser({ role: "moderator" });

        // when
        setup({ user: moderator, feed: { posts: [makePost()], total: 1 } });

        // then
        expect(screen.queryByRole("button", { name: "Mark as Done" })).not.toBeInTheDocument();
    });

    it("offers an admin the resolve controls", () => {
        // given
        const admin = makeUser({ role: "admin" });

        // when
        setup({ user: admin, feed: { posts: [makePost()], total: 1 } });

        // then
        expect(screen.getByRole("button", { name: "Mark as Done" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Archive" })).toBeInTheDocument();
    });

    it("marks a suggestion as done and refreshes the corner", async () => {
        // given
        const { user } = setup({ user: makeUser({ role: "admin" }), feed: { posts: [makePost()], total: 1 } });

        // when
        await user.click(screen.getByRole("button", { name: "Mark as Done" }));

        // then
        expect(mocks.resolve).toHaveBeenCalledWith({ id: "post-1", status: "done" });
        expect(mocks.refresh).toHaveBeenCalled();
    });

    it("archives a suggestion the site will not act on", async () => {
        // given
        const { user } = setup({ user: makeUser({ role: "admin" }), feed: { posts: [makePost()], total: 1 } });

        // when
        await user.click(screen.getByRole("button", { name: "Archive" }));

        // then
        expect(mocks.resolve).toHaveBeenCalledWith({ id: "post-1", status: "archived" });
    });

    it("offers only an undo on a suggestion that is already resolved", () => {
        // given
        const feed = { posts: [makePost({ resolved_status: "done" })], total: 1 };

        // when
        setup({ user: makeUser({ role: "admin" }), feed });

        // then
        expect(screen.getByRole("button", { name: "Undo" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Mark as Done" })).not.toBeInTheDocument();
    });

    it("reopens a resolved suggestion and refreshes the corner", async () => {
        // given
        const feed = { posts: [makePost({ resolved_status: "archived" })], total: 1 };
        const { user } = setup({ user: makeUser({ role: "admin" }), feed });

        // when
        await user.click(screen.getByRole("button", { name: "Undo" }));

        // then
        expect(mocks.unresolve).toHaveBeenCalledWith("post-1");
        expect(mocks.refresh).toHaveBeenCalled();
    });
});
