import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useTheme } from "../hooks/useTheme";
import type { UserProfile } from "../types/api";
import { makeUser } from "../test-utils/fixtures";
import { renderWithProviders } from "../test-utils/render";
import { ThemeProvider } from "./ThemeContext";

const { updateAppearance } = vi.hoisted(() => ({ updateAppearance: vi.fn() }));

vi.mock("../api/mutations/auth", () => ({
    useUpdateAppearance: () => ({ mutate: updateAppearance }),
}));

function Probe() {
    const {
        theme,
        font,
        wideLayout,
        particlesEnabled,
        setTheme,
        setFont,
        setWideLayout,
        setParticlesEnabled,
        hasSecret,
        addSecret,
    } = useTheme();

    return (
        <div>
            <p>{`theme: ${theme}`}</p>
            <p>{`font: ${font}`}</p>
            <p>{`wide: ${String(wideLayout)}`}</p>
            <p>{`particles: ${String(particlesEnabled)}`}</p>
            <p>{`witch hunter: ${String(hasSecret("witchHunter"))}`}</p>
            <button type="button" onClick={() => setTheme("erika")}>
                choose erika
            </button>
            <button type="button" onClick={() => setFont("im-fell")}>
                choose im-fell
            </button>
            <button type="button" onClick={() => setWideLayout(true)}>
                widen
            </button>
            <button type="button" onClick={() => setWideLayout(false)}>
                narrow
            </button>
            <button type="button" onClick={() => setParticlesEnabled(false)}>
                stop particles
            </button>
            <button type="button" onClick={() => addSecret("witchHunter")}>
                unlock witch hunter
            </button>
        </div>
    );
}

function renderProbe(options: { user?: UserProfile | null; defaultTheme?: string } = {}) {
    return renderWithProviders(
        <ThemeProvider>
            <Probe />
        </ThemeProvider>,
        { user: options.user ?? null, siteInfo: { default_theme: options.defaultTheme ?? "featherine" } },
    );
}

afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-font");
    document.documentElement.removeAttribute("data-width");
});

describe("ThemeProvider theme selection", () => {
    it("falls back to the theme the site nominates when nothing has been stored", () => {
        // given
        const defaultTheme = "bernkastel";

        // when
        renderProbe({ defaultTheme });

        // then
        expect(screen.getByText("theme: bernkastel")).toBeInTheDocument();
        expect(document.documentElement.getAttribute("data-theme")).toBe("bernkastel");
        expect(localStorage.getItem("ut-theme")).toBe("bernkastel");
    });

    it("prefers a valid stored theme over the site default", () => {
        // given
        localStorage.setItem("ut-theme", "erika");

        // when
        renderProbe({ defaultTheme: "bernkastel" });

        // then
        expect(screen.getByText("theme: erika")).toBeInTheDocument();
        expect(document.documentElement.getAttribute("data-theme")).toBe("erika");
    });

    it("ignores an unrecognised stored theme and repairs the stored value", () => {
        // given
        localStorage.setItem("ut-theme", "kinzo");

        // when
        renderProbe({ defaultTheme: "not-a-real-theme" });

        // then
        expect(screen.getByText("theme: featherine")).toBeInTheDocument();
        expect(localStorage.getItem("ut-theme")).toBe("featherine");
    });

    it("leaves the document without a data-theme attribute for the default theme", () => {
        // given
        document.documentElement.setAttribute("data-theme", "erika");

        // when
        renderProbe();

        // then
        expect(document.documentElement.hasAttribute("data-theme")).toBe(false);
    });

    it("applies a newly chosen theme to the document and to storage", async () => {
        // given
        const user = userEvent.setup();
        renderProbe({ defaultTheme: "bernkastel" });

        // when
        await user.click(screen.getByRole("button", { name: "choose erika" }));

        // then
        expect(screen.getByText("theme: erika")).toBeInTheDocument();
        expect(document.documentElement.getAttribute("data-theme")).toBe("erika");
        expect(localStorage.getItem("ut-theme")).toBe("erika");
    });

    it("prefers the signed in account's saved theme over the stored one", () => {
        // given
        localStorage.setItem("ut-theme", "erika");

        // when
        renderProbe({ user: makeUser({ private: { theme: "battler" } }) });

        // then
        expect(screen.getByText("theme: battler")).toBeInTheDocument();
    });

    it("uses an obfuscated document key for a theme that hides behind a secret", () => {
        // given
        const user = makeUser({ private: { theme: "maria" }, secrets: ["witchHunter"] });

        // when
        renderProbe({ user });

        // then
        expect(screen.getByText("theme: maria")).toBeInTheDocument();
        expect(document.documentElement.getAttribute("data-theme")).toBe("_0x9e2a1c");
    });

    it("refuses a secret theme when the account has not unlocked the secret", () => {
        // given
        const user = makeUser({ private: { theme: "maria" }, secrets: [] });

        // when
        renderProbe({ user, defaultTheme: "bernkastel" });

        // then
        expect(screen.getByText("theme: bernkastel")).toBeInTheDocument();
        expect(document.documentElement.getAttribute("data-theme")).toBe("bernkastel");
    });
});

