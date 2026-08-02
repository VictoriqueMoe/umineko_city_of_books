import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { Ship, ShipCharacter } from "../../types/api";
import { CharacterPills, ShipsListPage } from "./ShipsListPage";

const mocks = vi.hoisted(() => ({
    useShipList: vi.fn(),
    useRules: vi.fn(),
}));

vi.mock("../../api/queries/ship", () => ({ useShipList: mocks.useShipList }));
vi.mock("../../api/queries/misc", () => ({ useRules: mocks.useRules }));

function makeCharacter(overrides: Partial<ShipCharacter> = {}): ShipCharacter {
    return {
        series: "umineko",
        character_id: "battler",
        character_name: "Battler",
        sort_order: 0,
        ...overrides,
    };
}

function makeShip(overrides: Partial<Ship> = {}): Ship {
    return {
        id: "ship-1",
        author: { id: "user-1", username: "beatrice", display_name: "Beatrice" },
        title: "Battler and Beatrice",
        description: "",
        characters: [
            makeCharacter(),
            makeCharacter({ character_id: "beatrice", character_name: "Beatrice", sort_order: 1 }),
        ],
        vote_score: 0,
        comment_count: 0,
        is_crackship: false,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

interface ListState {
    ships?: Ship[];
    total?: number;
    loading?: boolean;
}

function stubList(state: ListState = {}) {
    mocks.useShipList.mockReturnValue({
        ships: state.ships ?? [],
        total: state.total ?? state.ships?.length ?? 0,
        loading: state.loading ?? false,
    });
}

function renderPage(user = makeUser({ id: "me" })) {
    return renderWithProviders(<ShipsListPage />, { user, route: "/ships" });
}

function sortSelect(): HTMLElement {
    return screen.getAllByRole("combobox")[0];
}

function seriesSelect(): HTMLElement {
    return screen.getAllByRole("combobox")[1];
}

beforeEach(() => {
    mocks.useRules.mockReturnValue({ rules: "", loading: false });
    stubList();
});

describe("ShipsListPage", () => {
    it("says it is loading while the first page of ships is on its way", () => {
        // given
        stubList({ loading: true });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading ships...")).toBeInTheDocument();
    });

    it("invites the first pairing when nothing has been declared yet", () => {
        // given
        stubList({ ships: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText("No ships found. Be the first to declare a pairing!")).toBeInTheDocument();
    });

    it("asks for the newest ships on the first page by default", () => {
        // given
        stubList();

        // when
        renderPage();

        // then
        expect(mocks.useShipList).toHaveBeenLastCalledWith({
            sort: "new",
            series: undefined,
            crackships: false,
            limit: 20,
            offset: 0,
        });
    });

    it("links each card through to that ship", () => {
        // given
        stubList({ ships: [makeShip({ id: "ship-9" })] });

        // when
        renderPage();

        // then
        expect(screen.getByRole("link", { name: /Battler and Beatrice/ })).toHaveAttribute("href", "/ships/ship-9");
    });

    it("marks a positive score with a plus sign", () => {
        // given
        stubList({ ships: [makeShip({ vote_score: 12 })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("+12")).toBeInTheDocument();
    });

    it("leaves a negative score to speak for itself", () => {
        // given
        stubList({ ships: [makeShip({ vote_score: -4 })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("-4")).toBeInTheDocument();
    });

    it("counts a lone comment in the singular", () => {
        // given
        stubList({ ships: [makeShip({ comment_count: 1 })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("1 comment")).toBeInTheDocument();
    });

    it("counts several comments in the plural", () => {
        // given
        stubList({ ships: [makeShip({ comment_count: 3 })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("3 comments")).toBeInTheDocument();
    });

    it("brands a low scoring pairing as a crackship", () => {
        // given
        stubList({ ships: [makeShip({ is_crackship: true })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("Crackship")).toBeInTheDocument();
    });

    it("falls back to a heart placeholder when the ship has no image", () => {
        // given
        stubList({ ships: [makeShip()] });

        // when
        renderPage();

        // then
        expect(screen.queryByRole("img", { name: "Battler and Beatrice" })).not.toBeInTheDocument();
        expect(screen.getByText("♥")).toBeInTheDocument();
    });

    it("prefers the thumbnail over the full image on a card", () => {
        // given
        stubList({ ships: [makeShip({ image_url: "/full.png", thumbnail_url: "/thumb.png" })] });

        // when
        renderPage();

        // then
        expect(screen.getByRole("img", { name: "Battler and Beatrice" })).toHaveAttribute("src", "/thumb.png");
    });

    it("offers the new ship shortcut to a signed in member", () => {
        // given
        stubList();

        // when
        renderPage();

        // then
        expect(screen.getByRole("link", { name: "+ New Ship" })).toHaveAttribute("href", "/ships/new");
    });

    it("hides the new ship shortcut from a signed out visitor", () => {
        // given
        stubList();

        // when
        renderWithProviders(<ShipsListPage />, { user: null, route: "/ships" });

        // then
        expect(screen.queryByRole("link", { name: "+ New Ship" })).not.toBeInTheDocument();
    });
});

describe("ShipsListPage filtering", () => {
    it("re-asks for the ships under the chosen sort", async () => {
        // given
        const user = userEvent.setup();
        renderPage();

        // when
        await user.selectOptions(sortSelect(), "crackship");

        // then
        expect(mocks.useShipList).toHaveBeenLastCalledWith(expect.objectContaining({ sort: "crackship", offset: 0 }));
    });

    it("narrows the ships to a single series", async () => {
        // given
        const user = userEvent.setup();
        renderPage();

        // when
        await user.selectOptions(seriesSelect(), "higurashi");

        // then
        expect(mocks.useShipList).toHaveBeenLastCalledWith(expect.objectContaining({ series: "higurashi" }));
    });

    it("drops the series filter again when all series are chosen", async () => {
        // given
        const user = userEvent.setup();
        renderPage();
        await user.selectOptions(seriesSelect(), "oc");

        // when
        await user.selectOptions(seriesSelect(), "");

        // then
        expect(mocks.useShipList).toHaveBeenLastCalledWith(expect.objectContaining({ series: undefined }));
    });

    it("limits the list to crackships when the toggle is switched on", async () => {
        // given
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("switch", { name: "Crackships only" }));

        // then
        expect(mocks.useShipList).toHaveBeenLastCalledWith(expect.objectContaining({ crackships: true }));
    });

    it("returns to the first page whenever a filter changes", async () => {
        // given
        stubList({ ships: [makeShip()], total: 60 });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Next" }));
        expect(mocks.useShipList).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 20 }));

        // when
        await user.selectOptions(sortSelect(), "top");

        // then
        expect(mocks.useShipList).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 0 }));
    });
});

describe("ShipsListPage paging", () => {
    it("hides the pager when there is nothing to page through", () => {
        // given
        stubList({ ships: [], total: 0 });

        // when
        renderPage();

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("walks forward twenty ships at a time", async () => {
        // given
        stubList({ ships: [makeShip()], total: 45 });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(mocks.useShipList).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 20 }));
    });

    it("refuses to walk back from the first page", () => {
        // given
        stubList({ ships: [makeShip()], total: 45 });

        // when
        renderPage();

        // then
        expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    });

    it("walks back to the previous page of ships", async () => {
        // given
        stubList({ ships: [makeShip()], total: 45 });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Next" }));

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(mocks.useShipList).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 0 }));
    });
});

describe("CharacterPills", () => {
    it("orders the characters by their declared position", () => {
        // given
        const characters = [
            makeCharacter({ character_name: "Beatrice", sort_order: 1 }),
            makeCharacter({ character_name: "Battler", sort_order: 0 }),
        ];

        // when
        const { container } = renderWithProviders(<CharacterPills characters={characters} />);

        // then
        expect(container.textContent).toBe("Battler×Beatrice");
    });

    it("puts a cross between every pair but never before the first", () => {
        // given
        const characters = [
            makeCharacter({ character_name: "Battler", sort_order: 0 }),
            makeCharacter({ character_name: "Beatrice", sort_order: 1 }),
            makeCharacter({ character_name: "Kanon", sort_order: 2 }),
        ];

        // when
        const { container } = renderWithProviders(<CharacterPills characters={characters} />);

        // then
        expect(container.querySelectorAll("span").length).toBe(5);
        expect(container.textContent).toBe("Battler×Beatrice×Kanon");
    });

    it("leaves the caller's array untouched while sorting", () => {
        // given
        const characters = [
            makeCharacter({ character_name: "Beatrice", sort_order: 1 }),
            makeCharacter({ character_name: "Battler", sort_order: 0 }),
        ];

        // when
        renderWithProviders(<CharacterPills characters={characters} />);

        // then
        expect(characters[0].character_name).toBe("Beatrice");
    });

    it("renders an original character with no character id", () => {
        // given
        const characters = [
            makeCharacter({ series: "oc", character_id: undefined, character_name: "Nanjo Junior", sort_order: 0 }),
        ];

        // when
        renderWithProviders(<CharacterPills characters={characters} />);

        // then
        expect(screen.getByText("Nanjo Junior")).toBeInTheDocument();
    });
});
