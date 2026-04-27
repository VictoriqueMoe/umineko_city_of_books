import { useState } from "react";
import { useGlobalBannedWords } from "../../api/queries/admin";
import {
    useCreateGlobalBannedWord,
    useDeleteGlobalBannedWord,
    useUpdateGlobalBannedWord,
} from "../../api/mutations/admin";
import type { BannedWordAction, BannedWordMatchMode, BannedWordRule } from "../../types/api";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { formatFullDateTime } from "../../utils/time";
import styles from "./AdminBannedWords.module.css";

function formatDate(s: string): string {
    return formatFullDateTime(s, "en-GB");
}

function validateRegex(pattern: string, mode: BannedWordMatchMode): string {
    if (mode !== "regex") {
        return "";
    }
    try {
        new RegExp(pattern);
        return "";
    } catch (e) {
        return e instanceof Error ? e.message : "Invalid regex";
    }
}

export function AdminBannedWords() {
    usePageTitle("Admin - Banned Words");
    const { rules, loading } = useGlobalBannedWords();
    const createMutation = useCreateGlobalBannedWord();
    const updateMutation = useUpdateGlobalBannedWord();
    const deleteMutation = useDeleteGlobalBannedWord();
    const [error, setError] = useState("");
    const [pattern, setPattern] = useState("");
    const [mode, setMode] = useState<BannedWordMatchMode>("substring");
    const [caseSensitive, setCaseSensitive] = useState(false);
    const [action, setAction] = useState<BannedWordAction>("delete");
    const [removing, setRemoving] = useState<string | null>(null);
    const [editingId, setEditingId] = useState<string | null>(null);

    const saving = createMutation.isPending || updateMutation.isPending;

    function resetForm() {
        setPattern("");
        setMode("substring");
        setCaseSensitive(false);
        setAction("delete");
        setEditingId(null);
    }

    function startEdit(rule: BannedWordRule) {
        setEditingId(rule.id);
        setPattern(rule.pattern);
        setMode(rule.match_mode);
        setCaseSensitive(rule.case_sensitive);
        setAction(rule.action);
        setError("");
    }

    const regexError = validateRegex(pattern, mode);

    async function handleSave() {
        if (!pattern.trim() || saving) {
            return;
        }
        if (regexError) {
            setError(regexError);
            return;
        }
        setError("");
        try {
            const payload = {
                pattern: pattern.trim(),
                match_mode: mode,
                case_sensitive: caseSensitive,
                action,
            };
            if (editingId) {
                await updateMutation.mutateAsync({ ruleId: editingId, req: payload });
            } else {
                await createMutation.mutateAsync(payload);
            }
            resetForm();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to save rule");
        }
    }

    async function handleRemove(rule: BannedWordRule) {
        if (!window.confirm(`Remove global rule for pattern "${rule.pattern}"?`)) {
            return;
        }
        setRemoving(rule.id);
        try {
            await deleteMutation.mutateAsync(rule.id);
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to remove rule");
        } finally {
            setRemoving(null);
        }
    }

    if (loading) {
        return <div className={styles.loading}>Loading rules...</div>;
    }

    return (
        <div className={styles.page}>
            <div className={styles.header}>
                <h1 className={styles.title}>Banned Words</h1>
            </div>

            <p className={styles.intro}>
                Global rules apply to every chat room. When a user sends a message matching a rule, the send is
                rejected. Rules with the <strong>Kick</strong> action also ban the sender from that room. Room hosts and
                site staff are immune from all rules. Automated hits are recorded in the audit log under the
                <code> chat_word_filter_*</code> actions.
            </p>

            <div className={styles.addCard}>
                <label className={styles.fieldLabel}>
                    Pattern
                    <Input
                        type="text"
                        value={pattern}
                        onChange={e => setPattern(e.target.value)}
                        placeholder="Word or regex to block"
                        fullWidth
                    />
                </label>
                <div className={styles.row}>
                    <label className={styles.fieldLabel}>
                        Match mode
                        <select
                            className={styles.select}
                            value={mode}
                            onChange={e => setMode(e.target.value as BannedWordMatchMode)}
                        >
                            <option value="substring">Substring (any occurrence)</option>
                            <option value="whole_word">Whole word</option>
                            <option value="regex">Regex</option>
                        </select>
                    </label>
                    <label className={styles.fieldLabel}>
                        Action
                        <select
                            className={styles.select}
                            value={action}
                            onChange={e => setAction(e.target.value as BannedWordAction)}
                        >
                            <option value="delete">Delete message</option>
                            <option value="kick">Kick</option>
                        </select>
                    </label>
                    <label className={styles.checkboxRow}>
                        <input
                            type="checkbox"
                            checked={caseSensitive}
                            onChange={e => setCaseSensitive(e.target.checked)}
                        />
                        <span>Case sensitive</span>
                    </label>
                </div>
                {regexError && <div className={styles.regexError}>Regex error: {regexError}</div>}
                <div className={styles.formActions}>
                    {editingId && (
                        <Button variant="secondary" onClick={resetForm} disabled={saving}>
                            Cancel
                        </Button>
                    )}
                    <Button variant="primary" onClick={handleSave} disabled={saving || !pattern.trim() || !!regexError}>
                        {saving ? "Saving..." : editingId ? "Save changes" : "Add rule"}
                    </Button>
                </div>
            </div>

            {error && <div className={styles.error}>{error}</div>}

            {rules.length === 0 ? (
                <div className={styles.empty}>No global rules yet.</div>
            ) : (
                <table className={styles.table}>
                    <thead>
                        <tr>
                            <th>Pattern</th>
                            <th>Mode</th>
                            <th>Case</th>
                            <th>Action</th>
                            <th>Added by</th>
                            <th>Added</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        {rules.map(rule => (
                            <tr key={rule.id}>
                                <td className={styles.mono}>{rule.pattern}</td>
                                <td>{rule.match_mode}</td>
                                <td>{rule.case_sensitive ? "Yes" : "No"}</td>
                                <td>
                                    <span className={rule.action === "kick" ? styles.badgeKick : styles.badgeDelete}>
                                        {rule.action}
                                    </span>
                                </td>
                                <td>{rule.created_by_name || "\u2014"}</td>
                                <td className={styles.date}>{formatDate(rule.created_at)}</td>
                                <td className={styles.actions}>
                                    <Button
                                        variant="secondary"
                                        size="small"
                                        onClick={() => startEdit(rule)}
                                        disabled={saving}
                                    >
                                        Edit
                                    </Button>
                                    <Button
                                        variant="danger"
                                        size="small"
                                        onClick={() => handleRemove(rule)}
                                        disabled={removing === rule.id}
                                    >
                                        {removing === rule.id ? "..." : "Remove"}
                                    </Button>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}
        </div>
    );
}
