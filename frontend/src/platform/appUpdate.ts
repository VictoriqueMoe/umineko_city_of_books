import { Capacitor } from "@capacitor/core";
import { App } from "@capacitor/app";
import { CapacitorUpdater, type BundleInfo } from "@capgo/capacitor-updater";
import type { OtaManifest } from "../types/api";
import { reportPlatformError } from "./errorReporter";

export interface OtaSource {
    getManifest: () => Promise<OtaManifest | null>;
    bundleUrl: (path: string) => string;
}

type ReadyListener = () => void;

let checking = false;
let pending: BundleInfo | null = null;
const readyListeners = new Set<ReadyListener>();

function announceReady(): void {
    for (const listener of readyListeners) {
        listener();
    }
}

async function notifyReady(): Promise<void> {
    try {
        await CapacitorUpdater.notifyAppReady();
    } catch (error) {
        reportPlatformError(error, "ota-update");
    }
}

async function downloadLatest({ getManifest, bundleUrl }: OtaSource): Promise<void> {
    if (checking) {
        return;
    }

    checking = true;
    try {
        const manifest = await getManifest();
        if (!manifest?.version || !manifest.path || !manifest.checksum || !manifest.session_key) {
            return;
        }

        const current = await CapacitorUpdater.current();
        if (current.bundle.version === manifest.version || pending?.version === manifest.version) {
            return;
        }

        const bundle = await CapacitorUpdater.download({
            url: bundleUrl(manifest.path),
            version: manifest.version,
            checksum: manifest.checksum,
            sessionKey: manifest.session_key,
        });
        await CapacitorUpdater.next({ id: bundle.id });
        pending = bundle;
        announceReady();
    } catch (error) {
        reportPlatformError(error, "ota-update");
    } finally {
        checking = false;
    }
}

export function hasOtaUpdate(): boolean {
    return pending !== null;
}

export function subscribeOtaReady(listener: ReadyListener): () => void {
    readyListeners.add(listener);

    return () => {
        readyListeners.delete(listener);
    };
}

export async function applyOtaUpdate(): Promise<void> {
    if (!pending) {
        return;
    }

    try {
        await CapacitorUpdater.set({ id: pending.id });
    } catch (error) {
        reportPlatformError(error, "ota-update");
    }
}

export function initAppUpdates(source: OtaSource): void {
    if (!Capacitor.isNativePlatform()) {
        return;
    }

    notifyReady()
        .then(() => downloadLatest(source))
        .catch(error => {
            reportPlatformError(error, "ota-update");
        });

    App.addListener("appStateChange", state => {
        if (state.isActive) {
            downloadLatest(source).catch(error => {
                reportPlatformError(error, "ota-update");
            });
        }
    }).catch(error => {
        reportPlatformError(error, "ota-update");
    });
}
