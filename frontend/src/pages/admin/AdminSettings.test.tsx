import { fireEvent, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { SiteSettings } from "../../types/api";
import { AdminSettings } from "./AdminSettings";

const mocks = vi.hoisted(() => ({
    useAdminSettings: vi.fn(),
    useChatbotModels: vi.fn(),
    useAdminPermissions: vi.fn(),
    update: vi.fn(),
    sendTestEmail: vi.fn(),
    testModel: vi.fn(),
    uploadOGImage: vi.fn(),
    savePending: false,
}));

vi.mock("../../api/queries/admin", () => ({
    useAdminSettings: mocks.useAdminSettings,
    useChatbotModels: mocks.useChatbotModels,
    useAdminPermissions: mocks.useAdminPermissions,
}));

vi.mock("../../api/mutations/admin", () => ({
    useUpdateAdminSettings: () => ({ mutateAsync: mocks.update, isPending: mocks.savePending }),
    useSendTestEmail: () => ({ mutateAsync: mocks.sendTestEmail, isPending: false }),
    useTestChatbotModel: () => ({ mutateAsync: mocks.testModel, isPending: false }),
    useUploadOGDefaultImage: () => ({ mutateAsync: mocks.uploadOGImage, isPending: false }),
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

function stubSettings(settings: SiteSettings | null, loading = false) {
    mocks.useAdminSettings.mockReturnValue({ settings, loading, refresh: vi.fn() });
}

function stubModels(models: string[], loading = false, refresh = vi.fn(), modelsError = "") {
    mocks.useChatbotModels.mockReturnValue({ models, modelsError, loading, refresh });
}

interface StubVanityRole {
    id: string;
    label: string;
    color: string;
    sort_order: number;
    permissions: string[];
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

function modelOptions(input: HTMLElement): string[] {
    const listID = input.getAttribute("list");
    if (!listID) {
        return [];
    }

    const list = document.getElementById(listID);
    if (!list) {
        throw new Error(`no datalist with id ${listID}`);
    }

    const values: string[] = [];
    for (const option of Array.from(list.querySelectorAll("option"))) {
        values.push(option.value);
    }

    return values;
}

function fieldFor(label: string): HTMLElement {
    const field = screen.getByText(label).closest("div");
    if (!field) {
        throw new Error(`no field wrapping ${label}`);
    }

    return field;
}

function numberInput(label: string): HTMLElement {
    return within(fieldFor(label)).getByRole("spinbutton");
}

function textInput(label: string): HTMLElement {
    return within(fieldFor(label)).getByRole("textbox");
}

function secretInput(label: string): HTMLElement {
    const input = fieldFor(label).querySelector("input");
    if (!input) {
        throw new Error(`no input inside the ${label} field`);
    }

    return input;
}

function selectFor(label: string): HTMLElement {
    return within(fieldFor(label)).getByRole("combobox");
}

beforeEach(() => {
    mocks.savePending = false;
    mocks.update.mockResolvedValue(undefined);
    mocks.sendTestEmail.mockResolvedValue(undefined);
    mocks.testModel.mockResolvedValue({ ok: true });
    mocks.uploadOGImage.mockResolvedValue({ url: "/uploads/og.jpg" });
    stubModels(["gpt-5.6-luna"]);
    stubVanityRoles([makeVanityRole()]);
});

describe("AdminSettings feature toggles", () => {
    it("waits while the settings are being fetched", () => {
        // given
        stubSettings(null, true);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByText("Loading settings...")).toBeInTheDocument();
    });

    it("keeps the maintenance wording hidden until maintenance mode is switched on", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.queryByText("Maintenance Title")).not.toBeInTheDocument();

        // when
        await user.click(screen.getByRole("switch", { name: "Maintenance Mode" }));

        // then
        expect(screen.getByText("Maintenance Title")).toBeInTheDocument();
        expect(screen.getByText("Maintenance Message")).toBeInTheDocument();
    });

    it("keeps the turnstile keys hidden until the challenge is switched on", () => {
        // given
        stubSettings({ ...VALID, turnstile_enabled: "true", turnstile_site_key: "0x4AAA" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(textInput("Site Key")).toHaveValue("0x4AAA");
    });

    it("hides the turnstile keys while the challenge is switched off", () => {
        // given
        stubSettings({ ...VALID, turnstile_enabled: "false" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.queryByText("Site Key")).not.toBeInTheDocument();
    });

    it("asks for the LiveKit connection once voice chat is switched on", () => {
        // given
        stubSettings({ ...VALID, voice_enabled: "true", livekit_url: "wss://livekit.example.com" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(textInput("LiveKit URL")).toHaveValue("wss://livekit.example.com");
        expect(screen.queryByText("Max Concurrent Streams")).not.toBeInTheDocument();
    });

    it("asks for the streaming limits and HLS directory once streaming is switched on", () => {
        // given
        stubSettings({
            ...VALID,
            streaming_enabled: "true",
            stream_hls_enabled: "true",
            stream_hls_output_dir: "/app/data/hls",
        });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByText("LiveKit URL")).toBeInTheDocument();
        expect(screen.getByText("Max Concurrent Streams")).toBeInTheDocument();
        expect(textInput("HLS Output Directory")).toHaveValue("/app/data/hls");
    });

    it("hides the HLS directory while smooth playback is switched off", () => {
        // given
        stubSettings({ ...VALID, streaming_enabled: "true", stream_hls_enabled: "false" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.queryByText("HLS Output Directory")).not.toBeInTheDocument();
    });
});

describe("AdminSettings email", () => {
    it("asks for SMTP details by default", () => {
        // given
        stubSettings({ ...VALID, smtp_host: "127.0.0.1" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(textInput("SMTP Host")).toHaveValue("127.0.0.1");
        expect(screen.queryByText("Account ID")).not.toBeInTheDocument();
    });

    it("swaps the SMTP details for Cloudflare ones when that provider is chosen", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.selectOptions(selectFor("Email Provider"), "cloudflare");

        // then
        expect(screen.getByText("Account ID")).toBeInTheDocument();
        expect(screen.queryByText("SMTP Host")).not.toBeInTheDocument();
    });

    it("confirms that a test email went out", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Send test email" }));

        // then
        expect(mocks.sendTestEmail).toHaveBeenCalledOnce();
        expect(await screen.findByText("Test email sent. Check your inbox.")).toBeInTheDocument();
    });

    it("reports why the test email failed", async () => {
        // given
        stubSettings({ ...VALID });
        mocks.sendTestEmail.mockRejectedValue(new Error("no relay is listening"));
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Send test email" }));

        // then
        expect(await screen.findByText("no relay is listening")).toBeInTheDocument();
    });
});

describe("AdminSettings chatbot model", () => {
    it("suggests every model the provider returned", () => {
        // given
        stubSettings({ ...CHATBOT });
        stubModels(["gpt-5.6-luna", "gpt-5.6-terra"]);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(modelOptions(screen.getByLabelText("Model"))).toEqual(["gpt-5.6-luna", "gpt-5.6-terra"]);
    });

    it("saves a model that the provider never listed", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_model: "" });
        stubModels(["gpt-5.6-luna"]);
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);
        await user.type(screen.getByLabelText("Model"), "gpt-6-unreleased");

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...CHATBOT, chatbot_model: "gpt-6-unreleased" });
    });

    it("keeps the hint text out of the field name and in its description", () => {
        // given
        stubSettings({ ...CHATBOT });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        const model = screen.getByLabelText("Model");
        expect(model).toHaveAccessibleName("Model");
        expect(model).toHaveAccessibleDescription(/anything not on it can still be typed in by hand/);
    });

    it("confirms that the model answered", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_model: "gpt-5.6-luna" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Test model" }));

        // then
        expect(mocks.testModel).toHaveBeenCalledWith("gpt-5.6-luna");
        expect(await screen.findByText("The model answered. Save your changes to put it live.")).toBeInTheDocument();
    });

    it("tests whatever model is currently typed in rather than the saved one", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_model: "gpt-5.6-luna" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);
        await user.clear(screen.getByLabelText("Model"));
        await user.type(screen.getByLabelText("Model"), "gpt-6-unreleased");

        // when
        await user.click(screen.getByRole("button", { name: "Test model" }));

        // then
        expect(mocks.testModel).toHaveBeenCalledWith("gpt-6-unreleased");
    });

    it("shows what the provider said when the model refused", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_model: "gpt-6-unreleased" });
        mocks.testModel.mockResolvedValue({ ok: false, error: "the model gpt-6-unreleased does not exist" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Test model" }));

        // then
        expect(await screen.findByText("the model gpt-6-unreleased does not exist")).toBeInTheDocument();
    });

    it("reports a test that could not be run at all", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_model: "gpt-5.6-luna" });
        mocks.testModel.mockRejectedValue(new Error("the admin API is unreachable"));
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Test model" }));

        // then
        expect(await screen.findByText("the admin API is unreachable")).toBeInTheDocument();
    });

    it("refuses to test while no model has been named", () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_model: "   " });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByRole("button", { name: "Test model" })).toBeDisabled();
    });

    it("offers nothing to test while the chatbot is switched off", () => {
        // given
        stubSettings({ ...VALID, chatbot_enabled: "false" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.queryByRole("button", { name: "Test model" })).not.toBeInTheDocument();
        expect(screen.queryByLabelText("Model")).not.toBeInTheDocument();
    });
});

