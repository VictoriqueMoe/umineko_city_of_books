import { act, renderHook } from "@testing-library/react";
import type { ChangeEvent } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { providerWrapper } from "../test-utils/render";
import type { SiteSettings } from "../types/api";
import { useAdminSettingsForm } from "./useAdminSettingsForm";

const mocks = vi.hoisted(() => ({
    useAdminSettings: vi.fn(),
    useChatbotModels: vi.fn(),
    useAdminPermissions: vi.fn(),
    update: vi.fn(),
    sendTestEmail: vi.fn(),
    testModel: vi.fn(),
    uploadOGImage: vi.fn(),
    savePending: false,
    sendingTestEmail: false,
    testingModel: false,
    uploadingOGImage: false,
}));

vi.mock("./queries/admin", () => ({
    useAdminSettings: mocks.useAdminSettings,
    useChatbotModels: mocks.useChatbotModels,
    useAdminPermissions: mocks.useAdminPermissions,
}));

vi.mock("./mutations/admin", () => ({
    useUpdateAdminSettings: () => ({ mutateAsync: mocks.update, isPending: mocks.savePending }),
    useSendTestEmail: () => ({ mutateAsync: mocks.sendTestEmail, isPending: mocks.sendingTestEmail }),
    useTestChatbotModel: () => ({ mutateAsync: mocks.testModel, isPending: mocks.testingModel }),
    useUploadOGDefaultImage: () => ({ mutateAsync: mocks.uploadOGImage, isPending: mocks.uploadingOGImage }),
}));

const VALID: SiteSettings = {
    max_body_size: String(50 * 1024 * 1024),
    max_image_size: String(10 * 1024 * 1024),
    max_image_pixels: "24000000",
    max_video_size: String(20 * 1024 * 1024),
    max_general_size: String(20 * 1024 * 1024),
    min_password_length: "8",
    session_duration_days: "30",
    max_theories_per_day: "5",
    max_responses_per_day: "20",
};

const SAVED_KEY = "********";

const CHATBOT: SiteSettings = {
    ...VALID,
    chatbot_enabled: "true",
    chatbot_max_output_tokens: "2048",
    chatbot_api_key: SAVED_KEY,
};

interface StubVanityRole {
    id: string;
    label: string;
    color: string;
    sort_order: number;
    permissions: string[];
}

function stubSettings(settings: SiteSettings | null, loading = false) {
    mocks.useAdminSettings.mockReturnValue({ settings, loading, refresh: vi.fn() });
}

function stubModels(models: string[], loading = false, refresh = vi.fn(), modelsError = "") {
    mocks.useChatbotModels.mockReturnValue({ models, modelsError, loading, refresh });
}

function stubVanityRoles(vanityRoles: StubVanityRole[], loading = false) {
    mocks.useAdminPermissions.mockReturnValue({
        catalogue: [],
        roles: [],
        vanityRoles,
        loading,
        refresh: vi.fn(),
    });
}

function makeVanityRole(overrides: Partial<StubVanityRole> = {}): StubVanityRole {
    return {
        id: "role-witch",
        label: "Witch's Familiar",
        color: "#d4af37",
        sort_order: 0,
        permissions: ["use_chatbot"],
        ...overrides,
    };
}

function setup() {
    return renderHook(() => useAdminSettingsForm(), { wrapper: providerWrapper() });
}

interface FakeFileInput {
    files: File[];
    value: string;
}

function fileEvent(file: File | null) {
    const target: FakeFileInput = { files: file ? [file] : [], value: "C:\\fakepath\\embed.jpg" };

    return { event: { target } as unknown as ChangeEvent<HTMLInputElement>, target };
}

function savedSettings(): SiteSettings {
    return mocks.update.mock.calls[0][0] as SiteSettings;
}

beforeEach(() => {
    mocks.savePending = false;
    mocks.sendingTestEmail = false;
    mocks.testingModel = false;
    mocks.uploadingOGImage = false;
    mocks.update.mockResolvedValue(undefined);
    mocks.sendTestEmail.mockResolvedValue(undefined);
    mocks.testModel.mockResolvedValue({ ok: true });
    mocks.uploadOGImage.mockResolvedValue({ image_url: "/uploads/og.jpg" });
    stubSettings({ ...VALID });
    stubModels(["gpt-5.6-luna"]);
    stubVanityRoles([makeVanityRole()]);
});

