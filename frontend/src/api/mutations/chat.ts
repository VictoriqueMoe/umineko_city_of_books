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
    updateChatRoomBannedWord,
    updateChatRoomNickname,
    uploadChatRoomAvatar,
} from "../endpoints";
import type { ChatRoom, CreateBannedWordRequest } from "../../types/api";

const ROOM_KEY = ["chat", "rooms"] as const;

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
            void qc.invalidateQueries({ queryKey: ROOM_KEY });
        },
    });
}

export function useJoinChatRoom() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ roomId, ghost }: { roomId: string; ghost?: boolean }) => joinChatRoom(roomId, { ghost }),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ROOM_KEY });
        },
    });
}

export function useLeaveChatRoom() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (roomId: string) => leaveChatRoom(roomId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ROOM_KEY });
        },
    });
}

export function useDeleteChatRoom() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (roomId: string) => deleteChatRoom(roomId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ROOM_KEY });
        },
    });
}

export function useSetChatRoomMuted() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ roomId, muted }: { roomId: string; muted: boolean }) => setChatRoomMuted(roomId, muted),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ROOM_KEY });
        },
    });
}

export function useKickChatRoomMember(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (userId: string) => kickChatRoomMember(roomId, userId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["chat", "rooms", roomId] });
        },
    });
}

export function useBanChatRoomMember(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ userId, reason }: { userId: string; reason: string }) =>
            banChatRoomMember(roomId, userId, reason),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["chat", "rooms", roomId] });
        },
    });
}

export function useUnbanChatRoomMember(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (userId: string) => unbanChatRoomMember(roomId, userId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["chat", "rooms", roomId] });
        },
    });
}

export function useCreateChatRoomBannedWord(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (req: CreateBannedWordRequest) => createChatRoomBannedWord(roomId, req),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["chat", "rooms", roomId, "banned-words"] });
        },
    });
}

export function useUpdateChatRoomBannedWord(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ ruleId, req }: { ruleId: string; req: CreateBannedWordRequest }) =>
            updateChatRoomBannedWord(roomId, ruleId, req),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["chat", "rooms", roomId, "banned-words"] });
        },
    });
}

export function useDeleteChatRoomBannedWord(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (ruleId: string) => deleteChatRoomBannedWord(roomId, ruleId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["chat", "rooms", roomId, "banned-words"] });
        },
    });
}

export function useInviteChatRoomMembers(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (userIds: string[]) => inviteChatRoomMembers(roomId, userIds),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["chat", "rooms", roomId] });
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
                void qc.invalidateQueries({ queryKey: ["chat", "room", roomId, "pinned"] });
            }
        },
    });
}

export function useUnpinChatMessage(roomId?: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (messageId: string) => unpinChatMessage(messageId),
        onSuccess: () => {
            if (roomId) {
                void qc.invalidateQueries({ queryKey: ["chat", "room", roomId, "pinned"] });
            }
        },
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
