import { apiFetch } from "../client";
import type { SiteInfo, User } from "../../types/api";

export async function getSiteInfo(): Promise<SiteInfo> {
    return apiFetch<SiteInfo>("/site-info");
}

export async function getStaff(): Promise<User[]> {
    return apiFetch<User[]>("/staff");
}

export async function getRules(page: string): Promise<{ page: string; rules: string }> {
    return apiFetch<{ page: string; rules: string }>(`/rules/${encodeURIComponent(page)}`);
}
