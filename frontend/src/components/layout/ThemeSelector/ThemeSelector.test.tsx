import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { ThemeSelector } from "./ThemeSelector";

function openSelector() {
    return userEvent.setup();
}

describe("ThemeSelector trigger", () => {
    it("names the theme that is currently in use", () => {
        // given
        const theme = { theme: "bernkastel" as const };

        // when
        renderWithProviders(<ThemeSelector />, { theme });

        // then
        expect(screen.getByText("Lady Bernkastel")).toBeInTheDocument();
    });

    it("leaves the list closed until the trigger is pressed", () => {
        // given
        const ui = <ThemeSelector />;

        // when
        renderWithProviders(ui);

        // then
        expect(screen.getByRole("button", { name: /Theme/ })).toHaveAttribute("aria-expanded", "false");
        expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    });

    it("opens the list of themes when the trigger is pressed", async () => {
        // given
        const user = openSelector();
        renderWithProviders(<ThemeSelector />);

        // when
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // then
        expect(screen.getByRole("listbox")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Theme/ })).toHaveAttribute("aria-expanded", "true");
    });

    it("closes the list again when the trigger is pressed a second time", async () => {
        // given
        const user = openSelector();
        renderWithProviders(<ThemeSelector />);
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // when
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // then
        expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    });

    it("closes the list when a press lands outside it", async () => {
        // given
        const user = openSelector();
        renderWithProviders(<ThemeSelector />);
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // when
        fireEvent.mouseDown(document.body);

        // then
        expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    });
});

describe("ThemeSelector themes", () => {
    it("groups the themes under each series", async () => {
        // given
        const user = openSelector();
        renderWithProviders(<ThemeSelector />);

        // when
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // then
        expect(screen.getByText("Umineko")).toBeInTheDocument();
        expect(screen.getByText("Higurashi")).toBeInTheDocument();
        expect(screen.getByText("Ciconia")).toBeInTheDocument();
    });

    it("marks the theme in use as the selected option", async () => {
        // given
        const user = openSelector();
        renderWithProviders(<ThemeSelector />, { theme: { theme: "erika" } });

        // when
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // then
        expect(screen.getByRole("option", { name: /Erika Furudo/ })).toHaveAttribute("aria-selected", "true");
        expect(screen.getByRole("option", { name: /Battler Ushiromiya/ })).toHaveAttribute("aria-selected", "false");
    });

    it("hands the chosen theme to the theme context and closes the list", async () => {
        // given
        const setTheme = vi.fn();
        const user = openSelector();
        renderWithProviders(<ThemeSelector />, { theme: { setTheme } });
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // when
        await user.click(screen.getByRole("option", { name: /Satoko Houjou/ }));

        // then
        expect(setTheme).toHaveBeenCalledWith("satoko");
        expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    });

    it("hides the Maria theme from anyone who has not unlocked the witch hunter secret", async () => {
        // given
        const user = openSelector();
        renderWithProviders(<ThemeSelector />, { theme: { hasSecret: () => false } });

        // when
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // then
        expect(screen.queryByRole("option", { name: /Maria Ushiromiya/ })).not.toBeInTheDocument();
    });

    it("offers the Maria theme once the witch hunter secret is unlocked", async () => {
        // given
        const setTheme = vi.fn();
        const user = openSelector();
        renderWithProviders(<ThemeSelector />, { theme: { setTheme, hasSecret: id => id === "witchHunter" } });
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // when
        await user.click(screen.getByRole("option", { name: /Maria Ushiromiya/ }));

        // then
        expect(setTheme).toHaveBeenCalledWith("maria");
    });

    it("still names a secret theme on the trigger while it is hidden from the list", async () => {
        // given
        const user = openSelector();
        renderWithProviders(<ThemeSelector />, { theme: { theme: "maria", hasSecret: () => false } });

        // when
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // then
        expect(screen.getByRole("button", { name: /Maria Ushiromiya/ })).toBeInTheDocument();
        expect(screen.queryByRole("option", { name: /Maria Ushiromiya/ })).not.toBeInTheDocument();
    });
});

describe("ThemeSelector fonts", () => {
    it("marks the font in use as the selected option", async () => {
        // given
        const user = openSelector();
        renderWithProviders(<ThemeSelector />, { theme: { font: "im-fell" } });

        // when
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // then
        expect(screen.getByRole("option", { name: /IM Fell English/ })).toHaveAttribute("aria-selected", "true");
        expect(screen.getByRole("option", { name: /Cinzel & Garamond/ })).toHaveAttribute("aria-selected", "false");
    });

    it("hands the chosen font to the theme context and closes the list", async () => {
        // given
        const setFont = vi.fn();
        const user = openSelector();
        renderWithProviders(<ThemeSelector />, { theme: { setFont } });
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // when
        await user.click(screen.getByRole("option", { name: /IM Fell English/ }));

        // then
        expect(setFont).toHaveBeenCalledWith("im-fell");
        expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    });
});

describe("ThemeSelector layout preferences", () => {
    it("reflects the current layout and particle preferences on the switches", async () => {
        // given
        const user = openSelector();
        renderWithProviders(<ThemeSelector />, { theme: { wideLayout: true, particlesEnabled: false } });

        // when
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // then
        expect(screen.getByRole("switch", { name: "Wide layout" })).toHaveAttribute("aria-checked", "true");
        expect(screen.getByRole("switch", { name: "Particles" })).toHaveAttribute("aria-checked", "false");
    });

    it("turns the wide layout on through its switch", async () => {
        // given
        const setWideLayout = vi.fn();
        const user = openSelector();
        renderWithProviders(<ThemeSelector />, { theme: { wideLayout: false, setWideLayout } });
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // when
        await user.click(screen.getByRole("switch", { name: "Wide layout" }));

        // then
        expect(setWideLayout).toHaveBeenCalledWith(true);
    });

    it("turns the particles off through their switch and leaves the list open", async () => {
        // given
        const setParticlesEnabled = vi.fn();
        const user = openSelector();
        renderWithProviders(<ThemeSelector />, { theme: { particlesEnabled: true, setParticlesEnabled } });
        await user.click(screen.getByRole("button", { name: /Theme/ }));

        // when
        await user.click(screen.getByRole("switch", { name: "Particles" }));

        // then
        expect(setParticlesEnabled).toHaveBeenCalledWith(false);
        expect(screen.getByRole("listbox")).toBeInTheDocument();
    });
});
