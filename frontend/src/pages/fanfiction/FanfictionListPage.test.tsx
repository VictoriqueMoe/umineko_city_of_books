import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Link } from "react-router";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { Fanfic, UserProfile } from "../../types/api";
import { FanfictionListPage } from "./FanfictionListPage";

const { useFanficList, useFanficSeries, useFanficLanguages, useCharactersFlat, useOCCharacters, navigate } = vi.hoisted(
    () => ({
        useFanficList: vi.fn(),
        useFanficSeries: vi.fn(),
        useFanficLanguages: vi.fn(),
        useCharactersFlat: vi.fn(),
        useOCCharacters: vi.fn(),
        navigate: vi.fn(),
    }),
);

vi.mock("../../api/queries/fanfic", () => ({ useFanficList, useFanficSeries, useFanficLanguages }));
vi.mock("../../api/queries/characters", () => ({ useCharactersFlat, useOCCharacters }));
vi.mock("../../components/RulesBox/RulesBox", () => ({
    RulesBox: (props: { page: string }) => <div>{`rules for ${props.page}`}</div>,
}));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

const author = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });
const reader = makeUser({ id: "reader-1", username: "battler", display_name: "Battler" });

function makeFanfic(overrides: Partial<Fanfic> = {}): Fanfic {
    return {
        id: "fanfic-1",
        author,
        title: "Golden Land",
        summary: "",
        series: "Umineko",
        rating: "T",
        language: "English",
        status: "in_progress",
        is_oneshot: false,
        contains_lemons: false,
        genres: [],
        tags: [],
        characters: [],
        is_pairing: false,
        word_count: 1200,
        chapter_count: 3,
        favourite_count: 4,
        view_count: 90,
        comment_count: 0,
        user_favourited: false,
        published_at: "2026-01-01T00:00:00Z",
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

interface StubOptions {
    fanfics?: Fanfic[];
    total?: number;
    loading?: boolean;
    series?: string[];
    languages?: string[];
    ocCharacters?: string[];
}

function stubList(options: StubOptions = {}) {
    useFanficList.mockReturnValue({
        fanfics: options.fanfics ?? [],
        total: options.total ?? options.fanfics?.length ?? 0,
        loading: options.loading ?? false,
    });
    useFanficSeries.mockReturnValue({ series: options.series ?? ["Umineko", "Higurashi"] });
    useFanficLanguages.mockReturnValue({ languages: options.languages ?? ["English", "Japanese"] });
    useCharactersFlat.mockImplementation((series: string) =>
        series === "umineko"
            ? { characters: { "1": "Beatrice", "2": "Battler" } }
            : { characters: { "3": "Rena", "4": "Keiichi" } },
    );
    useOCCharacters.mockReturnValue({ characters: options.ocCharacters ?? [] });
}

function renderPage(user: UserProfile | null, route = "/fanfiction") {
    return renderWithProviders(<FanfictionListPage />, { user, route });
}

async function openFilters(user: ReturnType<typeof userEvent.setup>) {
    await user.click(screen.getByRole("button", { name: /Filters/ }));
}

describe("FanfictionListPage", () => {
    it("welcomes the reader to the archive", () => {
        // given
        stubList();

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Welcome to the Archive")).toBeInTheDocument();
        expect(screen.getByText("rules for fanfiction")).toBeInTheDocument();
    });

    it("hides the new fanfic button from a signed out visitor", () => {
        // given
        stubList();

        // when
        renderPage(null);

        // then
        expect(screen.queryByRole("button", { name: "+ New Fanfic" })).not.toBeInTheDocument();
    });

    it("sends a signed in member to the fanfic editor", async () => {
        // given
        stubList();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "+ New Fanfic" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/new");
    });

    it("asks for the recently updated stories by default", () => {
        // given
        stubList();

        // when
        renderPage(null);

        // then
        expect(useFanficList).toHaveBeenLastCalledWith(
            expect.objectContaining({ sort: "updated", limit: 25, offset: 0 }),
        );
    });

    it("waits while the archive is loading", () => {
        // given
        stubList({ loading: true, fanfics: [makeFanfic()] });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Loading fanfiction...")).toBeInTheDocument();
        expect(screen.queryByText("Golden Land")).not.toBeInTheDocument();
    });

    it("says nothing matched when the filters are too narrow", () => {
        // given
        stubList({ fanfics: [] });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("No fanfics found matching your filters.")).toBeInTheDocument();
    });

    it("links every story in the list to its detail page", () => {
        // given
        stubList({
            fanfics: [
                makeFanfic({ id: "fanfic-1", title: "Golden Land" }),
                makeFanfic({ id: "fanfic-2", title: "Hinamizawa Nights" }),
            ],
        });

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("link", { name: /Golden Land/ })).toHaveAttribute("href", "/fanfiction/fanfic-1");
        expect(screen.getByRole("link", { name: /Hinamizawa Nights/ })).toHaveAttribute("href", "/fanfiction/fanfic-2");
    });

    it("abbreviates a word count of a thousand or more", () => {
        // given
        stubList({ fanfics: [makeFanfic({ word_count: 12345 })] });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("12.3k words")).toBeInTheDocument();
    });

    it("prints a small word count in full", () => {
        // given
        stubList({ fanfics: [makeFanfic({ word_count: 940 })] });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("940 words")).toBeInTheDocument();
    });

    it("uses singular wording for a lone chapter and a lone favourite", () => {
        // given
        stubList({ fanfics: [makeFanfic({ chapter_count: 1, favourite_count: 1 })] });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("1 chapter")).toBeInTheDocument();
        expect(screen.getByText("1 fav")).toBeInTheDocument();
    });

    it("badges a completed story", () => {
        // given
        stubList({ fanfics: [makeFanfic({ status: "complete" })] });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Complete")).toBeInTheDocument();
    });

    it("badges an unpublished draft", () => {
        // given
        stubList({ fanfics: [makeFanfic({ status: "draft" })] });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Draft")).toBeInTheDocument();
    });

    it("shows the genres, tags, characters and content warnings of a story", () => {
        // given
        stubList({
            fanfics: [
                makeFanfic({
                    genres: ["Mystery"],
                    tags: ["closed room"],
                    characters: [{ series: "umineko", character_name: "Kanon", sort_order: 0 }],
                    is_pairing: true,
                    contains_lemons: true,
                }),
            ],
        });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("Mystery")).toBeInTheDocument();
        expect(screen.getByText("closed room")).toBeInTheDocument();
        expect(screen.getByText("Kanon")).toBeInTheDocument();
        expect(screen.getByText("Pairing")).toBeInTheDocument();
        expect(screen.getByText("Lemons")).toBeInTheDocument();
    });

    it("falls back to the first letter of the title when a story has no cover", () => {
        // given
        stubList({ fanfics: [makeFanfic({ title: "Golden Land", cover_image_url: undefined })] });

        // when
        renderPage(null);

        // then
        expect(screen.getByText("G")).toBeInTheDocument();
    });

    it("prefers the thumbnail over the full cover in the list", () => {
        // given
        stubList({
            fanfics: [
                makeFanfic({
                    cover_image_url: "https://cdn.test/full.png",
                    cover_thumbnail_url: "https://cdn.test/thumb.png",
                }),
            ],
        });

        // when
        renderPage(null);

        // then
        expect(document.querySelector("img")).toHaveAttribute("src", "https://cdn.test/thumb.png");
    });

    it("reads every filter out of the url and forwards it to the archive", () => {
        // given
        stubList();

        // when
        renderPage(
            null,
            "/fanfiction?sort=favourites&series=Umineko&rating=M&status=complete&language=Japanese&genre_a=Mystery&genre_b=Horror&tag=closed+room&char_a=Beatrice&char_b=Battler&char_c=Rena&char_d=Keiichi&pairing=true&lemons=true&search=golden&offset=50",
        );

        // then
        expect(useFanficList).toHaveBeenLastCalledWith({
            sort: "favourites",
            series: "Umineko",
            rating: "M",
            status: "complete",
            language: "Japanese",
            genre_a: "Mystery",
            genre_b: "Horror",
            tag: "closed room",
            char_a: "Beatrice",
            char_b: "Battler",
            char_c: "Rena",
            char_d: "Keiichi",
            pairing: true,
            lemons: true,
            search: "golden",
            limit: 25,
            offset: 50,
        });
    });

    it("counts how many filters are narrowing the archive", () => {
        // given
        stubList();

        // when
        renderPage(null, "/fanfiction?series=Umineko&rating=M&pairing=true&lemons=true");

        // then
        expect(screen.getByRole("button", { name: /Filters/ })).toHaveTextContent("Filters4");
    });

    it("keeps the filter panel closed until it is asked for", async () => {
        // given
        stubList();
        const user = userEvent.setup();
        renderPage(null);

        // when
        expect(screen.queryByLabelText("Rating")).not.toBeInTheDocument();
        await openFilters(user);

        // then
        expect(screen.getByLabelText("Rating")).toBeInTheDocument();
        expect(screen.getByLabelText("Series")).toBeInTheDocument();
    });

    it("offers the series and languages the archive actually holds", async () => {
        // given
        stubList({ series: ["Umineko", "Ciconia"], languages: ["English", "Spanish"] });
        const user = userEvent.setup();
        renderPage(null);

        // when
        await openFilters(user);

        // then
        expect(screen.getByRole("option", { name: "Ciconia" })).toBeInTheDocument();
        expect(screen.getByRole("option", { name: "Spanish" })).toBeInTheDocument();
    });

    it("groups canon and original characters in the character filters", async () => {
        // given
        stubList({ ocCharacters: ["Featherine's Understudy"] });
        const user = userEvent.setup();
        renderPage(null);

        // when
        await openFilters(user);

        // then
        expect(screen.getAllByRole("option", { name: "Beatrice" })).toHaveLength(4);
        expect(screen.getAllByRole("option", { name: "Rena" })).toHaveLength(4);
        expect(screen.getAllByRole("option", { name: "Featherine's Understudy (OC)" })).toHaveLength(4);
    });

    it("re-queries with the sort the reader picked", async () => {
        // given
        stubList();
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.selectOptions(screen.getByLabelText("Sort"), "favourites");

        // then
        expect(useFanficList).toHaveBeenLastCalledWith(expect.objectContaining({ sort: "favourites" }));
    });

    it("narrows the archive by rating", async () => {
        // given
        stubList();
        const user = userEvent.setup();
        renderPage(null);
        await openFilters(user);

        // when
        await user.selectOptions(screen.getByLabelText("Rating"), "M");

        // then
        expect(useFanficList).toHaveBeenLastCalledWith(expect.objectContaining({ rating: "M" }));
    });

    it("goes back to the first page whenever a filter changes", async () => {
        // given
        stubList({ total: 200 });
        const user = userEvent.setup();
        renderPage(null, "/fanfiction?offset=75");
        await openFilters(user);

        // when
        await user.selectOptions(screen.getByLabelText("Status"), "complete");

        // then
        expect(useFanficList).toHaveBeenLastCalledWith(expect.objectContaining({ status: "complete", offset: 0 }));
    });

    it("only searches once the reader submits the search box", async () => {
        // given
        stubList();
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.type(screen.getByPlaceholderText("Search by title or summary..."), "golden");

        // then
        expect(useFanficList).toHaveBeenLastCalledWith(expect.objectContaining({ search: undefined }));
        await user.type(screen.getByPlaceholderText("Search by title or summary..."), "{Enter}");
        expect(useFanficList).toHaveBeenLastCalledWith(expect.objectContaining({ search: "golden" }));
    });

    it("follows the address bar when the search term changes underneath it", async () => {
        // given
        stubList();
        const user = userEvent.setup();
        renderWithProviders(
            <>
                <FanfictionListPage />
                <Link to="/fanfiction?search=witch">jump to the witch search</Link>
            </>,
            { user: null, route: "/fanfiction?search=golden" },
        );
        expect(screen.getByPlaceholderText("Search by title or summary...")).toHaveValue("golden");

        // when
        await user.click(screen.getByRole("link", { name: "jump to the witch search" }));

        // then
        expect(screen.getByPlaceholderText("Search by title or summary...")).toHaveValue("witch");
        expect(useFanficList).toHaveBeenLastCalledWith(expect.objectContaining({ search: "witch" }));
    });

    it("turns the lemons filter on and off again", async () => {
        // given
        stubList();
        const user = userEvent.setup();
        renderPage(null);
        await openFilters(user);

        // when
        await user.click(screen.getByRole("switch", { name: "Show lemons" }));

        // then
        expect(useFanficList).toHaveBeenLastCalledWith(expect.objectContaining({ lemons: true }));
        await user.click(screen.getByRole("switch", { name: "Show lemons" }));
        expect(useFanficList).toHaveBeenLastCalledWith(expect.objectContaining({ lemons: undefined }));
    });

    it("pages forward through the archive", async () => {
        // given
        stubList({ fanfics: [makeFanfic()], total: 80 });
        const user = userEvent.setup();
        renderPage(null);

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(useFanficList).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 25 }));
        expect(screen.getByText("26-50 of 80")).toBeInTheDocument();
    });

    it("pages back to the start of the archive", async () => {
        // given
        stubList({ fanfics: [makeFanfic()], total: 80 });
        const user = userEvent.setup();
        renderPage(null, "/fanfiction?offset=25");

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(useFanficList).toHaveBeenLastCalledWith(expect.objectContaining({ offset: 0 }));
        expect(screen.getByText("1-25 of 80")).toBeInTheDocument();
    });

    it("cannot page past either end of the archive", () => {
        // given
        stubList({ fanfics: [makeFanfic()], total: 10 });

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
    });
});
