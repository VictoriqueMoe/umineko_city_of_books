import { screen, within } from "@testing-library/react";
import userEvent, { type UserEvent } from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type {
    Chatbot,
    ChatbotBasePrompt,
    ChatbotChannelUsage,
    ChatbotPayload,
    ChatbotUsage,
    SiteSettings,
} from "../../types/api";
import { AdminChatbots } from "./AdminChatbots";
import styles from "./AdminChatbots.module.css";

type FieldEntry = [label: string, value: string];

const FAILURE_NOTE =
    "Failures are almost always a model id the provider does not recognise, a revoked or expired API key, or a quota that has run out.";

const KEY_SAVED: SiteSettings = { chatbot_api_key: "********" };

const NO_KEY: SiteSettings = {};

const IDENTITY: FieldEntry[] = [
    ["Username", "beato"],
    ["Display Name", "Beato"],
];

const LOCKED_FIELDS = ["Username", "Display Name", "System Prompt", "Model", "Reasoning Effort", "Max Output Tokens"];

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

vi.mock("../../hooks/queries/admin", () => ({
    useChatbots: mocks.useChatbots,
    useChatbotUsage: mocks.useChatbotUsage,
    useChatbotModels: mocks.useChatbotModels,
    useAdminSettings: mocks.useAdminSettings,
    useChatbotBasePrompts: mocks.useChatbotBasePrompts,
}));

