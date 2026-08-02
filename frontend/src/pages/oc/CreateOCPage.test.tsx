import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { OCDetail, OCImage } from "../../types/api";
import { CreateOCPage } from "./CreateOCPage";

const mocks = vi.hoisted(() => ({
    useOC: vi.fn(),
    createOC: vi.fn(),
    updateOC: vi.fn(),
    deleteOC: vi.fn(),
    uploadImage: vi.fn(),
    addGalleryImage: vi.fn(),
    deleteGalleryImage: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/oc", () => ({ useOC: mocks.useOC }));

vi.mock("../../api/mutations/oc", () => ({
    useCreateOC: () => ({ mutateAsync: mocks.createOC }),
    useUpdateOC: () => ({ mutateAsync: mocks.updateOC }),
    useDeleteOC: () => ({ mutateAsync: mocks.deleteOC }),
    useUploadOCImageById: () => ({ mutateAsync: mocks.uploadImage }),
    useAddOCGalleryImage: () => ({ mutateAsync: mocks.addGalleryImage }),
    useDeleteOCGalleryImage: () => ({ mutateAsync: mocks.deleteGalleryImage }),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocks.navigate };
});

vi.mock("../../components/MentionTextArea/MentionTextArea", () => ({
    MentionTextArea: ({
        value,
        onChange,
        placeholder,
    }: {
        value: string;
        onChange: (v: string) => void;
        placeholder?: string;
    }) => <textarea placeholder={placeholder} value={value} onChange={e => onChange(e.target.value)} />,
}));

function makeImage(overrides: Partial<OCImage> = {}): OCImage {
    return {
        id: 11,
        image_url: "/gallery-full.png",
        thumbnail_url: "/gallery-thumb.png",
        sort_order: 0,
        ...overrides,
    };
}

function makeOC(overrides: Partial<OCDetail> = {}): OCDetail {
    return {
        id: "oc-1",
        author: { id: "author-1", username: "ronove", display_name: "Ronove" },
        name: "Featherine Junior",
        description: "a witch in training",
        series: "higurashi",
        gallery: [],
        vote_score: 0,
        favourite_count: 0,
        user_favourited: false,
        comment_count: 0,
        is_crack_oc: false,
        created_at: "2026-07-01T10:00:00Z",
        comments: [],
        viewer_blocked: false,
        ...overrides,
    };
}

function stubOC(oc: OCDetail | null = null, loading = false) {
    mocks.useOC.mockReturnValue({ oc, loading, refresh: vi.fn(() => Promise.resolve()) });
}

function renderCreate() {
    return renderWithProviders(<CreateOCPage mode="create" />, { route: "/oc/new" });
}

function renderEdit(route = "/oc/oc-1/edit") {
    return renderWithProviders(<CreateOCPage mode="edit" />, { route, path: "/oc/:id/edit" });
}

function nameBox(): HTMLElement {
    return screen.getByLabelText(/Name/);
}

function seriesSelect(): HTMLElement {
    return screen.getByRole("combobox");
}

function fileInputs(container: HTMLElement): HTMLInputElement[] {
    return [...container.querySelectorAll<HTMLInputElement>('input[type="file"]')];
}

function imageFile(name = "portrait.png"): File {
    return new File(["butterflies"], name, { type: "image/png" });
}

beforeEach(() => {
    mocks.createOC.mockResolvedValue({ id: "oc-9" });
    mocks.updateOC.mockResolvedValue({});
    mocks.deleteOC.mockResolvedValue({});
    mocks.uploadImage.mockResolvedValue({});
    mocks.addGalleryImage.mockResolvedValue({});
    mocks.deleteGalleryImage.mockResolvedValue({});
    stubOC();
});

describe("CreateOCPage in create mode", () => {
    it("titles itself as a new character and offers no delete", () => {
        // given
        stubOC();

        // when
        renderCreate();

        // then
        expect(screen.getByRole("heading", { name: "New OC" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Create OC" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete OC" })).not.toBeInTheDocument();
    });

    it("starts on an empty form pinned to Umineko", () => {
        // given
        stubOC();

        // when
        renderCreate();

        // then
        expect(nameBox()).toHaveValue("");
        expect(seriesSelect()).toHaveValue("umineko");
    });

    it("never asks for the character while creating", () => {
        // given
        stubOC();

        // when
        renderCreate();

        // then
        expect(mocks.useOC).toHaveBeenCalledWith("");
    });

    it("refuses to save a character with no name", async () => {
        // given
        const user = userEvent.setup();
        renderCreate();

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        expect(await screen.findByText("Name is required")).toBeInTheDocument();
        expect(mocks.createOC).not.toHaveBeenCalled();
    });

    it("treats a name of only whitespace as no name at all", async () => {
        // given
        const user = userEvent.setup();
        renderCreate();
        await user.type(nameBox(), "    ");

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        expect(await screen.findByText("Name is required")).toBeInTheDocument();
    });

    it("demands a universe name when the series is custom", async () => {
        // given
        const user = userEvent.setup();
        renderCreate();
        await user.type(nameBox(), "Featherine Junior");
        await user.selectOptions(seriesSelect(), "custom");

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        expect(await screen.findByText("Custom series name is required when series is custom")).toBeInTheDocument();
        expect(mocks.createOC).not.toHaveBeenCalled();
    });

    it("sends the trimmed name and description with the chosen series", async () => {
        // given
        const user = userEvent.setup();
        renderCreate();
        await user.type(nameBox(), "  Featherine Junior  ");
        await user.type(screen.getByPlaceholderText(/Tell us about this OC/), "  a witch in training  ");
        await user.selectOptions(seriesSelect(), "ciconia");

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        expect(mocks.createOC).toHaveBeenCalledWith({
            name: "Featherine Junior",
            description: "a witch in training",
            series: "ciconia",
            custom_series_name: "",
        });
    });

    it("sends the trimmed universe name for a custom series", async () => {
        // given
        const user = userEvent.setup();
        renderCreate();
        await user.type(nameBox(), "Featherine Junior");
        await user.selectOptions(seriesSelect(), "custom");
        await user.type(screen.getByLabelText(/Custom series name/), "  Rose Guns Days  ");

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        expect(mocks.createOC).toHaveBeenCalledWith(
            expect.objectContaining({ series: "custom", custom_series_name: "Rose Guns Days" }),
        );
    });

    it("opens the freshly created character once it is saved", async () => {
        // given
        const user = userEvent.setup();
        renderCreate();
        await user.type(nameBox(), "Featherine Junior");

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/oc/oc-9");
        });
    });

    it("reports why the character could not be saved", async () => {
        // given
        mocks.createOC.mockRejectedValue(new Error("that name is taken"));
        const user = userEvent.setup();
        renderCreate();
        await user.type(nameBox(), "Featherine Junior");

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        expect(await screen.findByText("that name is taken")).toBeInTheDocument();
        expect(mocks.navigate).not.toHaveBeenCalled();
    });

    it("falls back to a generic message when the failure carries no reason", async () => {
        // given
        mocks.createOC.mockRejectedValue("boom");
        const user = userEvent.setup();
        renderCreate();
        await user.type(nameBox(), "Featherine Junior");

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        expect(await screen.findByText("Failed to save oc")).toBeInTheDocument();
    });
});

describe("CreateOCPage main image", () => {
    it("uploads the chosen portrait against the new character", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderCreate();
        await user.type(nameBox(), "Featherine Junior");
        const file = imageFile();
        await user.upload(fileInputs(container)[0], file);

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        await waitFor(() => {
            expect(mocks.uploadImage).toHaveBeenCalledWith({ id: "oc-9", file });
        });
    });

    it("uploads nothing when no portrait was chosen", async () => {
        // given
        const user = userEvent.setup();
        renderCreate();
        await user.type(nameBox(), "Featherine Junior");

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/oc/oc-9");
        });
        expect(mocks.uploadImage).not.toHaveBeenCalled();
    });

    it("still opens the character when the portrait upload fails", async () => {
        // given
        mocks.uploadImage.mockRejectedValue(new Error("the disk is full"));
        const user = userEvent.setup();
        const { container } = renderCreate();
        await user.type(nameBox(), "Featherine Junior");
        await user.upload(fileInputs(container)[0], imageFile());

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/oc/oc-9");
        });
    });

    it("offers to replace the portrait once one has been chosen", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderCreate();

        // when
        await user.upload(fileInputs(container)[0], imageFile());

        // then
        expect(screen.getByRole("button", { name: "Replace" })).toBeInTheDocument();
        expect(screen.getByRole("img", { name: "preview" })).toBeInTheDocument();
    });

    it("clears the preview when the portrait is dropped", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderCreate();
        await user.upload(fileInputs(container)[0], imageFile());

        // when
        await user.click(screen.getAllByRole("button", { name: "Remove" })[0]);

        // then
        expect(screen.queryByRole("img", { name: "preview" })).not.toBeInTheDocument();
    });
});

