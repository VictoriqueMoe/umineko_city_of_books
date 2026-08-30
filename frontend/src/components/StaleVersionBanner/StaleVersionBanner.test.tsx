import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { StaleVersionBanner } from "./StaleVersionBanner";

const { isNativeApp, hasOtaUpdate, applyOtaUpdate, subscribeOtaReady } = vi.hoisted(() => ({
    isNativeApp: vi.fn(),
    hasOtaUpdate: vi.fn(),
    applyOtaUpdate: vi.fn(),
    subscribeOtaReady: vi.fn(),
}));

vi.mock("../../platform/capabilities", () => ({ isNativeApp }));
vi.mock("../../platform/appUpdate", () => ({ hasOtaUpdate, applyOtaUpdate, subscribeOtaReady }));

const reload = vi.fn();

let readyListeners: Array<() => void>;

beforeEach(() => {
    readyListeners = [];
    isNativeApp.mockReturnValue(false);
    hasOtaUpdate.mockReturnValue(false);
    applyOtaUpdate.mockResolvedValue(undefined);
    subscribeOtaReady.mockImplementation((listener: () => void) => {
        readyListeners.push(listener);

        return () => {
            readyListeners = readyListeners.filter(entry => entry !== listener);
        };
    });
    vi.stubGlobal("__APP_VERSION__", "6.10.0");
    vi.stubGlobal("location", { reload, origin: "http://localhost:3000", href: "http://localhost:3000/" });
});

function announceOtaReady() {
    act(() => {
        for (const listener of readyListeners) {
            listener();
        }
    });
}

describe("StaleVersionBanner", () => {
    it("stays quiet while the browser is running the version the site serves", () => {
        // given
        const version = "6.10.0";

        // when
        const { container } = renderWithProviders(<StaleVersionBanner />, { siteInfo: { version } });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("asks the reader to reload when the site has moved on", () => {
        // given
        const version = "6.11.0";

        // when
        renderWithProviders(<StaleVersionBanner />, { siteInfo: { version } });

        // then
        expect(screen.getByRole("alert")).toHaveTextContent(
            "A new version of the site is available. Please reload to update.",
        );
    });

    it("reloads the page when the reader accepts", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<StaleVersionBanner />, { siteInfo: { version: "6.11.0" } });

        // when
        await user.click(screen.getByRole("button", { name: "Reload now" }));

        // then
        expect(reload).toHaveBeenCalledOnce();
    });

    it("stays quiet for a locally built bundle", () => {
        // given
        vi.stubGlobal("__APP_VERSION__", "dev");

        // when
        const { container } = renderWithProviders(<StaleVersionBanner />, { siteInfo: { version: "6.11.0" } });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("stays quiet when the site reports a dev version", () => {
        // given
        const version = "dev";

        // when
        const { container } = renderWithProviders(<StaleVersionBanner />, { siteInfo: { version } });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("stays quiet when the site does not report a version at all", () => {
        // given
        const version = "";

        // when
        const { container } = renderWithProviders(<StaleVersionBanner />, { siteInfo: { version } });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("ignores an over the air announcement in a browser session", () => {
        // given
        const { container } = renderWithProviders(<StaleVersionBanner />, { siteInfo: { version: "6.10.0" } });

        // when
        announceOtaReady();

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("stays quiet in the app until a bundle has been downloaded", () => {
        // given
        isNativeApp.mockReturnValue(true);

        // when
        const { container } = renderWithProviders(<StaleVersionBanner />, { siteInfo: { version: "6.11.0" } });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("offers the update straight away when a bundle was already staged", () => {
        // given
        isNativeApp.mockReturnValue(true);
        hasOtaUpdate.mockReturnValue(true);

        // when
        renderWithProviders(<StaleVersionBanner />, { siteInfo: { version: "6.10.0" } });

        // then
        expect(screen.getByRole("alert")).toHaveTextContent("A new version is available. Tap to update now.");
        expect(screen.getByRole("button", { name: "Update now" })).toBeInTheDocument();
    });

    it("offers the update as soon as a bundle finishes downloading", () => {
        // given
        isNativeApp.mockReturnValue(true);
        renderWithProviders(<StaleVersionBanner />, { siteInfo: { version: "6.10.0" } });

        // when
        announceOtaReady();

        // then
        expect(screen.getByRole("button", { name: "Update now" })).toBeInTheDocument();
    });

    it("applies the staged bundle when the reader taps update", async () => {
        // given
        isNativeApp.mockReturnValue(true);
        hasOtaUpdate.mockReturnValue(true);
        const user = userEvent.setup();
        renderWithProviders(<StaleVersionBanner />, { siteInfo: { version: "6.10.0" } });

        // when
        await user.click(screen.getByRole("button", { name: "Update now" }));

        // then
        expect(applyOtaUpdate).toHaveBeenCalledOnce();
        expect(reload).not.toHaveBeenCalled();
    });

    it("swallows a failure to apply the staged bundle", async () => {
        // given
        isNativeApp.mockReturnValue(true);
        hasOtaUpdate.mockReturnValue(true);
        applyOtaUpdate.mockRejectedValue(new Error("bundle is corrupt"));
        const user = userEvent.setup();
        renderWithProviders(<StaleVersionBanner />, { siteInfo: { version: "6.10.0" } });

        // when
        await user.click(screen.getByRole("button", { name: "Update now" }));

        // then
        expect(screen.getByRole("button", { name: "Update now" })).toBeInTheDocument();
    });
});
