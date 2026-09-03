import { apiDelete, apiFetch, apiPatch, apiPost, apiPostFormData, apiPut, buildQueryString } from "../client";
import type {
    BannedWordRule,
    ChatMessage,
    ChatMessageListResponse,
    ChatRoom,
    ChatRoomBan,
    ChatRoomMember,
    CreateBannedWordRequest,
    UpdateGroupRoomRequest,
    User,
    VoiceTokenResponse,
} from "../../types/api";

export async function resolveDMRoom(recipientId: string): Promise<{ room: ChatRoom | null; recipient: User }> {
    return apiFetch<{ room: ChatRoom | null; recipient: User }>(`/chat/dm/${encodeURIComponent(recipientId)}/resolve`);
}

export async function sendFirstDMMessage(
    recipientId: string,
    body: string,
    files?: File[],
): Promise<{ room: ChatRoom; message: ChatMessage }> {
    const formData = new FormData();
    formData.append("body", body);
    if (files) {
        for (let i = 0; i < files.length; i++) {
            formData.append("media", files[i]);
        }
    }
    return apiPostFormData<{ room: ChatRoom; message: ChatMessage }>(`/chat/dm/${recipientId}/messages`, formData);
}

export async function createGroupRoom(payload: {
    name: string;
    description: string;
    is_public: boolean;
    is_rp: boolean;
    tags: string[];
    member_ids: string[];
}): Promise<ChatRoom> {
    return apiPost<ChatRoom, typeof payload>("/chat/rooms", payload);
}

export async function updateChatRoom(roomId: string, payload: UpdateGroupRoomRequest): Promise<ChatRoom> {
    return apiPut<ChatRoom, UpdateGroupRoomRequest>(`/chat/rooms/${roomId}`, payload);
}

export async function listPublicChatRooms(params: {
    search?: string;
    rp?: boolean;
    tag?: string;
    includeArchived?: boolean;
    limit?: number;
    offset?: number;
}): Promise<{ rooms: ChatRoom[]; total: number }> {
    const qs = buildQueryString({
        search: params.search,
        rp: params.rp ? "true" : undefined,
        tag: params.tag,
        include_archived: params.includeArchived ? "true" : undefined,
        limit: params.limit ?? 20,
        offset: params.offset,
    });
    return apiFetch<{ rooms: ChatRoom[]; total: number }>(`/chat/rooms/public${qs}`);
}

export async function listMyChatRooms(params: {
    role?: "host" | "member";
    search?: string;
    rp?: boolean;
    tag?: string;
    includeArchived?: boolean;
    limit?: number;
    offset?: number;
}): Promise<{ rooms: ChatRoom[]; total: number }> {
    const qs = buildQueryString({
        role: params.role,
        search: params.search,
        rp: params.rp ? "true" : undefined,
        tag: params.tag,
        include_archived: params.includeArchived ? "true" : undefined,
        limit: params.limit ?? 20,
        offset: params.offset,
    });
    return apiFetch<{ rooms: ChatRoom[]; total: number }>(`/chat/rooms/mine${qs}`);
}

export async function joinChatRoom(roomId: string, opts: { ghost?: boolean } = {}): Promise<ChatRoom> {
    return apiPost<ChatRoom, { ghost?: boolean }>(`/chat/rooms/${roomId}/join`, { ghost: opts.ghost });
}

export async function leaveChatRoom(roomId: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/chat/rooms/${roomId}/leave`, {});
}

export async function setChatRoomMuted(roomId: string, muted: boolean): Promise<{ muted: boolean }> {
    return apiPut<{ muted: boolean }, { muted: boolean }>(`/chat/rooms/${roomId}/mute`, { muted });
}

export async function getChatRoomMembers(roomId: string): Promise<{ members: ChatRoomMember[] }> {
    return apiFetch<{ members: ChatRoomMember[] }>(`/chat/rooms/${roomId}/members`);
}

export async function kickChatRoomMember(roomId: string, userId: string): Promise<void> {
    await apiDelete<unknown>(`/chat/rooms/${roomId}/members/${userId}`);
}

export async function banChatRoomMember(roomId: string, userId: string, reason: string): Promise<void> {
    await apiPost<unknown, { reason: string }>(`/chat/rooms/${roomId}/bans/${userId}`, { reason });
}

export async function unbanChatRoomMember(roomId: string, userId: string): Promise<void> {
    await apiDelete<unknown>(`/chat/rooms/${roomId}/bans/${userId}`);
}

export async function listChatRoomBans(roomId: string): Promise<{ bans: ChatRoomBan[] }> {
    return apiFetch<{ bans: ChatRoomBan[] }>(`/chat/rooms/${roomId}/bans`);
}

export async function listChatRoomBannedWords(roomId: string): Promise<{ rules: BannedWordRule[] }> {
    return apiFetch<{ rules: BannedWordRule[] }>(`/chat/rooms/${roomId}/banned-words`);
}

export async function createChatRoomBannedWord(roomId: string, req: CreateBannedWordRequest): Promise<BannedWordRule> {
    return apiPost<BannedWordRule, CreateBannedWordRequest>(`/chat/rooms/${roomId}/banned-words`, req);
}

export async function updateChatRoomBannedWord(
    roomId: string,
    ruleId: string,
    req: CreateBannedWordRequest,
): Promise<BannedWordRule> {
    return apiPut<BannedWordRule, CreateBannedWordRequest>(`/chat/rooms/${roomId}/banned-words/${ruleId}`, req);
}

export async function deleteChatRoomBannedWord(roomId: string, ruleId: string): Promise<void> {
    await apiDelete<unknown>(`/chat/rooms/${roomId}/banned-words/${ruleId}`);
}

export async function inviteChatRoomMembers(
    roomId: string,
    userIds: string[],
): Promise<{ invited_count: number; skipped_count: number }> {
    return apiPost<{ invited_count: number; skipped_count: number }, { user_ids: string[] }>(
        `/chat/rooms/${roomId}/members`,
        { user_ids: userIds },
    );
}

export async function getVoiceToken(roomId: string): Promise<VoiceTokenResponse> {
    return apiPost<VoiceTokenResponse, Record<string, never>>(`/chat/rooms/${roomId}/voice/token`, {});
}

export async function forceMuteVoiceParticipant(roomId: string, userId: string, muted: boolean): Promise<void> {
    await apiPost<unknown, { muted: boolean }>(`/chat/rooms/${roomId}/voice/participants/${userId}/mute`, { muted });
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

export async function getRoomMessagesBefore(
    roomId: string,
    before: string,
    limit?: number,
): Promise<{ messages: ChatMessage[] }> {
    const qs = buildQueryString({ before, limit: limit ?? 50 });
    return apiFetch<{ messages: ChatMessage[] }>(`/chat/rooms/${roomId}/messages${qs}`);
}

export async function sendChatMessage(
    roomId: string,
    payload: { body: string; reply_to_id?: string; files?: File[] },
): Promise<ChatMessage> {
    const formData = new FormData();
    formData.append("body", payload.body);
    if (payload.reply_to_id) {
        formData.append("reply_to_id", payload.reply_to_id);
    }
    if (payload.files) {
        for (let i = 0; i < payload.files.length; i++) {
            formData.append("media", payload.files[i]);
        }
    }
    return apiPostFormData<ChatMessage>(`/chat/rooms/${roomId}/messages`, formData);
}

export async function deleteChatRoom(roomId: string): Promise<void> {
    await apiDelete<unknown>(`/chat/rooms/${roomId}`);
}

export async function getChatUnreadCount(): Promise<{ count: number }> {
    return apiFetch<{ count: number }>("/chat/unread-count");
}

export async function markChatRoomRead(roomId: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/chat/rooms/${roomId}/read`, {});
}

