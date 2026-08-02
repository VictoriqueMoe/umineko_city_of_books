import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CharacterGroups } from "../../api/endpoints";
import type { Quote, QuoteBrowseResponse } from "../../types/api";
import { renderWithProviders } from "../../test-utils/render";
import { QuoteBrowserPage } from "./QuoteBrowserPage";

const mocks = vi.hoisted(() => ({
    useBrowseQuotes: vi.fn(),
    useCharacterGroups: vi.fn(),
}));

vi.mock("../../api/queries/quote", () => ({ useBrowseQuotes: mocks.useBrowseQuotes }));

vi.mock("../../api/queries/characters", () => ({ useCharacterGroups: mocks.useCharacterGroups }));

vi.mock("../../components/truth/TruthCard/TruthCard", () => ({
    TruthCard: ({ quote, lang }: { quote: Quote; lang?: string }) => (
        <div data-testid="truth-card" data-lang={lang ?? ""}>
            {quote.text}
        </div>
    ),
}));

function makeQuote(overrides: Partial<Quote> = {}): Quote {
    return {
        text: "Without love, it cannot be seen",
        textHtml: "<p>Without love, it cannot be seen</p>",
        characterId: "beatrice",
        character: "Beatrice",
        audioId: "audio-1",
        episode: 1,
        contentType: "dialogue",
        hasRedTruth: false,
        hasBlueTruth: false,
        hasGoldTruth: false,
        hasPurpleTruth: false,
        index: 0,
        ...overrides,
    };
}

function makeResponse(overrides: Partial<QuoteBrowseResponse> = {}): QuoteBrowseResponse {
    return {
        character: "",
        characterId: "",
        quotes: [makeQuote()],
        total: 1,
        limit: 30,
        offset: 0,
        ...overrides,
    };
}

interface SetupOptions {
    data?: QuoteBrowseResponse | null;
    loading?: boolean;
    characters?: CharacterGroups;
}

function setup(options: SetupOptions = {}) {
    mocks.useBrowseQuotes.mockReturnValue({
        data: options.data === undefined ? makeResponse() : options.data,
        loading: options.loading ?? false,
    });
    mocks.useCharacterGroups.mockReturnValue({
        groups: options.characters ?? { main: {}, additional: {} },
        loading: false,
    });
    const user = userEvent.setup();
    const result = renderWithProviders(<QuoteBrowserPage />);

    return { user, ...result };
}

function selectHolding(optionLabel: string): HTMLSelectElement {
    const select = screen.getByRole("option", { name: optionLabel }).closest("select");
    if (!select) {
        throw new Error(`no select holds the option ${optionLabel}`);
    }

    return select as HTMLSelectElement;
}

function lastParams(): Record<string, unknown> {
    const call = mocks.useBrowseQuotes.mock.lastCall;
    if (!call) {
        throw new Error("the quote browser never asked for quotes");
    }

    return call[0] as Record<string, unknown>;
}

beforeEach(() => {
    mocks.useBrowseQuotes.mockReturnValue({ data: makeResponse(), loading: false });
    mocks.useCharacterGroups.mockReturnValue({ groups: { main: {}, additional: {} }, loading: false });
});

describe("QuoteBrowserPage default browse", () => {
    it("browses umineko with no filters to begin with", () => {
        // given
        const options = {};

        // when
        setup(options);

        // then
        expect(lastParams()).toEqual({
            episode: undefined,
            character: undefined,
            truth: undefined,
            arc: undefined,
            chapter: undefined,
            lang: undefined,
            limit: 30,
            offset: 0,
            series: "umineko",
        });
    });

    it("asks the character list for the series being browsed", () => {
        // given
        const options = {};

        // when
        setup(options);

        // then
        expect(mocks.useCharacterGroups).toHaveBeenLastCalledWith("umineko");
    });
});

describe("QuoteBrowserPage truth filters", () => {
    it("narrows the browse to the chosen truth colour", async () => {
        // given
        const { user } = setup();

        // when
        await user.click(screen.getByRole("button", { name: "Red Truth" }));

        // then
        expect(lastParams().truth).toBe("red");
    });

    it("clears the truth filter when the same colour is pressed again", async () => {
        // given
        const { user } = setup();
        await user.click(screen.getByRole("button", { name: "Blue Truth" }));

        // when
        await user.click(screen.getByRole("button", { name: "Blue Truth" }));

        // then
        expect(lastParams().truth).toBeUndefined();
    });

    it("clears the truth filter from the all button", async () => {
        // given
        const { user } = setup();
        await user.click(screen.getByRole("button", { name: "Gold Truth" }));

        // when
        await user.click(screen.getByRole("button", { name: "All" }));

        // then
        expect(lastParams().truth).toBeUndefined();
    });

    it("offers truth colours only while umineko is being browsed", async () => {
        // given
        const { user } = setup();

        // when
        await user.click(screen.getByRole("button", { name: "Higurashi" }));

        // then
        expect(screen.queryByRole("button", { name: "Purple Truth" })).not.toBeInTheDocument();
    });
});

