import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    acceptDraw,
    acceptGameInvite,
    cancelGameInvite,
    declineDraw,
    declineGameInvite,
    inviteToGame,
    offerDraw,
    postPlayerChat,
    postSpectatorChat,
    resignGame,
    submitGameAction,
} from "../../api/endpoints/gameRoom";
import type { GameType, SpectatorChatResponse, SpectatorMessage } from "../../types/api";
import { queryKeys } from "../../api/queryKeys";

const detail = (id: string) => queryKeys.gameRoom.detail(id);

const LIST_KEY = ["gameRoom", "list"] as const;

function withMessage(prev: SpectatorChatResponse | undefined, message: SpectatorMessage): SpectatorChatResponse {
    if (!prev) {
        return { messages: [message] };
    }

    const messages = prev.messages ?? [];
    if (messages.some(m => m.id === message.id)) {
        return prev;
    }

    return { messages: [...messages, message] };
}

export function useInviteToGame() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ opponentId, gameType }: { opponentId: string; gameType: GameType }) =>
            inviteToGame(opponentId, gameType),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.gameRoom.all }),
    });
}

export function useAcceptGameInvite() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => acceptGameInvite(id),
        onSuccess: (room, id) => {
            qc.setQueryData(detail(id), room);
            qc.invalidateQueries({ queryKey: LIST_KEY });
        },
    });
}

export function useDeclineGameInvite() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => declineGameInvite(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.gameRoom.all }),
    });
}

export function useCancelGameInvite() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => cancelGameInvite(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.gameRoom.all }),
    });
}

export function useSubmitGameAction(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (action: Record<string, unknown>) => submitGameAction(id, action),
        onSuccess: room => {
            qc.setQueryData(detail(id), room);
            qc.invalidateQueries({ queryKey: LIST_KEY });
        },
    });
}

export function useResignGame() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => resignGame(id),
        onSuccess: (room, id) => {
            qc.setQueryData(detail(id), room);
            qc.invalidateQueries({ queryKey: LIST_KEY });
        },
    });
}

export function useOfferDraw() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => offerDraw(id),
        onSuccess: (room, id) => {
            qc.setQueryData(detail(id), room);
            qc.invalidateQueries({ queryKey: LIST_KEY });
        },
    });
}

export function useAcceptDraw() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => acceptDraw(id),
        onSuccess: (room, id) => {
            qc.setQueryData(detail(id), room);
            qc.invalidateQueries({ queryKey: LIST_KEY });
        },
    });
}

export function useDeclineDraw() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => declineDraw(id),
        onSuccess: (room, id) => {
            qc.setQueryData(detail(id), room);
            qc.invalidateQueries({ queryKey: LIST_KEY });
        },
    });
}

export function usePostSpectatorChat(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (body: string) => postSpectatorChat(roomId, body),
        onSuccess: message => {
            qc.setQueryData<SpectatorChatResponse>(queryKeys.gameRoom.spectatorChat(roomId), prev =>
                withMessage(prev, message),
            );
        },
    });
}

export function usePostPlayerChat(roomId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (body: string) => postPlayerChat(roomId, body),
        onSuccess: message => {
            qc.setQueryData<SpectatorChatResponse>(queryKeys.gameRoom.playerChat(roomId), prev =>
                withMessage(prev, message),
            );
        },
    });
}
