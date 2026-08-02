import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { PostComment, PostDetail, UserProfile } from "../../types/api";
import { PostDetailPage } from "./PostDetailPage";

const { usePost, navigate } = vi.hoisted(() => ({
    usePost: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/post", () => ({ usePost }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});
vi.mock("../../components/post/PostCard/PostCard", () => ({
    PostCard: ({ post, onDelete, onEdit }: { post: PostDetail; onDelete?: () => void; onEdit?: () => void }) => (
        <article data-testid="post-card">
            <span>{post.body}</span>
            <button onClick={onDelete}>stub delete</button>
            <button onClick={onEdit}>stub edit</button>
        </article>
    ),
}));
vi.mock("../../components/post/CommentsSection/CommentsSection", () => ({
    CommentsSection: (props: {
        comments: PostComment[];
        targetId: string;
        user: UserProfile | null;
        onChanged: () => void;
        highlightedId?: string;
        viewerBlocked?: boolean;
        blockedText?: string;
    }) => (
        <section
            data-testid="comments-section"
            data-target={props.targetId}
            data-highlighted={props.highlightedId ?? ""}
            data-blocked={String(props.viewerBlocked)}
            data-blocked-text={props.blockedText ?? ""}
            data-viewer={props.user?.id ?? "anonymous"}
        >
            <span>{`${props.comments.length} comments`}</span>
            <button onClick={props.onChanged}>stub changed</button>
        </section>
    ),
}));

const author = { id: "author-1", username: "beatrice", display_name: "Beatrice" };
const reader = makeUser({ id: "reader-1", username: "battler", display_name: "Battler" });

function makeComment(overrides: Partial<PostComment> = {}): PostComment {
    return {
        id: "comment-1",
        author,
        body: "A fine theory.",
        media: [],
        like_count: 0,
        user_liked: false,
        created_at: "2026-07-01T11:00:00Z",
        ...overrides,
    };
}

function makePostDetail(overrides: Partial<PostDetail> = {}): PostDetail {
    return {
        id: "post-1",
        author,
        body: "Without love it cannot be seen.",
        media: [],
        share_count: 0,
        like_count: 2,
        comment_count: 1,
        view_count: 9,
        user_liked: false,
        created_at: "2026-07-01T10:00:00Z",
        comments: [makeComment()],
        liked_by: [],
        viewer_blocked: false,
        ...overrides,
    };
}

interface StubOptions {
    post?: PostDetail | null;
    loading?: boolean;
}

function stubPost(options: StubOptions = {}) {
    const refresh = vi.fn();
    usePost.mockReturnValue({
        post: options.post === undefined ? makePostDetail() : options.post,
        loading: options.loading ?? false,
        refresh,
    });

    return { refresh };
}

function renderPage(user: UserProfile | null = null, route = "/game-board/post-1") {
    return renderWithProviders(<PostDetailPage />, { user, route, path: "/game-board/:id" });
}

describe("PostDetailPage", () => {
    it("waits while the post is loading", () => {
        // given
        stubPost({ loading: true, post: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading post...")).toBeInTheDocument();
        expect(screen.queryByTestId("post-card")).not.toBeInTheDocument();
    });

    it("says the post is missing when the server has none", () => {
        // given
        stubPost({ post: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Post not found.")).toBeInTheDocument();
    });

    it("asks for the post named in the address", () => {
        // given
        stubPost();

        // when
        renderPage(null, "/game-board/post-42");

        // then
        expect(usePost).toHaveBeenLastCalledWith("post-42");
    });

    it("shows the post body and its comments", () => {
        // given
        stubPost({ post: makePostDetail({ comments: [makeComment({ id: "c1" }), makeComment({ id: "c2" })] }) });

        // when
        renderPage(reader);

        // then
        expect(screen.getByText("Without love it cannot be seen.")).toBeInTheDocument();
        expect(screen.getByText("2 comments")).toBeInTheDocument();
        expect(screen.getByTestId("comments-section")).toHaveAttribute("data-target", "post-1");
        expect(screen.getByTestId("comments-section")).toHaveAttribute("data-viewer", "reader-1");
    });

    it("leaves out the liked by row when nobody has liked the post", () => {
        // given
        stubPost({ post: makePostDetail({ liked_by: [] }) });

        // when
        renderPage();

        // then
        expect(screen.queryByRole("heading", { name: /Liked by/ })).not.toBeInTheDocument();
    });

    it("names everyone who liked the post", () => {
        // given
        stubPost({
            post: makePostDetail({
                liked_by: [author, { id: "u2", username: "ange", display_name: "Ange" }],
            }),
        });

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Liked by (2)" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Ange/ })).toHaveAttribute("href", "/user/ange");
    });

    it("tells the comments section the viewer is blocked from interacting", () => {
        // given
        stubPost({ post: makePostDetail({ viewer_blocked: true }) });

        // when
        renderPage(reader);

        // then
        const section = screen.getByTestId("comments-section");
        expect(section).toHaveAttribute("data-blocked", "true");
        expect(section).toHaveAttribute("data-blocked-text", "You cannot interact with this post.");
    });

    it("highlights the comment the notification linked to", () => {
        // given
        stubPost();

        // when
        renderPage(reader, "/game-board/post-1#comment-abc");

        // then
        expect(screen.getByTestId("comments-section")).toHaveAttribute("data-highlighted", "abc");
    });

    it("highlights nothing when the address carries no comment anchor", () => {
        // given
        stubPost();

        // when
        renderPage(reader, "/game-board/post-1#somewhere-else");

        // then
        expect(screen.getByTestId("comments-section")).toHaveAttribute("data-highlighted", "");
    });

    it("goes back to where the reader came from", async () => {
        // given
        stubPost();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText(/Back to Game Board/));

        // then
        expect(navigate).toHaveBeenCalledWith(-1);
    });

    it("returns to the game board once the post is deleted", async () => {
        // given
        stubPost();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "stub delete" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/game-board");
    });

    it("refetches the post after it is edited", async () => {
        // given
        const { refresh } = stubPost();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "stub edit" }));

        // then
        expect(refresh).toHaveBeenCalledOnce();
    });

    it("refetches the post after a comment changes", async () => {
        // given
        const { refresh } = stubPost();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "stub changed" }));

        // then
        expect(refresh).toHaveBeenCalledOnce();
    });
});
