import { detectWaifuvaultMedia } from "../components/WaifuvaultEmbed/detect";

export const URL_SOURCE = 'https?://[^\\s<>"]+';

const URL_RE = new RegExp(URL_SOURCE, "g");
const TRAILING_PUNCTUATION = /[.,;:!?'"]+$/;
const MAX_PREVIEWS = 5;

interface PreviewableURLOptions {
    limit?: number;
    keepInlineMedia?: boolean;
}

export function trimTrailingPunctuation(url: string): string {
    let trimmed = url.replace(TRAILING_PUNCTUATION, "");

    while (trimmed.endsWith(")") && countChar(trimmed, ")") > countChar(trimmed, "(")) {
        trimmed = trimmed.slice(0, -1);
    }

    while (trimmed.endsWith("]") && countChar(trimmed, "]") > countChar(trimmed, "[")) {
        trimmed = trimmed.slice(0, -1);
    }

    return trimmed.replace(TRAILING_PUNCTUATION, "");
}

function countChar(value: string, char: string): number {
    let count = 0;
    for (const c of value) {
        if (c === char) {
            count++;
        }
    }

    return count;
}

export function previewableURLs(body: string, options: number | PreviewableURLOptions = {}): string[] {
    const { limit = MAX_PREVIEWS, keepInlineMedia = false } =
        typeof options === "number" ? { limit: options } : options;

    if (limit <= 0) {
        return [];
    }

    const matches = body.match(URL_RE) ?? [];
    const seen = new Set<string>();
    const out: string[] = [];

    for (const raw of matches) {
        if (out.length >= limit) {
            break;
        }

        const url = trimTrailingPunctuation(raw);
        if (url === "" || seen.has(url)) {
            continue;
        }

        if (!keepInlineMedia && detectWaifuvaultMedia(url)) {
            continue;
        }

        seen.add(url);
        out.push(url);
    }

    return out;
}
