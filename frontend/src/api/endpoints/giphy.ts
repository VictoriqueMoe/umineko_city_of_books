import { apiDelete, apiFetch, apiPost, buildQueryString } from "../client";
import type { GiphyFavourite, GiphyFavouritesResponse, GiphyResponse } from "../../types/api";

export async function searchGiphy(q: string, offset?: number, limit?: number): Promise<GiphyResponse> {
    const qs = buildQueryString({ q, offset, limit });
    return apiFetch<GiphyResponse>(`/giphy/search${qs}`);
}

export async function trendingGiphy(offset?: number, limit?: number): Promise<GiphyResponse> {
    const qs = buildQueryString({ offset, limit });
    return apiFetch<GiphyResponse>(`/giphy/trending${qs}`);
}

export async function listGiphyFavourites(offset?: number, limit?: number): Promise<GiphyFavouritesResponse> {
    const qs = buildQueryString({ offset, limit });
    return apiFetch<GiphyFavouritesResponse>(`/giphy/favourites${qs}`);
}

export async function addGiphyFavourite(fav: GiphyFavourite): Promise<void> {
    await apiPost<unknown, GiphyFavourite>(`/giphy/favourites`, fav);
}

export async function removeGiphyFavourite(giphyId: string): Promise<void> {
    await apiDelete(`/giphy/favourites/${encodeURIComponent(giphyId)}`);
}