describe("useAdminSettingsForm loading", () => {
    it("waits while the settings are being fetched", () => {
        // given
        stubSettings(null, true);

        // when
        const { result } = setup();

        // then
        expect(result.current.loading).toBe(true);
        expect(result.current.settings).toEqual({});
    });

    it("shows the settings the server sent", () => {
        // given
        stubSettings({ ...VALID, site_name: "City of Books" });

        // when
        const { result } = setup();

        // then
        expect(result.current.loading).toBe(false);
        expect(result.current.settings.site_name).toBe("City of Books");
    });

    it("offers the running site name for the embed preview to fall back on", () => {
        // given a settings payload that carries no site name of its own
        stubSettings({ ...VALID });

        // when
        const { result } = setup();

        // then
        expect(result.current.defaultSiteName).toBe("When They Cry");
    });
});

describe("useAdminSettingsForm saving", () => {
    it("posts every loaded key back, including a secret the admin never retyped", async () => {
        // given a key the server only ever sends back masked
        stubSettings({ ...CHATBOT, smtp_password: SAVED_KEY });
        const { result } = setup();

        // when only the site name is edited
        act(() => result.current.updateField("site_name", "City of Books"));
        await act(() => result.current.save());

        // then the masked secrets are posted back untouched alongside the edit
        expect(savedSettings()).toEqual({ ...CHATBOT, smtp_password: SAVED_KEY, site_name: "City of Books" });
        expect(savedSettings().chatbot_api_key).toBe(SAVED_KEY);
        expect(savedSettings().smtp_password).toBe(SAVED_KEY);
    });

    it("saves the loaded settings untouched when nothing was edited", async () => {
        // given
        stubSettings({ ...VALID });
        const { result } = setup();

        // when
        await act(() => result.current.save());

        // then
        expect(mocks.update).toHaveBeenCalledWith(VALID);
        expect(result.current.success).toBe("Settings saved successfully");
    });

    it("lays the edits over the loaded settings", async () => {
        // given
        stubSettings({ ...VALID, site_name: "When They Cry" });
        const { result } = setup();

        // when
        act(() => result.current.updateField("site_name", "City of Books"));
        await act(() => result.current.save());

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...VALID, site_name: "City of Books" });
    });

    it("refuses to save settings the rules reject and says why", async () => {
        // given
        stubSettings({ ...VALID, max_body_size: "0" });
        const { result } = setup();

        // when
        await act(() => result.current.save());

        // then
        expect(result.current.error).toBe("Max body size must be greater than 0");
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("reports why the settings could not be saved", async () => {
        // given
        mocks.update.mockRejectedValue(new Error("the settings are sealed"));
        const { result } = setup();

        // when
        await act(() => result.current.save());

        // then
        expect(result.current.error).toBe("the settings are sealed");
        expect(result.current.success).toBe("");
    });

    it("names the failure itself when the server gave no reason", async () => {
        // given
        mocks.update.mockRejectedValue(new Error(""));
        const { result } = setup();

        // when
        await act(() => result.current.save());

        // then
        expect(result.current.error).toBe("Failed to save settings");
    });

    it("passes on that a save is in flight", () => {
        // given
        mocks.savePending = true;

        // when
        const { result } = setup();

        // then
        expect(result.current.saving).toBe(true);
    });

    it("drops the saved message the moment another field is edited", async () => {
        // given
        const { result } = setup();
        await act(() => result.current.save());
        expect(result.current.success).toBe("Settings saved successfully");

        // when
        act(() => result.current.updateField("site_name", "City of Books"));

        // then
        expect(result.current.success).toBe("");
    });
});

