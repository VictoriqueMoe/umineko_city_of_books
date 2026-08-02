import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WSMessageHandler } from "../../../context/notificationContextValue";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { Poll, Post, PostMedia, SharedContentPreview, User, UserProfile } from "../../../types/api";
import { PostCard } from "./PostCard";

const mocks = vi.hoisted(() => ({
    like: vi.fn(),
    unlike: vi.fn(),
    deletePost: vi.fn(),
    updatePost: vi.fn(),
    uploadPostMedia: vi.fn(),
    deletePostMedia: vi.fn(),
    createPost: vi.fn(),
    createComment: vi.fn(),
    uploadCommentMedia: vi.fn(),
    votePoll: vi.fn(),
    createReport: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../../api/mutations/post", () => ({
    useLikePost: () => ({ mutateAsync: mocks.like }),
    useUnlikePost: () => ({ mutateAsync: mocks.unlike }),
    useDeletePost: () => ({ mutateAsync: mocks.deletePost }),
    useUpdatePost: () => ({ mutateAsync: mocks.updatePost }),
    useUploadPostMedia: () => ({ mutateAsync: mocks.uploadPostMedia }),
    useDeletePostMedia: () => ({ mutateAsync: mocks.deletePostMedia }),
    useCreatePost: () => ({ mutateAsync: mocks.createPost }),
    useCreateComment: () => ({ mutateAsync: mocks.createComment }),
    useUploadCommentMedia: () => ({ mutateAsync: mocks.uploadCommentMedia }),
    useVotePoll: () => ({ mutateAsync: mocks.votePoll }),
}));

vi.mock("../../../api/mutations/misc", () => ({
    useCreateReport: () => ({ mutateAsync: mocks.createReport, isPending: false }),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocks.navigate };
});

const postId = "11111111-1111-1111-1111-111111111111";
const authorId = "00000000-0000-0000-0000-0000000000aa";
const strangerId = "00000000-0000-0000-0000-0000000000bb";

const author: User = {
    id: authorId,
    username: "beatrice",
    display_name: "Beatrice",
};

function makePost(overrides: Partial<Post> = {}): Post {
    return {
        id: postId,
        author,
        body: "Without love it cannot be seen",
        media: [],
        share_count: 0,
        like_count: 0,
        comment_count: 0,
        view_count: 0,
        user_liked: false,
        created_at: "2026-08-02T11:00:00Z",
        ...overrides,
    };
}

function makeMedia(id: number): PostMedia {
    return {
        id,
        media_url: `https://witch.test/media-${id}.png`,
        media_type: "image",
        sort_order: id,
    };
}

const ownerProfile = makeUser({ id: authorId, username: "beatrice", display_name: "Beatrice" });
const strangerProfile = makeUser({ id: strangerId, username: "battler", display_name: "Battler" });

interface CardOptions {
    user?: UserProfile | null;
    onDelete?: () => void;
    onEdit?: () => void;
    addWSListener?: (handler: WSMessageHandler) => () => void;
}

function renderCard(post: Post, options: CardOptions = {}) {
    return renderWithProviders(<PostCard post={post} onDelete={options.onDelete} onEdit={options.onEdit} />, {
        user: options.user ?? null,
        notification: options.addWSListener ? { addWSListener: options.addWSListener } : undefined,
    });
}

function likeButton(): HTMLElement {
    return screen.getByRole("button", { name: /[♡♥]/ });
}

beforeEach(() => {
    mocks.like.mockResolvedValue(undefined);
    mocks.unlike.mockResolvedValue(undefined);
    mocks.deletePost.mockResolvedValue(undefined);
    mocks.updatePost.mockResolvedValue(undefined);
    mocks.deletePostMedia.mockResolvedValue(undefined);
    mocks.uploadPostMedia.mockResolvedValue(makeMedia(9));
    mocks.createComment.mockResolvedValue({ id: "comment-1" });
});

