const IMAGE_EXTENSIONS = new Set(["jpg", "jpeg", "png", "gif", "webp", "avif"]);
const VIDEO_EXTENSIONS = new Set(["mp4", "webm", "mov", "m4v"]);

export type WaifuvaultMediaKind = "image" | "video";

export function detectWaifuvaultMedia(rawURL: string): WaifuvaultMediaKind | null {
    let parsed: URL;
    try {
        parsed = new URL(rawURL);
    } catch {
        return null;
    }
    if (parsed.hostname !== "waifuvault.moe" && !parsed.hostname.endsWith(".waifuvault.moe")) {
        return null;
    }
    const dot = parsed.pathname.lastIndexOf(".");
    if (dot === -1) {
        return null;
    }
    const ext = parsed.pathname.slice(dot + 1).toLowerCase();
    if (IMAGE_EXTENSIONS.has(ext)) {
        return "image";
    }
    if (VIDEO_EXTENSIONS.has(ext)) {
        return "video";
    }
    return null;
}
