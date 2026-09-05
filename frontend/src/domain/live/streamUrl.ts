export const RESERVED_PATH_SEGMENTS = new Set([
    "admin",
    "announcements",
    "api",
    "chat",
    "debug",
    "fanfiction",
    "forgot-password",
    "game-board",
    "gallery",
    "games",
    "health",
    "hls",
    "journals",
    "live",
    "livez",
    "login",
    "metrics",
    "mysteries",
    "mystery",
    "notifications",
    "oc",
    "og-image",
    "quotes",
    "reset-password",
    "robots",
    "rooms",
    "rules",
    "search",
    "secrets",
    "set-email",
    "settings",
    "ships",
    "sitemap",
    "suggestions",
    "theories",
    "theory",
    "uploads",
    "user",
    "users",
    "verify-email",
    "watch",
    "welcome",
]);

const STREAM_WATCH_PATH = /^\/([^/]+)\/live$/;

const STREAM_CHAT_POPOUT_PATH = /^\/([^/]+)\/live\/chat$/;

export function isReservedPathSegment(segment: string): boolean {
    return RESERVED_PATH_SEGMENTS.has(segment.toLowerCase());
}

export function streamWatchPath(username: string): string {
    return `/${username}/live`;
}

export function streamChatPopoutPath(username: string): string {
    return `/${username}/live/chat`;
}

export function isStreamWatchPath(pathname: string): boolean {
    const match = STREAM_WATCH_PATH.exec(pathname);

    return !!match && !isReservedPathSegment(match[1]);
}

export function isStreamChatPopoutPath(pathname: string): boolean {
    const match = STREAM_CHAT_POPOUT_PATH.exec(pathname);

    return !!match && !isReservedPathSegment(match[1]);
}
