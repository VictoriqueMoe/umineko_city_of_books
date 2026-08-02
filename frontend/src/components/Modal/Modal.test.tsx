import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { Modal } from "./Modal";

const closeLabel = "✕";

function noop() {}

describe("Modal", () => {
    it("renders nothing at all while it is closed", () => {
        // given
        const isOpen = false;

        // when
        const { container } = renderWithProviders(
            <Modal isOpen={isOpen} onClose={noop} title="Seal the letter">
                <p>hidden body</p>
            </Modal>,
        );

        // then
        expect(container).toBeEmptyDOMElement();
        expect(screen.queryByText("Seal the letter")).not.toBeInTheDocument();
    });

    it("shows its title and its children once it is open", () => {
        // given
        const isOpen = true;

        // when
        renderWithProviders(
            <Modal isOpen={isOpen} onClose={noop} title="Seal the letter">
                <p>the witch is waiting</p>
            </Modal>,
        );

        // then
        expect(screen.getByRole("heading", { name: "Seal the letter" })).toBeInTheDocument();
        expect(screen.getByText("the witch is waiting")).toBeInTheDocument();
    });

    it("closes when the close control is pressed", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <Modal isOpen onClose={onClose} title="Seal the letter">
                <p>the witch is waiting</p>
            </Modal>,
        );

        // when
        await user.click(screen.getByRole("button", { name: closeLabel }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("closes when the backdrop behind it is clicked", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        const { container } = renderWithProviders(
            <Modal isOpen onClose={onClose} title="Seal the letter">
                <p>the witch is waiting</p>
            </Modal>,
        );

        // when
        await user.click(container.firstElementChild as HTMLElement);

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("stays open when the panel itself is clicked", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <Modal isOpen onClose={onClose} title="Seal the letter">
                <p>the witch is waiting</p>
            </Modal>,
        );

        // when
        await user.click(screen.getByText("the witch is waiting"));

        // then
        expect(onClose).not.toHaveBeenCalled();
    });

    it("hides its whole subtree again when it is closed after being open", () => {
        // given
        const onClose = vi.fn();
        const { rerender } = renderWithProviders(
            <Modal isOpen onClose={onClose} title="Seal the letter">
                <p>the witch is waiting</p>
            </Modal>,
        );

        // when
        rerender(
            <Modal isOpen={false} onClose={onClose} title="Seal the letter">
                <p>the witch is waiting</p>
            </Modal>,
        );

        // then
        expect(screen.queryByText("the witch is waiting")).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: closeLabel })).not.toBeInTheDocument();
    });

    it("puts the close control in the tab order ahead of its own content", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(
            <Modal isOpen onClose={noop} title="Seal the letter">
                <button type="button">Confirm</button>
            </Modal>,
        );

        // when
        await user.tab();

        // then
        expect(screen.getByRole("button", { name: closeLabel })).toHaveFocus();
    });

    it("announces itself as a modal dialog named after its title", () => {
        // given
        const title = "Seal the letter";

        // when
        renderWithProviders(
            <Modal isOpen onClose={noop} title={title}>
                <p>the witch is waiting</p>
            </Modal>,
        );

        // then
        const dialog = screen.getByRole("dialog", { name: title });
        expect(dialog).toHaveAttribute("aria-modal", "true");
        expect(dialog).toContainElement(screen.getByRole("heading", { name: title }));
    });

    it("moves focus into the dialog when it opens", () => {
        // given
        const onClose = vi.fn();
        const { rerender } = renderWithProviders(
            <>
                <button type="button">Open the letter</button>
                <Modal isOpen={false} onClose={onClose} title="Seal the letter">
                    <p>the witch is waiting</p>
                </Modal>
            </>,
        );
        screen.getByRole("button", { name: "Open the letter" }).focus();

        // when
        rerender(
            <>
                <button type="button">Open the letter</button>
                <Modal isOpen onClose={onClose} title="Seal the letter">
                    <p>the witch is waiting</p>
                </Modal>
            </>,
        );

        // then
        expect(screen.getByRole("dialog")).toHaveFocus();
    });

    it("gives focus back to whatever held it once it closes", () => {
        // given
        const onClose = vi.fn();
        const { rerender } = renderWithProviders(
            <>
                <button type="button">Open the letter</button>
                <Modal isOpen={false} onClose={onClose} title="Seal the letter">
                    <p>the witch is waiting</p>
                </Modal>
            </>,
        );
        const trigger = screen.getByRole("button", { name: "Open the letter" });
        trigger.focus();
        rerender(
            <>
                <button type="button">Open the letter</button>
                <Modal isOpen onClose={onClose} title="Seal the letter">
                    <p>the witch is waiting</p>
                </Modal>
            </>,
        );
        screen.getByRole("button", { name: closeLabel }).focus();

        // when
        rerender(
            <>
                <button type="button">Open the letter</button>
                <Modal isOpen={false} onClose={onClose} title="Seal the letter">
                    <p>the witch is waiting</p>
                </Modal>
            </>,
        );

        // then
        expect(trigger).toHaveFocus();
    });

    it("closes when the escape key is pressed", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <Modal isOpen onClose={onClose} title="Seal the letter">
                <p>the witch is waiting</p>
            </Modal>,
        );

        // when
        await user.keyboard("{Escape}");

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("stops listening for escape once it has closed", () => {
        // given
        const onClose = vi.fn();
        const { rerender } = renderWithProviders(
            <Modal isOpen onClose={onClose} title="Seal the letter">
                <p>the witch is waiting</p>
            </Modal>,
        );

        // when
        rerender(
            <Modal isOpen={false} onClose={onClose} title="Seal the letter">
                <p>the witch is waiting</p>
            </Modal>,
        );
        fireEvent.keyDown(document, { key: "Escape" });

        // then
        expect(onClose).not.toHaveBeenCalled();
    });

    it("does not submit a form it is rendered inside when the close control is pressed", async () => {
        // given
        const onSubmit = vi.fn((e: React.FormEvent) => e.preventDefault());
        const user = userEvent.setup();
        renderWithProviders(
            <form onSubmit={onSubmit}>
                <Modal isOpen onClose={noop} title="Seal the letter">
                    <p>the witch is waiting</p>
                </Modal>
            </form>,
        );

        // when
        await user.click(screen.getByRole("button", { name: closeLabel }));

        // then
        expect(onSubmit).not.toHaveBeenCalled();
    });
});
