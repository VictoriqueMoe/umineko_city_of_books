import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { Journal } from "../../types/api";
import { JournalsFeedPage } from "./FeedPage";

const { useJournalFeed } = vi.hoisted(() => ({ useJournalFeed: vi.fn() }));

vi.mock("../../api/queries/journal", () => ({ useJournalFeed }));
vi.mock("../../components/RulesBox/RulesBox", () => ({
    RulesBox: (props: { page: string }) => <div>{`rules for ${props.page}`}</div>,
}));

const author = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });
const reader = makeUser({ id: "reader-1", username: "battler", display_name: "Battler" });

function makeJournal(overrides: Partial<Journal> = {}): Journal {
    return {
        id: "journal-1",
        title: "Rokkenjima Notes",
        work: "umineko",
        author,
        follower_count: 3,
        is_following: false,
        is_archived: false,
        comment_count: 2,
        entry_count: 5,
        latest_entry_excerpt: "",
        created_at: "2026-01-01T00:00:00Z",
        last_author_activity_at: "2026-02-01T11:00:00Z",
        ...overrides,
    };
}

interface StubOptions {
    journals?: Journal[];
    total?: number;
    loading?: boolean;
    offset?: number;
    hasNext?: boolean;
    hasPrev?: boolean;
}

function stubFeed(options: StubOptions = {}) {
    const goNext = vi.fn();
    const goPrev = vi.fn();
    useJournalFeed.mockReturnValue({
        journals: options.journals ?? [],
        total: options.total ?? options.journals?.length ?? 0,
        loading: options.loading ?? false,
        offset: options.offset ?? 0,
        limit: 20,
        goNext,
        goPrev,
        hasNext: options.hasNext ?? false,
        hasPrev: options.hasPrev ?? false,
        refresh: vi.fn(),
    });

    return { goNext, goPrev };
}

describe("JournalsFeedPage", () => {
    it("explains what a reading journal is", () => {
        // given
        stubFeed();

        // when
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // then
        expect(screen.getByText("What are Reading Journals?")).toBeInTheDocument();
        expect(screen.getByText("rules for journals")).toBeInTheDocument();
    });

    it("hides the new journal link from a signed out visitor", () => {
        // given
        stubFeed();

        // when
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // then
        expect(screen.queryByRole("link", { name: /New Journal/ })).not.toBeInTheDocument();
    });

    it("offers a signed in member a link to start their own journal", () => {
        // given
        stubFeed();

        // when
        renderWithProviders(<JournalsFeedPage />, { user: reader });

        // then
        expect(screen.getByRole("link", { name: /New Journal/ })).toHaveAttribute("href", "/journals/new");
    });

    it("asks for the recently active journals of every work by default", () => {
        // given
        stubFeed();

        // when
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // then
        expect(useJournalFeed).toHaveBeenLastCalledWith("recently_active", "", "", false);
    });

    it("waits while the journals are still loading", () => {
        // given
        stubFeed({ loading: true, journals: [makeJournal()] });

        // when
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // then
        expect(screen.getByText("Turning the pages...")).toBeInTheDocument();
        expect(screen.queryByText("Rokkenjima Notes")).not.toBeInTheDocument();
    });

    it("invites the first read-through when nothing has been written yet", () => {
        // given
        stubFeed({ journals: [] });

        // when
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // then
        expect(screen.getByText("No journals yet. Be the first to start your read-through.")).toBeInTheDocument();
    });

    it("lists a card for every journal it was given", () => {
        // given
        stubFeed({
            journals: [
                makeJournal({ id: "journal-1", title: "Rokkenjima Notes" }),
                makeJournal({ id: "journal-2", title: "Hinamizawa Diary", work: "higurashi" }),
            ],
        });

        // when
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // then
        expect(screen.getByRole("link", { name: "Rokkenjima Notes" })).toHaveAttribute("href", "/journals/journal-1");
        expect(screen.getByRole("link", { name: "Hinamizawa Diary" })).toHaveAttribute("href", "/journals/journal-2");
    });

    it("re-queries with the sort the reader picked", async () => {
        // given
        stubFeed();
        const user = userEvent.setup();
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // when
        await user.click(screen.getByRole("button", { name: "Most Followed" }));

        // then
        expect(useJournalFeed).toHaveBeenLastCalledWith("most_followed", "", "", false);
    });

    it("narrows the feed to a single work when a work chip is picked", async () => {
        // given
        stubFeed();
        const user = userEvent.setup();
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // when
        await user.click(screen.getByRole("button", { name: "Rose Guns Days" }));

        // then
        expect(useJournalFeed).toHaveBeenLastCalledWith("recently_active", "roseguns", "", false);
    });

    it("widens the feed back to every work", async () => {
        // given
        stubFeed();
        const user = userEvent.setup();
        renderWithProviders(<JournalsFeedPage />, { user: null });
        await user.click(screen.getByRole("button", { name: "Ciconia" }));

        // when
        await user.click(screen.getByRole("button", { name: "All works" }));

        // then
        expect(useJournalFeed).toHaveBeenLastCalledWith("recently_active", "", "", false);
    });

    it("includes archived journals once the reader asks for them", async () => {
        // given
        stubFeed();
        const user = userEvent.setup();
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // when
        await user.click(screen.getByRole("switch", { name: "Include archived" }));

        // then
        expect(useJournalFeed).toHaveBeenLastCalledWith("recently_active", "", "", true);
    });

    it("waits for the typing to settle before searching", () => {
        // given
        vi.useFakeTimers();
        stubFeed();
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // when
        fireEvent.change(screen.getByPlaceholderText("Search journals..."), { target: { value: "golden" } });

        // then
        expect(useJournalFeed).not.toHaveBeenCalledWith("recently_active", "", "golden", false);
        act(() => {
            vi.advanceTimersByTime(300);
        });
        expect(useJournalFeed).toHaveBeenLastCalledWith("recently_active", "", "golden", false);
    });

    it("only searches for the last thing typed", () => {
        // given
        vi.useFakeTimers();
        stubFeed();
        renderWithProviders(<JournalsFeedPage />, { user: null });
        const field = screen.getByPlaceholderText("Search journals...");

        // when
        fireEvent.change(field, { target: { value: "gol" } });
        act(() => {
            vi.advanceTimersByTime(200);
        });
        fireEvent.change(field, { target: { value: "golden" } });
        act(() => {
            vi.advanceTimersByTime(300);
        });

        // then
        expect(useJournalFeed).not.toHaveBeenCalledWith("recently_active", "", "gol", false);
        expect(useJournalFeed).toHaveBeenLastCalledWith("recently_active", "", "golden", false);
    });

    it("hides the pager while the journals are loading", () => {
        // given
        stubFeed({ loading: true, total: 40, hasNext: true });

        // when
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("pages forward through the feed", async () => {
        // given
        const { goNext } = stubFeed({ journals: [makeJournal()], total: 45, hasNext: true });
        const user = userEvent.setup();
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(goNext).toHaveBeenCalledOnce();
        expect(screen.getByText("1-20 of 45")).toBeInTheDocument();
    });

    it("pages back through the feed", async () => {
        // given
        const { goPrev } = stubFeed({ journals: [makeJournal()], total: 45, offset: 20, hasPrev: true });
        const user = userEvent.setup();
        renderWithProviders(<JournalsFeedPage />, { user: null });

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(goPrev).toHaveBeenCalledOnce();
        expect(screen.getByText("21-40 of 45")).toBeInTheDocument();
    });
});