export async function updateChatRoomNickname(roomId: string, nickname: string): Promise<ChatRoomMember> {
    return apiPut<ChatRoomMember, { nickname: string }>(`/chat/rooms/${roomId}/me`, { nickname });
}

export async function setChatRoomMemberNickname(
    roomId: string,
    userId: string,
    nickname: string,
): Promise<ChatRoomMember> {
    return apiPut<ChatRoomMember, { nickname: string }>(`/chat/rooms/${roomId}/members/${userId}/nickname`, {
        nickname,
    });
}

export async function unlockChatRoomMemberNickname(roomId: string, userId: string): Promise<ChatRoomMember> {
    return apiDelete<ChatRoomMember>(`/chat/rooms/${roomId}/members/${userId}/nickname`);
}

export async function setChatRoomMemberTimeout(
    roomId: string,
    userId: string,
    amount: number,
    unit: string,
): Promise<ChatRoomMember> {
    return apiPut<ChatRoomMember, { amount: number; unit: string }>(`/chat/rooms/${roomId}/members/${userId}/timeout`, {
        amount,
        unit,
    });
}

export async function clearChatRoomMemberTimeout(roomId: string, userId: string): Promise<ChatRoomMember> {
    return apiDelete<ChatRoomMember>(`/chat/rooms/${roomId}/members/${userId}/timeout`);
}

export async function uploadChatRoomAvatar(roomId: string, file: File): Promise<ChatRoomMember> {
    const formData = new FormData();
    formData.append("avatar", file);
    return apiPostFormData<ChatRoomMember>(`/chat/rooms/${roomId}/me/avatar`, formData);
}

export async function clearChatRoomAvatar(roomId: string): Promise<ChatRoomMember> {
    return apiDelete<ChatRoomMember>(`/chat/rooms/${roomId}/me/avatar`);
}

export async function deleteChatMessage(messageId: string): Promise<void> {
    await apiDelete<unknown>(`/chat/messages/${messageId}`);
}

export async function editChatMessage(messageId: string, body: string): Promise<ChatMessage> {
    return apiPatch<ChatMessage, { body: string }>(`/chat/messages/${messageId}`, { body });
}

export async function pinChatMessage(messageId: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/chat/messages/${messageId}/pin`, {});
}

export async function unpinChatMessage(messageId: string): Promise<void> {
    await apiDelete<unknown>(`/chat/messages/${messageId}/pin`);
}

export async function getChatRoomPinnedMessages(roomId: string): Promise<ChatMessageListResponse> {
    return apiFetch<ChatMessageListResponse>(`/chat/rooms/${roomId}/pins`);
}

export type AttachmentKind = "media" | "links";

export async function getChatRoomAttachments(
    roomId: string,
    kind: AttachmentKind,
    before?: string,
    limit?: number,
): Promise<ChatMessageListResponse> {
    const qs = buildQueryString({ kind, before, limit: limit ?? 50 });

    return apiFetch<ChatMessageListResponse>(`/chat/rooms/${roomId}/attachments${qs}`);
}

export async function addChatMessageReaction(messageId: string, emoji: string): Promise<void> {
    await apiPost<unknown, { emoji: string }>(`/chat/messages/${messageId}/reactions`, { emoji });
}

export async function removeChatMessageReaction(messageId: string, emoji: string): Promise<void> {
    await apiDelete<unknown>(`/chat/messages/${messageId}/reactions/${encodeURIComponent(emoji)}`);
}