describe("useAdminSettingsForm field helpers", () => {
    it("writes a switch as the string the backend reads", () => {
        // given
        const { result } = setup();

        // when
        act(() => result.current.toggleField("maintenance_mode", true));

        // then
        expect(result.current.settings.maintenance_mode).toBe("true");

        // when
        act(() => result.current.toggleField("maintenance_mode", false));

        // then
        expect(result.current.settings.maintenance_mode).toBe("false");
    });

    it("treats a missing number as zero", () => {
        // given
        stubSettings({ ...VALID });

        // when
        const { result } = setup();

        // then
        expect(result.current.getNumber("new_account_hours")).toBe("0");
    });

    it("shows the file size limits in megabytes", () => {
        // given
        stubSettings({ ...VALID });

        // when
        const { result } = setup();

        // then
        expect(result.current.getMB("max_image_size")).toBe("10");
        expect(result.current.getMB("max_body_size")).toBe("50");
    });

    it("stores a size typed in megabytes as bytes", () => {
        // given
        const { result } = setup();

        // when
        act(() => result.current.setMB("max_image_size", "8"));

        // then
        expect(result.current.settings.max_image_size).toBe(String(8 * 1024 * 1024));
    });

    it("shows the image pixel ceiling in megapixels", () => {
        // given
        stubSettings({ ...VALID });

        // when
        const { result } = setup();

        // then
        expect(result.current.getMP("max_image_pixels")).toBe("24");
    });

    it("stores a pixel ceiling typed in megapixels as pixels", () => {
        // given
        const { result } = setup();

        // when
        act(() => result.current.setMP("max_image_pixels", "12"));

        // then
        expect(result.current.settings.max_image_pixels).toBe("12000000");
    });

    it("gives every field on the page its own id", () => {
        // given
        const { result } = setup();

        // when
        const model = result.current.fieldID("chatbot-model");
        const role = result.current.fieldID("chatbot-opt-in-role");

        // then
        expect(model).not.toBe(role);
        expect(model.endsWith("-chatbot-model")).toBe(true);
    });
});

describe("useAdminSettingsForm dronebl", () => {
    it("reads back the classes the admin has already told it to ignore", () => {
        // given
        stubSettings({ ...VALID, dronebl_ignored_classes: "8,9" });

        // when
        const { result } = setup();

        // then
        expect(result.current.dronebl.ignoredClasses.has(8)).toBe(true);
        expect(result.current.dronebl.ignoredClasses.has(13)).toBe(false);
    });

    it("saves an ignored class as the number the backend reads", () => {
        // given
        stubSettings({ ...VALID, dronebl_ignored_classes: "" });
        const { result } = setup();

        // when
        act(() => result.current.dronebl.toggleClass(13, true));

        // then
        expect(result.current.settings.dronebl_ignored_classes).toBe("13");
    });

    it("stops ignoring a class without disturbing the others", () => {
        // given
        stubSettings({ ...VALID, dronebl_ignored_classes: "8,9" });
        const { result } = setup();

        // when
        act(() => result.current.dronebl.toggleClass(8, false));

        // then
        expect(result.current.settings.dronebl_ignored_classes).toBe("9");
    });
});

describe("useAdminSettingsForm crawler feeds", () => {
    const BING = "https://www.bing.com/toolbox/bingbot.json";

    const BING_FEED = { name: "bing", label: "Bing", url: BING };
    const GOOGLE_FEED = { name: "google", label: "Google", url: "https://example.com/googlebot.json" };

    it("ticks a crawler that is already configured", () => {
        // given
        stubSettings({ ...VALID, crawler_feeds: `bing=${BING}` });

        // when
        const { result } = setup();

        // then
        expect(result.current.feeds.isKnownEnabled(BING_FEED)).toBe(true);
        expect(result.current.feeds.isKnownEnabled(GOOGLE_FEED)).toBe(false);
    });

    it("writes the published url when a crawler is ticked", () => {
        // given
        stubSettings({ ...VALID, crawler_feeds: "" });
        const { result } = setup();

        // when
        act(() => result.current.feeds.toggleKnown(BING_FEED, true));

        // then the admin never types the url themselves
        expect(result.current.settings.crawler_feeds).toBe(`bing=${BING}`);
    });

    it("adds a custom feed alongside the ticked ones", () => {
        // given
        stubSettings({ ...VALID, crawler_feeds: `bing=${BING}` });
        const { result } = setup();

        // when
        act(() => result.current.feeds.add());
        act(() => result.current.feeds.update(0, { name: "mine" }));
        act(() => result.current.feeds.update(0, { url: "https://example.com/r.json" }));

        // then
        expect(result.current.settings.crawler_feeds).toBe(`bing=${BING}\nmine=https://example.com/r.json`);
    });

    it("keeps a half typed row that serialising would have thrown away", () => {
        // given
        stubSettings({ ...VALID, crawler_feeds: "" });
        const { result } = setup();

        // when the admin has typed a name but no url yet
        act(() => result.current.feeds.add());
        act(() => result.current.feeds.update(0, { name: "mine" }));

        // then the row survives in the draft even though nothing was written to the settings
        expect(result.current.feeds.custom).toEqual([{ name: "mine", url: "" }]);
        expect(result.current.settings.crawler_feeds).toBe("");
    });

    it("removes a custom feed", () => {
        // given
        stubSettings({ ...VALID, crawler_feeds: "mine=https://example.com/r.json" });
        const { result } = setup();

        // when
        act(() => result.current.feeds.remove(0));

        // then
        expect(result.current.feeds.custom).toEqual([]);
        expect(result.current.settings.crawler_feeds).toBe("");
    });
});

