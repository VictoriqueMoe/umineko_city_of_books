import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { Post, UserProfile } from "../../types/api";
import { SocialFeedPage } from "./SocialFeedPage";

const { usePostFeed, useUpdateGameBoardSort } = vi.hoisted(() => ({
    usePostFeed: vi.fn(),
    useUpdateGameBoardSort: vi.fn(),
}));

vi.mock("../../api/queries/post", () => ({ usePostFeed }));
vi.mock("../../api/mutations/auth", () => ({ useUpdateGameBoardSort }));
vi.mock("../../components/AnnouncementCard/AnnouncementCard", () => ({
    AnnouncementCard: () => <div data-testid="announcement-card" />,
}));
vi.mock("../../components/RulesBox/RulesBox", () => ({
    RulesBox: ({ page }: { page: string }) => <div data-testid="rules-box">{page}</div>,
}));
vi.mock("../../components/LiveStrip/LiveStrip", () => ({
    LiveStrip: ({ corner }: { corner: string }) => <div data-testid="live-strip">{corner}</div>,
}));
vi.mock("../../components/post/PostComposer/PostComposer", () => ({
    PostComposer: ({ corner }: { corner: string }) => <div data-testid="post-composer">{corner}</div>,
}));
vi.mock("../../components/post/PostCard/PostCard", () => ({
    PostCard: ({ post }: { post: Post }) => <article data-testid="post-card">{post.body}</article>,
}));

const author = { id: "author-1", username: "beatrice", display_name: "Beatrice" };
const reader = makeUser({ id: "reader-1", username: "battler", display_name: "Battler" });