describe("AdminSettings chatbot key gate", () => {
    it("locks everything below the API key until a key is saved", () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_api_key: "" });
        stubModels([]);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByText(/No OpenAI API key is saved yet/)).toBeInTheDocument();
        expect(secretInput("Admin Key")).toBeDisabled();
        expect(screen.getByLabelText("Model")).toBeDisabled();
        expect(selectFor("Reasoning Effort")).toBeDisabled();
        expect(numberInput("Max Replies Per Day")).toBeDisabled();
        expect(screen.getByRole("button", { name: "Test model" })).toBeDisabled();
        expect(screen.queryByRole("button", { name: "Try again" })).not.toBeInTheDocument();
    });

    it("keeps the API key itself editable while everything below it is locked", () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_api_key: "" });
        stubModels([]);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(secretInput("API Key")).toBeEnabled();
    });

    it("shows the reason the server gave rather than blaming the key", () => {
        // given
        stubSettings({ ...CHATBOT });
        stubModels([], false, vi.fn(), "OpenAI answered 403: Missing scopes: api.model.read.");

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByText(/OpenAI answered 403: Missing scopes: api\.model\.read\./)).toBeInTheDocument();
        expect(screen.queryByText(/No OpenAI API key is saved yet/)).not.toBeInTheDocument();
        expect(screen.getByLabelText("Model")).toBeDisabled();
        expect(document.querySelector("datalist")).toBeNull();
    });

    it("does not invent a cause when the server reported none", () => {
        // given
        stubSettings({ ...CHATBOT });
        stubModels([]);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByText(/listed no models for this key/)).toBeInTheDocument();
        expect(screen.queryByText(/No OpenAI API key is saved yet/)).not.toBeInTheDocument();
    });

    it("offers to fetch the model list again when the key could not be used", async () => {
        // given
        const refresh = vi.fn();
        stubSettings({ ...CHATBOT });
        stubModels([], false, refresh);
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Try again" }));

        // then
        expect(refresh).toHaveBeenCalledOnce();
    });

    it("says it is still checking the key while the model list is on its way", () => {
        // given
        stubSettings({ ...CHATBOT });
        stubModels([], true);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByText(/Checking the saved OpenAI API key/)).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Try again" })).not.toBeInTheDocument();
        expect(screen.getByLabelText("Model")).toBeDisabled();
    });

    it("unlocks the section once the key answers with a model list", () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_model: "gpt-5.6-luna" });
        stubModels(["gpt-5.6-luna"]);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(secretInput("Admin Key")).toBeEnabled();
        expect(screen.getByLabelText("Model")).toBeEnabled();
        expect(selectFor("Reasoning Effort")).toBeEnabled();
        expect(numberInput("Max Replies Per Day")).toBeEnabled();
        expect(screen.getByRole("button", { name: "Test model" })).toBeEnabled();
        expect(screen.queryByText(/stays locked/)).not.toBeInTheDocument();
    });

    it("leaves the master switch usable so the feature can be switched off while the provider is down", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_api_key: "" });
        stubModels([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("switch", { name: "Enable Chatbot" }));

        // then
        expect(screen.queryByLabelText("Model")).not.toBeInTheDocument();
        expect(screen.queryByText(/No OpenAI API key is saved yet/)).not.toBeInTheDocument();
    });
});

