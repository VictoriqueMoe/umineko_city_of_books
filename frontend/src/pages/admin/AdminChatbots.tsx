import { useId, useState } from "react";
import {
    useAdminSettings,
    useChatbotBasePrompts,
    useChatbotModels,
    useChatbotUsage,
    useChatbots,
} from "../../api/queries/admin";
import {
    useCreateChatbot,
    useCreateChatbotBasePrompt,
    useDeleteChatbot,
    useDeleteChatbotBasePrompt,
    useUpdateChatbot,
    useUpdateChatbotBasePrompt,
} from "../../api/mutations/admin";
import { checkUsernameAvailable } from "../../api/endpoints";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import { Select } from "../../components/Select/Select";
import { TextArea } from "../../components/TextArea/TextArea";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import type { Chatbot, ChatbotBasePrompt, ChatbotChannelUsage, ChatbotPayload, ChatbotUsage } from "../../types/api";
import { ChatbotKeyGate } from "./ChatbotKeyGate";
import styles from "./AdminChatbots.module.css";

interface ChatbotFields {
    username: string;
    displayName: string;
    avatarURL: string;
    prompt: string;
    basePromptID: string;
    model: string;
    effort: string;
    verbosity: string;
    maxTokens: string;
}

type UsernameCheck =
    | { state: "idle" }
    | { state: "checking" }
    | { state: "available"; username: string }
    | { state: "taken"; username: string }
    | { state: "failed" };

const CHARS_PER_TOKEN = 4;
const CACHING_TOKEN_THRESHOLD = 1000;
const CHANNEL_LABELS: Record<string, string> = {
    group: "Group chats",
    dm: "DMs",
    post: "Posts",
    post_comment: "Post comments",
};
const CHANNEL_ORDER = Object.keys(CHANNEL_LABELS);
const EMPTY_FIELDS: ChatbotFields = {
    username: "",
    displayName: "",
    avatarURL: "",
    prompt: "",
    basePromptID: "",
    model: "",
    effort: "",
    verbosity: "",
    maxTokens: "",
};

function fieldsFromBot(bot: Chatbot): ChatbotFields {
    return {
        username: bot.username,
        displayName: bot.display_name,
        avatarURL: bot.avatar_url,
        prompt: bot.system_prompt,
        basePromptID: bot.base_prompt_id ?? "",
        model: bot.model,
        effort: bot.reasoning_effort,
        verbosity: bot.verbosity,
        maxTokens: bot.max_output_tokens > 0 ? String(bot.max_output_tokens) : "",
    };
}

function buildPayload(fields: ChatbotFields, enabled: boolean): ChatbotPayload {
    const maxTokens = parseInt(fields.maxTokens, 10);

    return {
        username: fields.username.trim(),
        display_name: fields.displayName.trim(),
        avatar_url: fields.avatarURL.trim(),
        system_prompt: fields.prompt,
        base_prompt_id: fields.basePromptID === "" ? null : fields.basePromptID,
        model: fields.model.trim(),
        reasoning_effort: fields.effort,
        verbosity: fields.verbosity,
        max_output_tokens: isNaN(maxTokens) ? 0 : maxTokens,
        enabled,
    };
}

function channelLabel(channel: string): string {
    return CHANNEL_LABELS[channel] ?? channel;
}

function channelTokens(channel: ChatbotChannelUsage): number {
    return channel.prompt_tokens + channel.completion_tokens;
}

function emptyChannel(channel: string): ChatbotChannelUsage {
    return {
        channel,
        invocations: 0,
        prompt_tokens: 0,
        cached_prompt_tokens: 0,
        cache_write_tokens: 0,
        completion_tokens: 0,
        reasoning_tokens: 0,
    };
}

function channelRows(channels: ChatbotChannelUsage[]): ChatbotChannelUsage[] {
    const returned = new Map<string, ChatbotChannelUsage>();
    for (const channel of channels) {
        returned.set(channel.channel, channel);
    }

    const rows: ChatbotChannelUsage[] = [];
    for (const name of CHANNEL_ORDER) {
        rows.push(returned.get(name) ?? emptyChannel(name));
        returned.delete(name);
    }

    for (const channel of returned.values()) {
        rows.push(channel);
    }

    return rows;
}

