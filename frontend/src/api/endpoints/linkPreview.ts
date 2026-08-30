import { apiFetch, buildQueryString } from "../client";
import type { LinkPreview } from "../../types/api";

export async function getLinkPreview(url: string): Promise<LinkPreview> {
    const qs = buildQueryString({ url });
    return apiFetch<LinkPreview>(`/link-preview${qs}`);
}
