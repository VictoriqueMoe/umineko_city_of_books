import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    addChatMessageReaction,
    banChatRoomMember,
    clearChatRoomAvatar,
    clearChatRoomMemberTimeout,
    createChatRoomBannedWord,
    createGroupRoom,
    deleteChatMessage,
    deleteChatRoom,
    deleteChatRoomBannedWord,
    editChatMessage,
    forceMuteVoiceParticipant,
    inviteChatRoomMembers,
    joinChatRoom,
    kickChatRoomMember,
    leaveChatRoom,
    markChatRoomRead,
    pinChatMessage,
    removeChatMessageReaction,
    sendChatMessage,
    sendFirstDMMessage,
    setChatRoomMemberNickname,
    setChatRoomMemberTimeout,
    setChatRoomMuted,
    unbanChatRoomMember,
    unlockChatRoomMemberNickname,
    unpinChatMessage,
    updateChatRoom,
    updateChatRoomBannedWord,
    updateChatRoomNickname,
    uploadChatRoomAvatar,
} from "../../api/endpoints/chat";
import type {
    BotsWillBeKickedResponse,
    ChatMessageListResponse,
    ChatRoom,
    CreateBannedWordRequest,
    UpdateGroupRoomRequest,
    User,
} from "../../types/api";
import { ApiError } from "../../api/client";
import { queryKeys } from "../../api/queryKeys";

const ROOM_KEY = ["chat", "rooms"] as const;
const ROOMS_LIST_KEY = ["chat", "rooms-list"] as const;

const NO_ROOM_OPEN = "No chat room is open.";

export const FORCE_MUTE_FAILED = "Could not change that microphone.";

const BANNED_WORD_CODE = "banned_word";
const KICK_ACTION = "kick";
const BOTS_KICKED_CODE = "bots_will_be_kicked";
const BOTS_KICKED_STATUS = 409;

export interface BannedWordRejection {
    pattern: string;
    kicked: boolean;
}

export interface ChatSendRejection {
    bannedWord: BannedWordRejection | null;
    serverMessage: string | null;
}

function requireRoom(roomId: string | null | undefined): string {
    if (!roomId) {
        throw new Error(NO_ROOM_OPEN);
    }

    return roomId;
}

export function readChatSendRejection(error: unknown): ChatSendRejection | null {
    if (!(error instanceof ApiError)) {
        return null;
    }

    const body = error.body as { code?: string; pattern?: string; action?: string; error?: string } | null;
    const bannedWord =
        body?.code === BANNED_WORD_CODE && body.pattern
            ? { pattern: body.pattern, kicked: body.action === KICK_ACTION }
            : null;

    return { bannedWord, serverMessage: body?.error ?? null };
}

export function readBotsWillBeKicked(error: unknown): User[] | null {
    if (!(error instanceof ApiError) || error.status !== BOTS_KICKED_STATUS) {
        return null;
    }

    const body = error.body as BotsWillBeKickedResponse | null;
    if (!body || body.code !== BOTS_KICKED_CODE) {
        return null;
    }

    return body.bots ?? [];
}

export function useCreateGroupRoom() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: {
            name: string;
            description: string;
            is_public: boolean;
            is_rp: boolean;
            tags: string[];
            member_ids: string[];
        }): Promise<ChatRoom> => createGroupRoom(payload),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ROOM_KEY });
        },
    });
}

export function useUpdateChatRoom(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: UpdateGroupRoomRequest): Promise<ChatRoom> => updateChatRoom(roomId, payload),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ROOM_KEY });
            qc.invalidateQueries({ queryKey: ROOMS_LIST_KEY });
        },
    });
}

export function useJoinChatRoom() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ roomId, ghost }: { roomId: string; ghost?: boolean }) => joinChatRoom(roomId, { ghost }),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ROOM_KEY });
        },
    });
}

export function useLeaveChatRoom() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (roomId: string) => leaveChatRoom(roomId),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ROOM_KEY });
        },
    });
}

export function useDeleteChatRoom() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (roomId: string) => deleteChatRoom(roomId),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ROOM_KEY });
        },
    });
}

export function useSetChatRoomMuted() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ roomId, muted }: { roomId: string; muted: boolean }) => setChatRoomMuted(roomId, muted),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ROOM_KEY });
        },
    });
}

export function useKickChatRoomMember(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (userId: string) => kickChatRoomMember(roomId, userId),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.chat.roomSettings(roomId) });
            qc.invalidateQueries({ queryKey: queryKeys.chat.roomMembers(roomId) });
        },
    });
}

export function useBanChatRoomMember(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ userId, reason }: { userId: string; reason: string }) =>
            banChatRoomMember(roomId, userId, reason),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.chat.roomSettings(roomId) });
            qc.invalidateQueries({ queryKey: queryKeys.chat.roomMembers(roomId) });
        },
    });
}

