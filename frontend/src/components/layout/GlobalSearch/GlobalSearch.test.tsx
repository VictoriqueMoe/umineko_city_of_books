import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SearchResult } from "../../../types/api";
import { renderWithProviders } from "../../../test-utils/render";
import { GlobalSearch } from "./GlobalSearch";

const { navigate } = vi.hoisted(() => ({ navigate: vi.fn() }));
const { useQuickSearch } = vi.hoisted(() => ({ useQuickSearch: vi.fn() }));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => navigate };
});

vi.mock("../../../api/queries/search", () => ({ useQuickSearch }));

function makeResult(overrides: Partial<SearchResult> = {}): SearchResult {
    return {
        type: "theory",
        id: "theory-1",
        parent_id: null,
        parent_title: null,
        title: "Red truth",
        snippet: "",
        url: "/theory/theory-1",
        author: { id: "user-1", username: "beatrice", display_name: "Beatrice", avatar_url: "" },
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

const theories = [
    makeResult({ id: "theory-1", title: "Red truth", url: "/theory/theory-1" }),
    makeResult({ id: "theory-2", title: "Blue truth", url: "/theory/theory-2" }),
];

function setupSearch(results: SearchResult[] = [], loading = false) {
    useQuickSearch.mockReturnValue({ results, loading });
    const user = userEvent.setup();
    renderWithProviders(<GlobalSearch />);

    return user;
}

function searchBox(): HTMLElement {
    return screen.getByRole("combobox", { name: "Search the site" });
}

const mixedGroups = [
    makeResult({ type: "user", id: "user-9", title: "Beatrice", url: "/user/beatrice" }),
    makeResult({ type: "theory", id: "theory-1", title: "Red truth", url: "/theory/theory-1" }),
];

function flushDebounce(): void {
    act(() => {
        vi.advanceTimersByTime(200);
    });
}

beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    useQuickSearch.mockReturnValue({ results: [], loading: false });
});

describe("GlobalSearch querying", () => {
    it("asks for nothing while the search box has never been touched", () => {
        // given
        setupSearch(theories);

        // when
        const lastCall = useQuickSearch.mock.lastCall;

        // then
        expect(lastCall).toEqual(["", false]);
    });

    it("holds the query back until the typing has settled", async () => {
        // given
        const user = setupSearch(theories);

        // when
        await user.type(searchBox(), "beat");

        // then
        expect(useQuickSearch).toHaveBeenLastCalledWith("", true);
    });

    it("sends the query once the typing has settled", async () => {
        // given
        const user = setupSearch(theories);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        expect(useQuickSearch).toHaveBeenLastCalledWith("beat", true);
    });

    it("trims the surrounding whitespace off the query", async () => {
        // given
        const user = setupSearch(theories);

        // when
        await user.type(searchBox(), "  beatrice  ");
        flushDebounce();

        // then
        expect(useQuickSearch).toHaveBeenLastCalledWith("beatrice", true);
    });

    it("keeps the dropdown shut for a single character", async () => {
        // given
        const user = setupSearch(theories);

        // when
        await user.type(searchBox(), "b");
        flushDebounce();

        // then
        expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    });

    it("opens the dropdown once there are two characters to search on", async () => {
        // given
        const user = setupSearch(theories);

        // when
        await user.type(searchBox(), "be");
        flushDebounce();

        // then
        expect(screen.getByRole("listbox")).toBeInTheDocument();
    });
});

describe("GlobalSearch dropdown states", () => {
    it("says it is searching while the first results are still on their way", async () => {
        // given
        const user = setupSearch([], true);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        expect(screen.getByText("Searching...")).toBeInTheDocument();
    });

    it("reports an empty result set naming the query", async () => {
        // given
        const user = setupSearch([], false);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        expect(screen.getByText('No results for "beat".')).toBeInTheDocument();
    });

    it("keeps the previous results on screen while a refetch is in flight", async () => {
        // given
        const user = setupSearch(theories, true);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        expect(screen.getByText("Red truth")).toBeInTheDocument();
        expect(screen.queryByText("Searching...")).not.toBeInTheDocument();
    });

    it("groups the results under their section headings in display order", async () => {
        // given
        const mixed = [
            makeResult({ type: "user", id: "user-9", title: "Beatrice", url: "/user/beatrice" }),
            makeResult({ type: "post", id: "post-1", title: "Board post", url: "/post/post-1" }),
            makeResult({ type: "theory", id: "theory-1", title: "Red truth", url: "/theory/theory-1" }),
        ];
        const user = setupSearch(mixed);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        const text = screen.getByRole("listbox").textContent ?? "";
        expect(text.indexOf("Theories")).toBeGreaterThanOrEqual(0);
        expect(text.indexOf("Theories")).toBeLessThan(text.indexOf("Game Board"));
        expect(text.indexOf("Game Board")).toBeLessThan(text.indexOf("Users"));
    });

    it("leaves out the heading of a group with no results", async () => {
        // given
        const user = setupSearch(theories);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        expect(screen.getByText("Theories")).toBeInTheDocument();
        expect(screen.queryByText("Mysteries")).not.toBeInTheDocument();
    });

    it("offers a way through to the full results page", async () => {
        // given
        const user = setupSearch(theories);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        expect(screen.getByRole("button", { name: 'See all results for "beat"' })).toBeInTheDocument();
    });

    it("offers no way through to the full results page when nothing was found", async () => {
        // given
        const user = setupSearch([]);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        expect(screen.queryByRole("button", { name: /See all results/ })).not.toBeInTheDocument();
    });
});

