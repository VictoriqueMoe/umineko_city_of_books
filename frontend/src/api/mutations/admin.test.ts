import type { QueryClient } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { ChatbotPayload, CreateBannedWordRequest } from "../../types/api";
import {
    useAddBannedGif,
    useAdminDeleteUser,
    useAssignVanityRole,
    useBanUser,
    useCreateAnnouncement,
    useCreateChatbot,
    useCreateGlobalBannedWord,
    useCreateInvite,
    useCreateVanityRole,
    useDeleteAnnouncement,
    useDeleteChatbot,
    useDeleteGlobalBannedWord,
    useDeleteInvite,
    useDeleteVanityRole,
    useForceLogoutUser,
    useLockUser,
    usePinAnnouncement,
    useRemoveBannedGif,
    useRemoveUserRole,
    useResetUserPassword,
    useResolveReport,
    useSendTestEmail,
    useSetDisplayNameLock,
    useSetUserDisplayName,
    useSetUserEmail,
    useSetUserRole,
    useUnassignVanityRole,
    useUnbanUser,
    useUnlockUser,
    useUnverifyUserEmail,
    useUpdateAdminSettings,
    useUpdateAnnouncement,
    useUpdateChatbot,
    useUpdateDetectiveScore,
    useUpdateGlobalBannedWord,
    useUpdateGMScore,
    useUpdateVanityRole,
    useUploadOGDefaultImage,
    useVerifyUserEmail,
} from "./admin";

const mocks = vi.hoisted(() => ({
    addBannedGif: vi.fn(),
    adminDeleteUser: vi.fn(),
    assignVanityRole: vi.fn(),
    banUser: vi.fn(),
    createAnnouncement: vi.fn(),
    createChatbot: vi.fn(),
    createGlobalBannedWord: vi.fn(),
    createInvite: vi.fn(),
    createVanityRole: vi.fn(),
    deleteAnnouncement: vi.fn(),
    deleteChatbot: vi.fn(),
    deleteGlobalBannedWord: vi.fn(),
    deleteInvite: vi.fn(),
    deleteVanityRole: vi.fn(),
    forceLogoutUser: vi.fn(),
    lockUser: vi.fn(),
    pinAnnouncement: vi.fn(),
    removeBannedGif: vi.fn(),
    removeUserRole: vi.fn(),
    resetUserPassword: vi.fn(),
    resolveReport: vi.fn(),
    sendTestEmail: vi.fn(),
    setDisplayNameLock: vi.fn(),
    setUserDisplayName: vi.fn(),
    setUserEmail: vi.fn(),
    setUserRole: vi.fn(),
    unassignVanityRole: vi.fn(),
    unbanUser: vi.fn(),
    unlockUser: vi.fn(),
    unverifyUserEmail: vi.fn(),
    updateAdminSettings: vi.fn(),
    updateAnnouncement: vi.fn(),
    updateChatbot: vi.fn(),
    updateDetectiveScore: vi.fn(),
    updateGlobalBannedWord: vi.fn(),
    updateGMScore: vi.fn(),
    updateVanityRole: vi.fn(),
    uploadOGDefaultImage: vi.fn(),
    verifyUserEmail: vi.fn(),
}));

vi.mock("../endpoints", () => mocks);

const userId = "11111111-1111-1111-1111-111111111111";

function client() {
    const qc = createTestQueryClient();

    return { qc, invalidate: vi.spyOn(qc, "invalidateQueries") };
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

beforeEach(() => {
    for (const fn of Object.values(mocks)) {
        fn.mockResolvedValue(undefined);
    }
});

describe("admin role mutations", () => {
    it("grants a role to a user and refreshes every admin view", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useSetUserRole, { id: userId, role: "moderator" }, qc);

        // then
        expect(mocks.setUserRole).toHaveBeenCalledWith(userId, "moderator");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin"] });
    });

    it("removes a role from a user and refreshes every admin view", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useRemoveUserRole, { id: userId, role: "moderator" }, qc);

        // then
        expect(mocks.removeUserRole).toHaveBeenCalledWith(userId, "moderator");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin"] });
    });

    it("sets a detective score and refreshes only the admin user views", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUpdateDetectiveScore, { id: userId, desiredScore: 42 }, qc);

        // then
        expect(mocks.updateDetectiveScore).toHaveBeenCalledWith(userId, 42);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "user"] });
    });

    it("sets a game master score and refreshes only the admin user views", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUpdateGMScore, { id: userId, desiredScore: -3 }, qc);

        // then
        expect(mocks.updateGMScore).toHaveBeenCalledWith(userId, -3);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "user"] });
    });
});

