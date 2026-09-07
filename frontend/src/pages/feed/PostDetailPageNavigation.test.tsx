import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { PostDetail, PostMedia, SharedContentPreview, User } from "../../types/api";
import { PostDetailPage } from "./PostDetailPage";

const { usePost, refresh } = vi.hoisted(() => ({ usePost: vi.fn(), refresh: vi.fn() }));

vi.mock("../../hooks/queries/post", () => ({ usePost }));

vi.mock("../../components/post/CommentsSection/CommentsSection", () => ({
    CommentsSection: () => <section data-testid="comments-section" />,
}));

const featherine: User = { id: "u-featherine", username: "featherine", display_name: "Featherine" };
const aria: User = { id: "u-aria", username: "aria", display_name: "Aria" };

const outerBody = "based";
const innerBody = "furudo erika if she was a beautiful black woman";

const innerImage: PostMedia = {
    id: 77,
    media_url: "https://witch.test/erika.png",
    media_type: "image",
    sort_order: 0,
};

const innerPreview: SharedContentPreview = {
    id: "post-inner",
    content_type: "post",
    body: innerBody,
    author: aria,
    media: [innerImage],
    deleted: false,
    url: "/game-board/post-inner",
};

function makeDetail(overrides: Partial<PostDetail>): PostDetail {
    return {
        id: "post-outer",
        author: featherine,
        body: outerBody,
        media: [],
        share_count: 0,
        like_count: 0,
        comment_count: 0,
        view_count: 0,
        user_liked: false,
        created_at: "2026-09-01T10:00:00Z",
        comments: [],
        liked_by: [],
        viewer_blocked: false,
        ...overrides,
    };
}

const outerPost = makeDetail({ id: "post-outer", author: featherine, body: outerBody, shared_content: innerPreview });
const innerPost = makeDetail({ id: "post-inner", author: aria, body: innerBody, media: [innerImage] });

const posts: Record<string, PostDetail> = { "post-outer": outerPost, "post-inner": innerPost };

function servePostsImmediately(): void {
    usePost.mockImplementation((id: string) => ({ post: posts[id] ?? null, loading: false, refresh }));
}

function renderOuter() {
    return renderWithProviders(<PostDetailPage />, { route: "/game-board/post-outer", path: "/game-board/:id" });
}

describe("PostDetailPage navigation between posts", () => {
    it("shows the shared post's own body after following its embed", async () => {
        // given
        servePostsImmediately();
        const user = userEvent.setup();
        const { container } = renderOuter();
        expect(screen.getByText(outerBody)).toBeInTheDocument();

        // when
        await user.click(screen.getByRole("link", { name: new RegExp(innerBody) }));

        // then
        await waitFor(() => {
            expect(screen.getByRole("link", { name: /Aria/ })).toBeInTheDocument();
        });
        expect(usePost).toHaveBeenLastCalledWith("post-inner");
        expect(container.querySelector('a[href="/game-board/post-inner"]')).not.toBeNull();
        expect(screen.getByText(innerBody)).toBeInTheDocument();
        expect(screen.queryByText(outerBody)).not.toBeInTheDocument();
        expect(container.querySelector(`img[src="${innerImage.media_url}"]`)).not.toBeNull();
    });

    it("shows the shared post's own body when the target had to be fetched first", async () => {
        // given
        let innerReady = false;
        usePost.mockImplementation((id: string) => {
            if (id === "post-inner" && !innerReady) {
                return { post: null, loading: true, refresh };
            }

            return { post: posts[id] ?? null, loading: false, refresh };
        });
        const user = userEvent.setup();
        const { container, rerender } = renderOuter();

        // when
        await user.click(screen.getByRole("link", { name: new RegExp(innerBody) }));
        expect(screen.getByText("Loading post...")).toBeInTheDocument();
        innerReady = true;
        act(() => {
            rerender(<PostDetailPage />);
        });

        // then
        expect(screen.getByText(innerBody)).toBeInTheDocument();
        expect(screen.queryByText(outerBody)).not.toBeInTheDocument();
        expect(container.querySelector(`img[src="${innerImage.media_url}"]`)).not.toBeNull();
    });

    it("shows the shared post's own body on a fresh visit to its address", () => {
        // given
        servePostsImmediately();

        // when
        const { container } = renderWithProviders(<PostDetailPage />, {
            route: "/game-board/post-inner",
            path: "/game-board/:id",
        });

        // then
        expect(screen.getByText(innerBody)).toBeInTheDocument();
        expect(container.querySelector(`img[src="${innerImage.media_url}"]`)).not.toBeNull();
    });
});
