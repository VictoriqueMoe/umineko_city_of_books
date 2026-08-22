import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

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
vi.mock("../api/client", () => ({ apiUrl: (path: string) => `https://api.test${path}` }));

const fetchMock = vi.fn();
const otaReadyListener = vi.fn();

let stateHandler: AppStateHandler | null = null;

function signedManifest(): Record<string, string> {
    return { version: "1.1.0", path: "/app-bundles/1.1.0.zip", checksum: "enc-checksum", session_key: "iv:session" };
}

function manifestResponse(body: unknown, ok = true): Response {
    return { ok, json: () => Promise.resolve(body) } as unknown as Response;
}

async function loadAppUpdateModule(): Promise<typeof import("./appUpdate")> {
    vi.resetModules();
    return import("./appUpdate");
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
    fetchMock.mockResolvedValue(manifestResponse(signedManifest()));
    vi.stubGlobal("fetch", fetchMock);
    window.addEventListener("ota-update-ready", otaReadyListener);
});

afterEach(() => {
    window.removeEventListener("ota-update-ready", otaReadyListener);
});

describe("initAppUpdates", () => {
    it("does nothing at all in a browser session", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(false);
        const { initAppUpdates } = await loadAppUpdateModule();

        // when
        initAppUpdates();

        // then
        expect(updater.notifyAppReady).not.toHaveBeenCalled();
        expect(fetchMock).not.toHaveBeenCalled();
        expect(capacitorApp.addListener).not.toHaveBeenCalled();
    });

    it("tells the updater the running bundle is healthy before looking for a new one", async () => {
        // given
        const { initAppUpdates } = await loadAppUpdateModule();

        // when
        initAppUpdates();

        // then
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledWith("https://api.test/app-bundles/latest.json", { cache: "no-store" });
        });
        expect(updater.notifyAppReady).toHaveBeenCalledOnce();
        expect(updater.notifyAppReady.mock.invocationCallOrder[0]).toBeLessThan(fetchMock.mock.invocationCallOrder[0]);
    });

    it("still checks for a bundle when notifying readiness fails", async () => {
        // given
        updater.notifyAppReady.mockRejectedValue(new Error("no updater"));
        const { initAppUpdates } = await loadAppUpdateModule();

        // when
        initAppUpdates();

        // then
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledOnce();
        });
    });

    it("downloads a newer bundle, stages it and announces that it is ready", async () => {
        // given
        const appUpdate = await loadAppUpdateModule();

        // when
        appUpdate.initAppUpdates();

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

        // when
        appUpdate.initAppUpdates();

        // then
        await vi.waitFor(() => {
            expect(updater.current).toHaveBeenCalledOnce();
        });
        expect(updater.download).not.toHaveBeenCalled();
        expect(otaReadyListener).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("ignores a manifest request that comes back with an error status", async () => {
        // given
        fetchMock.mockResolvedValue(manifestResponse(signedManifest(), false));
        const appUpdate = await loadAppUpdateModule();

        // when
        appUpdate.initAppUpdates();

        // then
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledOnce();
        });
        expect(updater.current).not.toHaveBeenCalled();
        expect(updater.download).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("ignores a manifest that is missing the bundle version", async () => {
        // given
        fetchMock.mockResolvedValue(manifestResponse({ ...signedManifest(), version: undefined }));
        const appUpdate = await loadAppUpdateModule();

        // when
        appUpdate.initAppUpdates();

        // then
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledOnce();
        });
        expect(updater.download).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("ignores a manifest that is missing the bundle path", async () => {
        // given
        fetchMock.mockResolvedValue(manifestResponse({ ...signedManifest(), path: undefined }));
        const appUpdate = await loadAppUpdateModule();

        // when
        appUpdate.initAppUpdates();

        // then
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledOnce();
        });
        expect(updater.download).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("refuses an unsigned manifest that carries no checksum", async () => {
        // given
        fetchMock.mockResolvedValue(manifestResponse({ ...signedManifest(), checksum: undefined }));
        const appUpdate = await loadAppUpdateModule();

        // when
        appUpdate.initAppUpdates();

        // then
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledOnce();
        });
        expect(updater.download).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("refuses a manifest that carries no session key", async () => {
        // given
        fetchMock.mockResolvedValue(manifestResponse({ ...signedManifest(), session_key: undefined }));
        const appUpdate = await loadAppUpdateModule();

        // when
        appUpdate.initAppUpdates();

        // then
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledOnce();
        });
        expect(updater.download).not.toHaveBeenCalled();
        expect(appUpdate.hasOtaUpdate()).toBe(false);
    });

    it("checks again when the app comes back to the foreground", async () => {
        // given
        updater.current.mockResolvedValue({ bundle: { id: "bundle-1", version: "1.1.0" } });
        const { initAppUpdates } = await loadAppUpdateModule();
        initAppUpdates();
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledOnce();
        });

        // when
        stateHandler?.({ isActive: true });

        // then
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledTimes(2);
        });
    });

    it("does not check when the app goes to the background", async () => {
        // given
        updater.current.mockResolvedValue({ bundle: { id: "bundle-1", version: "1.1.0" } });
        const { initAppUpdates } = await loadAppUpdateModule();
        initAppUpdates();
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledOnce();
        });

        // when
        stateHandler?.({ isActive: false });
        await Promise.resolve();

        // then
        expect(fetchMock).toHaveBeenCalledOnce();
    });

    it("swallows a failed manifest request and is ready to try again later", async () => {
        // given
        fetchMock.mockRejectedValueOnce(new Error("offline"));
        const appUpdate = await loadAppUpdateModule();
        appUpdate.initAppUpdates();
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledOnce();
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

    it("does not download a bundle twice once it is already staged", async () => {
        // given
        const appUpdate = await loadAppUpdateModule();
        appUpdate.initAppUpdates();
        await vi.waitFor(() => {
            expect(updater.download).toHaveBeenCalledOnce();
        });

        // when
        stateHandler?.({ isActive: true });
        await vi.waitFor(() => {
            expect(fetchMock).toHaveBeenCalledTimes(2);
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
        appUpdate.initAppUpdates();
        await vi.waitFor(() => {
            expect(appUpdate.hasOtaUpdate()).toBe(true);
        });

        // when
        await appUpdate.applyOtaUpdate();

        // then
        expect(updater.set).toHaveBeenCalledWith({ id: "bundle-2" });
    });

    it("swallows a failure to activate the staged bundle", async () => {
        // given
        updater.set.mockRejectedValue(new Error("cannot switch bundle"));
        const appUpdate = await loadAppUpdateModule();
        appUpdate.initAppUpdates();
        await vi.waitFor(() => {
            expect(appUpdate.hasOtaUpdate()).toBe(true);
        });

        // when
        const result = appUpdate.applyOtaUpdate();

        // then
        await expect(result).resolves.toBeUndefined();
    });
});
