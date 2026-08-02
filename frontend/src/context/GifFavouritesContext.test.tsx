import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { GiphyFavourite } from "../api/endpoints";
import { useGifFavourites } from "../hooks/useGifFavourites";
import { makeUser } from "../test-utils/fixtures";
import { renderWithProviders } from "../test-utils/render";
import { GifFavouritesProvider } from "./GifFavouritesContext";

const { useGiphyFavourites, refetch, addFavourite, removeFavourite } = vi.hoisted(() => ({
    useGiphyFavourites: vi.fn(),
    refetch: vi.fn(),
    addFavourite: vi.fn(),
    removeFavourite: vi.fn(),
}));

vi.mock("../api/queries/giphy", () => ({ useGiphyFavourites }));

vi.mock("../api/mutations/giphy", () => ({
    useAddGiphyFavourite: () => ({ mutateAsync: addFavourite }),
    useRemoveGiphyFavourite: () => ({ mutateAsync: removeFavourite }),
}));

function makeFavourite(overrides: Partial<GiphyFavourite> = {}): GiphyFavourite {
    return {
        giphy_id: "gif-1",
        url: "https://giphy.test/gif-1.gif",
        title: "Beato laughing",
        preview_url: "https://giphy.test/gif-1-preview.gif",
        width: 240,
        height: 180,
        ...overrides,
    };
}

function Probe({ fav }: { fav: GiphyFavourite }) {
    const { favourites, isFavourite, toggle, refresh } = useGifFavourites();
    const [outcome, setOutcome] = useState("idle");

    function handleToggle() {
        toggle(fav)
            .then(() => setOutcome("settled"))
            .catch(() => setOutcome("rejected"));
    }

    function handleRefresh() {
        refresh().catch(() => setOutcome("rejected"));
    }

    return (
        <div>
            <p>{`saved: ${favourites.length}`}</p>
            <p>{`favourite: ${String(isFavourite(fav.giphy_id))}`}</p>
            <p>{`outcome: ${outcome}`}</p>
            <button type="button" onClick={handleToggle}>
                toggle
            </button>
            <button type="button" onClick={handleRefresh}>
                refresh
            </button>
        </div>
    );
}

function renderProbe(favourites: GiphyFavourite[], user: ReturnType<typeof makeUser> | null, fav: GiphyFavourite) {
    useGiphyFavourites.mockReturnValue({ favourites, total: favourites.length, loading: false, refresh: refetch });

    return renderWithProviders(
        <GifFavouritesProvider>
            <Probe fav={fav} />
        </GifFavouritesProvider>,
        { user },
    );
}

beforeEach(() => {
    refetch.mockResolvedValue(undefined);
    addFavourite.mockResolvedValue(undefined);
    removeFavourite.mockResolvedValue(undefined);
});

describe("GifFavouritesProvider", () => {
    it("hides the saved gifs from a signed out visitor", () => {
        // given
        const fav = makeFavourite();

        // when
        renderProbe([fav, makeFavourite({ giphy_id: "gif-2" })], null, fav);

        // then
        expect(screen.getByText("saved: 0")).toBeInTheDocument();
        expect(screen.getByText("favourite: false")).toBeInTheDocument();
    });

    it("exposes the saved gifs of a signed in user", () => {
        // given
        const fav = makeFavourite();

        // when
        renderProbe([fav, makeFavourite({ giphy_id: "gif-2" })], makeUser(), fav);

        // then
        expect(screen.getByText("saved: 2")).toBeInTheDocument();
        expect(screen.getByText("favourite: true")).toBeInTheDocument();
    });

    it("reports a gif that is not in the saved set as not a favourite", () => {
        // given
        const fav = makeFavourite({ giphy_id: "gif-9" });

        // when
        renderProbe([makeFavourite()], makeUser(), fav);

        // then
        expect(screen.getByText("saved: 1")).toBeInTheDocument();
        expect(screen.getByText("favourite: false")).toBeInTheDocument();
    });

    it("asks for a large page so the whole favourites set is available", () => {
        // given
        const fav = makeFavourite();

        // when
        renderProbe([], makeUser(), fav);

        // then
        expect(useGiphyFavourites).toHaveBeenCalledWith(0, 500);
    });

    it("saves a gif that has not been favourited yet", async () => {
        // given
        const fav = makeFavourite();
        const user = userEvent.setup();
        renderProbe([], makeUser(), fav);

        // when
        await user.click(screen.getByRole("button", { name: "toggle" }));

        // then
        expect(addFavourite).toHaveBeenCalledWith(fav);
        expect(removeFavourite).not.toHaveBeenCalled();
    });

    it("removes a gif that is already saved", async () => {
        // given
        const fav = makeFavourite();
        const user = userEvent.setup();
        renderProbe([fav], makeUser(), fav);

        // when
        await user.click(screen.getByRole("button", { name: "toggle" }));

        // then
        expect(removeFavourite).toHaveBeenCalledWith("gif-1");
        expect(addFavourite).not.toHaveBeenCalled();
    });

    it("does nothing when a signed out visitor toggles a gif", async () => {
        // given
        const fav = makeFavourite();
        const user = userEvent.setup();
        renderProbe([], null, fav);

        // when
        await user.click(screen.getByRole("button", { name: "toggle" }));

        // then
        expect(addFavourite).not.toHaveBeenCalled();
        expect(removeFavourite).not.toHaveBeenCalled();
    });

    it("does nothing when the gif has no giphy id", async () => {
        // given
        const fav = makeFavourite({ giphy_id: "" });
        const user = userEvent.setup();
        renderProbe([], makeUser(), fav);

        // when
        await user.click(screen.getByRole("button", { name: "toggle" }));

        // then
        expect(addFavourite).not.toHaveBeenCalled();
        expect(removeFavourite).not.toHaveBeenCalled();
    });

    it("swallows a failure to save so the caller never sees a rejection", async () => {
        // given
        addFavourite.mockRejectedValue(new Error("giphy is unhappy"));
        const fav = makeFavourite();
        const user = userEvent.setup();
        renderProbe([], makeUser(), fav);

        // when
        await user.click(screen.getByRole("button", { name: "toggle" }));

        // then
        expect(await screen.findByText("outcome: settled")).toBeInTheDocument();
    });

    it("swallows a failure to remove so the caller never sees a rejection", async () => {
        // given
        removeFavourite.mockRejectedValue(new Error("giphy is unhappy"));
        const fav = makeFavourite();
        const user = userEvent.setup();
        renderProbe([fav], makeUser(), fav);

        // when
        await user.click(screen.getByRole("button", { name: "toggle" }));

        // then
        expect(await screen.findByText("outcome: settled")).toBeInTheDocument();
    });

    it("delegates a refresh to the underlying query", async () => {
        // given
        const fav = makeFavourite();
        const user = userEvent.setup();
        renderProbe([fav], makeUser(), fav);

        // when
        await user.click(screen.getByRole("button", { name: "refresh" }));

        // then
        expect(refetch).toHaveBeenCalledOnce();
    });
});