function makePost(overrides: Partial<Post> = {}): Post {
    return {
        id: "post-1",
        author,
        body: "The witch is watching.",
        media: [],
        share_count: 0,
        like_count: 0,
        comment_count: 0,
        view_count: 0,
        user_liked: false,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

interface StubOptions {
    posts?: Post[];
    total?: number;
    loading?: boolean;
    offset?: number;
    hasNext?: boolean;
    hasPrev?: boolean;
}

function stubFeed(options: StubOptions = {}) {
    const refresh = vi.fn();
    usePostFeed.mockReturnValue({
        posts: options.posts ?? [],
        total: options.total ?? options.posts?.length ?? 0,
        loading: options.loading ?? false,
        offset: options.offset ?? 0,
        limit: 20,
        hasNext: options.hasNext ?? false,
        hasPrev: options.hasPrev ?? false,
        refresh,
    });

    const mutate = vi.fn();
    useUpdateGameBoardSort.mockReturnValue({ mutate });

    return { refresh, mutate };
}

function renderPage(user: UserProfile | null, route = "/game-board", setUser = vi.fn(), corner?: string) {
    return renderWithProviders(<SocialFeedPage corner={corner} />, { user, auth: { setUser }, route });
}

describe("SocialFeedPage", () => {
    it("asks for the newest page of the general corner for everyone by default", () => {
        // given
        stubFeed();

        // when
        renderPage(null);

        // then
        expect(usePostFeed).toHaveBeenLastCalledWith("everyone", "general", undefined, "relevance", 1);
    });

    it("reads the tab, sort, search and page the address bar asked for", () => {
        // given
        stubFeed();

        // when
        renderPage(reader, "/game-board?tab=following&sort=likes&search=golden&page=3");

        // then
        expect(usePostFeed).toHaveBeenLastCalledWith("following", "general", "golden", "likes", 3);
    });

    it("falls back to the sort the reader saved on their profile", () => {
        // given
        stubFeed();
        const saved = makeUser({ id: "reader-1", private: { game_board_sort: "views" } });

        // when
        renderPage(saved);

        // then
        expect(usePostFeed).toHaveBeenLastCalledWith("everyone", "general", undefined, "views", 1);
    });

    it("names the corner it was mounted for and asks for that corner's rules", () => {
        // given
        stubFeed();

        // when
        renderPage(null, "/game-board/umineko", vi.fn(), "umineko");

        // then
        expect(screen.getByRole("heading", { name: "Umineko Corner" })).toBeInTheDocument();
        expect(screen.getByTestId("rules-box")).toHaveTextContent("game_board_umineko");
        expect(screen.getByTestId("live-strip")).toHaveTextContent("umineko");
    });

    it("leaves the general board without a corner title", () => {
        // given
        stubFeed();

        // when
        renderPage(null);

        // then
        expect(screen.queryByRole("heading", { name: /Corner/ })).not.toBeInTheDocument();
        expect(screen.getByTestId("rules-box")).toHaveTextContent("game_board");
    });

    it("keeps the following tab out of reach of a signed out visitor", () => {
        // given
        stubFeed();

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("button", { name: "Following" })).toBeDisabled();
    });

    it("lets a signed in member switch to the people they follow", async () => {
        // given
        stubFeed();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "Following" }));

        // then
        expect(usePostFeed).toHaveBeenLastCalledWith("following", "general", undefined, "relevance", 1);
    });

    it("hides the composer from a signed out visitor", () => {
        // given
        stubFeed();

        // when
        renderPage(null);

        // then
        expect(screen.queryByTestId("post-composer")).not.toBeInTheDocument();
    });

    it("offers a signed in member a composer for the corner they are reading", () => {
        // given
        stubFeed();

        // when
        renderPage(reader, "/game-board/higurashi", vi.fn(), "higurashi");

        // then
        expect(screen.getByTestId("post-composer")).toHaveTextContent("higurashi");
    });

    it("remembers the sort a signed in member picked", async () => {
        // given
        const { mutate } = stubFeed();
        const setUser = vi.fn();
        const user = userEvent.setup();
        renderPage(reader, "/game-board", setUser);

        // when
        await user.click(screen.getByRole("button", { name: "Most Liked" }));

        // then
        expect(usePostFeed).toHaveBeenLastCalledWith("everyone", "general", undefined, "likes", 1);
        expect(mutate).toHaveBeenCalledWith("likes");
        expect(setUser).toHaveBeenCalledWith(expect.objectContaining({ private: { game_board_sort: "likes" } }));
    });

    it("does not try to save a sort for a signed out visitor", async () => {
        // given
        const { mutate } = stubFeed();
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.click(screen.getByRole("button", { name: "Most Viewed" }));

        // then
        expect(usePostFeed).toHaveBeenLastCalledWith("everyone", "general", undefined, "views", 1);
        expect(mutate).not.toHaveBeenCalled();
    });

    it("consults the game board while the posts are loading", () => {
        // given
        stubFeed({ loading: true, posts: [makePost()] });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Consulting the game board...")).toBeInTheDocument();
        expect(screen.queryByTestId("post-card")).not.toBeInTheDocument();
    });

    it("invites the first post when the board is empty", () => {
        // given
        stubFeed({ posts: [] });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("No posts yet. Be the first to post.")).toBeInTheDocument();
    });

    it("says the search found nothing when a search is active", () => {
        // given
        stubFeed({ posts: [] });

        // when
        renderPage(null, "/game-board?search=golden");

        // then
        expect(screen.getByText("No posts match your search.")).toBeInTheDocument();
    });

    it("says the following tab is quiet when nobody followed has posted", () => {
        // given
        stubFeed({ posts: [] });

        // when
        renderPage(reader, "/game-board?tab=following");

        // then
        expect(screen.getByText("No posts from people you follow yet.")).toBeInTheDocument();
    });

    it("lists a card for every post on the board", () => {
        // given
        stubFeed({
            posts: [
                makePost({ id: "post-1", body: "Without love it cannot be seen." }),
                makePost({ id: "post-2", body: "The golden witch laughs." }),
            ],
        });

        // when
        renderPage(null);

        // then
        expect(screen.getAllByTestId("post-card")).toHaveLength(2);
        expect(screen.getByText("Without love it cannot be seen.")).toBeInTheDocument();
    });

    it("waits for the typing to settle before searching", () => {
        // given
        vi.useFakeTimers();
        stubFeed();
        renderPage(null);

        // when
        fireEvent.change(screen.getByPlaceholderText("Search posts..."), { target: { value: "golden" } });

        // then
        expect(usePostFeed).not.toHaveBeenCalledWith("everyone", "general", "golden", "relevance", 1);
        act(() => {
            vi.advanceTimersByTime(300);
        });
        expect(usePostFeed).toHaveBeenLastCalledWith("everyone", "general", "golden", "relevance", 1);
    });

    it("only searches for the last thing typed", () => {
        // given
        vi.useFakeTimers();
        stubFeed();
        renderPage(null);
        const field = screen.getByPlaceholderText("Search posts...");

        // when
        fireEvent.change(field, { target: { value: "gol" } });
        act(() => {
            vi.advanceTimersByTime(200);
        });
        fireEvent.change(field, { target: { value: "golden" } });
        act(() => {
            vi.advanceTimersByTime(300);
        });

        // then
        expect(usePostFeed).not.toHaveBeenCalledWith("everyone", "general", "gol", "relevance", 1);
        expect(usePostFeed).toHaveBeenLastCalledWith("everyone", "general", "golden", "relevance", 1);
    });

    it("hides the pager while the posts are loading", () => {
        // given
        stubFeed({ loading: true, total: 40, hasNext: true });

        // when
        renderPage(null);

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("pages forward through the board", async () => {
        // given
        stubFeed({ posts: [makePost()], total: 45, hasNext: true });
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(usePostFeed).toHaveBeenLastCalledWith("everyone", "general", undefined, "relevance", 2);
    });

    it("pages back through the board", async () => {
        // given
        stubFeed({ posts: [makePost()], total: 45, offset: 20, hasPrev: true });
        const user = userEvent.setup();
        renderPage(null, "/game-board?page=2");

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(usePostFeed).toHaveBeenLastCalledWith("everyone", "general", undefined, "relevance", 1);
    });
});
