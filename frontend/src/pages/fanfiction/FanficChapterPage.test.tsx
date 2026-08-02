import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { FanficChapter, FanficDetail } from "../../types/api";
import { FanficChapterPage } from "./FanficChapterPage";

const { useFanfic, useFanficChapter, navigate } = vi.hoisted(() => ({
    useFanfic: vi.fn(),
    useFanficChapter: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/fanfic", () => ({ useFanfic, useFanficChapter }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

const author = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });

function makeChapter(overrides: Partial<FanficChapter> = {}): FanficChapter {
    return {
        id: "chapter-2",
        chapter_number: 2,
        title: "The Witch",
        body: "<p>Beatrice laughed at the closed room.</p>",
        word_count: 1500,
        has_prev: true,
        has_next: true,
        created_at: "2026-01-01T00:00:00Z",
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
        chapter_count: 3,
        favourite_count: 4,
        view_count: 90,
        comment_count: 0,
        user_favourited: false,
        published_at: "2026-01-01T00:00:00Z",
        created_at: "2026-01-01T00:00:00Z",
        chapters: [],
        comments: [],
        reading_progress: 0,
        viewer_blocked: false,
        ...overrides,
    };
}

interface StubOptions {
    chapter?: FanficChapter | null;
    fanfic?: FanficDetail | null;
    chapterLoading?: boolean;
    fanficLoading?: boolean;
}

function stubChapter(options: StubOptions = {}) {
    useFanficChapter.mockReturnValue({
        chapter: options.chapter === undefined ? makeChapter() : options.chapter,
        loading: options.chapterLoading ?? false,
        refresh: vi.fn(),
    });
    useFanfic.mockReturnValue({
        fanfic: options.fanfic === undefined ? makeFanfic() : options.fanfic,
        loading: options.fanficLoading ?? false,
        refresh: vi.fn(),
    });
}

function renderPage(route = "/fanfiction/fanfic-1/chapter/2") {
    return renderWithProviders(<FanficChapterPage />, { user: null, route, path: "/fanfiction/:id/chapter/:number" });
}

describe("FanficChapterPage", () => {
    it("waits while the chapter is loading", () => {
        // given
        stubChapter({ chapterLoading: true });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("waits while the story around the chapter is loading", () => {
        // given
        stubChapter({ fanficLoading: true });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("says so when the chapter does not exist", () => {
        // given
        stubChapter({ chapter: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Chapter not found.")).toBeInTheDocument();
    });

    it("says so when the story behind the chapter does not exist", () => {
        // given
        stubChapter({ fanfic: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Chapter not found.")).toBeInTheDocument();
    });

    it("asks for the chapter number named in the route", () => {
        // given
        stubChapter();

        // when
        renderPage("/fanfiction/fanfic-9/chapter/5");

        // then
        expect(useFanficChapter).toHaveBeenCalledWith("fanfic-9", 5);
        expect(useFanfic).toHaveBeenCalledWith("fanfic-9");
    });

    it("heads a chapter of a serial with its number and title", () => {
        // given
        stubChapter();

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Chapter 2: The Witch" })).toBeInTheDocument();
    });

    it("heads a one-shot with the story title instead", () => {
        // given
        stubChapter({ fanfic: makeFanfic({ is_oneshot: true, title: "Golden Land" }) });

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Golden Land" })).toBeInTheDocument();
    });

    it("leaves out the chapter navigation on a one-shot", () => {
        // given
        stubChapter({ fanfic: makeFanfic({ is_oneshot: true }) });

        // when
        renderPage();

        // then
        expect(screen.queryByRole("button", { name: "← Previous" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Next →" })).not.toBeInTheDocument();
    });

    it("puts the navigation both above and below a serial chapter", () => {
        // given
        stubChapter();

        // when
        renderPage();

        // then
        expect(screen.getAllByRole("button", { name: "← Previous" })).toHaveLength(2);
        expect(screen.getAllByRole("button", { name: "Next →" })).toHaveLength(2);
    });

    it("walks to the previous chapter", async () => {
        // given
        stubChapter();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getAllByRole("button", { name: "← Previous" })[0]);

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1/chapter/1");
    });

    it("walks to the next chapter", async () => {
        // given
        stubChapter();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getAllByRole("button", { name: "Next →" })[1]);

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1/chapter/3");
    });

    it("blocks the walk at either end of the story", () => {
        // given
        stubChapter({ chapter: makeChapter({ has_prev: false, has_next: false }) });

        // when
        renderPage();

        // then
        for (const button of screen.getAllByRole("button", { name: "← Previous" })) {
            expect(button).toBeDisabled();
        }
        for (const button of screen.getAllByRole("button", { name: "Next →" })) {
            expect(button).toBeDisabled();
        }
    });

    it("renders the formatting the author wrote", () => {
        // given
        stubChapter({ chapter: makeChapter({ body: "<p><em>Beatrice</em> laughed.</p>" }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("Beatrice").tagName).toBe("EM");
    });

    it("strips scripts out of the chapter body", () => {
        // given
        stubChapter({
            chapter: makeChapter({
                body: '<p>Safe text.</p><script>alert("boom")</script><img src="x" onerror="go()">',
            }),
        });

        // when
        const { container } = renderPage();

        // then
        expect(screen.getByText("Safe text.")).toBeInTheDocument();
        expect(container.querySelector("script")).toBeNull();
        expect(container.querySelector("img")?.getAttribute("onerror")).toBeNull();
    });

    it("abbreviates a long chapter's word count", () => {
        // given
        stubChapter({ chapter: makeChapter({ word_count: 1500 }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("1.5K words")).toBeInTheDocument();
    });

    it("prints a short chapter's word count in full", () => {
        // given
        stubChapter({ chapter: makeChapter({ word_count: 820 }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("820 words")).toBeInTheDocument();
    });

    it("returns to the story from the back link", async () => {
        // given
        stubChapter();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText("← Back to Golden Land"));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
    });
});
