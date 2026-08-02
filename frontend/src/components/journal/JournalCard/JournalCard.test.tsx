import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Journal } from "../../../types/api";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import { JournalCard } from "./JournalCard";

const author = makeUser({ id: "user-1", username: "beatrice", display_name: "Beatrice" });

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

describe("JournalCard", () => {
    beforeEach(() => {
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-02-01T12:00:00Z"));
    });

    it("links the whole card to the journal it represents", () => {
        // given
        const journal = makeJournal({ id: "journal-42", title: "Rokkenjima Notes" });

        // when
        renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByRole("link", { name: "Rokkenjima Notes" })).toHaveAttribute("href", "/journals/journal-42");
    });

    it("credits the journal to its author", () => {
        // given
        const journal = makeJournal();

        // when
        renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByText("'s Reading Journal")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Beatrice/ })).toHaveAttribute("href", "/user/beatrice");
    });

    it("names the work the journal covers", () => {
        // given
        const journal = makeJournal({ work: "roseguns" });

        // when
        renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByText("Rose Guns Days")).toBeInTheDocument();
    });

    it("badges archived journals only", () => {
        // given
        const journal = makeJournal({ is_archived: true });

        // when
        const { rerender } = renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByText("Archived")).toBeInTheDocument();
        rerender(<JournalCard journal={makeJournal({ is_archived: false })} />);
        expect(screen.queryByText("Archived")).not.toBeInTheDocument();
    });

    it("hides the latest entry line for a journal with nothing written yet", () => {
        // given
        const journal = makeJournal({ latest_entry_number: undefined });

        // when
        renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.queryByText("Latest:")).not.toBeInTheDocument();
    });

    it("puts the entry title into the latest entry heading", () => {
        // given
        const journal = makeJournal({ latest_entry_number: 3, latest_entry_title: "The Golden Truth" });

        // when
        renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByText("Latest:")).toBeInTheDocument();
        expect(screen.getByText("Entry 3: The Golden Truth")).toBeInTheDocument();
    });

    it("falls back to the bare entry number when the entry title is only whitespace", () => {
        // given
        const journal = makeJournal({ latest_entry_number: 4, latest_entry_title: "   " });

        // when
        renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByText("Entry 4")).toBeInTheDocument();
    });

    it("says how long ago the latest entry was written", () => {
        // given
        const journal = makeJournal({ latest_entry_number: 1, latest_entry_at: "2026-02-01T10:00:00Z" });

        // when
        renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByText("· 2h ago")).toBeInTheDocument();
    });

    it("leaves the timestamp off when the latest entry has no date", () => {
        // given
        const journal = makeJournal({ latest_entry_number: 1, latest_entry_at: undefined });

        // when
        renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByText("Entry 1")).toBeInTheDocument();
        expect(screen.queryByText(/·/)).not.toBeInTheDocument();
    });

    it("shows the excerpt only when the latest entry has one", () => {
        // given
        const journal = makeJournal({ latest_entry_excerpt: "The witch smiled at the closed room." });

        // when
        const { rerender } = renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByText("The witch smiled at the closed room.")).toBeInTheDocument();
        rerender(<JournalCard journal={makeJournal({ latest_entry_excerpt: "" })} />);
        expect(screen.queryByText("The witch smiled at the closed room.")).not.toBeInTheDocument();
    });

    it("uses singular wording for a lone follower and a lone entry", () => {
        // given
        const journal = makeJournal({ follower_count: 1, entry_count: 1 });

        // when
        renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByText("★ 1 follower")).toBeInTheDocument();
        expect(screen.getByText("📖 1 entry")).toBeInTheDocument();
    });

    it("uses plural wording for every other count", () => {
        // given
        const journal = makeJournal({ follower_count: 0, entry_count: 5, comment_count: 2 });

        // when
        renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByText("★ 0 followers")).toBeInTheDocument();
        expect(screen.getByText("📖 5 entries")).toBeInTheDocument();
        expect(screen.getByText("💬 2")).toBeInTheDocument();
    });

    it("shows when the author last touched the journal", () => {
        // given
        const journal = makeJournal({ last_author_activity_at: "2026-01-30T12:00:00Z" });

        // when
        renderWithProviders(<JournalCard journal={journal} />);

        // then
        expect(screen.getByText("Last update 2d ago")).toBeInTheDocument();
    });
});
