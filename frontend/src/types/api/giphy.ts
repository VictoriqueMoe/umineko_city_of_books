export interface GiphyImage {
    url: string;
    width: string;
    height: string;
}

export interface GiphyGif {
    id: string;
    title: string;
    url: string;
    images: Record<string, GiphyImage>;
}

export interface GiphyPagination {
    total_count: number;
    count: number;
    offset: number;
}

export interface GiphyResponse {
    data: GiphyGif[];
    pagination: GiphyPagination;
}

export interface GiphyFavourite {
    giphy_id: string;
    url: string;
    title: string;
    preview_url: string;
    width: number;
    height: number;
}

export interface GiphyFavouritesResponse {
    data: GiphyFavourite[];
    total: number;
}

export interface BannedGiphyEntry {
    kind: "gif" | "user";
    value: string;
    reason: string;
    created_at: string;
    created_by?: string;
}