describe("CreateOCPage gallery staging", () => {
    it("keeps the add button shut until an image has been picked", () => {
        // given
        stubOC();

        // when
        renderCreate();

        // then
        expect(screen.getByRole("button", { name: "Add to gallery" })).toBeDisabled();
    });

    it("names the picked gallery file before it is staged", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderCreate();

        // when
        await user.upload(fileInputs(container)[1], imageFile("rose-garden.png"));

        // then
        expect(screen.getByRole("button", { name: "Selected: rose-garden.png" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Add to gallery" })).toBeEnabled();
    });

    it("stages a gallery image with its caption and clears the picker", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderCreate();
        await user.upload(fileInputs(container)[1], imageFile("rose-garden.png"));
        await user.type(screen.getByPlaceholderText("Optional caption"), "in the rose garden");

        // when
        await user.click(screen.getByRole("button", { name: "Add to gallery" }));

        // then
        expect(screen.getByText("in the rose garden - pending upload")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Optional caption")).toHaveValue("");
        expect(screen.getByRole("button", { name: "Add to gallery" })).toBeDisabled();
    });

    it("marks a staged image with no caption as such", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderCreate();

        // when
        await user.upload(fileInputs(container)[1], imageFile("rose-garden.png"));
        await user.click(screen.getByRole("button", { name: "Add to gallery" }));

        // then
        expect(screen.getByText("(no caption) - pending upload")).toBeInTheDocument();
    });

    it("unstages a gallery image again", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderCreate();
        await user.upload(fileInputs(container)[1], imageFile("rose-garden.png"));
        await user.click(screen.getByRole("button", { name: "Add to gallery" }));

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(screen.queryByText("(no caption) - pending upload")).not.toBeInTheDocument();
    });

    it("uploads every staged gallery image against the new character", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderCreate();
        await user.type(nameBox(), "Featherine Junior");
        const first = imageFile("one.png");
        await user.upload(fileInputs(container)[1], first);
        await user.type(screen.getByPlaceholderText("Optional caption"), "first");
        await user.click(screen.getByRole("button", { name: "Add to gallery" }));
        const second = imageFile("two.png");
        await user.upload(fileInputs(container)[1], second);
        await user.click(screen.getByRole("button", { name: "Add to gallery" }));

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        await waitFor(() => {
            expect(mocks.addGalleryImage).toHaveBeenCalledTimes(2);
        });
        expect(mocks.addGalleryImage).toHaveBeenNthCalledWith(1, { id: "oc-9", file: first, caption: "first" });
        expect(mocks.addGalleryImage).toHaveBeenNthCalledWith(2, { id: "oc-9", file: second, caption: "" });
    });

    it("still opens the character when a gallery upload fails", async () => {
        // given
        mocks.addGalleryImage.mockRejectedValue(new Error("the disk is full"));
        const user = userEvent.setup();
        const { container } = renderCreate();
        await user.type(nameBox(), "Featherine Junior");
        await user.upload(fileInputs(container)[1], imageFile("one.png"));
        await user.click(screen.getByRole("button", { name: "Add to gallery" }));

        // when
        await user.click(screen.getByRole("button", { name: "Create OC" }));

        // then
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/oc/oc-9");
        });
    });
});

