import type { QueryClient } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { UpdateProfilePayload } from "../../types/api";
import {
    useChangePassword,
    useDeleteAccount,
    useForgotPassword,
    useLogin,
    useLogout,
    useRegister,
    useResendVerification,
    useResetPassword,
    useSetEmail,
    useUpdateAppearance,
    useUpdateGameBoardSort,
    useUpdateProfile,
    useUploadAvatar,
    useUploadBanner,
    useVerifyEmail,
} from "./auth";

const mocks = vi.hoisted(() => ({
    changePassword: vi.fn(),
    deleteAccount: vi.fn(),
    forgotPassword: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    register: vi.fn(),
    resendVerification: vi.fn(),
    resetPassword: vi.fn(),
    setEmail: vi.fn(),
    verifyEmail: vi.fn(),
    updateAppearance: vi.fn(),
    updateGameBoardSort: vi.fn(),
    updateProfile: vi.fn(),
    uploadAvatar: vi.fn(),
    uploadBanner: vi.fn(),
}));

vi.mock("../endpoints", () => mocks);

function client() {
    const qc = createTestQueryClient();

    return { qc, invalidate: vi.spyOn(qc, "invalidateQueries"), clear: vi.spyOn(qc, "clear") };
}

async function runMutation<V>(
    use: () => { mutateAsync: (variables: V) => Promise<unknown> },
    variables: V,
    qc: QueryClient,
): Promise<unknown> {
    const { result } = renderHook(use, { wrapper: providerWrapper({ queryClient: qc }) });

    let data: unknown;
    await act(async () => {
        data = await result.current.mutateAsync(variables);
    });

    return data;
}

function makeProfilePayload(overrides: Partial<UpdateProfilePayload> = {}): UpdateProfilePayload {
    return {
        display_name: "Beatrice",
        bio: "the golden witch",
        avatar_url: "",
        banner_url: "",
        banner_position: 50,
        favourite_character: "beatrice",
        gender: "",
        pronoun_subject: "she",
        pronoun_possessive: "her",
        social_twitter: "",
        social_discord: "",
        social_waifulist: "",
        social_tumblr: "",
        social_github: "",
        website: "",
        dms_enabled: true,
        episode_progress: 8,
        higurashi_arc_progress: 0,
        ciconia_chapter_progress: 0,
        dob: "",
        dob_public: false,
        email: "beato@rokkenjima.test",
        email_public: false,
        email_notifications: true,
        play_message_sound: true,
        play_notification_sound: true,
        home_page: "/",
        game_board_sort: "newest",
        default_profile_tab: "overview",
        ...overrides,
    };
}

beforeEach(() => {
    for (const fn of Object.values(mocks)) {
        fn.mockResolvedValue(undefined);
    }
});

describe("useRegister", () => {
    it("forwards every field of the sign up form in the order the endpoint expects", async () => {
        // given
        const { qc } = client();

        // when
        await runMutation(
            useRegister,
            {
                username: "beato",
                email: "beato@rokkenjima.test",
                password: "goldenland",
                displayName: "Beatrice",
                inviteCode: "invite-1",
                turnstileToken: "token-1",
            },
            qc,
        );

        // then
        expect(mocks.register).toHaveBeenCalledWith(
            "beato",
            "beato@rokkenjima.test",
            "goldenland",
            "Beatrice",
            "invite-1",
            "token-1",
        );
    });

    it("leaves the invite code and the turnstile token undefined when the site needs neither", async () => {
        // given
        const { qc, invalidate, clear } = client();

        // when
        await runMutation(
            useRegister,
            {
                username: "beato",
                email: "beato@rokkenjima.test",
                password: "goldenland",
                displayName: "Beatrice",
            },
            qc,
        );

        // then
        expect(mocks.register).toHaveBeenCalledWith(
            "beato",
            "beato@rokkenjima.test",
            "goldenland",
            "Beatrice",
            undefined,
            undefined,
        );
        expect(invalidate).not.toHaveBeenCalled();
        expect(clear).not.toHaveBeenCalled();
    });
});

describe("useLogin", () => {
    it("returns the signed in user and does not disturb the cache by itself", async () => {
        // given
        const { qc, invalidate, clear } = client();
        mocks.login.mockResolvedValue({ id: "u1", username: "beato" });

        // when
        const data = await runMutation(useLogin, { username: "beato", password: "goldenland" }, qc);

        // then
        expect(mocks.login).toHaveBeenCalledWith("beato", "goldenland", undefined);
        expect(data).toEqual({ id: "u1", username: "beato" });
        expect(invalidate).not.toHaveBeenCalled();
        expect(clear).not.toHaveBeenCalled();
    });

    it("passes the turnstile token through when the site demands one", async () => {
        // given
        const { qc } = client();

        // when
        await runMutation(useLogin, { username: "beato", password: "goldenland", turnstileToken: "token-1" }, qc);

        // then
        expect(mocks.login).toHaveBeenCalledWith("beato", "goldenland", "token-1");
    });
});

describe("useLogout", () => {
    it("throws away the whole cache once the server has ended the session", async () => {
        // given
        const { qc, clear } = client();

        // when
        await runMutation(useLogout, undefined, qc);

        // then
        expect(mocks.logout).toHaveBeenCalledWith();
        expect(clear).toHaveBeenCalledOnce();
    });

    it("keeps the cache when the logout request fails", async () => {
        // given
        const { qc, clear } = client();
        mocks.logout.mockRejectedValue(new Error("network down"));
        const { result } = renderHook(useLogout, { wrapper: providerWrapper({ queryClient: qc }) });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync()).rejects.toThrow("network down");
        });

        // then
        expect(clear).not.toHaveBeenCalled();
    });
});

