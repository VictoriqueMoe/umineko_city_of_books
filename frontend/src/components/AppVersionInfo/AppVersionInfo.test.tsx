import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AppVersionInfo } from "./AppVersionInfo";

const { isNativeApp, getInfo, current } = vi.hoisted(() => ({
    isNativeApp: vi.fn(),
    getInfo: vi.fn(),
    current: vi.fn(),
}));

vi.mock("../../utils/authToken", () => ({ isNativeApp }));
vi.mock("@capacitor/app", () => ({ App: { getInfo } }));
vi.mock("@capgo/capacitor-updater", () => ({ CapacitorUpdater: { current } }));

beforeEach(() => {
    isNativeApp.mockReturnValue(true);
    getInfo.mockResolvedValue({ version: "1.4.0", build: "42" });
    current.mockResolvedValue({ bundle: { version: "2026.07.01" } });
});

describe("AppVersionInfo", () => {
    it("stays out of the way in a browser session", async () => {
        // given
        isNativeApp.mockReturnValue(false);

        // when
        const { container } = render(<AppVersionInfo />);

        // then
        await Promise.resolve();
        expect(container).toBeEmptyDOMElement();
        expect(getInfo).not.toHaveBeenCalled();
        expect(current).not.toHaveBeenCalled();
    });

    it("reports the installed app build and the running bundle", async () => {
        // given
        getInfo.mockResolvedValue({ version: "1.4.0", build: "42" });

        // when
        render(<AppVersionInfo />);

        // then
        expect(await screen.findByText("app 1.4.0 (42) · bundle 2026.07.01")).toBeInTheDocument();
    });

    it("shortens a long bundle version so it stays readable", async () => {
        // given
        current.mockResolvedValue({ bundle: { version: "0123456789abcdefgh" } });

        // when
        render(<AppVersionInfo />);

        // then
        expect(await screen.findByText("app 1.4.0 (42) · bundle 0123456789ab")).toBeInTheDocument();
    });

    it("calls an unnamed bundle a dev bundle", async () => {
        // given
        current.mockResolvedValue({ bundle: { version: "" } });

        // when
        render(<AppVersionInfo />);

        // then
        expect(await screen.findByText("app 1.4.0 (42) · bundle dev")).toBeInTheDocument();
    });

    it("falls back to a dev bundle when the updater cannot say what is running", async () => {
        // given
        current.mockRejectedValue(new Error("no updater plugin"));

        // when
        render(<AppVersionInfo />);

        // then
        expect(await screen.findByText("app 1.4.0 (42) · bundle dev")).toBeInTheDocument();
    });

    it("still reports the bundle when the app info is unavailable", async () => {
        // given
        getInfo.mockRejectedValue(new Error("no app plugin"));

        // when
        render(<AppVersionInfo />);

        // then
        expect(await screen.findByText("bundle 2026.07.01")).toBeInTheDocument();
    });
});
