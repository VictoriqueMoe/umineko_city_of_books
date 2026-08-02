import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { PostComment, User, UserProfile } from "../../../types/api";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import { CommentsSection } from "./CommentsSection";

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

const AUTHOR: User = { id: "user-1", username: "beatrice", display_name: "Beatrice" };

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
    comments?: PostComment[] | null;
    viewer?: UserProfile | null;
    onChanged?: () => void;
    title?: string;
    emptyText?: string | null;
    blockedText?: string;
    viewerBlocked?: boolean;
    showComposer?: boolean;
    composerPosition?: "top" | "bottom";
    createCommentFn?: (targetId: string, body: string, parentId?: string) => Promise<{ id: string }>;
}

function setup(options: SetupOptions = {}) {
    const viewer = options.viewer === undefined ? makeUser({ id: "user-2", username: "battler" }) : options.viewer;

    return renderWithProviders(
        <CommentsSection
            comments={options.comments === undefined ? [makeComment()] : options.comments}
            targetId="post-1"
            user={viewer}
            onChanged={options.onChanged ?? (() => {})}
            title={options.title}
            emptyText={options.emptyText}
            blockedText={options.blockedText}
            viewerBlocked={options.viewerBlocked}
            showComposer={options.showComposer}
            composerPosition={options.composerPosition}
            createCommentFn={options.createCommentFn}
        />,
        { user: viewer },
    );
}

describe("CommentsSection", () => {
    it("counts the comments beside its heading", () => {
        // given
        const comments = [makeComment(), makeComment({ id: "comment-2", body: "I deny that" })];

        // when
        setup({ comments });

        // then
        expect(screen.getByRole("heading", { name: "Comments (2)" })).toBeInTheDocument();
    });

    it("leaves the count off the heading when nothing has been said yet", () => {
        // given
        const comments: PostComment[] = [];

        // when
        setup({ comments });

        // then
        expect(screen.getByRole("heading", { name: "Comments" })).toBeInTheDocument();
    });

    it("takes a heading of its own when one is supplied", () => {
        // given
        const title = "Responses";

        // when
        setup({ title, comments: [makeComment()] });

        // then
        expect(screen.getByRole("heading", { name: "Responses (1)" })).toBeInTheDocument();
    });

    it("renders every comment it is given", () => {
        // given
        const comments = [makeComment(), makeComment({ id: "comment-2", body: "I deny that" })];

        // when
        setup({ comments });

        // then
        expect(screen.getByText("Without love it cannot be seen")).toBeInTheDocument();
        expect(screen.getByText("I deny that")).toBeInTheDocument();
    });

    it("explains the emptiness when there are no comments", () => {
        // given
        const comments: PostComment[] = [];

        // when
        setup({ comments });

        // then
        expect(screen.getByText("No comments yet.")).toBeInTheDocument();
    });

    it("uses the empty wording it is given", () => {
        // given
        const emptyText = "Nobody has theorised yet.";

        // when
        setup({ comments: [], emptyText });

        // then
        expect(screen.getByText("Nobody has theorised yet.")).toBeInTheDocument();
    });

    it("stays quiet about emptiness when the wording is suppressed", () => {
        // given
        const emptyText = null;

        // when
        setup({ comments: [], emptyText });

        // then
        expect(screen.queryByText("No comments yet.")).not.toBeInTheDocument();
    });

    it("treats a missing comment list as an empty one", () => {
        // given
        const comments = null;

        // when
        setup({ comments });

        // then
        expect(screen.getByRole("heading", { name: "Comments" })).toBeInTheDocument();
        expect(screen.getByText("No comments yet.")).toBeInTheDocument();
    });

    it("offers the composer to a signed in viewer", () => {
        // given
        const viewer = makeUser({ id: "user-2" });

        // when
        setup({ viewer });

        // then
        expect(screen.getByPlaceholderText("Write a comment...")).toBeInTheDocument();
    });

    it("keeps the composer away from signed out viewers", () => {
        // given
        const viewer = null;

        // when
        setup({ viewer });

        // then
        expect(screen.queryByPlaceholderText("Write a comment...")).not.toBeInTheDocument();
    });

    it("hides the composer when it has been switched off", () => {
        // given
        const showComposer = false;

        // when
        setup({ showComposer });

        // then
        expect(screen.queryByPlaceholderText("Write a comment...")).not.toBeInTheDocument();
    });

    it("explains the block instead of offering a composer to a blocked viewer", () => {
        // given
        const viewerBlocked = true;

        // when
        setup({ viewerBlocked });

        // then
        expect(screen.getByText("You cannot interact with this post.")).toBeInTheDocument();
        expect(screen.queryByPlaceholderText("Write a comment...")).not.toBeInTheDocument();
    });

    it("uses the blocked wording it is given", () => {
        // given
        const blockedText = "This detective has shut you out.";

        // when
        setup({ viewerBlocked: true, blockedText });

        // then
        expect(screen.getByText("This detective has shut you out.")).toBeInTheDocument();
    });

    it("puts the composer after the comments by default", () => {
        // given
        const comments = [makeComment()];

        // when
        setup({ comments });

        // then
        const body = screen.getByText("Without love it cannot be seen");
        const composer = screen.getByPlaceholderText("Write a comment...");
        expect(body.compareDocumentPosition(composer) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    });

    it("puts the composer before the comments when asked to", () => {
        // given
        const composerPosition = "top" as const;

        // when
        setup({ composerPosition, comments: [makeComment()] });

        // then
        const body = screen.getByText("Without love it cannot be seen");
        const composer = screen.getByPlaceholderText("Write a comment...");
        expect(composer.compareDocumentPosition(body) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    });

    it("posts a new comment against the target it was given", async () => {
        // given
        const onChanged = vi.fn();
        const createCommentFn = vi.fn(() => Promise.resolve({ id: "comment-9" }));
        const user = userEvent.setup();
        setup({ onChanged, createCommentFn });

        // when
        await user.type(screen.getByPlaceholderText("Write a comment..."), "the seventh twilight");
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        expect(createCommentFn).toHaveBeenCalledWith("post-1", "the seventh twilight", undefined);
        await waitFor(() => expect(onChanged).toHaveBeenCalledOnce());
    });

    it("hands the same create handler to a reply on an existing comment", async () => {
        // given
        const createCommentFn = vi.fn(() => Promise.resolve({ id: "comment-9" }));
        const user = userEvent.setup();
        setup({ createCommentFn });

        // when
        await user.click(screen.getByRole("button", { name: "Reply" }));
        await user.type(screen.getByPlaceholderText("Write a reply..."), "I reject that");
        await user.click(screen.getAllByRole("button", { name: "Reply" })[1]);

        // then
        expect(createCommentFn).toHaveBeenCalledWith("post-1", "I reject that", "comment-1");
    });
});
