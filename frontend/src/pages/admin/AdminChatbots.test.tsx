import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { Chatbot, ChatbotBasePrompt, ChatbotChannelUsage, ChatbotUsage, SiteSettings } from "../../types/api";
import { AdminChatbots } from "./AdminChatbots";
import styles from "./AdminChatbots.module.css";

const FAILURE_NOTE =
    "Failures are almost always a model id the provider does not recognise, a revoked or expired API key, or a quota that has run out.";

const SAVED_KEY = "********";

const mocks = vi.hoisted(() => ({
    useChatbots: vi.fn(),
    useChatbotUsage: vi.fn(),
    useChatbotModels: vi.fn(),
    useAdminSettings: vi.fn(),
    useChatbotBasePrompts: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
    createBase: vi.fn(),
    updateBase: vi.fn(),
    removeBase: vi.fn(),
    checkUsername: vi.fn(),
}));

vi.mock("../../api/endpoints", async importOriginal => ({
    ...(await importOriginal<typeof import("../../api/endpoints")>()),
    checkUsernameAvailable: mocks.checkUsername,
}));

vi.mock("../../api/queries/admin", () => ({
    useChatbots: mocks.useChatbots,
    useChatbotUsage: mocks.useChatbotUsage,
    useChatbotModels: mocks.useChatbotModels,
    useAdminSettings: mocks.useAdminSettings,
    useChatbotBasePrompts: mocks.useChatbotBasePrompts,
}));

vi.mock("../../api/mutations/admin", () => ({
    useCreateChatbot: () => ({ mutateAsync: mocks.create, isPending: false }),
    useUpdateChatbot: () => ({ mutateAsync: mocks.update, isPending: false }),
    useDeleteChatbot: () => ({ mutateAsync: mocks.remove, isPending: false }),
    useCreateChatbotBasePrompt: () => ({ mutateAsync: mocks.createBase, isPending: false }),
    useUpdateChatbotBasePrompt: () => ({ mutateAsync: mocks.updateBase, isPending: false }),
    useDeleteChatbotBasePrompt: () => ({ mutateAsync: mocks.removeBase, isPending: false }),
}));

function makeBot(overrides: Partial<Chatbot> = {}): Chatbot {
    return {
        id: "bot-1",
        user_id: "user-1",
        username: "beatrice",
        display_name: "Beatrice",
        avatar_url: "",
        system_prompt: "You are the Golden Witch.",
        base_prompt_id: null,
        model: "",
        reasoning_effort: "",
        verbosity: "",
        max_output_tokens: 0,
        enabled: true,
        ...overrides,
    };
}

function makeUsage(overrides: Partial<ChatbotUsage> = {}): ChatbotUsage {
    return {
        invocations: 1234,
        prompt_tokens: 56000,
        cached_prompt_tokens: 48000,
        cache_write_tokens: 1200,
        completion_tokens: 7800,
        reasoning_tokens: 900,
        billed_usd: null,
        failed: 0,
        quota: 0,
        channels: [],
        ...overrides,
    };
}

function makeChannel(overrides: Partial<ChatbotChannelUsage> = {}): ChatbotChannelUsage {
    return {
        channel: "group",
        invocations: 10,
        prompt_tokens: 1000,
        cached_prompt_tokens: 800,
        cache_write_tokens: 100,
        completion_tokens: 200,
        reasoning_tokens: 50,
        ...overrides,
    };
}

function stubBots(bots: Chatbot[], loading = false) {
    mocks.useChatbots.mockReturnValue({ bots, loading, refresh: vi.fn() });
}

function makeBasePrompt(overrides: Partial<ChatbotBasePrompt> = {}): ChatbotBasePrompt {
    return {
        id: "base-1",
        name: "game witch",
        prompt: "You are a witch of the game boards.",
        bot_count: 0,
        created_at: "2026-08-11T00:00:00Z",
        updated_at: "2026-08-11T00:00:00Z",
        ...overrides,
    };
}

function stubBasePrompts(basePrompts: ChatbotBasePrompt[]) {
    mocks.useChatbotBasePrompts.mockReturnValue({ basePrompts, loading: false, refresh: vi.fn() });
}

