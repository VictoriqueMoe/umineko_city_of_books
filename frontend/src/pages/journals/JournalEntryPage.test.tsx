import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { JournalComment, JournalDetail, JournalEntry, PostComment, UserProfile } from "../../types/api";
import { JournalEntryPage } from "./JournalEntryPage";

const {
    useJournal,
    useJournalEntry,
    useCreateJournalComment,
    useUpdateJournalComment,
    useDeleteJournalComment,
    useLikeJournalComment,
    useUnlikeJournalComment,
    useUploadJournalCommentMedia,
    useDeleteJournalEntry,
    navigate,
} = vi.hoisted(() => ({
    useJournal: vi.fn(),
    useJournalEntry: vi.fn(),
    useCreateJournalComment: vi.fn(),
    useUpdateJournalComment: vi.fn(),
    useDeleteJournalComment: vi.fn(),
    useLikeJournalComment: vi.fn(),
    useUnlikeJournalComment: vi.fn(),
    useUploadJournalCommentMedia: vi.fn(),
    useDeleteJournalEntry: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/journal", () => ({ useJournal, useJournalEntry }));
vi.mock("../../api/mutations/journal", () => ({
    useCreateJournalComment,
    useDeleteJournalComment,
    useDeleteJournalEntry,
    useLikeJournalComment,
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
    linkPrefix?: string;
}

vi.mock("../../components/post/CommentsSection/CommentsSection", () => ({
    CommentsSection: (props: CommentsStubProps) => (
        <section aria-label="comments">
            <p>{`${props.title} on ${props.targetId} as ${props.user?.display_name ?? "nobody"}`}</p>
            <p>{`${props.comments.length} comments`}</p>
            <p>{`empty text: ${props.emptyText ?? "none"}`}</p>
            <p>{`highlighting ${props.highlightedId ?? "nothing"}`}</p>
            <p>{`links under ${props.linkPrefix}`}</p>
        </section>
    ),
}));

const author = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });
const stranger = makeUser({ id: "stranger-1", username: "battler", display_name: "Battler" });
const moderator = makeUser({ id: "mod-1", username: "ronove", display_name: "Ronove", role: "moderator" });

function makeEntry(overrides: Partial<JournalEntry> = {}): JournalEntry {
    return {
        id: "entry-9",
        journal_id: "journal-1",
        entry_number: 2,
        title: null,
        body: "The witch smiled at the closed room.",
        word_count: 420,
        is_draft: false,
        has_prev: true,
        has_next: true,
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
        comment_count: 0,
        entry_count: 3,
        latest_entry_excerpt: "",
        created_at: "2026-01-01T00:00:00Z",
        last_author_activity_at: "2026-02-01T11:00:00Z",
        entries: [],
        latest_entry: null,
        comments: [],
        ...overrides,
    };
}

interface StubOptions {
    journal?: JournalDetail | null;
    entry?: JournalEntry | null;
    comments?: JournalComment[];
    journalLoading?: boolean;
    entryLoading?: boolean;
    remove?: () => Promise<unknown>;
}

function stubEntry(options: StubOptions = {}) {
    const refresh = vi.fn(() => Promise.resolve());
    useJournal.mockReturnValue({
        journal: options.journal === undefined ? makeJournal() : options.journal,
        loading: options.journalLoading ?? false,
        refresh: vi.fn(),
    });
    useJournalEntry.mockReturnValue({
        entry: options.entry === undefined ? makeEntry() : options.entry,
        comments: options.comments ?? [],
        loading: options.entryLoading ?? false,
        refresh,
    });

    const deleteEntryAsync = vi.fn(options.remove ?? (() => Promise.resolve({})));
    useDeleteJournalEntry.mockReturnValue({ mutateAsync: deleteEntryAsync });
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

    return { refresh, deleteEntryAsync };
}

function renderPage(user: UserProfile | null, route = "/journals/journal-1/entry/2") {
    return renderWithProviders(<JournalEntryPage />, { user, route, path: "/journals/:id/entry/:number" });
}

describe("JournalEntryPage", () => {
    it("waits while the journal is loading", () => {
        // given
        stubEntry({ journalLoading: true });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Loading entry...")).toBeInTheDocument();
    });

    it("waits while the entry itself is loading", () => {
        // given
        stubEntry({ entryLoading: true });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Loading entry...")).toBeInTheDocument();
    });

    it("says so when the entry does not exist", () => {
        // given
        stubEntry({ entry: null });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Entry not found.")).toBeInTheDocument();
    });

    it("says so when the journal behind the entry does not exist", () => {
        // given
        stubEntry({ journal: null });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Entry not found.")).toBeInTheDocument();
    });

    it("asks for the entry number named in the route", () => {
        // given
        stubEntry();

        // when
        renderPage(null, "/journals/journal-5/entry/7");

        // then
        expect(useJournalEntry).toHaveBeenCalledWith("journal-5", 7);
    });

    it("heads the entry with its number when it has no title", () => {
        // given
        stubEntry({ entry: makeEntry({ entry_number: 2, title: null }) });

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("heading", { name: "Entry 2" })).toBeInTheDocument();
    });

    it("heads the entry with its title when it has one", () => {
        // given
        stubEntry({ entry: makeEntry({ entry_number: 2, title: "The Golden Truth" }) });

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("heading", { name: "Entry 2: The Golden Truth" })).toBeInTheDocument();
    });

    it("prints the body of the entry", () => {
        // given
        stubEntry();

        // when
        renderPage(null);

        // then
        expect(screen.getByText("The witch smiled at the closed room.")).toBeInTheDocument();
        expect(screen.getAllByText("420 words")).toHaveLength(2);
    });

    it("embeds a gif entry instead of printing its url", () => {
        // given
        stubEntry({ entry: makeEntry({ body: "https://media1.giphy.com/media/abc123/beato.gif" }) });

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("img", { name: "GIF" })).toHaveAttribute(
            "src",
            "https://media1.giphy.com/media/abc123/beato.gif",
        );
    });

    it("marks an entry that was edited after it was written", () => {
        // given
        stubEntry({
            entry: makeEntry({ created_at: "2026-02-01T10:00:00Z", updated_at: "2026-02-02T10:00:00Z" }),
        });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("(edited)")).toBeInTheDocument();
    });

    it("does not call an entry edited when it was only ever saved once", () => {
        // given
        stubEntry({
            entry: makeEntry({ created_at: "2026-02-01T10:00:00Z", updated_at: "2026-02-01T10:00:00Z" }),
        });

        // when
        renderPage(null);

        // then
        expect(screen.queryByText("(edited)")).not.toBeInTheDocument();
    });

    it("warns that a draft entry has not notified anyone yet", () => {
        // given
        stubEntry({ entry: makeEntry({ is_draft: true }) });

        // when
        renderPage(author);

        // then
        expect(screen.getByText("Draft")).toBeInTheDocument();
        expect(screen.getByText(/Only you can see this draft\./)).toBeInTheDocument();
    });

    it("walks to the previous entry", async () => {
        // given
        stubEntry();
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.click(screen.getAllByRole("button", { name: "← Previous" })[0]);

        // then
        expect(navigate).toHaveBeenCalledWith("/journals/journal-1/entry/1");
    });

    it("walks to the next entry", async () => {
        // given
        stubEntry();
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.click(screen.getAllByRole("button", { name: "Next →" })[0]);

        // then
        expect(navigate).toHaveBeenCalledWith("/journals/journal-1/entry/3");
    });

    it("blocks the walk at either end of the journal", () => {
        // given
        stubEntry({ entry: makeEntry({ has_prev: false, has_next: false }) });

        // when
        renderPage(null);

        // then
        for (const button of screen.getAllByRole("button", { name: "← Previous" })) {
            expect(button).toBeDisabled();
        }
        for (const button of screen.getAllByRole("button", { name: "Next →" })) {
            expect(button).toBeDisabled();
        }
    });

    it("keeps the entry controls away from an unrelated reader", () => {
        // given
        stubEntry();

        // when
        renderPage(stranger);

        // then
        expect(screen.queryByRole("button", { name: "Edit entry" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete entry" })).not.toBeInTheDocument();
    });

    it("lets the author open the entry editor", async () => {
        // given
        stubEntry();
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getByRole("button", { name: "Edit entry" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/journals/journal-1/entry/2/edit");
    });

    it("lets a moderator edit and delete somebody else's entry", () => {
        // given
        stubEntry();

        // when
        renderPage(moderator);

        // then
        expect(screen.getByRole("button", { name: "Edit entry" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete entry" })).toBeInTheDocument();
    });

    it("asks before deleting an entry", async () => {
        // given
        const { deleteEntryAsync } = stubEntry();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getByRole("button", { name: "Delete entry" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this entry? This cannot be undone.");
        expect(deleteEntryAsync).not.toHaveBeenCalled();
    });

    it("deletes the entry and returns to the journal once confirmed", async () => {
        // given
        const { deleteEntryAsync } = stubEntry();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getByRole("button", { name: "Delete entry" }));

        // then
        expect(deleteEntryAsync).toHaveBeenCalledWith("entry-9");
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/journals/journal-1");
        });
    });

    it("stays on the entry when the delete request fails", async () => {
        // given
        stubEntry({ remove: () => Promise.reject(new Error("nope")) });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getByRole("button", { name: "Delete entry" }));

        // then
        expect(navigate).not.toHaveBeenCalledWith("/journals/journal-1");
    });

    it("hangs the discussion off the entry rather than the journal", () => {
        // given
        stubEntry();

        // when
        renderPage(stranger);

        // then
        expect(screen.getByText("Comments on entry 2 on entry-9 as Battler")).toBeInTheDocument();
        expect(screen.getByText("links under /journals/journal-1/entry/2")).toBeInTheDocument();
    });

    it("closes the entry to comments once the journal is archived", () => {
        // given
        stubEntry({ journal: makeJournal({ is_archived: true }) });

        // when
        renderPage(stranger);

        // then
        expect(screen.getByText("Comments on entry 2 on entry-9 as nobody")).toBeInTheDocument();
        expect(screen.getByText("empty text: none")).toBeInTheDocument();
    });

    it("passes the comment named in the url fragment down to the discussion", () => {
        // given
        stubEntry();

        // when
        renderPage(null, "/journals/journal-1/entry/2#comment-xyz");

        // then
        expect(screen.getByText("highlighting xyz")).toBeInTheDocument();
    });

    it("returns to the journal from the back link", async () => {
        // given
        stubEntry();
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.click(screen.getByText("← Back to Rokkenjima Notes"));

        // then
        expect(navigate).toHaveBeenCalledWith("/journals/journal-1");
    });
});
