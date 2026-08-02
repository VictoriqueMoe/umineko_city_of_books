import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { UserProfile } from "../../types/api";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { SetEmailPage } from "./SetEmailPage";

const mocks = vi.hoisted(() => ({
    navigate: vi.fn(),
    setEmail: vi.fn(),
}));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => mocks.navigate };
});

vi.mock("../../api/mutations/auth", () => ({
    useSetEmail: () => ({ mutateAsync: mocks.setEmail }),
}));

interface SetupOptions {
    user?: UserProfile | null;
    loading?: boolean;
}

function setup(options: SetupOptions = {}) {
    const user = userEvent.setup();
    const result = renderWithProviders(
        <Routes>
            <Route path="/set-email" element={<SetEmailPage />} />
            <Route path="/login" element={<div>sign in page</div>} />
            <Route path="/" element={<div>city of books</div>} />
        </Routes>,
        {
            route: "/set-email",
            user: options.user === undefined ? makeUser() : options.user,
            auth: { loading: options.loading ?? false },
        },
    );

    return { user, ...result };
}

async function fillForm(user: ReturnType<typeof userEvent.setup>) {
    await user.type(screen.getByPlaceholderText("Email"), "beato@example.com");
    await user.type(screen.getByPlaceholderText("Current password"), "goldentruth");
}

beforeEach(() => {
    mocks.setEmail.mockResolvedValue(undefined);
});

describe("SetEmailPage gating", () => {
    it("waits quietly while the session is still being restored", () => {
        // given
        const loading = true;

        // when
        setup({ loading });

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
        expect(screen.queryByPlaceholderText("Email")).not.toBeInTheDocument();
    });

    it("sends a signed out visitor to the sign in page", () => {
        // given
        const signedOut = null;

        // when
        setup({ user: signedOut });

        // then
        expect(screen.getByText("sign in page")).toBeInTheDocument();
    });

    it("sends a member who already has an email back to the city", () => {
        // given
        const user = makeUser({ email: "beato@example.com" });

        // when
        setup({ user });

        // then
        expect(screen.getByText("city of books")).toBeInTheDocument();
    });

    it("asks a member without an email to add one", () => {
        // given
        const user = makeUser({ email: undefined });

        // when
        setup({ user });

        // then
        expect(screen.getByRole("heading", { name: "Add your email" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Current password")).toBeInTheDocument();
    });
});

describe("SetEmailPage saving an address", () => {
    it("refuses to save until both the address and the password are given", async () => {
        // given
        const { user } = setup();

        // when
        await user.type(screen.getByPlaceholderText("Email"), "beato@example.com");

        // then
        expect(screen.getByRole("button", { name: "Save and continue" })).toBeDisabled();
    });

    it("sends the address together with the current password", async () => {
        // given
        const { user } = setup();
        await fillForm(user);

        // when
        await user.click(screen.getByRole("button", { name: "Save and continue" }));

        // then
        expect(mocks.setEmail).toHaveBeenCalledWith({ email: "beato@example.com", password: "goldentruth" });
    });

    it("returns the member to the city once the address is saved", async () => {
        // given
        const { user } = setup();
        await fillForm(user);

        // when
        await user.click(screen.getByRole("button", { name: "Save and continue" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/");
    });

    it("shows the reason the address was rejected", async () => {
        // given
        mocks.setEmail.mockRejectedValue(new Error("That email is already in use."));
        const { user } = setup();
        await fillForm(user);

        // when
        await user.click(screen.getByRole("button", { name: "Save and continue" }));

        // then
        expect(await screen.findByText("That email is already in use.")).toBeInTheDocument();
        expect(mocks.navigate).not.toHaveBeenCalled();
    });

    it("falls back to a plain apology when the failure carries no message", async () => {
        // given
        mocks.setEmail.mockRejectedValue("the witch is silent");
        const { user } = setup();
        await fillForm(user);

        // when
        await user.click(screen.getByRole("button", { name: "Save and continue" }));

        // then
        expect(await screen.findByText("Something went wrong.")).toBeInTheDocument();
    });

    it("locks the submit control while the address is being saved", async () => {
        // given
        mocks.setEmail.mockImplementation(() => new Promise(() => {}));
        const { user } = setup();
        await fillForm(user);

        // when
        await user.click(screen.getByRole("button", { name: "Save and continue" }));

        // then
        expect(screen.getByRole("button", { name: "..." })).toBeDisabled();
    });
});
