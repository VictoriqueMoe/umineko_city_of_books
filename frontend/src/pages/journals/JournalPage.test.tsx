import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, renderWithProviders } from "../../test-utils/render";
import type { JournalDetail, JournalEntry, JournalEntrySummary, PostComment, UserProfile } from "../../types/api";
import { JournalPage } from "./JournalPage";

const {
    useJournal,
    useFollowJournal,
    useUnfollowJournal,
    useDeleteJournal,
    useSetJournalPaused,
    useCreateJournalComment,
    useUpdateJournalComment,
    useDeleteJournalComment,
    useLikeJournalComment,
    useUnlikeJournalComment,
    useUploadJournalCommentMedia,
    navigate,
} = vi.hoisted(() => ({
    useJournal: vi.fn(),
    useFollowJournal: vi.fn(),
    useUnfollowJournal: vi.fn(),
    useDeleteJournal: vi.fn(),
    useSetJournalPaused: vi.fn(),
    useCreateJournalComment: vi.fn(),
    useUpdateJournalComment: vi.fn(),
    useDeleteJournalComment: vi.fn(),
    useLikeJournalComment: vi.fn(),
    useUnlikeJournalComment: vi.fn(),
    useUploadJournalCommentMedia: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/journal", () => ({ useJournal }));
vi.mock("../../api/mutations/journal", () => ({
    useCreateJournalComment,
    useDeleteJournal,
    useDeleteJournalComment,
    useFollowJournal,
    useLikeJournalComment,
    useSetJournalPaused,
    useUnfollowJournal,
    useUnlikeJournalComment,
    useUpdateJournalComment,
    useUploadJournalCommentMedia,
}));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

interface CommentsStubProps {
    comments: PostComment[];
    targetId: string;
    user: UserProfile | null;
    title?: string;
    emptyText?: string | null;
    highlightedId?: string;
}

vi.mock("../../components/post/CommentsSection/CommentsSection", () => ({
    CommentsSection: (props: CommentsStubProps) => (
        <section aria-label="comments">
            <p>{`${props.title} on ${props.targetId} as ${props.user?.display_name ?? "nobody"}`}</p>
            <p>{`${props.comments.length} comments`}</p>
            <p>{`empty text: ${props.emptyText ?? "none"}`}</p>
            <p>{`highlighting ${props.highlightedId ?? "nothing"}`}</p>
        </section>
    ),
}));

const author = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });
const stranger = makeUser({ id: "stranger-1", username: "battler", display_name: "Battler" });
const moderator = makeUser({ id: "mod-1", username: "ronove", display_name: "Ronove", role: "moderator" });

function makeEntrySummary(overrides: Partial<JournalEntrySummary> = {}): JournalEntrySummary {
    return {
        id: "entry-1",
        entry_number: 1,
        title: null,
        word_count: 420,
        is_draft: false,
        created_at: "2026-02-01T10:00:00Z",
        ...overrides,
    };
}

function makeEntry(overrides: Partial<JournalEntry> = {}): JournalEntry {
    return {
        id: "entry-1",
        journal_id: "journal-1",
        entry_number: 1,
        title: null,
        body: "The witch smiled at the closed room.",
        word_count: 420,
        is_draft: false,
        has_prev: false,
        has_next: false,
        created_at: "2026-02-01T10:00:00Z",
        media: [],
        ...overrides,
    };
}

function makeJournal(overrides: Partial<JournalDetail> = {}): JournalDetail {
    return {
        id: "journal-1",
        title: "Rokkenjima Notes",
        work: "umineko",
        author,
        follower_count: 3,
        is_following: false,
        is_archived: false,
        is_paused: false,
        comment_count: 0,
        entry_count: 1,
        latest_entry_excerpt: "",
        created_at: "2026-01-01T00:00:00Z",
        last_author_activity_at: "2026-02-01T11:00:00Z",
        entries: [makeEntrySummary()],
        latest_entry: makeEntry(),
        comments: [],
        ...overrides,
    };
}

interface StubOptions {
    journal?: JournalDetail | null;
    loading?: boolean;
    follow?: () => Promise<unknown>;
    unfollow?: () => Promise<unknown>;
    remove?: () => Promise<unknown>;
    setPaused?: () => Promise<unknown>;
}

function stubJournal(options: StubOptions = {}) {
    const refresh = vi.fn(() => Promise.resolve());
    useJournal.mockReturnValue({
        journal: options.journal === undefined ? makeJournal() : options.journal,
        loading: options.loading ?? false,
        refresh,
    });

    const followAsync = vi.fn(options.follow ?? (() => Promise.resolve({})));
    const unfollowAsync = vi.fn(options.unfollow ?? (() => Promise.resolve({})));
    const deleteAsync = vi.fn(options.remove ?? (() => Promise.resolve({})));
    const setPausedAsync = vi.fn(options.setPaused ?? (() => Promise.resolve({})));
    useFollowJournal.mockReturnValue({ mutateAsync: followAsync });
    useUnfollowJournal.mockReturnValue({ mutateAsync: unfollowAsync });
    useDeleteJournal.mockReturnValue({ mutateAsync: deleteAsync });
    useSetJournalPaused.mockReturnValue({ mutateAsync: setPausedAsync });
    for (const hook of [
        useCreateJournalComment,
        useUpdateJournalComment,
        useDeleteJournalComment,
        useLikeJournalComment,
        useUnlikeJournalComment,
        useUploadJournalCommentMedia,
    ]) {
        hook.mockReturnValue({ mutateAsync: vi.fn(() => Promise.resolve({ id: "comment-1" })) });
    }

    return { refresh, followAsync, unfollowAsync, deleteAsync, setPausedAsync };
}

function renderPage(user: UserProfile | null, route = "/journals/journal-1") {
    return renderWithProviders(<JournalPage />, { user, route, path: "/journals/:id" });
}

describe("JournalPage", () => {
    it("waits while the journal is loading", () => {
        // given
        stubJournal({ loading: true });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Loading journal...")).toBeInTheDocument();
        expect(screen.queryByText("Rokkenjima Notes")).not.toBeInTheDocument();
    });

    it("says so when there is no such journal", () => {
        // given
        stubJournal({ journal: null });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Journal not found.")).toBeInTheDocument();
    });

    it("asks for the journal named in the route", () => {
        // given
        stubJournal();

        // when
        renderPage(null, "/journals/journal-77");

        // then
        expect(useJournal).toHaveBeenCalledWith("journal-77");
    });

    it("heads the page with the journal title and the work it covers", () => {
        // given
        stubJournal({ journal: makeJournal({ title: "Rokkenjima Notes", work: "higurashi", follower_count: 1 }) });

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("heading", { name: "Rokkenjima Notes" })).toBeInTheDocument();
        expect(screen.getByText("Higurashi")).toBeInTheDocument();
        expect(screen.getByText("★ 1 follower")).toBeInTheDocument();
    });

    it("pluralises the follower count", () => {
        // given
        stubJournal({ journal: makeJournal({ follower_count: 4 }) });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("★ 4 followers")).toBeInTheDocument();
    });

    it("marks an edited journal", () => {
        // given
        stubJournal({ journal: makeJournal({ updated_at: "2026-02-02T00:00:00Z" }) });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("(edited)")).toBeInTheDocument();
    });

    it("hides the follow button from a signed out visitor", () => {
        // given
        stubJournal();

        // when
        renderPage(null);

        // then
        expect(screen.queryByRole("button", { name: "Follow" })).not.toBeInTheDocument();
    });

    it("hides the follow button from the journal's own author", () => {
        // given
        stubJournal();

        // when
        renderPage(author);

        // then
        expect(screen.queryByRole("button", { name: "Follow" })).not.toBeInTheDocument();
    });

    it("follows the journal for a signed in reader", async () => {
        // given
        const { followAsync } = stubJournal();
        const user = userEvent.setup();
        renderPage(stranger);

        // when
        await user.click(screen.getByRole("button", { name: "Follow" }));

        // then
        expect(followAsync).toHaveBeenCalledWith("journal-1");
    });

    it("unfollows a journal the reader already follows", async () => {
        // given
        const { unfollowAsync } = stubJournal({ journal: makeJournal({ is_following: true }) });
        const user = userEvent.setup();
        renderPage(stranger);

        // when
        await user.click(screen.getByRole("button", { name: "Following" }));

        // then
        expect(unfollowAsync).toHaveBeenCalledWith("journal-1");
    });

    it("puts the follow state back when following fails", async () => {
        // given
        stubJournal({ follow: () => Promise.reject(new Error("nope")) });
        const queryClient = createTestQueryClient();
        const setQueryData = vi.spyOn(queryClient, "setQueryData");
        const user = userEvent.setup();
        renderWithProviders(<JournalPage />, {
            user: stranger,
            route: "/journals/journal-1",
            path: "/journals/:id",
            queryClient,
        });

        // when
        await user.click(screen.getByRole("button", { name: "Follow" }));

        // then
        await waitFor(() => {
            expect(setQueryData).toHaveBeenCalledTimes(2);
        });
        const optimistic = setQueryData.mock.calls[0][1] as (prev: JournalDetail) => JournalDetail;
        const rollback = setQueryData.mock.calls[1][1] as (prev: JournalDetail) => JournalDetail;
        expect(optimistic(makeJournal()).is_following).toBe(true);
        expect(rollback(makeJournal()).is_following).toBe(false);
    });

    it("keeps the edit and delete controls away from an unrelated reader", () => {
        // given
        stubJournal();

        // when
        renderPage(stranger);

        // then
        expect(screen.queryByRole("link", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("lets the author edit and delete their own journal", () => {
        // given
        stubJournal();

        // when
        renderPage(author);

        // then
        expect(screen.getByRole("link", { name: "Edit" })).toHaveAttribute("href", "/journals/journal-1/edit");
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("lets a moderator edit and delete somebody else's journal", () => {
        // given
        stubJournal();

        // when
        renderPage(moderator);

        // then
        expect(screen.getByRole("link", { name: "Edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("asks before deleting the journal", async () => {
        // given
        const { deleteAsync } = stubJournal();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this journal? This cannot be undone.");
        expect(deleteAsync).not.toHaveBeenCalled();
    });

    it("deletes the journal and goes back to the feed once confirmed", async () => {
        // given
        const { deleteAsync } = stubJournal();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(deleteAsync).toHaveBeenCalledWith("journal-1");
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/journals");
        });
    });

    it("stays put when the delete request fails", async () => {
        // given
        stubJournal({ remove: () => Promise.reject(new Error("nope")) });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(navigate).not.toHaveBeenCalledWith("/journals");
    });

    it("offers a report button to everyone but the author", () => {
        // given
        stubJournal();

        // when
        renderPage(stranger);

        // then
        expect(screen.getByRole("button", { name: /Report/ })).toBeInTheDocument();
    });

    it("warns that an archived journal is closed to new comments", () => {
        // given
        stubJournal({ journal: makeJournal({ is_archived: true }) });

        // when
        renderPage(stranger);

        // then
        expect(screen.getByText("Archived")).toBeInTheDocument();
        expect(
            screen.getByText("This journal was archived after 7 days of inactivity. New comments are disabled."),
        ).toBeInTheDocument();
        expect(screen.getByText("Journal discussion on journal-1 as nobody")).toBeInTheDocument();
    });

    it("lets a signed in reader comment on a live journal", () => {
        // given
        stubJournal();

        // when
        renderPage(stranger);

        // then
        expect(screen.getByText("Journal discussion on journal-1 as Battler")).toBeInTheDocument();
    });

    it("spotlights the latest entry with a link into it", () => {
        // given
        stubJournal({
            journal: makeJournal({ latest_entry: makeEntry({ entry_number: 3, title: "The Golden Truth" }) }),
        });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Latest update")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Entry 3: The Golden Truth" })).toHaveAttribute(
            "href",
            "/journals/journal-1/entry/3",
        );
        expect(screen.getByText("The witch smiled at the closed room.")).toBeInTheDocument();
        expect(screen.getByText("420 words")).toBeInTheDocument();
    });

    it("embeds a gif entry instead of printing its url", () => {
        // given
        stubJournal({
            journal: makeJournal({
                latest_entry: makeEntry({ body: "https://media1.giphy.com/media/abc123/beato.gif" }),
            }),
        });

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("img", { name: "GIF" })).toHaveAttribute(
            "src",
            "https://media1.giphy.com/media/abc123/beato.gif",
        );
    });

    it("nudges the author to write the first entry", () => {
        // given
        stubJournal({ journal: makeJournal({ latest_entry: null, entries: [] }) });

        // when
        renderPage(author);

        // then
        expect(screen.getByText(/Add the first one below\./)).toBeInTheDocument();
    });

    it("tells a visitor to check back when nothing has been written", () => {
        // given
        stubJournal({ journal: makeJournal({ latest_entry: null, entries: [] }) });

        // when
        renderPage(stranger);

        // then
        expect(screen.getByText(/Check back soon\./)).toBeInTheDocument();
        expect(screen.getByText("No entries to list yet.")).toBeInTheDocument();
    });

    it("lists every entry with its number, word count and draft state", () => {
        // given
        stubJournal({
            journal: makeJournal({
                entries: [
                    makeEntrySummary({ id: "entry-1", entry_number: 1, title: "Arrival" }),
                    makeEntrySummary({ id: "entry-2", entry_number: 2, is_draft: true, word_count: 90 }),
                ],
            }),
        });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("All entries (2)")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Entry 1: Arrival/ })).toHaveAttribute(
            "href",
            "/journals/journal-1/entry/1",
        );
        expect(screen.getByText("Draft")).toBeInTheDocument();
        expect(screen.getByText(/90 words/)).toBeInTheDocument();
    });

    it("offers the author a link to add a new entry", () => {
        // given
        stubJournal();

        // when
        renderPage(author);

        // then
        expect(screen.getByRole("link", { name: /New Entry/ })).toHaveAttribute(
            "href",
            "/journals/journal-1/entry/new",
        );
    });

    it("keeps the new entry link on an archived journal, because posting is how the author reopens it", () => {
        // given
        stubJournal({ journal: makeJournal({ is_archived: true }) });

        // when
        renderPage(author);

        // then
        expect(screen.getByRole("link", { name: /New Entry/ })).toHaveAttribute(
            "href",
            "/journals/journal-1/entry/new",
        );
    });

    it("keeps the new entry link away from a reader on an archived journal", () => {
        // given
        stubJournal({ journal: makeJournal({ is_archived: true }) });

        // when
        renderPage(null);

        // then
        expect(screen.queryByRole("link", { name: /New Entry/ })).not.toBeInTheDocument();
    });

    it("offers the author a pause control and hides it from a reader", () => {
        // given
        stubJournal();

        // when
        renderPage(author);

        // then
        expect(screen.getByRole("button", { name: "Pause" })).toBeInTheDocument();
    });

    it("offers to resume an already paused journal", () => {
        // given
        stubJournal({ journal: makeJournal({ is_paused: true }) });

        // when
        renderPage(author);

        // then
        expect(screen.getByRole("button", { name: "Resume" })).toBeInTheDocument();
    });

    it("takes the pause control away once the journal is archived, since pausing cannot undo that", () => {
        // given
        stubJournal({ journal: makeJournal({ is_archived: true }) });

        // when
        renderPage(author);

        // then
        expect(screen.queryByRole("button", { name: /Pause|Resume/ })).not.toBeInTheDocument();
    });

    it("returns to the feed from the back link", async () => {
        // given
        stubJournal();
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.click(screen.getByText("← All Journals"));

        // then
        expect(navigate).toHaveBeenCalledWith("/journals");
    });

    it("passes the comment named in the url fragment down to the discussion", () => {
        // given
        stubJournal();

        // when
        renderPage(null, "/journals/journal-1#comment-abc");

        // then
        expect(screen.getByText("highlighting abc")).toBeInTheDocument();
    });

    it("highlights nothing when the url fragment is not a comment", () => {
        // given
        stubJournal();

        // when
        renderPage(null, "/journals/journal-1#entries");

        // then
        expect(screen.getByText("highlighting nothing")).toBeInTheDocument();
    });
});
