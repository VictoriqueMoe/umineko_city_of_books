import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { Art, Gallery, TagCount, UserProfile } from "../../types/api";
import { ArtGalleryPage } from "./ArtGalleryPage";

const mocks = vi.hoisted(() => ({
    useArtFeed: vi.fn(),
    useAllGalleries: vi.fn(),
    usePopularTags: vi.fn(),
    useRules: vi.fn(),
    useUserGalleries: vi.fn(),
    createGallery: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/art", () => ({
    useArtFeed: mocks.useArtFeed,
    useAllGalleries: mocks.useAllGalleries,
}));

vi.mock("../../api/queries/misc", () => ({
    usePopularTags: mocks.usePopularTags,
    useRules: mocks.useRules,
}));

vi.mock("../../api/queries/user", () => ({ useUserGalleries: mocks.useUserGalleries }));

vi.mock("../../api/mutations/art", () => ({
    useCreateGallery: () => ({ mutateAsync: mocks.createGallery, isPending: false }),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocks.navigate };
});

vi.mock("../../components/art/ArtGrid/ArtGrid", () => ({
    ArtGrid: ({ art }: { art: Art[] }) => <div data-testid="art-grid" data-count={art.length} />,
}));

vi.mock("../../components/art/ArtUploadForm/ArtUploadForm", () => ({
    ArtUploadForm: ({ galleryId, corner }: { galleryId: string; corner?: string }) => (
        <div data-testid="upload-form" data-gallery={galleryId} data-corner={corner} />
    ),
}));

const beatrice = { id: "beatrice-id", username: "beatrice", display_name: "Beatrice" };
const ronove = { id: "ronove-id", username: "ronove", display_name: "Ronove" };

