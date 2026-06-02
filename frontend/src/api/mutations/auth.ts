import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    changePassword,
    deleteAccount,
    forgotPassword,
    login,
    logout,
    register,
    resendVerification,
    resetPassword,
    setEmail,
    verifyEmail,
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
            email,
            password,
            displayName,
            inviteCode,
            turnstileToken,
        }: {
            username: string;
            email: string;
            password: string;
            displayName: string;
            inviteCode?: string;
            turnstileToken?: string;
        }) => register(username, email, password, displayName, inviteCode, turnstileToken),
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
            qc.invalidateQueries({ queryKey: ["auth", "me"] });
            qc.invalidateQueries({ queryKey: ["profile"] });
        },
    });
}

export function useChangePassword() {
    return useMutation({
        mutationFn: (payload: ChangePasswordPayload) => changePassword(payload),
    });
}

export function useSetEmail() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (email: string) => setEmail(email),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["auth", "me"] }),
    });
}

export function useVerifyEmail() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (token: string) => verifyEmail(token),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["auth", "me"] }),
    });
}

export function useResendVerification() {
    return useMutation({
        mutationFn: () => resendVerification(),
    });
}

export function useForgotPassword() {
    return useMutation({
        mutationFn: ({ username, turnstileToken }: { username: string; turnstileToken?: string }) =>
            forgotPassword(username, turnstileToken),
    });
}

export function useResetPassword() {
    return useMutation({
        mutationFn: ({ token, newPassword }: { token: string; newPassword: string }) =>
            resetPassword(token, newPassword),
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
            qc.invalidateQueries({ queryKey: ["auth", "me"] });
        },
    });
}

export function useUploadBanner() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (file: File) => uploadBanner(file),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ["auth", "me"] });
        },
    });
}

export function useUpdateGameBoardSort() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (sort: string) => updateGameBoardSort(sort),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ["auth", "me"] });
        },
    });
}

export function useUpdateAppearance() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ theme, font, wideLayout }: { theme: string; font: string; wideLayout: boolean }) =>
            updateAppearance(theme, font, wideLayout),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ["auth", "me"] });
        },
    });
}

export function useUnlockSecret() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ secret, phrase }: { secret: string; phrase: string }) => unlockSecret(secret, phrase),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ["auth", "me"] });
            qc.invalidateQueries({ queryKey: ["site-info"] });
        },
    });
}