describe("AdminSettings chatbot opt in role", () => {
    const RESTRICTED: SiteSettings = { ...CHATBOT, chatbot_require_permission: "true" };

    it("keeps the role out of sight while characters are open to everyone", () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_require_permission: "false" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.queryByLabelText("Opt In Role")).not.toBeInTheDocument();
    });

    it("does not fetch the roles it will never offer", () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_require_permission: "false" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(mocks.useAdminPermissions).toHaveBeenCalledWith(false);
    });

    it("asks for a role the moment the restriction is switched on", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_require_permission: "false" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("switch", { name: "Restrict To Chatbot Permission" }));

        // then
        expect(screen.getByLabelText("Opt In Role")).toBeInTheDocument();
    });

    it("offers only the vanity roles that already carry the permission", () => {
        // given
        stubSettings({ ...RESTRICTED });
        stubVanityRoles([
            makeVanityRole({ id: "role-witch", label: "Witch's Familiar" }),
            makeVanityRole({ id: "role-goat", label: "Goat Butler", permissions: [] }),
        ]);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        const select = screen.getByLabelText("Opt In Role");
        expect(select).toHaveTextContent("Witch's Familiar");
        expect(select).not.toHaveTextContent("Goat Butler");
    });

    it("shows the role the site is already handing out", () => {
        // given
        stubSettings({ ...RESTRICTED, chatbot_opt_in_role: "role-witch" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByLabelText("Opt In Role")).toHaveValue("role-witch");
    });

    it("keeps a saved role visible after it has lost the permission", () => {
        // given
        stubSettings({ ...RESTRICTED, chatbot_opt_in_role: "role-forgotten" });
        stubVanityRoles([makeVanityRole()]);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByLabelText("Opt In Role")).toHaveValue("role-forgotten");
        expect(screen.getByText("The saved role no longer carries Summon Chatbots")).toBeInTheDocument();
    });

    it("warns that opting in hands the member the whole role", () => {
        // given
        stubSettings({ ...RESTRICTED, chatbot_opt_in_role: "role-witch" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByLabelText("Opt In Role")).toHaveAccessibleDescription(
            /Opting in grants the whole role, so anything else it carries is granted with it/,
        );
    });

    it("says where to grant the permission when no role holds it yet", () => {
        // given
        stubSettings({ ...RESTRICTED });
        stubVanityRoles([]);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByText(/No vanity role holds Summon Chatbots yet/)).toBeInTheDocument();
    });

    it("refuses to save the restriction with no role chosen", async () => {
        // given
        stubSettings({ ...RESTRICTED });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(
            screen.getByText("Restricting characters to a permission requires an opt-in role so members can opt in"),
        ).toBeInTheDocument();
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("points at the master switch when the restriction was left on with the chatbot off", async () => {
        // given
        stubSettings({ ...VALID, chatbot_enabled: "false", chatbot_require_permission: "true" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(
            screen.getByText(
                "Restricting characters to a permission requires an opt-in role. Switch Enable Chatbot on to choose one.",
            ),
        ).toBeInTheDocument();
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("saves the role the admin picked", async () => {
        // given
        stubSettings({ ...RESTRICTED });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.selectOptions(screen.getByLabelText("Opt In Role"), "role-witch");
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...RESTRICTED, chatbot_opt_in_role: "role-witch" });
    });

    it("leaves the role alone when the restriction is switched off", async () => {
        // given
        stubSettings({ ...CHATBOT, chatbot_require_permission: "false" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...CHATBOT, chatbot_require_permission: "false" });
    });

    it("holds the list still while the roles are on their way", () => {
        // given
        stubSettings({ ...RESTRICTED });
        stubVanityRoles([], true);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByLabelText("Opt In Role")).toBeDisabled();
        expect(screen.queryByText(/No vanity role holds Summon Chatbots yet/)).not.toBeInTheDocument();
    });
});

describe("AdminSettings validation", () => {
    it("refuses to save while the max body size is still zero", async () => {
        // given
        stubSettings({});
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(screen.getByText("Max body size must be greater than 0")).toBeInTheDocument();
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("refuses an image limit that is larger than the whole request limit", async () => {
        // given
        stubSettings({ ...VALID, max_image_size: String(60 * 1024 * 1024) });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(screen.getByText("Max image size (60 MB) cannot exceed max body size (50 MB)")).toBeInTheDocument();
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("refuses a session shorter than a day", async () => {
        // given
        stubSettings({ ...VALID, session_duration_days: "0" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(screen.getByText("Session duration must be at least 1 day")).toBeInTheDocument();
    });

    it("refuses to switch voice chat on without the LiveKit credentials", async () => {
        // given
        stubSettings({ ...VALID, voice_enabled: "true" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(screen.getByText("Voice chat requires LiveKit URL, API key and API secret")).toBeInTheDocument();
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("refuses the Cloudflare email provider without its credentials", async () => {
        // given
        stubSettings({ ...VALID, email_provider: "cloudflare" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(
            screen.getByText("Cloudflare email requires account ID, API token and from address"),
        ).toBeInTheDocument();
    });
});

describe("AdminSettings saving", () => {
    it("saves the loaded settings untouched when nothing was edited", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith(VALID);
        expect(await screen.findByText("Settings saved successfully")).toBeInTheDocument();
    });

    it("lays the edits over the loaded settings", async () => {
        // given
        stubSettings({ ...VALID, site_name: "When They Cry" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);
        await user.clear(textInput("Site Name"));
        await user.type(textInput("Site Name"), "City of Books");

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...VALID, site_name: "City of Books" });
    });

    it("reports why the settings could not be saved", async () => {
        // given
        stubSettings({ ...VALID });
        mocks.update.mockRejectedValue(new Error("the settings are sealed"));
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(await screen.findByText("the settings are sealed")).toBeInTheDocument();
    });

    it("locks the save control while a save is in flight", () => {
        // given
        stubSettings({ ...VALID });
        mocks.savePending = true;

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
    });
});

describe("AdminSettings unit conversion", () => {
    it("shows the file size limits in megabytes", () => {
        // given
        stubSettings({ ...VALID });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(numberInput("Max Image Size (MB)")).toHaveValue(10);
        expect(numberInput("Max Body Size (MB)")).toHaveValue(50);
    });

    it("stores a size typed in megabytes as bytes", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        fireEvent.change(numberInput("Max Image Size (MB)"), { target: { value: "8" } });
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...VALID, max_image_size: String(8 * 1024 * 1024) });
    });

    it("treats a size that is not a number as zero", () => {
        // given
        stubSettings({ ...VALID, max_video_size: "not a number" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(numberInput("Max Video Size (MB)")).toHaveValue(0);
    });

    it("shows the image pixel ceiling in megapixels", () => {
        // given
        stubSettings({ ...VALID });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(numberInput("Max Image Pixels (megapixels)")).toHaveValue(24);
    });

    it("stores a pixel ceiling typed in megapixels as pixels", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        fireEvent.change(numberInput("Max Image Pixels (megapixels)"), { target: { value: "12" } });
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...VALID, max_image_pixels: "12000000" });
    });
});

describe("AdminSettings link previews", () => {
    it("only offers to reset the embed image once a custom one is set", () => {
        // given
        stubSettings({ ...VALID, og_default_image: "" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.queryByRole("button", { name: "Reset to built-in" })).not.toBeInTheDocument();
    });

    it("resets the embed image back to the built-in one", async () => {
        // given
        stubSettings({ ...VALID, og_default_image: "/uploads/og.jpg" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Reset to built-in" }));
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...VALID, og_default_image: "" });
    });

    it("shows the embed previews for Discord and X", () => {
        // given
        stubSettings({ ...VALID });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByText("Discord")).toBeInTheDocument();
        expect(screen.getByText("X / Twitter")).toBeInTheDocument();
        expect(screen.getAllByAltText("Embed preview")).toHaveLength(2);
    });
});
