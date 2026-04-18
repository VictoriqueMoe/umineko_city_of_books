const API_BASE = "/api/v1";

export class ApiError extends Error {
    status: number;
    body: unknown;
    constructor(status: number, message: string, body: unknown) {
        super(message);
        this.status = status;
        this.body = body;
    }
}

async function handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
        const body = await response.json().catch(() => null);
        const message = (body as { error?: string } | null)?.error ?? `API error: ${response.status}`;
        throw new ApiError(response.status, message, body);
    }
    if (response.status === 204 || response.headers.get("content-length") === "0") {
        return undefined as T;
    }
    return response.json();
}

export async function apiFetch<T>(path: string): Promise<T> {
    const response = await fetch(`${API_BASE}${path}`, {
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiPost<T, B>(path: string, body: B): Promise<T> {
    const response = await fetch(`${API_BASE}${path}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiPut<T, B>(path: string, body: B): Promise<T> {
    const response = await fetch(`${API_BASE}${path}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiPatch<T, B>(path: string, body: B): Promise<T> {
    const response = await fetch(`${API_BASE}${path}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiDelete<T>(path: string): Promise<T> {
    const response = await fetch(`${API_BASE}${path}`, {
        method: "DELETE",
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiDeleteWithBody<T, B>(path: string, body: B): Promise<T> {
    const response = await fetch(`${API_BASE}${path}`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
        credentials: "include",
    });
    return handleResponse<T>(response);
}

export async function apiPostFormData<T>(path: string, formData: FormData): Promise<T> {
    const response = await fetch(`${API_BASE}${path}`, {
        method: "POST",
        body: formData,
        credentials: "include",
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
