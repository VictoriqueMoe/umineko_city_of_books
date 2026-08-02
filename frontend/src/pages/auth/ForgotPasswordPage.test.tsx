import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useImperativeHandle, type Ref } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PublicUser, SiteInfo } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import { ForgotPasswordPage } from "./ForgotPasswordPage";

const mocks = vi.hoisted(() => ({
    navigate: vi.fn(),
    turnstileReset: vi.fn(),
    forgotPassword: vi.fn(),
    useStaff: vi.fn(),
}));

interface TurnstileStubProps {
    siteKey: string;
    onSuccess: (token: string) => void;
    onExpire: () => void;
    ref?: Ref<{ reset: () => void }>;
}

function TurnstileStub({ siteKey, onSuccess, onExpire, ref }: TurnstileStubProps) {
    useImperativeHandle(ref, () => ({ reset: mocks.turnstileReset }));

    return (
        <div data-testid="turnstile" data-site-key={siteKey}>
            <button type="button" onClick={() => onSuccess("turnstile-token")}>
                pass verification
            </button>
            <button type="button" onClick={() => onExpire()}>
                expire verification
            </button>
        </div>
    );
}

vi.mock("@marsidev/react-turnstile", () => ({ Turnstile: TurnstileStub }));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => mocks.navigate };
});

vi.mock("../../api/mutations/auth", () => ({
    useForgotPassword: () => ({ mutateAsync: mocks.forgotPassword }),
}));

vi.mock("../../api/queries/auth", () => ({ useStaff: mocks.useStaff }));

const SENT_NOTICE =
    "If an account with that username exists and has an email address, a reset link has been sent to it.";

function makeStaffMember(overrides: Partial<PublicUser> = {}): PublicUser {
    return {
        id: "staff-1",
        username: "battler",
        display_name: "Battler",
        avatar_url: "",
        role: "moderator",
        online: false,
        ...overrides,
    };
}

function setup(siteInfo: Partial<SiteInfo> = {}) {
    const user = userEvent.setup();
    const result = renderWithProviders(<ForgotPasswordPage />, { siteInfo });

    return { user, ...result };
}

function resetForm(): HTMLFormElement {
    const form = screen.getByPlaceholderText("Username").closest("form");
    if (!form) {
        throw new Error("the reset form is missing");
    }

    return form as HTMLFormElement;
}

beforeEach(() => {
    mocks.forgotPassword.mockResolvedValue(undefined);
    mocks.useStaff.mockReturnValue({ staff: [], loading: false });
});

