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

        const bundle = await CapacitorUpdater.download({ url: apiUrl(manifest.path), version: manifest.version });
        await CapacitorUpdater.next({ id: bundle.id });
        pending = bundle;
        window.dispatchEvent(new CustomEvent("ota-update-ready"));
    } catch {
    } finally {
        checking = false;
    }
}

export function hasOtaUpdate(): boolean {
    return pending !== null;
}

export async function applyOtaUpdate(): Promise<void> {
    if (!pending) {
        return;
    }

    try {
        await CapacitorUpdater.set({ id: pending.id });
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
        }
    }).catch(() => {});
}