function makeGallery(overrides: Partial<Gallery> = {}): Gallery {
    return {
        id: "gallery-1",
        author: beatrice,
        name: "Golden Butterflies",
        description: "",
        cover_image_url: "",
        cover_thumbnail_url: "",
        art_count: 3,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

function makeArt(overrides: Partial<Art> = {}): Art {
    return {
        id: "art-1",
        author: beatrice,
        corner: "general",
        art_type: "drawing",
        title: "Beatrice at dusk",
        description: "",
        image_url: "/art-1-full.png",
        thumbnail_url: "/art-1-thumb.png",
        tags: [],
        like_count: 0,
        comment_count: 0,
        view_count: 0,
        user_liked: false,
        is_spoiler: false,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

interface PageState {
    art?: Art[];
    total?: number;
    feedLoading?: boolean;
    galleries?: Gallery[];
    galleriesLoading?: boolean;
    userGalleries?: Gallery[];
    tags?: TagCount[];
    hasPrev?: boolean;
}

function stubPage(state: PageState = {}) {
    const refreshUserGalleries = vi.fn(() => Promise.resolve());
    mocks.useArtFeed.mockReturnValue({
        art: state.art ?? [],
        total: state.total ?? state.art?.length ?? 0,
        loading: state.feedLoading ?? false,
        offset: 0,
        limit: 24,
        hasNext: (state.total ?? 0) > 24,
        hasPrev: state.hasPrev ?? false,
        refresh: vi.fn(),
    });
    mocks.useAllGalleries.mockReturnValue({
        galleries: state.galleries ?? [],
        loading: state.galleriesLoading ?? false,
        refresh: vi.fn(),
    });
    mocks.usePopularTags.mockReturnValue({ tags: state.tags ?? [], loading: false });
    mocks.useUserGalleries.mockReturnValue({
        galleries: state.userGalleries ?? [],
        loading: false,
        refresh: refreshUserGalleries,
    });

    return { refreshUserGalleries };
}

function renderPage(route = "/gallery", viewer: UserProfile | null = makeUser({ id: "me", username: "me" })) {
    return renderWithProviders(<ArtGalleryPage />, { user: viewer, route });
}

beforeEach(() => {
    mocks.useRules.mockReturnValue({ rules: "", loading: false });
    mocks.createGallery.mockResolvedValue({ id: "gallery-new" });
    stubPage();
});

describe("ArtGalleryPage corners", () => {
    it("calls the general gallery simply the gallery", () => {
        // given
        stubPage();

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Gallery" })).toBeInTheDocument();
        expect(mocks.useRules).toHaveBeenCalledWith("gallery");
    });

    it("names the Umineko corner and shows its own rules", () => {
        // given
        stubPage();

        // when
        renderWithProviders(<ArtGalleryPage corner="umineko" />, { user: null, route: "/umineko/gallery" });

        // then
        expect(screen.getByRole("heading", { name: "Umineko Gallery" })).toBeInTheDocument();
        expect(mocks.useRules).toHaveBeenCalledWith("gallery_umineko");
    });

    it("names the Higurashi corner and shows its own rules", () => {
        // given
        stubPage();

        // when
        renderWithProviders(<ArtGalleryPage corner="higurashi" />, { user: null, route: "/higurashi/gallery" });

        // then
        expect(screen.getByRole("heading", { name: "Higurashi Gallery" })).toBeInTheDocument();
        expect(mocks.useRules).toHaveBeenCalledWith("gallery_higurashi");
    });

    it("names the Ciconia corner and shows its own rules", () => {
        // given
        stubPage();

        // when
        renderWithProviders(<ArtGalleryPage corner="ciconia" />, { user: null, route: "/ciconia/gallery" });

        // then
        expect(screen.getByRole("heading", { name: "Ciconia Gallery" })).toBeInTheDocument();
        expect(mocks.useRules).toHaveBeenCalledWith("gallery_ciconia");
    });

    it("asks the feed and the tag list for the corner it belongs to", () => {
        // given
        stubPage();

        // when
        renderWithProviders(<ArtGalleryPage corner="umineko" />, { user: null, route: "/umineko/gallery?view=all" });

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("umineko", undefined, undefined, undefined, "new", 1, 0);
        expect(mocks.usePopularTags).toHaveBeenCalledWith("umineko");
    });
});

describe("ArtGalleryPage by artist view", () => {
    it("opens on the by artist view", () => {
        // given
        stubPage();

        // when
        renderPage();

        // then
        expect(mocks.useAllGalleries).toHaveBeenLastCalledWith("general", true);
        expect(screen.queryByPlaceholderText("Search art...")).not.toBeInTheDocument();
    });

    it("says it is loading while the galleries are on their way", () => {
        // given
        stubPage({ galleriesLoading: true });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading galleries...")).toBeInTheDocument();
    });

    it("invites the first gallery when nobody has made one yet", () => {
        // given
        stubPage({ galleries: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText("No galleries yet. Be the first to create one.")).toBeInTheDocument();
    });

    it("groups every gallery under the artist who made it", () => {
        // given
        stubPage({
            galleries: [
                makeGallery({ id: "g1", name: "Golden Butterflies" }),
                makeGallery({ id: "g2", name: "Sketchbook" }),
                makeGallery({ id: "g3", name: "Tea Time", author: ronove }),
            ],
        });

        // when
        renderPage();

        // then
        expect(screen.getByText("2 galleries")).toBeInTheDocument();
        expect(screen.getByText("1 gallery")).toBeInTheDocument();
    });

    it("orders the artists by their display name", () => {
        // given
        stubPage({
            galleries: [makeGallery({ id: "g3", author: ronove }), makeGallery({ id: "g1", author: beatrice })],
        });

        // when
        const { container } = renderPage();

        // then
        const names = [...container.querySelectorAll("a")].map(a => a.getAttribute("href"));
        expect(names[0]).toBe("/user/beatrice");
    });

    it("links a gallery card through to that gallery", () => {
        // given
        stubPage({ galleries: [makeGallery({ id: "gallery-9" })] });

        // when
        renderPage();

        // then
        expect(screen.getByRole("link", { name: /Golden Butterflies/ })).toHaveAttribute(
            "href",
            "/gallery/view/gallery-9",
        );
    });

    it("counts the pieces on a gallery card", () => {
        // given
        stubPage({ galleries: [makeGallery({ art_count: 3 })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("3 pieces")).toBeInTheDocument();
    });

    it("calls a gallery with no cover and no art empty", () => {
        // given
        stubPage({ galleries: [makeGallery()] });

        // when
        renderPage();

        // then
        expect(screen.getByText("Empty")).toBeInTheDocument();
    });

    it("prefers the cover thumbnail over the full cover", () => {
        // given
        stubPage({
            galleries: [makeGallery({ cover_image_url: "/cover-full.png", cover_thumbnail_url: "/cover-thumb.png" })],
        });

        // when
        renderPage();

        // then
        const card = screen.getByRole("link", { name: /Golden Butterflies/ });
        expect(within(card).getByRole("presentation")).toHaveAttribute("src", "/cover-thumb.png");
    });

    it("falls back to a mosaic of recent pieces when there is no cover", () => {
        // given
        stubPage({
            galleries: [
                makeGallery({
                    preview_images: [
                        { thumbnail: "/one-thumb.png", full: "/one-full.png" },
                        { thumbnail: "/two-thumb.png", full: "/two-full.png" },
                        { thumbnail: "/three-thumb.png", full: "/three-full.png" },
                    ],
                }),
            ],
        });

        // when
        renderPage();

        // then
        const card = screen.getByRole("link", { name: /Golden Butterflies/ });
        expect(within(card).getAllByRole("presentation")).toHaveLength(3);
        expect(screen.queryByText("Empty")).not.toBeInTheDocument();
    });

    it("shows a lone preview image on its own", () => {
        // given
        stubPage({
            galleries: [makeGallery({ preview_images: [{ thumbnail: "", full: "/one-full.png" }] })],
        });

        // when
        renderPage();

        // then
        const card = screen.getByRole("link", { name: /Golden Butterflies/ });
        expect(within(card).getByRole("presentation")).toHaveAttribute("src", "/one-full.png");
    });

    it("shows a pair of preview images side by side", () => {
        // given
        stubPage({
            galleries: [
                makeGallery({
                    preview_images: [
                        { thumbnail: "/one-thumb.png", full: "/one-full.png" },
                        { thumbnail: "/two-thumb.png", full: "/two-full.png" },
                    ],
                }),
            ],
        });

        // when
        renderPage();

        // then
        const card = screen.getByRole("link", { name: /Golden Butterflies/ });
        expect(within(card).getAllByRole("presentation")).toHaveLength(2);
    });
});

describe("ArtGalleryPage all art view", () => {
    it("stops asking for the artist galleries once the visitor switches", async () => {
        // given
        stubPage();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "All Art" }));

        // then
        expect(mocks.useAllGalleries).toHaveBeenLastCalledWith("general", false);
        expect(screen.getByPlaceholderText("Search art...")).toBeInTheDocument();
    });

    it("says it is loading while the feed is on its way", () => {
        // given
        stubPage({ feedLoading: true });

        // when
        renderPage("/gallery?view=all");

        // then
        expect(screen.getByText("Loading gallery...")).toBeInTheDocument();
    });

    it("invites the first upload when the feed is empty", () => {
        // given
        stubPage({ art: [] });

        // when
        renderPage("/gallery?view=all");

        // then
        expect(screen.getByText("No art yet. Be the first to upload.")).toBeInTheDocument();
    });

    it("blames the search when a filtered feed comes back empty", () => {
        // given
        stubPage({ art: [] });

        // when
        renderPage("/gallery?view=all&search=beatrice");

        // then
        expect(screen.getByText("No art matches your search.")).toBeInTheDocument();
    });

    it("blames the tag filter when a tagged feed comes back empty", () => {
        // given
        stubPage({ art: [] });

        // when
        renderPage("/gallery?view=all&tag=sunset");

        // then
        expect(screen.getByText("No art matches your search.")).toBeInTheDocument();
    });

    it("hands every piece in the feed to the grid", () => {
        // given
        stubPage({ art: [makeArt(), makeArt({ id: "art-2" })] });

        // when
        renderPage("/gallery?view=all");

        // then
        expect(screen.getByTestId("art-grid")).toHaveAttribute("data-count", "2");
    });

    it("re-asks for the feed under the chosen sort", async () => {
        // given
        stubPage();
        const user = userEvent.setup();
        renderPage("/gallery?view=all");

        // when
        await user.click(screen.getByRole("button", { name: "Most Viewed" }));

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, undefined, "views", 1, 0);
    });

    it("drops the sort back out of the address bar when new is chosen again", async () => {
        // given
        stubPage();
        const user = userEvent.setup();
        renderPage("/gallery?view=all&sort=popular");

        // when
        await user.click(screen.getByRole("button", { name: "New" }));

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, undefined, "new", 1, 0);
    });

    it("narrows the feed to a single kind of art", async () => {
        // given
        stubPage();
        const user = userEvent.setup();
        renderPage("/gallery?view=all");

        // when
        await user.click(screen.getByRole("button", { name: "Cosplay" }));

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", "cosplay", undefined, undefined, "new", 1, 0);
    });

    it("clears the kind filter again", async () => {
        // given
        stubPage();
        const user = userEvent.setup();
        renderPage("/gallery?view=all&type=figure");

        // when
        await user.click(screen.getByRole("button", { name: "All" }));

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, undefined, "new", 1, 0);
    });

    it("hides the tag bar while there are no popular tags", () => {
        // given
        stubPage({ tags: [] });

        // when
        renderPage("/gallery?view=all");

        // then
        expect(screen.queryByRole("button", { name: "Clear filter" })).not.toBeInTheDocument();
    });

    it("shows each popular tag with how often it is used", () => {
        // given
        stubPage({ tags: [{ tag: "beatrice", count: 12 }] });

        // when
        renderPage("/gallery?view=all");

        // then
        expect(screen.getByRole("button", { name: "beatrice (12)" })).toBeInTheDocument();
    });

    it("filters the feed by a tag that was clicked", async () => {
        // given
        stubPage({ tags: [{ tag: "beatrice", count: 12 }] });
        const user = userEvent.setup();
        renderPage("/gallery?view=all");

        // when
        await user.click(screen.getByRole("button", { name: "beatrice (12)" }));

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, "beatrice", "new", 1, 0);
    });

    it("unpicks a tag that is already the active filter", async () => {
        // given
        stubPage({ tags: [{ tag: "beatrice", count: 12 }] });
        const user = userEvent.setup();
        renderPage("/gallery?view=all&tag=beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "beatrice (12)" }));

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, undefined, "new", 1, 0);
    });

    it("clears the tag filter from the clear chip", async () => {
        // given
        stubPage({ tags: [{ tag: "beatrice", count: 12 }] });
        const user = userEvent.setup();
        renderPage("/gallery?view=all&tag=beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Clear filter" }));

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, undefined, "new", 1, 0);
    });

    it("walks forward a page at a time", async () => {
        // given
        stubPage({ art: [makeArt()], total: 50 });
        const user = userEvent.setup();
        renderPage("/gallery?view=all");

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, undefined, "new", 2, 0);
    });

    it("walks back a page at a time", async () => {
        // given
        stubPage({ art: [makeArt()], total: 100, hasPrev: true });
        const user = userEvent.setup();
        renderPage("/gallery?view=all&page=3");

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, undefined, "new", 2, 0);
    });

    it("refuses to walk back past the first page", async () => {
        // given
        stubPage({ art: [makeArt()], total: 100, hasPrev: true });
        const user = userEvent.setup();
        renderPage("/gallery?view=all");

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, undefined, "new", 1, 0);
    });

    it("greys out the previous button on the first page", () => {
        // given
        stubPage({ art: [makeArt()], total: 100 });

        // when
        renderPage("/gallery?view=all");

        // then
        expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    });

    it("reads the page number straight out of the address bar", () => {
        // given
        stubPage({ art: [makeArt()], total: 100 });

        // when
        renderPage("/gallery?view=all&page=3");

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, undefined, "new", 3, 0);
    });
});

