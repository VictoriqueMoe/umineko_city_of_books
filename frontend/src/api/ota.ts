import { apiUrl } from "./client";
import type { OtaManifest } from "../types/api";

const MANIFEST_PATH = "/app-bundles/latest.json";

export async function getOtaManifest(): Promise<OtaManifest | null> {
    const response = await fetch(apiUrl(MANIFEST_PATH), { cache: "no-store" });

    if (!response.ok) {
        return null;
    }

    return (await response.json()) as OtaManifest;
}

export function otaBundleUrl(path: string): string {
    return apiUrl(path);
}
