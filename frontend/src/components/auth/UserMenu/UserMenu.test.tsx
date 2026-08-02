import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import { UserMenu } from "./UserMenu";

const { navigate } = vi.hoisted(() => ({ navigate: vi.fn() }));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => navigate };
});

describe("UserMenu signed out", () => {
    it("renders nothing at all when nobody is signed in", () => {
        // given
        const user = null;

        // when
        const { container } = renderWithProviders(<UserMenu />, { user });

        // then
        expect(container).toBeEmptyDOMElement();
    });
});

describe("UserMenu signed in", () => {
    it("shows the display name of the signed in account", () => {
        // given
        const account = makeUser({ display_name: "Featherine" });

        // when
        renderWithProviders(<UserMenu />, { user: account });

        // then
        expect(screen.getByRole("button", { name: /Featherine/ })).toBeInTheDocument();
    });

    it("shows the avatar when the account has one", () => {
        // given
        const account = makeUser({ avatar_url: "https://example.test/beato.png" });

        // when
        const { container } = renderWithProviders(<UserMenu />, { user: account });

        // then
        expect(container.querySelector("img")).toHaveAttribute("src", "https://example.test/beato.png");
    });

    it("falls back to the first letter of the display name when there is no avatar", () => {
        // given
        const account = makeUser({ display_name: "Virgilia", avatar_url: "" });

        // when
        const { container } = renderWithProviders(<UserMenu />, { user: account });

        // then
        expect(container.querySelector("img")).toBeNull();
        expect(screen.getByText("V")).toBeInTheDocument();
    });

    it("keeps the menu closed until the trigger is pressed", () => {
        // given
        const account = makeUser();

        // when
        renderWithProviders(<UserMenu />, { user: account });

        // then
        expect(screen.queryByRole("button", { name: "Profile" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Logout" })).not.toBeInTheDocument();
    });

    it("reveals the profile, settings and logout choices when opened", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<UserMenu />, { user: makeUser({ display_name: "Beatrice" }) });

        // when
        await user.click(screen.getByRole("button", { name: /Beatrice/ }));

        // then
        expect(screen.getByRole("button", { name: "Profile" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Settings" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Logout" })).toBeInTheDocument();
    });

    it("closes the menu again when the trigger is pressed a second time", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<UserMenu />, { user: makeUser({ display_name: "Beatrice" }) });
        await user.click(screen.getByRole("button", { name: /Beatrice/ }));

        // when
        await user.click(screen.getByRole("button", { name: /Beatrice/ }));

        // then
        expect(screen.queryByRole("button", { name: "Profile" })).not.toBeInTheDocument();
    });

    it("opens the profile of the signed in account and closes the menu", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<UserMenu />, { user: makeUser({ username: "bernkastel" }) });
        await user.click(screen.getByRole("button", { name: /Beatrice/ }));

        // when
        await user.click(screen.getByRole("button", { name: "Profile" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/user/bernkastel");
        expect(screen.queryByRole("button", { name: "Profile" })).not.toBeInTheDocument();
    });

    it("opens the settings page and closes the menu", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<UserMenu />, { user: makeUser() });
        await user.click(screen.getByRole("button", { name: /Beatrice/ }));

        // when
        await user.click(screen.getByRole("button", { name: "Settings" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/settings");
        expect(screen.queryByRole("button", { name: "Settings" })).not.toBeInTheDocument();
    });

    it("signs the account out and returns to the home page", async () => {
        // given
        const logoutUser = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderWithProviders(<UserMenu />, { user: makeUser(), auth: { logoutUser } });
        await user.click(screen.getByRole("button", { name: /Beatrice/ }));

        // when
        await user.click(screen.getByRole("button", { name: "Logout" }));

        // then
        expect(logoutUser).toHaveBeenCalledOnce();
        await waitFor(() => expect(navigate).toHaveBeenCalledWith("/"));
    });

    it("closes the menu when a press lands outside it", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<UserMenu />, { user: makeUser({ display_name: "Beatrice" }) });
        await user.click(screen.getByRole("button", { name: /Beatrice/ }));

        // when
        fireEvent.mouseDown(document.body);

        // then
        expect(screen.queryByRole("button", { name: "Logout" })).not.toBeInTheDocument();
    });
});
