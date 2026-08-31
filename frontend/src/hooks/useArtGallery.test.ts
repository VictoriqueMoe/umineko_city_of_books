import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { makeGallery as makeContentGallery, makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import type { Gallery, UserProfile } from "../types/api";
import { useArtGallery } from "./useArtGallery";

const mocks = vi.hoisted(() => ({
    useArtFeed: vi.fn(),
    useAllGalleries: vi.fn(),
    usePopularTags: vi.fn(),
    useUserGalleries: vi.fn(),
    createGallery: vi.fn(),
    refreshUserGalleries: vi.fn(),
}));

vi.mock("./queries/art", () => ({
    useArtFeed: mocks.useArtFeed,
    useAllGalleries: mocks.useAllGalleries,
    usePopularTags: mocks.usePopularTags,
}));
vi.mock("./queries/user", () => ({ useUserGalleries: mocks.useUserGalleries }));
vi.mock("./mutations/art", () => ({
    useCreateGallery: () => ({ mutateAsync: mocks.createGallery, isPending: false }),
}));

const beatrice = { id: "beatrice-id", username: "beatrice", display_name: "Beatrice" };
const ronove = { id: "ronove-id", username: "ronove", display_name: "Ronove" };

function makeGallery(overrides: Partial<Gallery> = {}): Gallery {
    return makeContentGallery({ author: beatrice, ...overrides });
}

function render(route = "/gallery", viewer: UserProfile | null = makeUser({ id: "me", username: "me" })) {
    return renderHook(() => useArtGallery("general"), { wrapper: providerWrapper({ user: viewer, route }) });
}

function lastFeedCall(): unknown[] {
    const calls = mocks.useArtFeed.mock.calls;
    return calls[calls.length - 1];
}

beforeEach(() => {
    mocks.useArtFeed.mockReturnValue({ art: [], total: 0, loading: false, refresh: vi.fn() });
    mocks.useAllGalleries.mockReturnValue({ galleries: [], loading: false, refresh: vi.fn() });
    mocks.usePopularTags.mockReturnValue({ tags: [], loading: false });
    mocks.useUserGalleries.mockReturnValue({
        galleries: [],
        loading: false,
        refresh: mocks.refreshUserGalleries,
    });
    mocks.refreshUserGalleries.mockResolvedValue(undefined);
    mocks.createGallery.mockResolvedValue({ id: "gallery-new" });
});

afterEach(() => {
    vi.useRealTimers();
});

describe("useArtGallery reading the address bar", () => {
    it("opens on the by artist view", () => {
        // when
        const { result } = render();

        // then
        expect(result.current.viewMode).toBe("galleries");
        expect(mocks.useAllGalleries).toHaveBeenLastCalledWith("general", true);
    });

    it("stops asking for the artist galleries once the all art view is chosen", () => {
        // when
        render("/gallery?view=all");

        // then
        expect(mocks.useAllGalleries).toHaveBeenLastCalledWith("general", false);
    });

    it("takes the sort, the type and the tag straight out of the address bar", () => {
        // when
        render("/gallery?view=all&sort=views&type=cosplay&tag=beatrice");

        // then
        expect(lastFeedCall()).toEqual(["general", "cosplay", undefined, "beatrice", "views", 0, 24]);
    });

    it("sorts by new when the address bar names no sort", () => {
        // when
        const { result } = render("/gallery?view=all");

        // then
        expect(result.current.sort).toBe("new");
        expect(lastFeedCall()).toEqual(["general", undefined, undefined, undefined, "new", 0, 24]);
    });

    it("turns the page number in the address bar into an offset", () => {
        // when
        const { result } = render("/gallery?view=all&page=3");

        // then
        expect(result.current.page.offset).toBe(48);
        expect(lastFeedCall()).toEqual(["general", undefined, undefined, undefined, "new", 48, 24]);
    });

    it("asks the corner it belongs to for its own tags", () => {
        // when
        render();

        // then
        expect(mocks.usePopularTags).toHaveBeenCalledWith("general");
    });
});

describe("useArtGallery paging", () => {
    it("walks forward a page at a time", async () => {
        // given
        const { result } = render("/gallery?view=all");

        // when
        act(() => result.current.page.goNext());

        // then
        await waitFor(() => expect(result.current.page.offset).toBe(24));
    });

    it("walks back a page at a time", async () => {
        // given
        const { result } = render("/gallery?view=all&page=3");

        // when
        act(() => result.current.page.goPrev());

        // then
        await waitFor(() => expect(result.current.page.offset).toBe(24));
    });

    it("knows there is nowhere back to go from the first page", () => {
        // when
        const { result } = render("/gallery?view=all");

        // then
        expect(result.current.page.hasPrev).toBe(false);
    });

    it("returns to the first page when a filter changes", async () => {
        // given
        const { result } = render("/gallery?view=all&page=3");

        // when
        act(() => result.current.updateParams({ sort: "views", page: "1" }));

        // then
        await waitFor(() => expect(result.current.page.offset).toBe(0));
        expect(result.current.sort).toBe("views");
    });

    it("drops a default value out of the address bar rather than writing it down", async () => {
        // given
        const { result } = render("/gallery?view=all&sort=popular");

        // when
        act(() => result.current.updateParams({ sort: "new" }));

        // then
        await waitFor(() => expect(result.current.sort).toBe("new"));
        expect(lastFeedCall()).toEqual(["general", undefined, undefined, undefined, "new", 0, 24]);
    });
});

describe("useArtGallery searching", () => {
    it("fills the search box from the address bar", () => {
        // when
        const { result } = render("/gallery?view=all&search=beatrice");

        // then
        expect(result.current.searchInput).toBe("beatrice");
        expect(lastFeedCall()).toEqual(["general", undefined, "beatrice", undefined, "new", 0, 24]);
    });

    it("waits for the typing to settle before searching", () => {
        // given
        vi.useFakeTimers();
        const { result } = render("/gallery?view=all");

        // when
        act(() => result.current.setSearchInput("beatrice"));

        // then
        expect(lastFeedCall()).toEqual(["general", undefined, undefined, undefined, "new", 0, 24]);
        act(() => {
            vi.advanceTimersByTime(300);
        });
        expect(lastFeedCall()).toEqual(["general", undefined, "beatrice", undefined, "new", 0, 24]);
    });

    it("searches only once for a run of typing", () => {
        // given
        vi.useFakeTimers();
        const { result } = render("/gallery?view=all");

        // when
        act(() => result.current.setSearchInput("bea"));
        act(() => {
            vi.advanceTimersByTime(200);
        });
        act(() => result.current.setSearchInput("beatrice"));
        act(() => {
            vi.advanceTimersByTime(300);
        });

        // then
        expect(lastFeedCall()).toEqual(["general", undefined, "beatrice", undefined, "new", 0, 24]);
        expect(mocks.useArtFeed.mock.calls.some(call => call[2] === "bea")).toBe(false);
    });

    it("goes back to the first page once a new search settles", () => {
        // given
        vi.useFakeTimers();
        const { result } = render("/gallery?view=all&page=3");

        // when
        act(() => result.current.setSearchInput("beatrice"));
        act(() => {
            vi.advanceTimersByTime(300);
        });

        // then
        expect(result.current.page.offset).toBe(0);
    });
});

describe("useArtGallery artists", () => {
    it("groups the galleries under the artist who made them", () => {
        // given
        mocks.useAllGalleries.mockReturnValue({
            galleries: [
                makeGallery({ id: "g1", author: ronove }),
                makeGallery({ id: "g2", author: beatrice }),
                makeGallery({ id: "g3", author: ronove }),
            ],
            loading: false,
            refresh: vi.fn(),
        });

        // when
        const { result } = render();

        // then
        expect(result.current.artists.map(a => a.user.display_name)).toEqual(["Beatrice", "Ronove"]);
        expect(result.current.artists[1].galleries).toHaveLength(2);
    });
});

describe("useArtGallery uploading", () => {
    it("uploads into the member's first gallery until another is picked", () => {
        // given
        mocks.useUserGalleries.mockReturnValue({
            galleries: [makeGallery({ id: "mine-1" }), makeGallery({ id: "mine-2" })],
            loading: false,
            refresh: mocks.refreshUserGalleries,
        });

        // when
        const { result } = render();

        // then
        expect(result.current.selectedGallery).toBe("mine-1");
    });

    it("uploads into the gallery that was picked instead", () => {
        // given
        mocks.useUserGalleries.mockReturnValue({
            galleries: [makeGallery({ id: "mine-1" }), makeGallery({ id: "mine-2" })],
            loading: false,
            refresh: mocks.refreshUserGalleries,
        });
        const { result } = render();

        // when
        act(() => result.current.setSelectedGallery("mine-2"));

        // then
        expect(result.current.selectedGallery).toBe("mine-2");
    });

    it("asks for nobody's galleries while the visitor is signed out", () => {
        // when
        render("/gallery", null);

        // then
        expect(mocks.useUserGalleries).toHaveBeenCalledWith("");
    });

    it("creates the gallery under the trimmed name and reloads the member's galleries", async () => {
        // given
        const { result } = render();
        act(() => result.current.setNewGalleryName("  Golden Butterflies  "));

        // when
        await act(() => result.current.createGallery());

        // then
        expect(mocks.createGallery).toHaveBeenCalledWith({ name: "Golden Butterflies" });
        expect(mocks.refreshUserGalleries).toHaveBeenCalledOnce();
    });

    it("uploads into the gallery that was just created", async () => {
        // given
        const { result } = render();
        act(() => result.current.setNewGalleryName("Golden Butterflies"));

        // when
        await act(() => result.current.createGallery());

        // then
        expect(result.current.selectedGallery).toBe("gallery-new");
        expect(result.current.newGalleryName).toBe("");
    });

    it("refuses to create a gallery with no name", async () => {
        // given
        const { result } = render();
        act(() => result.current.setNewGalleryName("   "));

        // when
        await act(() => result.current.createGallery());

        // then
        expect(mocks.createGallery).not.toHaveBeenCalled();
    });

    it("keeps the typed name and says why the gallery could not be created", async () => {
        // given
        mocks.createGallery.mockRejectedValue(new Error("that name is taken"));
        const { result } = render();
        act(() => result.current.setNewGalleryName("Golden Butterflies"));

        // when
        await act(() => result.current.createGallery());

        // then
        expect(result.current.galleryError).toBe("that name is taken");
        expect(result.current.newGalleryName).toBe("Golden Butterflies");
    });

    it("falls back to a plain complaint when the refusal carries no reason", async () => {
        // given
        mocks.createGallery.mockRejectedValue({ status: 500 });
        const { result } = render();
        act(() => result.current.setNewGalleryName("Golden Butterflies"));

        // when
        await act(() => result.current.createGallery());

        // then
        expect(result.current.galleryError).toBe("Failed to create the gallery");
    });

    it("clears the earlier complaint when the gallery is created on the second try", async () => {
        // given
        mocks.createGallery.mockRejectedValueOnce(new Error("that name is taken"));
        const { result } = render();
        act(() => result.current.setNewGalleryName("Golden Butterflies"));
        await act(() => result.current.createGallery());

        // when
        await act(() => result.current.createGallery());

        // then
        expect(result.current.galleryError).toBe("");
    });
});