describe("CreateOCPage in edit mode", () => {
    it("says it is loading while the character is on its way", () => {
        // given
        stubOC(null, true);

        // when
        renderEdit();

        // then
        expect(screen.getByText("Loading OC...")).toBeInTheDocument();
    });

    it("says the character could not be found once the fetch has settled", () => {
        // given
        stubOC(null);

        // when
        renderEdit();

        // then
        expect(screen.getByText("OC not found.")).toBeInTheDocument();
    });

    it("fills the form with the character as it stands", () => {
        // given
        stubOC(makeOC());

        // when
        renderEdit();

        // then
        expect(screen.getByRole("heading", { name: "Edit OC" })).toBeInTheDocument();
        expect(nameBox()).toHaveValue("Featherine Junior");
        expect(seriesSelect()).toHaveValue("higurashi");
        expect(screen.getByPlaceholderText(/Tell us about this OC/)).toHaveValue("a witch in training");
    });

    it("shows the universe name for a character in a custom universe", () => {
        // given
        stubOC(makeOC({ series: "custom", custom_series_name: "Rose Guns Days" }));

        // when
        renderEdit();

        // then
        expect(screen.getByLabelText(/Custom series name/)).toHaveValue("Rose Guns Days");
    });

    it("updates the character instead of creating a new one", async () => {
        // given
        stubOC(makeOC());
        const user = userEvent.setup();
        renderEdit();
        await user.clear(nameBox());
        await user.type(nameBox(), "Featherine Senior");

        // when
        await user.click(screen.getByRole("button", { name: "Save changes" }));

        // then
        expect(mocks.updateOC).toHaveBeenCalledWith({
            name: "Featherine Senior",
            description: "a witch in training",
            series: "higurashi",
            custom_series_name: "",
        });
        expect(mocks.createOC).not.toHaveBeenCalled();
    });

    it("returns to the character it just saved", async () => {
        // given
        stubOC(makeOC());
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByRole("button", { name: "Save changes" }));

        // then
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/oc/oc-1");
        });
    });

    it("shows the images the character already has", () => {
        // given
        stubOC(makeOC({ gallery: [makeImage({ caption: "in the rose garden" })] }));

        // when
        renderEdit();

        // then
        expect(screen.getByRole("img", { name: "in the rose garden" })).toHaveAttribute("src", "/gallery-thumb.png");
    });

    it("deletes an existing gallery image that was marked for removal", async () => {
        // given
        stubOC(makeOC({ gallery: [makeImage({ id: 11, caption: "in the rose garden" })] }));
        const user = userEvent.setup();
        renderEdit();
        await user.click(screen.getByRole("button", { name: "Remove" }));
        expect(screen.queryByRole("img", { name: "in the rose garden" })).not.toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "Save changes" }));

        // then
        await waitFor(() => {
            expect(mocks.deleteGalleryImage).toHaveBeenCalledWith({ ocId: "oc-1", imageId: 11 });
        });
    });

    it("asks before deleting the character", async () => {
        // given
        stubOC(makeOC());
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByRole("button", { name: "Delete OC" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this OC permanently? This cannot be undone.");
        expect(mocks.deleteOC).not.toHaveBeenCalled();
    });

    it("deletes the character and returns to the list once confirmed", async () => {
        // given
        stubOC(makeOC());
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByRole("button", { name: "Delete OC" }));

        // then
        expect(mocks.deleteOC).toHaveBeenCalledWith("oc-1");
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/oc");
        });
    });

    it("reports why the character could not be deleted", async () => {
        // given
        mocks.deleteOC.mockRejectedValue(new Error("that character is referenced elsewhere"));
        stubOC(makeOC());
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByRole("button", { name: "Delete OC" }));

        // then
        expect(await screen.findByText("that character is referenced elsewhere")).toBeInTheDocument();
        expect(mocks.navigate).not.toHaveBeenCalled();
    });
});
