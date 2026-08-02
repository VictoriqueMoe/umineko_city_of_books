import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { Lightbox } from "./Lightbox";

const SRC = "https://waifuvault.moe/f/beatrice.png";

function noop() {}

afterEach(() => {
    document.body.style.overflow = "";
});

describe("Lightbox", () => {
    it("presents the image in a modal dialog", () => {
        // given
        const alt = "the golden truth";

        // when
        renderWithProviders(<Lightbox src={SRC} alt={alt} onClose={noop} />);

        // then
        const dialog = screen.getByRole("dialog");
        expect(dialog).toHaveAttribute("aria-modal", "true");
        expect(screen.getByAltText(alt)).toHaveAttribute("src", SRC);
    });

    it("closes when escape is pressed", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Lightbox src={SRC} onClose={onClose} />);

        // when
        await user.keyboard("{Escape}");

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("ignores keys other than escape", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Lightbox src={SRC} onClose={onClose} />);

        // when
        await user.keyboard("{Enter}{ArrowLeft}{ArrowRight} x");

        // then
        expect(onClose).not.toHaveBeenCalled();
    });

    it("closes when the backdrop is clicked", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Lightbox src={SRC} onClose={onClose} />);

        // when
        await user.click(screen.getByRole("dialog"));

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("stays open when the image itself is clicked", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Lightbox src={SRC} alt="the golden truth" onClose={onClose} />);

        // when
        await user.click(screen.getByAltText("the golden truth"));

        // then
        expect(onClose).not.toHaveBeenCalled();
    });

    it("closes from the close control", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Lightbox src={SRC} onClose={onClose} />);

        // when
        await user.click(screen.getByRole("button", { name: "Close" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("keeps the close press from reaching the backdrop behind it", async () => {
        // given
        const onBackdropClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <div onClick={onBackdropClick}>
                <Lightbox src={SRC} onClose={noop} />
            </div>,
        );

        // when
        await user.click(screen.getByRole("button", { name: "Close" }));

        // then
        expect(onBackdropClick).not.toHaveBeenCalled();
    });

    it("locks the page behind it and restores the previous scrolling on close", () => {
        // given
        document.body.style.overflow = "scroll";
        const { unmount } = renderWithProviders(<Lightbox src={SRC} onClose={noop} />);
        expect(document.body.style.overflow).toBe("hidden");

        // when
        unmount();

        // then
        expect(document.body.style.overflow).toBe("scroll");
    });

    it("stops listening for escape once it has closed", () => {
        // given
        const onClose = vi.fn();
        const { unmount } = renderWithProviders(<Lightbox src={SRC} onClose={onClose} />);

        // when
        unmount();
        fireEvent.keyDown(window, { key: "Escape" });

        // then
        expect(onClose).not.toHaveBeenCalled();
    });
});
