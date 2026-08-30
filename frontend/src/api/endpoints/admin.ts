import { apiDelete, apiDeleteWithBody, apiFetch, apiPost, apiPostFormData, apiPut, buildQueryString } from "../client";
import type {
    AdminIPMatches,
    AdminStats,
    AdminUserDetail,
    AdminUserListResponse,
    AuditLogListResponse,
    BannedGiphyEntry,
    BannedWordRule,
    CreateBannedWordRequest,
    InviteItem,
    InviteListResponse,
    PermissionSettingsResponse,
    SiteSettings,
    UsernameAvailability,
    VanityRoleDefinition,
    VanityRoleUsersResponse,
} from "../../types/api";

export async function getAdminStats(): Promise<AdminStats> {
    return apiFetch<AdminStats>("/admin/stats");
}

export async function getAdminUsers(params: {
    search?: string;
    limit?: number;
    offset?: number;
}): Promise<AdminUserListResponse> {
    const qs = buildQueryString({ search: params.search, limit: params.limit ?? 20, offset: params.offset });
    return apiFetch<AdminUserListResponse>(`/admin/users${qs}`);
}

export async function getAdminUser(id: string): Promise<AdminUserDetail> {
    return apiFetch<AdminUserDetail>(`/admin/users/${id}`);
}

export async function setUserRole(id: string, role: string): Promise<void> {
    await apiPost<unknown, { role: string }>(`/admin/users/${id}/role`, { role });
}

export async function updateDetectiveScore(id: string, desiredScore: number): Promise<void> {
    await apiPut<unknown, { desired_score: number }>(`/admin/users/${id}/mystery-score`, {
        desired_score: desiredScore,
    });
}

export async function updateGMScore(id: string, desiredScore: number): Promise<void> {
    await apiPut<unknown, { desired_score: number }>(`/admin/users/${id}/gm-score`, { desired_score: desiredScore });
}

export async function removeUserRole(id: string, role: string): Promise<void> {
    await apiDeleteWithBody<unknown, { role: string }>(`/admin/users/${id}/role`, { role });
}

export async function banUser(id: string, reason: string): Promise<void> {
    await apiPost<unknown, { reason: string }>(`/admin/users/${id}/ban`, { reason });
}

export async function unbanUser(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/admin/users/${id}/unban`, undefined);
}

export async function lockUser(id: string, reason: string): Promise<void> {
    await apiPost<unknown, { reason: string }>(`/admin/users/${id}/lock`, { reason });
}

export async function unlockUser(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/admin/users/${id}/unlock`, undefined);
}

export async function approveUser(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/admin/users/${id}/approve`, undefined);
}

export async function unapproveUser(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/admin/users/${id}/unapprove`, undefined);
}

export async function adminDeleteUser(id: string): Promise<void> {
    await apiDelete<unknown>(`/admin/users/${id}`);
}

export async function resetUserPassword(id: string): Promise<{ password: string }> {
    return apiPost<{ password: string }, undefined>(`/admin/users/${id}/reset-password`, undefined);
}

export async function setUserEmail(id: string, email: string): Promise<void> {
    await apiPut<unknown, { email: string }>(`/admin/users/${id}/email`, { email });
}

export async function verifyUserEmail(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/admin/users/${id}/verify-email`, undefined);
}

export async function unverifyUserEmail(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/admin/users/${id}/unverify-email`, undefined);
}

export async function setUserDisplayName(id: string, displayName: string): Promise<void> {
    await apiPut<unknown, { display_name: string }>(`/admin/users/${id}/display-name`, { display_name: displayName });
}

export async function setDisplayNameLock(id: string, locked: boolean): Promise<void> {
    await apiPut<unknown, { locked: boolean }>(`/admin/users/${id}/display-name-lock`, { locked });
}

