import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { OC } from "../../types/api";
import { OCListPage } from "./OCListPage";

const mocks = vi.hoisted(() => ({ useOCList: vi.fn() }));

vi.mock("../../api/queries/oc", () => ({ useOCList: mocks.useOCList }));

function makeOC(overrides: Partial<OC> = {}): OC {
    return {
        id: "oc-1",
        author: { id: "user-1", username: "ronove", display_name: "Ronove" },
        name: "Featherine Junior",
        description: "",
        series: "umineko",
        gallery: [],
        vote_score: 0,
        favourite_count: 0,
        user_favourited: false,
        comment_count: 0,
        is_crack_oc: false,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

interface ListState {
    ocs?: OC[];
    total?: number;
    loading?: boolean;
}

function stubList(state: ListState = {}) {
    mocks.useOCList.mockReturnValue({
        ocs: state.ocs ?? [],
        total: state.total ?? state.ocs?.length ?? 0,
        loading: state.loading ?? false,
    });
}

function renderPage() {
    return renderWithProviders(<OCListPage />, { route: "/oc" });
}

function sortSelect(): HTMLElement {
    return screen.getAllByRole("combobox")[0];
}

function seriesSelect(): HTMLElement {
    return screen.getAllByRole("combobox")[1];
}

function card(): HTMLElement {
    return screen.getByRole("link", { name: /Featherine Junior/ });
}

beforeEach(() => {
    stubList();
});

describe("OCListPage", () => {
    it("says it is loading while the first page of characters is on its way", () => {
        // given
        stubList({ loading: true });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading OCs...")).toBeInTheDocument();
    });

    it("invites the first character when nobody has added one yet", () => {
        // given
        stubList({ ocs: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText("No OCs found. Be the first to add one!")).toBeInTheDocument();
    });

    it("asks for the newest characters across every series by default", () => {
        // given
        stubList();

        // when
        renderPage();

        // then
        expect(mocks.useOCList).toHaveBeenLastCalledWith({
            sort: "new",
            series: undefined,
            custom: undefined,
            limit: 20,
            offset: 0,
        });
    });

    it("links each card through to that character", () => {
        // given
        stubList({ ocs: [makeOC({ id: "oc-9" })] });

        // when
        renderPage();

        // then
        expect(screen.getByRole("link", { name: /Featherine Junior/ })).toHaveAttribute("href", "/oc/oc-9");
    });

    it("labels a canon series by its proper name", () => {
        // given
        stubList({ ocs: [makeOC({ series: "higurashi" })] });

        // when
        renderPage();

        // then
        expect(within(card()).getByText("Higurashi")).toBeInTheDocument();
    });

    it("labels a custom universe by the name its author gave it", () => {
        // given
        stubList({ ocs: [makeOC({ series: "custom", custom_series_name: "Rose Guns Days" })] });

        // when
        renderPage();

        // then
        expect(within(card()).getByText("Rose Guns Days")).toBeInTheDocument();
    });

    it("falls back to Custom when a custom universe has no name", () => {
        // given
        stubList({ ocs: [makeOC({ series: "custom" })] });

        // when
        renderPage();

        // then
        expect(within(card()).getByText("Custom")).toBeInTheDocument();
    });

    it("marks a positive score with a plus sign", () => {
        // given
        stubList({ ocs: [makeOC({ vote_score: 7 })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("+7")).toBeInTheDocument();
    });

    it("leaves a negative score to speak for itself", () => {
        // given
        stubList({ ocs: [makeOC({ vote_score: -2 })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("-2")).toBeInTheDocument();
    });

    it("counts a lone comment in the singular", () => {
        // given
        stubList({ ocs: [makeOC({ comment_count: 1 })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("1 comment")).toBeInTheDocument();
    });

    it("counts several comments in the plural", () => {
        // given
        stubList({ ocs: [makeOC({ comment_count: 4 })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("4 comments")).toBeInTheDocument();
    });

    it("shows how many people have favourited the character", () => {
        // given
        stubList({ ocs: [makeOC({ favourite_count: 12 })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("♥ 12")).toBeInTheDocument();
    });

    it("falls back to a star placeholder when the character has no image", () => {
        // given
        stubList({ ocs: [makeOC()] });

        // when
        renderPage();

        // then
        expect(screen.queryByRole("img", { name: "Featherine Junior" })).not.toBeInTheDocument();
        expect(screen.getByText("★")).toBeInTheDocument();
    });

    it("prefers the thumbnail over the full image on a card", () => {
        // given
        stubList({ ocs: [makeOC({ image_url: "/full.png", thumbnail_url: "/thumb.png" })] });

        // when
        renderPage();

        // then
        expect(screen.getByRole("img", { name: "Featherine Junior" })).toHaveAttribute("src", "/thumb.png");
    });

    it("always offers the new character shortcut", () => {
        // given
        stubList();

        // when
        renderPage();

        // then
        expect(screen.getByRole("link", { name: "+ New OC" })).toHaveAttribute("href", "/oc/new");
    });
});

describe("OCListPage filtering", () => {
    it("re-asks for the characters under the chosen sort", async () => {
        // given
        const user = userEvent.setup();
        renderPage();

        // when
        await user.selectOptions(sortSelect(), "favourites");

        // then
        expect(mocks.useOCList).toHaveBeenLastCalledWith(expect.objectContaining({ sort: "favourites", offset: 0 }));
    });

    it("narrows the characters to a single series", async () => {
        // given
        const user = userEvent.setup();
        renderPage();

        // when
        await user.selectOptions(seriesSelect(), "ciconia");

        // then
        expect(mocks.useOCList).toHaveBeenLastCalledWith(expect.objectContaining({ series: "ciconia" }));
    });

    it("keeps the custom universe box hidden until custom is chosen", () => {
        // given
        stubList();

        // when
        renderPage();

        // then
        expect(screen.queryByPlaceholderText("Custom series name (optional)")).not.toBeInTheDocument();
    });

    it("offers a custom universe box once custom is chosen", async () => {
        // given
        const user = userEvent.setup();
        renderPage();

        // when
        await user.selectOptions(seriesSelect(), "custom");

        // then
        expect(screen.getByPlaceholderText("Custom series name (optional)")).toBeInTheDocument();
    });

    it("passes the typed universe name alongside the custom series", async () => {
        // given
        const user = userEvent.setup();
        renderPage();
        await user.selectOptions(seriesSelect(), "custom");

        // when
        await user.type(screen.getByPlaceholderText("Custom series name (optional)"), "Higanbana");

        // then
        expect(mocks.useOCList).toHaveBeenLastCalledWith(
            expect.objectContaining({ series: "custom", custom: "Higanbana" }),
        );
    });

    it("sends no universe name while the custom box is still empty", async () => {
        // given
        const user = userEvent.setup();
        renderPage();

        // when
        await user.selectOptions(seriesSelect(), "custom");

        // then
        expect(mocks.useOCList).toHaveBeenLastCalledWith(
            expect.objectContaining({ series: "custom", custom: undefined }),
        );
    });

    it("forgets the typed universe name when the series changes again", async () => {
        // given
        const user = userEvent.setup();
        renderPage();
        await user.selectOptions(seriesSelect(), "custom");
        await user.type(screen.getByPlaceholderText("Custom series name (optional)"), "Higanbana");

        // when
        await user.selectOptions(seriesSelect(), "umineko");

        // then
        expect(mocks.useOCList).toHaveBeenLastCalledWith(
            expect.objectContaining({ series: "umineko", custom: undefined }),
        );
        expect(screen.queryByPlaceholderText("Custom series name (optional)")).not.toBeInTheDocument();
    });

    it("returns to the first page whenever a filter changes", async () => {
        // given
        stubList({ ocs: [makeOC()], total: 60 });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Next" }));
        expect(mocks.useOCList).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 20 }));

        // when
        await user.selectOptions(sortSelect(), "name");

        // then
        expect(mocks.useOCList).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 0 }));
    });
});

describe("OCListPage paging", () => {
    it("hides the pager when there is nothing to page through", () => {
        // given
        stubList({ ocs: [], total: 0 });

        // when
        renderPage();

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("walks forward twenty characters at a time", async () => {
        // given
        stubList({ ocs: [makeOC()], total: 45 });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(mocks.useOCList).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 20 }));
    });

    it("refuses to walk back from the first page", () => {
        // given
        stubList({ ocs: [makeOC()], total: 45 });

        // when
        renderPage();

        // then
        expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    });

    it("walks back to the previous page of characters", async () => {
        // given
        stubList({ ocs: [makeOC()], total: 45 });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Next" }));

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(mocks.useOCList).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 0 }));
    });
});
