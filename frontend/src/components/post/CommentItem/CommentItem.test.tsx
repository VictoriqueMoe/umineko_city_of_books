import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { PostComment, User, UserProfile } from "../../../types/api";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import { CommentItem } from "./CommentItem";

const { likeComment, unlikeComment, deleteComment, updateComment, createComment, uploadMedia } = vi.hoisted(() => ({
    likeComment: vi.fn(),
    unlikeComment: vi.fn(),
    deleteComment: vi.fn(),
    updateComment: vi.fn(),
    createComment: vi.fn(),
    uploadMedia: vi.fn(),
}));

vi.mock("../../../api/mutations/post", () => ({
    useLikeComment: () => ({ mutateAsync: likeComment }),
    useUnlikeComment: () => ({ mutateAsync: unlikeComment }),
    useDeleteComment: () => ({ mutateAsync: deleteComment }),
    useUpdateComment: () => ({ mutateAsync: updateComment }),
    useCreateComment: () => ({ mutateAsync: createComment }),
    useUploadCommentMedia: () => ({ mutateAsync: uploadMedia }),
}));

const EMPTY_HEART = "♡";
const FULL_HEART = "♥";
const NOW = new Date("2026-08-02T12:00:00Z");

const AUTHOR: User = { id: "user-1", username: "beatrice", display_name: "Beatrice" };
const OTHER: User = { id: "user-2", username: "battler", display_name: "Battler" };

function makeComment(overrides: Partial<PostComment> = {}): PostComment {
    return {
        id: "comment-1",
        author: AUTHOR,
        body: "Without love it cannot be seen",
        media: [],
        like_count: 0,
        user_liked: false,
        created_at: "2026-08-02T11:00:00Z",
        ...overrides,
    };
}

interface SetupOptions {
    comment?: PostComment;
    viewer?: UserProfile | null;
    onDelete?: () => void;
    viewerBlocked?: boolean;
    linkPrefix?: string;
    likeFn?: (id: string) => Promise<void>;
    unlikeFn?: (id: string) => Promise<void>;
    deleteFn?: (id: string) => Promise<void>;
    updateFn?: (id: string, body: string) => Promise<void>;
    createCommentFn?: (postId: string, body: string, parentId?: string) => Promise<{ id: string }>;
}

function setup(options: SetupOptions = {}) {
    return renderWithProviders(
        <CommentItem
            comment={options.comment ?? makeComment()}
            postId="post-1"
            onDelete={options.onDelete ?? (() => {})}
            viewerBlocked={options.viewerBlocked}
            linkPrefix={options.linkPrefix}
            likeFn={options.likeFn}
            unlikeFn={options.unlikeFn}
            deleteFn={options.deleteFn}
            updateFn={options.updateFn}
            createCommentFn={options.createCommentFn}
        />,
        { user: options.viewer === undefined ? makeUser({ id: OTHER.id, username: OTHER.username }) : options.viewer },
    );
}

function ownerViewer(): UserProfile {
    return makeUser({ id: AUTHOR.id, username: AUTHOR.username, display_name: AUTHOR.display_name });
}

