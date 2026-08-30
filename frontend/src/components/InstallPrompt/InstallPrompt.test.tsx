import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { clearInstallPrompt, watchInstallPrompt } from "../../platform/installPrompt";
import { InstallPrompt } from "./InstallPrompt";

const { capacitor } = vi.hoisted(() => ({ capacitor: { isNativePlatform: vi.fn() } }));

vi.mock("@capacitor/core", () => ({ Capacitor: capacitor }));

const DISMISSED_KEY = "dismissed_install_prompt";

const CHROME_UA = "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/126.0.0.0 Mobile Safari/537.36";
const IOS_SAFARI_UA =
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 Version/17.5 Mobile/15E148 Safari/604.1";
const IOS_CHROME_UA =
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 CriOS/126.0.0.0 Mobile/15E148 Safari/604.1";

function setUserAgent(ua: string) {
    Object.defineProperty(window.navigator, "userAgent", { value: ua, configurable: true });
}

function setStandalone(standalone: boolean) {
    vi.stubGlobal(
        "matchMedia",
        vi.fn((media: string) => ({
            matches: standalone && media === "(display-mode: standalone)",
            media,
            onchange: null,
            addListener: () => {},
            removeListener: () => {},
            addEventListener: () => {},
            removeEventListener: () => {},
            dispatchEvent: () => false,
        })),
    );
}

function fireInstallPrompt(prompt: () => Promise<void>) {
    act(() => {
        const event = new Event("beforeinstallprompt");
        Object.assign(event, { prompt });
        window.dispatchEvent(event);
    });
}

beforeEach(() => {
    watchInstallPrompt();
    clearInstallPrompt();
    capacitor.isNativePlatform.mockReturnValue(false);
    setUserAgent(CHROME_UA);
    setStandalone(false);
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe("InstallPrompt when the browser offers an install", () => {
    it("stays hidden until the browser fires beforeinstallprompt", () => {
        // given
        renderWithProviders(<InstallPrompt />);

        // when
        const banner = screen.queryByRole("region", { name: "Install City of Books" });

        // then
        expect(banner).toBeNull();
    });

    it("offers an install button once the browser fires beforeinstallprompt", () => {
        // given
        renderWithProviders(<InstallPrompt />);

        // when
        fireInstallPrompt(() => Promise.resolve());

        // then
        expect(screen.getByRole("button", { name: "Install" })).toBeInTheDocument();
    });

    it("triggers the browser install flow when the button is pressed", async () => {
        // given
        const prompt = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderWithProviders(<InstallPrompt />);
        fireInstallPrompt(prompt);

        // when
        await user.click(screen.getByRole("button", { name: "Install" }));

        // then
        expect(prompt).toHaveBeenCalledOnce();
        expect(screen.queryByRole("button", { name: "Install" })).toBeNull();
    });

    it("remembers a dismissal so the prompt does not come back", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<InstallPrompt />);
        fireInstallPrompt(() => Promise.resolve());

        // when
        await user.click(screen.getByRole("button", { name: "Dismiss" }));

        // then
        expect(screen.queryByRole("region", { name: "Install City of Books" })).toBeNull();
        expect(window.localStorage.getItem(DISMISSED_KEY)).toBe("1");
    });
});

describe("InstallPrompt on iOS Safari, which fires no install event", () => {
    it("explains the Share menu route instead of offering a button", () => {
        // given
        setUserAgent(IOS_SAFARI_UA);

        // when
        renderWithProviders(<InstallPrompt />);

        // then
        expect(screen.getByText(/tap Share, then Add to Home Screen/)).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Install" })).toBeNull();
    });

    it("stays hidden in a browser on iOS that cannot install web apps", () => {
        // given
        setUserAgent(IOS_CHROME_UA);

        // when
        renderWithProviders(<InstallPrompt />);

        // then
        expect(screen.queryByRole("region", { name: "Install City of Books" })).toBeNull();
    });
});

describe("InstallPrompt where installing is not on offer", () => {
    it("stays hidden inside the native app", () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(true);
        renderWithProviders(<InstallPrompt />);

        // when
        fireInstallPrompt(() => Promise.resolve());

        // then
        expect(screen.queryByRole("region", { name: "Install City of Books" })).toBeNull();
    });

    it("stays hidden once the site is already running as an installed app", () => {
        // given
        setStandalone(true);
        renderWithProviders(<InstallPrompt />);

        // when
        fireInstallPrompt(() => Promise.resolve());

        // then
        expect(screen.queryByRole("region", { name: "Install City of Books" })).toBeNull();
    });

    it("stays hidden when it was dismissed on an earlier visit", () => {
        // given
        window.localStorage.setItem(DISMISSED_KEY, "1");
        renderWithProviders(<InstallPrompt />);

        // when
        fireInstallPrompt(() => Promise.resolve());

        // then
        expect(screen.queryByRole("region", { name: "Install City of Books" })).toBeNull();
    });
});

describe("InstallPrompt when the browser fired beforeinstallprompt before React mounted", () => {
    it("still offers the install, because the event is captured at startup", () => {
        // given
        fireInstallPrompt(() => Promise.resolve());

        // when
        renderWithProviders(<InstallPrompt />);

        // then
        expect(screen.getByRole("button", { name: "Install" })).toBeInTheDocument();
    });
});
