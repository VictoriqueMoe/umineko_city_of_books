import { useAdminSettings, useChatbotBasePrompts, useChatbotModels, useChatbots } from "../../hooks/queries/admin";
import { useChatbotEditor } from "../../hooks/useChatbotEditor";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { ConfirmDialog } from "../../components/ConfirmDialog/ConfirmDialog";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import type { Chatbot } from "../../types/api";
import { BasePromptSection } from "./BasePromptSection";
import { ChatbotForm } from "./ChatbotForm";
import { ChatbotUsageCards } from "./ChatbotUsageCards";
import styles from "./AdminChatbots.module.css";

export function AdminChatbots() {
    usePageTitle("Admin - Chatbots");
    const { bots, loading } = useChatbots();
    const { basePrompts } = useChatbotBasePrompts();
    const { settings } = useAdminSettings();
    const { models, modelsError, loading: modelsLoading, refresh: refreshModels } = useChatbotModels();
    const editor = useChatbotEditor();

    const apiKeySaved = (settings?.chatbot_api_key ?? "").trim() !== "";
    const pendingDelete = editor.pendingDelete;

    if (loading) {
        return <div className={styles.loading}>Loading chatbots...</div>;
    }

    return (
        <div className={styles.page}>
            <div className={styles.header}>
                <h1 className={styles.title}>Chatbots</h1>
                <Button variant="primary" onClick={editor.openCreate}>
                    Create Bot
                </Button>
            </div>

            <p className={styles.intro}>
                Each bot is a real account members can mention or reply to in chat. Its personality lives entirely in
                the system prompt you write below, and anything left blank falls back to the site defaults on the
                Settings page.
            </p>

            {editor.error && <div className={styles.error}>{editor.error}</div>}

            <BasePromptSection basePrompts={basePrompts} />

            <ChatbotUsageCards />

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
                                <BotEnabledToggle bot={bot} onChange={editor.toggleEnabled} />
                                {editor.togglingID === bot.id && <span className={styles.botMeta}>Saving...</span>}
                            </div>
                            <div className={styles.actions}>
                                <Button
                                    variant="secondary"
                                    size="small"
                                    aria-label={`Edit ${bot.display_name}`}
                                    onClick={() => editor.openEdit(bot)}
                                >
                                    Edit
                                </Button>
                                <Button
                                    variant="danger"
                                    size="small"
                                    aria-label={`Delete ${bot.display_name}`}
                                    onClick={() => editor.requestDelete(bot)}
                                >
                                    Delete
                                </Button>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            <ChatbotForm
                open={editor.open}
                editing={editor.editingBot !== null}
                fields={editor.fields}
                usernameCheck={editor.usernameCheck}
                usernameKnownTaken={editor.usernameKnownTaken}
                error={editor.formError}
                saving={editor.saving}
                basePrompts={basePrompts}
                models={models}
                apiKeySaved={apiKeySaved}
                modelsLoading={modelsLoading}
                modelsError={modelsError}
                onFieldChange={editor.setField}
                onUsernameBlur={editor.checkUsername}
                onRetryModels={refreshModels}
                onSave={editor.save}
                onClose={editor.close}
            />

            <ConfirmDialog
                open={pendingDelete !== null}
                title="Confirm Delete"
                body={
                    pendingDelete
                        ? `Delete ${pendingDelete.display_name} (@${pendingDelete.username})? The bot account and its replies go with it.`
                        : ""
                }
                confirmLabel="Delete"
                destructive
                onConfirm={editor.confirmDelete}
                onCancel={editor.cancelDelete}
            />
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
