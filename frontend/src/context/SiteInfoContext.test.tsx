import { screen } from "@testing-library/react";
import { useContext } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { makeSiteInfo } from "../test-utils/fixtures";
import { renderWithProviders } from "../test-utils/render";
import { SiteInfoProvider } from "./SiteInfoContext";
import { SiteInfoContext } from "./siteInfoContextValue";

const { useSiteInfoQuery } = vi.hoisted(() => ({ useSiteInfoQuery: vi.fn() }));

vi.mock("../api/queries/auth", () => ({ useSiteInfoQuery }));

function Probe() {
    const siteInfo = useContext(SiteInfoContext);
    if (!siteInfo) {
        return <p>no site info</p>;
    }

    return <p>{`site: ${siteInfo.site_name}`}</p>;
}

function stubQuery(siteInfo: ReturnType<typeof makeSiteInfo> | null, dataUpdatedAt: number) {
    const refresh = vi.fn().mockResolvedValue(undefined);
    useSiteInfoQuery.mockReturnValue({ siteInfo, loading: false, refresh, dataUpdatedAt });

    return refresh;
}

function setVisibility(state: "visible" | "hidden") {
    Object.defineProperty(document, "visibilityState", { configurable: true, get: () => state });
}

interface RejectionWatcher {
    on(event: "unhandledRejection", listener: (reason: unknown) => void): void;
    off(event: "unhandledRejection", listener: (reason: unknown) => void): void;
}

const rejectionWatcher = (globalThis as unknown as { process: RejectionWatcher }).process;

afterEach(() => {
    Reflect.deleteProperty(document, "visibilityState");
});

describe("SiteInfoProvider", () => {
    it("renders nothing at all until the site info has loaded", () => {
        // given
        stubQuery(null, 0);

        // when
        const { container } = renderWithProviders(
            <SiteInfoProvider>
                <Probe />
            </SiteInfoProvider>,
        );

        // then
        expect(container).toBeEmptyDOMElement();
        expect(screen.queryByText("no site info")).not.toBeInTheDocument();
    });

    it("hands the loaded site info down to its children", () => {
        // given
        stubQuery(makeSiteInfo({ site_name: "City of Books" }), Date.now());

        // when
        renderWithProviders(
            <SiteInfoProvider>
                <Probe />
            </SiteInfoProvider>,
        );

        // then
        expect(screen.getByText("site: City of Books")).toBeInTheDocument();
    });

    it("refetches when a site-info-refresh event arrives and the data is stale", () => {
        // given
        const refresh = stubQuery(makeSiteInfo(), Date.now() - 10_000);
        renderWithProviders(
            <SiteInfoProvider>
                <Probe />
            </SiteInfoProvider>,
        );

        // when
        window.dispatchEvent(new Event("site-info-refresh"));

        // then
        expect(refresh).toHaveBeenCalledOnce();
    });

    it("swallows a failed refetch instead of leaving the rejection unhandled", async () => {
        // given
        let calls = 0;
        function failingRefresh() {
            calls += 1;
            return Promise.reject(new Error("site info is unreachable"));
        }
        useSiteInfoQuery.mockReturnValue({
            siteInfo: makeSiteInfo(),
            loading: false,
            refresh: failingRefresh,
            dataUpdatedAt: Date.now() - 10_000,
        });
        const unhandled: unknown[] = [];
        const record = (reason: unknown) => unhandled.push(reason);
        rejectionWatcher.on("unhandledRejection", record);
        renderWithProviders(
            <SiteInfoProvider>
                <Probe />
            </SiteInfoProvider>,
        );

        // when
        window.dispatchEvent(new Event("site-info-refresh"));
        await new Promise(resolve => setTimeout(resolve, 0));
        rejectionWatcher.off("unhandledRejection", record);

        // then
        expect(calls).toBe(1);
        expect(unhandled).toEqual([]);
    });

    it("ignores a refresh event that arrives while the data is still fresh", () => {
        // given
        const refresh = stubQuery(makeSiteInfo(), Date.now());
        renderWithProviders(
            <SiteInfoProvider>
                <Probe />
            </SiteInfoProvider>,
        );

        // when
        window.dispatchEvent(new Event("site-info-refresh"));

        // then
        expect(refresh).not.toHaveBeenCalled();
    });

    it("refetches once the data is exactly as old as the minimum interval", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        const refresh = stubQuery(makeSiteInfo(), Date.now() - 2000);
        renderWithProviders(
            <SiteInfoProvider>
                <Probe />
            </SiteInfoProvider>,
        );

        // when
        window.dispatchEvent(new Event("site-info-refresh"));

        // then
        expect(refresh).toHaveBeenCalledOnce();
    });

    it("refetches when the tab becomes visible again", () => {
        // given
        setVisibility("visible");
        const refresh = stubQuery(makeSiteInfo(), Date.now() - 10_000);
        renderWithProviders(
            <SiteInfoProvider>
                <Probe />
            </SiteInfoProvider>,
        );

        // when
        document.dispatchEvent(new Event("visibilitychange"));

        // then
        expect(refresh).toHaveBeenCalledOnce();
    });

    it("does not refetch on a visibility change that hides the tab", () => {
        // given
        setVisibility("hidden");
        const refresh = stubQuery(makeSiteInfo(), Date.now() - 10_000);
        renderWithProviders(
            <SiteInfoProvider>
                <Probe />
            </SiteInfoProvider>,
        );

        // when
        document.dispatchEvent(new Event("visibilitychange"));

        // then
        expect(refresh).not.toHaveBeenCalled();
    });

    it("stops listening for refresh events once it is unmounted", () => {
        // given
        const refresh = stubQuery(makeSiteInfo(), Date.now() - 10_000);
        const { unmount } = renderWithProviders(
            <SiteInfoProvider>
                <Probe />
            </SiteInfoProvider>,
        );

        // when
        unmount();
        window.dispatchEvent(new Event("site-info-refresh"));
        document.dispatchEvent(new Event("visibilitychange"));

        // then
        expect(refresh).not.toHaveBeenCalled();
    });
});
