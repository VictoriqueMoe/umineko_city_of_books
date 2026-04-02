import { apiDelete, apiDeleteWithBody, apiFetch, apiPost, apiPostFormData, apiPut, buildQueryString } from "./client";
import type {
    ActivityListResponse,
    AdminStats,
    AdminUserDetail,
    AdminUserListResponse,
    ArtDetail,
    ArtListResponse,
    AuditLogListResponse,
    ChangePasswordPayload,
    ChatMessage,
    ChatRoom,
    CreateResponsePayload,
    CreateTheoryPayload,
    DeleteAccountPayload,
    FollowStats,
    Gallery,
    GalleryDetailResponse,
    NotificationListResponse,
    PostDetail,
    PostListResponse,
    PostMedia,
    QuoteBrowseResponse,
    QuoteSearchResponse,
    SiteSettings,
    TagCount,
    TheoryDetail,
    TheoryListResponse,
    UpdateProfilePayload,
    User,
    UserProfile,
    VotePayload,
} from "../types/api";

const QUOTE_API = "https://quotes.auaurora.moe/api/v1";

export interface SiteInfo {
    site_name: string;
    site_description: string;
    registration_type: string;
    announcement_banner: string;
    default_theme: string;
    maintenance_mode: boolean;
    maintenance_title: string;
    maintenance_message: string;
    turnstile_enabled: boolean;
    turnstile_site_key: string;
    max_image_size: number;
    max_video_size: number;
}

export async function getSiteInfo(): Promise<SiteInfo> {
    return apiFetch<SiteInfo>("/site-info");
}

export async function register(
    username: string,
    password: string,
    displayName: string,
    inviteCode?: string,
    turnstileToken?: string,
): Promise<User> {
    return apiPost<
        User,
        { username: string; password: string; display_name: string; invite_code?: string; turnstile_token?: string }
    >("/auth/register", {
        username,
        password,
        display_name: displayName,
        invite_code: inviteCode,
        turnstile_token: turnstileToken,
    });
}

export async function login(username: string, password: string, turnstileToken?: string): Promise<User> {
    return apiPost<User, { username: string; password: string; turnstile_token?: string }>("/auth/login", {
        username,
        password,
        turnstile_token: turnstileToken,
    });
}

export async function logout(): Promise<void> {
    await apiPost<unknown, undefined>("/auth/logout", undefined);
}

export async function getMe(): Promise<User> {
    return apiFetch<User>("/auth/me");
}

export async function searchQuotes(params: {
    query?: string;
    character?: string;
    episode?: number;
    truth?: string;
    limit?: number;
    offset?: number;
}): Promise<QuoteSearchResponse> {
    const qs = buildQueryString({
        q: params.query,
        character: params.character,
        episode: params.episode,
        truth: params.truth,
        limit: params.limit ?? 30,
        offset: params.offset,
    });
    const response = await fetch(`${QUOTE_API}/search${qs}`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    return response.json();
}

export async function browseQuotes(params: {
    character?: string;
    episode?: number;
    truth?: string;
    limit?: number;
    offset?: number;
}): Promise<QuoteBrowseResponse> {
    const qs = buildQueryString({
        character: params.character,
        episode: params.episode,
        truth: params.truth,
        limit: params.limit ?? 30,
        offset: params.offset,
    });
    const response = await fetch(`${QUOTE_API}/browse${qs}`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    return response.json();
}

export async function getCharacters(): Promise<Record<string, string>> {
    const response = await fetch(`${QUOTE_API}/characters`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    return response.json();
}

export async function listTheories(params: {
    sort?: string;
    episode?: number;
    author?: string;
    search?: string;
    limit?: number;
    offset?: number;
}): Promise<TheoryListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        episode: params.episode,
        author: params.author,
        search: params.search,
        limit: params.limit ?? 20,
        offset: params.offset,
    });
    return apiFetch<TheoryListResponse>(`/theories${qs}`);
}

export async function updateTheory(id: string, payload: CreateTheoryPayload): Promise<{ status: string }> {
    return apiPut<{ status: string }, CreateTheoryPayload>(`/theories/${id}`, payload);
}

export async function getTheory(id: string): Promise<TheoryDetail> {
    return apiFetch<TheoryDetail>(`/theories/${id}`);
}

