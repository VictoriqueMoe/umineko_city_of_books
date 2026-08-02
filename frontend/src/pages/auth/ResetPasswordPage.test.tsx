import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { ResetPasswordPage } from "./ResetPasswordPage";

const mocks = vi.hoisted(() => ({
    navigate: vi.fn(),
    resetPassword: vi.fn(),
}));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => mocks.navigate };
});

vi.mock("../../api/mutations/auth", () => ({
    useResetPassword: () => ({ mutateAsync: mocks.resetPassword }),
}));

function setup(route = "/reset-password?token=kakera-token") {
    const user = userEvent.setup();
    const result = renderWithProviders(<ResetPasswordPage />, { route });

    return { user, ...result };
}

async function fillPasswords(user: ReturnType<typeof userEvent.setup>, first: string, second: string) {
    await user.type(screen.getByPlaceholderText("New password"), first);
    await user.type(screen.getByPlaceholderText("Confirm new password"), second);
}

beforeEach(() => {
    mocks.resetPassword.mockResolvedValue(undefined);
});

describe("ResetPasswordPage without a usable link", () => {
    it("tells the visitor the link is broken when no token is present", () => {
        // given
        const route = "/reset-password";

        // when
        setup(route);

        // then
        expect(screen.getByText("This reset link is invalid or incomplete.")).toBeInTheDocument();
        expect(screen.queryByPlaceholderText("New password")).not.toBeInTheDocument();
    });

    it("offers to start the recovery again", async () => {
        // given
        const { user } = setup("/reset-password");

        // when
        await user.click(screen.getByRole("button", { name: "Request a new link" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/forgot-password");
    });
});

describe("ResetPasswordPage choosing a new password", () => {
    it("shows the password form when the link carries a token", () => {
        // given
        const route = "/reset-password?token=kakera-token";

        // when
        setup(route);

        // then
        expect(screen.getByRole("heading", { name: "Choose a new password" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Confirm new password")).toBeInTheDocument();
    });

    it("refuses to submit until both password fields are filled", async () => {
        // given
        const { user } = setup();

        // when
        await user.type(screen.getByPlaceholderText("New password"), "goldenland");

        // then
        expect(screen.getByRole("button", { name: "Reset password" })).toBeDisabled();
    });

    it("complains when the two passwords disagree", async () => {
        // given
        const { user } = setup();
        await fillPasswords(user, "goldenland", "goldenlend");

        // when
        await user.click(screen.getByRole("button", { name: "Reset password" }));

        // then
        expect(screen.getByText("Passwords do not match.")).toBeInTheDocument();
        expect(mocks.resetPassword).not.toHaveBeenCalled();
    });

    it("sends the token from the link with the chosen password", async () => {
        // given
        const { user } = setup("/reset-password?token=kakera-token");
        await fillPasswords(user, "goldenland", "goldenland");

        // when
        await user.click(screen.getByRole("button", { name: "Reset password" }));

        // then
        expect(mocks.resetPassword).toHaveBeenCalledWith({ token: "kakera-token", newPassword: "goldenland" });
    });

    it("confirms the change and offers the way back to sign in", async () => {
        // given
        const { user } = setup();
        await fillPasswords(user, "goldenland", "goldenland");

        // when
        await user.click(screen.getByRole("button", { name: "Reset password" }));

        // then
        expect(
            await screen.findByText("Your password has been reset. You can now sign in with your new password."),
        ).toBeInTheDocument();
        expect(screen.queryByPlaceholderText("New password")).not.toBeInTheDocument();
    });

    it("takes the visitor to sign in once the password is changed", async () => {
        // given
        const { user } = setup();
        await fillPasswords(user, "goldenland", "goldenland");
        await user.click(screen.getByRole("button", { name: "Reset password" }));
        await screen.findByRole("button", { name: "Go to sign in" });

        // when
        await user.click(screen.getByRole("button", { name: "Go to sign in" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/login");
    });

    it("keeps the form and shows the reason when the reset is refused", async () => {
        // given
        mocks.resetPassword.mockRejectedValue(new Error("This link has already been used."));
        const { user } = setup();
        await fillPasswords(user, "goldenland", "goldenland");

        // when
        await user.click(screen.getByRole("button", { name: "Reset password" }));

        // then
        expect(await screen.findByText("This link has already been used.")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("New password")).toBeInTheDocument();
    });

    it("falls back to a plain apology when the failure carries no message", async () => {
        // given
        mocks.resetPassword.mockRejectedValue("the witch is silent");
        const { user } = setup();
        await fillPasswords(user, "goldenland", "goldenland");

        // when
        await user.click(screen.getByRole("button", { name: "Reset password" }));

        // then
        expect(await screen.findByText("Something went wrong.")).toBeInTheDocument();
    });

    it("locks the submit control while the reset is in flight", async () => {
        // given
        mocks.resetPassword.mockImplementation(() => new Promise(() => {}));
        const { user } = setup();
        await fillPasswords(user, "goldenland", "goldenland");

        // when
        await user.click(screen.getByRole("button", { name: "Reset password" }));

        // then
        expect(screen.getByRole("button", { name: "..." })).toBeDisabled();
    });
});