describe("ArtGalleryPage searching", () => {
    it("fills the search box from the address bar", () => {
        // given
        stubPage();

        // when
        renderPage("/gallery?view=all&search=beatrice");

        // then
        expect(screen.getByPlaceholderText("Search art...")).toHaveValue("beatrice");
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, "beatrice", undefined, "new", 1, 0);
    });

    it("waits for the typing to settle before searching", () => {
        // given
        vi.useFakeTimers();
        stubPage();
        renderPage("/gallery?view=all");

        // when
        fireEvent.change(screen.getByPlaceholderText("Search art..."), { target: { value: "beatrice" } });

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, undefined, "new", 1, 0);
        act(() => {
            vi.advanceTimersByTime(300);
        });
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, "beatrice", undefined, "new", 1, 0);
    });

    it("returns to the first page when a new search settles", () => {
        // given
        vi.useFakeTimers();
        stubPage({ art: [makeArt()], total: 100 });
        renderPage("/gallery?view=all&page=3");

        // when
        fireEvent.change(screen.getByPlaceholderText("Search art..."), { target: { value: "beatrice" } });
        act(() => {
            vi.advanceTimersByTime(300);
        });

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, "beatrice", undefined, "new", 1, 0);
    });

    it("drops the search again when the box is emptied", () => {
        // given
        vi.useFakeTimers();
        stubPage();
        renderPage("/gallery?view=all&search=beatrice");

        // when
        fireEvent.change(screen.getByPlaceholderText("Search art..."), { target: { value: "" } });
        act(() => {
            vi.advanceTimersByTime(300);
        });

        // then
        expect(mocks.useArtFeed).toHaveBeenLastCalledWith("general", undefined, undefined, undefined, "new", 1, 0);
    });
});

