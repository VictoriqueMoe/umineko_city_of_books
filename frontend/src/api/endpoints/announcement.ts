import { apiFetch, buildQueryString } from "../client";
import type { Announcement, AnnouncementListResponse } from "../../types/api";

export async function listAnnouncements(limit: number = 20, offset: number = 0): Promise<AnnouncementListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<AnnouncementListResponse>(`/announcements${qs}`);
}

export async function getAnnouncement(id: string): Promise<Announcement> {
    return apiFetch<Announcement>(`/announcements/${id}`);
}

export async function getLatestAnnouncement(): Promise<{ announcement: Announcement | null }> {
    return apiFetch<{ announcement: Announcement | null }>("/announcements-latest");
}