describe("ThemeProvider font selection", () => {
    it("loads the stored font and marks it on the document", () => {
        // given
        localStorage.setItem("ut-font", "im-fell");

        // when
        renderProbe();

        // then
        expect(screen.getByText("font: im-fell")).toBeInTheDocument();
        expect(document.documentElement.getAttribute("data-font")).toBe("im-fell");
    });

    it("ignores an unrecognised stored font and leaves the document unmarked", () => {
        // given
        localStorage.setItem("ut-font", "comic-sans");
        document.documentElement.setAttribute("data-font", "im-fell");

        // when
        renderProbe();

        // then
        expect(screen.getByText("font: default")).toBeInTheDocument();
        expect(document.documentElement.hasAttribute("data-font")).toBe(false);
    });

    it("records a font change on the document and in storage", async () => {
        // given
        const user = userEvent.setup();
        renderProbe();

        // when
        await user.click(screen.getByRole("button", { name: "choose im-fell" }));

        // then
        expect(screen.getByText("font: im-fell")).toBeInTheDocument();
        expect(document.documentElement.getAttribute("data-font")).toBe("im-fell");
        expect(localStorage.getItem("ut-font")).toBe("im-fell");
    });
});

describe("ThemeProvider layout width", () => {
    it("defaults to the narrow layout with no width marker", () => {
        // given
        document.documentElement.setAttribute("data-width", "wide");

        // when
        renderProbe();

        // then
        expect(screen.getByText("wide: false")).toBeInTheDocument();
        expect(document.documentElement.hasAttribute("data-width")).toBe(false);
    });

    it("honours a stored preference for the wide layout", () => {
        // given
        localStorage.setItem("ut-wide-layout", "true");

        // when
        renderProbe();

        // then
        expect(screen.getByText("wide: true")).toBeInTheDocument();
        expect(document.documentElement.getAttribute("data-width")).toBe("wide");
    });

    it("clears the width marker when the wide layout is turned off", async () => {
        // given
        localStorage.setItem("ut-wide-layout", "true");
        const user = userEvent.setup();
        renderProbe();

        // when
        await user.click(screen.getByRole("button", { name: "narrow" }));

        // then
        expect(screen.getByText("wide: false")).toBeInTheDocument();
        expect(document.documentElement.hasAttribute("data-width")).toBe(false);
        expect(localStorage.getItem("ut-wide-layout")).toBe("false");
    });
});

describe("ThemeProvider particles", () => {
    it("leaves particles on when no preference has been stored", () => {
        // given
        localStorage.removeItem("ut-particles");

        // when
        renderProbe();

        // then
        expect(screen.getByText("particles: true")).toBeInTheDocument();
    });

    it("honours a stored preference to switch particles off", () => {
        // given
        localStorage.setItem("ut-particles", "false");

        // when
        renderProbe();

        // then
        expect(screen.getByText("particles: false")).toBeInTheDocument();
    });

    it("persists a change to the particle preference", async () => {
        // given
        const user = userEvent.setup();
        renderProbe();

        // when
        await user.click(screen.getByRole("button", { name: "stop particles" }));

        // then
        expect(screen.getByText("particles: false")).toBeInTheDocument();
        expect(localStorage.getItem("ut-particles")).toBe("false");
    });
});