describe("useAdminSettingsForm chatbot", () => {
    it("locks the section until a key is saved", () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_api_key: "" });
        stubModels([]);

        // when
        const { result } = setup();

        // then
        expect(result.current.chatbot.keySaved).toBe(false);
        expect(result.current.chatbot.locked).toBe(true);
    });

    it("keeps the section locked while the saved key lists no models", () => {
        // given
        stubSettings({ ...CHATBOT });
        stubModels([]);

        // when
        const { result } = setup();

        // then
        expect(result.current.chatbot.keySaved).toBe(true);
        expect(result.current.chatbot.locked).toBe(true);
    });

    it("judges the key on what was saved rather than on what is being typed", () => {
        // given a key typed into the form but not yet saved
        stubSettings({ ...CHATBOT, chatbot_api_key: "" });
        stubModels([]);
        const { result } = setup();

        // when
        act(() => result.current.updateField("chatbot_api_key", "sk-typed-just-now"));

        // then
        expect(result.current.chatbot.keySaved).toBe(false);
    });

    it("unlocks the section once the key answers with a model list", () => {
        // given
        stubSettings({ ...CHATBOT });
        stubModels(["gpt-5.6-luna"]);

        // when
        const { result } = setup();

        // then
        expect(result.current.chatbot.locked).toBe(false);
        expect(result.current.chatbot.models).toEqual(["gpt-5.6-luna"]);
    });

    it("confirms that the model answered", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_model: " gpt-5.6-luna " });
        const { result } = setup();

        // when
        await act(() => result.current.chatbot.testModel());

        // then
        expect(mocks.testModel).toHaveBeenCalledWith("gpt-5.6-luna");
        expect(result.current.chatbot.testMessage).toBe("The model answered. Save your changes to put it live.");
    });

    it("shows what the provider said when the model refused", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_model: "gpt-6-unreleased" });
        mocks.testModel.mockResolvedValue({ ok: false, error: "the model gpt-6-unreleased does not exist" });
        const { result } = setup();

        // when
        await act(() => result.current.chatbot.testModel());

        // then
        expect(result.current.chatbot.testError).toBe("the model gpt-6-unreleased does not exist");
    });

    it("does not invent a reason when the provider gave none", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_model: "gpt-6-unreleased" });
        mocks.testModel.mockResolvedValue({ ok: false });
        const { result } = setup();

        // when
        await act(() => result.current.chatbot.testModel());

        // then
        expect(result.current.chatbot.testError).toBe("The model did not answer");
    });

    it("reports a test that could not be run at all", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_model: "gpt-5.6-luna" });
        mocks.testModel.mockRejectedValue(new Error("the admin API is unreachable"));
        const { result } = setup();

        // when
        await act(() => result.current.chatbot.testModel());

        // then
        expect(result.current.chatbot.testError).toBe("the admin API is unreachable");
    });
});

