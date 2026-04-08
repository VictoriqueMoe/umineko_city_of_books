import { apiDelete, apiDeleteWithBody, apiFetch, apiPost, apiPostFormData, apiPut, buildQueryString } from "./client";
import type {
    ActivityListResponse,
    AdminStats,
    AdminUserDetail,
    AdminUserListResponse,
    Announcement,
    AnnouncementListResponse,
    ArtDetail,
    ArtListResponse,
    AuditLogListResponse,
    ChangePasswordPayload,
    CharacterListResponse,
    ChatMessage,
    ChatRoom,
    CreateResponsePayload,
    CreateTheoryPayload,
    DeleteAccountPayload,
    FanficChapter,
    FanficDetail,
    FanficListResponse,
    FollowStats,
    Gallery,
    GalleryDetailResponse,
    MysteryAttachment,
    MysteryDetail,
    MysteryLeaderboardResponse,
    MysteryListResponse,
    NotificationListResponse,
    Poll,
    PostDetail,
    PostListResponse,
    PostMedia,
    QuoteBrowseResponse,
    QuoteSearchResponse,
    ShipCharacter,
    ShipDetail,
    ShipListResponse,
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
    top_detective_ids: string[];
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

export async function getMe(): Promise<UserProfile> {
    const session = await apiFetch<{ username: string }>("/auth/session");
    return getUserProfile(session.username);
}

export type Series = "umineko" | "higurashi";

export async function searchQuotes(params: {
    query?: string;
    character?: string;
    episode?: number;
    arc?: string;
    truth?: string;
    lang?: string;
    limit?: number;
    offset?: number;
    series?: Series;
}): Promise<QuoteSearchResponse> {
    const series = params.series ?? "umineko";
    const qs = buildQueryString({
        q: params.query,
        character: params.character,
        episode: params.episode,
        arc: params.arc,
        truth: params.truth,
        lang: series === "umineko" ? params.lang : undefined,
        limit: params.limit ?? 30,
        offset: params.offset,
    });
    const response = await fetch(`${QUOTE_API}/${series}/search${qs}`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    return response.json();
}

export async function browseQuotes(params: {
    character?: string;
    episode?: number;
    truth?: string;
    arc?: string;
    lang?: string;
    limit?: number;
    offset?: number;
    series?: Series;
}): Promise<QuoteBrowseResponse> {
    const series = params.series ?? "umineko";
    const qs = buildQueryString({
        character: params.character,
        episode: params.episode,
        truth: params.truth,
        arc: params.arc,
        lang: series === "umineko" ? params.lang : undefined,
        limit: params.limit ?? 30,
        offset: params.offset,
    });
    const response = await fetch(`${QUOTE_API}/${series}/browse${qs}`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    return response.json();
}

export async function getCharacters(series: Series = "umineko"): Promise<Record<string, string>> {
    const response = await fetch(`${QUOTE_API}/${series}/characters`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    const data = await response.json();
    return data.characters;
}

export async function listTheories(params: {
    sort?: string;
    episode?: number;
    author?: string;
    search?: string;
    series?: Series;
    limit?: number;
    offset?: number;
}): Promise<TheoryListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        episode: params.episode,
        author: params.author,
        search: params.search,
        series: params.series ?? "umineko",
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

export async function updateGameBoardSort(sort: string): Promise<void> {
    await apiPut<unknown, { sort: string }>("/preferences/game-board-sort", { sort });
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

export async function updateMysteryScoreAdjustment(id: string, adjustment: number): Promise<void> {
    await apiPut<unknown, { adjustment: number }>(`/admin/users/${id}/mystery-score`, { adjustment });
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

export async function resolveReport(id: number, comment: string): Promise<void> {
    await apiPost<unknown, { comment: string }>(`/admin/reports/${id}/resolve`, { comment });
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
    resolved?: string;
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

export interface CreatePollPayload {
    options: { label: string }[];
    duration_seconds: number;
}

export async function createPost(
    body: string,
    corner: string = "general",
    poll?: CreatePollPayload,
    sharedContentId?: string,
    sharedContentType?: string,
): Promise<{ id: string }> {
    return apiPost<
        { id: string },
        {
            body: string;
            corner: string;
            poll?: CreatePollPayload;
            shared_content_id?: string;
            shared_content_type?: string;
        }
    >("/posts", {
        body,
        corner,
        poll,
        shared_content_id: sharedContentId,
        shared_content_type: sharedContentType,
    });
}

export async function getShareCount(contentType: string, contentId: string): Promise<{ share_count: number }> {
    return apiFetch<{ share_count: number }>(`/share-count/${contentType}/${contentId}`);
}

export async function votePoll(postId: string, optionId: number): Promise<Poll> {
    return apiPost<Poll, { option_id: number }>(`/posts/${postId}/poll/vote`, { option_id: optionId });
}

export async function resolveSuggestion(postId: string, status: string = "done"): Promise<void> {
    await apiPost<unknown, { status: string }>(`/posts/${postId}/resolve`, { status });
}

export async function unresolveSuggestion(postId: string): Promise<void> {
    await apiDelete(`/posts/${postId}/resolve`);
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

export async function blockUser(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/users/${id}/block`, undefined);
}

export async function unblockUser(id: string): Promise<void> {
    await apiDelete(`/users/${id}/block`);
}

export interface BlockStatus {
    blocking: boolean;
    blocked_by: boolean;
}

export async function getBlockStatus(id: string): Promise<BlockStatus> {
    return apiFetch<BlockStatus>(`/users/${id}/block-status`);
}

export interface BlockedUserItem {
    id: string;
    username: string;
    display_name: string;
    avatar_url: string;
    blocked_at: string;
}

export async function getBlockedUsers(): Promise<{ users: BlockedUserItem[] }> {
    return apiFetch<{ users: BlockedUserItem[] }>("/blocked-users");
}

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

export async function listMysteries(params: {
    sort?: string;
    solved?: string;
    limit?: number;
    offset?: number;
}): Promise<MysteryListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        solved: params.solved,
        limit: params.limit ?? 20,
        offset: params.offset,
    });
    return apiFetch<MysteryListResponse>(`/mysteries${qs}`);
}

export async function getMystery(id: string): Promise<MysteryDetail> {
    return apiFetch<MysteryDetail>(`/mysteries/${id}`);
}

export async function createMystery(data: {
    title: string;
    body: string;
    difficulty: string;
    clues: { body: string; truth_type: string }[];
}): Promise<{ id: string }> {
    return apiPost<{ id: string }, typeof data>("/mysteries", data);
}

export async function updateMystery(
    id: string,
    data: {
        title: string;
        body: string;
        difficulty: string;
        clues: { body: string; truth_type: string }[];
    },
): Promise<void> {
    await apiPut<unknown, typeof data>(`/mysteries/${id}`, data);
}

export async function deleteMystery(id: string): Promise<void> {
    await apiDelete(`/mysteries/${id}`);
}

export async function createMysteryAttempt(
    mysteryId: string,
    body: string,
    parentId?: string,
): Promise<{ id: string }> {
    return apiPost<{ id: string }, { body: string; parent_id?: string }>(`/mysteries/${mysteryId}/attempts`, {
        body,
        parent_id: parentId,
    });
}

export async function deleteMysteryAttempt(id: string): Promise<void> {
    await apiDelete(`/mystery-attempts/${id}`);
}

export async function voteMysteryAttempt(id: string, value: number): Promise<void> {
    await apiPost<unknown, { value: number }>(`/mystery-attempts/${id}/vote`, { value });
}

export async function markMysterySolved(mysteryId: string, attemptId: string): Promise<void> {
    await apiPost<unknown, { attempt_id: string }>(`/mysteries/${mysteryId}/solve`, { attempt_id: attemptId });
}

export async function addMysteryClue(
    mysteryId: string,
    body: string,
    truthType: string,
    playerId?: string,
): Promise<void> {
    await apiPost<unknown, { body: string; truth_type: string; player_id?: string }>(`/mysteries/${mysteryId}/clues`, {
        body,
        truth_type: truthType,
        player_id: playerId,
    });
}

export async function createMysteryComment(
    mysteryId: string,
    body: string,
    parentId?: string,
): Promise<{ id: string }> {
    return apiPost<{ id: string }, { body: string; parent_id?: string }>(`/mysteries/${mysteryId}/comments`, {
        body,
        parent_id: parentId,
    });
}

export async function updateMysteryComment(id: string, body: string): Promise<void> {
    await apiPut<unknown, { body: string }>(`/mystery-comments/${id}`, { body });
}

export async function deleteMysteryComment(id: string): Promise<void> {
    await apiDelete(`/mystery-comments/${id}`);
}

export async function likeMysteryComment(id: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/mystery-comments/${id}/like`, {});
}

export async function unlikeMysteryComment(id: string): Promise<void> {
    await apiDelete(`/mystery-comments/${id}/like`);
}

export async function uploadMysteryCommentMedia(commentId: string, file: File): Promise<PostMedia> {
    const formData = new FormData();
    formData.append("media", file);
    return apiPostFormData<PostMedia>(`/mystery-comments/${commentId}/media`, formData);
}

export async function uploadMysteryAttachment(mysteryId: string, file: File): Promise<MysteryAttachment> {
    const formData = new FormData();
    formData.append("file", file);
    return apiPostFormData<MysteryAttachment>(`/mysteries/${mysteryId}/attachments`, formData);
}

export async function deleteMysteryAttachment(mysteryId: string, attachmentId: number): Promise<void> {
    await apiDelete(`/mysteries/${mysteryId}/attachments/${attachmentId}`);
}

export async function getMysteryLeaderboard(limit?: number): Promise<MysteryLeaderboardResponse> {
    const qs = buildQueryString({ limit });
    return apiFetch<MysteryLeaderboardResponse>(`/mysteries/leaderboard${qs}`);
}

export async function getUserShips(userId: string, limit = 20, offset = 0): Promise<ShipListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<ShipListResponse>(`/users/${userId}/ships${qs}`);
}

export async function getUserMysteries(userId: string, limit = 20, offset = 0): Promise<MysteryListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<MysteryListResponse>(`/users/${userId}/mysteries${qs}`);
}

// Fanfiction

export async function listFanfics(params: {
    sort?: string;
    series?: string;
    rating?: string;
    genre_a?: string;
    genre_b?: string;
    language?: string;
    status?: string;
    tag?: string;
    char_a?: string;
    char_b?: string;
    char_c?: string;
    char_d?: string;
    pairing?: boolean;
    lemons?: boolean;
    search?: string;
    limit?: number;
    offset?: number;
}): Promise<FanficListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        series: params.series,
        rating: params.rating,
        genre_a: params.genre_a,
        genre_b: params.genre_b,
        language: params.language,
        status: params.status,
        tag: params.tag,
        char_a: params.char_a,
        char_b: params.char_b,
        char_c: params.char_c,
        char_d: params.char_d,
        pairing: params.pairing ? "true" : undefined,
        lemons: params.lemons ? "true" : undefined,
        search: params.search,
        limit: params.limit ?? 25,
        offset: params.offset,
    });
    return apiFetch<FanficListResponse>(`/fanfics${qs}`);
}

export async function getFanfic(id: string): Promise<FanficDetail> {
    return apiFetch<FanficDetail>(`/fanfics/${id}`);
}

export async function createFanfic(data: {
    title: string;
    summary: string;
    series: string;
    rating: string;
    language: string;
    status?: string;
    is_oneshot: boolean;
    contains_lemons: boolean;
    genres: string[];
    tags: string[];
    characters: { series: string; character_id?: string; character_name: string; sort_order: number }[];
    is_pairing: boolean;
    body?: string;
}): Promise<{ id: string }> {
    return apiPost<{ id: string }, typeof data>("/fanfics", data);
}

export async function updateFanfic(
    id: string,
    data: {
        title: string;
        summary: string;
        series: string;
        rating: string;
        language: string;
        status: string;
        is_oneshot: boolean;
        contains_lemons: boolean;
        genres: string[];
        tags: string[];
        characters: { series: string; character_id?: string; character_name: string; sort_order: number }[];
        is_pairing: boolean;
    },
): Promise<void> {
    await apiPut<unknown, typeof data>(`/fanfics/${id}`, data);
}

export async function deleteFanfic(id: string): Promise<void> {
    await apiDelete(`/fanfics/${id}`);
}

export async function uploadFanficCover(fanficId: string, file: File): Promise<{ image_url: string }> {
    const formData = new FormData();
    formData.append("image", file);
    return apiPostFormData<{ image_url: string }>(`/fanfics/${fanficId}/cover`, formData);
}

export async function deleteFanficCover(fanficId: string): Promise<void> {
    await apiDelete(`/fanfics/${fanficId}/cover`);
}

export async function getFanficChapter(fanficId: string, chapterNumber: number): Promise<FanficChapter> {
    return apiFetch<FanficChapter>(`/fanfics/${fanficId}/chapters/${chapterNumber}`);
}

export async function createFanficChapter(fanficId: string, title: string, body: string): Promise<{ id: string }> {
    return apiPost<{ id: string }, { title: string; body: string }>(`/fanfics/${fanficId}/chapters`, { title, body });
}

export async function updateFanficChapter(chapterId: string, title: string, body: string): Promise<void> {
    await apiPut<unknown, { title: string; body: string }>(`/fanfic-chapters/${chapterId}`, { title, body });
}

export async function deleteFanficChapter(chapterId: string): Promise<void> {
    await apiDelete(`/fanfic-chapters/${chapterId}`);
}

export async function favouriteFanfic(id: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/fanfics/${id}/favourite`, {});
}

export async function unfavouriteFanfic(id: string): Promise<void> {
    await apiDelete(`/fanfics/${id}/favourite`);
}

export async function createFanficComment(fanficId: string, body: string, parentId?: string): Promise<{ id: string }> {
    return apiPost<{ id: string }, { body: string; parent_id?: string }>(`/fanfics/${fanficId}/comments`, {
        body,
        parent_id: parentId,
    });
}

export async function updateFanficComment(id: string, body: string): Promise<void> {
    await apiPut<unknown, { body: string }>(`/fanfic-comments/${id}`, { body });
}

export async function deleteFanficComment(id: string): Promise<void> {
    await apiDelete(`/fanfic-comments/${id}`);
}

export async function likeFanficComment(id: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/fanfic-comments/${id}/like`, {});
}

export async function unlikeFanficComment(id: string): Promise<void> {
    await apiDelete(`/fanfic-comments/${id}/like`);
}

export async function uploadFanficCommentMedia(commentId: string, file: File): Promise<PostMedia> {
    const formData = new FormData();
    formData.append("media", file);
    return apiPostFormData<PostMedia>(`/fanfic-comments/${commentId}/media`, formData);
}

export async function getFanficLanguages(): Promise<string[]> {
    const res = await apiFetch<{ languages: string[] }>("/fanfic-languages");
    return res.languages;
}

export async function getFanficSeries(): Promise<string[]> {
    const res = await apiFetch<{ series: string[] }>("/fanfic-series");
    return res.series;
}

export async function searchOCCharacters(query: string): Promise<string[]> {
    const qs = buildQueryString({ q: query });
    const res = await apiFetch<{ characters: string[] }>(`/fanfic-oc-characters${qs}`);
    return res.characters;
}

export async function getUserFanfics(userId: string, limit = 20, offset = 0): Promise<FanficListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<FanficListResponse>(`/users/${userId}/fanfics${qs}`);
}