describe("ForgotPasswordPage requesting a link", () => {
    it("explains what the reset request will do", () => {
        // given
        const siteInfo = {};

        // when
        setup(siteInfo);

        // then
        expect(screen.getByRole("heading", { name: "Reset your password" })).toBeInTheDocument();
        expect(
            screen.getByText("Enter your username and we will email a reset link to the address on your account."),
        ).toBeInTheDocument();
    });

    it("refuses to send anything until a username is typed", () => {
        // given
        setup();

        // when
        const button = screen.getByRole("button", { name: "Send reset link" });

        // then
        expect(button).toBeDisabled();
    });

    it("sends the typed username to the reset endpoint", async () => {
        // given
        const { user } = setup();
        await user.type(screen.getByPlaceholderText("Username"), "beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Send reset link" }));

        // then
        expect(mocks.forgotPassword).toHaveBeenCalledWith({ username: "beatrice", turnstileToken: undefined });
    });

    it("replaces the form with a deliberately vague confirmation", async () => {
        // given
        const { user } = setup();
        await user.type(screen.getByPlaceholderText("Username"), "beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Send reset link" }));

        // then
        expect(await screen.findByText(SENT_NOTICE)).toBeInTheDocument();
        expect(screen.queryByPlaceholderText("Username")).not.toBeInTheDocument();
    });

    it("keeps the form and shows the reason when the request fails", async () => {
        // given
        mocks.forgotPassword.mockRejectedValue(new Error("Too many attempts, try again later."));
        const { user } = setup();
        await user.type(screen.getByPlaceholderText("Username"), "beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Send reset link" }));

        // then
        expect(await screen.findByText("Too many attempts, try again later.")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Username")).toBeInTheDocument();
    });

    it("falls back to a plain apology when the failure carries no message", async () => {
        // given
        mocks.forgotPassword.mockRejectedValue("the witch is silent");
        const { user } = setup();
        await user.type(screen.getByPlaceholderText("Username"), "beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Send reset link" }));

        // then
        expect(await screen.findByText("Something went wrong.")).toBeInTheDocument();
    });

    it("locks the submit control while the request is in flight", async () => {
        // given
        mocks.forgotPassword.mockImplementation(() => new Promise(() => {}));
        const { user } = setup();
        await user.type(screen.getByPlaceholderText("Username"), "beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Send reset link" }));

        // then
        expect(screen.getByRole("button", { name: "..." })).toBeDisabled();
    });

    it("leads back to the sign in page", async () => {
        // given
        const { user } = setup();

        // when
        await user.click(screen.getByRole("button", { name: "Back to sign in" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/login");
    });
});

describe("ForgotPasswordPage verification challenge", () => {
    it("leaves the challenge out when the site has it switched off", () => {
        // given
        const siteInfo = { turnstile_enabled: false, turnstile_site_key: "0x4AAA" };

        // when
        setup(siteInfo);

        // then
        expect(screen.queryByTestId("turnstile")).not.toBeInTheDocument();
    });

    it("holds the submit control back until the challenge is passed", async () => {
        // given
        const { user } = setup({ turnstile_enabled: true, turnstile_site_key: "0x4AAA" });

        // when
        await user.type(screen.getByPlaceholderText("Username"), "beatrice");

        // then
        expect(screen.getByRole("button", { name: "Send reset link" })).toBeDisabled();
    });

    it("sends the challenge token along with the username", async () => {
        // given
        const { user } = setup({ turnstile_enabled: true, turnstile_site_key: "0x4AAA" });
        await user.type(screen.getByPlaceholderText("Username"), "beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "pass verification" }));
        await user.click(screen.getByRole("button", { name: "Send reset link" }));

        // then
        expect(mocks.forgotPassword).toHaveBeenCalledWith({
            username: "beatrice",
            turnstileToken: "turnstile-token",
        });
    });

    it("asks the visitor to finish the challenge when the form is submitted without a token", () => {
        // given
        setup({ turnstile_enabled: true, turnstile_site_key: "" });

        // when
        fireEvent.submit(resetForm());

        // then
        expect(screen.getByText("Please complete the verification.")).toBeInTheDocument();
        expect(mocks.forgotPassword).not.toHaveBeenCalled();
    });

    it("starts a fresh challenge after a failed request", async () => {
        // given
        mocks.forgotPassword.mockRejectedValue(new Error("Too many attempts, try again later."));
        const { user } = setup({ turnstile_enabled: true, turnstile_site_key: "0x4AAA" });
        await user.type(screen.getByPlaceholderText("Username"), "beatrice");
        await user.click(screen.getByRole("button", { name: "pass verification" }));

        // when
        await user.click(screen.getByRole("button", { name: "Send reset link" }));

        // then
        expect(await screen.findByText("Too many attempts, try again later.")).toBeInTheDocument();
        expect(mocks.turnstileReset).toHaveBeenCalledOnce();
    });
});

describe("ForgotPasswordPage staff contacts", () => {
    it("says nothing about staff when none are listed", () => {
        // given
        mocks.useStaff.mockReturnValue({ staff: [], loading: false });

        // when
        setup();

        // then
        expect(screen.queryByText("Reality Author")).not.toBeInTheDocument();
        expect(screen.queryByText(/No email on your account\?/)).not.toBeInTheDocument();
    });

    it("lists the staff a user without email can turn to", () => {
        // given
        mocks.useStaff.mockReturnValue({
            staff: [
                makeStaffMember({ id: "staff-1", display_name: "Bernkastel", role: "super_admin" }),
                makeStaffMember({ id: "staff-2", display_name: "Lambdadelta", role: "moderator" }),
            ],
            loading: false,
        });

        // when
        setup();

        // then
        expect(screen.getByText("Reality Author")).toBeInTheDocument();
        expect(screen.getByText("Bernkastel")).toBeInTheDocument();
        expect(screen.getByText("Witches")).toBeInTheDocument();
        expect(screen.getByText("Lambdadelta")).toBeInTheDocument();
    });

    it("leaves out a staff group that nobody belongs to", () => {
        // given
        mocks.useStaff.mockReturnValue({
            staff: [makeStaffMember({ id: "staff-1", display_name: "Bernkastel", role: "super_admin" })],
            loading: false,
        });

        // when
        setup();

        // then
        expect(screen.getByText("Reality Author")).toBeInTheDocument();
        expect(screen.queryByText("Voyager Witches")).not.toBeInTheDocument();
        expect(screen.queryByText("Witches")).not.toBeInTheDocument();
    });

    it("keeps the staff contacts on screen after the link has been sent", async () => {
        // given
        mocks.useStaff.mockReturnValue({
            staff: [makeStaffMember({ id: "staff-1", display_name: "Bernkastel", role: "super_admin" })],
            loading: false,
        });
        const { user } = setup();
        await user.type(screen.getByPlaceholderText("Username"), "beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Send reset link" }));

        // then
        expect(await screen.findByText(SENT_NOTICE)).toBeInTheDocument();
        expect(screen.getByText("Bernkastel")).toBeInTheDocument();
    });
});
