import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuth } from "../hooks/useAuth";
import { makeUser } from "../test-utils/fixtures";
import { createTestQueryClient, renderWithProviders } from "../test-utils/render";
import type { UserProfile } from "../types/api";
import { AuthProvider } from "./AuthContext";

const { useMe, refresh, login, register, logout } = vi.hoisted(() => ({
    useMe: vi.fn(),
    refresh: vi.fn(),
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
}));

vi.mock("../api/queries/auth", () => ({ useMe }));

vi.mock("../api/mutations/auth", () => ({
    useLogin: () => ({ mutateAsync: login }),
    useRegister: () => ({ mutateAsync: register }),
    useLogout: () => ({ mutateAsync: logout }),
}));

function Probe({ next }: { next: UserProfile | null }) {
    const { user, loading, setUser, loginUser, registerUser, logoutUser } = useAuth();
    const [outcome, setOutcome] = useState("idle");

    function run(work: Promise<void>) {
        work.then(() => setOutcome("settled")).catch(() => setOutcome("rejected"));
    }

    return (
        <div>
            <p>{`user: ${user?.username ?? "nobody"}`}</p>
            <p>{`loading: ${String(loading)}`}</p>
            <p>{`outcome: ${outcome}`}</p>
            <button type="button" onClick={() => run(loginUser("beatrice", "golden", "turnstile-token"))}>
                log in
            </button>
            <button type="button" onClick={() => run(loginUser("beatrice", "golden"))}>
                log in plainly
            </button>
            <button
                type="button"
                onClick={() =>
                    run(
                        registerUser(
                            "beatrice",
                            "beato@example.test",
                            "golden",
                            "Beatrice",
                            "invite-1",
                            "turnstile-token",
                        ),
                    )
                }
            >
                register
            </button>
            <button type="button" onClick={() => run(registerUser("ange", "ange@example.test", "gold", "Ange"))}>
                register plainly
            </button>
            <button type="button" onClick={() => run(logoutUser())}>
                log out
            </button>
            <button type="button" onClick={() => setUser(next)}>
                set user
            </button>
        </div>
    );
}

function renderProvider(me: UserProfile | null, loading = false, next: UserProfile | null = null) {
    useMe.mockReturnValue({ me, loading, refresh });
    const queryClient = createTestQueryClient();

    return renderWithProviders(
        <AuthProvider>
            <Probe next={next} />
        </AuthProvider>,
        { queryClient },
    );
}

beforeEach(() => {
    refresh.mockResolvedValue(undefined);
    login.mockResolvedValue(undefined);
    register.mockResolvedValue(undefined);
    logout.mockResolvedValue(undefined);
});

