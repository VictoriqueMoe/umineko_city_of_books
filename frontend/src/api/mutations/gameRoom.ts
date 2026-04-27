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
} from "../endpoints";
import type { GameType } from "../../types/api";
import { queryKeys } from "../queryKeys";

const detail = (id: string) => queryKeys.gameRoom.detail(id);

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
        onSuccess: (_d, id) => qc.invalidateQueries({ queryKey: detail(id) }),
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
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useResignGame() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => resignGame(id),
        onSuccess: (_d, id) => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useOfferDraw() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => offerDraw(id),
        onSuccess: (_d, id) => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useAcceptDraw() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => acceptDraw(id),
        onSuccess: (_d, id) => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useDeclineDraw() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => declineDraw(id),
        onSuccess: (_d, id) => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function usePostSpectatorChat(roomId: string) {
    return useMutation({
        mutationFn: (body: string) => postSpectatorChat(roomId, body),
    });
}

export function usePostPlayerChat(roomId: string) {
    return useMutation({
        mutationFn: (body: string) => postPlayerChat(roomId, body),
    });
}