function stubUsage(usage: ChatbotUsage | null, loading = false) {
    mocks.useChatbotUsage.mockReturnValue({ usage, loading, refresh: vi.fn() });
}

function stubUsagePerRange(byDays: Record<number, ChatbotUsage>) {
    mocks.useChatbotUsage.mockImplementation((days: number) => ({
        usage: byDays[days] ?? null,
        loading: !byDays[days],
        refresh: vi.fn(),
    }));
}

function stubModels(models: string[], loading = false, refresh = vi.fn(), modelsError = "") {
    mocks.useChatbotModels.mockReturnValue({ models, modelsError, loading, refresh });
}

function stubSettings(settings: SiteSettings) {
    mocks.useAdminSettings.mockReturnValue({ settings, loading: false, refresh: vi.fn() });
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

function usagePanel(label: string): HTMLElement {
    const panel = screen.getByText(label).closest("div");
    if (!panel) {
        throw new Error(`no usage panel wrapping ${label}`);
    }

    return panel;
}

beforeEach(() => {
    mocks.create.mockResolvedValue(undefined);
    mocks.update.mockResolvedValue(undefined);
    mocks.remove.mockResolvedValue(undefined);
    mocks.createBase.mockResolvedValue(undefined);
    mocks.updateBase.mockResolvedValue(undefined);
    mocks.removeBase.mockResolvedValue(undefined);
    stubBasePrompts([]);
    stubUsage(null, true);
    stubModels(["gpt-5.6-luna"]);
    stubSettings({ chatbot_api_key: SAVED_KEY });
    mocks.checkUsername.mockImplementation((username: string) => Promise.resolve({ username, available: true }));
});

describe("AdminChatbots base prompts", () => {
    it("says so when no base prompt has been written yet", () => {
        // given
        stubBots([]);
        stubBasePrompts([]);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(screen.getByText("No base prompts yet.")).toBeInTheDocument();
    });

    it("lists a base prompt with how many bots extend it", () => {
        // given
        stubBots([]);
        stubBasePrompts([makeBasePrompt({ name: "game witch", bot_count: 3 })]);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(screen.getByText("game witch")).toBeInTheDocument();
        expect(screen.getByText(/3 bots/)).toBeInTheDocument();
    });

    it("creates a base prompt from the name and text typed into the form", async () => {
        // given
        stubBots([]);
        stubBasePrompts([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Base Prompt" }));
        await user.type(screen.getByLabelText("Name"), "game witch");
        await user.type(screen.getByLabelText("Prompt"), "You are a witch of the game boards.");
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.createBase).toHaveBeenCalledWith({
            name: "game witch",
            prompt: "You are a witch of the game boards.",
        });
    });

    it("saves an edit against the base prompt it came from", async () => {
        // given
        stubBots([]);
        stubBasePrompts([makeBasePrompt({ id: "base-9", name: "game witch" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Edit game witch" }));
        await user.clear(screen.getByLabelText("Name"));
        await user.type(screen.getByLabelText("Name"), "voyager");
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.updateBase).toHaveBeenCalledWith({
            id: "base-9",
            data: { name: "voyager", prompt: "You are a witch of the game boards." },
        });
    });

    it("refuses to delete a base prompt that bots still extend", async () => {
        // given
        stubBots([]);
        stubBasePrompts([makeBasePrompt({ name: "game witch", bot_count: 2 })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete game witch" }));

        // then
        expect(mocks.removeBase).not.toHaveBeenCalled();
        expect(screen.getByText(/still used by 2 bot/)).toBeInTheDocument();
    });

    it("sends the base prompt chosen on the bot form", async () => {
        // given
        stubBots([]);
        stubBasePrompts([makeBasePrompt({ id: "base-7", name: "game witch" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Bot" }));
        await user.type(screen.getByLabelText("Username"), "beato");
        await user.type(screen.getByLabelText("Display Name"), "Beato");
        await user.type(screen.getByLabelText("System Prompt"), "You are the Golden Witch.");
        await user.selectOptions(screen.getByLabelText("Base Prompt"), "base-7");
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({ base_prompt_id: "base-7" }));
    });

    it("preloads the base prompt a bot already extends when editing it", async () => {
        // given
        stubBots([makeBot({ id: "bot-5", base_prompt_id: "base-7" })]);
        stubBasePrompts([makeBasePrompt({ id: "base-7", name: "game witch" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Edit Beatrice" }));

        // then
        expect(screen.getByLabelText("Base Prompt")).toHaveValue("base-7");
    });

    it("deletes an unused base prompt once it is confirmed", async () => {
        // given
        stubBots([]);
        stubBasePrompts([makeBasePrompt({ id: "base-4", bot_count: 0 })]);
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete game witch" }));

        // then
        expect(mocks.removeBase).toHaveBeenCalledWith("base-4");
        confirm.mockRestore();
    });
});

describe("AdminChatbots list", () => {
    it("waits while the bots are being fetched", () => {
        // given
        stubBots([], true);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(screen.getByText("Loading chatbots...")).toBeInTheDocument();
    });

    it("says so when no bot has been built yet", () => {
        // given
        stubBots([]);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(screen.getByText("No chatbots yet.")).toBeInTheDocument();
    });

    it("lists each bot with its name, handle and enabled state", () => {
        // given
        stubBots([
            makeBot({ display_name: "Beatrice", username: "beatrice", enabled: true }),
            makeBot({ id: "bot-2", display_name: "Bernkastel", username: "bern", enabled: false }),
        ]);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("@beatrice")).toBeInTheDocument();
        expect(screen.getByText("Bernkastel")).toBeInTheDocument();
        expect(screen.getByText("@bern")).toBeInTheDocument();
        expect(screen.getByRole("switch", { name: "Enabled Beatrice" })).toHaveAttribute("aria-checked", "true");
        expect(screen.getByRole("switch", { name: "Enabled Bernkastel" })).toHaveAttribute("aria-checked", "false");
    });

    it("names every row control after the bot it acts on", () => {
        // given
        stubBots([makeBot({ display_name: "Beatrice" }), makeBot({ id: "bot-2", display_name: "Bernkastel" })]);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(screen.getByRole("button", { name: "Edit Beatrice" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete Beatrice" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Edit Bernkastel" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete Bernkastel" })).toBeInTheDocument();
    });

    it("switches a bot off without opening the form", async () => {
        // given
        stubBots([makeBot({ id: "bot-7", model: "gpt-5", reasoning_effort: "high", max_output_tokens: 2048 })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("switch", { name: "Enabled Beatrice" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({
            id: "bot-7",
            data: expect.objectContaining({
                username: "beatrice",
                model: "gpt-5",
                reasoning_effort: "high",
                max_output_tokens: 2048,
                enabled: false,
            }),
        });
    });

    it("switches the bot that was clicked and no other", async () => {
        // given
        stubBots([
            makeBot({ id: "bot-1", display_name: "Beatrice", enabled: true }),
            makeBot({ id: "bot-2", display_name: "Bernkastel", username: "bern", enabled: false }),
        ]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("switch", { name: "Enabled Bernkastel" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({
            id: "bot-2",
            data: expect.objectContaining({ username: "bern", enabled: true }),
        });
    });

    it("says which bot could not be switched", async () => {
        // given
        stubBots([makeBot({ display_name: "Beatrice" })]);
        mocks.update.mockRejectedValue(new Error("the model is unreachable"));
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("switch", { name: "Enabled Beatrice" }));

        // then
        expect(await screen.findByText("Could not switch Beatrice off: the model is unreachable")).toBeInTheDocument();
    });
});

describe("AdminChatbots creating", () => {
    it("refuses to save a bot with no handle or name", async () => {
        // given
        stubBots([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Bot" }));

        // then
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("labels every form field", async () => {
        // given
        stubBots([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Bot" }));

        // then
        expect(screen.getByLabelText("Username")).toBeInTheDocument();
        expect(screen.getByLabelText("Display Name")).toBeInTheDocument();
        expect(screen.getByLabelText("Avatar URL")).toBeInTheDocument();
        expect(screen.getByLabelText("System Prompt")).toBeInTheDocument();
        expect(screen.getByLabelText("Model")).toBeInTheDocument();
        expect(screen.getByLabelText("Reasoning Effort")).toBeInTheDocument();
        expect(screen.getByLabelText("Max Output Tokens")).toBeInTheDocument();
    });

    it("keeps the hint text out of a field name and in its description", async () => {
        // given
        stubBots([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Bot" }));

        // then
        const username = screen.getByLabelText("Username");
        expect(username).toHaveAccessibleName("Username");
        expect(username).toHaveAccessibleDescription(
            "The handle members type to reach the bot. It has to be free, exactly like a human account.",
        );
    });

    it("checks the handle when the username field loses focus", async () => {
        // given
        stubBots([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Create Bot" }));
        await user.type(screen.getByLabelText("Username"), "  beato  ");

        // when
        await user.tab();

        // then
        expect(mocks.checkUsername).toHaveBeenCalledWith("beato");
        expect(await screen.findByText("@beato is free.", undefined, { timeout: 5000 })).toBeInTheDocument();
    });

    it("refuses to save a handle the server says is taken", async () => {
        // given
        stubBots([]);
        mocks.checkUsername.mockResolvedValue({ username: "beato", available: false });
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Create Bot" }));
        await user.type(screen.getByLabelText("Username"), "beato");
        await user.type(screen.getByLabelText("Display Name"), "Beato");

        // when
        await user.click(screen.getByLabelText("Display Name"));

        // then
        expect(
            await screen.findByText("@beato is already taken. Pick another.", undefined, { timeout: 5000 }),
        ).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("lets the save proceed when the availability check itself fails", async () => {
        // given
        stubBots([]);
        mocks.checkUsername.mockRejectedValue(new Error("network down"));
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Create Bot" }));
        await user.type(screen.getByLabelText("Username"), "beato");
        await user.type(screen.getByLabelText("Display Name"), "Beato");

        // when
        await user.click(screen.getByLabelText("Display Name"));

        // then
        expect(
            await screen.findByText(
                "Could not check that handle just now. Saving will still tell you if it is taken.",
                undefined,
                { timeout: 5000 },
            ),
        ).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    });

    it("does not offer to change the handle of an existing bot", async () => {
        // given
        stubBots([makeBot({ username: "bern", display_name: "Bernkastel" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Edit Bernkastel" }));

        // then
        expect(screen.getByLabelText("Username")).toBeDisabled();
        expect(screen.getByText("A bot's handle cannot be changed after it is created.")).toBeInTheDocument();
        expect(mocks.checkUsername).not.toHaveBeenCalled();
    });

    it("creates a bot from the values typed into the form", async () => {
        // given
        stubBots([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Create Bot" }));
        await user.type(screen.getByLabelText("Username"), "  beato  ");
        await user.type(screen.getByLabelText("Display Name"), "Beato");
        await user.type(screen.getByLabelText("Avatar URL"), "https://example.com/beato.png");
        await user.type(screen.getByLabelText("System Prompt"), "You are the Golden Witch.");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.create).toHaveBeenCalledWith({
            username: "beato",
            display_name: "Beato",
            avatar_url: "https://example.com/beato.png",
            system_prompt: "You are the Golden Witch.",
            base_prompt_id: null,
            model: "",
            reasoning_effort: "",
            verbosity: "",
            max_output_tokens: 0,
            enabled: true,
        });
    });

    it("carries the per-bot overrides into the new bot", async () => {
        // given
        stubBots([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Create Bot" }));
        await user.type(screen.getByLabelText("Username"), "beato");
        await user.type(screen.getByLabelText("Display Name"), "Beato");
        await user.type(screen.getByLabelText("Model"), "gpt-5");
        await user.selectOptions(screen.getByLabelText("Reasoning Effort"), "high");
        await user.type(screen.getByLabelText("Max Output Tokens"), "2048");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.create).toHaveBeenCalledWith(
            expect.objectContaining({ model: "gpt-5", reasoning_effort: "high", max_output_tokens: 2048 }),
        );
    });

    it("reports why a bot could not be saved", async () => {
        // given
        stubBots([]);
        mocks.create.mockRejectedValue(new Error("that username is already taken"));
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Create Bot" }));
        await user.type(screen.getByLabelText("Username"), "beato");
        await user.type(screen.getByLabelText("Display Name"), "Beato");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(await screen.findByText("Could not save the bot: that username is already taken")).toBeInTheDocument();
    });

    it("shows the save failure inside the form, not behind it", async () => {
        // given
        stubBots([]);
        mocks.create.mockRejectedValue(new Error("a chatbot needs a username, a display name and a system prompt"));
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Create Bot" }));
        await user.type(screen.getByLabelText("Username"), "beato");
        await user.type(screen.getByLabelText("Display Name"), "Beato");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        const alert = await screen.findByRole("alert");
        expect(alert).toHaveTextContent(/a chatbot needs a username/);
        expect(screen.getByRole("dialog")).toContainElement(alert);
    });

    it("clears a previous save failure when the form is reopened", async () => {
        // given
        stubBots([]);
        mocks.create.mockRejectedValue(new Error("boom"));
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Create Bot" }));
        await user.type(screen.getByLabelText("Username"), "beato");
        await user.type(screen.getByLabelText("Display Name"), "Beato");
        await user.click(screen.getByRole("button", { name: "Save" }));
        await screen.findByRole("alert");
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // when
        await user.click(screen.getByRole("button", { name: "Create Bot" }));

        // then
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });
});

describe("AdminChatbots editing", () => {
    it("loads a bot into the form for editing", async () => {
        // given
        stubBots([
            makeBot({
                username: "bern",
                display_name: "Bernkastel",
                avatar_url: "https://example.com/bern.png",
                system_prompt: "You are the Witch of Miracles.",
                model: "gpt-5",
                reasoning_effort: "high",
                max_output_tokens: 2048,
            }),
        ]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Edit Bernkastel" }));

        // then
        expect(screen.getByText("Edit Chatbot")).toBeInTheDocument();
        expect(screen.getByLabelText("Username")).toHaveValue("bern");
        expect(screen.getByLabelText("Display Name")).toHaveValue("Bernkastel");
        expect(screen.getByLabelText("Avatar URL")).toHaveValue("https://example.com/bern.png");
        expect(screen.getByLabelText("System Prompt")).toHaveValue("You are the Witch of Miracles.");
        expect(screen.getByLabelText("Model")).toHaveValue("gpt-5");
        expect(screen.getByLabelText("Reasoning Effort")).toHaveValue("high");
        expect(screen.getByLabelText("Max Output Tokens")).toHaveValue(2048);
    });

    it("leaves the token limit blank when the bot inherits it", async () => {
        // given
        stubBots([makeBot({ max_output_tokens: 0 })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Edit Beatrice" }));

        // then
        expect(screen.getByLabelText("Max Output Tokens")).toHaveValue(null);
    });

    it("tells the two inherit-the-default fields apart by their placeholder", async () => {
        // given
        stubBots([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Bot" }));

        // then
        expect(screen.getByLabelText("Model")).toHaveAttribute("placeholder", "Inherit the site default model");
        expect(screen.getByLabelText("Max Output Tokens")).toHaveAttribute(
            "placeholder",
            "Inherit the site default cap",
        );
    });

    it("saves an edit against the bot it came from and keeps it switched off", async () => {
        // given
        stubBots([makeBot({ id: "bot-9", enabled: false })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Edit Beatrice" }));
        await user.clear(screen.getByLabelText("Display Name"));
        await user.type(screen.getByLabelText("Display Name"), "Beato");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({
            id: "bot-9",
            data: {
                username: "beatrice",
                display_name: "Beato",
                avatar_url: "",
                system_prompt: "You are the Golden Witch.",
                base_prompt_id: null,
                model: "",
                reasoning_effort: "",
                verbosity: "",
                max_output_tokens: 0,
                enabled: false,
            },
        });
    });

    it("clears the form when an edit is abandoned", async () => {
        // given
        stubBots([makeBot()]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Edit Beatrice" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByText("Edit Chatbot")).not.toBeInTheDocument();
        expect(mocks.update).not.toHaveBeenCalled();
    });
});

describe("AdminChatbots model picker", () => {
    it("suggests every model the provider returned", async () => {
        // given
        stubBots([]);
        stubModels(["gpt-5.6-luna", "gpt-5.6-terra", "o5-mini"]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Bot" }));

        // then
        expect(modelOptions(screen.getByLabelText("Model"))).toEqual(["gpt-5.6-luna", "gpt-5.6-terra", "o5-mini"]);
    });

    it("saves a model that the provider never listed", async () => {
        // given
        stubBots([]);
        stubModels(["gpt-5.6-luna"]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Create Bot" }));
        await user.type(screen.getByLabelText("Username"), "beato");
        await user.type(screen.getByLabelText("Display Name"), "Beato");
        await user.type(screen.getByLabelText("Model"), "gpt-6-unreleased");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({ model: "gpt-6-unreleased" }));
    });

    it("suggests nothing while the provider list is empty", async () => {
        // given
        stubBots([]);
        stubModels([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Bot" }));

        // then
        expect(screen.getByLabelText("Model")).not.toHaveAttribute("list");
        expect(document.querySelector("datalist")).toBeNull();
    });
});

describe("AdminChatbots key gate", () => {
    it("locks the form until an API key is saved", async () => {
        // given
        stubBots([]);
        stubSettings({});
        stubModels([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Bot" }));

        // then
        expect(screen.getByText(/No OpenAI API key is saved yet/)).toBeInTheDocument();
        expect(screen.getByLabelText("Username")).toBeDisabled();
        expect(screen.getByLabelText("Display Name")).toBeDisabled();
        expect(screen.getByLabelText("System Prompt")).toBeDisabled();
        expect(screen.getByLabelText("Model")).toBeDisabled();
        expect(screen.getByLabelText("Reasoning Effort")).toBeDisabled();
        expect(screen.getByLabelText("Max Output Tokens")).toBeDisabled();
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
        expect(screen.queryByRole("button", { name: "Try again" })).not.toBeInTheDocument();
    });

    it("locks the edit form too, without hiding what is already there", async () => {
        // given
        stubBots([makeBot({ display_name: "Bernkastel", model: "gpt-5" })]);
        stubSettings({ chatbot_api_key: SAVED_KEY });
        stubModels([], false, vi.fn(), "OpenAI answered 401: Incorrect API key provided.");
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Edit Bernkastel" }));

        // then
        expect(screen.getByText(/OpenAI answered 401: Incorrect API key provided\./)).toBeInTheDocument();
        expect(screen.getByLabelText("Model")).toHaveValue("gpt-5");
        expect(screen.getByLabelText("Model")).toBeDisabled();
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("offers to fetch the model list again from the locked form", async () => {
        // given
        const refresh = vi.fn();
        stubBots([]);
        stubSettings({ chatbot_api_key: SAVED_KEY });
        stubModels([], false, refresh);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);
        await user.click(screen.getByRole("button", { name: "Create Bot" }));

        // when
        await user.click(screen.getByRole("button", { name: "Try again" }));

        // then
        expect(refresh).toHaveBeenCalledOnce();
    });

    it("unlocks the form once the key answers with a model list", async () => {
        // given
        stubBots([]);
        stubSettings({ chatbot_api_key: SAVED_KEY });
        stubModels(["gpt-5.6-luna"]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Bot" }));

        // then
        expect(screen.queryByText(/stays locked/)).not.toBeInTheDocument();
        expect(screen.getByLabelText("Username")).toBeEnabled();
        expect(screen.getByLabelText("Model")).toBeEnabled();
    });

    it("leaves the row controls usable so a bot can still be switched off", async () => {
        // given
        stubBots([makeBot({ id: "bot-4", display_name: "Beatrice", enabled: true })]);
        stubSettings({});
        stubModels([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("switch", { name: "Enabled Beatrice" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({
            id: "bot-4",
            data: expect.objectContaining({ enabled: false }),
        });
        expect(screen.getByRole("button", { name: "Delete Beatrice" })).toBeEnabled();
    });
});

describe("AdminChatbots deleting", () => {
    it("asks before deleting a bot", async () => {
        // given
        stubBots([makeBot()]);
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete Beatrice" }));

        // then
        expect(confirm).toHaveBeenCalledWith(
            "Delete Beatrice (@beatrice)? The bot account and its replies go with it.",
        );
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("deletes the bot once confirmed", async () => {
        // given
        stubBots([makeBot({ id: "bot-3" })]);
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete Beatrice" }));

        // then
        expect(mocks.remove).toHaveBeenCalledWith("bot-3");
    });

    it("says which bot could not be deleted", async () => {
        // given
        stubBots([makeBot({ display_name: "Beatrice" })]);
        vi.spyOn(window, "confirm").mockReturnValue(true);
        mocks.remove.mockRejectedValue(new Error("the bot still owns messages"));
        const user = userEvent.setup();
        renderWithProviders(<AdminChatbots />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete Beatrice" }));

        // then
        expect(await screen.findByText("Could not delete Beatrice: the bot still owns messages")).toBeInTheDocument();
    });
});

describe("AdminChatbots usage", () => {
    it("counts the replies and tokens spent over each range", () => {
        // given
        stubBots([]);
        stubUsage(makeUsage());

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        const panel = usagePanel("Last 7 days");
        expect(within(panel).getByText("1,234 replies")).toBeInTheDocument();
        expect(within(panel).getByText("56,000")).toBeInTheDocument();
        expect(within(panel).getByText("48,000")).toBeInTheDocument();
        expect(within(panel).getByText("7,800")).toBeInTheDocument();
        expect(within(panel).getByText("900")).toBeInTheDocument();
    });

    it("gives each range the figures fetched for that range", () => {
        // given
        stubBots([]);
        stubUsagePerRange({
            1: makeUsage({ invocations: 11, prompt_tokens: 100 }),
            7: makeUsage({ invocations: 222, prompt_tokens: 2000 }),
            30: makeUsage({ invocations: 3333, prompt_tokens: 30000 }),
        });

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(within(usagePanel("Last 24 hours")).getByText("11 replies")).toBeInTheDocument();
        expect(within(usagePanel("Last 24 hours")).getByText("100")).toBeInTheDocument();
        expect(within(usagePanel("Last 7 days")).getByText("222 replies")).toBeInTheDocument();
        expect(within(usagePanel("Last 7 days")).getByText("2,000")).toBeInTheDocument();
        expect(within(usagePanel("Last 30 days")).getByText("3,333 replies")).toBeInTheDocument();
        expect(within(usagePanel("Last 30 days")).getByText("30,000")).toBeInTheDocument();
    });

    it("waits on only the ranges that are still loading", () => {
        // given
        stubBots([]);
        stubUsagePerRange({ 1: makeUsage({ invocations: 11 }) });

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(within(usagePanel("Last 24 hours")).getByText("11 replies")).toBeInTheDocument();
        expect(within(usagePanel("Last 7 days")).getByText("Loading...")).toBeInTheDocument();
        expect(within(usagePanel("Last 30 days")).getByText("Loading...")).toBeInTheDocument();
    });

    it("hides the billed figure when no admin key can price the calls", () => {
        // given
        stubBots([]);
        stubUsage(makeUsage({ billed_usd: null }));

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(screen.queryByText(/^Billed/)).not.toBeInTheDocument();
    });

    it("shows the billed figure once the calls are priced", () => {
        // given
        stubBots([]);
        stubUsage(makeUsage({ billed_usd: 12.3 }));

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(within(usagePanel("Last 30 days")).getByText("Billed $12.30")).toBeInTheDocument();
    });

    it("counts the calls that failed and the ones a quota turned away", () => {
        // given
        stubBots([]);
        stubUsage(makeUsage({ failed: 17, quota: 4 }));

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        const panel = usagePanel("Last 7 days");
        expect(within(panel).getByText("Failed")).toBeInTheDocument();
        expect(within(panel).getByText("17")).toBeInTheDocument();
        expect(within(panel).getByText("Quota blocked")).toBeInTheDocument();
        expect(within(panel).getByText("4")).toBeInTheDocument();
    });

    it("marks the failure count once anything has failed", () => {
        // given
        stubBots([]);
        stubUsage(makeUsage({ failed: 17 }));

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(within(usagePanel("Last 7 days")).getByText("17")).toHaveClass(styles.error);
    });

    it("leaves the failure count unmarked while nothing has failed", () => {
        // given
        stubBots([]);
        stubUsage(makeUsage({ failed: 0, quota: 0 }));

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        const panel = usagePanel("Last 7 days");
        expect(within(panel).getAllByText("0")[0]).not.toHaveClass(styles.error);
        expect(within(panel).queryByText(FAILURE_NOTE)).not.toBeInTheDocument();
    });

    it("explains what a failure usually means once one shows up", () => {
        // given
        stubBots([]);
        stubUsage(makeUsage({ failed: 17 }));

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(within(usagePanel("Last 7 days")).getByText(FAILURE_NOTE)).toBeInTheDocument();
    });

    it("splits the replies and tokens by the channel they came from", () => {
        // given
        stubBots([]);
        stubUsage(
            makeUsage({
                channels: [
                    makeChannel({
                        channel: "group",
                        invocations: 180,
                        prompt_tokens: 700000,
                        completion_tokens: 12000,
                    }),
                    makeChannel({ channel: "dm", invocations: 62, prompt_tokens: 400000, completion_tokens: 10000 }),
                    makeChannel({ channel: "post", invocations: 18, prompt_tokens: 88000, completion_tokens: 2000 }),
                    makeChannel({
                        channel: "post_comment",
                        invocations: 5,
                        prompt_tokens: 21000,
                        completion_tokens: 1000,
                    }),
                ],
            }),
        );

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        const panel = usagePanel("Last 7 days");
        expect(within(panel).getByRole("row", { name: "Group chats 180 712,000" })).toBeInTheDocument();
        expect(within(panel).getByRole("row", { name: "DMs 62 410,000" })).toBeInTheDocument();
        expect(within(panel).getByRole("row", { name: "Posts 18 90,000" })).toBeInTheDocument();
        expect(within(panel).getByRole("row", { name: "Post comments 5 22,000" })).toBeInTheDocument();
    });

    it("lists the channels in the same order whatever order the server sent them", () => {
        // given
        stubBots([]);
        stubUsage(
            makeUsage({
                channels: [
                    makeChannel({ channel: "post_comment", invocations: 90 }),
                    makeChannel({ channel: "dm", invocations: 40 }),
                ],
            }),
        );

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        const rows = within(usagePanel("Last 7 days")).getAllByRole("row");
        expect(rows[1]).toHaveTextContent("Group chats");
        expect(rows[2]).toHaveTextContent("DMs");
        expect(rows[3]).toHaveTextContent("Posts");
        expect(rows[4]).toHaveTextContent("Post comments");
    });

    it("names a channel it has never heard of rather than dropping the row", () => {
        // given
        stubBots([]);
        stubUsage(makeUsage({ channels: [makeChannel({ channel: "carrier_pigeon", invocations: 3 })] }));

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        const panel = usagePanel("Last 7 days");
        expect(within(panel).getByRole("rowheader", { name: "carrier_pigeon" })).toBeInTheDocument();
        expect(within(panel).getAllByRole("row")).toHaveLength(6);
    });

    it("keeps every channel on the card at zero so the three ranges stay the same height", () => {
        // given
        stubBots([]);
        stubUsage(makeUsage({ invocations: 0, channels: [] }));

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        const panel = usagePanel("Last 7 days");
        expect(within(panel).getByRole("row", { name: "Group chats 0 0" })).toBeInTheDocument();
        expect(within(panel).getByRole("row", { name: "DMs 0 0" })).toBeInTheDocument();
        expect(within(panel).getByRole("row", { name: "Posts 0 0" })).toBeInTheDocument();
        expect(within(panel).getByRole("row", { name: "Post comments 0 0" })).toBeInTheDocument();
    });

    it("waits while the usage is being fetched", () => {
        // given
        stubBots([]);
        stubUsage(null, true);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(screen.getAllByText("Loading...")).toHaveLength(3);
    });
});
