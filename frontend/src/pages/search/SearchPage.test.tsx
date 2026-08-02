import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SearchResult } from "../../types/api";
import { renderWithProviders } from "../../test-utils/render";
import { SearchPage } from "./SearchPage";

const mocks = vi.hoisted(() => ({ useSiteSearch: vi.fn() }));

vi.mock("../../api/queries/search", () => ({ useSiteSearch: mocks.useSiteSearch }));

vi.mock("../../components/layout/GlobalSearch/SearchResultRow", () => ({
    SearchResultRow: ({ result }: { result: SearchResult }) => <div data-testid="search-result">{result.title}</div>,
}));

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

interface SearchState {
    results?: SearchResult[];
    total?: number;
    loading?: boolean;
    fetching?: boolean;
}

function setup(route: string, state: SearchState = {}) {
    mocks.useSiteSearch.mockReturnValue({
        results: state.results ?? [],
        total: state.total ?? 0,
        loading: state.loading ?? false,
        fetching: state.fetching ?? false,
    });
    const user = userEvent.setup();
    const result = renderWithProviders(<SearchPage />, { route });

    return { user, ...result };
}

function searchBox(): HTMLElement {
    return screen.getByPlaceholderText("Search anything...");
}

beforeEach(() => {
    mocks.useSiteSearch.mockReturnValue({ results: [], total: 0, loading: false, fetching: false });
});

describe("SearchPage reading the address bar", () => {
    it("asks for nothing while the address bar carries no query", () => {
        // given
        const route = "/search";

        // when
        setup(route);

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("", "", 20, 0);
        expect(screen.getByText("Enter at least 2 characters to search.")).toBeInTheDocument();
    });

    it("nudges the visitor when the query is a single character", () => {
        // given
        const route = "/search?q=b";

        // when
        setup(route);

        // then
        expect(screen.getByText("Enter at least 2 characters.")).toBeInTheDocument();
        expect(screen.queryByTestId("search-result")).not.toBeInTheDocument();
    });

    it("passes the query and the section filter straight through", () => {
        // given
        const route = "/search?q=beatrice&type=theory,response";

        // when
        setup(route);

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("beatrice", "theory,response", 20, 0);
    });

    it("turns the page number into an offset", () => {
        // given
        const route = "/search?q=beatrice&page=2";

        // when
        setup(route);

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("beatrice", "", 20, 40);
    });

    it("treats a negative page as the first one", () => {
        // given
        const route = "/search?q=beatrice&page=-3";

        // when
        setup(route);

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("beatrice", "", 20, 0);
    });

    it("treats a page that is not a number as the first one", () => {
        // given
        const route = "/search?q=beatrice&page=abc";

        // when
        setup(route);

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("beatrice", "", 20, 0);
    });

    it("fills the search box with the query from the address bar", () => {
        // given
        const route = "/search?q=beatrice";

        // when
        setup(route);

        // then
        expect(searchBox()).toHaveValue("beatrice");
    });
});

describe("SearchPage result states", () => {
    it("says it is searching while the first results are on their way", () => {
        // given
        const route = "/search?q=beatrice";

        // when
        setup(route, { loading: true });

        // then
        expect(screen.getByText("Searching...")).toBeInTheDocument();
    });

    it("counts a lone result in the singular", () => {
        // given
        const route = "/search?q=beatrice";

        // when
        setup(route, { results: [makeResult()], total: 1 });

        // then
        expect(screen.getByText('1 result for "beatrice"')).toBeInTheDocument();
    });

    it("counts several results in the plural", () => {
        // given
        const route = "/search?q=beatrice";

        // when
        setup(route, { results: [makeResult(), makeResult({ id: "theory-2", title: "Blue truth" })], total: 2 });

        // then
        expect(screen.getByText('2 results for "beatrice"')).toBeInTheDocument();
    });

    it("keeps the old results visible while a refetch is in flight", () => {
        // given
        const route = "/search?q=beatrice";

        // when
        setup(route, { results: [makeResult()], total: 1, fetching: true });

        // then
        expect(screen.getByText("updating...")).toBeInTheDocument();
        expect(screen.getByTestId("search-result")).toHaveTextContent("Red truth");
    });

    it("suggests loosening the filter when nothing was found", () => {
        // given
        const route = "/search?q=beatrice";

        // when
        setup(route, { results: [], total: 0 });

        // then
        expect(screen.getByText("Nothing found. Try different keywords or remove the filter.")).toBeInTheDocument();
    });

    it("shows a row for every result", () => {
        // given
        const route = "/search?q=beatrice";

        // when
        setup(route, {
            results: [makeResult(), makeResult({ id: "theory-2", title: "Blue truth" })],
            total: 2,
        });

        // then
        expect(screen.getAllByTestId("search-result")).toHaveLength(2);
    });
});

