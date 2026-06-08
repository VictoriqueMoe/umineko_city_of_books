import { Capacitor } from "@capacitor/core";
import { App } from "@capacitor/app";
import { CapacitorUpdater, type BundleInfo } from "@capgo/capacitor-updater";
import { apiUrl } from "../api/client";

interface OtaManifest {
    version: string;
    path: string;
}

let checking = false;
let pending: BundleInfo | null = null;

async function notifyReady(): Promise<void> {
    try {
        await CapacitorUpdater.notifyAppReady();
    } catch {}
}

async function downloadLatest(): Promise<void> {
    if (checking) {
        return;
    }

    checking = true;
    try {
        const response = await fetch(apiUrl("/app-bundles/latest.json"), { cache: "no-store" });
        if (!response.ok) {
            return;
        }

        const manifest = (await response.json()) as OtaManifest;
        if (!manifest.version || !manifest.path) {
            return;
        }

        const current = await CapacitorUpdater.current();
        if (current.bundle.version === manifest.version || pending?.version === manifest.version) {
            return;
        }

        pending = await CapacitorUpdater.download({ url: apiUrl(manifest.path), version: manifest.version });
    } catch {
    } finally {
        checking = false;
    }
}

async function applyPending(): Promise<void> {
    if (!pending) {
        return;
    }

    const bundle = pending;
    pending = null;
    try {
        await CapacitorUpdater.set(bundle);
    } catch {}
}

export function initAppUpdates(): void {
    if (!Capacitor.isNativePlatform()) {
        return;
    }

    notifyReady()
        .then(downloadLatest)
        .catch(() => {});

    App.addListener("appStateChange", state => {
        if (state.isActive) {
            downloadLatest().catch(() => {});
        } else {
            applyPending().catch(() => {});
        }
    }).catch(() => {});
}
