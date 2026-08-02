import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../../../api/client";
import type { GiphyFavourite, GiphyGif } from "../../../api/endpoints";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import { GifPicker } from "./GifPicker";

const mocks = vi.hoisted(() => ({
    useGiphySearch: vi.fn(),
    useGiphyTrending: vi.fn(),
    searchRefresh: vi.fn(),
    trendingRefresh: vi.fn(),
}));

vi.mock("../../../api/queries/giphy", () => ({
    useGiphySearch: mocks.useGiphySearch,
    useGiphyTrending: mocks.useGiphyTrending,
}));

interface QueryResult {
    data?: { data: GiphyGif[] };
    loading?: boolean;
    error?: unknown;
}

function makeGif(overrides: Partial<GiphyGif> = {}): GiphyGif {
    return {
        id: "gif-1",
        title: "Beato laughing",
        url: "https://giphy.test/gif-1",
        images: {
            fixed_height: { url: "https://giphy.test/gif-1-full.gif", width: "320", height: "240" },
            fixed_width_small: { url: "https://giphy.test/gif-1-small.gif", width: "100", height: "80" },
        },
        ...overrides,
    };
}

function makeFavourite(overrides: Partial<GiphyFavourite> = {}): GiphyFavourite {
    return {
        giphy_id: "fav-1",
        url: "https://giphy.test/fav-1.gif",
        title: "Bern smirking",
        preview_url: "https://giphy.test/fav-1-small.gif",
        width: 200,
        height: 150,
        ...overrides,
    };
}

function setTrending(result: QueryResult) {
    mocks.useGiphyTrending.mockReturnValue({
        data: result.data,
        loading: result.loading ?? false,
        error: result.error ?? null,
        refresh: mocks.trendingRefresh,
    });
}

function noop() {}

beforeEach(() => {
    setTrending({});
    mocks.useGiphySearch.mockReturnValue({
        data: undefined,
        loading: false,
        error: null,
        refresh: mocks.searchRefresh,
    });
});

