import { apiFetch, apiPost, buildQueryString } from "../client";
import type { ReportListResponse } from "../../types/api";

export async function createReport(
    targetType: string,
    targetId: string,
    reason: string,
    contextId?: string,
): Promise<void> {
    await apiPost<unknown, { target_type: string; target_id: string; context_id?: string; reason: string }>("/report", {
        target_type: targetType,
        target_id: targetId,
        context_id: contextId,
        reason,
    });
}

export async function getReports(
    status: string = "open",
    limit: number = 50,
    offset: number = 0,
): Promise<ReportListResponse> {
    const qs = buildQueryString({ status, limit, offset });
    return apiFetch<ReportListResponse>(`/admin/reports${qs}`);
}

export async function resolveReport(id: number, comment: string): Promise<void> {
    await apiPost<unknown, { comment: string }>(`/admin/reports/${id}/resolve`, { comment });
}
