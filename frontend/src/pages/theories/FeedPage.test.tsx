import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { Theory } from "../../types/api";
import type { TheorySort } from "../../types/app";
import { FeedPage } from "./FeedPage";

const { useTheoryFeed } = vi.hoisted(() => ({ useTheoryFeed: vi.fn() }));

vi.mock("../../api/queries/theory", () => ({ useTheoryFeed }));
vi.mock("../../components/RulesBox/RulesBox", () => ({
    RulesBox: ({ page }: { page: string }) => <div data-testid="rules-box">{page}</div>,
}));

const viewer = makeUser({ id: "me", username: "me", display_name: "Me" });

function makeTheory(overrides: Partial<Theory> = {}): Theory {
    return {
        id: "theory-1",
        title: "The culprit is Kanon",
        body: "Without love it cannot be seen.",
        episode: 0,
        series: "umineko",
        author: { id: "author-1", username: "beatrice", display_name: "Beatrice" },
        vote_score: 3,
        with_love_count: 2,
        without_love_count: 1,
        credibility_score: 55,
        status: "open" as const,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

interface StubOptions {
    theories?: Theory[];
    total?: number;
    loading?: boolean;
    hasNext?: boolean;
    hasPrev?: boolean;
}

function stubFeed(options: StubOptions = {}) {
    const goNext = vi.fn();
    const goPrev = vi.fn();
    useTheoryFeed.mockReturnValue({
        theories: options.theories ?? [],
        total: options.total ?? options.theories?.length ?? 0,
        loading: options.loading ?? false,
        offset: 0,
        limit: 20,
        goNext,
        goPrev,
        hasNext: options.hasNext ?? false,
        hasPrev: options.hasPrev ?? false,
        refresh: vi.fn(),
    });

    return { goNext, goPrev };
}

function lastFeedCall(): [TheorySort, number, undefined, string, string] {
    const calls = useTheoryFeed.mock.calls;
    return calls[calls.length - 1] as [TheorySort, number, undefined, string, string];
}

describe("FeedPage", () => {
    it("consults the game board while the feed is still loading", () => {
        // given
        stubFeed({ loading: true, theories: [makeTheory()] });

        // when
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // then
        expect(screen.getByText("Consulting the game board...")).toBeInTheDocument();
        expect(screen.queryByText("The culprit is Kanon")).not.toBeInTheDocument();
    });

    it("invites the first blue truth when no theories exist", () => {
        // given
        stubFeed({ theories: [] });

        // when
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // then
        expect(screen.getByText("No theories yet. Be the first to declare your blue truth.")).toBeInTheDocument();
    });

    it("renders a card for every theory in the feed", () => {
        // given
        stubFeed({
            theories: [
                makeTheory({ id: "a", title: "Beatrice did it" }),
                makeTheory({ id: "b", title: "The witch does not exist" }),
            ],
        });

        // when
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // then
        expect(screen.getByRole("link", { name: "Beatrice did it" })).toHaveAttribute("href", "/theory/a");
        expect(screen.getByRole("link", { name: "The witch does not exist" })).toHaveAttribute("href", "/theory/b");
        expect(screen.queryByText("No theories yet. Be the first to declare your blue truth.")).not.toBeInTheDocument();
    });

    it("hides the new theory button from a signed out visitor", () => {
        // given
        stubFeed();

        // when
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // then
        expect(screen.queryByRole("button", { name: "+ New Theory" })).not.toBeInTheDocument();
    });

    it("points a signed in member at the umineko composer", () => {
        // given
        stubFeed();

        // when
        renderWithProviders(<FeedPage />, { user: viewer, route: "/theories" });

        // then
        expect(screen.getByRole("link", { name: "+ New Theory" })).toHaveAttribute("href", "/theory/new");
    });

    it("points a member at the series composer when the feed is not umineko", () => {
        // given
        stubFeed();

        // when
        renderWithProviders(<FeedPage series="higurashi" />, { user: viewer, route: "/theories/higurashi" });

        // then
        expect(screen.getByRole("link", { name: "+ New Theory" })).toHaveAttribute("href", "/theory/higurashi/new");
    });

    it("titles the page after the series it is showing", () => {
        // given
        stubFeed();

        // when
        renderWithProviders(<FeedPage series="ciconia" />, { route: "/theories/ciconia" });

        // then
        expect(screen.getByRole("heading", { name: "Ciconia Theories" })).toBeInTheDocument();
        expect(screen.getByTestId("rules-box")).toHaveTextContent("theories_ciconia");
    });

    it("asks for the newest theories of the requested series by default", () => {
        // given
        stubFeed();

        // when
        renderWithProviders(<FeedPage series="higurashi" />, { route: "/theories/higurashi" });

        // then
        expect(lastFeedCall()[0]).toBe("new");
        expect(lastFeedCall()[1]).toBe(0);
        expect(lastFeedCall()[4]).toBe("higurashi");
    });

    it("flips the active sort category between descending and ascending", async () => {
        // given
        stubFeed();
        const user = userEvent.setup();
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // when
        await user.click(screen.getByRole("button", { name: /^New/ }));

        // then
        expect(lastFeedCall()[0]).toBe("old");
        await user.click(screen.getByRole("button", { name: /^New/ }));
        expect(lastFeedCall()[0]).toBe("new");
    });

    it("starts a newly chosen sort category in descending order", async () => {
        // given
        stubFeed();
        const user = userEvent.setup();
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // when
        await user.click(screen.getByRole("button", { name: /^Popular/ }));

        // then
        expect(lastFeedCall()[0]).toBe("popular");
        await user.click(screen.getByRole("button", { name: /^Popular/ }));
        expect(lastFeedCall()[0]).toBe("popular_asc");
    });

    it("marks the active sort with a direction arrow", async () => {
        // given
        stubFeed();
        const user = userEvent.setup();
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // then
        expect(screen.getByRole("button", { name: /^New/ })).toHaveTextContent("▼");

        // when
        await user.click(screen.getByRole("button", { name: /^New/ }));

        // then
        expect(screen.getByRole("button", { name: /^New/ })).toHaveTextContent("▲");
        expect(screen.getByRole("button", { name: /^Popular$/ })).toBeInTheDocument();
    });

    it("sorts by credibility when that category is picked", async () => {
        // given
        stubFeed();
        const user = userEvent.setup();
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // when
        await user.click(screen.getByRole("button", { name: /^Credibility/ }));

        // then
        expect(lastFeedCall()[0]).toBe("credibility");
    });

    it("narrows the feed to a single episode", async () => {
        // given
        stubFeed();
        const user = userEvent.setup();
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // when
        await user.selectOptions(screen.getByRole("combobox"), "4");

        // then
        expect(lastFeedCall()[1]).toBe(4);
    });

    it("offers arcs rather than episodes for a series that has them", () => {
        // given
        stubFeed();

        // when
        renderWithProviders(<FeedPage series="higurashi" />, { route: "/theories/higurashi" });

        // then
        expect(screen.getByRole("option", { name: "All Arcs" })).toBeInTheDocument();
        expect(screen.getByRole("option", { name: "Onikakushi" })).toBeInTheDocument();
        expect(screen.queryByRole("option", { name: "Episode 1" })).not.toBeInTheDocument();
    });

    it("waits for the typing to settle before searching the feed", async () => {
        // given
        stubFeed();
        const user = userEvent.setup();
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // when
        await user.type(screen.getByPlaceholderText("Search theories..."), "beato");

        // then
        expect(lastFeedCall()[3]).toBe("");
        await waitFor(() => {
            expect(lastFeedCall()[3]).toBe("beato");
        });
    });

    it("hides the pager until the feed knows how many theories exist", () => {
        // given
        stubFeed({ theories: [makeTheory()], total: 0 });

        // when
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("pages forward through a feed with more theories to show", async () => {
        // given
        const { goNext } = stubFeed({ theories: [makeTheory()], total: 40, hasNext: true });
        const user = userEvent.setup();
        renderWithProviders(<FeedPage />, { route: "/theories" });

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(goNext).toHaveBeenCalledOnce();
    });
});