export async function createTheory(payload: CreateTheoryPayload): Promise<{ id: string }> {
    return apiPost<{ id: string }, CreateTheoryPayload>("/theories", payload);
}

export async function deleteTheory(id: string): Promise<void> {
    await apiDelete<unknown>(`/theories/${id}`);
}

export async function createResponse(theoryId: string, payload: CreateResponsePayload): Promise<{ id: string }> {
    return apiPost<{ id: string }, CreateResponsePayload>(`/theories/${theoryId}/responses`, payload);
}

export async function deleteResponse(id: string): Promise<void> {
    await apiDelete<unknown>(`/responses/${id}`);
}

export async function voteTheory(id: string, value: number): Promise<void> {
    await apiPost<unknown, VotePayload>(`/theories/${id}/vote`, { value });
}

export async function voteResponse(id: string, value: number): Promise<void> {
    await apiPost<unknown, VotePayload>(`/responses/${id}/vote`, { value });
}

export async function getUserProfile(username: string): Promise<UserProfile> {
    return apiFetch<UserProfile>(`/users/${username}`);
}

export async function updateProfile(payload: UpdateProfilePayload): Promise<{ status: string }> {
    return apiPut<{ status: string }, UpdateProfilePayload>("/auth/profile", payload);
}

export async function uploadAvatar(file: File): Promise<{ avatar_url: string }> {
    const formData = new FormData();
    formData.append("avatar", file);
    return apiPostFormData<{ avatar_url: string }>("/auth/avatar", formData);
}

export async function getNotifications(params: { limit?: number; offset?: number }): Promise<NotificationListResponse> {
    const qs = buildQueryString({ limit: params.limit ?? 20, offset: params.offset });
    return apiFetch<NotificationListResponse>(`/notifications${qs}`);
}

export async function markNotificationRead(id: number): Promise<void> {
    await apiPost<unknown, undefined>(`/notifications/${id}/read`, undefined);
}

export async function markAllNotificationsRead(): Promise<void> {
    await apiPost<unknown, undefined>("/notifications/read", undefined);
}

export async function getUnreadCount(): Promise<{ count: number }> {
    return apiFetch<{ count: number }>("/notifications/unread-count");
}

export async function uploadBanner(file: File): Promise<{ banner_url: string }> {
    const formData = new FormData();
    formData.append("banner", file);
    return apiPostFormData<{ banner_url: string }>("/auth/banner", formData);
}

export async function changePassword(payload: ChangePasswordPayload): Promise<{ status: string }> {
    return apiPut<{ status: string }, ChangePasswordPayload>("/auth/password", payload);
}

export async function deleteAccount(payload: DeleteAccountPayload): Promise<{ status: string }> {
    return apiDeleteWithBody<{ status: string }, DeleteAccountPayload>("/auth/account", payload);
}

export async function getUserActivity(
    username: string,
    limit?: number,
    offset?: number,
): Promise<ActivityListResponse> {
    const qs = buildQueryString({ limit: limit ?? 20, offset });
    return apiFetch<ActivityListResponse>(`/users/${username}/activity${qs}`);
}

export async function getOnlineStatus(ids: string[]): Promise<Record<string, boolean>> {
    return apiFetch<Record<string, boolean>>(`/users/online?ids=${ids.join(",")}`);
}

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

export async function removeUserRole(id: string, role: string): Promise<void> {
    await apiDeleteWithBody<unknown, { role: string }>(`/admin/users/${id}/role`, { role });
}

export async function banUser(id: string, reason: string): Promise<void> {
    await apiPost<unknown, { reason: string }>(`/admin/users/${id}/ban`, { reason });
}

