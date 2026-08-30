import { useId } from "react";
import { usernameHint, type ChatbotFields, type UsernameCheck } from "../../domain/chatbots";
import {
    CACHING_TOKEN_THRESHOLD,
    estimateTokens,
    meetsCachingThreshold,
    tokensToCachingThreshold,
} from "../../domain/chatbotUsage";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import { Select } from "../../components/Select/Select";
import { TextArea } from "../../components/TextArea/TextArea";
import type { ChatbotBasePrompt } from "../../types/api";
import { ChatbotKeyGate } from "./ChatbotKeyGate";
import styles from "./AdminChatbots.module.css";

interface ChatbotFormProps {
    open: boolean;
    editing: boolean;
    fields: ChatbotFields;
    usernameCheck: UsernameCheck;
    usernameKnownTaken: boolean;
    error: string;
    saving: boolean;
    basePrompts: ChatbotBasePrompt[];
    models: string[];
    apiKeySaved: boolean;
    modelsLoading: boolean;
    modelsError: string;
    onFieldChange: <K extends keyof ChatbotFields>(key: K, value: ChatbotFields[K]) => void;
    onUsernameBlur: () => void;
    onRetryModels: () => void;
    onSave: () => void;
    onClose: () => void;
}

export function ChatbotForm({
    open,
    editing,
    fields,
    usernameCheck,
    usernameKnownTaken,
    error,
    saving,
    basePrompts,
    models,
    apiKeySaved,
    modelsLoading,
    modelsError,
    onFieldChange,
    onUsernameBlur,
    onRetryModels,
    onSave,
    onClose,
}: ChatbotFormProps) {
    const baseID = useId();

    const locked = !apiKeySaved || models.length === 0;
    const promptTokenEstimate = estimateTokens(fields.prompt);

    function fieldID(name: string) {
        return `${baseID}-${name}`;
    }

    return (
        <Modal isOpen={open} onClose={onClose} title={editing ? "Edit Chatbot" : "Create Chatbot"}>
            <div className={styles.form}>
                {locked && (
                    <ChatbotKeyGate
                        apiKeySaved={apiKeySaved}
                        checking={modelsLoading}
                        reason={modelsError}
                        onRetry={onRetryModels}
                    />
                )}
                <div className={styles.fieldLabel}>
                    <label htmlFor={fieldID("username")}>Username</label>
                    <Input
                        id={fieldID("username")}
                        type="text"
                        value={fields.username}
                        onChange={e => onFieldChange("username", e.target.value)}
                        onBlur={onUsernameBlur}
                        placeholder="beatrice"
                        aria-describedby={fieldID("username-hint")}
                        fullWidth
                        disabled={locked || editing}
                    />
                    <span id={fieldID("username-hint")} className={styles.fieldHint} aria-live="polite">
                        {usernameHint(usernameCheck, editing)}
                    </span>
                </div>
                <div className={styles.fieldLabel}>
                    <label htmlFor={fieldID("display-name")}>Display Name</label>
                    <Input
                        id={fieldID("display-name")}
                        type="text"
                        value={fields.displayName}
                        onChange={e => onFieldChange("displayName", e.target.value)}
                        placeholder="Beatrice"
                        aria-describedby={fieldID("display-name-hint")}
                        fullWidth
                        disabled={locked}
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
                        onChange={e => onFieldChange("avatarURL", e.target.value)}
                        placeholder="https://example.com/avatar.png"
                        aria-describedby={fieldID("avatar-url-hint")}
                        fullWidth
                        disabled={locked}
                    />
                    <span id={fieldID("avatar-url-hint")} className={styles.fieldHint}>
                        Leave it empty and the bot falls back to an initial, the same as any member without a picture.
                    </span>
                </div>
                <div className={styles.fieldLabel}>
                    <label htmlFor={fieldID("base-prompt")}>Base Prompt</label>
                    <Select
                        id={fieldID("base-prompt")}
                        value={fields.basePromptID}
                        onChange={e => onFieldChange("basePromptID", e.target.value)}
                        aria-describedby={fieldID("base-prompt-hint")}
                        disabled={locked}
                    >
                        <option value="">None</option>
                        {basePrompts.map(base => (
                            <option key={base.id} value={base.id}>
                                {base.name}
                            </option>
                        ))}
                    </Select>
                    <span id={fieldID("base-prompt-hint")} className={styles.fieldHint}>
                        Shared text that every bot extending it receives before its own personality, for the rules they
                        all obey rather than anything one character does. Edit it once and every bot using it picks the
                        change up immediately. Because it sits in front of the persona it is identical across those
                        bots, which is exactly what prompt caching keys on.
                    </span>
                </div>
                <div className={styles.fieldLabel}>
                    <label htmlFor={fieldID("system-prompt")}>System Prompt</label>
                    <TextArea
                        id={fieldID("system-prompt")}
                        rows={14}
                        value={fields.prompt}
                        onChange={e => onFieldChange("prompt", e.target.value)}
                        placeholder="You are Beatrice, the Golden Witch of Rokkenjima. You speak with..."
                        aria-describedby={`${fieldID("system-prompt-hint")} ${fieldID("system-prompt-caching")} ${fieldID("system-prompt-count")}`}
                        disabled={locked}
                    />
                    <span id={fieldID("system-prompt-hint")} className={styles.fieldHint}>
                        This is the entire personality. There is no fine-tuning and no training step behind it, so
                        whatever the bot knows about who it is, how it talks, what it refuses to discuss and how long
                        its answers run has to be written here. Be generous and specific: give it voice, history,
                        quirks, relationships and worked examples of good replies.
                    </span>
                    <span id={fieldID("system-prompt-caching")} className={styles.fieldHint}>
                        Past roughly {CACHING_TOKEN_THRESHOLD} tokens the prompt gets noticeably better in character,
                        and it also becomes eligible for prompt caching, which makes the input on every repeat call
                        about ten times cheaper. A long, stable persona is therefore both better and cheaper than a
                        short one.
                    </span>
                    <span
                        id={fieldID("system-prompt-count")}
                        className={
                            meetsCachingThreshold(promptTokenEstimate)
                                ? `${styles.tokenCount} ${styles.tokenCountGood}`
                                : styles.tokenCount
                        }
                    >
                        About {promptTokenEstimate} tokens
                        {meetsCachingThreshold(promptTokenEstimate)
                            ? " - long enough for prompt caching"
                            : ` - ${tokensToCachingThreshold(promptTokenEstimate)} more to reach the caching threshold`}
                    </span>
                </div>

                <div className={styles.overridesTitle}>Per-bot overrides</div>
                <span id={fieldID("overrides-hint")} className={styles.fieldHint}>
                    Leave these blank to inherit whatever the Settings page has configured. Only set them when this one
                    bot genuinely needs to differ.
                </span>
                <div className={styles.fieldLabel}>
                    <label htmlFor={fieldID("model")}>Model</label>
                    <Input
                        id={fieldID("model")}
                        type="text"
                        value={fields.model}
                        onChange={e => onFieldChange("model", e.target.value)}
                        placeholder="Inherit the site default model"
                        aria-describedby={fieldID("overrides-hint")}
                        list={models.length > 0 ? fieldID("model-options") : undefined}
                        fullWidth
                        disabled={locked}
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
                        onChange={e => onFieldChange("effort", e.target.value)}
                        aria-describedby={fieldID("overrides-hint")}
                        disabled={locked}
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
                        onChange={e => onFieldChange("verbosity", e.target.value)}
                        aria-describedby={fieldID("overrides-hint")}
                        disabled={locked}
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
                        onChange={e => onFieldChange("maxTokens", e.target.value)}
                        placeholder="Inherit the site default cap"
                        aria-describedby={`${fieldID("overrides-hint")} ${fieldID("max-output-tokens-hint")}`}
                        fullWidth
                        disabled={locked}
                    />
                    <span id={fieldID("max-output-tokens-hint")} className={styles.fieldHint}>
                        Caps reasoning and visible text together for this bot. It is a safety limit, not a way to ask
                        for shorter replies.
                    </span>
                </div>

                {error && (
                    <div className={styles.error} role="alert">
                        {error}
                    </div>
                )}

                <div className={styles.formActions}>
                    <Button variant="ghost" size="small" onClick={onClose}>
                        Cancel
                    </Button>
                    <Button
                        variant="primary"
                        size="small"
                        onClick={onSave}
                        disabled={
                            saving ||
                            locked ||
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
    );
}
