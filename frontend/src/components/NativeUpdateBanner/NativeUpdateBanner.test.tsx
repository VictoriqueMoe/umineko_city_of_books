import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SiteInfo } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import { NativeUpdateBanner } from "./NativeUpdateBanner";

const { capacitor } = vi.hoisted(() => ({ capacitor: { isNativePlatform: vi.fn() } }));
const { capacitorApp } = vi.hoisted(() => ({ capacitorApp: { getInfo: vi.fn() } }));

vi.mock("@capacitor/core", () => ({ Capacitor: capacitor }));
vi.mock("@capacitor/app", () => ({ App: capacitorApp }));

const advertised: Partial<SiteInfo> = {
    app_latest_version: "1.1.0",
    app_download_url: "https://whentheycry.social/app.apk",
};

function renderBanner(siteInfo: Partial<SiteInfo> = advertised) {
    return renderWithProviders(<NativeUpdateBanner />, { siteInfo });
}

async function settleAppInfo(): Promise<void> {
    await act(async () => {
        await Promise.resolve();
        await Promise.resolve();
    });
}

beforeEach(() => {
    capacitor.isNativePlatform.mockReturnValue(true);
    capacitorApp.getInfo.mockResolvedValue({ version: "1.0.0" });
});

describe("NativeUpdateBanner when an update is waiting", () => {
    it("tells the visitor a newer version of the app exists", async () => {
        // given
        renderBanner();

        // when
        const banner = await screen.findByRole("alert");

        // then
        expect(banner).toHaveTextContent("A new version of the app is available.");
    });

    it("opens the download in a new tab when the button is pressed", async () => {
        // given
        const open = vi.spyOn(window, "open").mockReturnValue(null);
        const user = userEvent.setup();
        renderBanner();
        await screen.findByRole("alert");

        // when
        await user.click(screen.getByRole("button", { name: "Download update" }));

        // then
        expect(open).toHaveBeenCalledWith("https://whentheycry.social/app.apk", "_blank");
        open.mockRestore();
    });

    it("compares each part of the version as a number rather than as text", async () => {
        // given
        capacitorApp.getInfo.mockResolvedValue({ version: "1.9.0" });

        // when
        renderBanner({ ...advertised, app_latest_version: "1.10.0" });

        // then
        expect(await screen.findByRole("alert")).toBeInTheDocument();
    });

    it("still offers the update when the advertised version carries a pre-release suffix", async () => {
        // given
        capacitorApp.getInfo.mockResolvedValue({ version: "1.1.0" });

        // when
        renderBanner({ ...advertised, app_latest_version: "1.1.1-rc" });

        // then
        expect(await screen.findByRole("alert")).toBeInTheDocument();
    });

    it("still offers the update when the installed build carries a pre-release suffix", async () => {
        // given
        capacitorApp.getInfo.mockResolvedValue({ version: "1.0.0-beta" });

        // when
        renderBanner({ ...advertised, app_latest_version: "1.0.1" });

        // then
        expect(await screen.findByRole("alert")).toBeInTheDocument();
    });

    it("treats a version part the installed build leaves out as a zero", async () => {
        // given
        capacitorApp.getInfo.mockResolvedValue({ version: "1.2" });

        // when
        renderBanner({ ...advertised, app_latest_version: "1.2.1" });

        // then
        expect(await screen.findByRole("alert")).toBeInTheDocument();
    });
});

describe("NativeUpdateBanner when there is nothing to install", () => {
    it("stays out of the way when the installed version is the advertised one", async () => {
        // given
        capacitorApp.getInfo.mockResolvedValue({ version: "1.1.0" });
        renderBanner();

        // when
        await settleAppInfo();

        // then
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    it("stays out of the way when the installed version is ahead of the advertised one", async () => {
        // given
        capacitorApp.getInfo.mockResolvedValue({ version: "2.0.0" });
        renderBanner();

        // when
        await settleAppInfo();

        // then
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    it("counts a missing trailing version part as equal", async () => {
        // given
        capacitorApp.getInfo.mockResolvedValue({ version: "1.1" });
        renderBanner({ ...advertised, app_latest_version: "1.1.0" });

        // when
        await settleAppInfo();

        // then
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    it("stays out of the way when the site advertises no version", async () => {
        // given
        renderBanner({ ...advertised, app_latest_version: "" });

        // when
        await settleAppInfo();

        // then
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    it("stays out of the way when the site advertises no download link", async () => {
        // given
        renderBanner({ ...advertised, app_download_url: "" });

        // when
        await settleAppInfo();

        // then
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    it("stays out of the way while the installed version is still unknown", () => {
        // given
        capacitorApp.getInfo.mockReturnValue(new Promise(() => {}));

        // when
        renderBanner();

        // then
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    it("stays out of the way when the installed version cannot be read", async () => {
        // given
        capacitorApp.getInfo.mockRejectedValue(new Error("no app info"));
        renderBanner();

        // when
        await settleAppInfo();

        // then
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });
});

describe("NativeUpdateBanner on the web", () => {
    beforeEach(() => {
        capacitor.isNativePlatform.mockReturnValue(false);
    });

    it("never asks the shell which version is installed", async () => {
        // given
        renderBanner();

        // when
        await settleAppInfo();

        // then
        expect(capacitorApp.getInfo).not.toHaveBeenCalled();
    });

    it("shows nothing even when a newer version is advertised", async () => {
        // given
        const { container } = renderBanner();

        // when
        await settleAppInfo();

        // then
        expect(container).toBeEmptyDOMElement();
    });
});