describe("admin moderation mutations", () => {
    it("bans a user with the reason the moderator typed", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useBanUser, { id: userId, reason: "repeated harassment" }, qc);

        // then
        expect(mocks.banUser).toHaveBeenCalledWith(userId, "repeated harassment");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin"] });
    });

    it("leaves the cache untouched when the ban is rejected by the server", async () => {
        // given
        const { qc, invalidate } = client();
        mocks.banUser.mockRejectedValue(new Error("forbidden"));
        const { result } = renderHook(useBanUser, { wrapper: providerWrapper({ queryClient: qc }) });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync({ id: userId, reason: "nope" })).rejects.toThrow("forbidden");
        });

        // then
        expect(invalidate).not.toHaveBeenCalled();
    });

    it("unbans a user by id alone", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUnbanUser, userId, qc);

        // then
        expect(mocks.unbanUser).toHaveBeenCalledWith(userId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin"] });
    });

    it("locks a user with a reason", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useLockUser, { id: userId, reason: "under review" }, qc);

        // then
        expect(mocks.lockUser).toHaveBeenCalledWith(userId, "under review");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin"] });
    });

    it("unlocks a user by id alone", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUnlockUser, userId, qc);

        // then
        expect(mocks.unlockUser).toHaveBeenCalledWith(userId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin"] });
    });

    it("deletes a user account and refreshes every admin view", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useAdminDeleteUser, userId, qc);

        // then
        expect(mocks.adminDeleteUser).toHaveBeenCalledWith(userId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin"] });
    });

    it("forces a user to log out and refreshes the admin user views", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useForceLogoutUser, userId, qc);

        // then
        expect(mocks.forceLogoutUser).toHaveBeenCalledWith(userId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "user"] });
    });
});

describe("admin account detail mutations", () => {
    it("returns the freshly generated password without touching the cache", async () => {
        // given
        const { qc, invalidate } = client();
        mocks.resetUserPassword.mockResolvedValue({ password: "golden-butterfly" });

        // when
        const data = await runMutation(useResetUserPassword, userId, qc);

        // then
        expect(mocks.resetUserPassword).toHaveBeenCalledWith(userId);
        expect(data).toEqual({ password: "golden-butterfly" });
        expect(invalidate).not.toHaveBeenCalled();
    });

    it("changes the email address of a user", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useSetUserEmail, { id: userId, email: "beato@rokkenjima.test" }, qc);

        // then
        expect(mocks.setUserEmail).toHaveBeenCalledWith(userId, "beato@rokkenjima.test");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "user"] });
    });

    it("marks an email address as verified", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useVerifyUserEmail, userId, qc);

        // then
        expect(mocks.verifyUserEmail).toHaveBeenCalledWith(userId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "user"] });
    });

    it("marks an email address as unverified", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUnverifyUserEmail, userId, qc);

        // then
        expect(mocks.unverifyUserEmail).toHaveBeenCalledWith(userId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "user"] });
    });

    it("renames a user and refreshes both the admin views and their public profile", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useSetUserDisplayName, { id: userId, displayName: "Beatrice" }, qc);

        // then
        expect(mocks.setUserDisplayName).toHaveBeenCalledWith(userId, "Beatrice");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["profile"] });
    });

    it("locks the display name of a user", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useSetDisplayNameLock, { id: userId, locked: true }, qc);

        // then
        expect(mocks.setDisplayNameLock).toHaveBeenCalledWith(userId, true);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "user"] });
    });
});

