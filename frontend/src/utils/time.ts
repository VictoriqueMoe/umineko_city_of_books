export function parseServerDate(input: string | null | undefined): Date | null {
    if (!input) {
        return null;
    }
    const trimmed = input.trim();
    if (trimmed === "") {
        return null;
    }
    let s = trimmed.replace(" ", "T");
    const hasTZ = /(Z|[+-]\d{2}:?\d{2})$/.test(s);
    if (!hasTZ) {
        s = `${s}Z`;
    }
    const d = new Date(s);
    if (Number.isNaN(d.getTime())) {
        return null;
    }
    return d;
}

function diffSeconds(dateStr: string | null | undefined): number | null {
    const d = parseServerDate(dateStr);
    if (!d) {
        return null;
    }
    return Math.floor((Date.now() - d.getTime()) / 1000);
}

function relativeLadder(dateStr: string | null | undefined, suffix: string): string {
    const diff = diffSeconds(dateStr);
    if (diff === null) {
        return "";
    }
    if (diff < 60) {
        return "just now";
    }

    const mins = Math.floor(diff / 60);
    if (mins < 60) {
        return `${mins}m${suffix}`;
    }

    const hours = Math.floor(mins / 60);
    if (hours < 24) {
        return `${hours}h${suffix}`;
    }

    const days = Math.floor(hours / 24);
    if (days < 30) {
        return `${days}d${suffix}`;
    }

    const months = Math.floor(days / 30);
    return `${months}mo${suffix}`;
}

export function relativeTime(dateStr: string | null | undefined): string {
    return relativeLadder(dateStr, " ago");
}

export function shortRelativeTime(dateStr: string | null | undefined): string {
    return relativeLadder(dateStr, "");
}

const TIME_FORMAT = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit" });
const SHORT_DATE_FORMAT = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short" });
const SHORT_DATE_YEAR_FORMAT = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric" });

function isSameLocalDay(a: Date, b: Date): boolean {
    return a.getDate() === b.getDate() && a.getMonth() === b.getMonth() && a.getFullYear() === b.getFullYear();
}

export function formatMessageTime(dateStr: string | null | undefined): string {
    const d = parseServerDate(dateStr);
    if (!d) {
        return "";
    }

    const time = TIME_FORMAT.format(d);
    const now = new Date();
    if (isSameLocalDay(d, now)) {
        return time;
    }

    const yesterday = new Date(now);
    yesterday.setDate(now.getDate() - 1);
    if (isSameLocalDay(d, yesterday)) {
        return `Yesterday ${time}`;
    }

    const sameYear = d.getFullYear() === now.getFullYear();
    const datePart = sameYear ? SHORT_DATE_FORMAT.format(d) : SHORT_DATE_YEAR_FORMAT.format(d);
    return `${datePart} ${time}`;
}

export function formatActiveLabel(dateStr: string | null | undefined): string {
    const d = parseServerDate(dateStr);
    if (!d) {
        return "No activity yet";
    }
    const mins = Math.floor((Date.now() - d.getTime()) / 60000);
    if (mins < 1) {
        return "Active just now";
    }
    if (mins < 60) {
        return `Active ${mins}m ago`;
    }
    const hours = Math.floor(mins / 60);
    if (hours < 24) {
        return `Active ${hours}h ago`;
    }
    const days = Math.floor(hours / 24);
    if (days < 30) {
        return `Active ${days}d ago`;
    }
    return `Active ${d.toLocaleDateString()}`;
}

const EXACT_DATE_TIME_OPTIONS: Intl.DateTimeFormatOptions = { dateStyle: "long", timeStyle: "short" };
const EXACT_DATE_TIME_FORMAT = new Intl.DateTimeFormat(undefined, EXACT_DATE_TIME_OPTIONS);

export function formatExactDateTime(dateStr: string | null | undefined, locale?: string | string[]): string {
    const d = parseServerDate(dateStr);
    if (!d) {
        return "";
    }

    if (locale === undefined) {
        return EXACT_DATE_TIME_FORMAT.format(d);
    }

    return new Intl.DateTimeFormat(locale, EXACT_DATE_TIME_OPTIONS).format(d);
}

export function formatFullDateTime(dateStr: string | null | undefined, locale?: string | string[]): string {
    const d = parseServerDate(dateStr);
    if (!d) {
        return "";
    }
    return d.toLocaleString(locale);
}

export function formatDate(dateStr: string | null | undefined, locale?: string | string[]): string {
    const d = parseServerDate(dateStr);
    if (!d) {
        return "";
    }
    return d.toLocaleDateString(locale);
}

export function formatTimeOfDay(dateStr: string | null | undefined, locale?: string | string[]): string {
    const d = parseServerDate(dateStr);
    if (!d) {
        return "";
    }
    return d.toLocaleTimeString(locale, { hour: "2-digit", minute: "2-digit" });
}

const SHORT_DATE_TIME_FORMAT = new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
});

export function formatShortDateTime(dateStr: string | null | undefined): string {
    const d = parseServerDate(dateStr);
    if (!d) {
        return "";
    }
    return SHORT_DATE_TIME_FORMAT.format(d);
}
