import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { DangerZoneSection } from "./DangerZoneSection";

const mocks = vi.hoisted(() => ({
    navigate: vi.fn(),
    useDeleteAccount: vi.fn(),
    deleteAccount: vi.fn(),
    setUser: vi.fn(),
}));

vi.mock("../../api/mutations/auth", () => ({ useDeleteAccount: mocks.useDeleteAccount }));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocks.navigate };
});

function setup() {
    const user = userEvent.setup();
    const result = renderWithProviders(<DangerZoneSection />, {
        user: makeUser(),
        auth: { setUser: mocks.setUser },
    });

    return { ...result, user };
}

async function openModal(user: ReturnType<typeof userEvent.setup>) {
    await user.click(screen.getByRole("button", { name: "Delete Account" }));
}

function confirmButton(): HTMLElement {
    return screen.getByRole("button", { name: "Delete My Account" });
}

beforeEach(() => {
    mocks.deleteAccount.mockResolvedValue(undefined);
    mocks.useDeleteAccount.mockReturnValue({ mutateAsync: mocks.deleteAccount, isPending: false });
});

describe("DangerZoneSection", () => {
    it("warns that deleting the account cannot be undone", () => {
        // given
        setup();

        // when
        const warning = screen.getByText(/Deleting your account is permanent/);

        // then
        expect(warning).toBeInTheDocument();
        expect(screen.queryByText("This action cannot be undone. Please enter your password to confirm.")).toBe(null);
    });

    it("asks for the password in a dialog before deleting anything", async () => {
        // given
        const { user } = setup();

        // when
        await openModal(user);

        // then
        expect(
            screen.getByText("This action cannot be undone. Please enter your password to confirm."),
        ).toBeInTheDocument();
        expect(confirmButton()).toBeDisabled();
    });

    it("keeps the confirm control locked until a password is given", async () => {
        // given
        const { user } = setup();
        await openModal(user);

        // when
        await user.type(screen.getByLabelText("Password"), "goldentruth");

        // then
        expect(confirmButton()).toBeEnabled();
    });

    it("sends the typed password with the deletion", async () => {
        // given
        const { user } = setup();
        await openModal(user);
        await user.type(screen.getByLabelText("Password"), "goldentruth");

        // when
        await user.click(confirmButton());

        // then
        expect(mocks.deleteAccount).toHaveBeenCalledWith({ password: "goldentruth" });
    });

    it("signs the player out and returns them to the feed once the account is gone", async () => {
        // given
        const { user } = setup();
        await openModal(user);
        await user.type(screen.getByLabelText("Password"), "goldentruth");

        // when
        await user.click(confirmButton());

        // then
        expect(mocks.setUser).toHaveBeenCalledWith(null);
        expect(mocks.navigate).toHaveBeenCalledWith("/");
    });

    it("keeps the player signed in and explains why the deletion failed", async () => {
        // given
        mocks.deleteAccount.mockRejectedValue(new Error("Password is incorrect."));
        const { user } = setup();
        await openModal(user);
        await user.type(screen.getByLabelText("Password"), "wrongone");

        // when
        await user.click(confirmButton());

        // then
        expect(await screen.findByText("Password is incorrect.")).toBeInTheDocument();
        expect(mocks.setUser).not.toHaveBeenCalled();
        expect(mocks.navigate).not.toHaveBeenCalled();
    });

    it("falls back to a plain apology when the failure carries no message", async () => {
        // given
        mocks.deleteAccount.mockRejectedValue("the witch is silent");
        const { user } = setup();
        await openModal(user);
        await user.type(screen.getByLabelText("Password"), "goldentruth");

        // when
        await user.click(confirmButton());

        // then
        expect(await screen.findByText("Failed to delete account.")).toBeInTheDocument();
    });

    it("lets the player back out of the dialog", async () => {
        // given
        const { user } = setup();
        await openModal(user);

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByText("This action cannot be undone. Please enter your password to confirm.")).toBe(null);
        expect(mocks.deleteAccount).not.toHaveBeenCalled();
    });

    it("shows the deletion is under way while it is in flight", async () => {
        // given
        mocks.useDeleteAccount.mockReturnValue({ mutateAsync: mocks.deleteAccount, isPending: true });
        const { user } = setup();

        // when
        await openModal(user);

        // then
        expect(screen.getByRole("button", { name: "Deleting..." })).toBeDisabled();
    });
});
