import { StrictMode } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { MediaPickerButton, MediaPreviews } from "./MediaPicker";

const MB = 1024 * 1024;

const limits = { max_image_size: MB, max_video_size: 2 * MB };

function makeFile(name: string, type: string, size = 8): File {
    const file = new File(["beatrice"], name, { type });
    Object.defineProperty(file, "size", { value: size });

    return file;
}

function previewOf(removeButton: HTMLElement): HTMLElement {
    const preview = removeButton.parentElement;
    if (!preview) {
        throw new Error("expected the remove control to sit inside a preview");
    }

    return preview;
}

function previews(): HTMLElement[] {
    return screen.getAllByRole("button", { name: "Remove" }).map(previewOf);
}

function fileInput(container: HTMLElement): HTMLInputElement {
    const input = container.querySelector<HTMLInputElement>("input[type='file']");
    if (!input) {
        throw new Error("expected a file input to be rendered");
    }

    return input;
}

function noop() {}

describe("MediaPreviews", () => {
    let revoked = vi.fn();

    beforeEach(() => {
        revoked = vi.fn();
        let issued = 0;
        vi.spyOn(URL, "createObjectURL").mockImplementation(() => `blob:preview-${++issued}`);
        vi.spyOn(URL, "revokeObjectURL").mockImplementation(revoked);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("renders nothing when nothing has been picked", () => {
        // given
        const files: File[] = [];

        // when
        const { container } = renderWithProviders(<MediaPreviews files={files} onRemove={noop} />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("previews images as pictures and videos as players", () => {
        // given
        const files = [makeFile("beatrice.png", "image/png"), makeFile("clip.mp4", "video/mp4")];

        // when
        const { container } = renderWithProviders(<MediaPreviews files={files} onRemove={noop} />);

        // then
        expect(container.querySelectorAll("img")).toHaveLength(1);
        expect(container.querySelectorAll("video")).toHaveLength(1);
        expect(screen.getAllByRole("button", { name: "Remove" })).toHaveLength(2);
    });

    it("removes the file whose remove control was pressed", async () => {
        // given
        const onRemove = vi.fn();
        const user = userEvent.setup();
        const files = [makeFile("a.png", "image/png"), makeFile("b.png", "image/png"), makeFile("c.png", "image/png")];
        renderWithProviders(<MediaPreviews files={files} onRemove={onRemove} />);

        // when
        await user.click(screen.getAllByRole("button", { name: "Remove" })[1]);

        // then
        expect(onRemove).toHaveBeenCalledWith(1);
    });

    it("offers no reordering for a lone file", () => {
        // given
        const files = [makeFile("a.png", "image/png")];

        // when
        renderWithProviders(<MediaPreviews files={files} onRemove={noop} onReorder={noop} />);

        // then
        expect(screen.queryByRole("button", { name: "Move earlier" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Move later" })).not.toBeInTheDocument();
    });

    it("offers no reordering when the caller cannot handle it", () => {
        // given
        const files = [makeFile("a.png", "image/png"), makeFile("b.png", "image/png")];

        // when
        renderWithProviders(<MediaPreviews files={files} onRemove={noop} />);

        // then
        expect(screen.queryByRole("button", { name: "Move earlier" })).not.toBeInTheDocument();
    });

    it("disables moving the first file earlier and the last file later", () => {
        // given
        const files = [makeFile("a.png", "image/png"), makeFile("b.png", "image/png"), makeFile("c.png", "image/png")];

        // when
        renderWithProviders(<MediaPreviews files={files} onRemove={noop} onReorder={noop} />);

        // then
        const earlier = screen.getAllByRole("button", { name: "Move earlier" });
        const later = screen.getAllByRole("button", { name: "Move later" });
        expect(earlier[0]).toBeDisabled();
        expect(earlier[1]).toBeEnabled();
        expect(later[2]).toBeDisabled();
        expect(later[1]).toBeEnabled();
    });

    it("moves a file one place earlier", async () => {
        // given
        const onReorder = vi.fn();
        const user = userEvent.setup();
        const files = [makeFile("a.png", "image/png"), makeFile("b.png", "image/png")];
        renderWithProviders(<MediaPreviews files={files} onRemove={noop} onReorder={onReorder} />);

        // when
        await user.click(screen.getAllByRole("button", { name: "Move earlier" })[1]);

        // then
        expect(onReorder).toHaveBeenCalledWith(1, 0);
    });

    it("moves a file one place later", async () => {
        // given
        const onReorder = vi.fn();
        const user = userEvent.setup();
        const files = [makeFile("a.png", "image/png"), makeFile("b.png", "image/png")];
        renderWithProviders(<MediaPreviews files={files} onRemove={noop} onReorder={onReorder} />);

        // when
        await user.click(screen.getAllByRole("button", { name: "Move later" })[0]);

        // then
        expect(onReorder).toHaveBeenCalledWith(0, 1);
    });

    it("reorders when one preview is dragged onto another", () => {
        // given
        const onReorder = vi.fn();
        const files = [makeFile("a.png", "image/png"), makeFile("b.png", "image/png"), makeFile("c.png", "image/png")];
        renderWithProviders(<MediaPreviews files={files} onRemove={noop} onReorder={onReorder} />);
        const items = previews();

        // when
        fireEvent.dragStart(items[0]);
        fireEvent.dragOver(items[2]);
        fireEvent.drop(items[2]);

        // then
        expect(onReorder).toHaveBeenCalledWith(0, 2);
    });

    it("ignores a preview dropped back onto itself", () => {
        // given
        const onReorder = vi.fn();
        const files = [makeFile("a.png", "image/png"), makeFile("b.png", "image/png")];
        renderWithProviders(<MediaPreviews files={files} onRemove={noop} onReorder={onReorder} />);
        const items = previews();

        // when
        fireEvent.dragStart(items[1]);
        fireEvent.drop(items[1]);

        // then
        expect(onReorder).not.toHaveBeenCalled();
    });

    it("ignores a drop that no drag started", () => {
        // given
        const onReorder = vi.fn();
        const files = [makeFile("a.png", "image/png"), makeFile("b.png", "image/png")];
        renderWithProviders(<MediaPreviews files={files} onRemove={noop} onReorder={onReorder} />);
        const items = previews();

        // when
        fireEvent.drop(items[0]);

        // then
        expect(onReorder).not.toHaveBeenCalled();
    });

    it("releases the preview url of a picture that was taken back off", () => {
        // given
        const a = makeFile("a.png", "image/png");
        const b = makeFile("b.png", "image/png");
        const { rerender } = renderWithProviders(<MediaPreviews files={[a, b]} onRemove={noop} />);
        const before = screen.getAllByRole("presentation").map(img => img.getAttribute("src"));
        revoked.mockClear();

        // when the second picture is dropped from the list
        rerender(<MediaPreviews files={[a]} onRemove={noop} />);

        // then only the urls that nothing points at any more are released
        const after = screen.getAllByRole("presentation").map(img => img.getAttribute("src"));
        for (const url of before) {
            if (!after.includes(url)) {
                expect(revoked).toHaveBeenCalledWith(url);
            }
        }
        for (const url of after) {
            expect(revoked).not.toHaveBeenCalledWith(url);
        }
    });

    it("keeps a staged picture visible when effects are torn down and set up again", () => {
        // given StrictMode, which mounts effects, tears them down, then mounts them again
        const files = [makeFile("a.png", "image/png")];

        // when rendered without the test providers, which suppress that second pass
        render(
            <StrictMode>
                <MediaPreviews files={files} onRemove={noop} />
            </StrictMode>,
        );

        // then the url the img still points at has not been revoked underneath it
        const src = screen.getByRole("presentation").getAttribute("src");
        expect(src).toBeTruthy();
        expect(revoked).not.toHaveBeenCalledWith(src);
    });

    it("shows a note for a staged audio file rather than a broken picture", () => {
        // given an audio attachment, which no browser can render as an image
        const files = [makeFile("theme.mp3", "audio/mpeg")];

        // when
        renderWithProviders(<MediaPreviews files={files} onRemove={noop} />);

        // then
        expect(screen.getByLabelText("Audio file")).toBeInTheDocument();
        expect(screen.queryByRole("presentation")).not.toBeInTheDocument();
    });

    it("still shows a picture preview for an image", () => {
        // given
        const files = [makeFile("a.png", "image/png")];

        // when
        renderWithProviders(<MediaPreviews files={files} onRemove={noop} />);

        // then
        expect(screen.getByRole("presentation")).toBeInTheDocument();
        expect(screen.queryByLabelText("Audio file")).not.toBeInTheDocument();
    });
});

describe("MediaPickerButton", () => {
    it("labels the picker as media by default", () => {
        // given
        const onFiles = vi.fn();

        // when
        renderWithProviders(<MediaPickerButton onFiles={onFiles} />, { siteInfo: limits });

        // then
        expect(screen.getByRole("button", { name: "+ Media" })).toBeInTheDocument();
    });

    it("uses the label it was given", () => {
        // given
        const label = "Attach evidence";

        // when
        renderWithProviders(<MediaPickerButton onFiles={noop} label={label} />, { siteInfo: limits });

        // then
        expect(screen.getByRole("button", { name: label })).toBeInTheDocument();
    });

    it("accepts images, videos and audio, and takes several at once", () => {
        // given
        const onFiles = vi.fn();

        // when
        const { container } = renderWithProviders(<MediaPickerButton onFiles={onFiles} />, { siteInfo: limits });

        // then
        const input = fileInput(container);
        expect(input).toHaveAttribute("accept", "image/*,video/*,audio/*,.mkv,.avi");
        expect(input).toHaveAttribute("multiple");
    });

    it("takes a single file when told not to allow several", () => {
        // given
        const multiple = false;

        // when
        const { container } = renderWithProviders(<MediaPickerButton onFiles={noop} multiple={multiple} />, {
            siteInfo: limits,
        });

        // then
        expect(fileInput(container)).not.toHaveAttribute("multiple");
    });

    it("opens the hidden file chooser when the button is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<MediaPickerButton onFiles={noop} />, { siteInfo: limits });
        const open = vi.spyOn(fileInput(container), "click").mockImplementation(noop);

        // when
        await user.click(screen.getByRole("button", { name: "+ Media" }));

        // then
        expect(open).toHaveBeenCalledOnce();
    });

    it("hands over a file that is within the limits", async () => {
        // given
        const onFiles = vi.fn();
        const user = userEvent.setup();
        const file = makeFile("beatrice.png", "image/png", 512 * 1024);
        const { container } = renderWithProviders(<MediaPickerButton onFiles={onFiles} />, { siteInfo: limits });

        // when
        await user.upload(fileInput(container), file);

        // then
        expect(onFiles).toHaveBeenCalledWith([file]);
    });

    it("refuses an image that is over the image limit and says how big it was", async () => {
        // given
        const onFiles = vi.fn();
        const onError = vi.fn();
        const user = userEvent.setup();
        const file = makeFile("beatrice.png", "image/png", 2 * MB);
        const { container } = renderWithProviders(<MediaPickerButton onFiles={onFiles} onError={onError} />, {
            siteInfo: limits,
        });

        // when
        await user.upload(fileInput(container), file);

        // then
        expect(onError).toHaveBeenCalledWith("beatrice.png is too large (2.0 MB). Maximum image size is 1.0 MB.");
        expect(onFiles).not.toHaveBeenCalled();
    });

    it("measures a video against the larger video limit", async () => {
        // given
        const onFiles = vi.fn();
        const onError = vi.fn();
        const user = userEvent.setup();
        const withinVideoLimit = makeFile("clip.mp4", "video/mp4", 1.5 * MB);
        const { container } = renderWithProviders(<MediaPickerButton onFiles={onFiles} onError={onError} />, {
            siteInfo: limits,
        });

        // when
        await user.upload(fileInput(container), withinVideoLimit);

        // then
        expect(onError).not.toHaveBeenCalled();
        expect(onFiles).toHaveBeenCalledWith([withinVideoLimit]);
    });

    it("refuses a video that is over the video limit", async () => {
        // given
        const onError = vi.fn();
        const user = userEvent.setup();
        const file = makeFile("clip.mp4", "video/mp4", 3 * MB);
        const { container } = renderWithProviders(<MediaPickerButton onFiles={noop} onError={onError} />, {
            siteInfo: limits,
        });

        // when
        await user.upload(fileInput(container), file);

        // then
        expect(onError).toHaveBeenCalledWith("clip.mp4 is too large (3.0 MB). Maximum video size is 2.0 MB.");
    });

    it("keeps the acceptable files and reports every rejection together", async () => {
        // given
        const onFiles = vi.fn();
        const onError = vi.fn();
        const user = userEvent.setup();
        const good = makeFile("good.png", "image/png", 128 * 1024);
        const tooBig = makeFile("huge.png", "image/png", 4 * MB);
        const alsoTooBig = makeFile("massive.png", "image/png", 8 * MB);
        const { container } = renderWithProviders(<MediaPickerButton onFiles={onFiles} onError={onError} />, {
            siteInfo: limits,
        });

        // when
        await user.upload(fileInput(container), [good, tooBig, alsoTooBig]);

        // then
        expect(onFiles).toHaveBeenCalledWith([good]);
        expect(onError).toHaveBeenCalledWith(
            "huge.png is too large (4.0 MB). Maximum image size is 1.0 MB. massive.png is too large (8.0 MB). Maximum image size is 1.0 MB.",
        );
    });

    it("survives a rejection when no error handler was supplied", async () => {
        // given
        const onFiles = vi.fn();
        const user = userEvent.setup();
        const file = makeFile("beatrice.png", "image/png", 2 * MB);
        const { container } = renderWithProviders(<MediaPickerButton onFiles={onFiles} />, { siteInfo: limits });

        // when
        await user.upload(fileInput(container), file);

        // then
        expect(onFiles).not.toHaveBeenCalled();
    });

    it("clears the input so the same file can be picked twice", async () => {
        // given
        const onFiles = vi.fn();
        const user = userEvent.setup();
        const file = makeFile("beatrice.png", "image/png", 128 * 1024);
        const { container } = renderWithProviders(<MediaPickerButton onFiles={onFiles} />, { siteInfo: limits });

        // when
        await user.upload(fileInput(container), file);

        // then
        expect(fileInput(container).value).toBe("");
    });
});
