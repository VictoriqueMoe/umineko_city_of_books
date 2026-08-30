import { useState } from "react";
import { errorMessage } from "../utils/errorMessage";
import { useCreateChatbotBasePrompt, useDeleteChatbotBasePrompt, useUpdateChatbotBasePrompt } from "./mutations/admin";
import type { ChatbotBasePrompt } from "../types/api";

export interface ChatbotBasePromptEditor {
    error: string;
    open: boolean;
    editing: ChatbotBasePrompt | null;
    name: string;
    prompt: string;
    saving: boolean;
    pendingDelete: ChatbotBasePrompt | null;
    setName: (name: string) => void;
    setPrompt: (prompt: string) => void;
    openCreate: () => void;
    openEdit: (base: ChatbotBasePrompt) => void;
    close: () => void;
    save: () => void;
    requestDelete: (base: ChatbotBasePrompt) => void;
    confirmDelete: () => void;
    cancelDelete: () => void;
}

export function useChatbotBasePromptEditor(): ChatbotBasePromptEditor {
    const createMutation = useCreateChatbotBasePrompt();
    const updateMutation = useUpdateChatbotBasePrompt();
    const deleteMutation = useDeleteChatbotBasePrompt();

    const [error, setError] = useState("");
    const [open, setOpen] = useState(false);
    const [editing, setEditing] = useState<ChatbotBasePrompt | null>(null);
    const [name, setName] = useState("");
    const [prompt, setPrompt] = useState("");
    const [pendingDelete, setPendingDelete] = useState<ChatbotBasePrompt | null>(null);

    const saving = createMutation.isPending || updateMutation.isPending;

    function openCreate() {
        setEditing(null);
        setName("");
        setPrompt("");
        setError("");
        setOpen(true);
    }

    function openEdit(base: ChatbotBasePrompt) {
        setEditing(base);
        setName(base.name);
        setPrompt(base.prompt);
        setError("");
        setOpen(true);
    }

    function close() {
        setOpen(false);
        setEditing(null);
        setError("");
    }

    async function runSave() {
        if (name.trim() === "" || prompt.trim() === "") {
            setError("A base prompt needs a name and some text.");

            return;
        }

        const payload = { name: name.trim(), prompt };

        try {
            if (editing) {
                await updateMutation.mutateAsync({ id: editing.id, data: payload });
            } else {
                await createMutation.mutateAsync(payload);
            }

            close();
        } catch (e) {
            setError(errorMessage(e, "unknown error"));
        }
    }

    function save() {
        runSave().catch(() => undefined);
    }

    function requestDelete(base: ChatbotBasePrompt) {
        if (base.bot_count > 0) {
            setError(`${base.name} is still used by ${base.bot_count} bot(s). Unassign them first.`);

            return;
        }

        setPendingDelete(base);
    }

    function cancelDelete() {
        setPendingDelete(null);
    }

    async function runDelete(base: ChatbotBasePrompt) {
        try {
            await deleteMutation.mutateAsync(base.id);
            setError("");
        } catch (e) {
            setError(errorMessage(e, "unknown error"));
        }
    }

    function confirmDelete() {
        const base = pendingDelete;
        if (!base) {
            return;
        }

        setPendingDelete(null);
        runDelete(base).catch(() => undefined);
    }

    return {
        error,
        open,
        editing,
        name,
        prompt,
        saving,
        pendingDelete,
        setName,
        setPrompt,
        openCreate,
        openEdit,
        close,
        save,
        requestDelete,
        confirmDelete,
        cancelDelete,
    };
}