describe("ArtGalleryPage uploading", () => {
    it("hides the upload button from a signed out visitor", () => {
        // given
        stubPage();

        // when
        renderPage("/gallery", null);

        // then
        expect(screen.queryByRole("button", { name: "Upload Art" })).not.toBeInTheDocument();
    });

    it("offers the upload button to a signed in member", () => {
        // given
        stubPage();

        // when
        renderPage();

        // then
        expect(screen.getByRole("button", { name: "Upload Art" })).toBeInTheDocument();
    });

    it("asks a member with no gallery to make one first", async () => {
        // given
        stubPage({ userGalleries: [] });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Upload Art" }));

        // then
        expect(screen.getByText("You need a gallery first. Create one to start uploading art.")).toBeInTheDocument();
        expect(screen.queryByTestId("upload-form")).not.toBeInTheDocument();
    });

    it("keeps the create button shut while the gallery has no name", async () => {
        // given
        stubPage({ userGalleries: [] });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Upload Art" }));

        // then
        expect(screen.getByRole("button", { name: "Create" })).toBeDisabled();
    });

    it("creates the gallery under the trimmed name and reloads the member's galleries", async () => {
        // given
        const { refreshUserGalleries } = stubPage({ userGalleries: [] });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Upload Art" }));
        await user.type(screen.getByPlaceholderText("Gallery name"), "  Golden Butterflies  ");

        // when
        await user.click(screen.getByRole("button", { name: "Create" }));

        // then
        expect(mocks.createGallery).toHaveBeenCalledWith({ name: "Golden Butterflies" });
        await waitFor(() => {
            expect(refreshUserGalleries).toHaveBeenCalled();
        });
    });

    it("keeps the typed name and says why the gallery could not be created", async () => {
        // given
        mocks.createGallery.mockRejectedValue(new Error("that name is taken"));
        stubPage({ userGalleries: [] });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Upload Art" }));
        await user.type(screen.getByPlaceholderText("Gallery name"), "Golden Butterflies");

        // when
        await user.click(screen.getByRole("button", { name: "Create" }));

        // then
        expect(await screen.findByText("that name is taken")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Gallery name")).toHaveValue("Golden Butterflies");
    });

    it("clears the earlier complaint when the gallery is created on the second try", async () => {
        // given
        mocks.createGallery.mockRejectedValueOnce(new Error("that name is taken"));
        stubPage({ userGalleries: [] });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Upload Art" }));
        await user.type(screen.getByPlaceholderText("Gallery name"), "Golden Butterflies");
        await user.click(screen.getByRole("button", { name: "Create" }));
        expect(await screen.findByText("that name is taken")).toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "Create" }));

        // then
        await waitFor(() => {
            expect(screen.queryByText("that name is taken")).not.toBeInTheDocument();
        });
    });

    it("opens the upload form against the member's first gallery", async () => {
        // given
        stubPage({ userGalleries: [makeGallery({ id: "mine-1" }), makeGallery({ id: "mine-2" })] });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Upload Art" }));

        // then
        expect(screen.getByTestId("upload-form")).toHaveAttribute("data-gallery", "mine-1");
        expect(screen.getByTestId("upload-form")).toHaveAttribute("data-corner", "general");
    });

    it("closes the upload form again", async () => {
        // given
        stubPage({ userGalleries: [makeGallery({ id: "mine-1" })] });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Upload Art" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByTestId("upload-form")).not.toBeInTheDocument();
    });
});

describe("ArtGalleryPage how it works panel", () => {
    it("sends a member to their own profile to make a gallery", async () => {
        // given
        stubPage();
        const user = userEvent.setup();
        renderPage("/gallery", makeUser({ id: "me", username: "kujo" }));

        // when
        await user.click(screen.getByText("profile"));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/user/kujo");
    });

    it("sends a signed out visitor to sign in first", async () => {
        // given
        stubPage();
        const user = userEvent.setup();
        renderPage("/gallery", null);

        // when
        await user.click(screen.getByText("profile"));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/login");
    });
});
