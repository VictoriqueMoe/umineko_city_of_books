import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { Quote } from "../../../types/api";
import { TruthPicker } from "./TruthPicker";

const mocks = vi.hoisted(() => ({
    useSearchQuotes: vi.fn(),
    useBrowseQuotes: vi.fn(),
    useCharacterGroups: vi.fn(),
}));

vi.mock("../../../api/queries/quote", () => ({
    useSearchQuotes: mocks.useSearchQuotes,
    useBrowseQuotes: mocks.useBrowseQuotes,
}));

vi.mock("../../../api/queries/characters", () => ({
    useCharacterGroups: mocks.useCharacterGroups,
}));

function makeQuote(overrides: Partial<Quote> = {}): Quote {
    return {
        text: "Without love, it cannot be seen.",
        textHtml: "Without love, it cannot be seen.",
        characterId: "beatrice",
        character: "Beatrice",
        audioId: "ep1_0001",
        episode: 4,
        contentType: "dialogue",
        hasRedTruth: false,
        hasBlueTruth: false,
        hasGoldTruth: false,
        hasPurpleTruth: false,
        index: 12,
        ...overrides,
    };
}

function browseResult(quotes: Quote[], total = quotes.length, loading = false) {
    return { data: { quotes, total }, loading };
}

function searchResult(quotes: Quote[], total = quotes.length, loading = false) {
    return { data: { results: quotes.map(quote => ({ quote, score: 1 })), total }, loading };
}

function lastCall(calls: unknown[][]): [Record<string, unknown>, boolean] {
    return calls[calls.length - 1] as [Record<string, unknown>, boolean];
}

function lastBrowseCall(): [Record<string, unknown>, boolean] {
    return lastCall(mocks.useBrowseQuotes.mock.calls);
}

function lastSearchCall(): [Record<string, unknown>, boolean] {
    return lastCall(mocks.useSearchQuotes.mock.calls);
}

function selectOwning(optionLabel: string): HTMLSelectElement {
    const select = screen.getByRole("option", { name: optionLabel }).closest("select");
    if (!select) {
        throw new Error(`no select owns the option ${optionLabel}`);
    }
    return select;
}

function noop() {}

interface PickerOverrides {
    onClose?: () => void;
    onSelect?: (quote: Quote, lang: string) => void;
    selectedKeys?: string[];
    series?: "umineko" | "higurashi" | "ciconia";
    isOpen?: boolean;
}

function renderPicker(overrides: PickerOverrides = {}) {
    return renderWithProviders(
        <TruthPicker
            isOpen={overrides.isOpen ?? true}
            onClose={overrides.onClose ?? noop}
            onSelect={overrides.onSelect ?? noop}
            selectedKeys={overrides.selectedKeys ?? []}
            series={overrides.series}
        />,
    );
}

beforeEach(() => {
    mocks.useBrowseQuotes.mockReturnValue(browseResult([]));
    mocks.useSearchQuotes.mockReturnValue({ data: null, loading: false });
    mocks.useCharacterGroups.mockReturnValue({ groups: { main: {}, additional: {} }, loading: false });
});