describe("GifPicker", () => {
    it("hides the tabs and the stars from a signed out visitor", () => {
        // given
        setTrending({ data: { data: [makeGif()] } });

        // when
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });

        // then
        expect(screen.queryByRole("button", { name: "Trending" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /favourites/i })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Beato laughing" })).toBeInTheDocument();
    });

    it("offers the favourites tab to a signed in user", () => {
        // given
        setTrending({ data: { data: [makeGif()] } });

        // when
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: makeUser() });

        // then
        expect(screen.getByRole("button", { name: "Trending" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Favourites/ })).toBeInTheDocument();
    });

    it("skips a gif that has no usable rendition", () => {
        // given
        setTrending({ data: { data: [makeGif(), makeGif({ id: "gif-2", title: "Broken", images: {} })] } });

        // when
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });

        // then
        expect(screen.getByRole("button", { name: "Beato laughing" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Broken" })).not.toBeInTheDocument();
    });

    it("names an untitled gif so it can still be identified", () => {
        // given
        setTrending({ data: { data: [makeGif({ title: "" })] } });

        // when
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });

        // then
        expect(screen.getByRole("button", { name: "GIF" })).toBeInTheDocument();
    });

    it("hands the full size rendition to the caller when a gif is picked", async () => {
        // given
        const onPick = vi.fn();
        setTrending({ data: { data: [makeGif()] } });
        const user = userEvent.setup();
        renderWithProviders(<GifPicker onPick={onPick} onClose={noop} />, { user: null });

        // when
        await user.click(screen.getByRole("button", { name: "Beato laughing" }));

        // then
        expect(onPick).toHaveBeenCalledWith({
            id: "gif-1",
            url: "https://giphy.test/gif-1-full.gif",
            description: "Beato laughing",
        });
    });

    it("falls back through the rendition preferences when the best one is missing", async () => {
        // given
        const onPick = vi.fn();
        setTrending({
            data: {
                data: [
                    makeGif({
                        images: {
                            downsized_medium: { url: "https://giphy.test/medium.gif", width: "200", height: "100" },
                            original: { url: "https://giphy.test/original.gif", width: "500", height: "400" },
                        },
                    }),
                ],
            },
        });
        const user = userEvent.setup();
        renderWithProviders(<GifPicker onPick={onPick} onClose={noop} />, { user: null });

        // when
        await user.click(screen.getByRole("button", { name: "Beato laughing" }));

        // then
        expect(onPick).toHaveBeenCalledWith(expect.objectContaining({ url: "https://giphy.test/medium.gif" }));
        expect(screen.getByAltText("Beato laughing")).toHaveAttribute("src", "https://giphy.test/original.gif");
    });

    it("waits for the typing to settle before searching giphy", () => {
        // given
        vi.useFakeTimers();
        setTrending({});
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });

        // when
        fireEvent.change(screen.getByPlaceholderText("Search GIPHY"), { target: { value: "beato" } });

        // then
        expect(mocks.useGiphySearch).not.toHaveBeenCalledWith("beato", 0, 0, expect.anything());
        act(() => {
            vi.advanceTimersByTime(600);
        });
        expect(mocks.useGiphySearch).toHaveBeenLastCalledWith("beato", 0, 0, true);
    });

    it("does not search for a single character", () => {
        // given
        vi.useFakeTimers();
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });

        // when
        fireEvent.change(screen.getByPlaceholderText("Search GIPHY"), { target: { value: "b" } });
        act(() => {
            vi.advanceTimersByTime(600);
        });

        // then
        expect(mocks.useGiphySearch).not.toHaveBeenCalledWith("b", 0, 0, expect.anything());
        expect(mocks.useGiphyTrending).toHaveBeenLastCalledWith(0, 0, true);
    });

    it("goes back to trending when the search is cleared", () => {
        // given
        vi.useFakeTimers();
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });
        const field = screen.getByPlaceholderText("Search GIPHY");
        fireEvent.change(field, { target: { value: "beato" } });
        act(() => {
            vi.advanceTimersByTime(600);
        });
        expect(mocks.useGiphyTrending).toHaveBeenLastCalledWith(0, 0, false);

        // when
        fireEvent.change(field, { target: { value: "" } });
        act(() => {
            vi.advanceTimersByTime(600);
        });

        // then
        expect(mocks.useGiphyTrending).toHaveBeenLastCalledWith(0, 0, true);
        expect(mocks.useGiphySearch).toHaveBeenLastCalledWith("", 0, 0, false);
    });

    it("stops asking giphy for anything while the favourites tab is open", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: makeUser() });

        // when
        await user.click(screen.getByRole("button", { name: /Favourites/ }));

        // then
        expect(mocks.useGiphyTrending).toHaveBeenLastCalledWith(0, 0, false);
        expect(screen.queryByPlaceholderText("Search GIPHY")).not.toBeInTheDocument();
    });

    it("shows that results are on their way", () => {
        // given
        setTrending({ loading: true });

        // when
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
        expect(screen.queryByText("No GIFs found")).not.toBeInTheDocument();
    });

    it("shows why giphy could not be reached", () => {
        // given
        setTrending({ error: new Error("giphy is unreachable") });

        // when
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });

        // then
        expect(screen.getByText("giphy is unreachable")).toBeInTheDocument();
    });

    it("explains when the search is empty", () => {
        // given
        setTrending({ data: { data: [] } });

        // when
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });

        // then
        expect(screen.getByText("No GIFs found")).toBeInTheDocument();
    });

    it("invites a signed in user to save their first favourite", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: makeUser() });

        // when
        await user.click(screen.getByRole("button", { name: /Favourites/ }));

        // then
        expect(screen.getByText("No favourites yet. Star a GIF to save it.")).toBeInTheDocument();
    });

    it("lists the saved favourites on the favourites tab", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, {
            user: makeUser(),
            gifFavourites: { favourites: [makeFavourite()], ids: new Set(["fav-1"]) },
        });

        // when
        await user.click(screen.getByRole("button", { name: /Favourites/ }));

        // then
        expect(screen.getByAltText("Bern smirking")).toHaveAttribute("src", "https://giphy.test/fav-1-small.gif");
        expect(screen.getByRole("button", { name: "Remove from favourites" })).toBeInTheDocument();
    });

    it("uses the full size favourite when it has no stored preview", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, {
            user: makeUser(),
            gifFavourites: { favourites: [makeFavourite({ preview_url: "", title: "" })], ids: new Set(["fav-1"]) },
        });

        // when
        await user.click(screen.getByRole("button", { name: /Favourites/ }));

        // then
        expect(screen.getByAltText("GIF")).toHaveAttribute("src", "https://giphy.test/fav-1.gif");
    });

    it("saves a gif with the details giphy reported", async () => {
        // given
        const toggle = vi.fn(() => Promise.resolve());
        setTrending({ data: { data: [makeGif()] } });
        const user = userEvent.setup();
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, {
            user: makeUser(),
            gifFavourites: { toggle },
        });

        // when
        await user.click(screen.getByRole("button", { name: "Add to favourites" }));

        // then
        expect(toggle).toHaveBeenCalledWith({
            giphy_id: "gif-1",
            url: "https://giphy.test/gif-1-full.gif",
            title: "Beato laughing",
            preview_url: "https://giphy.test/gif-1-small.gif",
            width: 320,
            height: 240,
        });
    });

    it("treats a size giphy cannot express as zero", async () => {
        // given
        const toggle = vi.fn(() => Promise.resolve());
        setTrending({
            data: {
                data: [
                    makeGif({
                        images: {
                            fixed_height: { url: "https://giphy.test/gif-1-full.gif", width: "auto", height: "" },
                            fixed_width_small: {
                                url: "https://giphy.test/gif-1-small.gif",
                                width: "100",
                                height: "80",
                            },
                        },
                    }),
                ],
            },
        });
        const user = userEvent.setup();
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, {
            user: makeUser(),
            gifFavourites: { toggle },
        });

        // when
        await user.click(screen.getByRole("button", { name: "Add to favourites" }));

        // then
        expect(toggle).toHaveBeenCalledWith(expect.objectContaining({ width: 0, height: 0 }));
    });

    it("does not pick the gif when its star is pressed", async () => {
        // given
        const onPick = vi.fn();
        setTrending({ data: { data: [makeGif()] } });
        const user = userEvent.setup();
        renderWithProviders(<GifPicker onPick={onPick} onClose={noop} />, {
            user: makeUser(),
            gifFavourites: { toggle: () => Promise.resolve() },
        });

        // when
        await user.click(screen.getByRole("button", { name: "Add to favourites" }));

        // then
        expect(onPick).not.toHaveBeenCalled();
    });

    it("marks a gif that is already saved as removable", () => {
        // given
        setTrending({ data: { data: [makeGif()] } });

        // when
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, {
            user: makeUser(),
            gifFavourites: { ids: new Set(["gif-1"]) },
        });

        // then
        expect(screen.getByRole("button", { name: "Remove from favourites" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Add to favourites" })).not.toBeInTheDocument();
    });

    it("pauses browsing until the giphy rate limit resets", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T09:00:00Z"));
        const resetAt = new Date("2026-08-02T09:30:00Z");
        setTrending({ error: new ApiError(429, "too many requests", { reset_at: "2026-08-02T09:30:00Z" }) });

        // when
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });

        // then
        const clock = resetAt.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
        expect(screen.getByText(`GIF search is paused. Try again at ${clock}.`)).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Search GIPHY")).toBeDisabled();
        expect(screen.queryByText("too many requests")).not.toBeInTheDocument();
    });

    it("still says browsing is paused when the rate limit carries no reset time", () => {
        // given
        setTrending({ error: new ApiError(429, "too many requests", {}) });

        // when
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });

        // then
        expect(screen.getByText("GIF search is paused. Try again shortly.")).toBeInTheDocument();
        expect(screen.queryByText("No GIFs found")).not.toBeInTheDocument();
        expect(screen.getByPlaceholderText("Search GIPHY")).toBeEnabled();
        expect(screen.queryByText("too many requests")).not.toBeInTheDocument();
    });

    it("retries by itself once the rate limit window has passed", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T09:00:00Z"));
        let currentError: unknown = new ApiError(429, "too many requests", { reset_at: "2026-08-02T09:30:00Z" });
        mocks.useGiphyTrending.mockImplementation(() => ({
            data: undefined,
            loading: false,
            error: currentError,
            refresh: mocks.trendingRefresh,
        }));
        renderWithProviders(<GifPicker onPick={noop} onClose={noop} />, { user: null });

        // when
        currentError = null;
        act(() => {
            vi.advanceTimersByTime(30 * 60 * 1000 + 500);
        });

        // then
        expect(mocks.trendingRefresh).toHaveBeenCalledOnce();
        expect(screen.getByPlaceholderText("Search GIPHY")).toBeEnabled();
    });

    it("closes when escape is pressed", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<GifPicker onPick={noop} onClose={onClose} />, { user: null });

        // when
        await user.keyboard("{Escape}");

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("closes when a press lands outside the picker", () => {
        // given
        const onClose = vi.fn();
        renderWithProviders(<GifPicker onPick={noop} onClose={onClose} />, { user: null });

        // when
        fireEvent.mouseDown(document.body);

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("stays open when a press lands inside the picker", () => {
        // given
        const onClose = vi.fn();
        setTrending({ data: { data: [makeGif()] } });
        renderWithProviders(<GifPicker onPick={noop} onClose={onClose} />, { user: null });

        // when
        fireEvent.mouseDown(screen.getByRole("button", { name: "Beato laughing" }));

        // then
        expect(onClose).not.toHaveBeenCalled();
    });

    it("stops listening to the document once it is unmounted", () => {
        // given
        const onClose = vi.fn();
        const { unmount } = renderWithProviders(<GifPicker onPick={noop} onClose={onClose} />, { user: null });

        // when
        unmount();
        fireEvent.mouseDown(document.body);

        // then
        expect(onClose).not.toHaveBeenCalled();
    });
});