export async function unbanUser(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/admin/users/${id}/unban`, undefined);
}

export async function adminDeleteUser(id: string): Promise<void> {
    await apiDelete<unknown>(`/admin/users/${id}`);
}

export async function getAdminSettings(): Promise<SiteSettings> {
    return apiFetch<{ settings: SiteSettings }>("/admin/settings").then(r => r.settings);
}

export async function updateAdminSettings(settings: SiteSettings): Promise<void> {
    await apiPut<unknown, { settings: SiteSettings }>("/admin/settings", { settings });
}

export async function getAuditLog(params: {
    action?: string;
    limit?: number;
    offset?: number;
}): Promise<AuditLogListResponse> {
    const qs = buildQueryString({ action: params.action, limit: params.limit ?? 50, offset: params.offset });
    return apiFetch<AuditLogListResponse>(`/admin/audit-log${qs}`);
}

export interface InviteItem {
    code: string;
    created_by: string;
    used_by?: string;
    used_at?: string;
    created_at: string;
}

export interface InviteListResponse {
    invites: InviteItem[];
    total: number;
    limit: number;
    offset: number;
}

export async function createInvite(): Promise<InviteItem> {
    return apiPost<InviteItem, undefined>("/admin/invites", undefined);
}

export async function getInvites(params: { limit?: number; offset?: number }): Promise<InviteListResponse> {
    const qs = buildQueryString({ limit: params.limit ?? 50, offset: params.offset });
    return apiFetch<InviteListResponse>(`/admin/invites${qs}`);
}

export async function deleteInvite(code: string): Promise<void> {
    await apiDelete<unknown>(`/admin/invites/${code}`);
}

export async function createDMRoom(recipientId: string): Promise<ChatRoom> {
    return apiPost<ChatRoom, { recipient_id: string }>("/chat/dm", { recipient_id: recipientId });
}

export async function createGroupRoom(name: string, memberIds: string[]): Promise<ChatRoom> {
    return apiPost<ChatRoom, { name: string; member_ids: string[] }>("/chat/rooms", { name, member_ids: memberIds });
}

export async function getUserRooms(): Promise<{ rooms: ChatRoom[] }> {
    return apiFetch<{ rooms: ChatRoom[] }>("/chat/rooms");
}

export async function getRoomMessages(
    roomId: string,
    limit?: number,
    offset?: number,
): Promise<{ messages: ChatMessage[]; total: number }> {
    const qs = buildQueryString({ limit: limit ?? 50, offset });
    return apiFetch<{ messages: ChatMessage[]; total: number }>(`/chat/rooms/${roomId}/messages${qs}`);
}

export async function sendChatMessage(roomId: string, payload: { body: string }): Promise<ChatMessage> {
    return apiPost<ChatMessage, typeof payload>(`/chat/rooms/${roomId}/messages`, payload);
}

export async function deleteChatRoom(roomId: string): Promise<void> {
    await apiDelete<unknown>(`/chat/rooms/${roomId}`);
}

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

export interface ReportItem {
    id: number;
    reporter_name: string;
    reporter_avatar: string;
    target_type: string;
    target_id: string;
    context_id?: string;
    reason: string;
    status: string;
    resolved_by?: string;
    created_at: string;
}

export interface ReportListResponse {
    reports: ReportItem[];
    total: number;
    limit: number;
    offset: number;
}

export async function getReports(
    status: string = "open",
    limit: number = 50,
    offset: number = 0,
): Promise<ReportListResponse> {
    const qs = buildQueryString({ status, limit, offset });
    return apiFetch<ReportListResponse>(`/admin/reports${qs}`);
}

export async function resolveReport(id: number): Promise<void> {
    await apiPost<unknown, undefined>(`/admin/reports/${id}/resolve`, undefined);
}

export async function getRules(page: string): Promise<{ page: string; rules: string }> {
    return apiFetch<{ page: string; rules: string }>(`/rules/${page}`);
}

export async function searchUsers(query: string): Promise<User[]> {
    return apiFetch<User[]>(`/users/search?q=${encodeURIComponent(query)}`);
}

export async function getMutualFollowers(): Promise<User[]> {
    return apiFetch<User[]>("/users/mutuals");
}

export async function getCornerCounts(): Promise<Record<string, number>> {
    return apiFetch<Record<string, number>>("/posts/corner-counts");
}

export async function listPosts(params: {
    tab?: string;
    corner?: string;
    search?: string;
    sort?: string;
    seed?: number;
    limit?: number;
    offset?: number;
}): Promise<PostListResponse> {
    const qs = buildQueryString(params);
    return apiFetch<PostListResponse>(`/posts${qs}`);
}

export async function getPost(id: string): Promise<PostDetail> {
    return apiFetch<PostDetail>(`/posts/${id}`);
}

export async function updatePost(id: string, body: string): Promise<void> {
    await apiPut<unknown, { body: string }>(`/posts/${id}`, { body });
}

export async function createPost(body: string, corner: string = "general"): Promise<{ id: string }> {
    return apiPost<{ id: string }, { body: string; corner: string }>("/posts", { body, corner });
}

export async function deletePost(id: string): Promise<void> {
    await apiDelete(`/posts/${id}`);
}

export async function uploadPostMedia(postId: string, file: File): Promise<PostMedia> {
    const formData = new FormData();
    formData.append("media", file);
    return apiPostFormData<PostMedia>(`/posts/${postId}/media`, formData);
}

export async function deletePostMedia(postId: string, mediaId: number): Promise<void> {
    await apiDelete(`/posts/${postId}/media/${mediaId}`);
}

export async function likePost(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/posts/${id}/like`, undefined);
}