describe("GlobalSearch keyboard handling", () => {
    it("opens the highlighted result on enter", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.keyboard("{ArrowDown}{ArrowDown}{Enter}");

        // then
        expect(navigate).toHaveBeenCalledWith("/theory/theory-2");
    });

    it("stops at the last result when the arrow key is held past the end", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.keyboard("{ArrowDown}{ArrowDown}{ArrowDown}{ArrowDown}{Enter}");

        // then
        expect(navigate).toHaveBeenCalledWith("/theory/theory-2");
    });

    it("gives up the highlight when the arrow key walks back past the top", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.keyboard("{ArrowDown}{ArrowUp}{ArrowUp}{Enter}");

        // then
        expect(navigate).toHaveBeenCalledWith("/search?q=beat");
    });

    it("forgets the highlight when the query is edited", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.keyboard("{ArrowDown}r");
        flushDebounce();
        await user.keyboard("{Enter}");

        // then
        expect(navigate).toHaveBeenCalledWith("/search?q=beatr");
    });

    it("falls back to the search page when the highlighted result has no destination", async () => {
        // given
        const user = setupSearch([makeResult({ url: "" })]);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.keyboard("{ArrowDown}{Enter}");

        // then
        expect(navigate).toHaveBeenCalledWith("/search?q=beat");
    });

    it("opens the search page on enter when nothing is highlighted", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.keyboard("{Enter}");

        // then
        expect(navigate).toHaveBeenCalledWith("/search?q=beat");
    });

    it("walks the results in the order they are shown rather than the order they arrived", async () => {
        // given
        const user = setupSearch(mixedGroups);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.keyboard("{ArrowDown}{Enter}");

        // then
        expect(navigate).toHaveBeenCalledWith("/theory/theory-1");
    });

    it("opens the last shown result when the arrow key walks to the bottom of a regrouped list", async () => {
        // given
        const user = setupSearch(mixedGroups);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.keyboard("{ArrowDown}{ArrowDown}{Enter}");

        // then
        expect(navigate).toHaveBeenCalledWith("/user/beatrice");
    });

    it("highlights the row the visitor can see rather than one further down the list", async () => {
        // given
        const user = setupSearch(mixedGroups);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.keyboard("{ArrowDown}");

        // then
        const options = screen.getAllByRole("option");
        expect(options[0]).toHaveAttribute("aria-selected", "true");
        expect(options[1]).toHaveAttribute("aria-selected", "false");
    });

    it("closes the dropdown when escape is pressed", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.keyboard("{Escape}");

        // then
        expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
    });

    it("reopens the dropdown when the search box is focused again", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();
        await user.keyboard("{Escape}");

        // when
        await user.click(searchBox());

        // then
        expect(screen.getByRole("listbox")).toBeInTheDocument();
    });
});

describe("GlobalSearch assistive technology wiring", () => {
    it("announces the search box as a combobox that owns the results list", async () => {
        // given
        const user = setupSearch(theories);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        expect(searchBox()).toHaveAttribute("aria-expanded", "true");
        expect(searchBox()).toHaveAttribute("aria-controls", screen.getByRole("listbox").id);
        expect(searchBox()).toHaveAttribute("aria-autocomplete", "list");
    });

    it("says the results list is shut while the dropdown is away", () => {
        // given
        setupSearch(theories);

        // then
        expect(searchBox()).toHaveAttribute("aria-expanded", "false");
    });

    it("offers every row as an option of the results list", async () => {
        // given
        const user = setupSearch(theories);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        expect(screen.getAllByRole("option").map(option => option.textContent)).toEqual([
            expect.stringContaining("Red truth"),
            expect.stringContaining("Blue truth"),
        ]);
    });

    it("names each section of the results list", async () => {
        // given
        const user = setupSearch(mixedGroups);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        expect(screen.getAllByRole("group").map(group => group.getAttribute("aria-label"))).toEqual([
            "Theories",
            "Users",
        ]);
    });

    it("points the search box at the highlighted option", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.keyboard("{ArrowDown}{ArrowDown}");

        // then
        expect(searchBox()).toHaveAttribute("aria-activedescendant", screen.getAllByRole("option")[1].id);
    });

    it("points the search box at nothing while no option is highlighted", async () => {
        // given
        const user = setupSearch(theories);

        // when
        await user.type(searchBox(), "beat");
        flushDebounce();

        // then
        expect(searchBox()).not.toHaveAttribute("aria-activedescendant");
    });
});

describe("GlobalSearch submitting", () => {
    it("opens the search page for the typed query when the button is pressed", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.click(screen.getByRole("button", { name: "Open search page" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/search?q=beat");
    });

    it("encodes a query that contains reserved characters", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beatrice & battler");
        flushDebounce();

        // when
        await user.click(screen.getByRole("button", { name: "Open search page" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/search?q=beatrice%20%26%20battler");
    });

    it("opens the bare search page when the box is empty", async () => {
        // given
        const user = setupSearch(theories);

        // when
        await user.click(screen.getByRole("button", { name: "Open search page" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/search");
    });

    it("opens the bare search page when the box holds only whitespace", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "   ");
        flushDebounce();

        // when
        await user.click(screen.getByRole("button", { name: "Open search page" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/search");
    });

    it("opens the search page from the see all results row", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.click(screen.getByRole("button", { name: 'See all results for "beat"' }));

        // then
        expect(navigate).toHaveBeenCalledWith("/search?q=beat");
        expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    });

    it("closes the dropdown when a result is chosen", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        await user.click(screen.getByRole("option", { name: /Red truth/ }));

        // then
        expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    });

    it("closes the dropdown when the visitor clicks elsewhere on the page", async () => {
        // given
        const user = setupSearch(theories);
        await user.type(searchBox(), "beat");
        flushDebounce();

        // when
        fireEvent.mouseDown(document.body);

        // then
        expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    });
});
