import { useState } from "react";
import { Button } from "../../Button/Button";
import { Input } from "../../Input/Input";
import { Modal } from "../../Modal/Modal";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { ToggleSwitch } from "../../ToggleSwitch/ToggleSwitch";
import type {
    BannedWordAction,
    BannedWordMatchMode,
    BannedWordRule,
    BotsWillBeKickedResponse,
    ChatRoom,
    CreateBannedWordRequest,
    UpdateGroupRoomRequest,
    User,
} from "../../../types/api";
import { ApiError } from "../../../api/client";
import { useChatRoomBannedWords, useChatRoomBans } from "../../../api/queries/chat";
import {
    useCreateChatRoomBannedWord,
    useDeleteChatRoomBannedWord,
    useUnbanChatRoomMember,
    useUpdateChatRoom,
    useUpdateChatRoomBannedWord,
} from "../../../api/mutations/chat";
import {
    addRoomTags,
    finaliseRoomTags,
    isRoomTagCommitKey,
    MAX_ROOM_TAGS,
    removeRoomTag,
} from "../../../utils/roomTags";
import { formatFullDateTime } from "../../../utils/time";
import styles from "./RoomModerationDialog.module.css";

interface RoomModerationDialogProps {
    isOpen: boolean;
    room: ChatRoom;
    onClose: () => void;
    onSaved: (room: ChatRoom) => void;
}

type Tab = "bans" | "words" | "room";

type PendingConfirm = { kind: "public" } | { kind: "bots"; bots: User[] };

function formatDate(s: string): string {
    return formatFullDateTime(s, "en-GB");
}

function botsFromError(err: unknown): User[] | null {
    if (!(err instanceof ApiError) || err.status !== 409) {
        return null;
    }

    const body = err.body as BotsWillBeKickedResponse | null;
    if (!body || body.code !== "bots_will_be_kicked") {
        return null;
    }

    return body.bots ?? [];
}

