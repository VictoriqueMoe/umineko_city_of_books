import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    changePassword,
    deleteAccount,
    login,
    logout,
    register,
    unlockSecret,
    updateAppearance,
    updateGameBoardSort,
    updateProfile,
    uploadAvatar,
    uploadBanner,
} from "../endpoints";
import type { ChangePasswordPayload, DeleteAccountPayload, UpdateProfilePayload } from "../../types/api";

export function useRegister() {
    return useMutation({
        mutationFn: ({
            username,
            password,
            displayName,
            inviteCode,
            turnstileToken,
        }: {
            username: string;
            password: string;
            displayName: string;
            inviteCode?: string;
            turnstileToken?: string;
        }) => register(username, password, displayName, inviteCode, turnstileToken),
    });
}

export function useLogin() {
    return useMutation({
        mutationFn: ({
            username,
            password,
            turnstileToken,
        }: {
            username: string;
            password: string;
            turnstileToken?: string;
        }) => login(username, password, turnstileToken),
    });
}

export function useLogout() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: () => logout(),
        onSuccess: () => {
            qc.clear();
        },
    });
}

export function useUpdateProfile() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: UpdateProfilePayload) => updateProfile(payload),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["auth", "me"] });
            void qc.invalidateQueries({ queryKey: ["profile"] });
        },
    });
}

export function useChangePassword() {
    return useMutation({
        mutationFn: (payload: ChangePasswordPayload) => changePassword(payload),
    });
}

export function useDeleteAccount() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: DeleteAccountPayload) => deleteAccount(payload),
        onSuccess: () => {
            qc.clear();
        },
    });
}

export function useUploadAvatar() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (file: File) => uploadAvatar(file),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["auth", "me"] });
        },
    });
}

export function useUploadBanner() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (file: File) => uploadBanner(file),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["auth", "me"] });
        },
    });
}

export function useUpdateGameBoardSort() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (sort: string) => updateGameBoardSort(sort),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["auth", "me"] });
        },
    });
}

export function useUpdateAppearance() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ theme, font, wideLayout }: { theme: string; font: string; wideLayout: boolean }) =>
            updateAppearance(theme, font, wideLayout),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["auth", "me"] });
        },
    });
}

export function useUnlockSecret() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ secret, phrase }: { secret: string; phrase: string }) => unlockSecret(secret, phrase),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["auth", "me"] });
            void qc.invalidateQueries({ queryKey: ["site-info"] });
        },
    });
}
