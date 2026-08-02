import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { FanficChapterSummary, FanficDetail, PostComment, UserProfile } from "../../types/api";
import { FanficDetailPage } from "./FanficDetailPage";

const {
    useFanfic,
    useFavouriteFanfic,
    useUnfavouriteFanfic,
    useDeleteFanfic,
    useDeleteFanficChapter,
    useCreateFanficComment,
    useUpdateFanficComment,
    useDeleteFanficComment,
    useLikeFanficComment,
    useUnlikeFanficComment,
    useUploadFanficCommentMedia,
    navigate,
} = vi.hoisted(() => ({
    useFanfic: vi.fn(),
    useFavouriteFanfic: vi.fn(),
    useUnfavouriteFanfic: vi.fn(),
    useDeleteFanfic: vi.fn(),
    useDeleteFanficChapter: vi.fn(),
    useCreateFanficComment: vi.fn(),
    useUpdateFanficComment: vi.fn(),
    useDeleteFanficComment: vi.fn(),
    useLikeFanficComment: vi.fn(),
    useUnlikeFanficComment: vi.fn(),
    useUploadFanficCommentMedia: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/fanfic", () => ({ useFanfic }));
vi.mock("../../api/mutations/fanfic", () => ({
    useCreateFanficComment,
    useDeleteFanfic,
    useDeleteFanficChapter,
    useDeleteFanficComment,
    useFavouriteFanfic,
    useLikeFanficComment,
    useUnfavouriteFanfic,
    useUnlikeFanficComment,
    useUpdateFanficComment,
    useUploadFanficCommentMedia,
}));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

interface CommentsStubProps {
    comments: PostComment[];
    targetId: string;
    user: UserProfile | null;
    viewerBlocked?: boolean;
    highlightedId?: string;
}

vi.mock("../../components/post/CommentsSection/CommentsSection", () => ({
    CommentsSection: (props: CommentsStubProps) => (
        <section aria-label="comments">
            <p>{`${props.comments.length} comments on ${props.targetId} as ${props.user?.display_name ?? "nobody"}`}</p>
            <p>{`blocked: ${props.viewerBlocked ? "yes" : "no"}`}</p>
            <p>{`highlighting ${props.highlightedId ?? "nothing"}`}</p>
        </section>
    ),
}));

vi.mock("../../components/ShareButton/ShareButton", () => ({
    ShareButton: (props: { contentId: string; contentType: string }) => (
        <div>{`share ${props.contentType} ${props.contentId}`}</div>
    ),
}));

const author = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });
const stranger = makeUser({ id: "stranger-1", username: "battler", display_name: "Battler" });
const moderator = makeUser({ id: "mod-1", username: "ronove", display_name: "Ronove", role: "moderator" });

function makeChapter(overrides: Partial<FanficChapterSummary> = {}): FanficChapterSummary {
    return {
        id: "chapter-1",
        chapter_number: 1,
        title: "The Arrival",
        word_count: 1500,
        ...overrides,
    };
}

function makeFanfic(overrides: Partial<FanficDetail> = {}): FanficDetail {
    return {
        id: "fanfic-1",
        author,
        title: "Golden Land",
        summary: "",
        series: "Umineko",
        rating: "T",
        language: "English",
        status: "In Progress",
        is_oneshot: false,
        contains_lemons: false,
        genres: [],
        tags: [],
        characters: [],
        is_pairing: false,
        word_count: 2500,
        chapter_count: 2,
        favourite_count: 4,
        view_count: 90,
        comment_count: 0,
        user_favourited: false,
        published_at: "2026-01-01T00:00:00Z",
        created_at: "2026-01-01T00:00:00Z",
        chapters: [makeChapter(), makeChapter({ id: "chapter-2", chapter_number: 2, title: "The Witch" })],
        comments: [],
        reading_progress: 0,
        viewer_blocked: false,
        ...overrides,
    };
}

interface StubOptions {
    fanfic?: FanficDetail | null;
    loading?: boolean;
    favourite?: () => Promise<unknown>;
    unfavourite?: () => Promise<unknown>;
    removeChapter?: () => Promise<unknown>;
    removeFanfic?: () => Promise<unknown>;
}

function stubFanfic(options: StubOptions = {}) {
    const refresh = vi.fn(() => Promise.resolve());
    useFanfic.mockReturnValue({
        fanfic: options.fanfic === undefined ? makeFanfic() : options.fanfic,
        loading: options.loading ?? false,
        refresh,
    });

    const favouriteAsync = vi.fn(options.favourite ?? (() => Promise.resolve({})));
    const unfavouriteAsync = vi.fn(options.unfavourite ?? (() => Promise.resolve({})));
    const deleteFanficAsync = vi.fn(options.removeFanfic ?? (() => Promise.resolve({})));
    const deleteChapterAsync = vi.fn(options.removeChapter ?? (() => Promise.resolve({})));
    useFavouriteFanfic.mockReturnValue({ mutateAsync: favouriteAsync });
    useUnfavouriteFanfic.mockReturnValue({ mutateAsync: unfavouriteAsync });
    useDeleteFanfic.mockReturnValue({ mutateAsync: deleteFanficAsync });
    useDeleteFanficChapter.mockReturnValue({ mutateAsync: deleteChapterAsync });
    for (const hook of [
        useCreateFanficComment,
        useUpdateFanficComment,
        useDeleteFanficComment,
        useLikeFanficComment,
        useUnlikeFanficComment,
        useUploadFanficCommentMedia,
    ]) {
        hook.mockReturnValue({ mutateAsync: vi.fn(() => Promise.resolve({ id: "comment-1" })) });
    }

    return { refresh, favouriteAsync, unfavouriteAsync, deleteFanficAsync, deleteChapterAsync };
}

function renderPage(user: UserProfile | null, route = "/fanfiction/fanfic-1") {
    return renderWithProviders(<FanficDetailPage />, { user, route, path: "/fanfiction/:id" });
}

function storyButton(name: string): HTMLElement {
    return screen.getAllByRole("button", { name })[0];
}

describe("FanficDetailPage", () => {
    it("waits while the fanfic is loading", () => {
        // given
        stubFanfic({ loading: true });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("says so when there is no such fanfic", () => {
        // given
        stubFanfic({ fanfic: null });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Fanfic not found.")).toBeInTheDocument();
    });

    it("asks for the fanfic named in the route", () => {
        // given
        stubFanfic();

        // when
        renderPage(null, "/fanfiction/fanfic-77");

        // then
        expect(useFanfic).toHaveBeenCalledWith("fanfic-77");
    });

    it("heads the page with the story and its badges", () => {
        // given
        stubFanfic({
            fanfic: makeFanfic({
                title: "Golden Land",
                rating: "M",
                status: "Complete",
                genres: ["Mystery"],
                tags: ["closed room"],
                is_pairing: true,
                contains_lemons: true,
            }),
        });

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("heading", { name: "Golden Land" })).toBeInTheDocument();
        expect(screen.getByText("M")).toBeInTheDocument();
        expect(screen.getByText("Complete")).toBeInTheDocument();
        expect(screen.getByText("Mystery")).toBeInTheDocument();
        expect(screen.getByText("closed room")).toBeInTheDocument();
        expect(screen.getByText("Pairing")).toBeInTheDocument();
        expect(screen.getByText("Contains Lemons")).toBeInTheDocument();
    });

    it("abbreviates the larger stats", () => {
        // given
        stubFanfic({ fanfic: makeFanfic({ word_count: 1_500_000, view_count: 2400, favourite_count: 12 }) });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("1.5M")).toBeInTheDocument();
        expect(screen.getByText("2.4K")).toBeInTheDocument();
        expect(screen.getByText("12")).toBeInTheDocument();
    });

    it("uses singular wording for a story of one chapter", () => {
        // given
        stubFanfic({ fanfic: makeFanfic({ chapter_count: 1 }) });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Chapter")).toBeInTheDocument();
    });

    it("hides the favourite button from a signed out visitor", () => {
        // given
        stubFanfic();

        // when
        renderPage(null);

        // then
        expect(screen.queryByRole("button", { name: /4/ })).not.toBeInTheDocument();
    });

    it("favourites the story for a signed in reader", async () => {
        // given
        const { favouriteAsync, refresh } = stubFanfic();
        const user = userEvent.setup();
        renderPage(stranger);

        // when
        await user.click(screen.getByRole("button", { name: "♡ 4" }));

        // then
        expect(favouriteAsync).toHaveBeenCalledWith("fanfic-1");
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });

    it("unfavourites a story the reader already favourited", async () => {
        // given
        const { unfavouriteAsync } = stubFanfic({ fanfic: makeFanfic({ user_favourited: true }) });
        const user = userEvent.setup();
        renderPage(stranger);

        // when
        await user.click(screen.getByRole("button", { name: "♥ 4" }));

        // then
        expect(unfavouriteAsync).toHaveBeenCalledWith("fanfic-1");
    });

    it("does not refresh when favouriting fails", async () => {
        // given
        const { refresh } = stubFanfic({ favourite: () => Promise.reject(new Error("nope")) });
        const user = userEvent.setup();
        renderPage(stranger);

        // when
        await user.click(screen.getByRole("button", { name: "♡ 4" }));

        // then
        expect(refresh).not.toHaveBeenCalled();
        await waitFor(() => {
            expect(screen.getByRole("button", { name: "♡ 4" })).toBeEnabled();
        });
    });

    it("keeps the author controls away from an unrelated reader", () => {
        // given
        stubFanfic();

        // when
        renderPage(stranger);

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Add Chapter" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("lets the author edit their own story", async () => {
        // given
        stubFanfic();
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(storyButton("Edit"));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1/edit");
    });

    it("lets the author add another chapter", async () => {
        // given
        stubFanfic();
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getByRole("button", { name: "Add Chapter" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1/chapter/new");
    });

    it("offers no add chapter button on a one-shot", () => {
        // given
        stubFanfic({ fanfic: makeFanfic({ is_oneshot: true }) });

        // when
        renderPage(author);

        // then
        expect(screen.queryByRole("button", { name: "Add Chapter" })).not.toBeInTheDocument();
    });

    it("lets a moderator edit and delete somebody else's story", () => {
        // given
        stubFanfic();

        // when
        renderPage(moderator);

        // then
        expect(storyButton("Edit")).toBeInTheDocument();
        expect(storyButton("Delete")).toBeInTheDocument();
    });

    it("asks before deleting the story", async () => {
        // given
        const { deleteFanficAsync } = stubFanfic();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(storyButton("Delete"));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this fanfic? This cannot be undone.");
        expect(deleteFanficAsync).not.toHaveBeenCalled();
    });

    it("deletes the story and returns to the archive once confirmed", async () => {
        // given
        const { deleteFanficAsync } = stubFanfic();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(storyButton("Delete"));

        // then
        expect(deleteFanficAsync).toHaveBeenCalledWith("fanfic-1");
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/fanfiction");
        });
    });

    it("keeps the reader on the story when the delete fails", async () => {
        // given
        const { deleteFanficAsync } = stubFanfic({ removeFanfic: () => Promise.reject(new Error("nope")) });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(storyButton("Delete"));

        // then
        expect(deleteFanficAsync).toHaveBeenCalledWith("fanfic-1");
        await waitFor(() => {
            expect(screen.getByRole("heading", { name: "Golden Land" })).toBeInTheDocument();
        });
        expect(navigate).not.toHaveBeenCalledWith("/fanfiction");
    });

    it("opens a one-shot straight at its only chapter", async () => {
        // given
        stubFanfic({ fanfic: makeFanfic({ is_oneshot: true }) });
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.click(screen.getByRole("button", { name: "Read Story" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1/chapter/1");
        expect(screen.queryByText(/^Chapters \(/)).not.toBeInTheDocument();
    });

    it("lists the chapters of a multi-chapter story", () => {
        // given
        stubFanfic();

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Chapters (2)")).toBeInTheDocument();
        expect(screen.getByText("The Arrival")).toBeInTheDocument();
        expect(screen.getByText("The Witch")).toBeInTheDocument();
        expect(screen.getAllByText("1.5K words")).toHaveLength(2);
    });

    it("opens a chapter the reader clicks in the list", async () => {
        // given
        stubFanfic();
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.click(screen.getByText("The Witch"));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1/chapter/2");
    });

    it("offers to continue from where the reader left off", async () => {
        // given
        stubFanfic({ fanfic: makeFanfic({ reading_progress: 2 }) });
        const user = userEvent.setup();
        renderPage(stranger);

        // when
        await user.click(screen.getByRole("button", { name: "Continue from Chapter 2" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1/chapter/2");
    });

    it("does not offer to continue past the last chapter that exists", () => {
        // given
        stubFanfic({ fanfic: makeFanfic({ reading_progress: 9 }) });

        // when
        renderPage(stranger);

        // then
        expect(screen.queryByRole("button", { name: /Continue from Chapter/ })).not.toBeInTheDocument();
    });

    it("asks before deleting a chapter", async () => {
        // given
        const { deleteChapterAsync } = stubFanfic();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getAllByRole("button", { name: "Delete" })[1]);

        // then
        expect(confirm).toHaveBeenCalledWith("Delete chapter 1? This cannot be undone.");
        expect(deleteChapterAsync).not.toHaveBeenCalled();
    });

    it("deletes a chapter and refreshes the story once confirmed", async () => {
        // given
        const { deleteChapterAsync, refresh } = stubFanfic();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getAllByRole("button", { name: "Delete" })[1]);

        // then
        expect(deleteChapterAsync).toHaveBeenCalledWith("chapter-1");
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });

    it("lists the characters the story is about", () => {
        // given
        stubFanfic({
            fanfic: makeFanfic({
                characters: [
                    { series: "umineko", character_name: "Kanon", sort_order: 0 },
                    { series: "umineko", character_name: "Shannon", sort_order: 1 },
                ],
            }),
        });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Kanon")).toBeInTheDocument();
        expect(screen.getByText("Shannon")).toBeInTheDocument();
    });

    it("shows the summary only when the story has one", () => {
        // given
        stubFanfic({ fanfic: makeFanfic({ summary: "A closed room on Rokkenjima." }) });

        // when
        const { rerender } = renderPage(null);

        // then
        expect(screen.getByText("A closed room on Rokkenjima.")).toBeInTheDocument();
        stubFanfic({ fanfic: makeFanfic({ summary: "" }) });
        rerender(<FanficDetailPage />);
        expect(screen.queryByText("A closed room on Rokkenjima.")).not.toBeInTheDocument();
    });

    it("opens the cover in a lightbox", async () => {
        // given
        stubFanfic({ fanfic: makeFanfic({ cover_image_url: "https://cdn.test/cover.png" }) });
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.click(screen.getAllByRole("img", { name: "Golden Land" })[0]);

        // then
        expect(screen.getByRole("dialog")).toBeInTheDocument();
        expect(screen.getAllByRole("img", { name: "Golden Land" })).toHaveLength(2);
    });

    it("tells the discussion when the viewer is blocked", () => {
        // given
        stubFanfic({ fanfic: makeFanfic({ viewer_blocked: true }) });

        // when
        renderPage(stranger);

        // then
        expect(screen.getByText("blocked: yes")).toBeInTheDocument();
    });

    it("passes the comment named in the url fragment down to the discussion", () => {
        // given
        stubFanfic();

        // when
        renderPage(null, "/fanfiction/fanfic-1#comment-abc");

        // then
        expect(screen.getByText("highlighting abc")).toBeInTheDocument();
    });

    it("returns to the archive from the back link", async () => {
        // given
        stubFanfic();
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.click(screen.getByText("← All Fanfiction"));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction");
    });
});
