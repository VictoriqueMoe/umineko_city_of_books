import { Capacitor } from "@capacitor/core";
import { getAuthToken, setAuthToken } from "./authToken";
import { markSessionLost } from "./sessionLost";
import { absolutizeMedia, apiUrl } from "./origin";

const API_PREFIX = "/api/v1";

export type UploadProgress = (sent: number, total: number) => void;

const BODYLESS_STATUSES = new Set([204, 205, 304]);

const ZERO_MEANS_UNSET_KEYS = new Set(["offset", "page"]);

function endpoint(path: string): string {
    return apiUrl(`${API_PREFIX}${path}`);
}

export function authHeaders(): Record<string, string> {
    if (!Capacitor.isNativePlatform()) {
        return {};
    }

    const headers: Record<string, string> = { "X-Client-Platform": Capacitor.getPlatform() };
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

async function failIfNotOk(response: Response): Promise<void> {
    if (response.ok) {
        return;
    }

    if (response.status === 401) {
        markSessionLost();
    }

    const body = await response.json().catch(() => null);
    const message = (body as { error?: string } | null)?.error ?? `API error: ${response.status}`;
    throw new ApiError(response.status, message, body);
}

async function handleResponse<T>(response: Response): Promise<T> {
    captureSessionToken(response);

    await failIfNotOk(response);

    if (response.status === 204 || response.headers.get("content-length") === "0") {
        return undefined as T;
    }
    return absolutizeMedia<T>(await response.json());
}

async function handleTextResponse(response: Response): Promise<string> {
    captureSessionToken(response);

    await failIfNotOk(response);

    return response.text();
}

export async function apiFetch<T>(path: string): Promise<T> {
    const response = await fetch(endpoint(path), {
        credentials: "include",
        headers: authHeaders(),
    });
    return handleResponse<T>(response);
}

export async function apiFetchText(path: string): Promise<string> {
    const response = await fetch(endpoint(path), {
        credentials: "include",
        headers: authHeaders(),
    });
    return handleTextResponse(response);
}

export async function apiPost<T, B = unknown>(path: string, body: B): Promise<T> {
    const response = await fetch(endpoint(path), {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify(body),
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiPut<T, B = unknown>(path: string, body: B): Promise<T> {
    const response = await fetch(endpoint(path), {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify(body),
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiPatch<T, B = unknown>(path: string, body: B): Promise<T> {
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

export async function apiDeleteWithBody<T, B = unknown>(path: string, body: B): Promise<T> {
    const response = await fetch(endpoint(path), {
        method: "DELETE",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify(body),
        credentials: "include",
    });
    return handleResponse<T>(response);
}

function parseResponseHeaders(raw: string): Headers {
    const headers = new Headers();
    for (const line of raw.trim().split(/[\r\n]+/)) {
        const separator = line.indexOf(":");
        if (separator <= 0) {
            continue;
        }

        headers.append(line.slice(0, separator).trim(), line.slice(separator + 1).trim());
    }

    return headers;
}

function sendWithProgress(path: string, formData: FormData, onProgress: UploadProgress): Promise<Response> {
    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open("POST", endpoint(path));
        xhr.withCredentials = true;
        for (const [name, value] of Object.entries(authHeaders())) {
            xhr.setRequestHeader(name, value);
        }

        xhr.upload.onprogress = event => {
            if (event.lengthComputable) {
                onProgress(event.loaded, event.total);
            }
        };
        xhr.onload = () => {
            const body = BODYLESS_STATUSES.has(xhr.status) ? null : xhr.responseText;
            resolve(
                new Response(body, { status: xhr.status, headers: parseResponseHeaders(xhr.getAllResponseHeaders()) }),
            );
        };
        xhr.onerror = () => {
            reject(new TypeError("Network request failed"));
        };
        xhr.onabort = () => {
            reject(new DOMException("The upload was cancelled", "AbortError"));
        };

        xhr.send(formData);
    });
}

export async function apiPostFormData<T>(path: string, formData: FormData, onProgress?: UploadProgress): Promise<T> {
    if (onProgress) {
        return handleResponse<T>(await sendWithProgress(path, formData, onProgress));
    }

    const response = await fetch(endpoint(path), {
        method: "POST",
        body: formData,
        credentials: "include",
        headers: authHeaders(),
    });
    return handleResponse<T>(response);
}

export async function postFile<T>(path: string, field: string, file: File): Promise<T> {
    const formData = new FormData();
    formData.append(field, file);

    return apiPostFormData<T>(path, formData);
}

export function buildQueryString(params: Record<string, string | number | undefined>): string {
    const search = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
        if (value === undefined || value === "") {
            continue;
        }
        if (value === 0 && ZERO_MEANS_UNSET_KEYS.has(key)) {
            continue;
        }

        search.set(key, String(value));
    }

    const qs = search.toString();
    return qs ? `?${qs}` : "";
}
