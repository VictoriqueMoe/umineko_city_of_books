import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import type { OtaManifest } from "../types/api";
import type { PlatformErrorReporter } from "./errorReporter";

type AppStateHandler = (state: { isActive: boolean }) => void;

const capacitor = vi.hoisted(() => ({
    isNativePlatform: vi.fn(),
}));

const capacitorApp = vi.hoisted(() => ({
    addListener: vi.fn(),
}));

const updater = vi.hoisted(() => ({
    notifyAppReady: vi.fn(),
    current: vi.fn(),
    download: vi.fn(),
    next: vi.fn(),
    set: vi.fn(),
}));

vi.mock("@capacitor/core", () => ({ Capacitor: capacitor }));
vi.mock("@capacitor/app", () => ({ App: capacitorApp }));
vi.mock("@capgo/capacitor-updater", () => ({ CapacitorUpdater: updater }));

const getManifest = vi.fn();
const bundleUrl = vi.fn((path: string) => `https://api.test${path}`);
const source = { getManifest, bundleUrl };
const otaReadyListener = vi.fn();

let report: Mock<PlatformErrorReporter>;
let stateHandler: AppStateHandler | null = null;

function signedManifest(): OtaManifest {
    return { version: "1.1.0", path: "/app-bundles/1.1.0.zip", checksum: "enc-checksum", session_key: "iv:session" };
}

async function loadAppUpdateModule(): Promise<typeof import("./appUpdate")> {
    vi.resetModules();
    const [appUpdate, errorReporter] = await Promise.all([import("./appUpdate"), import("./errorReporter")]);

    report = vi.fn<PlatformErrorReporter>();
    errorReporter.setPlatformErrorReporter(report);

    return appUpdate;
}

beforeEach(() => {
    stateHandler = null;
    capacitor.isNativePlatform.mockReturnValue(true);
    capacitorApp.addListener.mockImplementation((_event: string, handler: AppStateHandler) => {
        stateHandler = handler;
        return Promise.resolve({ remove: () => Promise.resolve() });
    });
    updater.notifyAppReady.mockResolvedValue(undefined);
    updater.current.mockResolvedValue({ bundle: { id: "bundle-1", version: "1.0.0" } });
    updater.download.mockResolvedValue({ id: "bundle-2", version: "1.1.0" });
    updater.next.mockResolvedValue(undefined);
    updater.set.mockResolvedValue(undefined);
    getManifest.mockReset();
    getManifest.mockResolvedValue(signedManifest());
    otaReadyListener.mockReset();
});

