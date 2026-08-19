import { detectWaifuvaultMedia } from "../WaifuvaultEmbed/detect";

const URL_RE = /https?:\/\/[^\s<>"]+/g;
const MAX_PREVIEWS = 5;

export function previewableURLs(body: string, limit = MAX_PREVIEWS): string[] {
    if (limit <= 0) {
        return [];
    }

    const matches = body.match(URL_RE) ?? [];
    const seen = new Set<string>();
    const out: string[] = [];

    for (const url of matches) {
        if (out.length >= limit) {
            break;
        }

        if (seen.has(url) || detectWaifuvaultMedia(url)) {
            continue;
        }

        seen.add(url);
        out.push(url);
    }

    return out;
}