describe("TruthPicker", () => {
    it("renders nothing and asks for nothing while it is closed", () => {
        // given
        const isOpen = false;

        // when
        const { container } = renderPicker({ isOpen });

        // then
        expect(container).toBeEmptyDOMElement();
        expect(mocks.useBrowseQuotes).not.toHaveBeenCalled();
        expect(mocks.useCharacterGroups).not.toHaveBeenCalled();
    });

    it("lists the browsed quotes when nothing has been searched for", () => {
        // given
        mocks.useBrowseQuotes.mockReturnValue(
            browseResult([
                makeQuote({ audioId: "a1", text: "First truth", textHtml: "First truth" }),
                makeQuote({ audioId: "a2", text: "Second truth", textHtml: "Second truth" }),
            ]),
        );

        // when
        renderPicker();

        // then
        expect(screen.getByText("First truth")).toBeInTheDocument();
        expect(screen.getByText("Second truth")).toBeInTheDocument();
        expect(lastBrowseCall()[1]).toBe(true);
        expect(lastSearchCall()[1]).toBe(false);
    });

    it("shows an empty state when the browse comes back with nothing", () => {
        // given
        mocks.useBrowseQuotes.mockReturnValue(browseResult([]));

        // when
        renderPicker();

        // then
        expect(screen.getByText("No quotes found.")).toBeInTheDocument();
    });

    it("holds back the empty state while the results are still loading", () => {
        // given
        mocks.useBrowseQuotes.mockReturnValue(browseResult([], 0, true));

        // when
        renderPicker();

        // then
        expect(screen.queryByText("No quotes found.")).not.toBeInTheDocument();
    });

    it("switches to searching with a trimmed query once the form is submitted", async () => {
        // given
        const user = userEvent.setup();
        mocks.useSearchQuotes.mockReturnValue(
            searchResult([makeQuote({ text: "A found truth", textHtml: "A found truth" })]),
        );
        renderPicker();

        // when
        await user.type(screen.getByPlaceholderText("Search quotes..."), "   the epitaph   ");
        await user.click(screen.getByRole("button", { name: "Search" }));

        // then
        expect(lastSearchCall()[0].query).toBe("the epitaph");
        expect(lastSearchCall()[1]).toBe(true);
        expect(lastBrowseCall()[1]).toBe(false);
        expect(screen.getByText("A found truth")).toBeInTheDocument();
    });

    it("keeps browsing while the query has only been typed and not submitted", async () => {
        // given
        const user = userEvent.setup();
        renderPicker();

        // when
        await user.type(screen.getByPlaceholderText("Search quotes..."), "epitaph");

        // then
        expect(lastSearchCall()[1]).toBe(false);
        expect(lastBrowseCall()[1]).toBe(true);
    });

    it("passes the chosen filters through to the browse query", async () => {
        // given
        const user = userEvent.setup();
        mocks.useCharacterGroups.mockReturnValue({
            groups: { main: { beatrice: "Beatrice", battler: "Battler" }, additional: {} },
            loading: false,
        });
        renderPicker();

        // when
        await user.selectOptions(selectOwning("All Characters"), "beatrice");
        await user.selectOptions(selectOwning("All Types"), "red");
        await user.selectOptions(selectOwning("Default Language"), "ja");
        await user.selectOptions(selectOwning("All Episodes"), "4");

        // then
        expect(lastBrowseCall()[0]).toMatchObject({
            character: "beatrice",
            truth: "red",
            lang: "ja",
            episode: 4,
            series: "umineko",
            limit: 20,
            offset: 0,
        });
    });

    it("leaves untouched filters out of the browse query", () => {
        // given
        const series = "umineko";

        // when
        renderPicker({ series });

        // then
        expect(lastBrowseCall()[0]).toMatchObject({
            character: undefined,
            episode: undefined,
            arc: undefined,
            chapter: undefined,
            truth: undefined,
            lang: undefined,
        });
    });

    it("reports the picked quote with English as the default language", async () => {
        // given
        const onSelect = vi.fn();
        const user = userEvent.setup();
        const quote = makeQuote({ text: "Pick me", textHtml: "Pick me" });
        mocks.useBrowseQuotes.mockReturnValue(browseResult([quote]));
        renderPicker({ onSelect });

        // when
        await user.click(screen.getByText("Pick me"));

        // then
        expect(onSelect).toHaveBeenCalledWith(quote, "en");
    });

    it("reports the picked quote with the language that is currently chosen", async () => {
        // given
        const onSelect = vi.fn();
        const user = userEvent.setup();
        const quote = makeQuote({ text: "Pick me", textHtml: "Pick me" });
        mocks.useBrowseQuotes.mockReturnValue(browseResult([quote]));
        renderPicker({ onSelect });

        // when
        await user.selectOptions(selectOwning("Default Language"), "ru");
        await user.click(screen.getByText("Pick me"));

        // then
        expect(onSelect).toHaveBeenCalledWith(quote, "ru");
    });

    it("offers arcs instead of episodes for higurashi", () => {
        // given
        const series = "higurashi";

        // when
        renderPicker({ series });

        // then
        expect(screen.getByRole("option", { name: "All Arcs" })).toBeInTheDocument();
        expect(screen.getByRole("option", { name: "Onikakushi" })).toBeInTheDocument();
        expect(screen.queryByRole("option", { name: "All Episodes" })).not.toBeInTheDocument();
    });

    it("sends the chosen arc as the arc filter", async () => {
        // given
        const user = userEvent.setup();
        renderPicker({ series: "higurashi" });

        // when
        await user.selectOptions(selectOwning("All Arcs"), "meakashi");

        // then
        expect(lastBrowseCall()[0]).toMatchObject({ arc: "meakashi", episode: undefined, chapter: undefined });
    });

    it("offers chapters for ciconia", async () => {
        // given
        const user = userEvent.setup();
        renderPicker({ series: "ciconia" });

        // when
        await user.selectOptions(selectOwning("All Chapters"), "00");

        // then
        expect(screen.getByRole("option", { name: "Prologue" })).toBeInTheDocument();
        expect(lastBrowseCall()[0]).toMatchObject({ chapter: "00", arc: undefined, episode: undefined });
    });

    it("groups the characters into main cast and additional when there are both", () => {
        // given
        mocks.useCharacterGroups.mockReturnValue({
            groups: { main: { beatrice: "Beatrice" }, additional: { kanon: "Kanon" } },
            loading: false,
        });

        // when
        const { container } = renderPicker();

        // then
        const labels = Array.from(container.querySelectorAll("optgroup")).map(group => group.getAttribute("label"));
        expect(labels).toEqual(["Main cast", "Additional"]);
    });

    it("lists the characters flat when there are no additional ones", () => {
        // given
        mocks.useCharacterGroups.mockReturnValue({
            groups: { main: { beatrice: "Beatrice", battler: "Battler" }, additional: {} },
            loading: false,
        });

        // when
        const { container } = renderPicker();

        // then
        expect(container.querySelectorAll("optgroup")).toHaveLength(0);
        expect(screen.getByRole("option", { name: "Beatrice" })).toBeInTheDocument();
    });

    it("sorts the character options by name", () => {
        // given
        mocks.useCharacterGroups.mockReturnValue({
            groups: { main: { z: "Zepar", b: "Beatrice", l: "Lambdadelta" }, additional: {} },
            loading: false,
        });

        // when
        renderPicker();

        // then
        const names = Array.from(selectOwning("All Characters").options).map(option => option.textContent);
        expect(names).toEqual(["All Characters", "Beatrice", "Lambdadelta", "Zepar"]);
    });

    it("names every filter after what it actually filters", () => {
        // given
        const series = "umineko";

        // when
        renderPicker({ series });

        // then
        expect(screen.getByLabelText("Filter by episode")).toBe(selectOwning("All Episodes"));
        expect(screen.getByLabelText("Filter by character")).toBe(selectOwning("All Characters"));
        expect(screen.getByLabelText("Filter by truth type")).toBe(selectOwning("All Types"));
        expect(screen.getByLabelText("Quote language")).toBe(selectOwning("Default Language"));
    });

    it("names the segment filter after the segment the series actually uses", () => {
        // given
        const series = "higurashi";

        // when
        renderPicker({ series });

        // then
        expect(screen.getByLabelText("Filter by arc")).toBe(selectOwning("All Arcs"));
        expect(screen.queryByLabelText("Filter by arc character")).not.toBeInTheDocument();
    });

    it("hides the pagination when everything fits on one page", () => {
        // given
        mocks.useBrowseQuotes.mockReturnValue(browseResult([makeQuote()], 20));

        // when
        renderPicker();

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("pages forward through the results", async () => {
        // given
        const user = userEvent.setup();
        mocks.useBrowseQuotes.mockReturnValue(browseResult([makeQuote()], 45));
        renderPicker();

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(lastBrowseCall()[0]).toMatchObject({ offset: 20 });
        expect(screen.getByText("21-40 of 45")).toBeInTheDocument();
    });

    it("returns to the first page when a filter changes", async () => {
        // given
        const user = userEvent.setup();
        mocks.useBrowseQuotes.mockReturnValue(browseResult([makeQuote()], 45));
        renderPicker();
        await user.click(screen.getByRole("button", { name: "Next" }));

        // when
        await user.selectOptions(selectOwning("All Types"), "blue");

        // then
        expect(lastBrowseCall()[0]).toMatchObject({ offset: 0, truth: "blue" });
    });

    it("closes when the modal close control is pressed", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderPicker({ onClose });

        // when
        await user.click(screen.getByRole("button", { name: "✕" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });
});
