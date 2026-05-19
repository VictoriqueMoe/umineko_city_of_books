import { getRegionInfo } from "@hyperbeam/web";

const STORAGE_KEY = "hyperbeam_region";
const STORAGE_TTL_MS = 24 * 60 * 60 * 1000;

interface CachedRegion {
    region: string;
    storedAt: number;
}

export async function resolveOptimalRegion(): Promise<string> {
    const cached = readCache();
    if (cached) {
        return cached;
    }
    try {
        const info = await getRegionInfo();
        if (info && typeof info.region === "string" && info.region !== "") {
            writeCache(info.region);
            return info.region;
        }
    } catch (err: unknown) {
        console.warn("hyperbeam getRegionInfo failed; falling back to server default", err);
    }
    return "";
}

function readCache(): string | null {
    if (typeof window === "undefined") {
        return null;
    }
    try {
        const raw = window.localStorage.getItem(STORAGE_KEY);
        if (!raw) {
            return null;
        }
        const parsed = JSON.parse(raw) as CachedRegion;
        if (!parsed.region || typeof parsed.storedAt !== "number") {
            return null;
        }
        if (Date.now() - parsed.storedAt > STORAGE_TTL_MS) {
            return null;
        }
        return parsed.region;
    } catch {
        return null;
    }
}

function writeCache(region: string): void {
    if (typeof window === "undefined") {
        return;
    }
    try {
        const payload: CachedRegion = { region, storedAt: Date.now() };
        window.localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
    } catch {
        // localStorage unavailable, ignore
    }
}