describe("SearchPage changing the query", () => {
    it("searches for whatever was typed into the box", async () => {
        // given
        const { user } = setup("/search");

        // when
        await user.type(searchBox(), "battler");
        await user.click(screen.getByRole("button", { name: "Search" }));

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("battler", "", 20, 0);
    });

    it("trims the whitespace around the typed query", async () => {
        // given
        const { user } = setup("/search");

        // when
        await user.type(searchBox(), "   battler   ");
        await user.click(screen.getByRole("button", { name: "Search" }));

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("battler", "", 20, 0);
    });

    it("drops the query when the box is emptied", async () => {
        // given
        const { user } = setup("/search?q=beatrice");

        // when
        await user.clear(searchBox());
        await user.click(screen.getByRole("button", { name: "Search" }));

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("", "", 20, 0);
        expect(screen.getByText("Enter at least 2 characters to search.")).toBeInTheDocument();
    });

    it("returns to the first page when a new query is searched", async () => {
        // given
        const { user } = setup("/search?q=beatrice&page=3");

        // when
        await user.clear(searchBox());
        await user.type(searchBox(), "battler");
        await user.click(screen.getByRole("button", { name: "Search" }));

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("battler", "", 20, 0);
    });
});

describe("SearchPage section filters", () => {
    it("narrows the search to the chosen section", async () => {
        // given
        const { user } = setup("/search?q=beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Theories" }));

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("beatrice", "theory,response", 20, 0);
    });

    it("returns to the first page when the section changes", async () => {
        // given
        const { user } = setup("/search?q=beatrice&page=2");

        // when
        await user.click(screen.getByRole("button", { name: "Users" }));

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("beatrice", "user", 20, 0);
    });

    it("clears the section filter again", async () => {
        // given
        const { user } = setup("/search?q=beatrice&type=theory,response");

        // when
        await user.click(screen.getByRole("button", { name: "All" }));

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("beatrice", "", 20, 0);
    });

    it("spans every section with the comments only filter", async () => {
        // given
        const { user } = setup("/search?q=beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Comments only" }));

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("beatrice", "comments", 20, 0);
    });
});

describe("SearchPage paging", () => {
    it("hides the pager when there is nothing to page through", () => {
        // given
        const route = "/search?q=beatrice";

        // when
        setup(route, { results: [], total: 0 });

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("walks forward a page at a time", async () => {
        // given
        const { user } = setup("/search?q=beatrice", { results: [makeResult()], total: 45 });

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("beatrice", "", 20, 20);
    });

    it("walks back a page at a time", async () => {
        // given
        const { user } = setup("/search?q=beatrice&page=2", { results: [makeResult()], total: 45 });

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(mocks.useSiteSearch).toHaveBeenLastCalledWith("beatrice", "", 20, 20);
    });

    it("refuses to walk back from the first page", () => {
        // given
        const route = "/search?q=beatrice";

        // when
        setup(route, { results: [makeResult()], total: 45 });

        // then
        expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    });
});