function failureDetail(e: unknown): string {
    return e instanceof Error ? e.message : "unknown error";
}

function usernameHint(check: UsernameCheck, editing: boolean): string {
    if (editing) {
        return "A bot's handle cannot be changed after it is created.";
    }

    switch (check.state) {
        case "checking":
            return "Checking whether that handle is free...";
        case "available":
            return `@${check.username} is free.`;
        case "taken":
            return `@${check.username} is already taken. Pick another.`;
        case "failed":
            return "Could not check that handle just now. Saving will still tell you if it is taken.";
        default:
            return "The handle members type to reach the bot. It has to be free, exactly like a human account.";
    }
}

export function AdminChatbots() {
    usePageTitle("Admin - Chatbots");
    const baseID = useId();
    const { bots, loading } = useChatbots();
    const { basePrompts } = useChatbotBasePrompts();
    const { settings } = useAdminSettings();
    const { models, modelsError, loading: modelsLoading, refresh: refreshModels } = useChatbotModels();
    const dayUsage = useChatbotUsage(1);
    const weekUsage = useChatbotUsage(7);
    const monthUsage = useChatbotUsage(30);
    const createBotMutation = useCreateChatbot();
    const updateBotMutation = useUpdateChatbot();
    const deleteBotMutation = useDeleteChatbot();
    const [error, setError] = useState("");
    const [formError, setFormError] = useState("");

    const [editingBot, setEditingBot] = useState<Chatbot | null>(null);
    const [showForm, setShowForm] = useState(false);
    const [fields, setFields] = useState<ChatbotFields>(EMPTY_FIELDS);
    const [togglingID, setTogglingID] = useState<string | null>(null);
    const [usernameCheck, setUsernameCheck] = useState<UsernameCheck>({ state: "idle" });

    const createBaseMutation = useCreateChatbotBasePrompt();
    const updateBaseMutation = useUpdateChatbotBasePrompt();
    const deleteBaseMutation = useDeleteChatbotBasePrompt();
    const [showBaseForm, setShowBaseForm] = useState(false);
    const [editingBase, setEditingBase] = useState<ChatbotBasePrompt | null>(null);
    const [baseName, setBaseName] = useState("");
    const [basePrompt, setBasePrompt] = useState("");
    const [baseError, setBaseError] = useState("");

    const savingBase = createBaseMutation.isPending || updateBaseMutation.isPending;

    const saving = createBotMutation.isPending || updateBotMutation.isPending;
    const promptTokenEstimate = Math.ceil(fields.prompt.length / CHARS_PER_TOKEN);

    const apiKeySaved = (settings?.chatbot_api_key ?? "").trim() !== "";
    const formLocked = !apiKeySaved || models.length === 0;
    const usernameKnownTaken = usernameCheck.state === "taken" && usernameCheck.username === fields.username.trim();

    const usageRanges = [
        { label: "Last 24 hours", usage: dayUsage.usage, loading: dayUsage.loading },
        { label: "Last 7 days", usage: weekUsage.usage, loading: weekUsage.loading },
        { label: "Last 30 days", usage: monthUsage.usage, loading: monthUsage.loading },
    ];

    function fieldID(name: string) {
        return `${baseID}-${name}`;
    }

    function setField<K extends keyof ChatbotFields>(key: K, value: ChatbotFields[K]) {
        setFields(current => ({ ...current, [key]: value }));
    }

    function openCreateBase() {
        setEditingBase(null);
        setBaseName("");
        setBasePrompt("");
        setBaseError("");
        setShowBaseForm(true);
    }

    function openEditBase(base: ChatbotBasePrompt) {
        setEditingBase(base);
        setBaseName(base.name);
        setBasePrompt(base.prompt);
        setBaseError("");
        setShowBaseForm(true);
    }

    function closeBaseForm() {
        setShowBaseForm(false);
        setEditingBase(null);
        setBaseError("");
    }

    async function submitBase() {
        if (baseName.trim() === "" || basePrompt.trim() === "") {
            setBaseError("A base prompt needs a name and some text.");

            return;
        }

        const payload = { name: baseName.trim(), prompt: basePrompt };

        try {
            if (editingBase) {
                await updateBaseMutation.mutateAsync({ id: editingBase.id, data: payload });
            } else {
                await createBaseMutation.mutateAsync(payload);
            }

            closeBaseForm();
        } catch (e) {
            setBaseError(failureDetail(e));
        }
    }

    async function removeBase(base: ChatbotBasePrompt) {
        if (base.bot_count > 0) {
            setBaseError(`${base.name} is still used by ${base.bot_count} bot(s). Unassign them first.`);

            return;
        }

        if (!confirm(`Delete the base prompt "${base.name}"?`)) {
            return;
        }

        try {
            await deleteBaseMutation.mutateAsync(base.id);
            setBaseError("");
        } catch (e) {
            setBaseError(failureDetail(e));
        }
    }

    function openCreate() {
        setEditingBot(null);
        setFields(EMPTY_FIELDS);
        setUsernameCheck({ state: "idle" });
        setFormError("");
        setShowForm(true);
    }

    function openEdit(bot: Chatbot) {
        setEditingBot(bot);
        setFields(fieldsFromBot(bot));
        setUsernameCheck({ state: "idle" });
        setFormError("");
        setShowForm(true);
    }

    async function handleUsernameBlur() {
        if (editingBot) {
            return;
        }

        const username = fields.username.trim();
        if (!username) {
            setUsernameCheck({ state: "idle" });

            return;
        }

        setUsernameCheck({ state: "checking" });
        try {
            const result = await checkUsernameAvailable(username);

            setUsernameCheck({ state: result.available ? "available" : "taken", username: result.username });
        } catch {
            setUsernameCheck({ state: "failed" });
        }
    }

    function closeForm() {
        setShowForm(false);
        setEditingBot(null);
    }

    async function handleSave() {
        setFormError("");
        try {
            if (editingBot) {
                await updateBotMutation.mutateAsync({
                    id: editingBot.id,
                    data: buildPayload(fields, editingBot.enabled),
                });
            } else {
                await createBotMutation.mutateAsync(buildPayload(fields, true));
            }
            closeForm();
        } catch (e) {
            setFormError(`Could not save the bot: ${failureDetail(e)}`);
        }
    }

    async function handleToggleEnabled(bot: Chatbot, enabled: boolean) {
        setError("");
        setTogglingID(bot.id);
        try {
            await updateBotMutation.mutateAsync({ id: bot.id, data: buildPayload(fieldsFromBot(bot), enabled) });
        } catch (e) {
            const direction = enabled ? "on" : "off";

            setError(`Could not switch ${bot.display_name} ${direction}: ${failureDetail(e)}`);
        } finally {
            setTogglingID(null);
        }
    }

    async function handleDelete(bot: Chatbot) {
        if (
            !window.confirm(
                `Delete ${bot.display_name} (@${bot.username})? The bot account and its replies go with it.`,
            )
        ) {
            return;
        }

        setError("");
        try {
            await deleteBotMutation.mutateAsync(bot.id);
        } catch (e) {
            setError(`Could not delete ${bot.display_name}: ${failureDetail(e)}`);
        }
    }

    if (loading) {
        return <div className={styles.loading}>Loading chatbots...</div>;
    }

    return (
        <div className={styles.page}>
            <div className={styles.header}>
                <h1 className={styles.title}>Chatbots</h1>
                <Button variant="primary" onClick={openCreate}>
                    Create Bot
                </Button>
            </div>

            <p className={styles.intro}>
                Each bot is a real account members can mention or reply to in chat. Its personality lives entirely in
                the system prompt you write below, and anything left blank falls back to the site defaults on the
                Settings page.
            </p>

            {error && <div className={styles.error}>{error}</div>}

            <div className={styles.header}>
                <h2 className={styles.overridesTitle}>Base Prompts</h2>
                <Button variant="secondary" onClick={openCreateBase}>
                    Create Base Prompt
                </Button>
            </div>

            <p className={styles.intro}>
                A base prompt is shared text that every bot extending it receives before its own personality, for the
                rules they all obey rather than anything one character does. Edit it once and every bot using it picks
                the change up immediately.
            </p>

            {baseError && <div className={styles.error}>{baseError}</div>}

            {basePrompts.length === 0 ? (
                <div className={styles.empty}>No base prompts yet.</div>
            ) : (
                <div className={styles.list}>
                    {basePrompts.map(base => (
                        <div key={base.id} className={styles.botRow}>
                            <span className={styles.botIdentity}>
                                <span className={styles.botName}>{base.name}</span>
                                <span className={styles.botMeta}>
                                    {base.bot_count === 1 ? "1 bot" : `${base.bot_count} bots`} ·{" "}
                                    {base.prompt.length.toLocaleString()} characters
                                </span>
                            </span>
                            <span className={styles.actions}>
                                <Button
                                    variant="secondary"
                                    size="small"
                                    aria-label={`Edit ${base.name}`}
                                    onClick={() => openEditBase(base)}
                                >
                                    Edit
                                </Button>
                                <Button
                                    variant="danger"
                                    size="small"
                                    aria-label={`Delete ${base.name}`}
                                    onClick={() => {
                                        removeBase(base).catch(() => undefined);
                                    }}
                                >
                                    Delete
                                </Button>
                            </span>
                        </div>
                    ))}
                </div>
            )}

            <div className={styles.usageRow}>
                {usageRanges.map(range => (
                    <UsagePanel key={range.label} label={range.label} usage={range.usage} loading={range.loading} />
                ))}
            </div>

            {bots.length === 0 ? (
                <div className={styles.empty}>No chatbots yet.</div>
            ) : (
                <div className={styles.list}>
                    {bots.map(bot => (
                        <div key={bot.id} className={styles.botRow}>
                            <span className={styles.avatarWrapper}>
                                {bot.avatar_url ? (
                                    <img
                                        className={styles.avatar}
                                        src={bot.avatar_url}
                                        alt=""
                                        width={40}
                                        height={40}
                                        decoding="async"
                                        loading="lazy"
                                    />
                                ) : (
                                    <span className={styles.avatarPlaceholder}>
                                        {bot.display_name.charAt(0) || "?"}
                                    </span>
                                )}
                            </span>
                            <div className={styles.botIdentity}>
                                <span className={styles.botName}>{bot.display_name}</span>
                                <span className={styles.botHandle}>@{bot.username}</span>
                                <span className={styles.botMeta}>
                                    {bot.model || "default model"} &middot;{" "}
                                    {bot.reasoning_effort || "default reasoning"} &middot;{" "}
                                    {bot.max_output_tokens > 0 ? `${bot.max_output_tokens} tokens` : "default tokens"}
                                </span>
                            </div>
                            <div className={styles.botToggle}>
                                <BotEnabledToggle bot={bot} onChange={handleToggleEnabled} />
                                {togglingID === bot.id && <span className={styles.botMeta}>Saving...</span>}
                            </div>
                            <div className={styles.actions}>
                                <Button
                                    variant="secondary"
                                    size="small"
                                    aria-label={`Edit ${bot.display_name}`}
                                    onClick={() => openEdit(bot)}
                                >
                                    Edit
                                </Button>
                                <Button
                                    variant="danger"
                                    size="small"
                                    aria-label={`Delete ${bot.display_name}`}
                                    onClick={() => handleDelete(bot)}
                                >
                                    Delete
                                </Button>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            <Modal isOpen={showForm} onClose={closeForm} title={editingBot ? "Edit Chatbot" : "Create Chatbot"}>
                <div className={styles.form}>
                    {formLocked && (
                        <ChatbotKeyGate
                            apiKeySaved={apiKeySaved}
                            checking={modelsLoading}
                            reason={modelsError}
                            onRetry={refreshModels}
                        />
                    )}
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("username")}>Username</label>
                        <Input
                            id={fieldID("username")}
                            type="text"
                            value={fields.username}
                            onChange={e => {
                                setField("username", e.target.value);
                                setUsernameCheck({ state: "idle" });
                            }}
                            onBlur={handleUsernameBlur}
                            placeholder="beatrice"
                            aria-describedby={fieldID("username-hint")}
                            fullWidth
                            disabled={formLocked || editingBot !== null}
                        />
                        <span id={fieldID("username-hint")} className={styles.fieldHint} aria-live="polite">
                            {usernameHint(usernameCheck, editingBot !== null)}
                        </span>
                    </div>
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("display-name")}>Display Name</label>
                        <Input
                            id={fieldID("display-name")}
                            type="text"
                            value={fields.displayName}
                            onChange={e => setField("displayName", e.target.value)}
                            placeholder="Beatrice"
                            aria-describedby={fieldID("display-name-hint")}
                            fullWidth
                            disabled={formLocked}
                        />
                        <span id={fieldID("display-name-hint")} className={styles.fieldHint}>
                            The name shown on every message the bot posts.
                        </span>
                    </div>
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("avatar-url")}>Avatar URL</label>
                        <Input
                            id={fieldID("avatar-url")}
                            type="text"
                            value={fields.avatarURL}
                            onChange={e => setField("avatarURL", e.target.value)}
                            placeholder="https://example.com/avatar.png"
                            aria-describedby={fieldID("avatar-url-hint")}
                            fullWidth
                            disabled={formLocked}
                        />
                        <span id={fieldID("avatar-url-hint")} className={styles.fieldHint}>
                            Leave it empty and the bot falls back to an initial, the same as any member without a
                            picture.
                        </span>
                    </div>
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("base-prompt")}>Base Prompt</label>
                        <Select
                            id={fieldID("base-prompt")}
                            value={fields.basePromptID}
                            onChange={e => setField("basePromptID", e.target.value)}
                            aria-describedby={fieldID("base-prompt-hint")}
                            disabled={formLocked}
                        >
                            <option value="">None</option>
                            {basePrompts.map(base => (
                                <option key={base.id} value={base.id}>
                                    {base.name}
                                </option>
                            ))}
                        </Select>
                        <span id={fieldID("base-prompt-hint")} className={styles.fieldHint}>
                            Shared text that every bot extending it receives before its own personality, for the rules
                            they all obey rather than anything one character does. Edit it once and every bot using it
                            picks the change up immediately. Because it sits in front of the persona it is identical
                            across those bots, which is exactly what prompt caching keys on.
                        </span>
                    </div>
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("system-prompt")}>System Prompt</label>
                        <TextArea
                            id={fieldID("system-prompt")}
                            rows={14}
                            value={fields.prompt}
                            onChange={e => setField("prompt", e.target.value)}
                            placeholder="You are Beatrice, the Golden Witch of Rokkenjima. You speak with..."
                            aria-describedby={`${fieldID("system-prompt-hint")} ${fieldID("system-prompt-caching")} ${fieldID("system-prompt-count")}`}
                            disabled={formLocked}
                        />
                        <span id={fieldID("system-prompt-hint")} className={styles.fieldHint}>
                            This is the entire personality. There is no fine-tuning and no training step behind it, so
                            whatever the bot knows about who it is, how it talks, what it refuses to discuss and how
                            long its answers run has to be written here. Be generous and specific: give it voice,
                            history, quirks, relationships and worked examples of good replies.
                        </span>
                        <span id={fieldID("system-prompt-caching")} className={styles.fieldHint}>
                            Past roughly {CACHING_TOKEN_THRESHOLD} tokens the prompt gets noticeably better in
                            character, and it also becomes eligible for prompt caching, which makes the input on every
                            repeat call about ten times cheaper. A long, stable persona is therefore both better and
                            cheaper than a short one.
                        </span>
                        <span
                            id={fieldID("system-prompt-count")}
                            className={
                                promptTokenEstimate >= CACHING_TOKEN_THRESHOLD
                                    ? `${styles.tokenCount} ${styles.tokenCountGood}`
                                    : styles.tokenCount
                            }
                        >
                            About {promptTokenEstimate} tokens
                            {promptTokenEstimate >= CACHING_TOKEN_THRESHOLD
                                ? " - long enough for prompt caching"
                                : ` - ${CACHING_TOKEN_THRESHOLD - promptTokenEstimate} more to reach the caching threshold`}
                        </span>
                    </div>

                    <div className={styles.overridesTitle}>Per-bot overrides</div>
                    <span id={fieldID("overrides-hint")} className={styles.fieldHint}>
                        Leave these blank to inherit whatever the Settings page has configured. Only set them when this
                        one bot genuinely needs to differ.
                    </span>
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("model")}>Model</label>
                        <Input
                            id={fieldID("model")}
                            type="text"
                            value={fields.model}
                            onChange={e => setField("model", e.target.value)}
                            placeholder="Inherit the site default model"
                            aria-describedby={fieldID("overrides-hint")}
                            list={models.length > 0 ? fieldID("model-options") : undefined}
                            fullWidth
                            disabled={formLocked}
                        />
                        {models.length > 0 && (
                            <datalist id={fieldID("model-options")}>
                                {models.map(model => (
                                    <option key={model} value={model} />
                                ))}
                            </datalist>
                        )}
                    </div>
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("reasoning-effort")}>Reasoning Effort</label>
                        <Select
                            id={fieldID("reasoning-effort")}
                            value={fields.effort}
                            onChange={e => setField("effort", e.target.value)}
                            aria-describedby={fieldID("overrides-hint")}
                            disabled={formLocked}
                        >
                            <option value="">Inherit the site default</option>
                            <option value="none">None</option>
                            <option value="low">Low</option>
                            <option value="medium">Medium</option>
                            <option value="high">High</option>
                            <option value="xhigh">Extra high</option>
                            <option value="max">Max</option>
                        </Select>
                    </div>
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("verbosity")}>Verbosity</label>
                        <Select
                            id={fieldID("verbosity")}
                            value={fields.verbosity}
                            onChange={e => setField("verbosity", e.target.value)}
                            aria-describedby={fieldID("overrides-hint")}
                            disabled={formLocked}
                        >
                            <option value="">Inherit the site default</option>
                            <option value="low">Low</option>
                            <option value="medium">Medium</option>
                            <option value="high">High</option>
                        </Select>
                    </div>
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("max-output-tokens")}>Max Output Tokens</label>
                        <Input
                            id={fieldID("max-output-tokens")}
                            type="number"
                            value={fields.maxTokens}
                            onChange={e => setField("maxTokens", e.target.value)}
                            placeholder="Inherit the site default cap"
                            aria-describedby={`${fieldID("overrides-hint")} ${fieldID("max-output-tokens-hint")}`}
                            fullWidth
                            disabled={formLocked}
                        />
                        <span id={fieldID("max-output-tokens-hint")} className={styles.fieldHint}>
                            Caps reasoning and visible text together for this bot. It is a safety limit, not a way to
                            ask for shorter replies.
                        </span>
                    </div>

                    {formError && (
                        <div className={styles.error} role="alert">
                            {formError}
                        </div>
                    )}

                    <div className={styles.formActions}>
                        <Button variant="ghost" size="small" onClick={closeForm}>
                            Cancel
                        </Button>
                        <Button
                            variant="primary"
                            size="small"
                            onClick={handleSave}
                            disabled={
                                saving ||
                                formLocked ||
                                usernameKnownTaken ||
                                !fields.username.trim() ||
                                !fields.displayName.trim()
                            }
                        >
                            {saving ? "Saving..." : "Save"}
                        </Button>
                    </div>
                </div>
            </Modal>

            <Modal
                isOpen={showBaseForm}
                onClose={closeBaseForm}
                title={editingBase ? "Edit Base Prompt" : "Create Base Prompt"}
            >
                <div className={styles.form}>
                    {baseError && <div className={styles.error}>{baseError}</div>}
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("base-name")}>Name</label>
                        <Input
                            id={fieldID("base-name")}
                            value={baseName}
                            onChange={e => setBaseName(e.target.value)}
                            placeholder="game witch"
                            aria-describedby={fieldID("base-name-hint")}
                        />
                        <span id={fieldID("base-name-hint")} className={styles.fieldHint}>
                            How it appears in the dropdown on each bot. Names must be unique.
                        </span>
                    </div>
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("base-prompt-text")}>Prompt</label>
                        <TextArea
                            id={fieldID("base-prompt-text")}
                            rows={18}
                            value={basePrompt}
                            onChange={e => setBasePrompt(e.target.value)}
                            placeholder="You are a witch of the game boards, and this is the first half of your instructions..."
                            aria-describedby={fieldID("base-prompt-text-hint")}
                        />
                        <span id={fieldID("base-prompt-text-hint")} className={styles.fieldHint}>
                            This text is sent before the persona of every bot that extends it, so write only what they
                            all share: the hierarchy they obey, how the site works, and the rules none of them may
                            break. Anything that belongs to one character belongs in that bot's own prompt instead.
                            Roughly {Math.ceil(basePrompt.length / CHARS_PER_TOKEN).toLocaleString()} tokens, charged on
                            every reply from every bot using it.
                        </span>
                    </div>
                    <div className={styles.formActions}>
                        <Button variant="ghost" size="small" onClick={closeBaseForm}>
                            Cancel
                        </Button>
                        <Button
                            variant="primary"
                            size="small"
                            onClick={() => {
                                submitBase().catch(() => undefined);
                            }}
                            disabled={savingBase || baseName.trim() === "" || basePrompt.trim() === ""}
                        >
                            {savingBase ? "Saving..." : "Save"}
                        </Button>
                    </div>
                </div>
            </Modal>
        </div>
    );
}