describe("initAppUpdates", () => {
    it("does nothing at all in a browser session", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(false);
        const { initAppUpdates } = await loadAppUpdateModule();

        // when
        initAppUpdates(source);

        // then
        expect(updater.notifyAppReady).not.toHaveBeenCalled();
        expect(getManifest).not.toHaveBeenCalled();
        expect(capacitorApp.addListener).not.toHaveBeenCalled();
    });

    it("tells the updater the running bundle is healthy before looking for a new one", async () => {
        // given
        const { initAppUpdates } = await loadAppUpdateModule();

        // when
        initAppUpdates(source);

        // then
        await vi.waitFor(() => {
            expect(getManifest).toHaveBeenCalledOnce();
        });
        expect(updater.notifyAppReady).toHaveBeenCalledOnce();
        expect(updater.notifyAppReady.mock.invocationCallOrder[0]).toBeLessThan(
            getManifest.mock.invocationCallOrder[0],
        );
    });

    it("still checks for a bundle when notifying readiness fails", async () => {
        // given
        updater.notifyAppReady.mockRejectedValue(new Error("no updater"));
        const { initAppUpdates } = await loadAppUpdateModule();

        // when
        initAppUpdates(source);

        // then
        await vi.waitFor(() => {
            expect(getManifest).toHaveBeenCalledOnce();
        });
    });

    it("reports a failed readiness notification, which otherwise rolls the bundle back on the next launch", async () => {
        // given
        const failure = new Error("no updater");
        updater.notifyAppReady.mockRejectedValue(failure);
        const { initAppUpdates } = await loadAppUpdateModule();

        // when
        initAppUpdates(source);

        // then
        await vi.waitFor(() => {
            expect(report).toHaveBeenCalledWith(failure, { source: "ota-update" });
        });
    });

    it("downloads a newer bundle, stages it and announces that it is ready", async () => {
        // given
        const appUpdate = await loadAppUpdateModule();
        appUpdate.subscribeOtaReady(otaReadyListener);

        // when
        appUpdate.initAppUpdates(source);

        // then
        await vi.waitFor(() => {
            expect(updater.next).toHaveBeenCalledWith({ id: "bundle-2" });
        });
        expect(updater.download).toHaveBeenCalledWith({
            url: "https://api.test/app-bundles/1.1.0.zip",
            version: "1.1.0",
            checksum: "enc-checksum",
            sessionKey: "iv:session",
        });
        expect(otaReadyListener).toHaveBeenCalledOnce();
        expect(appUpdate.hasOtaUpdate()).toBe(true);
    });

    it("leaves the running bundle alone when the manifest matches it", async () => {
        // given
        updater.current.mockResolvedValue({ bundle: { id: "bundle-1", version: "1.1.0" } });
        const appUpdate = await loadAppUpdateModule();
        appUpdate.subscribeOtaReady(otaReadyListener);

        // when
        appUpdate.initAppUpdates(source);

        // then
        await vi.waitFor(() => {
            expect(updater.current).toHaveBeenCalledOnce();
        });
        expect(updater.download).not.toHaveBeenCalled();
        expect(otaReadyListener).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("ignores a manifest request that came back with an error status", async () => {
        // given
        getManifest.mockResolvedValue(null);
        const appUpdate = await loadAppUpdateModule();

        // when
        appUpdate.initAppUpdates(source);

        // then
        await vi.waitFor(() => {
            expect(getManifest).toHaveBeenCalledOnce();
        });
        expect(updater.current).not.toHaveBeenCalled();
        expect(updater.download).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("ignores a manifest that is missing the bundle version", async () => {
        // given
        getManifest.mockResolvedValue({ ...signedManifest(), version: undefined });
        const appUpdate = await loadAppUpdateModule();

        // when
        appUpdate.initAppUpdates(source);

        // then
        await vi.waitFor(() => {
            expect(getManifest).toHaveBeenCalledOnce();
        });
        expect(updater.download).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("ignores a manifest that is missing the bundle path", async () => {
        // given
        getManifest.mockResolvedValue({ ...signedManifest(), path: undefined });
        const appUpdate = await loadAppUpdateModule();

        // when
        appUpdate.initAppUpdates(source);

        // then
        await vi.waitFor(() => {
            expect(getManifest).toHaveBeenCalledOnce();
        });
        expect(updater.download).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("refuses an unsigned manifest that carries no checksum", async () => {
        // given
        getManifest.mockResolvedValue({ ...signedManifest(), checksum: undefined });
        const appUpdate = await loadAppUpdateModule();

        // when
        appUpdate.initAppUpdates(source);

        // then
        await vi.waitFor(() => {
            expect(getManifest).toHaveBeenCalledOnce();
        });
        expect(updater.download).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("refuses a manifest that carries no session key", async () => {
        // given
        getManifest.mockResolvedValue({ ...signedManifest(), session_key: undefined });
        const appUpdate = await loadAppUpdateModule();

        // when
        appUpdate.initAppUpdates(source);

        // then
        await vi.waitFor(() => {
            expect(getManifest).toHaveBeenCalledOnce();
        });
        expect(updater.download).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("checks again when the app comes back to the foreground", async () => {
        // given
        updater.current.mockResolvedValue({ bundle: { id: "bundle-1", version: "1.1.0" } });
        const { initAppUpdates } = await loadAppUpdateModule();
        initAppUpdates(source);
        await vi.waitFor(() => {
            expect(getManifest).toHaveBeenCalledOnce();
        });

        // when
        stateHandler?.({ isActive: true });

        // then
        await vi.waitFor(() => {
            expect(getManifest).toHaveBeenCalledTimes(2);
        });
    });

    it("does not check when the app goes to the background", async () => {
        // given
        updater.current.mockResolvedValue({ bundle: { id: "bundle-1", version: "1.1.0" } });
        const { initAppUpdates } = await loadAppUpdateModule();
        initAppUpdates(source);
        await vi.waitFor(() => {
            expect(getManifest).toHaveBeenCalledOnce();
        });

        // when
        stateHandler?.({ isActive: false });
        await Promise.resolve();

        // then
        expect(getManifest).toHaveBeenCalledOnce();
    });

    it("reports a failed manifest request and is ready to try again later", async () => {
        // given
        const failure = new Error("offline");
        getManifest.mockRejectedValueOnce(failure);
        const appUpdate = await loadAppUpdateModule();
        appUpdate.initAppUpdates(source);
        await vi.waitFor(() => {
            expect(report).toHaveBeenCalledWith(failure, { source: "ota-update" });
        });
        expect(appUpdate.hasOtaUpdate()).toBe(false);

        // when
        stateHandler?.({ isActive: true });

        // then
        await vi.waitFor(() => {
            expect(updater.download).toHaveBeenCalledOnce();
        });
        expect(appUpdate.hasOtaUpdate()).toBe(true);
    });

    it("reports a failure to subscribe to the foreground signal", async () => {
        // given
        const failure = new Error("the app plugin is missing");
        capacitorApp.addListener.mockRejectedValue(failure);
        const { initAppUpdates } = await loadAppUpdateModule();

        // when
        initAppUpdates(source);

        // then
        await vi.waitFor(() => {
            expect(report).toHaveBeenCalledWith(failure, { source: "ota-update" });
        });
    });

    it("does not download a bundle twice once it is already staged", async () => {
        // given
        const appUpdate = await loadAppUpdateModule();
        appUpdate.subscribeOtaReady(otaReadyListener);
        appUpdate.initAppUpdates(source);
        await vi.waitFor(() => {
            expect(updater.download).toHaveBeenCalledOnce();
        });

        // when
        stateHandler?.({ isActive: true });
        await vi.waitFor(() => {
            expect(getManifest).toHaveBeenCalledTimes(2);
        });

        // then
        expect(updater.download).toHaveBeenCalledOnce();
        expect(otaReadyListener).toHaveBeenCalledOnce();
    });
});

describe("hasOtaUpdate", () => {
    it("reports nothing waiting before any check has run", async () => {
        // given
        const { hasOtaUpdate } = await loadAppUpdateModule();

        // when
        const waiting = hasOtaUpdate();

        // then
        expect(waiting).toBe(false);
    });
});

describe("subscribeOtaReady", () => {
    it("stops announcing to a listener that has unsubscribed", async () => {
        // given
        const appUpdate = await loadAppUpdateModule();
        const stop = appUpdate.subscribeOtaReady(otaReadyListener);

        // when
        stop();
        appUpdate.initAppUpdates(source);
        await vi.waitFor(() => {
            expect(appUpdate.hasOtaUpdate()).toBe(true);
        });

        // then
        expect(otaReadyListener).not.toHaveBeenCalled();
    });
});

describe("applyOtaUpdate", () => {
    it("does nothing when no bundle is waiting", async () => {
        // given
        const { applyOtaUpdate } = await loadAppUpdateModule();

        // when
        await applyOtaUpdate();

        // then
        expect(updater.set).not.toHaveBeenCalled();
    });

    it("activates the staged bundle", async () => {
        // given
        const appUpdate = await loadAppUpdateModule();
        appUpdate.initAppUpdates(source);
        await vi.waitFor(() => {
            expect(appUpdate.hasOtaUpdate()).toBe(true);
        });

        // when
        await appUpdate.applyOtaUpdate();

        // then
        expect(updater.set).toHaveBeenCalledWith({ id: "bundle-2" });
    });

    it("reports a failure to activate the staged bundle rather than leaving the reader guessing", async () => {
        // given
        const failure = new Error("cannot switch bundle");
        updater.set.mockRejectedValue(failure);
        const appUpdate = await loadAppUpdateModule();
        appUpdate.initAppUpdates(source);
        await vi.waitFor(() => {
            expect(appUpdate.hasOtaUpdate()).toBe(true);
        });

        // when
        const result = appUpdate.applyOtaUpdate();

        // then
        await expect(result).resolves.toBeUndefined();
        expect(report).toHaveBeenCalledWith(failure, { source: "ota-update" });
    });
});
