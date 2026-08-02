import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useImperativeHandle, type Ref } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SiteInfo } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import { LoginPage } from "./LoginPage";

const mocks = vi.hoisted(() => ({
    navigate: vi.fn(),
    getLastLocation: vi.fn<() => string | null>(),
    turnstileReset: vi.fn(),
    loginUser: vi.fn(),
    registerUser: vi.fn(),
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

vi.mock("../../utils/lastLocation", () => ({ getLastLocation: mocks.getLastLocation }));

function setup(siteInfo: Partial<SiteInfo> = {}) {
    const user = userEvent.setup();
    const result = renderWithProviders(<LoginPage />, {
        siteInfo,
        auth: { loginUser: mocks.loginUser, registerUser: mocks.registerUser },
    });

    return { user, ...result };
}

function loginForm(): HTMLFormElement {
    const form = screen.getByPlaceholderText("Username").closest("form");
    if (!form) {
        throw new Error("the login form is missing");
    }

    return form as HTMLFormElement;
}

async function fillCredentials(user: ReturnType<typeof userEvent.setup>) {
    await user.type(screen.getByPlaceholderText("Username"), "beatrice");
    await user.type(screen.getByPlaceholderText("Password"), "goldentruth");
}

async function switchToRegister(user: ReturnType<typeof userEvent.setup>) {
    await user.click(screen.getByRole("button", { name: "Need an account? Register" }));
}

beforeEach(() => {
    mocks.getLastLocation.mockReturnValue(null);
    mocks.loginUser.mockResolvedValue(undefined);
    mocks.registerUser.mockResolvedValue(undefined);
});

describe("LoginPage signing in", () => {
    it("greets a returning player with the sign in form", () => {
        // given
        const siteInfo = {};

        // when
        setup(siteInfo);

        // then
        expect(screen.getByRole("heading", { name: "Enter the Game Board" })).toBeInTheDocument();
        expect(screen.queryByPlaceholderText("Email")).not.toBeInTheDocument();
    });

    it("refuses to submit until both a username and a password are given", async () => {
        // given
        const { user } = setup();

        // when
        await user.type(screen.getByPlaceholderText("Username"), "beatrice");

        // then
        expect(screen.getByRole("button", { name: "Sign In" })).toBeDisabled();
    });

    it("hands the typed credentials to the auth context", async () => {
        // given
        const { user } = setup();
        await fillCredentials(user);

        // when
        await user.click(screen.getByRole("button", { name: "Sign In" }));

        // then
        expect(mocks.loginUser).toHaveBeenCalledWith("beatrice", "goldentruth", undefined);
    });

    it("returns the player to the city entrance when nothing was visited before", async () => {
        // given
        const { user } = setup();
        await fillCredentials(user);

        // when
        await user.click(screen.getByRole("button", { name: "Sign In" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/", { replace: true });
    });

    it("returns the player to the page they were reading before signing in", async () => {
        // given
        mocks.getLastLocation.mockReturnValue("/theories?page=2");
        const { user } = setup();
        await fillCredentials(user);

        // when
        await user.click(screen.getByRole("button", { name: "Sign In" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/theories?page=2", { replace: true });
    });

    it("shows the reason the sign in was refused", async () => {
        // given
        mocks.loginUser.mockRejectedValue(new Error("Invalid username or password."));
        const { user } = setup();
        await fillCredentials(user);

        // when
        await user.click(screen.getByRole("button", { name: "Sign In" }));

        // then
        expect(await screen.findByText("Invalid username or password.")).toBeInTheDocument();
        expect(mocks.navigate).not.toHaveBeenCalled();
    });

    it("falls back to a plain apology when the failure carries no message", async () => {
        // given
        mocks.loginUser.mockRejectedValue("the witch is silent");
        const { user } = setup();
        await fillCredentials(user);

        // when
        await user.click(screen.getByRole("button", { name: "Sign In" }));

        // then
        expect(await screen.findByText("Something went wrong.")).toBeInTheDocument();
    });

    it("locks the submit control while the sign in is in flight", async () => {
        // given
        mocks.loginUser.mockImplementation(() => new Promise(() => {}));
        const { user } = setup();
        await fillCredentials(user);

        // when
        await user.click(screen.getByRole("button", { name: "Sign In" }));

        // then
        expect(screen.getByRole("button", { name: "..." })).toBeDisabled();
    });
});

describe("LoginPage registering", () => {
    it("asks for an email and a display name once the visitor switches to registering", async () => {
        // given
        const { user } = setup();

        // when
        await switchToRegister(user);

        // then
        expect(screen.getByRole("heading", { name: "Join the Game Board" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Email")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Display Name (optional)")).toBeInTheDocument();
    });

    it("refuses to register without an email address", async () => {
        // given
        const { user } = setup();
        await switchToRegister(user);

        // when
        await fillCredentials(user);

        // then
        expect(screen.getByRole("button", { name: "Register" })).toBeDisabled();
    });

    it("borrows the username as the display name when none was chosen", async () => {
        // given
        const { user } = setup();
        await switchToRegister(user);
        await fillCredentials(user);
        await user.type(screen.getByPlaceholderText("Email"), "beato@example.com");

        // when
        await user.click(screen.getByRole("button", { name: "Register" }));

        // then
        expect(mocks.registerUser).toHaveBeenCalledWith(
            "beatrice",
            "beato@example.com",
            "goldentruth",
            "beatrice",
            undefined,
            undefined,
        );
    });

    it("keeps the chosen display name when one was given", async () => {
        // given
        const { user } = setup();
        await switchToRegister(user);
        await fillCredentials(user);
        await user.type(screen.getByPlaceholderText("Email"), "beato@example.com");

        // when
        await user.type(screen.getByPlaceholderText("Display Name (optional)"), "The Golden Witch");
        await user.click(screen.getByRole("button", { name: "Register" }));

        // then
        expect(mocks.registerUser).toHaveBeenCalledWith(
            "beatrice",
            "beato@example.com",
            "goldentruth",
            "The Golden Witch",
            undefined,
            undefined,
        );
    });

    it("shows the reason the registration was refused", async () => {
        // given
        mocks.registerUser.mockRejectedValue(new Error("That username is already taken."));
        const { user } = setup();
        await switchToRegister(user);
        await fillCredentials(user);
        await user.type(screen.getByPlaceholderText("Email"), "beato@example.com");

        // when
        await user.click(screen.getByRole("button", { name: "Register" }));

        // then
        expect(await screen.findByText("That username is already taken.")).toBeInTheDocument();
    });

    it("lets the visitor go back to signing in", async () => {
        // given
        const { user } = setup();
        await switchToRegister(user);

        // when
        await user.click(screen.getByRole("button", { name: "Already have an account? Sign in" }));

        // then
        expect(screen.getByRole("heading", { name: "Enter the Game Board" })).toBeInTheDocument();
    });
});

describe("LoginPage registration type", () => {
    it("leaves out the invite code field while registration is open to everyone", async () => {
        // given
        const { user } = setup({ registration_type: "open" });

        // when
        await switchToRegister(user);

        // then
        expect(screen.queryByPlaceholderText("Invite Code")).not.toBeInTheDocument();
    });

    it("asks for an invite code while registration is invite only", async () => {
        // given
        const { user } = setup({ registration_type: "invite" });

        // when
        await switchToRegister(user);

        // then
        expect(screen.getByPlaceholderText("Invite Code")).toBeInTheDocument();
    });

    it("refuses to register on an invite only board without a code", async () => {
        // given
        const { user } = setup({ registration_type: "invite" });
        await switchToRegister(user);

        // when
        await fillCredentials(user);
        await user.type(screen.getByPlaceholderText("Email"), "beato@example.com");

        // then
        expect(screen.getByRole("button", { name: "Register" })).toBeDisabled();
    });

    it("forwards the invite code with the registration", async () => {
        // given
        const { user } = setup({ registration_type: "invite" });
        await switchToRegister(user);
        await fillCredentials(user);
        await user.type(screen.getByPlaceholderText("Email"), "beato@example.com");

        // when
        await user.type(screen.getByPlaceholderText("Invite Code"), "kakera-77");
        await user.click(screen.getByRole("button", { name: "Register" }));

        // then
        expect(mocks.registerUser).toHaveBeenCalledWith(
            "beatrice",
            "beato@example.com",
            "goldentruth",
            "beatrice",
            "kakera-77",
            undefined,
        );
    });

    it("closes the door entirely when registration is shut", () => {
        // given
        const siteInfo = { registration_type: "closed" };

        // when
        setup(siteInfo);

        // then
        expect(screen.queryByRole("button", { name: "Need an account? Register" })).not.toBeInTheDocument();
        expect(screen.getByText("Registration is currently closed.")).toBeInTheDocument();
    });
});

describe("LoginPage verification challenge", () => {
    it("leaves the challenge out when the site has it switched off", () => {
        // given
        const siteInfo = { turnstile_enabled: false, turnstile_site_key: "0x4AAA" };

        // when
        setup(siteInfo);

        // then
        expect(screen.queryByTestId("turnstile")).not.toBeInTheDocument();
    });

    it("shows the challenge with the configured site key", () => {
        // given
        const siteInfo = { turnstile_enabled: true, turnstile_site_key: "0x4AAA" };

        // when
        setup(siteInfo);

        // then
        expect(screen.getByTestId("turnstile")).toHaveAttribute("data-site-key", "0x4AAA");
    });

    it("holds the submit control back until the challenge is passed", async () => {
        // given
        const { user } = setup({ turnstile_enabled: true, turnstile_site_key: "0x4AAA" });

        // when
        await fillCredentials(user);

        // then
        expect(screen.getByRole("button", { name: "Sign In" })).toBeDisabled();
    });

    it("sends the challenge token along with the credentials", async () => {
        // given
        const { user } = setup({ turnstile_enabled: true, turnstile_site_key: "0x4AAA" });
        await fillCredentials(user);

        // when
        await user.click(screen.getByRole("button", { name: "pass verification" }));
        await user.click(screen.getByRole("button", { name: "Sign In" }));

        // then
        expect(mocks.loginUser).toHaveBeenCalledWith("beatrice", "goldentruth", "turnstile-token");
    });

    it("forgets the token once the challenge expires", async () => {
        // given
        const { user } = setup({ turnstile_enabled: true, turnstile_site_key: "0x4AAA" });
        await fillCredentials(user);
        await user.click(screen.getByRole("button", { name: "pass verification" }));

        // when
        await user.click(screen.getByRole("button", { name: "expire verification" }));

        // then
        expect(screen.getByRole("button", { name: "Sign In" })).toBeDisabled();
    });

    it("asks the visitor to finish the challenge when the form is submitted without a token", () => {
        // given
        setup({ turnstile_enabled: true, turnstile_site_key: "" });

        // when
        fireEvent.submit(loginForm());

        // then
        expect(screen.getByText("Please complete the verification.")).toBeInTheDocument();
        expect(mocks.loginUser).not.toHaveBeenCalled();
    });

    it("starts a fresh challenge after a refused sign in", async () => {
        // given
        mocks.loginUser.mockRejectedValue(new Error("Invalid username or password."));
        const { user } = setup({ turnstile_enabled: true, turnstile_site_key: "0x4AAA" });
        await fillCredentials(user);
        await user.click(screen.getByRole("button", { name: "pass verification" }));

        // when
        await user.click(screen.getByRole("button", { name: "Sign In" }));

        // then
        expect(await screen.findByText("Invalid username or password.")).toBeInTheDocument();
        expect(mocks.turnstileReset).toHaveBeenCalledOnce();
        expect(screen.getByRole("button", { name: "Sign In" })).toBeDisabled();
    });
});

describe("LoginPage password recovery", () => {
    it("offers a way to recover a forgotten password while email is available", async () => {
        // given
        const { user } = setup({ email_enabled: true });

        // when
        await user.click(screen.getByRole("button", { name: "Forgot your password?" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/forgot-password");
    });

    it("hides password recovery when the site cannot send email", () => {
        // given
        const siteInfo = { email_enabled: false };

        // when
        setup(siteInfo);

        // then
        expect(screen.queryByRole("button", { name: "Forgot your password?" })).not.toBeInTheDocument();
    });

    it("hides password recovery while the visitor is registering", async () => {
        // given
        const { user } = setup({ email_enabled: true });

        // when
        await switchToRegister(user);

        // then
        expect(screen.queryByRole("button", { name: "Forgot your password?" })).not.toBeInTheDocument();
    });
});