interface BotEnabledToggleProps {
    bot: Chatbot;
    onChange: (bot: Chatbot, enabled: boolean) => void;
}

function BotEnabledToggle({ bot, onChange }: BotEnabledToggleProps) {
    return (
        <ToggleSwitch
            label="Enabled"
            ariaLabel={`Enabled ${bot.display_name}`}
            enabled={bot.enabled}
            onChange={v => onChange(bot, v)}
        />
    );
}

interface UsagePanelProps {
    label: string;
    usage: ChatbotUsage | null;
    loading: boolean;
}

function UsagePanel({ label, usage, loading }: UsagePanelProps) {
    return (
        <div className={styles.usageCard}>
            <span className={styles.usageLabel}>{label}</span>
            {loading || !usage ? (
                <span className={styles.usageEmpty}>Loading...</span>
            ) : (
                <>
                    <span className={styles.usageHeadline}>{usage.invocations.toLocaleString()} replies</span>
                    <dl className={styles.usageStats}>
                        <div className={styles.usageStat}>
                            <dt>Input</dt>
                            <dd>{usage.prompt_tokens.toLocaleString()}</dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Cached input</dt>
                            <dd>{usage.cached_prompt_tokens.toLocaleString()}</dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Cache writes</dt>
                            <dd>{usage.cache_write_tokens.toLocaleString()}</dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Output</dt>
                            <dd>{usage.completion_tokens.toLocaleString()}</dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Reasoning</dt>
                            <dd>{usage.reasoning_tokens.toLocaleString()}</dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Failed</dt>
                            <dd className={usage.failed > 0 ? styles.error : undefined}>
                                {usage.failed.toLocaleString()}
                            </dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Quota blocked</dt>
                            <dd>{usage.quota.toLocaleString()}</dd>
                        </div>
                    </dl>
                    <table className={styles.channelTable}>
                        <caption className={styles.channelCaption}>Where the replies came from</caption>
                        <thead>
                            <tr>
                                <th scope="col">Channel</th>
                                <th scope="col">Replies</th>
                                <th scope="col">Tokens</th>
                            </tr>
                        </thead>
                        <tbody>
                            {channelRows(usage.channels).map(channel => (
                                <tr key={channel.channel}>
                                    <th scope="row">{channelLabel(channel.channel)}</th>
                                    <td>{channel.invocations.toLocaleString()}</td>
                                    <td>{channelTokens(channel).toLocaleString()}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                    {usage.failed > 0 && (
                        <span className={styles.fieldHint}>
                            Failures are almost always a model id the provider does not recognise, a revoked or expired
                            API key, or a quota that has run out.
                        </span>
                    )}
                    {usage.billed_usd !== null && (
                        <span className={styles.usageBilled}>Billed ${usage.billed_usd.toFixed(2)}</span>
                    )}
                </>
            )}
        </div>
    );
}