export function useUnbanChatRoomMember(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (userId: string) => unbanChatRoomMember(roomId, userId),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.chat.roomSettings(roomId) });
            qc.invalidateQueries({ queryKey: queryKeys.chat.roomMembers(roomId) });
        },
    });
}

export function useCreateChatRoomBannedWord(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (req: CreateBannedWordRequest) => createChatRoomBannedWord(roomId, req),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.chat.roomBannedWords(roomId) });
        },
    });
}

export function useUpdateChatRoomBannedWord(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ ruleId, req }: { ruleId: string; req: CreateBannedWordRequest }) =>
            updateChatRoomBannedWord(roomId, ruleId, req),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.chat.roomBannedWords(roomId) });
        },
    });
}

export function useDeleteChatRoomBannedWord(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (ruleId: string) => deleteChatRoomBannedWord(roomId, ruleId),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.chat.roomBannedWords(roomId) });
        },
    });
}

export function useInviteChatRoomMembers(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (userIds: string[]) => inviteChatRoomMembers(roomId, userIds),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.chat.roomSettings(roomId) });
            qc.invalidateQueries({ queryKey: queryKeys.chat.roomMembers(roomId) });
        },
    });
}

export function useSendChatMessage(roomId: string) {
    return useMutation({
        mutationFn: (payload: { body: string; reply_to_id?: string; files?: File[] }) =>
            sendChatMessage(roomId, payload),
    });
}

export function useSendFirstDMMessage() {
    return useMutation({
        mutationFn: ({ recipientId, body, files }: { recipientId: string; body: string; files?: File[] }) =>
            sendFirstDMMessage(recipientId, body, files),
    });
}

export function useMarkChatRoomRead() {
    return useMutation({
        mutationFn: (roomId: string) => markChatRoomRead(roomId),
    });
}

export function useUpdateChatRoomNickname(roomId: string) {
    return useMutation({
        mutationFn: (nickname: string) => updateChatRoomNickname(roomId, nickname),
    });
}

export function useSetChatRoomMemberNickname(roomId: string) {
    return useMutation({
        mutationFn: ({ userId, nickname }: { userId: string; nickname: string }) =>
            setChatRoomMemberNickname(roomId, userId, nickname),
    });
}

export function useUnlockChatRoomMemberNickname(roomId: string) {
    return useMutation({
        mutationFn: (userId: string) => unlockChatRoomMemberNickname(roomId, userId),
    });
}

export function useSetChatRoomMemberTimeout(roomId: string) {
    return useMutation({
        mutationFn: ({ userId, amount, unit }: { userId: string; amount: number; unit: string }) =>
            setChatRoomMemberTimeout(roomId, userId, amount, unit),
    });
}

export function useClearChatRoomMemberTimeout(roomId: string) {
    return useMutation({
        mutationFn: (userId: string) => clearChatRoomMemberTimeout(roomId, userId),
    });
}

export function useUploadChatRoomAvatar(roomId: string) {
    return useMutation({
        mutationFn: (file: File) => uploadChatRoomAvatar(roomId, file),
    });
}

export function useClearChatRoomAvatar(roomId: string) {
    return useMutation({
        mutationFn: () => clearChatRoomAvatar(roomId),
    });
}

export function useDeleteChatMessage() {
    return useMutation({
        mutationFn: (messageId: string) => deleteChatMessage(messageId),
    });
}

export function useEditChatMessage() {
    return useMutation({
        mutationFn: ({ messageId, body }: { messageId: string; body: string }) => editChatMessage(messageId, body),
    });
}

export function usePinChatMessage(roomId?: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (messageId: string) => pinChatMessage(messageId),
        onSuccess: () => {
            if (roomId) {
                qc.invalidateQueries({ queryKey: queryKeys.chat.pinned(roomId) });
            }
        },
    });
}

export function useUnpinChatMessage(roomId?: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (messageId: string) => unpinChatMessage(messageId),
        onSuccess: (_result, messageId) => {
            if (!roomId) {
                return;
            }

            qc.setQueryData<ChatMessageListResponse>(queryKeys.chat.pinned(roomId), prev =>
                prev ? { ...prev, messages: prev.messages.filter(m => m.id !== messageId) } : prev,
            );
            qc.invalidateQueries({ queryKey: queryKeys.chat.pinned(roomId) });
        },
    });
}

export function useForceMuteVoiceParticipant(roomId: string | null | undefined) {
    return useMutation({
        mutationFn: ({ userId, muted }: { userId: string; muted: boolean }) =>
            forceMuteVoiceParticipant(requireRoom(roomId), userId, muted),
    });
}

export function useAddChatMessageReaction() {
    return useMutation({
        mutationFn: ({ messageId, emoji }: { messageId: string; emoji: string }) =>
            addChatMessageReaction(messageId, emoji),
    });
}

export function useRemoveChatMessageReaction() {
    return useMutation({
        mutationFn: ({ messageId, emoji }: { messageId: string; emoji: string }) =>
            removeChatMessageReaction(messageId, emoji),
    });
}
