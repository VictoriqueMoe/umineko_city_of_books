import { apiFetch, buildQueryString } from "../client";
import type { QuickSearchResponse, SearchResponse } from "../../types/api";

export async function quickSearch(q: string, perType: number = 3): Promise<QuickSearchResponse> {
    const qs = buildQueryString({ q, perType });
    return apiFetch<QuickSearchResponse>(`/search/quick${qs}`);
}

export async function searchSite(
    q: string,
    types?: string,
    limit: number = 20,
    offset: number = 0,
    room?: string,
): Promise<SearchResponse> {
    const qs = buildQueryString({ q, types: types ?? "", limit, offset, room });
    return apiFetch<SearchResponse>(`/search${qs}`);
}