describe("AuthProvider", () => {
    it("hands the signed in user down to its children", () => {
        // given
        const me = makeUser({ username: "beatrice" });

        // when
        renderProvider(me);

        // then
        expect(screen.getByText("user: beatrice")).toBeInTheDocument();
    });

    it("reports nobody signed in when the session query came back empty", () => {
        // given
        const nobody = null;

        // when
        renderProvider(nobody);

        // then
        expect(screen.getByText("user: nobody")).toBeInTheDocument();
    });

    it("passes the loading flag of the session query straight through", () => {
        // given
        const stillLoading = true;

        // when
        renderProvider(null, stillLoading);

        // then
        expect(screen.getByText("loading: true")).toBeInTheDocument();
    });

    it("sends the credentials and the turnstile token to the login mutation", async () => {
        // given
        const user = userEvent.setup();
        renderProvider(null);

        // when
        await user.click(screen.getByRole("button", { name: "log in" }));

        // then
        expect(login).toHaveBeenCalledWith({
            username: "beatrice",
            password: "golden",
            turnstileToken: "turnstile-token",
        });
    });

    it("leaves the turnstile token out when the login form did not collect one", async () => {
        // given
        const user = userEvent.setup();
        renderProvider(null);

        // when
        await user.click(screen.getByRole("button", { name: "log in plainly" }));

        // then
        expect(login).toHaveBeenCalledWith({ username: "beatrice", password: "golden", turnstileToken: undefined });
    });

    it("refetches the session only after the login has gone through", async () => {
        // given
        const user = userEvent.setup();
        renderProvider(null);

        // when
        await user.click(screen.getByRole("button", { name: "log in" }));

        // then
        expect(refresh).toHaveBeenCalledOnce();
        expect(refresh.mock.invocationCallOrder[0]).toBeGreaterThan(login.mock.invocationCallOrder[0]);
    });

    it("never refetches the session when the login was rejected", async () => {
        // given
        login.mockRejectedValue(new Error("wrong password"));
        const user = userEvent.setup();
        renderProvider(null);

        // when
        await user.click(screen.getByRole("button", { name: "log in" }));

        // then
        expect(await screen.findByText("outcome: rejected")).toBeInTheDocument();
        expect(refresh).not.toHaveBeenCalled();
    });

    it("sends every registration field to the register mutation", async () => {
        // given
        const user = userEvent.setup();
        renderProvider(null);

        // when
        await user.click(screen.getByRole("button", { name: "register" }));

        // then
        expect(register).toHaveBeenCalledWith({
            username: "beatrice",
            email: "beato@example.test",
            password: "golden",
            displayName: "Beatrice",
            inviteCode: "invite-1",
            turnstileToken: "turnstile-token",
        });
    });

    it("registers without an invite code when none was supplied", async () => {
        // given
        const user = userEvent.setup();
        renderProvider(null);

        // when
        await user.click(screen.getByRole("button", { name: "register plainly" }));

        // then
        expect(register).toHaveBeenCalledWith({
            username: "ange",
            email: "ange@example.test",
            password: "gold",
            displayName: "Ange",
            inviteCode: undefined,
            turnstileToken: undefined,
        });
    });

    it("refetches the session only after the registration has gone through", async () => {
        // given
        const user = userEvent.setup();
        renderProvider(null);

        // when
        await user.click(screen.getByRole("button", { name: "register" }));

        // then
        expect(refresh.mock.invocationCallOrder[0]).toBeGreaterThan(register.mock.invocationCallOrder[0]);
    });

    it("never refetches the session when the registration was rejected", async () => {
        // given
        register.mockRejectedValue(new Error("username taken"));
        const user = userEvent.setup();
        renderProvider(null);

        // when
        await user.click(screen.getByRole("button", { name: "register" }));

        // then
        expect(await screen.findByText("outcome: rejected")).toBeInTheDocument();
        expect(refresh).not.toHaveBeenCalled();
    });

    it("empties the cached session once the logout has gone through", async () => {
        // given
        const user = userEvent.setup();
        const { queryClient } = renderProvider(makeUser());
        const setQueryData = vi.spyOn(queryClient, "setQueryData");

        // when
        await user.click(screen.getByRole("button", { name: "log out" }));

        // then
        expect(logout).toHaveBeenCalledOnce();
        expect(setQueryData).toHaveBeenCalledWith(["auth", "me"], null);
    });

    it("keeps the cached session when the logout request failed", async () => {
        // given
        logout.mockRejectedValue(new Error("the server is asleep"));
        const user = userEvent.setup();
        const { queryClient } = renderProvider(makeUser({ username: "beatrice" }));
        const setQueryData = vi.spyOn(queryClient, "setQueryData");

        // when
        await user.click(screen.getByRole("button", { name: "log out" }));

        // then
        expect(await screen.findByText("outcome: rejected")).toBeInTheDocument();
        expect(setQueryData).not.toHaveBeenCalled();
    });

    it("writes a replacement user straight into the session cache", async () => {
        // given
        const replacement = makeUser({ id: "user-9", username: "ange" });
        const user = userEvent.setup();
        const { queryClient } = renderProvider(makeUser(), false, replacement);
        const setQueryData = vi.spyOn(queryClient, "setQueryData");

        // when
        await user.click(screen.getByRole("button", { name: "set user" }));

        // then
        expect(setQueryData).toHaveBeenCalledWith(["auth", "me"], replacement);
    });
});
