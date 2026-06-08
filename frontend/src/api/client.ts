import { getAuthToken, setAuthToken, isNativeApp, clientPlatform } from "../utils/authToken";

const API_ORIGIN = import.meta.env.VITE_API_BASE ?? "";
const API_PREFIX = "/api/v1";

export function apiUrl(path: string): string {
    return `${API_ORIGIN}${path}`;
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
            if (
                typeof child === "string" &&
                child.startsWith("/") &&
                !child.startsWith("//") &&
                key.toLowerCase().endsWith("url")
            ) {
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

function endpoint(path: string): string {
    return apiUrl(`${API_PREFIX}${path}`);
}

export function authHeaders(): Record<string, string> {
    if (!isNativeApp()) {
        return {};
    }

    const headers: Record<string, string> = { "X-Client-Platform": clientPlatform() };
    const token = getAuthToken();
    if (token) {
        headers["Authorization"] = `Bearer ${token}`;
    }

    return headers;
}

export class ApiError extends Error {
    status: number;
    body: unknown;
    constructor(status: number, message: string, body: unknown) {
        super(message);
        this.status = status;
        this.body = body;
    }
}

function captureSessionToken(response: Response): void {
    const token = response.headers.get("X-Session-Token");
    if (token) {
        setAuthToken(token);
    }
}

async function handleResponse<T>(response: Response): Promise<T> {
    captureSessionToken(response);

    if (!response.ok) {
        const body = await response.json().catch(() => null);
        const message = (body as { error?: string } | null)?.error ?? `API error: ${response.status}`;
        throw new ApiError(response.status, message, body);
    }
    if (response.status === 204 || response.headers.get("content-length") === "0") {
        return undefined as T;
    }
    return absolutizeMedia<T>(await response.json());
}

export async function apiFetch<T>(path: string): Promise<T> {
    const response = await fetch(endpoint(path), {
        credentials: "include",
        headers: authHeaders(),
    });
    return handleResponse<T>(response);
}

export async function apiPost<T, B>(path: string, body: B): Promise<T> {
    const response = await fetch(endpoint(path), {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify(body),
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiPut<T, B>(path: string, body: B): Promise<T> {
    const response = await fetch(endpoint(path), {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify(body),
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiPatch<T, B>(path: string, body: B): Promise<T> {
    const response = await fetch(endpoint(path), {
        method: "PATCH",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify(body),
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiDelete<T>(path: string): Promise<T> {
    const response = await fetch(endpoint(path), {
        method: "DELETE",
        credentials: "include",
        headers: authHeaders(),
    });
    return handleResponse<T>(response);
}

export async function apiDeleteWithBody<T, B>(path: string, body: B): Promise<T> {
    const response = await fetch(endpoint(path), {
        method: "DELETE",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify(body),
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiPostFormData<T>(path: string, formData: FormData): Promise<T> {
    const response = await fetch(endpoint(path), {
        method: "POST",
        body: formData,
        credentials: "include",
        headers: authHeaders(),
    });
    return handleResponse<T>(response);
}

export function buildQueryString(params: Record<string, string | number | undefined>): string {
    const search = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
        if (value !== undefined && value !== "" && value !== 0) {
            search.set(key, String(value));
        }
    }
    const qs = search.toString();
    return qs ? `?${qs}` : "";
}