vi.mock("../../hooks/mutations/admin", () => ({
    useCreateChatbot: () => ({ mutateAsync: mocks.create, isPending: false }),
    useUpdateChatbot: () => ({ mutateAsync: mocks.update, isPending: false }),
    useDeleteChatbot: () => ({ mutateAsync: mocks.remove, isPending: false }),
    useCreateChatbotBasePrompt: () => ({ mutateAsync: mocks.createBase, isPending: false }),
    useUpdateChatbotBasePrompt: () => ({ mutateAsync: mocks.updateBase, isPending: false }),
    useDeleteChatbotBasePrompt: () => ({ mutateAsync: mocks.removeBase, isPending: false }),
    useCheckUsernameAvailable: () => ({ mutateAsync: mocks.checkUsername, isPending: false }),
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

function stubBots(bots: Chatbot[], loading = false) {
    mocks.useChatbots.mockReturnValue({ bots, loading, refresh: vi.fn() });
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

function renderPage(): UserEvent {
    const user = userEvent.setup();
    renderWithProviders(<AdminChatbots />);

    return user;
}

async function renderAndClick(button: string): Promise<UserEvent> {
    const user = renderPage();
    await user.click(screen.getByRole("button", { name: button }));

    return user;
}

async function fillIn(user: UserEvent, typed: FieldEntry[], selected: FieldEntry[] = []): Promise<void> {
    for (const [label, value] of typed) {
        await user.type(screen.getByLabelText(label), value);
    }

    for (const [label, value] of selected) {
        await user.selectOptions(screen.getByLabelText(label), value);
    }
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
    stubBots([]);
    stubBasePrompts([]);
    stubUsage(null, true);
    stubModels(["gpt-5.6-luna"]);
    stubSettings(KEY_SAVED);
    mocks.checkUsername.mockImplementation((username: string) => Promise.resolve({ username, available: true }));
});

describe("AdminChatbots page", () => {
    it("waits while the bots are being fetched", () => {
        // given
        stubBots([], true);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(screen.getByText("Loading chatbots...")).toBeInTheDocument();
    });

    it("says so when no bot or base prompt exists yet and waits on every usage range", () => {
        // given
        stubBots([]);
        stubBasePrompts([]);
        stubUsage(null, true);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(screen.getByText("No chatbots yet.")).toBeInTheDocument();
        expect(screen.getByText("No base prompts yet.")).toBeInTheDocument();
        expect(screen.getAllByText("Loading...")).toHaveLength(3);
    });

    it("lists every bot and base prompt, naming each row control after the bot it acts on", () => {
        // given
        stubBots([
            makeBot({ display_name: "Beatrice", username: "beatrice", enabled: true }),
            makeBot({ id: "bot-2", display_name: "Bernkastel", username: "bern", enabled: false }),
        ]);
        stubBasePrompts([makeBasePrompt({ name: "game witch", bot_count: 3 })]);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("@beatrice")).toBeInTheDocument();
        expect(screen.getByText("Bernkastel")).toBeInTheDocument();
        expect(screen.getByText("@bern")).toBeInTheDocument();
        expect(screen.getByRole("switch", { name: "Enabled Beatrice" })).toHaveAttribute("aria-checked", "true");
        expect(screen.getByRole("switch", { name: "Enabled Bernkastel" })).toHaveAttribute("aria-checked", "false");
        expect(screen.getByRole("button", { name: "Edit Beatrice" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete Beatrice" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Edit Bernkastel" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete Bernkastel" })).toBeInTheDocument();
        expect(screen.getByText("game witch")).toBeInTheDocument();
        expect(screen.getByText(/3 bots/)).toBeInTheDocument();
    });
});

describe("AdminChatbots base prompts", () => {
    it("creates a base prompt from the name and text typed into the form", async () => {
        // given
        const user = await renderAndClick("Create Base Prompt");
        await fillIn(user, [
            ["Name", "game witch"],
            ["Prompt", "You are a witch of the game boards."],
        ]);

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.createBase).toHaveBeenCalledWith({
            name: "game witch",
            prompt: "You are a witch of the game boards.",
        });
    });

    it("saves an edit against the base prompt it came from", async () => {
        // given
        stubBasePrompts([makeBasePrompt({ id: "base-9", name: "game witch" })]);
        const user = await renderAndClick("Edit game witch");
        await user.clear(screen.getByLabelText("Name"));
        await user.type(screen.getByLabelText("Name"), "voyager");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.updateBase).toHaveBeenCalledWith({
            id: "base-9",
            data: { name: "voyager", prompt: "You are a witch of the game boards." },
        });
    });

    it("refuses to delete a base prompt that bots still extend", async () => {
        // given
        stubBasePrompts([makeBasePrompt({ name: "game witch", bot_count: 2 })]);
        const user = renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Delete game witch" }));

        // then
        expect(mocks.removeBase).not.toHaveBeenCalled();
        expect(screen.getByText(/still used by 2 bot/)).toBeInTheDocument();
    });
});

describe("AdminChatbots switching a bot on and off", () => {
    const toggleCases = [
        {
            name: "an enabled bot off, keeping its overrides",
            bots: [makeBot({ id: "bot-7", model: "gpt-5", reasoning_effort: "high", max_output_tokens: 2048 })],
            settings: KEY_SAVED,
            models: ["gpt-5.6-luna"],
            clicked: "Beatrice",
            sent: {
                id: "bot-7",
                data: expect.objectContaining({
                    username: "beatrice",
                    model: "gpt-5",
                    reasoning_effort: "high",
                    max_output_tokens: 2048,
                    enabled: false,
                }),
            },
        },
        {
            name: "the clicked bot and no other, turning a disabled one back on",
            bots: [
                makeBot({ id: "bot-1", display_name: "Beatrice", enabled: true }),
                makeBot({ id: "bot-2", display_name: "Bernkastel", username: "bern", enabled: false }),
            ],
            settings: KEY_SAVED,
            models: ["gpt-5.6-luna"],
            clicked: "Bernkastel",
            sent: { id: "bot-2", data: expect.objectContaining({ username: "bern", enabled: true }) },
        },
        {
            name: "a bot off while no API key is saved and the form is locked",
            bots: [makeBot({ id: "bot-4", display_name: "Beatrice", enabled: true })],
            settings: NO_KEY,
            models: [],
            clicked: "Beatrice",
            sent: { id: "bot-4", data: expect.objectContaining({ enabled: false }) },
        },
    ];

    it.each(toggleCases)(
        "switches $name without opening the form",
        async ({ bots, settings, models, clicked, sent }) => {
            // given
            stubBots(bots);
            stubSettings(settings);
            stubModels(models);
            const user = renderPage();

            // when
            await user.click(screen.getByRole("switch", { name: `Enabled ${clicked}` }));

            // then
            expect(mocks.update).toHaveBeenCalledExactlyOnceWith(sent);
            expect(screen.getByRole("button", { name: `Delete ${clicked}` })).toBeEnabled();
            expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
        },
    );

    it("says which bot could not be switched", async () => {
        // given
        stubBots([makeBot({ display_name: "Beatrice" })]);
        mocks.update.mockRejectedValue(new Error("the model is unreachable"));
        const user = renderPage();

        // when
        await user.click(screen.getByRole("switch", { name: "Enabled Beatrice" }));

        // then
        expect(await screen.findByText("Could not switch Beatrice off: the model is unreachable")).toBeInTheDocument();
    });
});

describe("AdminChatbots creating", () => {
    it("opens a blank, unlocked form with every field labelled and every provider model suggested", async () => {
        // given
        stubModels(["gpt-5.6-luna", "gpt-5.6-terra", "o5-mini"]);
        const user = renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Create Bot" }));

        // then
        const username = screen.getByLabelText("Username");
        const model = screen.getByLabelText("Model");
        const maxTokens = screen.getByLabelText("Max Output Tokens");
        expect(screen.queryByText(/stays locked/)).not.toBeInTheDocument();
        expect(username).toBeEnabled();
        expect(username).toHaveValue("");
        expect(username).toHaveAccessibleName("Username");
        expect(username).toHaveAccessibleDescription(
            "The handle members type to reach the bot. It has to be free, exactly like a human account.",
        );
        expect(screen.getByLabelText("Display Name")).toHaveValue("");
        expect(screen.getByLabelText("Avatar URL")).toHaveValue("");
        expect(screen.getByLabelText("System Prompt")).toHaveValue("");
        expect(screen.getByLabelText("Base Prompt")).toHaveValue("");
        expect(model).toBeEnabled();
        expect(model).toHaveValue("");
        expect(model).toHaveAttribute("placeholder", "Inherit the site default model");
        expect(modelOptions(model)).toEqual(["gpt-5.6-luna", "gpt-5.6-terra", "o5-mini"]);
        expect(screen.getByLabelText("Reasoning Effort")).toHaveValue("");
        expect(maxTokens).toHaveValue(null);
        expect(maxTokens).toHaveAttribute("placeholder", "Inherit the site default cap");
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    const usernameAllowedCases = [
        {
            name: "checks the trimmed handle when the username field loses focus and says it is free",
            answer: { username: "beato", available: true },
            hint: "@beato is free.",
        },
        {
            name: "lets the save proceed when the availability check itself fails",
            answer: new Error("network down"),
            hint: "Could not check that handle just now. Saving will still tell you if it is taken.",
        },
    ];

    it.each(usernameAllowedCases)("$name", async ({ answer, hint }) => {
        // given
        if (answer instanceof Error) {
            mocks.checkUsername.mockRejectedValue(answer);
        } else {
            mocks.checkUsername.mockResolvedValue(answer);
        }

        const user = await renderAndClick("Create Bot");
        await fillIn(user, [
            ["Display Name", "Beato"],
            ["Username", "  beato  "],
        ]);

        // when
        await user.tab();

        // then
        expect(mocks.checkUsername).toHaveBeenCalledWith("beato");
        expect(await screen.findByText(hint, undefined, { timeout: 5000 })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    });

    it("refuses to save a handle the server says is taken", async () => {
        // given
        mocks.checkUsername.mockResolvedValue({ username: "beato", available: false });
        const user = await renderAndClick("Create Bot");
        await fillIn(user, [
            ["Display Name", "Beato"],
            ["Username", "  beato  "],
        ]);

        // when
        await user.tab();

        // then
        expect(mocks.checkUsername).toHaveBeenCalledWith("beato");
        expect(
            await screen.findByText("@beato is already taken. Pick another.", undefined, { timeout: 5000 }),
        ).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    const createCases: {
        name: string;
        basePrompts: ChatbotBasePrompt[];
        typed: FieldEntry[];
        selected: FieldEntry[];
        payload: ChatbotPayload;
    }[] = [
        {
            name: "the trimmed identity typed into the form, leaving every override to inherit",
            basePrompts: [],
            typed: [
                ["Username", "  beato  "],
                ["Display Name", "Beato"],
                ["Avatar URL", "https://example.com/beato.png"],
                ["System Prompt", "You are the Golden Witch."],
            ],
            selected: [],
            payload: {
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
            },
        },
        {
            name: "a chosen base prompt and per-bot overrides, including a model the provider never listed",
            basePrompts: [makeBasePrompt({ id: "base-7", name: "game witch" })],
            typed: [
                ["Username", "beato"],
                ["Display Name", "Beato"],
                ["System Prompt", "You are the Golden Witch."],
                ["Model", "gpt-6-unreleased"],
                ["Max Output Tokens", "2048"],
            ],
            selected: [
                ["Base Prompt", "base-7"],
                ["Reasoning Effort", "high"],
            ],
            payload: {
                username: "beato",
                display_name: "Beato",
                avatar_url: "",
                system_prompt: "You are the Golden Witch.",
                base_prompt_id: "base-7",
                model: "gpt-6-unreleased",
                reasoning_effort: "high",
                verbosity: "",
                max_output_tokens: 2048,
                enabled: true,
            },
        },
    ];

    it.each(createCases)("creates a bot from $name", async ({ basePrompts, typed, selected, payload }) => {
        // given
        stubBasePrompts(basePrompts);
        const user = await renderAndClick("Create Bot");
        await fillIn(user, typed, selected);

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.create).toHaveBeenCalledWith(payload);
    });

    it("reports why a bot could not be saved inside the form, not behind it", async () => {
        // given
        mocks.create.mockRejectedValue(new Error("that username is already taken"));
        const user = await renderAndClick("Create Bot");
        await fillIn(user, IDENTITY);

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        const alert = await screen.findByRole("alert");
        expect(alert).toHaveTextContent(/^Could not save the bot: that username is already taken$/);
        expect(screen.getByRole("dialog")).toContainElement(alert);
    });

    it("clears a previous save failure when the form is reopened", async () => {
        // given
        mocks.create.mockRejectedValue(new Error("boom"));
        const user = await renderAndClick("Create Bot");
        await fillIn(user, IDENTITY);
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
    const editCases: { name: string; bot: Chatbot; values: [label: string, value: string | number | null][] }[] = [
        {
            name: "a bot with every override set",
            bot: makeBot({
                username: "bern",
                display_name: "Bernkastel",
                avatar_url: "https://example.com/bern.png",
                system_prompt: "You are the Witch of Miracles.",
                base_prompt_id: "base-7",
                model: "gpt-5",
                reasoning_effort: "high",
                max_output_tokens: 2048,
            }),
            values: [
                ["Username", "bern"],
                ["Display Name", "Bernkastel"],
                ["Avatar URL", "https://example.com/bern.png"],
                ["System Prompt", "You are the Witch of Miracles."],
                ["Base Prompt", "base-7"],
                ["Model", "gpt-5"],
                ["Reasoning Effort", "high"],
                ["Max Output Tokens", 2048],
            ],
        },
        {
            name: "a bot that inherits every override, leaving the token limit blank",
            bot: makeBot({ max_output_tokens: 0 }),
            values: [
                ["Username", "beatrice"],
                ["Display Name", "Beatrice"],
                ["Avatar URL", ""],
                ["System Prompt", "You are the Golden Witch."],
                ["Base Prompt", ""],
                ["Model", ""],
                ["Reasoning Effort", ""],
                ["Max Output Tokens", null],
            ],
        },
    ];

    it.each(editCases)("loads $name into the form without offering to change its handle", async ({ bot, values }) => {
        // given
        stubBots([bot]);
        stubBasePrompts([makeBasePrompt({ id: "base-7", name: "game witch" })]);
        const user = renderPage();

        // when
        await user.click(screen.getByRole("button", { name: `Edit ${bot.display_name}` }));

        // then
        expect(screen.getByText("Edit Chatbot")).toBeInTheDocument();
        for (const [label, value] of values) {
            expect(screen.getByLabelText(label)).toHaveValue(value);
        }

        expect(screen.getByLabelText("Username")).toBeDisabled();
        expect(screen.getByText("A bot's handle cannot be changed after it is created.")).toBeInTheDocument();
        expect(mocks.checkUsername).not.toHaveBeenCalled();
    });

    it("saves an edit against the bot it came from and keeps it switched off", async () => {
        // given
        stubBots([makeBot({ id: "bot-9", enabled: false })]);
        const user = await renderAndClick("Edit Beatrice");
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
        const user = await renderAndClick("Edit Beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByText("Edit Chatbot")).not.toBeInTheDocument();
        expect(mocks.update).not.toHaveBeenCalled();
    });
});

describe("AdminChatbots key gate", () => {
    const lockCases = [
        {
            name: "until an API key is saved",
            bots: [],
            settings: NO_KEY,
            modelsError: "",
            opener: "Create Bot",
            message: /No OpenAI API key is saved yet/,
            model: "",
            retryButtons: 0,
        },
        {
            name: "on an existing bot whose saved key the provider refuses, without hiding what is already there",
            bots: [makeBot({ display_name: "Bernkastel", model: "gpt-5" })],
            settings: KEY_SAVED,
            modelsError: "OpenAI answered 401: Incorrect API key provided.",
            opener: "Edit Bernkastel",
            message: /OpenAI answered 401: Incorrect API key provided\./,
            model: "gpt-5",
            retryButtons: 1,
        },
    ];

    it.each(lockCases)(
        "locks the form $name",
        async ({ bots, settings, modelsError, opener, message, model, retryButtons }) => {
            // given
            stubBots(bots);
            stubSettings(settings);
            stubModels([], false, vi.fn(), modelsError);
            const user = renderPage();

            // when
            await user.click(screen.getByRole("button", { name: opener }));

            // then
            expect(screen.getByText(message)).toBeInTheDocument();
            for (const label of LOCKED_FIELDS) {
                expect(screen.getByLabelText(label)).toBeDisabled();
            }

            expect(screen.getByLabelText("Model")).toHaveValue(model);
            expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
            expect(screen.queryAllByRole("button", { name: "Try again" })).toHaveLength(retryButtons);
        },
    );

    it("offers to fetch the model list again from the locked form, suggesting nothing meanwhile", async () => {
        // given
        const refresh = vi.fn();
        stubModels([], false, refresh);
        const user = await renderAndClick("Create Bot");

        // when
        await user.click(screen.getByRole("button", { name: "Try again" }));

        // then
        expect(refresh).toHaveBeenCalledOnce();
        expect(screen.getByLabelText("Model")).not.toHaveAttribute("list");
        expect(document.querySelector("datalist")).toBeNull();
    });
});

describe("AdminChatbots deleting", () => {
    const deleteCases = [
        {
            name: "a bot",
            bots: [makeBot({ id: "bot-3" })],
            basePrompts: [],
            rowButton: "Delete Beatrice",
            question: "Delete Beatrice (@beatrice)? The bot account and its replies go with it.",
            remove: mocks.remove,
            id: "bot-3",
        },
        {
            name: "an unused base prompt",
            bots: [],
            basePrompts: [makeBasePrompt({ id: "base-4", name: "game witch", bot_count: 0 })],
            rowButton: "Delete game witch",
            question: 'Delete the base prompt "game witch"?',
            remove: mocks.removeBase,
            id: "base-4",
        },
    ];

    it.each(deleteCases)("asks before deleting $name", async ({ bots, basePrompts, rowButton, question, remove }) => {
        // given
        stubBots(bots);
        stubBasePrompts(basePrompts);
        const user = renderPage();

        // when
        await user.click(screen.getByRole("button", { name: rowButton }));

        // then
        expect(screen.getByText(question)).toBeInTheDocument();
        expect(remove).not.toHaveBeenCalled();
    });

    it.each(deleteCases)("deletes $name once confirmed", async ({ bots, basePrompts, rowButton, remove, id }) => {
        // given
        stubBots(bots);
        stubBasePrompts(basePrompts);
        const user = await renderAndClick(rowButton);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(remove).toHaveBeenCalledWith(id);
    });

    it.each(deleteCases)(
        "leaves $name alone when the delete is cancelled",
        async ({ bots, basePrompts, rowButton, remove }) => {
            // given
            stubBots(bots);
            stubBasePrompts(basePrompts);
            const user = await renderAndClick(rowButton);

            // when
            await user.click(screen.getByRole("button", { name: "Cancel" }));

            // then
            expect(remove).not.toHaveBeenCalled();
            expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
        },
    );

    it("says which bot could not be deleted", async () => {
        // given
        stubBots([makeBot({ display_name: "Beatrice" })]);
        mocks.remove.mockRejectedValue(new Error("the bot still owns messages"));
        const user = await renderAndClick("Delete Beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(await screen.findByText("Could not delete Beatrice: the bot still owns messages")).toBeInTheDocument();
    });
});

describe("AdminChatbots usage", () => {
    it("counts the replies, tokens, failures and quota blocks over a range, flagging and explaining the failures", () => {
        // given
        stubUsage(makeUsage({ failed: 17, quota: 4, billed_usd: null }));

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        const panel = usagePanel("Last 7 days");
        expect(within(panel).getByText("1,234 replies")).toBeInTheDocument();
        expect(within(panel).getByText("56,000")).toBeInTheDocument();
        expect(within(panel).getByText("48,000")).toBeInTheDocument();
        expect(within(panel).getByText("7,800")).toBeInTheDocument();
        expect(within(panel).getByText("900")).toBeInTheDocument();
        expect(within(panel).getByText("Failed")).toBeInTheDocument();
        expect(within(panel).getByText("17")).toHaveClass(styles.error);
        expect(within(panel).getByText("Quota blocked")).toBeInTheDocument();
        expect(within(panel).getByText("4")).toBeInTheDocument();
        expect(within(panel).getByText(FAILURE_NOTE)).toBeInTheDocument();
        expect(screen.queryByText(/^Billed/)).not.toBeInTheDocument();
    });

    it("leaves the failure count unmarked while nothing has failed", () => {
        // given
        stubUsage(makeUsage({ failed: 0, quota: 0 }));

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        const panel = usagePanel("Last 7 days");
        expect(within(panel).getAllByText("0")[0]).not.toHaveClass(styles.error);
        expect(within(panel).queryByText(FAILURE_NOTE)).not.toBeInTheDocument();
    });

    const rangeCases: { name: string; byDays: Record<number, ChatbotUsage>; shown: Record<string, string[]> }[] = [
        {
            name: "gives each range the figures fetched for that range, billed once the calls are priced",
            byDays: {
                1: makeUsage({ invocations: 11, prompt_tokens: 100 }),
                7: makeUsage({ invocations: 222, prompt_tokens: 2000 }),
                30: makeUsage({ invocations: 3333, prompt_tokens: 30000, billed_usd: 12.3 }),
            },
            shown: {
                "Last 24 hours": ["11 replies", "100"],
                "Last 7 days": ["222 replies", "2,000"],
                "Last 30 days": ["3,333 replies", "30,000", "Billed $12.30"],
            },
        },
        {
            name: "waits on only the ranges that are still loading",
            byDays: { 1: makeUsage({ invocations: 11 }) },
            shown: {
                "Last 24 hours": ["11 replies"],
                "Last 7 days": ["Loading..."],
                "Last 30 days": ["Loading..."],
            },
        },
    ];

    it.each(rangeCases)("$name", ({ byDays, shown }) => {
        // given
        stubUsagePerRange(byDays);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        for (const [label, texts] of Object.entries(shown)) {
            for (const text of texts) {
                expect(within(usagePanel(label)).getByText(text)).toBeInTheDocument();
            }
        }
    });

    const channelCases: { name: string; usage: ChatbotUsage; rows: [string, string, string][] }[] = [
        {
            name: "every known channel sent out of order, then one it has never heard of rather than dropping it",
            usage: makeUsage({
                channels: [
                    makeChannel({ channel: "carrier_pigeon", invocations: 3 }),
                    makeChannel({
                        channel: "post_comment",
                        invocations: 5,
                        prompt_tokens: 21000,
                        completion_tokens: 1000,
                    }),
                    makeChannel({ channel: "post", invocations: 18, prompt_tokens: 88000, completion_tokens: 2000 }),
                    makeChannel({ channel: "dm", invocations: 62, prompt_tokens: 400000, completion_tokens: 10000 }),
                    makeChannel({
                        channel: "group",
                        invocations: 180,
                        prompt_tokens: 700000,
                        completion_tokens: 12000,
                    }),
                ],
            }),
            rows: [
                ["Group chats", "180", "712,000"],
                ["DMs", "62", "410,000"],
                ["Posts", "18", "90,000"],
                ["Post comments", "5", "22,000"],
                ["carrier_pigeon", "3", "1,200"],
            ],
        },
        {
            name: "only some channels, sent in reverse order",
            usage: makeUsage({
                channels: [
                    makeChannel({ channel: "post_comment", invocations: 90 }),
                    makeChannel({ channel: "dm", invocations: 40 }),
                ],
            }),
            rows: [
                ["Group chats", "0", "0"],
                ["DMs", "40", "1,200"],
                ["Posts", "0", "0"],
                ["Post comments", "90", "1,200"],
            ],
        },
        {
            name: "an idle range, keeping every channel on the card at zero so the three ranges stay the same height",
            usage: makeUsage({ invocations: 0, channels: [] }),
            rows: [
                ["Group chats", "0", "0"],
                ["DMs", "0", "0"],
                ["Posts", "0", "0"],
                ["Post comments", "0", "0"],
            ],
        },
    ];

    it.each(channelCases)("splits the replies and tokens by channel in a fixed order for $name", ({ usage, rows }) => {
        // given
        stubUsage(usage);

        // when
        renderWithProviders(<AdminChatbots />);

        // then
        const bodyRows = within(usagePanel("Last 7 days")).getAllByRole("row").slice(1);
        expect(bodyRows).toHaveLength(rows.length);
        for (const [index, [channel, replies, tokens]] of rows.entries()) {
            expect(within(bodyRows[index]).getByRole("rowheader", { name: channel })).toBeInTheDocument();
            expect(bodyRows[index]).toHaveAccessibleName(`${channel} ${replies} ${tokens}`);
        }
    });
});