describe("useAdminSettingsForm chatbot opt in role", () => {
    const RESTRICTED: SiteSettings = { ...CHATBOT, chatbot_require_permission: "true" };

    it("does not fetch the roles it will never offer", () => {
        // given the restriction is off
        stubSettings({ ...CHATBOT, chatbot_require_permission: "false" });

        // when
        const { result } = setup();

        // then the roles query is asked to stay disabled
        expect(result.current.chatbot.restrict).toBe(false);
        expect(mocks.useAdminPermissions).toHaveBeenCalledWith(false);
        expect(mocks.useAdminPermissions).not.toHaveBeenCalledWith(true);
    });

    it("does not fetch the roles while the chatbot itself is switched off", () => {
        // given the restriction was left on with the chatbot off
        stubSettings({ ...VALID, chatbot_enabled: "false", chatbot_require_permission: "true" });

        // when
        const { result } = setup();

        // then
        expect(result.current.chatbot.restrict).toBe(false);
        expect(mocks.useAdminPermissions).toHaveBeenCalledWith(false);
        expect(mocks.useAdminPermissions).not.toHaveBeenCalledWith(true);
    });

    it("fetches the roles once the restriction is switched on", () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_require_permission: "false" });
        const { result } = setup();

        // when
        act(() => result.current.toggleField("chatbot_require_permission", true));

        // then
        expect(result.current.chatbot.restrict).toBe(true);
        expect(mocks.useAdminPermissions).toHaveBeenLastCalledWith(true);
    });

    it("offers only the vanity roles that already carry the permission", () => {
        // given
        stubSettings({ ...RESTRICTED });
        stubVanityRoles([
            makeVanityRole({ id: "role-witch", label: "Witch's Familiar" }),
            makeVanityRole({ id: "role-goat", label: "Goat Butler", permissions: [] }),
        ]);

        // when
        const { result } = setup();

        // then
        expect(result.current.chatbot.optInRoles.map(role => role.label)).toEqual(["Witch's Familiar"]);
    });

    it("knows when the saved role has lost the permission", () => {
        // given
        stubSettings({ ...RESTRICTED, chatbot_opt_in_role: "role-forgotten" });
        stubVanityRoles([makeVanityRole()]);

        // when
        const { result } = setup();

        // then
        expect(result.current.chatbot.optInRoleID).toBe("role-forgotten");
        expect(result.current.chatbot.optInRoleListed).toBe(false);
    });

    it("recognises a saved role that still carries the permission", () => {
        // given
        stubSettings({ ...RESTRICTED, chatbot_opt_in_role: "role-witch" });

        // when
        const { result } = setup();

        // then
        expect(result.current.chatbot.optInRoleListed).toBe(true);
    });

    it("holds the list still while the roles are on their way", () => {
        // given
        stubSettings({ ...RESTRICTED });
        stubVanityRoles([], true);

        // when
        const { result } = setup();

        // then
        expect(result.current.chatbot.rolesLoading).toBe(true);
        expect(result.current.chatbot.optInRoles).toEqual([]);
    });
});

describe("useAdminSettingsForm test email", () => {
    it("confirms that a test email went out", async () => {
        // given
        const { result } = setup();

        // when
        await act(() => result.current.emailTest.send());

        // then
        expect(mocks.sendTestEmail).toHaveBeenCalledOnce();
        expect(result.current.emailTest.message).toBe("Test email sent. Check your inbox.");
    });

    it("reports why the test email failed", async () => {
        // given
        mocks.sendTestEmail.mockRejectedValue(new Error("no relay is listening"));
        const { result } = setup();

        // when
        await act(() => result.current.emailTest.send());

        // then
        expect(result.current.emailTest.error).toBe("no relay is listening");
        expect(result.current.emailTest.message).toBe("");
    });

    it("passes on that a test email is on its way", () => {
        // given
        mocks.sendingTestEmail = true;

        // when
        const { result } = setup();

        // then
        expect(result.current.emailTest.sending).toBe(true);
    });
});

describe("useAdminSettingsForm embed image", () => {
    it("puts the uploaded image into the settings", async () => {
        // given
        const { result } = setup();
        const { event, target } = fileEvent(new File(["golden butterflies"], "embed.jpg", { type: "image/jpeg" }));

        // when
        await act(() => result.current.ogImage.onSelected(event));

        // then
        expect(result.current.settings.og_default_image).toBe("/uploads/og.jpg");
        expect(target.value).toBe("");
    });

    it("does nothing when the file picker was dismissed", async () => {
        // given
        const { result } = setup();
        const { event } = fileEvent(null);

        // when
        await act(() => result.current.ogImage.onSelected(event));

        // then
        expect(mocks.uploadOGImage).not.toHaveBeenCalled();
    });

    it("reports why the image could not be uploaded", async () => {
        // given
        mocks.uploadOGImage.mockRejectedValue(new Error("that is not a JPG"));
        const { result } = setup();
        const { event } = fileEvent(new File(["not a jpg"], "embed.png", { type: "image/png" }));

        // when
        await act(() => result.current.ogImage.onSelected(event));

        // then
        expect(result.current.ogImage.error).toBe("that is not a JPG");
    });

    it("only offers to reset once a custom image is set", () => {
        // given
        stubSettings({ ...VALID, og_default_image: "" });

        // when
        const { result } = setup();

        // then
        expect(result.current.ogImage.hasCustom).toBe(false);
    });

    it("resets the embed image back to the built-in one", () => {
        // given
        stubSettings({ ...VALID, og_default_image: "/uploads/og.jpg" });
        const { result } = setup();
        expect(result.current.ogImage.hasCustom).toBe(true);

        // when
        act(() => result.current.ogImage.clear());

        // then
        expect(result.current.settings.og_default_image).toBe("");
        expect(result.current.ogImage.hasCustom).toBe(false);
    });
});
