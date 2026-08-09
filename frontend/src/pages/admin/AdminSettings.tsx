import { useId, useRef, useState } from "react";
import type { ChangeEvent } from "react";
import { useAdminPermissions, useAdminSettings, useChatbotModels } from "../../api/queries/admin";
import {
    useSendTestEmail,
    useTestChatbotModel,
    useUpdateAdminSettings,
    useUploadOGDefaultImage,
} from "../../api/mutations/admin";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Select } from "../../components/Select/Select";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import type { SiteSettings } from "../../types/api";
import { ChatbotKeyGate } from "./ChatbotKeyGate";
import styles from "./AdminSettings.module.css";

const BYTES_PER_MB = 1024 * 1024;
const PIXELS_PER_MP = 1_000_000;
const CHATBOT_PERMISSION = "use_chatbot";

type EmailProvider = "smtp" | "cloudflare";
const EMAIL_PROVIDER_SMTP: EmailProvider = "smtp";
const EMAIL_PROVIDER_CLOUDFLARE: EmailProvider = "cloudflare";

export function AdminSettings() {
    usePageTitle("Admin - Settings");
    const { site_name } = useSiteInfo();
    const baseID = useId();
    const { settings: loadedSettings, loading } = useAdminSettings();
    const { models, modelsError, loading: modelsLoading, refresh: refreshModels } = useChatbotModels();
    const updateSettingsMutation = useUpdateAdminSettings();
    const sendTestEmailMutation = useSendTestEmail();
    const testModelMutation = useTestChatbotModel();
    const uploadOGImageMutation = useUploadOGDefaultImage();
    const ogImageInputRef = useRef<HTMLInputElement>(null);
    const [draft, setDraft] = useState<SiteSettings>({});
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");
    const [testMessage, setTestMessage] = useState("");
    const [testError, setTestError] = useState("");
    const [modelTestMessage, setModelTestMessage] = useState("");
    const [modelTestError, setModelTestError] = useState("");
    const [ogImageError, setOGImageError] = useState("");

    const saving = updateSettingsMutation.isPending;
    const settings: SiteSettings = { ...(loadedSettings ?? {}), ...draft };

    const chatbotKeySaved = (loadedSettings?.chatbot_api_key ?? "").trim() !== "";
    const chatbotLocked = !chatbotKeySaved || models.length === 0;

    const restrictChatbots = settings.chatbot_enabled === "true" && settings.chatbot_require_permission === "true";
    const { vanityRoles, loading: rolesLoading } = useAdminPermissions(restrictChatbots);
    const optInRoles = vanityRoles.filter(role => role.permissions.includes(CHATBOT_PERMISSION));
    const optInRoleID = (settings.chatbot_opt_in_role ?? "").trim();
    const optInRoleListed = optInRoles.some(role => role.id === optInRoleID);

    function fieldID(name: string) {
        return `${baseID}-${name}`;
    }

    function updateField(key: string, value: string) {
        setDraft(prev => ({ ...prev, [key]: value }));
        setSuccess("");
    }

    function toggleField(key: string, enabled: boolean) {
        updateField(key, enabled ? "true" : "false");
    }

    function getNumber(key: string): string {
        return settings[key] ?? "0";
    }

    function getMB(key: string): string {
        const bytes = parseInt(settings[key] ?? "0", 10);
        if (isNaN(bytes)) {
            return "0";
        }
        return String(Math.round(bytes / BYTES_PER_MB));
    }

    function setMB(key: string, mb: string) {
        const mbNum = parseFloat(mb);
        if (isNaN(mbNum)) {
            updateField(key, "0");
        } else {
            updateField(key, String(Math.round(mbNum * BYTES_PER_MB)));
        }
    }

    function getMP(key: string): string {
        const pixels = parseInt(settings[key] ?? "0", 10);
        if (isNaN(pixels)) {
            return "0";
        }
        return String(pixels / PIXELS_PER_MP);
    }

    function setMP(key: string, mp: string) {
        const mpNum = parseFloat(mp);
        if (isNaN(mpNum)) {
            updateField(key, "0");
        } else {
            updateField(key, String(Math.round(mpNum * PIXELS_PER_MP)));
        }
    }

    function validateSettings(): string | null {
        const maxBody = parseInt(settings.max_body_size ?? "0", 10);
        const maxImage = parseInt(settings.max_image_size ?? "0", 10);
        const maxImagePixels = parseInt(settings.max_image_pixels ?? "0", 10);
        const maxVideo = parseInt(settings.max_video_size ?? "0", 10);
        const maxGeneral = parseInt(settings.max_general_size ?? "0", 10);
        const minPassword = parseInt(settings.min_password_length ?? "0", 10);
        const sessionDays = parseInt(settings.session_duration_days ?? "0", 10);
        const maxTheories = parseInt(settings.max_theories_per_day ?? "0", 10);
        const maxResponses = parseInt(settings.max_responses_per_day ?? "0", 10);

        if (maxBody <= 0) {
            return "Max body size must be greater than 0";
        }
        if (maxImage <= 0) {
            return "Max image size must be greater than 0";
        }
        if (maxImagePixels <= 0) {
            return "Max image pixels must be greater than 0";
        }
        if (maxImage > maxBody) {
            return `Max image size (${Math.round(maxImage / BYTES_PER_MB)} MB) cannot exceed max body size (${Math.round(maxBody / BYTES_PER_MB)} MB)`;
        }
        if (maxVideo > maxBody) {
            return `Max video size (${Math.round(maxVideo / BYTES_PER_MB)} MB) cannot exceed max body size (${Math.round(maxBody / BYTES_PER_MB)} MB)`;
        }
        if (maxGeneral > maxBody) {
            return `Max general size (${Math.round(maxGeneral / BYTES_PER_MB)} MB) cannot exceed max body size (${Math.round(maxBody / BYTES_PER_MB)} MB)`;
        }
        if (minPassword < 1) {
            return "Minimum password length must be at least 1";
        }
        if (sessionDays < 1) {
            return "Session duration must be at least 1 day";
        }
        if (maxTheories < 0) {
            return "Max theories per day cannot be negative";
        }
        if (maxResponses < 0) {
            return "Max responses per day cannot be negative";
        }
        if (settings.voice_enabled === "true") {
            if (!settings.livekit_url || !settings.livekit_api_key || !settings.livekit_api_secret) {
                return "Voice chat requires LiveKit URL, API key and API secret";
            }
        }
        if (settings.chatbot_enabled === "true") {
            const maxOutputTokens = parseInt(settings.chatbot_max_output_tokens ?? "0", 10);
            const contextMessages = parseInt(settings.chatbot_context_messages ?? "0", 10);
            const maxReplyChain = parseInt(settings.chatbot_max_reply_chain ?? "0", 10);
            const replyCooldown = parseInt(settings.chatbot_reply_cooldown_seconds ?? "0", 10);
            const maxRepliesPerUser = parseInt(settings.chatbot_max_replies_per_user_per_day ?? "0", 10);
            const maxRepliesPerDay = parseInt(settings.chatbot_max_replies_per_day ?? "0", 10);

            if (maxOutputTokens < 1) {
                return "Chatbot max output tokens must be at least 1";
            }
            if (contextMessages < 0) {
                return "Chatbot context messages cannot be negative";
            }
            if (maxReplyChain < 0) {
                return "Chatbot max reply chain cannot be negative";
            }
            if (replyCooldown < 0) {
                return "Chatbot reply cooldown cannot be negative";
            }
            if (maxRepliesPerUser < 0) {
                return "Chatbot max replies per user per day cannot be negative";
            }
            if (maxRepliesPerDay < 0) {
                return "Chatbot max replies per day cannot be negative";
            }
        }
        if (settings.chatbot_require_permission === "true" && optInRoleID === "") {
            if (settings.chatbot_enabled !== "true") {
                return "Restricting characters to a permission requires an opt-in role. Switch Enable Chatbot on to choose one.";
            }
            return "Restricting characters to a permission requires an opt-in role so members can opt in";
        }
        if (settings.email_provider === EMAIL_PROVIDER_CLOUDFLARE) {
            if (!settings.cloudflare_account_id || !settings.cloudflare_api_token || !settings.cloudflare_email_from) {
                return "Cloudflare email requires account ID, API token and from address";
            }
        }
        return null;
    }

    async function handleSave() {
        const validationError = validateSettings();
        if (validationError) {
            setError(validationError);
            return;
        }

        setError("");
        setSuccess("");
        try {
            await updateSettingsMutation.mutateAsync(settings);
            setSuccess("Settings saved successfully");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to save settings");
        }
    }

    async function handleOGImageSelected(e: ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        e.target.value = "";
        if (!file) {
            return;
        }

        setOGImageError("");
        try {
            const res = await uploadOGImageMutation.mutateAsync(file);
            updateField("og_default_image", res.url);
        } catch (err) {
            setOGImageError(err instanceof Error ? err.message : "Failed to upload image");
        }
    }

    async function handleTestModel() {
        setModelTestMessage("");
        setModelTestError("");
        try {
            const result = await testModelMutation.mutateAsync((settings.chatbot_model ?? "").trim());

            if (result.ok) {
                setModelTestMessage("The model answered. Save your changes to put it live.");
            } else {
                setModelTestError(result.error ?? "The model did not answer");
            }
        } catch (e) {
            setModelTestError(e instanceof Error ? e.message : "Failed to reach the model");
        }
    }

    async function handleSendTestEmail() {
        setTestMessage("");
        setTestError("");
        try {
            await sendTestEmailMutation.mutateAsync();
            setTestMessage("Test email sent. Check your inbox.");
        } catch (e) {
            setTestError(e instanceof Error ? e.message : "Failed to send test email");
        }
    }

    if (loading) {
        return <div className={styles.loading}>Loading settings...</div>;
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.title}>Site Settings</h1>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Feature Toggles</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Registration</span>
                        <Select
                            value={settings.registration_type ?? "open"}
                            onChange={e => updateField("registration_type", e.target.value)}
                        >
                            <option value="open">Open (anyone can register)</option>
                            <option value="invite">Invite Only</option>
                            <option value="closed">Closed (no registration)</option>
                        </Select>
                    </div>
                    <ToggleSwitch
                        label="Maintenance Mode"
                        description="Put the site into maintenance mode"
                        enabled={settings.maintenance_mode === "true"}
                        onChange={v => toggleField("maintenance_mode", v)}
                    />
                    {settings.maintenance_mode === "true" && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Maintenance Title</span>
                                <Input
                                    value={settings.maintenance_title ?? ""}
                                    onChange={e => updateField("maintenance_title", e.target.value)}
                                    fullWidth
                                    placeholder="The game board is being prepared"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Maintenance Message</span>
                                <Input
                                    value={settings.maintenance_message ?? ""}
                                    onChange={e => updateField("maintenance_message", e.target.value)}
                                    fullWidth
                                    placeholder="Without love, it cannot be seen. Please check back shortly."
                                />
                            </div>
                        </>
                    )}
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Turnstile (Cloudflare)</h2>
                <div className={styles.fieldGroup}>
                    <ToggleSwitch
                        label="Enable Turnstile"
                        description="Require Cloudflare Turnstile verification on login and registration"
                        enabled={settings.turnstile_enabled === "true"}
                        onChange={v => toggleField("turnstile_enabled", v)}
                    />
                    {settings.turnstile_enabled === "true" && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Site Key</span>
                                <Input
                                    value={settings.turnstile_site_key ?? ""}
                                    onChange={e => updateField("turnstile_site_key", e.target.value)}
                                    fullWidth
                                    placeholder="0x..."
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Secret Key</span>
                                <Input
                                    type="password"
                                    value={settings.turnstile_secret_key ?? ""}
                                    onChange={e => updateField("turnstile_secret_key", e.target.value)}
                                    fullWidth
                                    placeholder="0x..."
                                />
                            </div>
                        </>
                    )}
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Watch Parties, Voice &amp; Streaming</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Watch party: shared browser (Hyperbeam)</span>
                        <Input
                            type="password"
                            value={settings.hyperbeam_api_key ?? ""}
                            onChange={e => updateField("hyperbeam_api_key", e.target.value)}
                            fullWidth
                            placeholder="sk_test_..."
                        />
                        <span className={styles.fieldHint}>
                            Lets members watch a shared virtual browser together. Leave it empty to offer screen sharing
                            only, which uses the LiveKit credentials below.
                        </span>
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Shared browser region</span>
                        <Select
                            value={settings.hyperbeam_region ?? "EU"}
                            onChange={e => updateField("hyperbeam_region", e.target.value)}
                        >
                            <option value="NA">North America</option>
                            <option value="EU">Europe</option>
                            <option value="AS">Asia</option>
                        </Select>
                        <span className={styles.fieldHint}>
                            Where the shared browser runs. Pick the one nearest most of your members.
                        </span>
                    </div>
                    <ToggleSwitch
                        label="Enable Voice Chat"
                        description="Allow voice calls in chat rooms and DMs (requires a self-hosted LiveKit server)"
                        enabled={settings.voice_enabled === "true"}
                        onChange={v => toggleField("voice_enabled", v)}
                    />
                    <ToggleSwitch
                        label="Enable Live Streaming"
                        description="Let members broadcast from OBS (WHIP) to a public /live page anyone can watch (requires the LiveKit ingress service)"
                        enabled={settings.streaming_enabled === "true"}
                        onChange={v => toggleField("streaming_enabled", v)}
                    />
                    {(settings.voice_enabled === "true" || settings.streaming_enabled === "true") && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>LiveKit URL</span>
                                <Input
                                    value={settings.livekit_url ?? ""}
                                    onChange={e => updateField("livekit_url", e.target.value)}
                                    fullWidth
                                    placeholder="wss://livekit.example.com"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>API Key</span>
                                <Input
                                    value={settings.livekit_api_key ?? ""}
                                    onChange={e => updateField("livekit_api_key", e.target.value)}
                                    fullWidth
                                    placeholder="APIxxxxxxxx"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>API Secret</span>
                                <Input
                                    type="password"
                                    value={settings.livekit_api_secret ?? ""}
                                    onChange={e => updateField("livekit_api_secret", e.target.value)}
                                    fullWidth
                                    placeholder="secret"
                                />
                            </div>
                        </>
                    )}
                    {settings.streaming_enabled === "true" && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Max Concurrent Streams</span>
                                <Input
                                    type="number"
                                    value={settings.stream_max_concurrent ?? ""}
                                    onChange={e => updateField("stream_max_concurrent", e.target.value)}
                                    fullWidth
                                    placeholder="3"
                                />
                            </div>
                            <ToggleSwitch
                                label="Enable Smooth (HLS) playback"
                                description="Record each live broadcaster to HLS so viewers can pick a buffered, freeze-resistant stream a few seconds behind live (requires the LiveKit egress service)"
                                enabled={settings.stream_hls_enabled === "true"}
                                onChange={v => toggleField("stream_hls_enabled", v)}
                            />
                            {settings.stream_hls_enabled === "true" && (
                                <div className={styles.field}>
                                    <span className={styles.fieldLabel}>HLS Output Directory</span>
                                    <Input
                                        value={settings.stream_hls_output_dir ?? ""}
                                        onChange={e => updateField("stream_hls_output_dir", e.target.value)}
                                        fullWidth
                                        placeholder="/app/data/hls"
                                    />
                                </div>
                            )}
                        </>
                    )}
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Mobile App</h2>
                <div className={styles.fieldGroup}>
                    <ToggleSwitch
                        label="Enable Push Notifications"
                        description="Send native push notifications to the mobile app when a recipient is offline (requires FCM_CREDENTIALS_FILE on the server)"
                        enabled={settings.push_enabled === "true"}
                        onChange={v => toggleField("push_enabled", v)}
                    />
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Latest App Version</span>
                        <Input
                            value={settings.app_latest_version ?? ""}
                            onChange={e => updateField("app_latest_version", e.target.value)}
                            fullWidth
                            placeholder="1.0.0"
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>App Download URL</span>
                        <Input
                            value={settings.app_download_url ?? ""}
                            onChange={e => updateField("app_download_url", e.target.value)}
                            fullWidth
                            placeholder="https://github.com/VictoriqueMoe/umineko_city_of_books/releases/latest"
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Chatbot</h2>
                <div className={styles.fieldGroup}>
                    <ToggleSwitch
                        label="Enable Chatbot"
                        description="Let members talk to bot accounts by mentioning or replying to them in chat"
                        enabled={settings.chatbot_enabled === "true"}
                        onChange={v => toggleField("chatbot_enabled", v)}
                    />
                    {settings.chatbot_enabled === "true" && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>API Key</span>
                                <Input
                                    type="password"
                                    value={settings.chatbot_api_key ?? ""}
                                    onChange={e => updateField("chatbot_api_key", e.target.value)}
                                    fullWidth
                                    placeholder="sk-..."
                                />
                                <span className={styles.fieldHint}>
                                    The API key every bot reply is charged against. Without it no bot can answer.
                                    Anything the bots generate appears on this key's bill, so treat the limits below as
                                    your spending controls.
                                </span>
                            </div>
                            {chatbotLocked && (
                                <ChatbotKeyGate
                                    apiKeySaved={chatbotKeySaved}
                                    checking={modelsLoading}
                                    reason={modelsError}
                                    onRetry={refreshModels}
                                />
                            )}
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Admin Key</span>
                                <Input
                                    type="password"
                                    value={settings.chatbot_admin_key ?? ""}
                                    onChange={e => updateField("chatbot_admin_key", e.target.value)}
                                    fullWidth
                                    placeholder="Optional, for reading the billed spend"
                                    disabled={chatbotLocked}
                                />
                                <span className={styles.fieldHint}>
                                    Optional organisation admin key used only to read back what has actually been
                                    billed, which is shown on the Chatbots page. Leave it empty and everything still
                                    works, you just see token counts instead of a money figure.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <label className={styles.fieldLabel} htmlFor={fieldID("chatbot-model")}>
                                    Model
                                </label>
                                <Input
                                    id={fieldID("chatbot-model")}
                                    value={settings.chatbot_model ?? ""}
                                    onChange={e => updateField("chatbot_model", e.target.value)}
                                    fullWidth
                                    placeholder="gpt-5.6-luna"
                                    aria-describedby={fieldID("chatbot-model-hint")}
                                    list={models.length > 0 ? fieldID("chatbot-model-options") : undefined}
                                    disabled={chatbotLocked}
                                />
                                {models.length > 0 && (
                                    <datalist id={fieldID("chatbot-model-options")}>
                                        {models.map(model => (
                                            <option key={model} value={model} />
                                        ))}
                                    </datalist>
                                )}
                                <span id={fieldID("chatbot-model-hint")} className={styles.fieldHint}>
                                    The default model every bot uses unless it overrides it on the Chatbots page. Larger
                                    models write better replies and cost more per token, so this is the single biggest
                                    lever on your bill. The list is whatever your key can actually see, and anything not
                                    on it can still be typed in by hand.
                                </span>
                                <Button
                                    variant="secondary"
                                    onClick={handleTestModel}
                                    disabled={
                                        testModelMutation.isPending ||
                                        chatbotLocked ||
                                        !(settings.chatbot_model ?? "").trim()
                                    }
                                >
                                    {testModelMutation.isPending ? "Testing..." : "Test model"}
                                </Button>
                                {modelTestMessage && <span className={styles.success}>{modelTestMessage}</span>}
                                {modelTestError && <span className={styles.saveError}>{modelTestError}</span>}
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Reasoning Effort</span>
                                <Select
                                    value={settings.chatbot_reasoning_effort ?? "low"}
                                    onChange={e => updateField("chatbot_reasoning_effort", e.target.value)}
                                    disabled={chatbotLocked}
                                >
                                    <option value="none">None</option>
                                    <option value="low">Low</option>
                                    <option value="medium">Medium</option>
                                    <option value="high">High</option>
                                    <option value="xhigh">Extra high</option>
                                    <option value="max">Max</option>
                                </Select>
                                <span className={styles.fieldHint}>
                                    How much hidden thinking the model does before it answers. Reasoning tokens are
                                    billed like any other output but are never shown to anyone, so higher settings cost
                                    real money and add latency for very little gain in casual chat. Low or none suits
                                    conversation.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Verbosity</span>
                                <Select
                                    value={settings.chatbot_verbosity ?? ""}
                                    onChange={e => updateField("chatbot_verbosity", e.target.value)}
                                    disabled={chatbotLocked}
                                >
                                    <option value="">Provider default</option>
                                    <option value="low">Low</option>
                                    <option value="medium">Medium</option>
                                    <option value="high">High</option>
                                </Select>
                                <span className={styles.fieldHint}>
                                    How much detail a reply carries, set on the request rather than written into the
                                    personality. Low keeps answers short, high lets them run on. Leave it on the
                                    provider default unless replies are consistently too long or too clipped, and note
                                    that anything a personality says about length still applies on top of this.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Max Output Tokens</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_max_output_tokens")}
                                    onChange={e => updateField("chatbot_max_output_tokens", e.target.value)}
                                    disabled={chatbotLocked}
                                />
                                <span className={styles.fieldHint}>
                                    The hard ceiling on a single reply, counting reasoning tokens and visible text
                                    together, so a heavy thinker can spend most of this budget before it writes a word.
                                    This is a safety limit that stops a runaway generation, not a way to ask for shorter
                                    replies. Set the length you want in the bot's persona instead, and leave this high
                                    enough that normal answers are never cut off mid-sentence.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Context Messages</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_context_messages")}
                                    onChange={e => updateField("chatbot_context_messages", e.target.value)}
                                    disabled={chatbotLocked}
                                />
                                <span className={styles.fieldHint}>
                                    How many recent messages from the room are sent along with the question so the bot
                                    knows what is being discussed. Every one of them is billed as input on each call, so
                                    doubling this roughly doubles the input cost of every reply.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Max Reply Chain</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_max_reply_chain")}
                                    onChange={e => updateField("chatbot_max_reply_chain", e.target.value)}
                                    disabled={chatbotLocked}
                                />
                                <span className={styles.fieldHint}>
                                    How far back a chain of replies is followed when someone answers a bot, so a long
                                    back-and-forth keeps its thread. Longer chains give better continuity and cost more,
                                    because the whole chain is resent as input each turn.
                                </span>
                            </div>
                            <ToggleSwitch
                                label="Restrict To Chatbot Permission"
                                description="Only members granted the Summon Chatbots permission can get a reply. Grant it on the Permissions page, to Moderator or to any vanity role."
                                enabled={settings.chatbot_require_permission === "true"}
                                onChange={v => toggleField("chatbot_require_permission", v)}
                            />
                            {restrictChatbots && (
                                <div className={styles.field}>
                                    <label className={styles.fieldLabel} htmlFor={fieldID("chatbot-opt-in-role")}>
                                        Opt In Role
                                    </label>
                                    <Select
                                        id={fieldID("chatbot-opt-in-role")}
                                        value={optInRoleID}
                                        onChange={e => updateField("chatbot_opt_in_role", e.target.value)}
                                        aria-describedby={fieldID("chatbot-opt-in-role-hint")}
                                        disabled={rolesLoading}
                                    >
                                        <option value="">Select a role...</option>
                                        {optInRoleID !== "" && !optInRoleListed && (
                                            <option value={optInRoleID}>
                                                The saved role no longer carries Summon Chatbots
                                            </option>
                                        )}
                                        {optInRoles.map(role => (
                                            <option key={role.id} value={role.id}>
                                                {role.label}
                                            </option>
                                        ))}
                                    </Select>
                                    <span id={fieldID("chatbot-opt-in-role-hint")} className={styles.fieldHint}>
                                        The vanity role a member is given when they opt in to characters from their own
                                        settings page. Opting in grants the whole role, so anything else it carries is
                                        granted with it. Only roles that already hold Summon Chatbots are offered, and
                                        moving to a different role moves everyone who opted in across to it.
                                    </span>
                                    {!rolesLoading && optInRoles.length === 0 && (
                                        <span className={styles.saveError}>
                                            No vanity role holds Summon Chatbots yet. Grant it to one on the Permissions
                                            page before restricting characters.
                                        </span>
                                    )}
                                </div>
                            )}
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Reply Cooldown (seconds)</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_reply_cooldown_seconds")}
                                    onChange={e => updateField("chatbot_reply_cooldown_seconds", e.target.value)}
                                    disabled={chatbotLocked}
                                />
                                <span className={styles.fieldHint}>
                                    The minimum wait between one member's replies. It stops someone hammering a bot in a
                                    tight loop and turning a quiet room into a large bill.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Max Replies Per Member (rolling 24 hours)</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_max_replies_per_user_per_day")}
                                    onChange={e => updateField("chatbot_max_replies_per_user_per_day", e.target.value)}
                                    disabled={chatbotLocked}
                                />
                                <span className={styles.fieldHint}>
                                    How many replies one member can pull out of the bots in the last 24 hours. This is a
                                    rolling window, not a calendar day: nothing resets at midnight, and allowance comes
                                    back gradually as each old reply passes the 24 hour mark. A member who hits the
                                    limit is told so by the character, with a rough idea of when they can try again, and
                                    nothing further is charged for them. Set it to 0 for no per-member limit.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Max Replies Site-wide (rolling 24 hours)</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_max_replies_per_day")}
                                    onChange={e => updateField("chatbot_max_replies_per_day", e.target.value)}
                                    disabled={chatbotLocked}
                                />
                                <span className={styles.fieldHint}>
                                    The ceiling across every member and every bot in the last 24 hours, also a rolling
                                    window. This is your last line of defence on cost, so pick a number whose worst-case
                                    bill you are happy to pay. 0 removes the ceiling entirely, which is rarely what you
                                    want here.
                                </span>
                            </div>
                        </>
                    )}
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>General</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Site Name</span>
                        <Input
                            value={settings.site_name ?? ""}
                            onChange={e => updateField("site_name", e.target.value)}
                            fullWidth
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Site Description</span>
                        <Input
                            value={settings.site_description ?? ""}
                            onChange={e => updateField("site_description", e.target.value)}
                            fullWidth
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Announcement Banner</span>
                        <Input
                            value={settings.announcement_banner ?? ""}
                            onChange={e => updateField("announcement_banner", e.target.value)}
                            fullWidth
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Cache (Valkey)</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Valkey URL</span>
                        <Input
                            value={settings.valkey_url ?? ""}
                            onChange={e => updateField("valkey_url", e.target.value)}
                            fullWidth
                            placeholder="redis://valkey-cache:6379/0"
                        />
                        <span className={styles.fieldHint}>
                            Connection URL for the app cache, separate from the LiveKit coordination Valkey. When set,
                            caching is enabled; leave it empty to disable and bypass the cache entirely. Changes take
                            effect immediately, with no restart needed.
                        </span>
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Limits</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Theories Per Day</span>
                        <Input
                            type="number"
                            value={getNumber("max_theories_per_day")}
                            onChange={e => updateField("max_theories_per_day", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Responses Per Day</span>
                        <Input
                            type="number"
                            value={getNumber("max_responses_per_day")}
                            onChange={e => updateField("max_responses_per_day", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Min Password Length</span>
                        <Input
                            type="number"
                            value={getNumber("min_password_length")}
                            onChange={e => updateField("min_password_length", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Session Duration (days)</span>
                        <Input
                            type="number"
                            value={getNumber("session_duration_days")}
                            onChange={e => updateField("session_duration_days", e.target.value)}
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>File Size Limits</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Image Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_image_size")}
                            onChange={e => setMB("max_image_size", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Image Pixels (megapixels)</span>
                        <Input
                            type="number"
                            value={getMP("max_image_pixels")}
                            onChange={e => setMP("max_image_pixels", e.target.value)}
                        />
                        <span className={styles.fieldHint}>
                            Rejects images whose width x height exceeds this, however small the file is. A highly
                            compressible image can be tiny on disk yet need gigabytes of memory to decode.
                        </span>
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Video Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_video_size")}
                            onChange={e => setMB("max_video_size", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max General Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_general_size")}
                            onChange={e => setMB("max_general_size", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Body Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_body_size")}
                            onChange={e => setMB("max_body_size", e.target.value)}
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Email</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Email Provider</span>
                        <Select
                            value={settings.email_provider ?? EMAIL_PROVIDER_SMTP}
                            onChange={e => updateField("email_provider", e.target.value)}
                        >
                            <option value={EMAIL_PROVIDER_SMTP}>SMTP</option>
                            <option value={EMAIL_PROVIDER_CLOUDFLARE}>Cloudflare Email Service</option>
                        </Select>
                    </div>
                    {(settings.email_provider ?? EMAIL_PROVIDER_SMTP) === EMAIL_PROVIDER_SMTP && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Host</span>
                                <Input
                                    value={settings.smtp_host ?? ""}
                                    onChange={e => updateField("smtp_host", e.target.value)}
                                    fullWidth
                                    placeholder="127.0.0.1"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Port</span>
                                <Input
                                    type="number"
                                    value={getNumber("smtp_port")}
                                    onChange={e => updateField("smtp_port", e.target.value)}
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>From Address</span>
                                <Input
                                    value={settings.smtp_from ?? ""}
                                    onChange={e => updateField("smtp_from", e.target.value)}
                                    fullWidth
                                    placeholder="noreply@example.com"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Username</span>
                                <Input
                                    value={settings.smtp_username ?? ""}
                                    onChange={e => updateField("smtp_username", e.target.value)}
                                    fullWidth
                                    placeholder="Leave empty for no auth"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Password</span>
                                <Input
                                    type="password"
                                    value={settings.smtp_password ?? ""}
                                    onChange={e => updateField("smtp_password", e.target.value)}
                                    fullWidth
                                    placeholder="Leave empty for no auth"
                                />
                            </div>
                        </>
                    )}
                    {settings.email_provider === EMAIL_PROVIDER_CLOUDFLARE && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Account ID</span>
                                <Input
                                    value={settings.cloudflare_account_id ?? ""}
                                    onChange={e => updateField("cloudflare_account_id", e.target.value)}
                                    fullWidth
                                    placeholder="Cloudflare account ID"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>API Token</span>
                                <Input
                                    type="password"
                                    value={settings.cloudflare_api_token ?? ""}
                                    onChange={e => updateField("cloudflare_api_token", e.target.value)}
                                    fullWidth
                                    placeholder="Token with email sending permission"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>From Address</span>
                                <Input
                                    value={settings.cloudflare_email_from ?? ""}
                                    onChange={e => updateField("cloudflare_email_from", e.target.value)}
                                    fullWidth
                                    placeholder="noreply@yourdomain.com"
                                />
                            </div>
                        </>
                    )}
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>
                            Sends a test email to your own account using the saved settings. Save changes first.
                        </span>
                        <Button
                            variant="secondary"
                            onClick={handleSendTestEmail}
                            disabled={sendTestEmailMutation.isPending}
                        >
                            {sendTestEmailMutation.isPending ? "Sending..." : "Send test email"}
                        </Button>
                        {testMessage && <span className={styles.success}>{testMessage}</span>}
                        {testError && <span className={styles.saveError}>{testError}</span>}
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Logging & Error Reporting</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Log Level</span>
                        <Select
                            value={settings.log_level ?? "info"}
                            onChange={e => updateField("log_level", e.target.value)}
                        >
                            <option value="trace">Trace</option>
                            <option value="debug">Debug</option>
                            <option value="info">Info</option>
                            <option value="warn">Warn</option>
                            <option value="error">Error</option>
                        </Select>
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Sentry DSN</span>
                        <Input
                            value={settings.sentry_dsn ?? ""}
                            onChange={e => updateField("sentry_dsn", e.target.value)}
                            fullWidth
                            placeholder="Leave empty to disable"
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>
                            OTLP endpoint (OpenTelemetry traces, e.g. http://tempo:4318)
                        </span>
                        <Input
                            value={settings.otlp_endpoint ?? ""}
                            onChange={e => updateField("otlp_endpoint", e.target.value)}
                            fullWidth
                            placeholder="Leave empty to disable tracing"
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>
                            Pyroscope URL (continuous profiling, e.g. http://pyroscope:4040)
                        </span>
                        <Input
                            value={settings.pyroscope_url ?? ""}
                            onChange={e => updateField("pyroscope_url", e.target.value)}
                            fullWidth
                            placeholder="Leave empty to disable profiling"
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Appearance</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Default Theme</span>
                        <Select
                            value={settings.default_theme ?? "featherine"}
                            onChange={e => updateField("default_theme", e.target.value)}
                        >
                            <option value="featherine">Featherine</option>
                            <option value="beatrice">Beatrice</option>
                            <option value="bernkastel">Bernkastel</option>
                            <option value="lambdadelta">Lambdadelta</option>
                            <option value="erika">Erika Furudo</option>
                            <option value="battler">Battler Ushiromiya</option>
                            <option value="virgilia">Virgilia</option>
                            <option value="rika">Rika Furude</option>
                            <option value="mion">Mion Sonozaki</option>
                            <option value="satoko">Satoko Houjou</option>
                            <option value="miyao">Miyao</option>
                            <option value="lingji">Lingji</option>
                            <option value="stanislaw">Stanis&#322;aw</option>
                        </Select>
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Link Previews</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>
                            Default embed image shown when a link to the site is shared on Discord, X, and other
                            platforms, and the page has no image of its own. JPG only.
                        </span>
                        <div className={styles.embedActions}>
                            <Button
                                variant="secondary"
                                onClick={() => ogImageInputRef.current?.click()}
                                disabled={uploadOGImageMutation.isPending}
                            >
                                {uploadOGImageMutation.isPending ? "Uploading..." : "Upload image"}
                            </Button>
                            {(settings.og_default_image ?? "") !== "" && (
                                <Button variant="secondary" onClick={() => updateField("og_default_image", "")}>
                                    Reset to built-in
                                </Button>
                            )}
                            {ogImageError && <span className={styles.saveError}>{ogImageError}</span>}
                        </div>
                        <input
                            ref={ogImageInputRef}
                            type="file"
                            accept="image/jpeg,.jpg"
                            className={styles.hiddenInput}
                            onChange={handleOGImageSelected}
                        />
                    </div>
                    <EmbedPreviews
                        image={settings.og_default_image || "/Featherine.jpg"}
                        siteName={settings.site_name ?? site_name}
                        baseURL={settings.base_url ?? ""}
                    />
                </div>
            </div>

            <div className={styles.saveRow}>
                <Button variant="primary" onClick={handleSave} disabled={saving}>
                    {saving ? "Saving..." : "Save Settings"}
                </Button>
                {error && <span className={styles.saveError}>{error}</span>}
                {success && <span className={styles.success}>{success}</span>}
            </div>
        </div>
    );
}

const EMBED_PREVIEW_DESCRIPTION =
    "Welcome to the game board. Declare blue truths, solve mysteries, debate pairings, read and write fanfiction, and chronicle your journey through When They Cry.";

function EmbedPreviews({ image, siteName, baseURL }: { image: string; siteName: string; baseURL: string }) {
    const domain = baseURL.replace(/^https?:\/\//, "").replace(/\/$/, "") || "whentheycry.social";

    return (
        <div className={styles.embedPreviews}>
            <div className={styles.embedPreviewColumn}>
                <span className={styles.embedPreviewLabel}>Discord</span>
                <div className={styles.discordPreview}>
                    <div className={styles.discordBar} />
                    <div className={styles.discordBody}>
                        <span className={styles.discordSite}>{siteName}</span>
                        <span className={styles.discordTitle}>{siteName}</span>
                        <span className={styles.discordDesc}>{EMBED_PREVIEW_DESCRIPTION}</span>
                        <img src={image} alt="Embed preview" className={styles.discordImage} />
                    </div>
                </div>
            </div>
            <div className={styles.embedPreviewColumn}>
                <span className={styles.embedPreviewLabel}>X / Twitter</span>
                <div className={styles.twitterPreview}>
                    <img src={image} alt="Embed preview" className={styles.twitterImage} />
                    <div className={styles.twitterBody}>
                        <span className={styles.twitterDomain}>{domain}</span>
                        <span className={styles.twitterTitle}>{siteName}</span>
                        <span className={styles.twitterDesc}>{EMBED_PREVIEW_DESCRIPTION}</span>
                    </div>
                </div>
            </div>
        </div>
    );
}
