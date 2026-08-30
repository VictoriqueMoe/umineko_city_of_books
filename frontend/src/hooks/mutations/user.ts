import { useMutation, useQueryClient } from "@tanstack/react-query";
import { blockUser, followUser, unblockUser, unfollowUser } from "../../api/endpoints/user";
import { queryKeys } from "../../api/queryKeys";
import { useAuth } from "../useAuth";

export function useFollowUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => followUser(id),
        onSuccess: (_d, id) => {
            qc.invalidateQueries({ queryKey: queryKeys.followStats(id) });
            qc.invalidateQueries({ queryKey: queryKeys.users.byId(id) });
        },
    });
}

export function useUnfollowUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unfollowUser(id),
        onSuccess: (_d, id) => {
            qc.invalidateQueries({ queryKey: queryKeys.followStats(id) });
            qc.invalidateQueries({ queryKey: queryKeys.users.byId(id) });
        },
    });
}

export function useBlockUser() {
    const qc = useQueryClient();
    const { user } = useAuth();
    return useMutation({
        mutationFn: (id: string) => blockUser(id),
        onSuccess: (_d, id) => {
            qc.invalidateQueries({ queryKey: queryKeys.blockStatus(id) });
            qc.invalidateQueries({ queryKey: queryKeys.profile.blockedUsers(user?.id ?? "") });
        },
    });
}

export function useUnblockUser() {
    const qc = useQueryClient();
    const { user } = useAuth();
    return useMutation({
        mutationFn: (id: string) => unblockUser(id),
        onSuccess: (_d, id) => {
            qc.invalidateQueries({ queryKey: queryKeys.blockStatus(id) });
            qc.invalidateQueries({ queryKey: queryKeys.profile.blockedUsers(user?.id ?? "") });
        },
    });
}