describe("admin site settings mutations", () => {
    it("saves the settings and refreshes the admin settings, the chatbot models and the public site info", async () => {
        // given
        const { qc, invalidate } = client();
        const settings = { site_name: "When They Cry", default_theme: "featherine" };

        // when
        await runMutation(useUpdateAdminSettings, settings, qc);

        // then
        expect(mocks.updateAdminSettings).toHaveBeenCalledWith(settings);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "settings"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "chatbots", "models"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["site-info"] });
    });

    it("sends a test email without invalidating anything", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useSendTestEmail, undefined, qc);

        // then
        expect(mocks.sendTestEmail).toHaveBeenCalledWith();
        expect(invalidate).not.toHaveBeenCalled();
    });

    it("uploads a default share image and hands back the stored url", async () => {
        // given
        const { qc, invalidate } = client();
        const file = new File(["png"], "og.png", { type: "image/png" });
        mocks.uploadOGDefaultImage.mockResolvedValue({ url: "/uploads/og.png" });

        // when
        const data = await runMutation(useUploadOGDefaultImage, file, qc);

        // then
        expect(mocks.uploadOGDefaultImage).toHaveBeenCalledWith(file);
        expect(data).toEqual({ url: "/uploads/og.png" });
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("admin invite mutations", () => {
    it("creates an invite and refreshes the invite list", async () => {
        // given
        const { qc, invalidate } = client();
        mocks.createInvite.mockResolvedValue({ code: "beato-1" });

        // when
        const data = await runMutation(useCreateInvite, undefined, qc);

        // then
        expect(mocks.createInvite).toHaveBeenCalledWith();
        expect(data).toEqual({ code: "beato-1" });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "invites"] });
    });

    it("revokes an invite by its code", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useDeleteInvite, "beato-1", qc);

        // then
        expect(mocks.deleteInvite).toHaveBeenCalledWith("beato-1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "invites"] });
    });
});

describe("admin report mutations", () => {
    it("resolves a report with the moderator comment and refreshes the report list", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useResolveReport, { id: 7, comment: "handled" }, qc);

        // then
        expect(mocks.resolveReport).toHaveBeenCalledWith(7, "handled");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "reports"] });
    });
});

describe("admin banned word mutations", () => {
    const rule: CreateBannedWordRequest = {
        pattern: "kanon",
        match_mode: "whole_word",
        case_sensitive: false,
        action: "delete",
    };

    it("creates a global rule and refreshes the global banned word list", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useCreateGlobalBannedWord, rule, qc);

        // then
        expect(mocks.createGlobalBannedWord).toHaveBeenCalledWith(rule);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "banned-words", "global"] });
    });

    it("updates a global rule by its id", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUpdateGlobalBannedWord, { ruleId: "rule-1", req: rule }, qc);

        // then
        expect(mocks.updateGlobalBannedWord).toHaveBeenCalledWith("rule-1", rule);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "banned-words", "global"] });
    });

    it("deletes a global rule by its id", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useDeleteGlobalBannedWord, "rule-1", qc);

        // then
        expect(mocks.deleteGlobalBannedWord).toHaveBeenCalledWith("rule-1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "banned-words", "global"] });
    });
});

describe("admin banned gif mutations", () => {
    it("bans a gif with the raw input and an optional reason", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useAddBannedGif, { input: "https://giphy.test/abc", reason: "gore" }, qc);

        // then
        expect(mocks.addBannedGif).toHaveBeenCalledWith({ input: "https://giphy.test/abc", reason: "gore" });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "banned-gifs"] });
    });

    it("bans a gif without a reason when none was given", async () => {
        // given
        const { qc } = client();

        // when
        await runMutation(useAddBannedGif, { input: "abc" }, qc);

        // then
        expect(mocks.addBannedGif).toHaveBeenCalledWith({ input: "abc" });
    });

    it("unbans a gif by its kind and value", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useRemoveBannedGif, { kind: "id", value: "abc" }, qc);

        // then
        expect(mocks.removeBannedGif).toHaveBeenCalledWith("id", "abc");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "banned-gifs"] });
    });
});