describe("PostCard", () => {
    it("shows the author, the body and how long ago it was posted", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));

        // when
        renderCard(makePost());

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("Without love it cannot be seen")).toBeInTheDocument();
        expect(screen.getByText("1h")).toBeInTheDocument();
    });

    it("marks a post that has been edited since it was written", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));

        // when
        renderCard(makePost({ updated_at: "2026-08-02T11:30:00Z" }));

        // then
        expect(screen.getByText("1h (edited)")).toBeInTheDocument();
    });

    it("offers the author both edit and delete", () => {
        // given
        const user = ownerProfile;

        // when
        renderCard(makePost(), { user });

        // then
        expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("hides edit and delete from another reader", () => {
        // given
        const user = strangerProfile;

        // when
        renderCard(makePost(), { user });

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("gives a moderator edit and delete over somebody else's post", () => {
        // given
        const user = makeUser({ id: strangerId, username: "battler", role: "moderator" });

        // when
        renderCard(makePost(), { user });

        // then
        expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("hides everything that needs an account from a signed out reader", () => {
        // given
        const user = null;

        // when
        renderCard(makePost({ share_count: 2 }), { user });

        // then
        expect(likeButton()).toBeDisabled();
        expect(screen.queryByRole("button", { name: /Reply/ })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /^Share/ })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Report" })).not.toBeInTheDocument();
    });

    it("offers report on another reader's post but never on your own", () => {
        // given
        const post = makePost();

        // when
        const { unmount } = renderCard(post, { user: strangerProfile });

        // then
        expect(screen.getByRole("button", { name: "Report" })).toBeInTheDocument();
        unmount();
        renderCard(post, { user: ownerProfile });
        expect(screen.queryByRole("button", { name: "Report" })).not.toBeInTheDocument();
    });

    it("likes the post optimistically before the request settles", async () => {
        // given
        const testUser = userEvent.setup();
        renderCard(makePost({ like_count: 3 }), { user: strangerProfile });

        // when
        await testUser.click(likeButton());

        // then
        expect(mocks.like).toHaveBeenCalledWith(postId);
        expect(screen.getByRole("button", { name: "♥ 4" })).toBeInTheDocument();
    });

    it("unlikes a post that the reader has already liked", async () => {
        // given
        const testUser = userEvent.setup();
        renderCard(makePost({ like_count: 3, user_liked: true }), { user: strangerProfile });

        // when
        await testUser.click(likeButton());

        // then
        expect(mocks.unlike).toHaveBeenCalledWith(postId);
        expect(screen.getByRole("button", { name: "♡ 2" })).toBeInTheDocument();
    });

    it("puts the like back when the request is refused", async () => {
        // given
        const testUser = userEvent.setup();
        mocks.like.mockRejectedValue(new Error("the golden truth denies it"));
        renderCard(makePost({ like_count: 3 }), { user: strangerProfile });

        // when
        await testUser.click(likeButton());

        // then
        expect(screen.getByRole("button", { name: "♡ 3" })).toBeInTheDocument();
    });

    it("counts a like that arrives over the websocket from someone else", () => {
        // given
        const handlers: WSMessageHandler[] = [];
        const addWSListener = (handler: WSMessageHandler) => {
            handlers.push(handler);
            return () => {};
        };
        renderCard(makePost({ like_count: 3 }), { user: strangerProfile, addWSListener });

        // when
        act(() => {
            for (const handler of handlers) {
                handler({ type: "post_like", data: { post_id: postId, delta: 1 } });
            }
        });

        // then
        expect(screen.getByRole("button", { name: "♡ 4" })).toBeInTheDocument();
    });

    it("ignores a websocket like that is the echo of the reader's own", async () => {
        // given
        const testUser = userEvent.setup();
        const handlers: WSMessageHandler[] = [];
        const addWSListener = (handler: WSMessageHandler) => {
            handlers.push(handler);
            return () => {};
        };
        renderCard(makePost({ like_count: 3 }), { user: strangerProfile, addWSListener });
        await testUser.click(likeButton());

        // when
        act(() => {
            for (const handler of handlers) {
                handler({ type: "post_like", data: { post_id: postId, delta: 1 } });
            }
        });

        // then
        expect(screen.getByRole("button", { name: "♥ 4" })).toBeInTheDocument();
    });

    it("ignores both echoes when two of the reader's own toggles are in flight", async () => {
        // given
        const testUser = userEvent.setup();
        const handlers: WSMessageHandler[] = [];
        const addWSListener = (handler: WSMessageHandler) => {
            handlers.push(handler);
            return () => {};
        };
        renderCard(makePost({ like_count: 3 }), { user: strangerProfile, addWSListener });
        await testUser.click(likeButton());
        await testUser.click(likeButton());

        // when
        act(() => {
            for (const handler of handlers) {
                handler({ type: "post_like", data: { post_id: postId, delta: 1 } });
                handler({ type: "post_like", data: { post_id: postId, delta: -1 } });
            }
        });

        // then
        expect(screen.getByRole("button", { name: "♡ 3" })).toBeInTheDocument();
    });

    it("ignores a websocket like meant for a different post", () => {
        // given
        const handlers: WSMessageHandler[] = [];
        const addWSListener = (handler: WSMessageHandler) => {
            handlers.push(handler);
            return () => {};
        };
        renderCard(makePost({ like_count: 3 }), { user: strangerProfile, addWSListener });

        // when
        act(() => {
            for (const handler of handlers) {
                handler({ type: "post_like", data: { post_id: "another-post", delta: 5 } });
            }
        });

        // then
        expect(screen.getByRole("button", { name: "♡ 3" })).toBeInTheDocument();
    });

    it("copies the canonical link to the post to the clipboard", async () => {
        // given
        const testUser = userEvent.setup();
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        renderCard(makePost());

        // when
        await testUser.click(screen.getByRole("button", { name: "Copy Link" }));

        // then
        expect(writeText).toHaveBeenCalledWith(`https://whentheycry.social/game-board/${postId}`);
    });

    it("handles a refused clipboard rather than leaving the rejection unhandled", async () => {
        // given
        const testUser = userEvent.setup();
        const attachCatch = vi.fn();
        vi.spyOn(navigator.clipboard, "writeText").mockReturnValue({
            catch: attachCatch,
        } as unknown as Promise<void>);
        renderCard(makePost());

        // when
        await testUser.click(screen.getByRole("button", { name: "Copy Link" }));

        // then
        expect(attachCatch).toHaveBeenCalledOnce();
    });

    it("asks before deleting and leaves the post alone when refused", async () => {
        // given
        const testUser = userEvent.setup();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const onDelete = vi.fn();
        renderCard(makePost(), { user: ownerProfile, onDelete });

        // when
        await testUser.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(confirm).toHaveBeenCalledOnce();
        expect(mocks.deletePost).not.toHaveBeenCalled();
        expect(onDelete).not.toHaveBeenCalled();
    });

    it("deletes the post and tells the owner once it is confirmed", async () => {
        // given
        const testUser = userEvent.setup();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const onDelete = vi.fn();
        renderCard(makePost(), { user: ownerProfile, onDelete });

        // when
        await testUser.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(mocks.deletePost).toHaveBeenCalledWith(postId);
        expect(onDelete).toHaveBeenCalledOnce();
    });

    it("leaves the list alone when the delete request fails", async () => {
        // given
        const testUser = userEvent.setup();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        mocks.deletePost.mockRejectedValue(new Error("the witch protects it"));
        const onDelete = vi.fn();
        renderCard(makePost(), { user: ownerProfile, onDelete });

        // when
        await testUser.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(mocks.deletePost).toHaveBeenCalledWith(postId);
        expect(onDelete).not.toHaveBeenCalled();
    });

    it("opens the editor seeded with the current body and saves it trimmed", async () => {
        // given
        const testUser = userEvent.setup();
        const onEdit = vi.fn();
        renderCard(makePost(), { user: ownerProfile, onEdit });

        // when
        await testUser.click(screen.getByRole("button", { name: "Edit" }));
        const textarea = screen.getByRole("textbox");
        expect(textarea).toHaveValue("Without love it cannot be seen");
        await testUser.clear(textarea);
        await testUser.type(textarea, "  the truth is golden  ");
        await testUser.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.updatePost).toHaveBeenCalledWith("the truth is golden");
        expect(onEdit).toHaveBeenCalledOnce();
        expect(screen.getByText("the truth is golden")).toBeInTheDocument();
    });

    it("refuses to save an edit that has been emptied", async () => {
        // given
        const testUser = userEvent.setup();
        renderCard(makePost(), { user: ownerProfile });

        // when
        await testUser.click(screen.getByRole("button", { name: "Edit" }));
        await testUser.clear(screen.getByRole("textbox"));

        // then
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
        expect(mocks.updatePost).not.toHaveBeenCalled();
    });

    it("leaves the body untouched when the edit is cancelled", async () => {
        // given
        const testUser = userEvent.setup();
        renderCard(makePost(), { user: ownerProfile });

        // when
        await testUser.click(screen.getByRole("button", { name: "Edit" }));
        await testUser.clear(screen.getByRole("textbox"));
        await testUser.type(screen.getByRole("textbox"), "a different truth");
        await testUser.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(mocks.updatePost).not.toHaveBeenCalled();
        expect(screen.getByText("Without love it cannot be seen")).toBeInTheDocument();
    });

    it("keeps the editor open when saving the edit fails", async () => {
        // given
        const testUser = userEvent.setup();
        mocks.updatePost.mockRejectedValue(new Error("the witch forbids it"));
        const onEdit = vi.fn();
        renderCard(makePost(), { user: ownerProfile, onEdit });

        // when
        await testUser.click(screen.getByRole("button", { name: "Edit" }));
        await testUser.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(screen.getByRole("textbox")).toBeInTheDocument();
        expect(onEdit).not.toHaveBeenCalled();
    });

    it("drops an attachment from the editor once the server has removed it", async () => {
        // given
        const testUser = userEvent.setup();
        const { container } = renderCard(makePost({ media: [makeMedia(4)] }), { user: ownerProfile });

        // when
        await testUser.click(screen.getByRole("button", { name: "Edit" }));
        await testUser.click(screen.getByRole("button", { name: "×" }));

        // then
        expect(mocks.deletePostMedia).toHaveBeenCalledWith(4);
        expect(container.querySelectorAll("img")).toHaveLength(0);
    });

    it("adds freshly uploaded media to the editor", async () => {
        // given
        const testUser = userEvent.setup();
        const { container } = renderCard(makePost(), { user: ownerProfile });
        await testUser.click(screen.getByRole("button", { name: "Edit" }));

        // when
        const input = container.querySelector('input[type="file"]') as HTMLInputElement;
        fireEvent.change(input, { target: { files: [new File(["x"], "beato.png", { type: "image/png" })] } });

        // then
        expect(await screen.findByRole("button", { name: "×" })).toBeInTheDocument();
        expect(mocks.uploadPostMedia).toHaveBeenCalledOnce();
    });

    it("shows the view count only once somebody has read the post", () => {
        // given
        const post = makePost({ view_count: 0 });

        // when
        const { unmount } = renderCard(post);

        // then
        expect(screen.queryByText(/views/)).not.toBeInTheDocument();
        unmount();
        renderCard(makePost({ view_count: 12 }));
        expect(screen.getByText("12 views")).toBeInTheDocument();
    });

    it("links the comment count through to the post", () => {
        // given
        const post = makePost({ comment_count: 4 });

        // when
        renderCard(post);

        // then
        expect(screen.getByRole("button", { name: /4/ })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /4/ })).toHaveAttribute("href", `/game-board/${postId}`);
    });

    it("opens the share dialog from the share control", async () => {
        // given
        const testUser = userEvent.setup();
        renderCard(makePost({ share_count: 2 }), { user: strangerProfile });

        // when
        await testUser.click(screen.getByRole("button", { name: "Share 2" }));

        // then
        expect(screen.getByText("Share to Game Board")).toBeInTheDocument();
        expect(screen.getByText("Sharing: Without love it cannot be seen")).toBeInTheDocument();
    });

    it("toggles the quick reply box open and shut", async () => {
        // given
        const testUser = userEvent.setup();
        renderCard(makePost(), { user: strangerProfile });

        // when
        await testUser.click(screen.getByRole("button", { name: /Reply/ }));

        // then
        expect(screen.getByPlaceholderText("Write a comment...")).toBeInTheDocument();
        await testUser.click(screen.getByRole("button", { name: /Reply/ }));
        expect(screen.queryByPlaceholderText("Write a comment...")).not.toBeInTheDocument();
    });

    it("renders the attached poll", () => {
        // given
        const poll: Poll = {
            id: "poll-1",
            options: [
                { id: 0, label: "Kanon", vote_count: 2, percent: 66.6 },
                { id: 1, label: "Shannon", vote_count: 1, percent: 33.4 },
            ],
            total_votes: 3,
            user_voted_option: null,
            expired: false,
            expires_at: "2126-01-01T00:00:00Z",
            duration_seconds: 86400,
        };

        // when
        renderCard(makePost({ poll }), { user: strangerProfile });

        // then
        expect(screen.getByText("Kanon")).toBeInTheDocument();
        expect(screen.getByText("Shannon")).toBeInTheDocument();
        expect(screen.getByText("3 votes")).toBeInTheDocument();
    });

    it("renders a gif instead of text when the body is nothing but a giphy link", () => {
        // given
        const body = "https://media.giphy.com/media/abc123/beato.gif";

        // when
        const { container } = renderCard(makePost({ body }));

        // then
        const image = container.querySelector("img");
        expect(image).toHaveAttribute("src", body);
        expect(screen.queryByText(body)).not.toBeInTheDocument();
    });

    it("opens the post when the body itself is clicked", async () => {
        // given
        const testUser = userEvent.setup();
        renderCard(makePost());

        // when
        await testUser.click(screen.getByText("Without love it cannot be seen"));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith(`/game-board/${postId}`);
    });

    it("leaves the post alone when a link inside the body is clicked", async () => {
        // given
        const testUser = userEvent.setup();
        const shared: SharedContentPreview = {
            id: "shared-1",
            content_type: "theory",
            title: "Kanon is Yasu",
            deleted: false,
            url: "/theory/shared-1",
        };
        renderCard(makePost({ shared_content: shared }));

        // when
        await testUser.click(screen.getByRole("link", { name: /Kanon is Yasu/ }));

        // then
        expect(mocks.navigate).not.toHaveBeenCalled();
    });

    it("opens the post in a new tab on a middle click", () => {
        // given
        const open = vi.spyOn(window, "open").mockReturnValue(null);
        renderCard(makePost());

        // when
        fireEvent(
            screen.getByText("Without love it cannot be seen"),
            new MouseEvent("auxclick", { bubbles: true, cancelable: true, button: 1 }),
        );

        // then
        expect(open).toHaveBeenCalledWith(`/game-board/${postId}`, "_blank");
    });
});