describe("useUpdateProfile", () => {
    it("sends the payload untouched and refreshes both the session user and their public profile", async () => {
        // given
        const { qc, invalidate } = client();
        const payload = makeProfilePayload();
        mocks.updateProfile.mockResolvedValue({ status: "ok" });

        // when
        const data = await runMutation(useUpdateProfile, payload, qc);

        // then
        expect(mocks.updateProfile).toHaveBeenCalledWith(payload);
        expect(data).toEqual({ status: "ok" });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["auth", "me"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["profile"] });
    });

    it("leaves the cache alone when the profile save is rejected", async () => {
        // given
        const { qc, invalidate } = client();
        mocks.updateProfile.mockRejectedValue(new Error("display name is locked"));
        const { result } = renderHook(useUpdateProfile, { wrapper: providerWrapper({ queryClient: qc }) });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync(makeProfilePayload())).rejects.toThrow("display name is locked");
        });

        // then
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("account credential mutations", () => {
    it("changes the password without refreshing anything", async () => {
        // given
        const { qc, invalidate } = client();
        const payload = { old_password: "old", new_password: "new" };

        // when
        await runMutation(useChangePassword, payload, qc);

        // then
        expect(mocks.changePassword).toHaveBeenCalledWith(payload);
        expect(invalidate).not.toHaveBeenCalled();
    });

    it("splits the email payload into the address and the confirming password", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useSetEmail, { email: "beato@rokkenjima.test", password: "goldenland" }, qc);

        // then
        expect(mocks.setEmail).toHaveBeenCalledWith("beato@rokkenjima.test", "goldenland");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["auth", "me"] });
    });

    it("verifies an email with the token from the link and refreshes the session user", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useVerifyEmail, "token-1", qc);

        // then
        expect(mocks.verifyEmail).toHaveBeenCalledWith("token-1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["auth", "me"] });
    });

    it("asks for another verification email with no arguments at all", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useResendVerification, undefined, qc);

        // then
        expect(mocks.resendVerification).toHaveBeenCalledWith();
        expect(invalidate).not.toHaveBeenCalled();
    });

    it("starts a password reset for a username and forwards the turnstile token", async () => {
        // given
        const { qc } = client();

        // when
        await runMutation(useForgotPassword, { username: "beato", turnstileToken: "token-1" }, qc);

        // then
        expect(mocks.forgotPassword).toHaveBeenCalledWith("beato", "token-1");
    });

    it("starts a password reset without a turnstile token when the site has it switched off", async () => {
        // given
        const { qc } = client();

        // when
        await runMutation(useForgotPassword, { username: "beato" }, qc);

        // then
        expect(mocks.forgotPassword).toHaveBeenCalledWith("beato", undefined);
    });

    it("finishes a password reset with the token and the new password", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useResetPassword, { token: "token-1", newPassword: "goldenland" }, qc);

        // then
        expect(mocks.resetPassword).toHaveBeenCalledWith("token-1", "goldenland");
        expect(invalidate).not.toHaveBeenCalled();
    });

    it("throws away the whole cache once the account has been deleted", async () => {
        // given
        const { qc, clear } = client();
        const payload = { password: "goldenland" };

        // when
        await runMutation(useDeleteAccount, payload, qc);

        // then
        expect(mocks.deleteAccount).toHaveBeenCalledWith(payload);
        expect(clear).toHaveBeenCalledOnce();
    });
});

describe("session preference mutations", () => {
    it("uploads an avatar and refreshes only the session user", async () => {
        // given
        const { qc, invalidate } = client();
        const file = new File(["png"], "avatar.png", { type: "image/png" });
        mocks.uploadAvatar.mockResolvedValue({ avatar_url: "/uploads/avatar.png" });

        // when
        const data = await runMutation(useUploadAvatar, file, qc);

        // then
        expect(mocks.uploadAvatar).toHaveBeenCalledWith(file);
        expect(data).toEqual({ avatar_url: "/uploads/avatar.png" });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["auth", "me"] });
        expect(invalidate).not.toHaveBeenCalledWith({ queryKey: ["profile"] });
    });

    it("uploads a banner and refreshes only the session user", async () => {
        // given
        const { qc, invalidate } = client();
        const file = new File(["png"], "banner.png", { type: "image/png" });
        mocks.uploadBanner.mockResolvedValue({ banner_url: "/uploads/banner.png" });

        // when
        const data = await runMutation(useUploadBanner, file, qc);

        // then
        expect(mocks.uploadBanner).toHaveBeenCalledWith(file);
        expect(data).toEqual({ banner_url: "/uploads/banner.png" });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["auth", "me"] });
    });

    it("stores the chosen game board sort order", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUpdateGameBoardSort, "oldest", qc);

        // then
        expect(mocks.updateGameBoardSort).toHaveBeenCalledWith("oldest");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["auth", "me"] });
    });

    it("stores the theme, the font and the wide layout preference together", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUpdateAppearance, { theme: "featherine", font: "serif", wideLayout: true }, qc);

        // then
        expect(mocks.updateAppearance).toHaveBeenCalledWith("featherine", "serif", true);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["auth", "me"] });
    });
});