export async function unlikePost(id: string): Promise<void> {
    await apiDelete(`/posts/${id}/like`);
}

export async function createComment(postId: string, body: string, parentId?: string): Promise<{ id: string }> {
    return apiPost<{ id: string }, { body: string; parent_id?: string }>(`/posts/${postId}/comments`, {
        body,
        parent_id: parentId,
    });
}

export async function updateComment(id: string, body: string): Promise<void> {
    await apiPut<unknown, { body: string }>(`/comments/${id}`, { body });
}

export async function deleteComment(id: string): Promise<void> {
    await apiDelete(`/comments/${id}`);
}

export async function likeComment(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/comments/${id}/like`, undefined);
}

export async function unlikeComment(id: string): Promise<void> {
    await apiDelete(`/comments/${id}/like`);
}

export async function uploadCommentMedia(commentId: string, file: File): Promise<PostMedia> {
    const formData = new FormData();
    formData.append("media", file);
    return apiPostFormData<PostMedia>(`/comments/${commentId}/media`, formData);
}

export async function getUserPosts(userId: string, limit: number = 20, offset: number = 0): Promise<PostListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<PostListResponse>(`/users/${userId}/posts${qs}`);
}

export async function followUser(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/users/${id}/follow`, undefined);
}

export async function unfollowUser(id: string): Promise<void> {
    await apiDelete(`/users/${id}/follow`);
}

export async function getFollowStats(id: string): Promise<FollowStats> {
    return apiFetch<FollowStats>(`/users/${id}/follow-stats`);
}

export async function getFollowers(
    id: string,
    limit: number = 50,
    offset: number = 0,
): Promise<{ users: User[]; total: number }> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<{ users: User[]; total: number }>(`/users/${id}/followers${qs}`);
}

export async function getFollowing(
    id: string,
    limit: number = 50,
    offset: number = 0,
): Promise<{ users: User[]; total: number }> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<{ users: User[]; total: number }>(`/users/${id}/following${qs}`);
}

export interface PublicUser extends User {
    online: boolean;
}

export async function listUsersPublic(): Promise<PublicUser[]> {
    return apiFetch<PublicUser[]>("/users");
}

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
    data: { title: string; description: string; tags: string[] },
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

export async function createArtComment(artId: string, body: string, parentId?: string): Promise<{ id: string }> {
    return apiPost<{ id: string }, { body: string; parent_id?: string }>(`/art/${artId}/comments`, {
        body,
        parent_id: parentId,
    });
}

export async function updateArtComment(id: string, body: string): Promise<void> {
    await apiPut<unknown, { body: string }>(`/art-comments/${id}`, { body });
}

export async function deleteArtComment(id: string): Promise<void> {
    await apiDelete(`/art-comments/${id}`);
}

export async function likeArtComment(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/art-comments/${id}/like`, undefined);
}

export async function unlikeArtComment(id: string): Promise<void> {
    await apiDelete(`/art-comments/${id}/like`);
}

export async function uploadArtCommentMedia(commentId: string, file: File): Promise<PostMedia> {
    const formData = new FormData();
    formData.append("media", file);
    return apiPostFormData<PostMedia>(`/art-comments/${commentId}/media`, formData);
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

export async function getUserGalleries(userId: string): Promise<Gallery[]> {
    return apiFetch<Gallery[]>(`/users/${userId}/galleries`);
}

export async function setArtGallery(artId: string, galleryId: string | null): Promise<void> {
    await apiPut<unknown, { gallery_id: string | null }>(`/art/${artId}/gallery`, {
        gallery_id: galleryId,
    });
}

export async function getUserArt(userId: string, limit: number = 24, offset: number = 0): Promise<ArtListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<ArtListResponse>(`/users/${userId}/art${qs}`);
}
