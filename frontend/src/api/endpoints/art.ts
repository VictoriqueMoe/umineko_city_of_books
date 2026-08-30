import { apiDelete, apiFetch, apiPost, apiPostFormData, apiPut, buildQueryString } from "../client";
import type { ArtDetail, ArtListResponse, Gallery, GalleryDetailResponse, TagCount } from "../../types/api";

export async function listArt(params: {
    corner?: string;
    type?: string;
    search?: string;
    tag?: string;
    sort?: string;
    limit?: number;
    offset?: number;
}): Promise<ArtListResponse> {
    const qs = buildQueryString(params);
    return apiFetch<ArtListResponse>(`/art${qs}`);
}

export async function getArt(id: string): Promise<ArtDetail> {
    return apiFetch<ArtDetail>(`/art/${id}`);
}

export async function createArt(
    metadata: {
        title: string;
        description: string;
        corner: string;
        art_type: string;
        tags: string[];
        is_spoiler: boolean;
        gallery_id?: string;
    },
    imageFile: File,
): Promise<{ id: string }> {
    const formData = new FormData();
    formData.append("metadata", JSON.stringify(metadata));
    formData.append("image", imageFile);
    return apiPostFormData<{ id: string }>("/art", formData);
}

export async function updateArt(
    id: string,
    data: { title: string; description: string; tags: string[]; is_spoiler: boolean },
): Promise<void> {
    await apiPut<unknown, typeof data>(`/art/${id}`, data);
}

export async function deleteArt(id: string): Promise<void> {
    await apiDelete(`/art/${id}`);
}

export async function likeArt(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/art/${id}/like`, undefined);
}

export async function unlikeArt(id: string): Promise<void> {
    await apiDelete(`/art/${id}/like`);
}

export async function getArtCornerCounts(): Promise<Record<string, number>> {
    return apiFetch<Record<string, number>>("/art/corner-counts");
}

export async function getPopularTags(corner?: string): Promise<TagCount[]> {
    const qs = corner ? `?corner=${encodeURIComponent(corner)}` : "";
    return apiFetch<TagCount[]>(`/art/tags${qs}`);
}

export async function createGallery(name: string, description: string = ""): Promise<{ id: string }> {
    return apiPost<{ id: string }, { name: string; description: string }>("/galleries", { name, description });
}

export async function updateGallery(id: string, name: string, description: string = ""): Promise<void> {
    await apiPut<unknown, { name: string; description: string }>(`/galleries/${id}`, { name, description });
}

export async function setGalleryCover(galleryId: string, coverArtId: string | null): Promise<void> {
    await apiPut<unknown, { cover_art_id: string | null }>(`/galleries/${galleryId}/cover`, {
        cover_art_id: coverArtId,
    });
}

export async function deleteGallery(id: string): Promise<void> {
    await apiDelete(`/galleries/${id}`);
}

export async function getGallery(id: string, limit: number = 24, offset: number = 0): Promise<GalleryDetailResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<GalleryDetailResponse>(`/galleries/${id}${qs}`);
}

export async function listAllGalleries(corner?: string): Promise<Gallery[]> {
    const qs = corner ? `?corner=${encodeURIComponent(corner)}` : "";
    return apiFetch<Gallery[]>(`/galleries${qs}`);
}

export async function setArtGallery(artId: string, galleryId: string | null): Promise<void> {
    await apiPut<unknown, { gallery_id: string | null }>(`/art/${artId}/gallery`, {
        gallery_id: galleryId,
    });
}
