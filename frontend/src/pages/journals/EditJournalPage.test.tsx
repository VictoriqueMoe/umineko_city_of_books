import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { JournalDetail, UserProfile } from "../../types/api";
import { EditJournalPage } from "./EditJournalPage";

const { useJournal, useUpdateJournal, navigate } = vi.hoisted(() => ({
    useJournal: vi.fn(),
    useUpdateJournal: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/journal", () => ({ useJournal }));
vi.mock("../../api/mutations/journal", () => ({ useUpdateJournal }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

const author = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });
const stranger = makeUser({ id: "stranger-1", username: "battler", display_name: "Battler" });
const moderator = makeUser({ id: "mod-1", username: "ronove", display_name: "Ronove", role: "moderator" });

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
        entry_count: 1,
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
    loading?: boolean;
    update?: () => Promise<unknown>;
}

function stubEdit(options: StubOptions = {}) {
    useJournal.mockReturnValue({
        journal: options.journal === undefined ? makeJournal() : options.journal,
        loading: options.loading ?? false,
        refresh: vi.fn(),
    });
    const updateAsync = vi.fn(options.update ?? (() => Promise.resolve({})));
    useUpdateJournal.mockReturnValue({ mutateAsync: updateAsync });

    return { updateAsync };
}

function renderPage(user: UserProfile, route = "/journals/journal-1/edit") {
    return renderWithProviders(<EditJournalPage />, { user, route, path: "/journals/:id/edit" });
}

describe("EditJournalPage", () => {
    it("waits while the journal is loading", () => {
        // given
        stubEdit({ loading: true });

        // when
        renderPage(author);

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("says so when there is no such journal", () => {
        // given
        stubEdit({ journal: null });

        // when
        renderPage(author);

        // then
        expect(screen.getByText("Journal not found.")).toBeInTheDocument();
    });

    it("turns away a reader who does not own the journal", () => {
        // given
        stubEdit();

        // when
        renderPage(stranger);

        // then
        expect(screen.getByText("You can't edit this journal.")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Save" })).not.toBeInTheDocument();
    });

    it("lets a moderator edit somebody else's journal", () => {
        // given
        stubEdit();

        // when
        renderPage(moderator);

        // then
        expect(screen.getByRole("heading", { name: "Edit Journal" })).toBeInTheDocument();
    });

    it("asks for the journal named in the route", () => {
        // given
        stubEdit();

        // when
        renderPage(author, "/journals/journal-88/edit");

        // then
        expect(useJournal).toHaveBeenCalledWith("journal-88");
        expect(useUpdateJournal).toHaveBeenCalledWith("journal-88");
    });

    it("seeds the form with the journal as it stands", () => {
        // given
        stubEdit({ journal: makeJournal({ title: "Rokkenjima Notes", work: "ciconia" }) });

        // when
        renderPage(author);

        // then
        expect(screen.getByDisplayValue("Rokkenjima Notes")).toBeInTheDocument();
        expect(screen.getByRole("combobox")).toHaveValue("ciconia");
    });

    it("saves the changed title and work", async () => {
        // given
        const { updateAsync } = stubEdit();
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.clear(screen.getByDisplayValue("Rokkenjima Notes"));
        await user.type(screen.getByRole("textbox"), "Rokkenjima Re-read");
        await user.selectOptions(screen.getByRole("combobox"), "roseguns");
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(updateAsync).toHaveBeenCalledWith({ title: "Rokkenjima Re-read", work: "roseguns" });
    });

    it("returns to the journal once the edit is saved", async () => {
        // given
        stubEdit({ journal: makeJournal({ id: "journal-42" }) });
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/journals/journal-42");
        });
    });

    it("reports why the edit could not be saved", async () => {
        // given
        stubEdit({ update: () => Promise.reject(new Error("The witch forbids it")) });
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(await screen.findByText("The witch forbids it")).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
    });

    it("refuses to save a journal whose title was emptied", async () => {
        // given
        const { updateAsync } = stubEdit();
        const user = userEvent.setup();
        renderPage(author);

        // when
        await user.clear(screen.getByDisplayValue("Rokkenjima Notes"));

        // then
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
        expect(updateAsync).not.toHaveBeenCalled();
    });
});
