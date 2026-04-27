import { useState } from "react";
import { Button } from "../../Button/Button";
import { Input } from "../../Input/Input";
import { Modal } from "../../Modal/Modal";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import type {
    BannedWordAction,
    BannedWordMatchMode,
    BannedWordRule,
    CreateBannedWordRequest,
} from "../../../types/api";
import { useChatRoomBannedWords, useChatRoomBans } from "../../../api/queries/chat";
import {
    useCreateChatRoomBannedWord,
    useDeleteChatRoomBannedWord,
    useUnbanChatRoomMember,
    useUpdateChatRoomBannedWord,
} from "../../../api/mutations/chat";
import { formatFullDateTime } from "../../../utils/time";
import styles from "./RoomModerationDialog.module.css";

interface RoomModerationDialogProps {
    isOpen: boolean;
    roomId: string;
    onClose: () => void;
}

type Tab = "bans" | "words";

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

export function RoomModerationDialog({ isOpen, roomId, onClose }: RoomModerationDialogProps) {
    const [tab, setTab] = useState<Tab>("bans");
    const bansQuery = useChatRoomBans(roomId, isOpen);
    const rulesQuery = useChatRoomBannedWords(roomId, isOpen);
    const bans = bansQuery.bans;
    const rules = rulesQuery.rules;
    const loading = bansQuery.loading || rulesQuery.loading;
    const refreshBans = bansQuery.refresh;
    const refreshRules = rulesQuery.refresh;
    const unbanMutation = useUnbanChatRoomMember(roomId);
    const createWordMutation = useCreateChatRoomBannedWord(roomId);
    const updateWordMutation = useUpdateChatRoomBannedWord(roomId);
    const deleteWordMutation = useDeleteChatRoomBannedWord(roomId);

    const [error, setError] = useState("");
    const [pattern, setPattern] = useState("");
    const [mode, setMode] = useState<BannedWordMatchMode>("substring");
    const [caseSensitive, setCaseSensitive] = useState(false);
    const [action, setAction] = useState<BannedWordAction>("delete");
    const [saving, setSaving] = useState(false);
    const [busyId, setBusyId] = useState<string | null>(null);
    const [editingId, setEditingId] = useState<string | null>(null);

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

    async function handleUnban(userId: string) {
        setBusyId(userId);
        try {
            await unbanMutation.mutateAsync(userId);
            await refreshBans();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to unban");
        } finally {
            setBusyId(null);
        }
    }

    async function handleSaveRule() {
        if (!pattern.trim() || saving || regexError) {
            return;
        }
        setSaving(true);
        setError("");
        try {
            const req: CreateBannedWordRequest = {
                pattern: pattern.trim(),
                match_mode: mode,
                case_sensitive: caseSensitive,
                action,
            };
            if (editingId) {
                await updateWordMutation.mutateAsync({ ruleId: editingId, req });
            } else {
                await createWordMutation.mutateAsync(req);
            }
            resetForm();
            await refreshRules();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to save rule");
        } finally {
            setSaving(false);
        }
    }

    async function handleDeleteRule(rule: BannedWordRule) {
        if (rule.scope !== "room") {
            return;
        }
        if (!window.confirm(`Remove local rule for "${rule.pattern}"?`)) {
            return;
        }
        setBusyId(rule.id);
        try {
            await deleteWordMutation.mutateAsync(rule.id);
            await refreshRules();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to delete rule");
        } finally {
            setBusyId(null);
        }
    }

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Room moderation">
            <div className={styles.tabs}>
                <button
                    type="button"
                    className={`${styles.tab}${tab === "bans" ? ` ${styles.tabActive}` : ""}`}
                    onClick={() => setTab("bans")}
                >
                    Bans ({bans.length})
                </button>
                <button
                    type="button"
                    className={`${styles.tab}${tab === "words" ? ` ${styles.tabActive}` : ""}`}
                    onClick={() => setTab("words")}
                >
                    Banned words ({rules.length})
                </button>
            </div>

            {error && <div className={styles.error}>{error}</div>}

            {loading && <div className={styles.muted}>Loading...</div>}

            {!loading && tab === "bans" && (
                <div className={styles.section}>
                    {bans.length === 0 ? (
                        <div className={styles.muted}>No bans in this room.</div>
                    ) : (
                        <ul className={styles.list}>
                            {bans.map(b => (
                                <li key={b.user.id} className={styles.banRow}>
                                    <div className={styles.banMain}>
                                        <ProfileLink user={b.user} size="small" />
                                        <span className={styles.banDate}>{formatDate(b.created_at)}</span>
                                    </div>
                                    {b.reason && <div className={styles.banReason}>Reason: {b.reason}</div>}
                                    {b.banned_by && (
                                        <div className={styles.banBy}>
                                            By <ProfileLink user={b.banned_by} size="small" />
                                        </div>
                                    )}
                                    <div className={styles.banActions}>
                                        <Button
                                            variant="secondary"
                                            size="small"
                                            disabled={busyId === b.user.id}
                                            onClick={() => handleUnban(b.user.id)}
                                        >
                                            {busyId === b.user.id ? "..." : "Unban"}
                                        </Button>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
            )}

            {!loading && tab === "words" && (
                <div className={styles.section}>
                    <p className={styles.intro}>
                        Local rules apply only to this room. Global rules (set by site admins) are shown for awareness
                        and cannot be edited here. Hosts, site moderators, and admins are immune from all rules.
                    </p>
                    <div className={styles.form}>
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
                                Mode
                                <select
                                    className={styles.select}
                                    value={mode}
                                    onChange={e => setMode(e.target.value as BannedWordMatchMode)}
                                >
                                    <option value="substring">Substring</option>
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
                                <Button variant="secondary" size="small" onClick={resetForm} disabled={saving}>
                                    Cancel
                                </Button>
                            )}
                            <Button
                                variant="primary"
                                size="small"
                                onClick={handleSaveRule}
                                disabled={saving || !pattern.trim() || !!regexError}
                            >
                                {saving ? "Saving..." : editingId ? "Save changes" : "Add rule"}
                            </Button>
                        </div>
                    </div>

                    {rules.length === 0 ? (
                        <div className={styles.muted}>No rules apply in this room.</div>
                    ) : (
                        <ul className={styles.list}>
                            {rules.map(rule => (
                                <li
                                    key={rule.id}
                                    className={`${styles.ruleRow}${rule.scope === "global" ? ` ${styles.ruleGlobal}` : ""}`}
                                >
                                    <div className={styles.ruleMain}>
                                        <span className={styles.mono}>{rule.pattern}</span>
                                        <span className={styles.metaPill}>{rule.match_mode}</span>
                                        {rule.case_sensitive && <span className={styles.metaPill}>case-sensitive</span>}
                                        <span
                                            className={rule.action === "kick" ? styles.badgeKick : styles.badgeDelete}
                                        >
                                            {rule.action}
                                        </span>
                                        <span
                                            className={rule.scope === "global" ? styles.scopeGlobal : styles.scopeRoom}
                                        >
                                            {rule.scope}
                                        </span>
                                    </div>
                                    {rule.scope === "room" && (
                                        <div className={styles.ruleActions}>
                                            <Button
                                                variant="secondary"
                                                size="small"
                                                disabled={saving || busyId === rule.id}
                                                onClick={() => startEdit(rule)}
                                            >
                                                Edit
                                            </Button>
                                            <Button
                                                variant="danger"
                                                size="small"
                                                disabled={busyId === rule.id}
                                                onClick={() => handleDeleteRule(rule)}
                                            >
                                                {busyId === rule.id ? "..." : "Remove"}
                                            </Button>
                                        </div>
                                    )}
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
            )}
        </Modal>
    );
}
