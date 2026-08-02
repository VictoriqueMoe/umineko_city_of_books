import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { ConfirmEmailPasswordModal } from "./ConfirmEmailPasswordModal";

interface SetupOptions {
    isOpen?: boolean;
    newEmail?: string;
}

function setup(options: SetupOptions = {}) {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    const user = userEvent.setup();

    const result = renderWithProviders(
        <ConfirmEmailPasswordModal
            isOpen={options.isOpen ?? true}
            newEmail={options.newEmail ?? "beato@example.com"}
            onConfirm={onConfirm}
            onCancel={onCancel}
        />,
    );

    return { ...result, user, onConfirm, onCancel };
}

function passwordBox(): HTMLElement {
    return screen.getByLabelText("Current password");
}

function confirmButton(): HTMLElement {
    return screen.getByRole("button", { name: "Confirm" });
}

describe("ConfirmEmailPasswordModal", () => {
    it("stays out of the way while it is closed", () => {
        // given
        const options = { isOpen: false };

        // when
        setup(options);

        // then
        expect(screen.queryByText("Confirm your password")).not.toBeInTheDocument();
    });

    it("names the address the player is moving to", () => {
        // given
        const options = { newEmail: "golden@witch.moe" };

        // when
        setup(options);

        // then
        expect(screen.getByText("golden@witch.moe")).toBeInTheDocument();
    });

    it("holds the confirm button back until a password is typed", async () => {
        // given
        const { user } = setup();

        // when
        await user.type(passwordBox(), "g");

        // then
        expect(confirmButton()).toBeEnabled();
    });

    it("refuses to confirm while the password box is empty", () => {
        // given
        setup();

        // when
        const button = confirmButton();

        // then
        expect(button).toBeDisabled();
    });

    it("hands the typed password to the parent", async () => {
        // given
        const { user, onConfirm } = setup();
        await user.type(passwordBox(), "goldentruth");

        // when
        await user.click(confirmButton());

        // then
        expect(onConfirm).toHaveBeenCalledWith("goldentruth");
    });

    it("forgets the typed password once it has been handed over", async () => {
        // given
        const { user } = setup();
        await user.type(passwordBox(), "goldentruth");

        // when
        await user.click(confirmButton());

        // then
        expect(passwordBox()).toHaveValue("");
        expect(confirmButton()).toBeDisabled();
    });

    it("confirms when the player presses enter in the password box", async () => {
        // given
        const { user, onConfirm } = setup();
        await user.type(passwordBox(), "goldentruth");

        // when
        await user.keyboard("{Enter}");

        // then
        expect(onConfirm).toHaveBeenCalledWith("goldentruth");
    });

    it("ignores enter while nothing has been typed", async () => {
        // given
        const { user, onConfirm } = setup();

        // when
        await user.click(passwordBox());
        await user.keyboard("{Enter}");

        // then
        expect(onConfirm).not.toHaveBeenCalled();
    });

    it("cancels and throws away whatever was typed", async () => {
        // given
        const { user, onCancel } = setup();
        await user.type(passwordBox(), "goldentruth");

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(onCancel).toHaveBeenCalledOnce();
        expect(passwordBox()).toHaveValue("");
    });

    it("treats dismissing the dialog as a cancel", async () => {
        // given
        const { user, onCancel, onConfirm } = setup();

        // when
        await user.click(screen.getByRole("button", { name: "✕" }));

        // then
        expect(onCancel).toHaveBeenCalledOnce();
        expect(onConfirm).not.toHaveBeenCalled();
    });
});
