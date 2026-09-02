import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { FormEvent } from "react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { Button } from "../Button/Button";
import { ConfirmDialog } from "./ConfirmDialog";

const closeLabel = "✕";

function noop() {}

describe("ConfirmDialog", () => {
    it("renders nothing at all while it is closed", () => {
        // given
        const open = false;

        // when
        const { container } = renderWithProviders(
            <ConfirmDialog
                open={open}
                title="Confirm Delete"
                body="This action cannot be undone."
                onConfirm={noop}
                onCancel={noop}
            />,
        );

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("shows its title and its body once it is open", () => {
        // given
        const title = "Confirm Delete";

        // when
        renderWithProviders(
            <ConfirmDialog
                open
                title={title}
                body={
                    <>
                        Are you sure you want to delete <strong>Beatrice</strong>? This action cannot be undone.
                    </>
                }
                onConfirm={noop}
                onCancel={noop}
            />,
        );

        // then
        expect(screen.getByRole("dialog", { name: title })).toBeInTheDocument();
        expect(screen.getByText(/Are you sure you want to delete/)).toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
    });

    it("defaults its two labels to Cancel and Confirm", () => {
        // given
        const open = true;

        // when
        renderWithProviders(
            <ConfirmDialog open={open} title="Confirm Delete" body="body" onConfirm={noop} onCancel={noop} />,
        );

        // then
        expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Confirm" })).toBeInTheDocument();
    });

    it("uses the labels it was given", () => {
        // given
        const confirmLabel = "Delete";
        const cancelLabel = "Keep it";

        // when
        renderWithProviders(
            <ConfirmDialog
                open
                title="Confirm Delete"
                body="body"
                confirmLabel={confirmLabel}
                cancelLabel={cancelLabel}
                onConfirm={noop}
                onCancel={noop}
            />,
        );

        // then
        expect(screen.getByRole("button", { name: cancelLabel })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: confirmLabel })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Confirm" })).not.toBeInTheDocument();
    });

    it("puts cancel ahead of confirm, matching the dialogues it replaces", () => {
        // given
        const open = true;

        // when
        renderWithProviders(
            <ConfirmDialog
                open={open}
                title="Confirm Delete"
                body="body"
                confirmLabel="Delete"
                onConfirm={noop}
                onCancel={noop}
            />,
        );

        // then
        const cancel = screen.getByRole("button", { name: "Cancel" });
        const confirm = screen.getByRole("button", { name: "Delete" });
        expect(cancel.parentElement).toBe(confirm.parentElement);
        expect(cancel.compareDocumentPosition(confirm)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    });

    it("confirms when the confirm control is pressed", async () => {
        // given
        const onConfirm = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <ConfirmDialog
                open
                title="Confirm Delete"
                body="body"
                confirmLabel="Delete"
                onConfirm={onConfirm}
                onCancel={noop}
            />,
        );

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(onConfirm).toHaveBeenCalledOnce();
    });

    it("cancels when the cancel control is pressed", async () => {
        // given
        const onCancel = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <ConfirmDialog open title="Confirm Delete" body="body" onConfirm={noop} onCancel={onCancel} />,
        );

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(onCancel).toHaveBeenCalledOnce();
    });

    it("cancels when the escape key is pressed", async () => {
        // given
        const onCancel = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <ConfirmDialog open title="Confirm Delete" body="body" onConfirm={noop} onCancel={onCancel} />,
        );

        // when
        await user.keyboard("{Escape}");

        // then
        expect(onCancel).toHaveBeenCalledOnce();
    });

    it("cancels when the backdrop behind it is clicked", async () => {
        // given
        const onCancel = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <ConfirmDialog open title="Confirm Delete" body="body" onConfirm={noop} onCancel={onCancel} />,
        );

        // when
        await user.click(screen.getByRole("dialog").parentElement as HTMLElement);

        // then
        expect(onCancel).toHaveBeenCalledOnce();
    });

    it("cancels when the close control is pressed", async () => {
        // given
        const onCancel = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <ConfirmDialog open title="Confirm Delete" body="body" onConfirm={noop} onCancel={onCancel} />,
        );

        // when
        await user.click(screen.getByRole("button", { name: closeLabel }));

        // then
        expect(onCancel).toHaveBeenCalledOnce();
    });

    it("gives the confirm control the danger affordance when the action is destructive", () => {
        // given
        const destructive = true;

        // when
        renderWithProviders(
            <>
                <Button variant="danger">reference</Button>
                <ConfirmDialog
                    open
                    title="Confirm Delete"
                    body="body"
                    confirmLabel="Delete"
                    destructive={destructive}
                    onConfirm={noop}
                    onCancel={noop}
                />
            </>,
        );

        // then
        expect(screen.getByRole("button", { name: "Delete" }).className).toBe(
            screen.getByRole("button", { name: "reference" }).className,
        );
    });

    it("gives the confirm control the primary affordance when the action is not destructive", () => {
        // given
        const destructive = false;

        // when
        renderWithProviders(
            <>
                <Button variant="primary">reference</Button>
                <ConfirmDialog
                    open
                    title="Leave the game"
                    body="body"
                    confirmLabel="Leave"
                    destructive={destructive}
                    onConfirm={noop}
                    onCancel={noop}
                />
            </>,
        );

        // then
        expect(screen.getByRole("button", { name: "Leave" }).className).toBe(
            screen.getByRole("button", { name: "reference" }).className,
        );
    });

    it("parks the opening focus on cancel when the action is destructive", () => {
        // given
        const destructive = true;

        // when
        renderWithProviders(
            <ConfirmDialog
                open
                title="Confirm Delete"
                body="body"
                confirmLabel="Delete"
                destructive={destructive}
                onConfirm={noop}
                onCancel={noop}
            />,
        );

        // then
        expect(screen.getByRole("button", { name: "Cancel" })).toHaveFocus();
    });

    it("parks the opening focus on confirm when the action is not destructive", () => {
        // given
        const destructive = false;

        // when
        renderWithProviders(
            <ConfirmDialog
                open
                title="Leave the game"
                body="body"
                confirmLabel="Leave"
                destructive={destructive}
                onConfirm={noop}
                onCancel={noop}
            />,
        );

        // then
        expect(screen.getByRole("button", { name: "Leave" })).toHaveFocus();
    });

    it("gives focus back to whatever held it once it closes", () => {
        // given
        const { rerender } = renderWithProviders(
            <>
                <button type="button">Delete User</button>
                <ConfirmDialog open={false} title="Confirm Delete" body="body" onConfirm={noop} onCancel={noop} />
            </>,
        );
        const trigger = screen.getByRole("button", { name: "Delete User" });
        trigger.focus();
        rerender(
            <>
                <button type="button">Delete User</button>
                <ConfirmDialog open title="Confirm Delete" body="body" onConfirm={noop} onCancel={noop} />
            </>,
        );

        // when
        rerender(
            <>
                <button type="button">Delete User</button>
                <ConfirmDialog open={false} title="Confirm Delete" body="body" onConfirm={noop} onCancel={noop} />
            </>,
        );

        // then
        expect(trigger).toHaveFocus();
    });

    it("keeps tab focus inside the dialog when it passes the last control", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(
            <>
                <button type="button">Delete User</button>
                <ConfirmDialog
                    open
                    title="Confirm Delete"
                    body="body"
                    confirmLabel="Delete"
                    onConfirm={noop}
                    onCancel={noop}
                />
            </>,
        );
        expect(screen.getByRole("button", { name: "Delete" })).toHaveFocus();

        // when
        await user.tab();

        // then
        expect(screen.getByRole("button", { name: closeLabel })).toHaveFocus();
    });

    it("wraps tab focus backwards from the first control to the last", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(
            <>
                <button type="button">Delete User</button>
                <ConfirmDialog
                    open
                    title="Confirm Delete"
                    body="body"
                    confirmLabel="Delete"
                    onConfirm={noop}
                    onCancel={noop}
                />
            </>,
        );
        screen.getByRole("button", { name: closeLabel }).focus();

        // when
        await user.tab({ shift: true });

        // then
        expect(screen.getByRole("button", { name: "Delete" })).toHaveFocus();
    });

    it("disables both controls while it is busy", () => {
        // given
        const busy = true;

        // when
        renderWithProviders(
            <ConfirmDialog
                open
                title="Confirm Delete"
                body="body"
                confirmLabel="Delete"
                busy={busy}
                onConfirm={noop}
                onCancel={noop}
            />,
        );

        // then
        expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Delete" })).toBeDisabled();
    });

    it("announces its controls as busy while it is busy", () => {
        // given
        const busy = true;

        // when
        renderWithProviders(
            <ConfirmDialog
                open
                title="Confirm Delete"
                body="body"
                confirmLabel="Delete"
                busy={busy}
                onConfirm={noop}
                onCancel={noop}
            />,
        );

        // then
        expect(screen.getByRole("button", { name: "Delete" }).parentElement).toHaveAttribute("aria-busy", "true");
    });

    it("does not confirm a second time once the first confirmation is in flight", async () => {
        // given
        const onConfirm = vi.fn();
        const user = userEvent.setup();
        const { rerender } = renderWithProviders(
            <ConfirmDialog
                open
                title="Confirm Delete"
                body="body"
                confirmLabel="Delete"
                onConfirm={onConfirm}
                onCancel={noop}
            />,
        );
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // when
        rerender(
            <ConfirmDialog
                open
                title="Confirm Delete"
                body="body"
                confirmLabel="Delete"
                busy
                onConfirm={onConfirm}
                onCancel={noop}
            />,
        );
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(onConfirm).toHaveBeenCalledOnce();
    });

    it("cannot be dismissed with the escape key while it is busy", async () => {
        // given
        const onCancel = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <ConfirmDialog open title="Confirm Delete" body="body" busy onConfirm={noop} onCancel={onCancel} />,
        );

        // when
        await user.keyboard("{Escape}");

        // then
        expect(onCancel).not.toHaveBeenCalled();
    });

    it("cannot be dismissed by the backdrop while it is busy", async () => {
        // given
        const onCancel = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <ConfirmDialog open title="Confirm Delete" body="body" busy onConfirm={noop} onCancel={onCancel} />,
        );

        // when
        await user.click(screen.getByRole("dialog").parentElement as HTMLElement);

        // then
        expect(onCancel).not.toHaveBeenCalled();
    });

    it("cannot be dismissed with the close control while it is busy", async () => {
        // given
        const onCancel = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <ConfirmDialog open title="Confirm Delete" body="body" busy onConfirm={noop} onCancel={onCancel} />,
        );

        // when
        await user.click(screen.getByRole("button", { name: closeLabel }));

        // then
        expect(onCancel).not.toHaveBeenCalled();
    });

    it("does not submit a form it is rendered inside when either control is pressed", async () => {
        // given
        const onSubmit = vi.fn((e: FormEvent) => e.preventDefault());
        const user = userEvent.setup();
        renderWithProviders(
            <form onSubmit={onSubmit}>
                <ConfirmDialog
                    open
                    title="Confirm Delete"
                    body="body"
                    confirmLabel="Delete"
                    onConfirm={noop}
                    onCancel={noop}
                />
            </form>,
        );

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(onSubmit).not.toHaveBeenCalled();
    });
});