describe("ThemeProvider secrets", () => {
    it("loads previously unlocked secrets from storage", () => {
        // given
        localStorage.setItem("ut-secrets", JSON.stringify(["witchHunter"]));

        // when
        renderProbe();

        // then
        expect(screen.getByText("witch hunter: true")).toBeInTheDocument();
    });

    it("treats a malformed stored secrets payload as no secrets at all", () => {
        // given
        localStorage.setItem("ut-secrets", "not json at all");

        // when
        renderProbe();

        // then
        expect(screen.getByText("witch hunter: false")).toBeInTheDocument();
    });

    it("treats a stored secrets payload that is not a list as no secrets at all", () => {
        // given
        localStorage.setItem("ut-secrets", JSON.stringify({ witchHunter: true }));

        // when
        renderProbe();

        // then
        expect(screen.getByText("witch hunter: false")).toBeInTheDocument();
    });

    it("unlocks a secret and writes it back to storage", async () => {
        // given
        const user = userEvent.setup();
        renderProbe();

        // when
        await user.click(screen.getByRole("button", { name: "unlock witch hunter" }));

        // then
        expect(screen.getByText("witch hunter: true")).toBeInTheDocument();
        expect(JSON.parse(localStorage.getItem("ut-secrets") ?? "[]")).toEqual(["witchHunter"]);
    });

    it("keeps a single entry when the same secret is unlocked twice", async () => {
        // given
        const user = userEvent.setup();
        renderProbe();

        // when
        await user.click(screen.getByRole("button", { name: "unlock witch hunter" }));
        await user.click(screen.getByRole("button", { name: "unlock witch hunter" }));

        // then
        expect(JSON.parse(localStorage.getItem("ut-secrets") ?? "[]")).toEqual(["witchHunter"]);
    });

    it("leaves the stored secrets alone when an unrelated preference changes", async () => {
        // given
        const user = userEvent.setup();
        renderProbe({ user: makeUser({ secrets: ["witchHunter"] }) });
        localStorage.removeItem("ut-secrets");

        // when
        await user.click(screen.getByRole("button", { name: "stop particles" }));

        // then
        expect(screen.getByText("particles: false")).toBeInTheDocument();
        expect(localStorage.getItem("ut-secrets")).toBeNull();
    });

    it("takes the unlocked secrets from the signed in account", () => {
        // given
        localStorage.setItem("ut-secrets", JSON.stringify([]));

        // when
        renderProbe({ user: makeUser({ secrets: ["witchHunter"] }) });

        // then
        expect(screen.getByText("witch hunter: true")).toBeInTheDocument();
    });
});

describe("ThemeProvider appearance persistence", () => {
    it("saves the whole appearance to the account when a signed in user changes theme", async () => {
        // given
        const user = userEvent.setup();
        renderProbe({ user: makeUser() });

        // when
        await user.click(screen.getByRole("button", { name: "choose erika" }));

        // then
        expect(updateAppearance).toHaveBeenCalledOnce();
        expect(updateAppearance).toHaveBeenCalledWith({ theme: "erika", font: "default", wideLayout: false });
    });

    it("sends the current theme and width alongside a font change", async () => {
        // given
        localStorage.setItem("ut-wide-layout", "true");
        const user = userEvent.setup();
        renderProbe({ user: makeUser({ private: { theme: "battler" } }) });

        // when
        await user.click(screen.getByRole("button", { name: "choose im-fell" }));

        // then
        expect(updateAppearance).toHaveBeenCalledWith({ theme: "battler", font: "im-fell", wideLayout: true });
    });

    it("keeps an earlier theme choice when only the layout width changes", async () => {
        // given
        const user = userEvent.setup();
        renderProbe({ user: makeUser({ private: { theme: "battler" } }) });
        await user.click(screen.getByRole("button", { name: "choose erika" }));

        // when
        await user.click(screen.getByRole("button", { name: "widen" }));

        // then
        expect(screen.getByText("theme: erika")).toBeInTheDocument();
        expect(screen.getByText("wide: true")).toBeInTheDocument();
        expect(updateAppearance).toHaveBeenLastCalledWith({ theme: "erika", font: "default", wideLayout: true });
    });

    it("does not touch the account when a signed out visitor changes theme", async () => {
        // given
        const user = userEvent.setup();
        renderProbe();

        // when
        await user.click(screen.getByRole("button", { name: "choose erika" }));

        // then
        expect(updateAppearance).not.toHaveBeenCalled();
        expect(screen.getByText("theme: erika")).toBeInTheDocument();
    });
});