export async function forceLogoutUser(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/admin/users/${id}/force-logout`, undefined);
}

export async function getUserIPMatches(id: string): Promise<AdminIPMatches> {
    return apiFetch<AdminIPMatches>(`/admin/users/${id}/ip-matches`);
}

export async function getUserAuditLog(id: string, limit = 20, offset = 0): Promise<AuditLogListResponse> {
    return apiFetch<AuditLogListResponse>(`/admin/users/${id}/audit-log${buildQueryString({ limit, offset })}`);
}

export async function getAdminSettings(): Promise<SiteSettings> {
    return apiFetch<{ settings: SiteSettings }>("/admin/settings").then(r => r.settings);
}

export async function updateAdminSettings(settings: SiteSettings): Promise<void> {
    await apiPut<unknown, { settings: SiteSettings }>("/admin/settings", { settings });
}

export async function uploadOGDefaultImage(file: File): Promise<{ image_url: string }> {
    const formData = new FormData();
    formData.append("image", file);
    return apiPostFormData<{ image_url: string }>("/admin/settings/og-image", formData);
}

export async function sendTestEmail(): Promise<void> {
    await apiPost<unknown, undefined>("/admin/settings/test-email", undefined);
}

export async function getAuditLog(params: {
    action?: string;
    limit?: number;
    offset?: number;
}): Promise<AuditLogListResponse> {
    const qs = buildQueryString({ action: params.action, limit: params.limit ?? 50, offset: params.offset });
    return apiFetch<AuditLogListResponse>(`/admin/audit-log${qs}`);
}

export async function createInvite(): Promise<InviteItem> {
    return apiPost<InviteItem, undefined>("/admin/invites", undefined);
}

export async function getInvites(params: { limit?: number; offset?: number }): Promise<InviteListResponse> {
    const qs = buildQueryString({ limit: params.limit ?? 50, offset: params.offset });
    return apiFetch<InviteListResponse>(`/admin/invites${qs}`);
}

export async function deleteInvite(code: string): Promise<void> {
    await apiDelete<unknown>(`/admin/invites/${encodeURIComponent(code)}`);
}

export async function listGlobalBannedWords(): Promise<{ rules: BannedWordRule[] }> {
    return apiFetch<{ rules: BannedWordRule[] }>("/admin/banned-words");
}

export async function createGlobalBannedWord(req: CreateBannedWordRequest): Promise<BannedWordRule> {
    return apiPost<BannedWordRule, CreateBannedWordRequest>("/admin/banned-words", req);
}

export async function updateGlobalBannedWord(ruleId: string, req: CreateBannedWordRequest): Promise<BannedWordRule> {
    return apiPut<BannedWordRule, CreateBannedWordRequest>(`/admin/banned-words/${ruleId}`, req);
}

export async function deleteGlobalBannedWord(ruleId: string): Promise<void> {
    await apiDelete<unknown>(`/admin/banned-words/${ruleId}`);
}

export async function createAnnouncement(title: string, body: string): Promise<{ id: string }> {
    return apiPost<{ id: string }, { title: string; body: string }>("/admin/announcements", { title, body });
}

export async function updateAnnouncement(id: string, title: string, body: string): Promise<void> {
    await apiPut<unknown, { title: string; body: string }>(`/admin/announcements/${id}`, { title, body });
}

export async function deleteAnnouncement(id: string): Promise<void> {
    await apiDelete(`/admin/announcements/${id}`);
}

export async function pinAnnouncement(id: string, pinned: boolean): Promise<void> {
    await apiPost<unknown, { pinned: boolean }>(`/admin/announcements/${id}/pin`, { pinned });
}

export async function getVanityRoles(): Promise<VanityRoleDefinition[]> {
    return apiFetch<VanityRoleDefinition[]>("/admin/vanity-roles");
}

export async function getBannedGifs(): Promise<{ entries: BannedGiphyEntry[] }> {
    return apiFetch<{ entries: BannedGiphyEntry[] }>("/admin/banned-gifs");
}

export async function addBannedGif(data: { input: string; reason?: string }): Promise<{ entry: BannedGiphyEntry }> {
    return apiPost<{ entry: BannedGiphyEntry }, typeof data>("/admin/banned-gifs", data);
}

export async function removeBannedGif(kind: string, value: string): Promise<void> {
    await apiDelete(`/admin/banned-gifs/${encodeURIComponent(kind)}/${encodeURIComponent(value)}`);
}

export async function createVanityRole(data: {
    label: string;
    color: string;
    sort_order: number;
}): Promise<VanityRoleDefinition> {
    return apiPost<VanityRoleDefinition, typeof data>("/admin/vanity-roles", data);
}

export async function updateVanityRole(
    id: string,
    data: { label: string; color: string; sort_order: number },
): Promise<void> {
    await apiPut<unknown, typeof data>(`/admin/vanity-roles/${id}`, data);
}

export async function deleteVanityRole(id: string): Promise<void> {
    await apiDelete(`/admin/vanity-roles/${id}`);
}

export async function getVanityRoleUsers(
    id: string,
    params: { search?: string; limit?: number; offset?: number },
): Promise<VanityRoleUsersResponse> {
    const parts: string[] = [];
    if (params.search) {
        parts.push(`search=${encodeURIComponent(params.search)}`);
    }
    parts.push(`limit=${params.limit ?? 20}`);
    if (params.offset) {
        parts.push(`offset=${params.offset}`);
    }
    return apiFetch<VanityRoleUsersResponse>(`/admin/vanity-roles/${id}/users?${parts.join("&")}`);
}

export async function assignVanityRole(roleId: string, userId: string): Promise<void> {
    await apiPost<unknown, { user_id: string }>(`/admin/vanity-roles/${roleId}/users`, { user_id: userId });
}

export async function unassignVanityRole(roleId: string, userId: string): Promise<void> {
    await apiDelete(`/admin/vanity-roles/${roleId}/users/${userId}`);
}

export async function getAdminPermissions(): Promise<PermissionSettingsResponse> {
    return apiFetch<PermissionSettingsResponse>("/admin/permissions");
}

export async function updateRolePermissions(role: string, permissions: string[]): Promise<void> {
    await apiPut<unknown, { permissions: string[] }>(`/admin/permissions/roles/${encodeURIComponent(role)}`, {
        permissions,
    });
}

export async function updateVanityRolePermissions(id: string, permissions: string[]): Promise<void> {
    await apiPut<unknown, { permissions: string[] }>(`/admin/permissions/vanity-roles/${encodeURIComponent(id)}`, {
        permissions,
    });
}

export async function checkUsernameAvailable(username: string): Promise<UsernameAvailability> {
    return apiFetch<UsernameAvailability>(`/admin/username-available${buildQueryString({ username })}`);
}
