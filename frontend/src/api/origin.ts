const API_ORIGIN = import.meta.env.VITE_API_BASE ?? "";

export function apiUrl(path: string): string {
    return `${API_ORIGIN}${path}`;
}

function isMediaUrlKey(key: string): boolean {
    const lower = key.toLowerCase();
    return lower.endsWith("url") && lower !== "url";
}

function absolutizeValue(value: unknown): unknown {
    if (Array.isArray(value)) {
        const out: unknown[] = [];
        for (let i = 0; i < value.length; i++) {
            out.push(absolutizeValue(value[i]));
        }
        return out;
    }

    if (value !== null && typeof value === "object") {
        const obj = value as Record<string, unknown>;
        const out: Record<string, unknown> = {};
        for (const key of Object.keys(obj)) {
            const child = obj[key];
            if (typeof child === "string" && child.startsWith("/") && !child.startsWith("//") && isMediaUrlKey(key)) {
                out[key] = `${API_ORIGIN}${child}`;
            } else {
                out[key] = absolutizeValue(child);
            }
        }
        return out;
    }

    return value;
}

export function absolutizeMedia<T>(data: T): T {
    if (!API_ORIGIN) {
        return data;
    }

    return absolutizeValue(data) as T;
}
