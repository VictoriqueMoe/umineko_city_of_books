import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SiteInfo } from "../../../api/endpoints";
import { renderWithProviders } from "../../../test-utils/render";
import type { Gallery } from "../../../types/api";
import { ArtUploadForm } from "./ArtUploadForm";

const mocks = vi.hoisted(() => ({
    createArt: vi.fn(),
    setArtGallery: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../../api/mutations/art", () => ({
    useCreateArt: () => ({ mutateAsync: mocks.createArt }),
    useSetArtGallery: () => ({ mutateAsync: mocks.setArtGallery }),
}));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => mocks.navigate };
});

const galleryId = "gallery-1";

function makeGallery(id: string, name: string): Gallery {
    return {
        id,
        author: { id: "user-1", username: "beatrice", display_name: "Beatrice" },
        name,
        description: "",
        cover_image_url: "",
        cover_thumbnail_url: "",
        art_count: 0,
        created_at: "2026-01-01T00:00:00Z",
    };
}

function imageFile(name = "butterflies.png", size = 64): File {
    return new File(["x".repeat(size)], name, { type: "image/png" });
}

interface FormOverrides {
    onCreated?: () => void;
    inline?: boolean;
    corner?: string;
    galleries?: Gallery[];
    selectedGallery?: string;
    onGalleryChange?: (id: string) => void;
    siteInfo?: Partial<SiteInfo>;
}

function noop() {}

function renderForm(overrides: FormOverrides = {}) {
    return renderWithProviders(
        <ArtUploadForm
            galleryId={galleryId}
            corner={overrides.corner}
            onCreated={overrides.onCreated ?? noop}
            inline={overrides.inline ?? true}
            galleries={overrides.galleries}
            selectedGallery={overrides.selectedGallery}
            onGalleryChange={overrides.onGalleryChange}
        />,
        { siteInfo: overrides.siteInfo },
    );
}

function pickFile(container: HTMLElement): HTMLInputElement {
    const input = container.querySelector<HTMLInputElement>('input[type="file"]');
    if (!input) {
        throw new Error("the form has no file input");
    }
    return input;
}

function uploadButton(): HTMLElement {
    return screen.getByRole("button", { name: "Upload" });
}

function selectOwning(optionLabel: string): HTMLSelectElement {
    const select = screen.getByRole("option", { name: optionLabel }).closest("select");
    if (!select) {
        throw new Error(`no select owns the option ${optionLabel}`);
    }
    return select;
}

beforeEach(() => {
    mocks.createArt.mockResolvedValue({ id: "art-9" });
    mocks.setArtGallery.mockResolvedValue(undefined);
});

