export const UPTIME_TICK_MS = 1000;

export function formatElapsed(ms: number): string {
    const total = Math.max(0, Math.floor(ms / 1000));
    const hours = Math.floor(total / 3600);
    const minutes = Math.floor((total % 3600) / 60);
    const seconds = total % 60;

    const mm = minutes.toString().padStart(2, "0");
    const ss = seconds.toString().padStart(2, "0");

    if (hours > 0) {
        return `${hours}:${mm}:${ss}`;
    }

    return `${mm}:${ss}`;
}

export function streamUptimeLabel(startedAt: string | undefined, now: number): string | null {
    if (!startedAt) {
        return null;
    }

    const start = Date.parse(startedAt);
    if (Number.isNaN(start)) {
        return null;
    }

    return formatElapsed(now - start);
}
