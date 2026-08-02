import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { ChangePasswordSection } from "./ChangePasswordSection";

const mocks = vi.hoisted(() => ({
    useChangePassword: vi.fn(),
    changePassword: vi.fn(),
}));

vi.mock("../../api/mutations/auth", () => ({ useChangePassword: mocks.useChangePassword }));

function setup() {
    const user = userEvent.setup();
    const result = renderWithProviders(<ChangePasswordSection />);

    return { ...result, user };
}

function submitButton(): HTMLElement {
    return screen.getByRole("button", { name: "Change Password" });
}

async function fillForm(
    user: ReturnType<typeof userEvent.setup>,
    values: { old?: string; next?: string; confirm?: string } = {},
) {
    await user.type(screen.getByLabelText("Current Password"), values.old ?? "oldsecret");
    await user.type(screen.getByLabelText("New Password"), values.next ?? "goldentruth");
    await user.type(screen.getByLabelText("Confirm New Password"), values.confirm ?? "goldentruth");
}

beforeEach(() => {
    mocks.changePassword.mockResolvedValue(undefined);
    mocks.useChangePassword.mockReturnValue({ mutateAsync: mocks.changePassword, isPending: false });
});

describe("ChangePasswordSection", () => {
    it("refuses a new password shorter than eight characters", async () => {
        // given
        const { user } = setup();
        await fillForm(user, { next: "short", confirm: "short" });

        // when
        await user.click(submitButton());

        // then
        expect(screen.getByText("New password must be at least 8 characters.")).toBeInTheDocument();
        expect(mocks.changePassword).not.toHaveBeenCalled();
    });

    it("refuses a confirmation that does not match", async () => {
        // given
        const { user } = setup();
        await fillForm(user, { next: "goldentruth", confirm: "redtruth99" });

        // when
        await user.click(submitButton());

        // then
        expect(screen.getByText("Passwords do not match.")).toBeInTheDocument();
        expect(mocks.changePassword).not.toHaveBeenCalled();
    });

    it("sends the old and the new password together", async () => {
        // given
        const { user } = setup();
        await fillForm(user);

        // when
        await user.click(submitButton());

        // then
        expect(mocks.changePassword).toHaveBeenCalledWith({
            old_password: "oldsecret",
            new_password: "goldentruth",
        });
    });

    it("confirms the change and empties the three boxes", async () => {
        // given
        const { user } = setup();
        await fillForm(user);

        // when
        await user.click(submitButton());

        // then
        expect(await screen.findByText("Password changed successfully.")).toBeInTheDocument();
        expect(screen.getByLabelText("Current Password")).toHaveValue("");
        expect(screen.getByLabelText("New Password")).toHaveValue("");
        expect(screen.getByLabelText("Confirm New Password")).toHaveValue("");
    });

    it("repeats the reason the server refused the change", async () => {
        // given
        mocks.changePassword.mockRejectedValue(new Error("Current password is incorrect."));
        const { user } = setup();
        await fillForm(user);

        // when
        await user.click(submitButton());

        // then
        expect(await screen.findByText("Current password is incorrect.")).toBeInTheDocument();
    });

    it("falls back to a plain apology when the failure carries no message", async () => {
        // given
        mocks.changePassword.mockRejectedValue("the witch is silent");
        const { user } = setup();
        await fillForm(user);

        // when
        await user.click(submitButton());

        // then
        expect(await screen.findByText("Failed to change password.")).toBeInTheDocument();
    });

    it("clears an earlier complaint once a valid attempt is made", async () => {
        // given
        const { user } = setup();
        await fillForm(user, { next: "short", confirm: "short" });
        await user.click(submitButton());

        // when
        await user.clear(screen.getByLabelText("New Password"));
        await user.clear(screen.getByLabelText("Confirm New Password"));
        await user.type(screen.getByLabelText("New Password"), "goldentruth");
        await user.type(screen.getByLabelText("Confirm New Password"), "goldentruth");
        await user.click(submitButton());

        // then
        expect(screen.queryByText("New password must be at least 8 characters.")).not.toBeInTheDocument();
        expect(await screen.findByText("Password changed successfully.")).toBeInTheDocument();
    });

    it("locks the submit control while the change is in flight", () => {
        // given
        mocks.useChangePassword.mockReturnValue({ mutateAsync: mocks.changePassword, isPending: true });

        // when
        setup();

        // then
        expect(screen.getByRole("button", { name: "Changing..." })).toBeDisabled();
    });
});
