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

export function relativeTime(dateStr: string | null | undefined): string {
    const diff = diffSeconds(dateStr);
    if (diff === null) {
        return "";
    }
    if (diff < 60) {
        return "just now";
    }
    const mins = Math.floor(diff / 60);
    if (mins < 60) {
        return `${mins}m ago`;
    }
    const hours = Math.floor(mins / 60);
    if (hours < 24) {
        return `${hours}h ago`;
    }
    const days = Math.floor(hours / 24);
    if (days < 30) {
        return `${days}d ago`;
    }
    const months = Math.floor(days / 30);
    return `${months}mo ago`;
}

export function shortRelativeTime(dateStr: string | null | undefined): string {
    const diff = diffSeconds(dateStr);
    if (diff === null) {
        return "";
    }
    if (diff < 60) {
        return "just now";
    }
    const mins = Math.floor(diff / 60);
    if (mins < 60) {
        return `${mins}m`;
    }
    const hours = Math.floor(mins / 60);
    if (hours < 24) {
        return `${hours}h`;
    }
    const days = Math.floor(hours / 24);
    if (days < 30) {
        return `${days}d`;
    }
    const months = Math.floor(days / 30);
    return `${months}mo`;
}

export function formatMessageTime(dateStr: string | null | undefined): string {
    const d = parseServerDate(dateStr);
    if (!d) {
        return "";
    }
    const time = d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    const now = new Date();
    if (d.toDateString() === now.toDateString()) {
        return time;
    }
    const yesterday = new Date(now);
    yesterday.setDate(now.getDate() - 1);
    if (d.toDateString() === yesterday.toDateString()) {
        return `Yesterday ${time}`;
    }
    const sameYear = d.getFullYear() === now.getFullYear();
    const datePart = d.toLocaleDateString([], {
        day: "numeric",
        month: "short",
        year: sameYear ? undefined : "numeric",
    });
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
