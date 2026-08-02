import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { Announcement } from "../../types/api";
import { AnnouncementsPage } from "./AnnouncementsPage";

const { useAnnouncementList } = vi.hoisted(() => ({ useAnnouncementList: vi.fn() }));

vi.mock("../../api/queries/announcement", () => ({ useAnnouncementList }));

const author = { id: "author-1", username: "beatrice", display_name: "Beatrice" };

function makeAnnouncement(overrides: Partial<Announcement> = {}): Announcement {
    return {
        id: "announcement-1",
        title: "The board reopens",
        body: "The game board is open again.",
        author,
        pinned: false,
        created_at: "2026-07-01T10:00:00Z",
        updated_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

interface StubOptions {
    announcements?: Announcement[];
    total?: number;
    loading?: boolean;
}

function stubList(options: StubOptions = {}) {
    useAnnouncementList.mockReturnValue({
        announcements: options.announcements ?? [],
        total: options.total ?? options.announcements?.length ?? 0,
        loading: options.loading ?? false,
        refresh: vi.fn(),
    });
}

function renderPage() {
    return renderWithProviders(<AnnouncementsPage />, { route: "/announcements" });
}

function cardFor(title: string): HTMLElement {
    return screen.getByText(title).closest("a") as HTMLElement;
}

describe("AnnouncementsPage", () => {
    it("waits while the announcements are loading", () => {
        // given
        stubList({ loading: true, announcements: [makeAnnouncement()] });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading announcements...")).toBeInTheDocument();
        expect(screen.queryByText("The board reopens")).not.toBeInTheDocument();
    });

    it("says there is nothing to read when the board has never spoken", () => {
        // given
        stubList({ announcements: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText("No announcements yet.")).toBeInTheDocument();
    });

    it("asks for the first page of twenty announcements", () => {
        // given
        stubList();

        // when
        renderPage();

        // then
        expect(useAnnouncementList).toHaveBeenLastCalledWith(20, 0);
    });

    it("links every announcement to its own page", () => {
        // given
        stubList({
            announcements: [
                makeAnnouncement({ id: "a-1", title: "The board reopens" }),
                makeAnnouncement({ id: "a-2", title: "New rules" }),
            ],
        });

        // when
        renderPage();

        // then
        expect(cardFor("The board reopens")).toHaveAttribute("href", "/announcements/a-1");
        expect(cardFor("New rules")).toHaveAttribute("href", "/announcements/a-2");
    });

    it("marks the announcement the staff pinned", () => {
        // given
        stubList({
            announcements: [
                makeAnnouncement({ id: "a-1", title: "Read this first", pinned: true }),
                makeAnnouncement({ id: "a-2", title: "New rules", pinned: false }),
            ],
        });

        // when
        renderPage();

        // then
        expect(cardFor("Read this first")).toHaveTextContent("Pinned");
        expect(cardFor("New rules")).not.toHaveTextContent("Pinned");
    });

    it("strips the markdown out of the preview", () => {
        // given
        stubList({
            announcements: [makeAnnouncement({ body: "## Heading\n\n**Bold** and _quiet_ words." })],
        });

        // when
        renderPage();

        // then
        expect(screen.getByText("Heading Bold and quiet words.")).toBeInTheDocument();
    });

    it("leaves hyphens and exclamation marks alone in the preview", () => {
        // given
        stubList({
            announcements: [makeAnnouncement({ body: "The well-known witch is back. Watch out!" })],
        });

        // when
        renderPage();

        // then
        expect(screen.getByText("The well-known witch is back. Watch out!")).toBeInTheDocument();
    });

    it("cuts a very long announcement down to a preview", () => {
        // given
        const body = "a".repeat(250);
        stubList({ announcements: [makeAnnouncement({ body })] });

        // when
        renderPage();

        // then
        expect(screen.getByText(`${"a".repeat(200)}...`)).toBeInTheDocument();
    });

    it("credits the author without turning the card into a nested link", () => {
        // given
        stubList({ announcements: [makeAnnouncement()] });

        // when
        renderPage();

        // then
        expect(cardFor("The board reopens")).toHaveTextContent("Beatrice");
        expect(screen.queryByRole("link", { name: "Beatrice" })).not.toBeInTheDocument();
    });

    it("hides the pager when there is nothing to page through", () => {
        // given
        stubList({ announcements: [], total: 0 });

        // when
        renderPage();

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("pages forward through the announcements", async () => {
        // given
        stubList({ announcements: [makeAnnouncement()], total: 45 });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(useAnnouncementList).toHaveBeenLastCalledWith(20, 20);
    });

    it("pages back through the announcements", async () => {
        // given
        stubList({ announcements: [makeAnnouncement()], total: 45 });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Next" }));

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(useAnnouncementList).toHaveBeenLastCalledWith(20, 0);
    });

    it("refuses to page past the first announcement", async () => {
        // given
        stubList({ announcements: [makeAnnouncement()], total: 45 });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
        expect(useAnnouncementList).toHaveBeenLastCalledWith(20, 0);
    });
});
