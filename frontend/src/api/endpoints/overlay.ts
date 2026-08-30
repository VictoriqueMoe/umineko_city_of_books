import { apiFetch, apiFetchText, apiPost } from "../client";
import type { OverlayConnection } from "../../types/api";

export async function getOverlayConnection(): Promise<OverlayConnection> {
    return apiFetch<OverlayConnection>("/overlay/token");
}

export async function resetOverlayToken(): Promise<OverlayConnection> {
    return apiPost<OverlayConnection, Record<string, never>>("/overlay/token/reset", {});
}

export async function testOverlay(): Promise<{ ok: boolean }> {
    return apiPost<{ ok: boolean }, Record<string, never>>("/overlay/test", {});
}

export async function fetchOverlayConnectorSEF(): Promise<string> {
    return apiFetchText("/overlay/connector.sef");
}