export async function getUserFanficFavourites(userId: string, limit = 20, offset = 0): Promise<FanficListResponse> {
    const qs = buildQueryString({ limit, offset });
    return apiFetch<FanficListResponse>(`/users/${userId}/fanfic-favourites${qs}`);
}

// Announcements

export async function createAnnouncementComment(
    announcementId: string,
    body: string,
    parentId?: string,
): Promise<{ id: string }> {
    return apiPost<{ id: string }, { body: string; parent_id?: string }>(`/announcements/${announcementId}/comments`, {
        body,
        parent_id: parentId,
    });
}

export async function updateAnnouncementComment(id: string, body: string): Promise<void> {
    await apiPut<unknown, { body: string }>(`/announcement-comments/${id}`, { body });
}

export async function deleteAnnouncementComment(id: string): Promise<void> {
    await apiDelete(`/announcement-comments/${id}`);
}

export async function likeAnnouncementComment(id: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/announcement-comments/${id}/like`, {});
}

export async function unlikeAnnouncementComment(id: string): Promise<void> {
    await apiDelete(`/announcement-comments/${id}/like`);
}

export async function uploadAnnouncementCommentMedia(commentId: string, file: File): Promise<PostMedia> {
    const formData = new FormData();
    formData.append("media", file);
    return apiPostFormData<PostMedia>(`/announcement-comments/${commentId}/media`, formData);
}

export async function listShips(params: {
    sort?: string;
    series?: string;
    character?: string;
    crackships?: boolean;
    limit?: number;
    offset?: number;
}): Promise<ShipListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        series: params.series,
        character: params.character,
        crackships: params.crackships ? "true" : undefined,
        limit: params.limit,
        offset: params.offset,
    });
    return apiFetch<ShipListResponse>(`/ships${qs}`);
}

export async function getShip(id: string): Promise<ShipDetail> {
    return apiFetch<ShipDetail>(`/ships/${id}`);
}

export async function createShip(data: {
    title: string;
    description: string;
    characters: ShipCharacter[];
}): Promise<{ id: string }> {
    return apiPost<{ id: string }, typeof data>("/ships", data);
}

export async function updateShip(
    id: string,
    data: {
        title: string;
        description: string;
        characters: ShipCharacter[];
    },
): Promise<void> {
    await apiPut<unknown, typeof data>(`/ships/${id}`, data);
}

export async function deleteShip(id: string): Promise<void> {
    await apiDelete(`/ships/${id}`);
}

export async function uploadShipImage(shipId: string, file: File): Promise<{ image_url: string }> {
    const formData = new FormData();
    formData.append("image", file);
    return apiPostFormData<{ image_url: string }>(`/ships/${shipId}/image`, formData);
}

export async function voteShip(shipId: string, value: number): Promise<void> {
    await apiPost<unknown, { value: number }>(`/ships/${shipId}/vote`, { value });
}

export async function createShipComment(shipId: string, body: string, parentId?: string): Promise<{ id: string }> {
    return apiPost<{ id: string }, { body: string; parent_id?: string }>(`/ships/${shipId}/comments`, {
        body,
        parent_id: parentId,
    });
}

export async function updateShipComment(id: string, body: string): Promise<void> {
    await apiPut<unknown, { body: string }>(`/ship-comments/${id}`, { body });
}

export async function deleteShipComment(id: string): Promise<void> {
    await apiDelete(`/ship-comments/${id}`);
}

export async function likeShipComment(id: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/ship-comments/${id}/like`, {});
}

export async function unlikeShipComment(id: string): Promise<void> {
    await apiDelete(`/ship-comments/${id}/like`);
}

export async function uploadShipCommentMedia(commentId: string, file: File): Promise<PostMedia> {
    const formData = new FormData();
    formData.append("media", file);
    return apiPostFormData<PostMedia>(`/ship-comments/${commentId}/media`, formData);
}

export async function listCharacters(series: string): Promise<CharacterListResponse> {
    return apiFetch<CharacterListResponse>(`/characters/${series}`);
}
