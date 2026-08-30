import { useId } from "react";
import { useChatbotBasePromptEditor } from "../../hooks/useChatbotBasePromptEditor";
import { estimateTokens } from "../../domain/chatbotUsage";
import { Button } from "../../components/Button/Button";
import { ConfirmDialog } from "../../components/ConfirmDialog/ConfirmDialog";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import { TextArea } from "../../components/TextArea/TextArea";
import type { ChatbotBasePrompt } from "../../types/api";
import styles from "./AdminChatbots.module.css";

interface BasePromptSectionProps {
    basePrompts: ChatbotBasePrompt[];
}

export function BasePromptSection({ basePrompts }: BasePromptSectionProps) {
    const baseID = useId();
    const editor = useChatbotBasePromptEditor();

    function fieldID(name: string) {
        return `${baseID}-${name}`;
    }

    return (
        <>
            <div className={styles.header}>
                <h2 className={styles.overridesTitle}>Base Prompts</h2>
                <Button variant="secondary" onClick={editor.openCreate}>
                    Create Base Prompt
                </Button>
            </div>

            <p className={styles.intro}>
                A base prompt is shared text that every bot extending it receives before its own personality, for the
                rules they all obey rather than anything one character does. Edit it once and every bot using it picks
                the change up immediately.
            </p>

            {editor.error && <div className={styles.error}>{editor.error}</div>}

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
                                    onClick={() => editor.openEdit(base)}
                                >
                                    Edit
                                </Button>
                                <Button
                                    variant="danger"
                                    size="small"
                                    aria-label={`Delete ${base.name}`}
                                    onClick={() => editor.requestDelete(base)}
                                >
                                    Delete
                                </Button>
                            </span>
                        </div>
                    ))}
                </div>
            )}

            <Modal
                isOpen={editor.open}
                onClose={editor.close}
                title={editor.editing ? "Edit Base Prompt" : "Create Base Prompt"}
            >
                <div className={styles.form}>
                    {editor.error && <div className={styles.error}>{editor.error}</div>}
                    <div className={styles.fieldLabel}>
                        <label htmlFor={fieldID("base-name")}>Name</label>
                        <Input
                            id={fieldID("base-name")}
                            value={editor.name}
                            onChange={e => editor.setName(e.target.value)}
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
                            value={editor.prompt}
                            onChange={e => editor.setPrompt(e.target.value)}
                            placeholder="You are a witch of the game boards, and this is the first half of your instructions..."
                            aria-describedby={fieldID("base-prompt-text-hint")}
                        />
                        <span id={fieldID("base-prompt-text-hint")} className={styles.fieldHint}>
                            This text is sent before the persona of every bot that extends it, so write only what they
                            all share: the hierarchy they obey, how the site works, and the rules none of them may
                            break. Anything that belongs to one character belongs in that bot's own prompt instead.
                            Roughly {estimateTokens(editor.prompt).toLocaleString()} tokens, charged on every reply from
                            every bot using it.
                        </span>
                    </div>
                    <div className={styles.formActions}>
                        <Button variant="ghost" size="small" onClick={editor.close}>
                            Cancel
                        </Button>
                        <Button
                            variant="primary"
                            size="small"
                            onClick={editor.save}
                            disabled={editor.saving || editor.name.trim() === "" || editor.prompt.trim() === ""}
                        >
                            {editor.saving ? "Saving..." : "Save"}
                        </Button>
                    </div>
                </div>
            </Modal>

            <ConfirmDialog
                open={editor.pendingDelete !== null}
                title="Confirm Delete"
                body={`Delete the base prompt "${editor.pendingDelete?.name ?? ""}"?`}
                confirmLabel="Delete"
                destructive
                onConfirm={editor.confirmDelete}
                onCancel={editor.cancelDelete}
            />
        </>
    );
}