describe("QuoteBrowserPage series", () => {
    it("browses umineko by episode", async () => {
        // given
        const { user } = setup();

        // when
        await user.selectOptions(selectHolding("All Episodes"), "3");

        // then
        expect(lastParams().episode).toBe(3);
    });

    it("browses higurashi by arc", async () => {
        // given
        const { user } = setup();
        await user.click(screen.getByRole("button", { name: "Higurashi" }));

        // when
        await user.selectOptions(selectHolding("All Arcs"), "onikakushi");

        // then
        expect(lastParams()).toMatchObject({ series: "higurashi", arc: "onikakushi" });
    });

    it("browses ciconia by chapter", async () => {
        // given
        const { user } = setup();
        await user.click(screen.getByRole("button", { name: "Ciconia" }));

        // when
        await user.selectOptions(selectHolding("All Chapters"), "00");

        // then
        expect(lastParams()).toMatchObject({ series: "ciconia", chapter: "00" });
    });

    it("forgets the umineko filters when another series is chosen", async () => {
        // given
        const { user } = setup();
        await user.click(screen.getByRole("button", { name: "Red Truth" }));
        await user.selectOptions(selectHolding("All Episodes"), "5");

        // when
        await user.click(screen.getByRole("button", { name: "Higurashi" }));

        // then
        expect(lastParams()).toEqual({
            episode: undefined,
            character: undefined,
            truth: undefined,
            arc: undefined,
            chapter: undefined,
            lang: undefined,
            limit: 30,
            offset: 0,
            series: "higurashi",
        });
    });

    it("offers only the languages the series has", async () => {
        // given
        const { user } = setup();
        expect(screen.getByRole("option", { name: "Witch Hunt" })).toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "Higurashi" }));

        // then
        expect(screen.queryByRole("option", { name: "Witch Hunt" })).not.toBeInTheDocument();
        expect(screen.getByRole("option", { name: "Japanese" })).toBeInTheDocument();
    });

    it("browses in the chosen language", async () => {
        // given
        const { user } = setup();

        // when
        await user.selectOptions(selectHolding("Default Language"), "ja");

        // then
        expect(lastParams().lang).toBe("ja");
    });
});

describe("QuoteBrowserPage characters", () => {
    it("lists the main cast alphabetically when there is nobody else", () => {
        // given
        const characters = { main: { battler: "Battler", beatrice: "Beatrice", ange: "Ange" }, additional: {} };

        // when
        const { container } = setup({ characters });

        // then
        const options = Array.from(selectHolding("All Characters").options).map(o => o.textContent);
        expect(options).toEqual(["All Characters", "Ange", "Battler", "Beatrice"]);
        expect(container.querySelector("optgroup")).toBeNull();
    });

    it("splits the cast into main and additional when both exist", () => {
        // given
        const characters = { main: { beatrice: "Beatrice" }, additional: { kanon: "Kanon" } };

        // when
        setup({ characters });

        // then
        expect(screen.getByRole("group", { name: "Main cast" })).toBeInTheDocument();
        expect(screen.getByRole("group", { name: "Additional" })).toBeInTheDocument();
    });

    it("narrows the browse to the chosen character", async () => {
        // given
        const characters = { main: { beatrice: "Beatrice" }, additional: {} };
        const { user } = setup({ characters });

        // when
        await user.selectOptions(selectHolding("All Characters"), "beatrice");

        // then
        expect(lastParams().character).toBe("beatrice");
    });
});

describe("QuoteBrowserPage results", () => {
    it("consults the game board while the quotes load", () => {
        // given
        const loading = true;

        // when
        setup({ loading, data: null });

        // then
        expect(screen.getByText("Consulting the game board...")).toBeInTheDocument();
        expect(screen.queryByTestId("truth-card")).not.toBeInTheDocument();
    });

    it("says when the filters match nothing", () => {
        // given
        const data = makeResponse({ quotes: [], total: 0 });

        // when
        setup({ data });

        // then
        expect(screen.getByText("No quotes found.")).toBeInTheDocument();
    });

    it("shows a card for every quote", () => {
        // given
        const data = makeResponse({
            quotes: [makeQuote(), makeQuote({ audioId: "audio-2", text: "The golden truth" })],
            total: 2,
        });

        // when
        setup({ data });

        // then
        expect(screen.getAllByTestId("truth-card")).toHaveLength(2);
    });

    it("passes the chosen language down to each card", async () => {
        // given
        const { user } = setup();

        // when
        await user.selectOptions(selectHolding("Default Language"), "ja");

        // then
        expect(screen.getByTestId("truth-card")).toHaveAttribute("data-lang", "ja");
    });

    it("walks forward through the quotes", async () => {
        // given
        const { user } = setup({ data: makeResponse({ total: 90 }) });

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(lastParams().offset).toBe(30);
    });

    it("walks back through the quotes", async () => {
        // given
        const { user } = setup({ data: makeResponse({ total: 90 }) });
        await user.click(screen.getByRole("button", { name: "Next" }));

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(lastParams().offset).toBe(0);
    });

    it("hides the pager while the quotes are still loading", () => {
        // given
        const loading = true;

        // when
        setup({ loading, data: makeResponse({ total: 90 }) });

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });
});
