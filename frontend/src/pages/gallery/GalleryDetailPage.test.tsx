import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { Art, Gallery, UserProfile } from "../../types/api";
import { GalleryDetailPage } from "./GalleryDetailPage";

const mocks = vi.hoisted(() => ({
    useGallery: vi.fn(),
    deleteGallery: vi.fn(),
    updateGallery: vi.fn(),
    setGalleryCover: vi.fn(),
    setArtGallery: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/art", () => ({ useGallery: mocks.useGallery }));

vi.mock("../../api/mutations/art", () => ({
    useDeleteGallery: () => ({ mutateAsync: mocks.deleteGallery }),
    useUpdateGallery: () => ({ mutateAsync: mocks.updateGallery }),
    useSetGalleryCover: () => ({ mutateAsync: mocks.setGalleryCover }),
    useSetArtGallery: () => ({ mutateAsync: mocks.setArtGallery }),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocks.navigate };
});

vi.mock("../../components/art/ArtUploadForm/ArtUploadForm", () => ({
    ArtUploadForm: ({ galleryId }: { galleryId: string }) => <div data-testid="upload-form" data-gallery={galleryId} />,
}));

const author = { id: "author-1", username: "ronove", display_name: "Ronove" };

function makeGallery(overrides: Partial<Gallery> = {}): Gallery {
    return {
        id: "gallery-1",
        author,
        name: "Golden Butterflies",
        description: "sketches from the rose garden",
        cover_image_url: "",
        cover_thumbnail_url: "",
        art_count: 2,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

function makeArt(overrides: Partial<Art> = {}): Art {
    return {
        id: "art-1",
        author,
        corner: "general",
        art_type: "drawing",
        title: "Beatrice at dusk",
        description: "",
        image_url: "/art-1-full.png",
        thumbnail_url: "/art-1-thumb.png",
        tags: [],
        like_count: 4,
        comment_count: 0,
        view_count: 0,
        user_liked: false,
        is_spoiler: false,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

interface GalleryState {
    gallery?: Gallery | null;
    art?: Art[];
    total?: number;
    loading?: boolean;
}

function stubGallery(state: GalleryState = {}) {
    const refresh = vi.fn(() => Promise.resolve());
    mocks.useGallery.mockReturnValue({
        gallery: state.gallery === undefined ? makeGallery() : state.gallery,
        art: state.art ?? [],
        total: state.total ?? state.art?.length ?? 0,
        loading: state.loading ?? false,
        refresh,
    });
    return { refresh };
}

function renderPage(viewer: UserProfile | null = makeUser({ id: "author-1" })) {
    return renderWithProviders(<GalleryDetailPage />, {
        user: viewer,
        route: "/gallery/view/gallery-1",
        path: "/gallery/view/:id",
    });
}

beforeEach(() => {
    mocks.deleteGallery.mockResolvedValue({});
    mocks.updateGallery.mockResolvedValue({});
    mocks.setGalleryCover.mockResolvedValue({});
    mocks.setArtGallery.mockResolvedValue({});
});

describe("GalleryDetailPage loading and missing states", () => {
    it("says it is loading while the gallery is on its way", () => {
        // given
        stubGallery({ gallery: null, loading: true });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading gallery...")).toBeInTheDocument();
    });

    it("says the gallery could not be found once the fetch has settled", () => {
        // given
        stubGallery({ gallery: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Gallery not found.")).toBeInTheDocument();
    });

    it("asks for the first two dozen pieces", () => {
        // given
        stubGallery();

        // when
        renderPage();

        // then
        expect(mocks.useGallery).toHaveBeenCalledWith("gallery-1", 24, 0);
    });
});

describe("GalleryDetailPage content", () => {
    it("shows the name, the description and how many pieces it holds", () => {
        // given
        stubGallery();

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Golden Butterflies" })).toBeInTheDocument();
        expect(screen.getByText("sketches from the rose garden")).toBeInTheDocument();
        expect(screen.getByText("2 pieces")).toBeInTheDocument();
    });

    it("shows the cover thumbnail when the gallery has a cover", () => {
        // given
        stubGallery({
            gallery: makeGallery({ cover_image_url: "/cover-full.png", cover_thumbnail_url: "/cover-thumb.png" }),
        });

        // when
        const { container } = renderPage();

        // then
        expect(container.querySelector("img")).toHaveAttribute("src", "/cover-thumb.png");
    });

    it("says the gallery is empty when it holds nothing", () => {
        // given
        stubGallery({ art: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText("This gallery is empty.")).toBeInTheDocument();
    });

    it("links every piece through to its own page", () => {
        // given
        stubGallery({ art: [makeArt({ id: "art-9" })] });

        // when
        renderPage();

        // then
        expect(screen.getByRole("link", { name: /Beatrice at dusk/ })).toHaveAttribute("href", "/gallery/art/art-9");
    });

    it("prefers the thumbnail for a piece in the grid", () => {
        // given
        stubGallery({ art: [makeArt()] });

        // when
        renderPage();

        // then
        expect(screen.getByRole("img", { name: "Beatrice at dusk" })).toHaveAttribute("src", "/art-1-thumb.png");
    });

    it("shows how many likes each piece has", () => {
        // given
        stubGallery({ art: [makeArt({ like_count: 4 })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("♥ 4")).toBeInTheDocument();
    });

    it("goes back the way the visitor came", async () => {
        // given
        stubGallery();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText("← Back"));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith(-1);
    });
});

describe("GalleryDetailPage ownership", () => {
    it("hides the owner tools from somebody else's gallery", () => {
        // given
        stubGallery();

        // when
        renderPage(makeUser({ id: "someone-else" }));

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Manage Art" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
        expect(screen.queryByTestId("upload-form")).not.toBeInTheDocument();
    });

    it("hides the owner tools from a signed out visitor", () => {
        // given
        stubGallery();

        // when
        renderPage(null);

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByTestId("upload-form")).not.toBeInTheDocument();
    });

    it("offers the owner the full set of tools and the upload form", () => {
        // given
        stubGallery();

        // when
        renderPage();

        // then
        expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Manage Art" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
        expect(screen.getByTestId("upload-form")).toHaveAttribute("data-gallery", "gallery-1");
    });
});

describe("GalleryDetailPage editing", () => {
    async function openEditor() {
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Edit" }));

        return user;
    }

    it("fills the editor with the gallery as it stands", async () => {
        // given
        stubGallery();

        // when
        await openEditor();

        // then
        expect(screen.getByPlaceholderText("Gallery name")).toHaveValue("Golden Butterflies");
        expect(screen.getByPlaceholderText("Description (optional)")).toHaveValue("sketches from the rose garden");
    });

    it("saves the trimmed name and description", async () => {
        // given
        stubGallery();
        const user = await openEditor();
        const nameBox = screen.getByPlaceholderText("Gallery name");
        await user.clear(nameBox);
        await user.type(nameBox, "  Silver Butterflies  ");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.updateGallery).toHaveBeenCalledWith({
            name: "Silver Butterflies",
            description: "sketches from the rose garden",
        });
    });

    it("refreshes the gallery once the edit has been saved", async () => {
        // given
        const { refresh } = stubGallery();
        const user = await openEditor();

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });

    it("locks saving once the name has been emptied", async () => {
        // given
        stubGallery();
        const user = await openEditor();

        // when
        await user.clear(screen.getByPlaceholderText("Gallery name"));

        // then
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("throws the editor away when the edit is cancelled", async () => {
        // given
        stubGallery();
        const user = await openEditor();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByPlaceholderText("Gallery name")).not.toBeInTheDocument();
        expect(mocks.updateGallery).not.toHaveBeenCalled();
    });
});

describe("GalleryDetailPage managing art", () => {
    async function openManager(art: Art[] = [makeArt()]) {
        stubGallery({ art });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Manage Art" }));

        return user;
    }

    it("swaps the grid for the manage tools", async () => {
        // given
        const art = [makeArt()];

        // when
        await openManager(art);

        // then
        expect(screen.getByRole("button", { name: "Set as Cover" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
        expect(screen.queryByRole("link", { name: /Beatrice at dusk/ })).not.toBeInTheDocument();
    });

    it("hides the empty state while managing an empty gallery", async () => {
        // given
        const art: Art[] = [];

        // when
        await openManager(art);

        // then
        expect(screen.queryByText("This gallery is empty.")).not.toBeInTheDocument();
    });

    it("makes a piece the cover and refreshes the gallery", async () => {
        // given
        stubGallery({ art: [makeArt({ id: "art-3" })] });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Manage Art" }));

        // when
        await user.click(screen.getByRole("button", { name: "Set as Cover" }));

        // then
        expect(mocks.setGalleryCover).toHaveBeenCalledWith("art-3");
    });

    it("asks before pulling a piece out of the gallery", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = await openManager();

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Remove this art from the gallery?");
        expect(mocks.setArtGallery).not.toHaveBeenCalled();
    });

    it("pulls a piece out of the gallery once confirmed", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        stubGallery({ art: [makeArt({ id: "art-3" })] });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Manage Art" }));

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(mocks.setArtGallery).toHaveBeenCalledWith({ artId: "art-3", galleryId: null });
    });

    it("leaves the manage view again", async () => {
        // given
        const user = await openManager();

        // when
        await user.click(screen.getByRole("button", { name: "Done" }));

        // then
        expect(screen.getByRole("link", { name: /Beatrice at dusk/ })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Set as Cover" })).not.toBeInTheDocument();
    });
});

describe("GalleryDetailPage deleting", () => {
    it("keeps the confirmation closed until delete is pressed", () => {
        // given
        stubGallery();

        // when
        renderPage();

        // then
        expect(screen.queryByRole("button", { name: "Delete Gallery" })).not.toBeInTheDocument();
    });

    it("warns that every piece goes with the gallery", async () => {
        // given
        stubGallery();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(
            screen.getByText(
                "Are you sure you want to delete this gallery? All art in it will be permanently deleted.",
            ),
        ).toBeInTheDocument();
    });

    it("deletes the gallery and goes back once confirmed", async () => {
        // given
        stubGallery();
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // when
        await user.click(screen.getByRole("button", { name: "Delete Gallery" }));

        // then
        expect(mocks.deleteGallery).toHaveBeenCalledWith("gallery-1");
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith(-1);
        });
    });

    it("keeps the gallery when the confirmation is dismissed", async () => {
        // given
        stubGallery();
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByRole("button", { name: "Delete Gallery" })).not.toBeInTheDocument();
        expect(mocks.deleteGallery).not.toHaveBeenCalled();
    });
});

describe("GalleryDetailPage paging", () => {
    it("hides the pager when everything fits on one page", () => {
        // given
        stubGallery({ art: [makeArt()], total: 24 });

        // when
        renderPage();

        // then
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("asks for the next two dozen pieces", async () => {
        // given
        stubGallery({ art: [makeArt()], total: 50 });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(mocks.useGallery).toHaveBeenLastCalledWith("gallery-1", 24, 24);
    });

    it("walks back to the previous page of pieces", async () => {
        // given
        stubGallery({ art: [makeArt()], total: 50 });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Next" }));

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(mocks.useGallery).toHaveBeenLastCalledWith("gallery-1", 24, 0);
    });
});
