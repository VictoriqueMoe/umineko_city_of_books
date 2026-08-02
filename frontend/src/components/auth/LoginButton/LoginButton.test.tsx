import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { LoginButton } from "./LoginButton";

const { navigate } = vi.hoisted(() => ({ navigate: vi.fn() }));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => navigate };
});

describe("LoginButton", () => {
    it("offers a sign in control that is ready to use", () => {
        // given
        const ui = <LoginButton />;

        // when
        renderWithProviders(ui);

        // then
        expect(screen.getByRole("button", { name: "Sign In" })).toBeEnabled();
    });

    it("stays put until the visitor presses it", () => {
        // given
        const ui = <LoginButton />;

        // when
        renderWithProviders(ui);

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("opens the login page when pressed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<LoginButton />);

        // when
        await user.click(screen.getByRole("button", { name: "Sign In" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/login");
    });
});
