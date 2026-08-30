const CANONICAL_ORIGIN = "https://whentheycry.social";

const SITE_ORIGIN = (() => {
    const base = import.meta.env.VITE_API_BASE;
    if (base) {
        try {
            return new URL(base).origin;
        } catch {}
    }
    return CANONICAL_ORIGIN;
})();

export function isInternalOrigin(origin: string): boolean {
    return origin === window.location.origin || origin === SITE_ORIGIN;
}

export function siteUrl(path: string): string {
    return SITE_ORIGIN + path;
}
