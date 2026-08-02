import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { Announcement, AnnouncementComment, PostComment, UserProfile } from "../../types/api";
import { AnnouncementDetailPage } from "./AnnouncementDetailPage";

const {
    useAnnouncement,
    useCreateAnnouncementComment,
    useUpdateAnnouncementComment,
    useDeleteAnnouncementComment,
    useLikeAnnouncementComment,
    useUnlikeAnnouncementComment,
    useUploadAnnouncementCommentMedia,
    navigate,
} = vi.hoisted(() => ({
    useAnnouncement: vi.fn(),
    useCreateAnnouncementComment: vi.fn(),
    useUpdateAnnouncementComment: vi.fn(),
    useDeleteAnnouncementComment: vi.fn(),
    useLikeAnnouncementComment: vi.fn(),
    useUnlikeAnnouncementComment: vi.fn(),
    useUploadAnnouncementCommentMedia: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/announcement", () => ({ useAnnouncement }));
vi.mock("../../api/mutations/announcement", () => ({
    useCreateAnnouncementComment,
    useUpdateAnnouncementComment,
    useDeleteAnnouncementComment,
    useLikeAnnouncementComment,
    useUnlikeAnnouncementComment,
    useUploadAnnouncementCommentMedia,
}));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});
vi.mock("../../components/post/CommentsSection/CommentsSection", () => ({
    CommentsSection: (props: {
        comments: PostComment[];
        targetId: string;
        highlightedId?: string;
        linkPrefix?: string;
        reportType?: string;
        onChanged: () => void;
        likeFn?: (id: string) => Promise<unknown>;
        unlikeFn?: (id: string) => Promise<unknown>;
        deleteFn?: (id: string) => Promise<unknown>;
        updateFn?: (id: string, body: string) => Promise<unknown>;
        createCommentFn?: (targetId: string, body: string, parentId?: string) => Promise<unknown>;
        uploadMediaFn?: (commentId: string, file: File) => Promise<unknown>;
    }) => (
        <section
            data-testid="comments-section"
            data-target={props.targetId}
            data-highlighted={props.highlightedId ?? ""}
            data-prefix={props.linkPrefix ?? ""}
            data-report={props.reportType ?? ""}
        >
            <span>{`${props.comments.length} comments`}</span>
            <button onClick={props.onChanged}>stub changed</button>
            <button onClick={() => props.likeFn?.("comment-9")}>stub like</button>
            <button onClick={() => props.unlikeFn?.("comment-9")}>stub unlike</button>
            <button onClick={() => props.deleteFn?.("comment-9")}>stub delete</button>
            <button onClick={() => props.updateFn?.("comment-9", "edited body")}>stub update</button>
            <button onClick={() => props.createCommentFn?.("ignored", "new body", "parent-1")}>stub create</button>
            <button onClick={() => props.uploadMediaFn?.("comment-9", new File(["x"], "note.png"))}>stub upload</button>
        </section>
    ),
}));

const author = { id: "author-1", username: "beatrice", display_name: "Beatrice" };
const reader = makeUser({ id: "reader-1", username: "battler", display_name: "Battler" });

function makeComment(overrides: Partial<AnnouncementComment> = {}): AnnouncementComment {
    return {
        id: "comment-1",
        author,
        body: "Understood.",
        media: [],
        like_count: 0,
        user_liked: false,
        created_at: "2026-07-01T11:00:00Z",
        ...overrides,
    };
}

function makeAnnouncement(overrides: Partial<Announcement> = {}): Announcement {
    return {
        id: "announcement-1",
        title: "The board reopens",
        body: "The game board is open again.",
        author,
        pinned: false,
        created_at: "2026-07-01T10:00:00Z",
        updated_at: "2026-07-01T10:00:00Z",
        comments: [],
        ...overrides,
    };
}

interface StubOptions {
    announcement?: Announcement | null;
    loading?: boolean;
}

function stubAnnouncement(options: StubOptions = {}) {
    const refresh = vi.fn();
    useAnnouncement.mockReturnValue({
        announcement: options.announcement === undefined ? makeAnnouncement() : options.announcement,
        loading: options.loading ?? false,
        refresh,
    });

    const createAsync = vi.fn(() => Promise.resolve({ id: "comment-new" }));
    const updateAsync = vi.fn(() => Promise.resolve({ id: "comment-9" }));
    const deleteAsync = vi.fn(() => Promise.resolve(undefined));
    const likeAsync = vi.fn(() => Promise.resolve(undefined));
    const unlikeAsync = vi.fn(() => Promise.resolve(undefined));
    const uploadAsync = vi.fn(() => Promise.resolve(undefined));

    useCreateAnnouncementComment.mockReturnValue({ mutateAsync: createAsync });
    useUpdateAnnouncementComment.mockReturnValue({ mutateAsync: updateAsync });
    useDeleteAnnouncementComment.mockReturnValue({ mutateAsync: deleteAsync });
    useLikeAnnouncementComment.mockReturnValue({ mutateAsync: likeAsync });
    useUnlikeAnnouncementComment.mockReturnValue({ mutateAsync: unlikeAsync });
    useUploadAnnouncementCommentMedia.mockReturnValue({ mutateAsync: uploadAsync });

    return { refresh, createAsync, updateAsync, deleteAsync, likeAsync, unlikeAsync, uploadAsync };
}

function renderPage(user: UserProfile | null = null, route = "/announcements/announcement-1") {
    return renderWithProviders(<AnnouncementDetailPage />, { user, route, path: "/announcements/:id" });
}

describe("AnnouncementDetailPage", () => {
    it("waits while the announcement is loading", () => {
        // given
        stubAnnouncement({ loading: true, announcement: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading announcement...")).toBeInTheDocument();
    });

    it("says the announcement is missing when the server has none", () => {
        // given
        stubAnnouncement({ announcement: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Announcement not found.")).toBeInTheDocument();
    });

    it("asks for the announcement named in the address", () => {
        // given
        stubAnnouncement();

        // when
        renderPage(null, "/announcements/announcement-42");

        // then
        expect(useAnnouncement).toHaveBeenLastCalledWith("announcement-42");
    });

    it("shows the title and credits the author", () => {
        // given
        stubAnnouncement();

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "The board reopens" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Beatrice/ })).toHaveAttribute("href", "/user/beatrice");
    });

    it("renders the body as markdown", () => {
        // given
        stubAnnouncement({ announcement: makeAnnouncement({ body: "The **golden** witch returns." }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("golden").tagName).toBe("STRONG");
    });

    it("scrubs dangerous markup out of the body", () => {
        // given
        stubAnnouncement({
            announcement: makeAnnouncement({ body: "Careful.<script>window.stolen = 1;</script>" }),
        });

        // when
        const { container } = renderPage();

        // then
        expect(container.querySelector("script")).toBeNull();
        expect(screen.getByText(/Careful\./)).toBeInTheDocument();
    });

    it("stays quiet about edits when nothing has been changed", () => {
        // given
        stubAnnouncement({
            announcement: makeAnnouncement({
                created_at: "2026-07-01T10:00:00Z",
                updated_at: "2026-07-01T10:00:00Z",
            }),
        });

        // when
        renderPage();

        // then
        expect(screen.queryByText("(edited)")).not.toBeInTheDocument();
    });

    it("admits when the announcement has been edited", () => {
        // given
        stubAnnouncement({
            announcement: makeAnnouncement({
                created_at: "2026-07-01T10:00:00Z",
                updated_at: "2026-07-03T10:00:00Z",
            }),
        });

        // when
        renderPage();

        // then
        expect(screen.getByText("(edited)")).toBeInTheDocument();
    });

    it("takes the reader back to the full list", async () => {
        // given
        stubAnnouncement();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText(/All Announcements/));

        // then
        expect(navigate).toHaveBeenCalledWith("/announcements");
    });

    it("hands the comments section the announcement's own comments and links", () => {
        // given
        stubAnnouncement({
            announcement: makeAnnouncement({ comments: [makeComment(), makeComment({ id: "comment-2" })] }),
        });

        // when
        renderPage(reader);

        // then
        const section = screen.getByTestId("comments-section");
        expect(section).toHaveAttribute("data-target", "announcement-1");
        expect(section).toHaveAttribute("data-prefix", "/announcements");
        expect(section).toHaveAttribute("data-report", "announcement_comment");
        expect(screen.getByText("2 comments")).toBeInTheDocument();
    });

    it("copes with an announcement whose comments the server left out", () => {
        // given
        stubAnnouncement({ announcement: makeAnnouncement({ comments: undefined }) });

        // when
        renderPage(reader);

        // then
        expect(screen.getByText("0 comments")).toBeInTheDocument();
    });

    it("highlights the comment a notification linked to", () => {
        // given
        stubAnnouncement();

        // when
        renderPage(reader, "/announcements/announcement-1#comment-abc");

        // then
        expect(screen.getByTestId("comments-section")).toHaveAttribute("data-highlighted", "abc");
    });

    it("highlights nothing when the address carries no comment anchor", () => {
        // given
        stubAnnouncement();

        // when
        renderPage(reader, "/announcements/announcement-1#top");

        // then
        expect(screen.getByTestId("comments-section")).toHaveAttribute("data-highlighted", "");
    });

    it("posts a new comment through the announcement's own mutation", async () => {
        // given
        const { createAsync } = stubAnnouncement();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "stub create" }));

        // then
        expect(createAsync).toHaveBeenCalledWith({ body: "new body", parentId: "parent-1" });
    });

    it("edits a comment through the announcement's own mutation", async () => {
        // given
        const { updateAsync } = stubAnnouncement();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "stub update" }));

        // then
        expect(updateAsync).toHaveBeenCalledWith({ id: "comment-9", body: "edited body" });
    });

    it("likes, unlikes and removes a comment through the announcement's own mutations", async () => {
        // given
        const { likeAsync, unlikeAsync, deleteAsync } = stubAnnouncement();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "stub like" }));
        await user.click(screen.getByRole("button", { name: "stub unlike" }));
        await user.click(screen.getByRole("button", { name: "stub delete" }));

        // then
        expect(likeAsync).toHaveBeenCalledWith("comment-9");
        expect(unlikeAsync).toHaveBeenCalledWith("comment-9");
        expect(deleteAsync).toHaveBeenCalledWith("comment-9");
    });

    it("uploads an attachment against the comment it belongs to", async () => {
        // given
        const { uploadAsync } = stubAnnouncement();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "stub upload" }));

        // then
        expect(uploadAsync).toHaveBeenCalledWith({ commentId: "comment-9", file: expect.any(File) });
    });

    it("refetches the announcement after a comment changes", async () => {
        // given
        const { refresh } = stubAnnouncement();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "stub changed" }));

        // then
        expect(refresh).toHaveBeenCalledOnce();
    });
});
