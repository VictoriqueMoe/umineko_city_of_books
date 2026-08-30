import { apiDeleteWithBody, apiFetch, apiPost, apiPostFormData, apiPut } from "../client";
import { clearAuthToken } from "../authToken";
import type {
    ChangePasswordPayload,
    DeleteAccountPayload,
    SessionResponse,
    UpdateProfilePayload,
    User,
} from "../../types/api";

export async function register(
    username: string,
    email: string,
    password: string,
    displayName: string,
    inviteCode?: string,
    turnstileToken?: string,
): Promise<User> {
    return apiPost<
        User,
        {
            username: string;
            email: string;
            password: string;
            display_name: string;
            invite_code?: string;
            turnstile_token?: string;
        }
    >("/auth/register", {
        username,
        email,
        password,
        display_name: displayName,
        invite_code: inviteCode,
        turnstile_token: turnstileToken,
    });
}

export async function setEmail(email: string, password: string): Promise<void> {
    await apiPost<unknown, { email: string; password: string }>("/auth/set-email", { email, password });
}

export async function verifyEmail(token: string): Promise<void> {
    await apiPost<unknown, { token: string }>("/auth/verify-email", { token });
}

export async function resendVerification(): Promise<void> {
    await apiPost<unknown, undefined>("/auth/resend-verification", undefined);
}

export async function login(username: string, password: string, turnstileToken?: string): Promise<User> {
    return apiPost<User, { username: string; password: string; turnstile_token?: string }>("/auth/login", {
        username,
        password,
        turnstile_token: turnstileToken,
    });
}

export async function forgotPassword(username: string, turnstileToken?: string): Promise<void> {
    await apiPost<unknown, { username: string; turnstile_token?: string }>("/auth/forgot-password", {
        username,
        turnstile_token: turnstileToken,
    });
}

export async function resetPassword(token: string, newPassword: string): Promise<void> {
    await apiPost<unknown, { token: string; new_password: string }>("/auth/reset-password", {
        token,
        new_password: newPassword,
    });
}

export async function logout(): Promise<void> {
    await apiPost<unknown, undefined>("/auth/logout", undefined);
    clearAuthToken();
}

export async function registerDeviceToken(token: string, platform: string): Promise<void> {
    await apiPost<unknown, { token: string; platform: string }>("/push/device", { token, platform });
}

export async function unregisterDeviceToken(token: string): Promise<void> {
    await apiDeleteWithBody<unknown, { token: string }>("/push/device", { token });
}

export async function getSession(): Promise<SessionResponse> {
    return apiFetch<SessionResponse>("/auth/session");
}

export async function updateProfile(payload: UpdateProfilePayload): Promise<{ status: string }> {
    return apiPut<{ status: string }, UpdateProfilePayload>("/auth/profile", payload);
}

export async function updateGameBoardSort(sort: string): Promise<void> {
    await apiPut<unknown, { sort: string }>("/preferences/game-board-sort", { sort });
}

export async function updateAppearance(theme: string, font: string, wideLayout: boolean): Promise<void> {
    await apiPut<unknown, { theme: string; font: string; wide_layout: boolean }>("/preferences/appearance", {
        theme,
        font,
        wide_layout: wideLayout,
    });
}

export async function updateChatbotOptIn(optedIn: boolean): Promise<void> {
    await apiPut<unknown, { opted_in: boolean }>("/preferences/chatbot-opt-in", { opted_in: optedIn });
}

export async function uploadAvatar(file: File): Promise<{ avatar_url: string }> {
    const formData = new FormData();
    formData.append("avatar", file);
    return apiPostFormData<{ avatar_url: string }>("/auth/avatar", formData);
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