describe("admin announcement mutations", () => {
    it("creates an announcement and refreshes both the admin list and the public list", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useCreateAnnouncement, { title: "Tea party", body: "at midnight" }, qc);

        // then
        expect(mocks.createAnnouncement).toHaveBeenCalledWith("Tea party", "at midnight");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "announcements"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["announcements"] });
    });

    it("updates an announcement by id", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUpdateAnnouncement, { id: "a1", title: "Tea party", body: "at dawn" }, qc);

        // then
        expect(mocks.updateAnnouncement).toHaveBeenCalledWith("a1", "Tea party", "at dawn");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "announcements"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["announcements"] });
    });

    it("deletes an announcement by id", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useDeleteAnnouncement, "a1", qc);

        // then
        expect(mocks.deleteAnnouncement).toHaveBeenCalledWith("a1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "announcements"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["announcements"] });
    });

    it("unpins an announcement when the pinned flag is false", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(usePinAnnouncement, { id: "a1", pinned: false }, qc);

        // then
        expect(mocks.pinAnnouncement).toHaveBeenCalledWith("a1", false);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["announcements"] });
    });
});

describe("admin vanity role mutations", () => {
    const role = { label: "Witch", color: "#ffd700", sort_order: 2 };

    it("creates a vanity role and refreshes the vanity role list", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useCreateVanityRole, role, qc);

        // then
        expect(mocks.createVanityRole).toHaveBeenCalledWith(role);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "vanity-roles"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "permissions"] });
    });

    it("updates a vanity role by id", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUpdateVanityRole, { id: "r1", data: role }, qc);

        // then
        expect(mocks.updateVanityRole).toHaveBeenCalledWith("r1", role);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "vanity-roles"] });
    });

    it("deletes a vanity role by id", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useDeleteVanityRole, "r1", qc);

        // then
        expect(mocks.deleteVanityRole).toHaveBeenCalledWith("r1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "vanity-roles"] });
    });

    it("assigns a vanity role and refreshes the holders of that role rather than the role list", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useAssignVanityRole, { roleId: "r1", userId }, qc);

        // then
        expect(mocks.assignVanityRole).toHaveBeenCalledWith("r1", userId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "vanity-role-users"] });
        expect(invalidate).not.toHaveBeenCalledWith({ queryKey: ["admin", "vanity-roles"] });
    });

    it("unassigns a vanity role and refreshes the holders of that role", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUnassignVanityRole, { roleId: "r1", userId }, qc);

        // then
        expect(mocks.unassignVanityRole).toHaveBeenCalledWith("r1", userId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "vanity-role-users"] });
    });
});

describe("admin chatbot mutations", () => {
    const bot: ChatbotPayload = {
        username: "beatrice",
        display_name: "Beatrice",
        avatar_url: "",
        system_prompt: "You are the Golden Witch.",
        base_prompt_id: null,
        model: "gpt-5.6-luna",
        reasoning_effort: "low",
        verbosity: "medium",
        max_output_tokens: 400,
        enabled: true,
    };

    it("creates a chatbot and refreshes the sidebar list as well as the admin list", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useCreateChatbot, bot, qc);

        // then
        expect(mocks.createChatbot).toHaveBeenCalledWith(bot);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "chatbots"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["chatbots"] });
    });

    it("updates a chatbot by id and refreshes the sidebar list", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useUpdateChatbot, { id: "b1", data: bot }, qc);

        // then
        expect(mocks.updateChatbot).toHaveBeenCalledWith("b1", bot);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "chatbots"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["chatbots"] });
    });

    it("deletes a chatbot by id and refreshes the sidebar list", async () => {
        // given
        const { qc, invalidate } = client();

        // when
        await runMutation(useDeleteChatbot, "b1", qc);

        // then
        expect(mocks.deleteChatbot).toHaveBeenCalledWith("b1");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "chatbots"] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["chatbots"] });
    });
});
