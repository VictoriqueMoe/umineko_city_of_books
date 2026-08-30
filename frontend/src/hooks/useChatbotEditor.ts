import { useState } from "react";
import {
    buildPayload,
    EMPTY_CHATBOT_FIELDS,
    fieldsFromBot,
    type ChatbotFields,
    type UsernameCheck,
} from "../domain/chatbots";
import { errorMessage } from "../utils/errorMessage";
import { useCreateChatbot, useDeleteChatbot, useUpdateChatbot } from "./mutations/admin";
import { useCheckUsernameAvailable } from "./mutations/admin";
import type { Chatbot } from "../types/api";

export interface ChatbotEditor {
    error: string;
    formError: string;
    open: boolean;
    editingBot: Chatbot | null;
    fields: ChatbotFields;
    usernameCheck: UsernameCheck;
    usernameKnownTaken: boolean;
    saving: boolean;
    togglingID: string | null;
    pendingDelete: Chatbot | null;
    setField: <K extends keyof ChatbotFields>(key: K, value: ChatbotFields[K]) => void;
    checkUsername: () => void;
    openCreate: () => void;
    openEdit: (bot: Chatbot) => void;
    close: () => void;
    save: () => void;
    toggleEnabled: (bot: Chatbot, enabled: boolean) => void;
    requestDelete: (bot: Chatbot) => void;
    confirmDelete: () => void;
    cancelDelete: () => void;
}

export function useChatbotEditor(): ChatbotEditor {
    const createMutation = useCreateChatbot();
    const updateMutation = useUpdateChatbot();
    const deleteMutation = useDeleteChatbot();
    const usernameMutation = useCheckUsernameAvailable();

    const [error, setError] = useState("");
    const [formError, setFormError] = useState("");
    const [open, setOpen] = useState(false);
    const [editingBot, setEditingBot] = useState<Chatbot | null>(null);
    const [fields, setFields] = useState<ChatbotFields>(EMPTY_CHATBOT_FIELDS);
    const [usernameCheck, setUsernameCheck] = useState<UsernameCheck>({ state: "idle" });
    const [togglingID, setTogglingID] = useState<string | null>(null);
    const [pendingDelete, setPendingDelete] = useState<Chatbot | null>(null);

    const saving = createMutation.isPending || updateMutation.isPending;
    const usernameKnownTaken = usernameCheck.state === "taken" && usernameCheck.username === fields.username.trim();

    function setField<K extends keyof ChatbotFields>(key: K, value: ChatbotFields[K]) {
        setFields(current => ({ ...current, [key]: value }));

        if (key === "username") {
            setUsernameCheck({ state: "idle" });
        }
    }

    function openCreate() {
        setEditingBot(null);
        setFields(EMPTY_CHATBOT_FIELDS);
        setUsernameCheck({ state: "idle" });
        setFormError("");
        setOpen(true);
    }

    function openEdit(bot: Chatbot) {
        setEditingBot(bot);
        setFields(fieldsFromBot(bot));
        setUsernameCheck({ state: "idle" });
        setFormError("");
        setOpen(true);
    }

    function close() {
        setOpen(false);
        setEditingBot(null);
    }

    async function runUsernameCheck() {
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
            const result = await usernameMutation.mutateAsync(username);

            setUsernameCheck({ state: result.available ? "available" : "taken", username: result.username });
        } catch {
            setUsernameCheck({ state: "failed" });
        }
    }

    function checkUsername() {
        runUsernameCheck().catch(() => undefined);
    }

    async function runSave() {
        setFormError("");
        try {
            if (editingBot) {
                await updateMutation.mutateAsync({
                    id: editingBot.id,
                    data: buildPayload(fields, editingBot.enabled),
                });
            } else {
                await createMutation.mutateAsync(buildPayload(fields, true));
            }
            close();
        } catch (e) {
            setFormError(`Could not save the bot: ${errorMessage(e, "unknown error")}`);
        }
    }

    function save() {
        runSave().catch(() => undefined);
    }

    async function runToggleEnabled(bot: Chatbot, enabled: boolean) {
        setError("");
        setTogglingID(bot.id);
        try {
            await updateMutation.mutateAsync({ id: bot.id, data: buildPayload(fieldsFromBot(bot), enabled) });
        } catch (e) {
            const direction = enabled ? "on" : "off";

            setError(`Could not switch ${bot.display_name} ${direction}: ${errorMessage(e, "unknown error")}`);
        } finally {
            setTogglingID(null);
        }
    }

    function toggleEnabled(bot: Chatbot, enabled: boolean) {
        runToggleEnabled(bot, enabled).catch(() => undefined);
    }

    function requestDelete(bot: Chatbot) {
        setPendingDelete(bot);
    }

    function cancelDelete() {
        setPendingDelete(null);
    }

    async function runDelete(bot: Chatbot) {
        setError("");
        try {
            await deleteMutation.mutateAsync(bot.id);
        } catch (e) {
            setError(`Could not delete ${bot.display_name}: ${errorMessage(e, "unknown error")}`);
        }
    }

    function confirmDelete() {
        const bot = pendingDelete;
        if (!bot) {
            return;
        }

        setPendingDelete(null);
        runDelete(bot).catch(() => undefined);
    }

    return {
        error,
        formError,
        open,
        editingBot,
        fields,
        usernameCheck,
        usernameKnownTaken,
        saving,
        togglingID,
        pendingDelete,
        setField,
        checkUsername,
        openCreate,
        openEdit,
        close,
        save,
        toggleEnabled,
        requestDelete,
        confirmDelete,
        cancelDelete,
    };
}