describe("ArtUploadForm", () => {
    it("shows only a trigger until the form is opened", async () => {
        // given
        const user = userEvent.setup();
        renderForm({ inline: false });

        // when
        await user.click(screen.getByRole("button", { name: "Upload Art" }));

        // then
        expect(screen.getByRole("heading", { name: "Upload Art" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Give your art a title")).toBeInTheDocument();
    });

    it("hides the form fields before the trigger is pressed", () => {
        // given
        const inline = false;

        // when
        renderForm({ inline });

        // then
        expect(screen.queryByPlaceholderText("Give your art a title")).not.toBeInTheDocument();
    });

    it("closes the form again when cancel is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderForm({ inline: false });
        await user.click(screen.getByRole("button", { name: "Upload Art" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByPlaceholderText("Give your art a title")).not.toBeInTheDocument();
    });

    it("shows the form straight away when it is inline and offers no cancel", () => {
        // given
        const inline = true;

        // when
        renderForm({ inline });

        // then
        expect(screen.getByPlaceholderText("Give your art a title")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Cancel" })).not.toBeInTheDocument();
    });

    it("keeps the upload disabled while no image has been chosen", async () => {
        // given
        const user = userEvent.setup();
        renderForm();

        // when
        await user.type(screen.getByPlaceholderText("Give your art a title"), "Golden Butterflies");

        // then
        expect(uploadButton()).toBeDisabled();
    });

    it("keeps the upload disabled while there is no title", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderForm();

        // when
        await user.upload(pickFile(container), imageFile());

        // then
        expect(uploadButton()).toBeDisabled();
    });

    it("enables the upload once both a title and an image are present", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderForm();
        await user.type(screen.getByPlaceholderText("Give your art a title"), "Golden Butterflies");

        // when
        await user.upload(pickFile(container), imageFile());

        // then
        expect(uploadButton()).toBeEnabled();
    });

    it("treats a title of only whitespace as no title at all", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderForm();
        await user.upload(pickFile(container), imageFile());

        // when
        await user.type(screen.getByPlaceholderText("Give your art a title"), "   ");

        // then
        expect(uploadButton()).toBeDisabled();
    });

    it("rejects an image that is larger than the site limit", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderForm({ siteInfo: { max_image_size: 32 } });

        // when
        await user.upload(pickFile(container), imageFile("huge.png", 64));

        // then
        expect(screen.getByText(/huge\.png is too large/)).toBeInTheDocument();
        expect(screen.queryByAltText("Preview")).not.toBeInTheDocument();
    });

    it("accepts an image that sits inside the site limit", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderForm({ siteInfo: { max_image_size: 128 } });

        // when
        await user.upload(pickFile(container), imageFile("small.png", 64));

        // then
        expect(screen.getByAltText("Preview")).toBeInTheDocument();
    });

    it("rejects a file that is not an image", async () => {
        // given
        const user = userEvent.setup({ applyAccept: false });
        const { container } = renderForm();

        // when
        await user.upload(pickFile(container), new File(["notes"], "notes.txt", { type: "text/plain" }));

        // then
        expect(screen.getByText("Only image files are allowed")).toBeInTheDocument();
        expect(screen.queryByAltText("Preview")).not.toBeInTheDocument();
    });

    it("lets the chosen image be removed again", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderForm();
        await user.upload(pickFile(container), imageFile());

        // when
        await user.click(screen.getByRole("button", { name: /Remove/ }));

        // then
        expect(screen.queryByAltText("Preview")).not.toBeInTheDocument();
        expect(screen.getByText("Click to select an image")).toBeInTheDocument();
    });

    it("releases the old preview object URL when another image is chosen", async () => {
        // given
        const urls: string[] = [];
        const create = vi.spyOn(URL, "createObjectURL").mockImplementation(() => {
            const url = `blob:preview-${urls.length}`;
            urls.push(url);
            return url;
        });
        const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
        const user = userEvent.setup();
        const { container } = renderForm();
        await user.upload(pickFile(container), imageFile("first.png"));

        // when
        await user.upload(pickFile(container), imageFile("second.png"));

        // then
        expect(revoke).toHaveBeenCalledWith(urls[0]);
        expect(revoke).not.toHaveBeenCalledWith(urls[1]);
        create.mockRestore();
        revoke.mockRestore();
    });

    it("releases the preview object URL when the form goes away", async () => {
        // given
        const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:preview-only");
        const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
        const user = userEvent.setup();
        const { container, unmount } = renderForm();
        await user.upload(pickFile(container), imageFile());

        // when
        unmount();

        // then
        expect(revoke).toHaveBeenCalledWith("blob:preview-only");
        create.mockRestore();
        revoke.mockRestore();
    });

    it("submits the trimmed details along with the tags and the spoiler flag", async () => {
        // given
        const user = userEvent.setup();
        const file = imageFile();
        const { container } = renderForm({ corner: "fanart" });
        await user.type(screen.getByPlaceholderText("Give your art a title"), "  Golden Butterflies  ");
        await user.type(screen.getByPlaceholderText("Describe your art (optional)"), "  drawn for Beato  ");
        await user.type(screen.getByPlaceholderText("Add tag..."), "epitaph{Enter}");
        await user.click(screen.getByRole("switch", { name: "Contains spoilers" }));
        await user.selectOptions(screen.getByRole("combobox"), "cosplay");
        await user.upload(pickFile(container), file);

        // when
        await user.click(uploadButton());

        // then
        await waitFor(() => expect(mocks.createArt).toHaveBeenCalledOnce());
        expect(mocks.createArt).toHaveBeenCalledWith({
            metadata: {
                title: "Golden Butterflies",
                description: "drawn for Beato",
                corner: "fanart",
                art_type: "cosplay",
                tags: ["epitaph"],
                is_spoiler: true,
                gallery_id: galleryId,
            },
            imageFile: file,
        });
    });

    it("defaults the corner and the art type when nothing else is chosen", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderForm();
        await user.type(screen.getByPlaceholderText("Give your art a title"), "Rokkenjima");
        await user.upload(pickFile(container), imageFile());

        // when
        await user.click(uploadButton());

        // then
        await waitFor(() => expect(mocks.createArt).toHaveBeenCalledOnce());
        expect(mocks.createArt.mock.calls[0][0].metadata).toMatchObject({
            corner: "general",
            art_type: "drawing",
            tags: [],
            is_spoiler: false,
        });
    });

    it("files the new art into the gallery and tells the parent it is done", async () => {
        // given
        const onCreated = vi.fn();
        const user = userEvent.setup();
        const { container } = renderForm({ onCreated });
        await user.type(screen.getByPlaceholderText("Give your art a title"), "Rokkenjima");
        await user.upload(pickFile(container), imageFile());

        // when
        await user.click(uploadButton());

        // then
        await waitFor(() => expect(onCreated).toHaveBeenCalledOnce());
        expect(mocks.setArtGallery).toHaveBeenCalledWith({ artId: "art-9", galleryId });
        expect(mocks.navigate).toHaveBeenCalledWith("/gallery/art/art-9");
    });

    it("still finishes when filing the art into the gallery fails", async () => {
        // given
        const onCreated = vi.fn();
        const user = userEvent.setup();
        mocks.setArtGallery.mockRejectedValue(new Error("the gallery refused"));
        const { container } = renderForm({ onCreated });
        await user.type(screen.getByPlaceholderText("Give your art a title"), "Rokkenjima");
        await user.upload(pickFile(container), imageFile());

        // when
        await user.click(uploadButton());

        // then
        await waitFor(() => expect(onCreated).toHaveBeenCalledOnce());
        expect(mocks.navigate).toHaveBeenCalledWith("/gallery/art/art-9");
    });

    it("shows why the upload failed and lets it be tried again", async () => {
        // given
        const onCreated = vi.fn();
        const user = userEvent.setup();
        mocks.createArt.mockRejectedValue(new Error("Beato refused the offering"));
        const { container } = renderForm({ onCreated });
        await user.type(screen.getByPlaceholderText("Give your art a title"), "Rokkenjima");
        await user.upload(pickFile(container), imageFile());

        // when
        await user.click(uploadButton());

        // then
        expect(await screen.findByText("Beato refused the offering")).toBeInTheDocument();
        expect(onCreated).not.toHaveBeenCalled();
        expect(uploadButton()).toBeEnabled();
    });

    it("falls back to a generic message when the failure carries no message", async () => {
        // given
        const user = userEvent.setup();
        mocks.createArt.mockRejectedValue("something odd");
        const { container } = renderForm();
        await user.type(screen.getByPlaceholderText("Give your art a title"), "Rokkenjima");
        await user.upload(pickFile(container), imageFile());

        // when
        await user.click(uploadButton());

        // then
        expect(await screen.findByText("Failed to upload art")).toBeInTheDocument();
    });

    it("offers no gallery picker unless galleries and a change handler are supplied", () => {
        // given
        const galleries = [makeGallery("gallery-1", "Main gallery")];

        // when
        renderForm({ galleries });

        // then
        expect(screen.queryByRole("option", { name: "Main gallery" })).not.toBeInTheDocument();
    });

    it("forwards the newly chosen gallery to the parent", async () => {
        // given
        const onGalleryChange = vi.fn();
        const user = userEvent.setup();
        const galleries = [makeGallery("gallery-1", "Main gallery"), makeGallery("gallery-2", "Side gallery")];
        renderForm({ galleries, onGalleryChange, selectedGallery: "gallery-1" });

        // when
        await user.selectOptions(selectOwning("Main gallery"), "gallery-2");

        // then
        expect(onGalleryChange).toHaveBeenCalledWith("gallery-2");
    });
});
