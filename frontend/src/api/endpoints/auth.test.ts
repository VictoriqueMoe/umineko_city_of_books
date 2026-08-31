import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./auth";
import { clearAuthToken } from "../authToken";
import {
    deleteWithBodyMock,
    postFormDataMock,
    postMock,
    putMock,
    resetTransports,
    runRequestCases,
    type RequestCase,
} from "./testHarness";

vi.mock("../authToken", () => ({
    clearAuthToken: vi.fn(),
    getAuthToken: () => null,
    setAuthToken: vi.fn(),
    loadAuthToken: vi.fn(),
}));

vi.mock("../../platform/capabilities", () => ({
    isNativeApp: () => false,
    clientPlatform: () => "web",
}));

vi.mock("../client", async importOriginal => {
    const actual = await importOriginal<typeof import("../client")>();
    return {
        ...actual,
        apiFetch: vi.fn(),
        apiFetchText: vi.fn(),
        apiPost: vi.fn(),
        apiPut: vi.fn(),
        apiPatch: vi.fn(),
        apiDelete: vi.fn(),
        apiDeleteWithBody: vi.fn(),
        apiPostFormData: vi.fn(),
    };
});

beforeEach(resetTransports);

describe("session helpers", () => {
    it("logout clears the stored auth token after the server call", async () => {
        // given
        const clearToken = vi.mocked(clearAuthToken);

        // when
        await api.logout();

        // then
        expect(postMock).toHaveBeenCalledWith("/auth/logout", undefined);
        expect(clearToken).toHaveBeenCalledOnce();
    });
});

describe("email verification and password recovery", () => {
    const cases: RequestCase[] = [
        {
            name: "setEmail posts the new address alongside the current password",
            call: () => api.setEmail("kujo@example.com", "pw"),
            transport: postMock,
            request: ["/auth/set-email", { email: "kujo@example.com", password: "pw" }],
        },
        {
            name: "verifyEmail posts the token from the email",
            call: () => api.verifyEmail("verify-1"),
            transport: postMock,
            request: ["/auth/verify-email", { token: "verify-1" }],
        },
        {
            name: "resendVerification posts with no body",
            call: () => api.resendVerification(),
            transport: postMock,
            request: ["/auth/resend-verification", undefined],
        },
        {
            name: "forgotPassword leaves the turnstile token unset when it is not supplied",
            call: () => api.forgotPassword("kujo"),
            transport: postMock,
            strict: true,
            request: ["/auth/forgot-password", { username: "kujo", turnstile_token: undefined }],
        },
        {
            name: "forgotPassword forwards the turnstile token",
            call: () => api.forgotPassword("kujo", "token-1"),
            transport: postMock,
            strict: true,
            request: ["/auth/forgot-password", { username: "kujo", turnstile_token: "token-1" }],
        },
        {
            name: "resetPassword posts the token with the snake cased new password",
            call: () => api.resetPassword("reset-1", "new-pw"),
            transport: postMock,
            request: ["/auth/reset-password", { token: "reset-1", new_password: "new-pw" }],
        },
    ];

    runRequestCases(cases);
});

describe("push notification devices", () => {
    const cases: RequestCase[] = [
        {
            name: "registerDeviceToken posts the token and its platform",
            call: () => api.registerDeviceToken("device-1", "android"),
            transport: postMock,
            request: ["/push/device", { token: "device-1", platform: "android" }],
        },
        {
            name: "unregisterDeviceToken deletes with the token in the body",
            call: () => api.unregisterDeviceToken("device-1"),
            transport: deleteWithBodyMock,
            request: ["/push/device", { token: "device-1" }],
        },
    ];

    runRequestCases(cases);
});

describe("profile and preference updates", () => {
    const profilePayload = {
        display_name: "Victorique",
        bio: "the golden witch of the parlour",
        avatar_url: "/media/avatar.png",
        banner_url: "/media/banner.png",
        banner_position: 50,
        favourite_character: "Beatrice",
        gender: "female",
        pronoun_subject: "she",
        pronoun_possessive: "her",
        social_twitter: "",
        social_discord: "",
        social_waifulist: "",
        social_tumblr: "",
        social_github: "",
        social_bluesky: "",
        website: "",
        dms_enabled: true,
        episode_progress: 8,
        higurashi_arc_progress: 0,
        ciconia_chapter_progress: 0,
        dob: "1986-06-06",
        dob_public: false,
        email: "victorique@example.com",
        email_public: false,
        email_notifications: true,
        play_message_sound: true,
        play_notification_sound: false,
        follow_activity_notifications: true,
        echoes_enabled: true,
        home_page: "home",
        game_board_sort: "recent",
        default_profile_tab: "posts",
    };

    const cases: RequestCase[] = [
        {
            name: "updateProfile puts the whole payload to the auth profile",
            call: () => api.updateProfile(profilePayload),
            transport: putMock,
            request: ["/auth/profile", profilePayload],
        },
        {
            name: "updateGameBoardSort puts the chosen sort",
            call: () => api.updateGameBoardSort("recent"),
            transport: putMock,
            request: ["/preferences/game-board-sort", { sort: "recent" }],
        },
        {
            name: "updateAppearance puts the theme, font and the snake cased wide layout flag",
            call: () => api.updateAppearance("gold", "serif", true),
            transport: putMock,
            request: ["/preferences/appearance", { theme: "gold", font: "serif", wide_layout: true }],
        },
        {
            name: "updateAppearance can switch the wide layout back off",
            call: () => api.updateAppearance("gold", "serif", false),
            transport: putMock,
            request: ["/preferences/appearance", { theme: "gold", font: "serif", wide_layout: false }],
        },
        {
            name: "updateChatbotOptIn puts the opt in under a snake cased flag",
            call: () => api.updateChatbotOptIn(true),
            transport: putMock,
            request: ["/preferences/chatbot-opt-in", { opted_in: true }],
        },
        {
            name: "updateChatbotOptIn puts the opt out the same way",
            call: () => api.updateChatbotOptIn(false),
            transport: putMock,
            request: ["/preferences/chatbot-opt-in", { opted_in: false }],
        },
        {
            name: "changePassword puts the old and new passwords",
            call: () => api.changePassword({ old_password: "old-pw", new_password: "new-pw" }),
            transport: putMock,
            request: ["/auth/password", { old_password: "old-pw", new_password: "new-pw" }],
        },
        {
            name: "deleteAccount deletes with the password in the body",
            call: () => api.deleteAccount({ password: "pw" }),
            transport: deleteWithBodyMock,
            request: ["/auth/account", { password: "pw" }],
        },
    ];

    runRequestCases(cases);
});

describe("profile uploads", () => {
    it("uploadBanner posts the file under the banner field", async () => {
        // given
        const file = new File(["x"], "banner.png", { type: "image/png" });

        // when
        await api.uploadBanner(file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/auth/banner");
        expect(postFormDataMock.mock.calls[0][1].get("banner")).toBe(file);
    });
});