function botLabel(bot: User): string {
    return bot.display_name?.trim() ? bot.display_name : bot.username;
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

export function RoomModerationDialog({ isOpen, room, onClose, onSaved }: RoomModerationDialogProps) {
    const roomId = room.id;
    const canEditRoom = room.type === "group" && !room.is_system;
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

    const updateRoomMutation = useUpdateChatRoom(roomId);
    const [roomName, setRoomName] = useState(room.name);
    const [roomDescription, setRoomDescription] = useState(room.description);
    const [roomTags, setRoomTags] = useState<string[]>(room.tags ?? []);
    const [roomTagInput, setRoomTagInput] = useState("");
    const [roomIsPublic, setRoomIsPublic] = useState(room.is_public);
    const [roomIsRP, setRoomIsRP] = useState(room.is_rp);
    const [roomSaving, setRoomSaving] = useState(false);
    const [pendingConfirm, setPendingConfirm] = useState<PendingConfirm | null>(null);

    const [openInstance, setOpenInstance] = useState(isOpen ? 1 : 0);
    const [prevIsOpen, setPrevIsOpen] = useState(isOpen);
    if (isOpen !== prevIsOpen) {
        setPrevIsOpen(isOpen);
        if (isOpen) {
            setOpenInstance(n => n + 1);
        }
    }

    const [seededForOpenInstance, setSeededForOpenInstance] = useState(0);
    if (seededForOpenInstance !== openInstance && isOpen) {
        setSeededForOpenInstance(openInstance);
        setRoomName(room.name);
        setRoomDescription(room.description);
        setRoomTags(room.tags ?? []);
        setRoomTagInput("");
        setRoomIsPublic(room.is_public);
        setRoomIsRP(room.is_rp);
        setPendingConfirm(null);
        setError("");
        setTab("bans");
    }

    function commitRoomTagInput() {
        if (!roomTagInput) {
            return;
        }

        setRoomTags(prev => addRoomTags(prev, roomTagInput));
        setRoomTagInput("");
    }

    function handleRoomTagKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
        if (isRoomTagCommitKey(e.key)) {
            e.preventDefault();
            commitRoomTagInput();
            return;
        }
        if (e.key === "Backspace" && roomTagInput === "" && roomTags.length > 0) {
            e.preventDefault();
            setRoomTags(prev => prev.slice(0, -1));
        }
    }

    async function submitRoom(confirmBotRemoval: boolean) {
        if (!roomName.trim() || roomSaving) {
            return;
        }

        const payload: UpdateGroupRoomRequest = {
            name: roomName.trim(),
            description: roomDescription.trim(),
            tags: finaliseRoomTags(roomTags, roomTagInput),
            is_public: roomIsPublic,
            is_rp: roomIsRP,
            confirm_bot_removal: confirmBotRemoval,
        };

        setRoomSaving(true);
        setError("");
        try {
            const updated = await updateRoomMutation.mutateAsync(payload);
            setPendingConfirm(null);
            onSaved(updated);
            onClose();
        } catch (e) {
            const bots = botsFromError(e);
            if (bots) {
                setPendingConfirm({ kind: "bots", bots });
                return;
            }

            setPendingConfirm(null);
            setError(e instanceof Error ? e.message : "Failed to update room");
        } finally {
            setRoomSaving(false);
        }
    }

    function handleSaveRoom() {
        if (roomIsPublic && !room.is_public) {
            setError("");
            setPendingConfirm({ kind: "public" });
            return;
        }

        submitRoom(false).catch(() => {});
    }

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
                {canEditRoom && (
                    <button
                        type="button"
                        className={`${styles.tab}${tab === "room" ? ` ${styles.tabActive}` : ""}`}
                        onClick={() => setTab("room")}
                    >
                        Room
                    </button>
                )}
            </div>

            {error && <div className={styles.error}>{error}</div>}

            {loading && tab !== "room" && <div className={styles.muted}>Loading...</div>}

            {canEditRoom && tab === "room" && (
                <div className={styles.section}>
                    <p className={styles.intro}>
                        Change how this room appears and who can find it. Only the host and site staff can edit these.
                    </p>

                    <div className={styles.field}>
                        <label className={styles.label}>Name</label>
                        <Input
                            fullWidth
                            type="text"
                            value={roomName}
                            onChange={e => setRoomName(e.target.value)}
                            placeholder="e.g. Higurashi book club"
                            maxLength={80}
                        />
                    </div>

                    <div className={styles.field}>
                        <label className={styles.label}>Description (optional)</label>
                        <Input
                            fullWidth
                            type="text"
                            value={roomDescription}
                            onChange={e => setRoomDescription(e.target.value)}
                            placeholder="What's the room about?"
                            maxLength={500}
                        />
                    </div>

                    <div className={styles.field}>
                        <label className={styles.label}>Tags (optional)</label>
                        {roomTags.length > 0 && (
                            <div className={styles.tagBar}>
                                {roomTags.map(t => (
                                    <button
                                        key={t}
                                        type="button"
                                        className={styles.tagChip}
                                        onClick={() => setRoomTags(prev => removeRoomTag(prev, t))}
                                    >
                                        #{t} ✕
                                    </button>
                                ))}
                            </div>
                        )}
                        <Input
                            fullWidth
                            type="text"
                            placeholder={`Type a tag and press Enter or comma (max ${MAX_ROOM_TAGS})`}
                            value={roomTagInput}
                            onChange={e => setRoomTagInput(e.target.value)}
                            onKeyDown={handleRoomTagKeyDown}
                            onBlur={commitRoomTagInput}
                            disabled={roomTags.length >= MAX_ROOM_TAGS}
                        />
                    </div>

                    <ToggleSwitch
                        enabled={roomIsPublic}
                        onChange={setRoomIsPublic}
                        label="Public"
                        description="Public rooms appear in Browse and anyone can join. Private rooms are invite-only."
                        disabled={roomSaving}
                    />
                    <ToggleSwitch
                        enabled={roomIsRP}
                        onChange={setRoomIsRP}
                        label="Roleplay (RP)"
                        description="Mark this as a roleplay room. Turning this off removes any bots in the room."
                        disabled={roomSaving}
                    />

                    {pendingConfirm?.kind === "public" && (
                        <div className={styles.confirm}>
                            <p className={styles.confirmText}>
                                Making this room public lets anyone join and read everything ever said in it, including
                                messages sent while it was private. Are you sure?
                            </p>
                            <div className={styles.confirmActions}>
                                <Button
                                    variant="secondary"
                                    size="small"
                                    onClick={() => setPendingConfirm(null)}
                                    disabled={roomSaving}
                                >
                                    Cancel
                                </Button>
                                <Button
                                    variant="danger"
                                    size="small"
                                    onClick={() => {
                                        submitRoom(false).catch(() => {});
                                    }}
                                    disabled={roomSaving}
                                >
                                    {roomSaving ? "Saving..." : "Make it public"}
                                </Button>
                            </div>
                        </div>
                    )}

                    {pendingConfirm?.kind === "bots" && (
                        <div className={styles.confirm}>
                            <p className={styles.confirmText}>Turning roleplay off removes these bots from the room:</p>
                            <ul className={styles.confirmList}>
                                {pendingConfirm.bots.map(b => (
                                    <li key={b.id}>{botLabel(b)}</li>
                                ))}
                            </ul>
                            <div className={styles.confirmActions}>
                                <Button
                                    variant="secondary"
                                    size="small"
                                    onClick={() => setPendingConfirm(null)}
                                    disabled={roomSaving}
                                >
                                    Cancel
                                </Button>
                                <Button
                                    variant="danger"
                                    size="small"
                                    onClick={() => {
                                        submitRoom(true).catch(() => {});
                                    }}
                                    disabled={roomSaving}
                                >
                                    {roomSaving ? "Saving..." : "Remove them and save"}
                                </Button>
                            </div>
                        </div>
                    )}

                    <div className={styles.formActions}>
                        <Button variant="ghost" size="small" onClick={onClose} disabled={roomSaving}>
                            Cancel
                        </Button>
                        <Button
                            variant="primary"
                            size="small"
                            onClick={handleSaveRoom}
                            disabled={roomSaving || !roomName.trim() || pendingConfirm !== null}
                        >
                            {roomSaving ? "Saving..." : "Save changes"}
                        </Button>
                    </div>
                </div>
            )}

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