describe("CommentItem", () => {
    it("shows the author, the body and how long ago it was written", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(NOW);

        // when
        setup({ comment: makeComment() });

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("Without love it cannot be seen")).toBeInTheDocument();
        expect(screen.getByText("1h")).toBeInTheDocument();
    });

    it("marks a comment that has been edited", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(NOW);

        // when
        setup({ comment: makeComment({ updated_at: "2026-08-02T11:30:00Z" }) });

        // then
        expect(screen.getByText("1h").parentElement).toHaveTextContent("1h (edited)");
    });

    it("shows a GIF in place of the body when the body is nothing but a GIF link", () => {
        // given
        const comment = makeComment({ body: "https://media.giphy.com/media/abc123/beato.gif" });

        // when
        setup({ comment });

        // then
        expect(screen.getByAltText("GIF")).toHaveAttribute("src", "https://media.giphy.com/media/abc123/beato.gif");
    });

    it("keeps the like control disabled for signed out viewers", () => {
        // given
        const viewer = null;

        // when
        setup({ comment: makeComment({ like_count: 3 }), viewer });

        // then
        expect(screen.getByRole("button", { name: `${EMPTY_HEART} 3` })).toBeDisabled();
        expect(screen.queryByRole("button", { name: "Reply" })).not.toBeInTheDocument();
    });

    it("counts a like straight away", async () => {
        // given
        const likeFn = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        setup({ comment: makeComment({ like_count: 3 }), likeFn });

        // when
        await user.click(screen.getByRole("button", { name: `${EMPTY_HEART} 3` }));

        // then
        expect(likeFn).toHaveBeenCalledWith("comment-1");
        expect(screen.getByRole("button", { name: `${FULL_HEART} 4` })).toBeInTheDocument();
    });

    it("takes back a like that the server refused", async () => {
        // given
        const likeFn = vi.fn(() => Promise.reject(new Error("no")));
        const user = userEvent.setup();
        setup({ comment: makeComment({ like_count: 3 }), likeFn });

        // when
        await user.click(screen.getByRole("button", { name: `${EMPTY_HEART} 3` }));

        // then
        await waitFor(() => expect(screen.getByRole("button", { name: `${EMPTY_HEART} 3` })).toBeInTheDocument());
    });

    it("removes a like that was already given", async () => {
        // given
        const unlikeFn = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        setup({ comment: makeComment({ like_count: 3, user_liked: true }), unlikeFn });

        // when
        await user.click(screen.getByRole("button", { name: `${FULL_HEART} 3` }));

        // then
        expect(unlikeFn).toHaveBeenCalledWith("comment-1");
        expect(screen.getByRole("button", { name: `${EMPTY_HEART} 2` })).toBeInTheDocument();
    });

    it("hides the count once the last like is taken back", async () => {
        // given
        const unlikeFn = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        setup({ comment: makeComment({ like_count: 1, user_liked: true }), unlikeFn });

        // when
        await user.click(screen.getByRole("button", { name: `${FULL_HEART} 1` }));

        // then
        expect(screen.getByRole("button", { name: EMPTY_HEART })).toBeInTheDocument();
    });

    it("falls back to the like mutation when no handler is injected", async () => {
        // given
        likeComment.mockResolvedValue(undefined);
        const user = userEvent.setup();
        setup({ comment: makeComment({ like_count: 0 }) });

        // when
        await user.click(screen.getByRole("button", { name: EMPTY_HEART }));

        // then
        expect(likeComment).toHaveBeenCalledWith("comment-1");
    });

    it("takes every interaction away from a blocked viewer", () => {
        // given
        const viewerBlocked = true;

        // when
        setup({ comment: makeComment({ like_count: 3 }), viewerBlocked });

        // then
        expect(screen.queryByRole("button", { name: `${EMPTY_HEART} 3` })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Reply" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Copy Link" })).toBeInTheDocument();
    });

    it("opens a reply box under the comment", async () => {
        // given
        const user = userEvent.setup();
        setup();

        // when
        await user.click(screen.getByRole("button", { name: "Reply" }));

        // then
        expect(screen.getByPlaceholderText("Write a reply...")).toBeInTheDocument();
    });

    it("closes the reply box and refreshes once a reply is posted", async () => {
        // given
        const onDelete = vi.fn();
        const createCommentFn = vi.fn(() => Promise.resolve({ id: "comment-2" }));
        const user = userEvent.setup();
        setup({ onDelete, createCommentFn });
        await user.click(screen.getByRole("button", { name: "Reply" }));

        // when
        await user.type(screen.getByPlaceholderText("Write a reply..."), "I reject that theory");
        await user.click(screen.getAllByRole("button", { name: "Reply" })[1]);

        // then
        expect(createCommentFn).toHaveBeenCalledWith("post-1", "I reject that theory", "comment-1");
        await waitFor(() => expect(screen.queryByPlaceholderText("Write a reply...")).not.toBeInTheDocument());
        expect(onDelete).toHaveBeenCalled();
    });

    it("hides editing and deleting from a viewer who owns neither the comment nor the site", () => {
        // given
        const viewer = makeUser({ id: OTHER.id });

        // when
        setup({ viewer });

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("lets the author edit and delete their own comment", () => {
        // given
        const viewer = ownerViewer();

        // when
        setup({ viewer });

        // then
        expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("lets a moderator edit and delete somebody else's comment", () => {
        // given
        const viewer = makeUser({ id: OTHER.id, role: "moderator" });

        // when
        setup({ viewer });

        // then
        expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("asks for confirmation before deleting and does nothing when it is refused", async () => {
        // given
        const deleteFn = vi.fn(() => Promise.resolve());
        const onDelete = vi.fn();
        vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        setup({ viewer: ownerViewer(), deleteFn, onDelete });

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(deleteFn).not.toHaveBeenCalled();
        expect(onDelete).not.toHaveBeenCalled();
    });

    it("deletes the comment and refreshes once the viewer confirms", async () => {
        // given
        const deleteFn = vi.fn(() => Promise.resolve());
        const onDelete = vi.fn();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        setup({ viewer: ownerViewer(), deleteFn, onDelete });

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(deleteFn).toHaveBeenCalledWith("comment-1");
        await waitFor(() => expect(onDelete).toHaveBeenCalledOnce());
    });

    it("saves an edit with the surrounding whitespace stripped", async () => {
        // given
        const updateFn = vi.fn(() => Promise.resolve());
        const onDelete = vi.fn();
        const user = userEvent.setup();
        setup({ viewer: ownerViewer(), updateFn, onDelete });

        // when
        await user.click(screen.getByRole("button", { name: "Edit" }));
        await user.clear(screen.getByRole("textbox"));
        await user.type(screen.getByRole("textbox"), "  the truth is gold  ");
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(updateFn).toHaveBeenCalledWith("comment-1", "the truth is gold");
        await waitFor(() => expect(screen.queryByRole("button", { name: "Save" })).not.toBeInTheDocument());
        expect(onDelete).toHaveBeenCalledOnce();
    });

    it("starts the edit box from the existing body", async () => {
        // given
        const user = userEvent.setup();
        setup({ viewer: ownerViewer() });

        // when
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // then
        expect(screen.getByRole("textbox")).toHaveValue("Without love it cannot be seen");
    });

    it("refuses to save an edit that empties the comment", async () => {
        // given
        const updateFn = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        setup({ viewer: ownerViewer(), updateFn });

        // when
        await user.click(screen.getByRole("button", { name: "Edit" }));
        await user.clear(screen.getByRole("textbox"));

        // then
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
        expect(updateFn).not.toHaveBeenCalled();
    });

    it("keeps the comment as it was when the edit is cancelled", async () => {
        // given
        const updateFn = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        setup({ viewer: ownerViewer(), updateFn });
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // when
        await user.type(screen.getByRole("textbox"), " and rewritten");
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(updateFn).not.toHaveBeenCalled();
        expect(screen.getByText("Without love it cannot be seen")).toBeInTheDocument();
    });

    it("keeps the edit box open when saving fails", async () => {
        // given
        const updateFn = vi.fn(() => Promise.reject(new Error("the golden truth denies it")));
        const onDelete = vi.fn();
        const user = userEvent.setup();
        setup({ viewer: ownerViewer(), updateFn, onDelete });

        // when
        await user.click(screen.getByRole("button", { name: "Edit" }));
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        await waitFor(() => expect(screen.getByRole("button", { name: "Save" })).toBeEnabled());
        expect(onDelete).not.toHaveBeenCalled();
    });

    it("copies a permanent link that points at this comment", async () => {
        // given
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        const user = userEvent.setup();
        setup({ linkPrefix: "/forum" });

        // when
        await user.click(screen.getByRole("button", { name: "Copy Link" }));

        // then
        expect(writeText).toHaveBeenCalledWith("https://whentheycry.social/forum/post-1#comment-comment-1");
    });

    it("defaults the permanent link to the game board", async () => {
        // given
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        const user = userEvent.setup();
        setup();

        // when
        await user.click(screen.getByRole("button", { name: "Copy Link" }));

        // then
        expect(writeText).toHaveBeenCalledWith("https://whentheycry.social/game-board/post-1#comment-comment-1");
    });

    it("offers reporting to everybody except the author", () => {
        // given
        const viewer = makeUser({ id: OTHER.id });

        // when
        setup({ viewer });

        // then
        expect(screen.getByRole("button", { name: "Report" })).toBeInTheDocument();
    });

    it("does not offer the author a way to report themselves", () => {
        // given
        const viewer = ownerViewer();

        // when
        setup({ viewer });

        // then
        expect(screen.queryByRole("button", { name: "Report" })).not.toBeInTheDocument();
    });

    it("flattens a nested thread and names who each reply answers", () => {
        // given
        const comment = makeComment({
            replies: [
                makeComment({
                    id: "comment-2",
                    author: OTHER,
                    body: "I deny that",
                    replies: [makeComment({ id: "comment-3", author: AUTHOR, body: "Useless" })],
                }),
            ],
        });

        // when
        setup({ comment });

        // then
        expect(screen.getByText("I deny that")).toBeInTheDocument();
        expect(screen.getByText("Useless")).toBeInTheDocument();
        expect(screen.getByText("@Beatrice")).toBeInTheDocument();
        expect(screen.getByText("@Battler")).toBeInTheDocument();
    });

    it("collapses and reopens the reply thread", async () => {
        // given
        const user = userEvent.setup();
        setup({
            comment: makeComment({
                replies: [makeComment({ id: "comment-2", author: OTHER, body: "I deny that" })],
            }),
        });

        // when
        await user.click(screen.getByRole("button", { name: "Hide 1 reply" }));

        // then
        expect(screen.queryByText("I deny that")).not.toBeInTheDocument();
        await user.click(screen.getByRole("button", { name: "Show 1 reply" }));
        expect(screen.getByText("I deny that")).toBeInTheDocument();
    });

    it("counts every nested reply in the collapse control", () => {
        // given
        const comment = makeComment({
            replies: [
                makeComment({
                    id: "comment-2",
                    author: OTHER,
                    body: "I deny that",
                    replies: [makeComment({ id: "comment-3", author: AUTHOR, body: "Useless" })],
                }),
            ],
        });

        // when
        setup({ comment });

        // then
        expect(screen.getByRole("button", { name: "Hide 2 replies" })).toBeInTheDocument();
    });

    it("offers no collapse control when there are no replies", () => {
        // given
        const comment = makeComment({ replies: [] });

        // when
        setup({ comment });

        // then
        expect(screen.queryByRole("button", { name: /^(Hide|Show) / })).not.toBeInTheDocument();
    });
});
