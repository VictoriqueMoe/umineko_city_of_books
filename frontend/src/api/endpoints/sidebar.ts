import { apiFetch, apiPost } from "../client";
import type {
    HomeActivityResponse,
    MarkSidebarVisitedRequest,
    SidebarActivityResponse,
    SidebarLastVisitedResponse,
} from "../../types/api";

export async function getHomeActivity(): Promise<HomeActivityResponse> {
    return apiFetch<HomeActivityResponse>("/home/activity");
}

export async function getSidebarActivity(): Promise<SidebarActivityResponse> {
    return apiFetch<SidebarActivityResponse>("/sidebar/activity");
}

export async function getSidebarLastVisited(): Promise<SidebarLastVisitedResponse> {
    return apiFetch<SidebarLastVisitedResponse>("/sidebar/last-visited");
}

export async function markSidebarVisited(key: string): Promise<void> {
    await apiPost<unknown, MarkSidebarVisitedRequest>("/sidebar/last-visited", { key });
}
