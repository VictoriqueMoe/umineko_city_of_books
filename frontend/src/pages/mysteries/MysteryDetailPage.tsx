import { type MouseEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import type { MysteryAttachment, MysteryAttempt, MysteryClue, PostComment } from "../../types/api";
import { useMystery } from "../../api/queries/mystery";
import {
    useAddMysteryClue,
    useCreateMysteryAttempt,
    useCreateMysteryComment,
    useDeleteMystery,
    useDeleteMysteryAttachment,
    useDeleteMysteryClue,
    useDeleteMysteryComment,
    useLikeMysteryComment,
    useSetMysteryGmAway,
    useSetMysteryPaused,
    useUnlikeMysteryComment,
    useUpdateMysteryClue,
    useUpdateMysteryComment,
    useUploadMysteryAttachment,
    useUploadMysteryCommentMedia,
} from "../../api/mutations/mystery";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import { useThrottled } from "../../hooks/useThrottled";
import { can } from "../../utils/permissions";
import { renderRich } from "../../utils/richText";
import { parseServerDate } from "../../utils/time";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { relativeTime } from "../../utils/notifications";
import { CommentComposer } from "../../components/post/CommentComposer/CommentComposer";
import { CommentItem } from "../../components/post/CommentItem/CommentItem";
import { AttemptItem } from "./AttemptItem";
import { ShareButton } from "../../components/ShareButton/ShareButton";
import { ReportButton } from "../../components/ReportButton/ReportButton";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import styles from "./MysteryPages.module.css";

function ClueCopyBtn({ text }: { text: string }) {
    const [copied, setCopied] = useState(false);

    function handleCopy(e: MouseEvent) {
        e.stopPropagation();
        navigator.clipboard.writeText(text).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
        });
    }

    return (
        <button type="button" className={styles.clueCopy} onClick={handleCopy} title="Copy to clipboard">
            {copied ? (
                "\u2713"
            ) : (
                <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                >
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                </svg>
            )}
        </button>
    );
}

function PrivateCluesDisplay({
    clues,
    playerId,
    mysteryId,
    canEditClues,
    onAdded,
    title,
}: {
    clues: MysteryClue[];
    playerId: string;
    mysteryId: string;
    canEditClues: boolean;
    onAdded: () => void;
    title?: string;
}) {
    const storageKey = `mystery:${mysteryId}:private-clues:${playerId}:collapsed`;
    const [editingClueId, setEditingClueId] = useState<number | null>(null);
    const [editClueBody, setEditClueBody] = useState("");
    const [collapsed, setCollapsed] = useState<boolean>(() => {
        try {
            return window.localStorage.getItem(storageKey) === "1";
        } catch {
            return false;
        }
    });
    const deleteClueMutation = useDeleteMysteryClue(mysteryId);
    const updateClueMutation = useUpdateMysteryClue(mysteryId);
    const playerClues = clues.filter(c => c.player_id === playerId);

    function toggleCollapsed() {
        setCollapsed(prev => {
            const next = !prev;
            try {
                if (next) {
                    window.localStorage.setItem(storageKey, "1");
                } else {
                    window.localStorage.removeItem(storageKey);
                }
            } catch {
                // localStorage unavailable; in-memory state still works
            }
            return next;
        });
    }

    async function handleDeleteClue(clueId: number) {
        if (!window.confirm("Delete this red truth? This cannot be undone.")) {
            return;
        }
        await deleteClueMutation.mutateAsync(clueId);
        onAdded();
    }

    async function handleSaveClue(clueId: number) {
        if (!editClueBody.trim()) {
            return;
        }
        await updateClueMutation.mutateAsync({ clueId, body: editClueBody.trim() });
        setEditingClueId(null);
        onAdded();
    }

    if (playerClues.length === 0) {
        return null;
    }

    const heading = title ?? "Private Red Truths";

    return (
        <div style={{ padding: "0 0.5rem", marginBottom: "0.5rem" }}>
            <button
                type="button"
                onClick={toggleCollapsed}
                aria-expanded={!collapsed}
                style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "0.4rem",
                    background: "none",
                    border: "none",
                    color: "#ef9a9a",
                    cursor: "pointer",
                    fontSize: "0.8rem",
                    fontFamily: "inherit",
                    fontStyle: "italic",
                    padding: "0.2rem 0",
                    marginBottom: "0.25rem",
                }}
            >
                <span>{collapsed ? "\u25B6" : "\u25BC"}</span>
                <span>
                    {heading} ({playerClues.length})
                </span>
            </button>
            {!collapsed && (
                <div className={styles.cluesSection} style={{ marginBottom: "0.5rem" }}>
                    {playerClues.map(clue => (
                        <div key={clue.id} className={styles.clue} style={{ fontSize: "0.85rem" }}>
                            {editingClueId === clue.id ? (
                                <div style={{ display: "flex", gap: "0.4rem", alignItems: "center", flex: 1 }}>
                                    <input
                                        type="text"
                                        value={editClueBody}
                                        onChange={e => setEditClueBody(e.target.value)}
                                        onKeyDown={e => {
                                            if (e.key === "Enter") {
                                                handleSaveClue(clue.id);
                                            }
                                            if (e.key === "Escape") {
                                                setEditingClueId(null);
                                            }
                                        }}
                                        style={{
                                            flex: 1,
                                            background: "var(--bg-void)",
                                            border: "1px solid rgba(229, 57, 53, 0.3)",
                                            color: "#ef9a9a",
                                            padding: "0.3rem 0.5rem",
                                            borderRadius: "4px",
                                            fontSize: "0.8rem",
                                            fontFamily: "inherit",
                                            fontStyle: "italic",
                                        }}
                                        autoFocus
                                    />
                                    <Button variant="primary" size="small" onClick={() => handleSaveClue(clue.id)}>
                                        Save
                                    </Button>
                                    <Button variant="ghost" size="small" onClick={() => setEditingClueId(null)}>
                                        Cancel
                                    </Button>
                                </div>
                            ) : (
                                <>
                                    {clue.body}
                                    <span className={styles.clueActions}>
                                        {canEditClues && (
                                            <>
                                                <button
                                                    className={styles.clueActionBtn}
                                                    onClick={() => {
                                                        setEditingClueId(clue.id);
                                                        setEditClueBody(clue.body);
                                                    }}
                                                >
                                                    edit
                                                </button>
                                                <button
                                                    className={styles.clueActionBtn}
                                                    onClick={() => handleDeleteClue(clue.id)}
                                                >
                                                    delete
                                                </button>
                                            </>
                                        )}
                                        <ClueCopyBtn text={clue.body} />
                                    </span>
                                </>
                            )}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}

function PrivateClueInput({
    playerId,
    mysteryId,
    onAdded,
}: {
    playerId: string;
    mysteryId: string;
    onAdded: () => void;
}) {
    const [body, setBody] = useState("");
    const [adding, setAdding] = useState(false);
    const [added, setAdded] = useState(false);
    const addClueMutation = useAddMysteryClue(mysteryId);

    async function handleAdd() {
        if (!body.trim() || adding) {
            return;
        }
        setAdding(true);
        try {
            await addClueMutation.mutateAsync({ body: body.trim(), truthType: "red", playerId });
            setBody("");
            setAdded(true);
            setTimeout(() => setAdded(false), 2000);
            onAdded();
        } catch {
        } finally {
            setAdding(false);
        }
    }

    return (
        <div style={{ padding: "0 0.5rem", marginTop: "0.5rem" }}>
            <div style={{ display: "flex", gap: "0.4rem", alignItems: "center" }}>
                <input
                    type="text"
                    value={body}
                    onChange={e => setBody(e.target.value)}
                    placeholder="Private red truth for this player..."
                    onKeyDown={e => {
                        if (e.key === "Enter") {
                            handleAdd();
                        }
                    }}
                    style={{
                        flex: 1,
                        background: "var(--bg-void)",
                        border: "1px solid rgba(229, 57, 53, 0.3)",
                        color: "#ef9a9a",
                        padding: "0.35rem 0.6rem",
                        borderRadius: "4px",
                        fontSize: "0.8rem",
                        fontFamily: "inherit",
                        fontStyle: "italic",
                    }}
                />
                <Button variant="danger" size="small" onClick={handleAdd} disabled={!body.trim() || adding}>
                    {adding ? "..." : "Add private Red Truth"}
                </Button>
                {added && (
                    <span style={{ color: "#ef9a9a", fontSize: "0.8rem", fontStyle: "italic", whiteSpace: "nowrap" }}>
                        Red truth added
                    </span>
                )}
            </div>
        </div>
    );
}

function findWinningAttempt(attempts: MysteryAttempt[]): MysteryAttempt | null {
    for (const a of attempts) {
        if (a.is_winner) {
            return a;
        }
        if (a.replies && a.replies.length > 0) {
            const nested = findWinningAttempt(a.replies);
            if (nested) {
                return nested;
            }
        }
    }
    return null;
}

export function MysteryDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const { addWSListener } = useNotifications();
    const mysteryId = id ?? "";
    const { mystery, loading, refresh } = useMystery(mysteryId);
    usePageTitle(mystery?.title ?? "Mystery");
    const createAttemptMutation = useCreateMysteryAttempt(mysteryId);
    const addClueMutation = useAddMysteryClue(mysteryId);
    const deleteMysteryMutation = useDeleteMystery();
    const setPausedMutation = useSetMysteryPaused(mysteryId);
    const setGmAwayMutation = useSetMysteryGmAway(mysteryId);
    const uploadAttachmentMutation = useUploadMysteryAttachment(mysteryId);
    const deleteAttachmentMutation = useDeleteMysteryAttachment(mysteryId);
    const createCommentMutation = useCreateMysteryComment(mysteryId);
    const updateCommentMutation = useUpdateMysteryComment(mysteryId);
    const deleteCommentMutation = useDeleteMysteryComment(mysteryId);
    const likeCommentMutation = useLikeMysteryComment(mysteryId);
    const unlikeCommentMutation = useUnlikeMysteryComment(mysteryId);
    const uploadCommentMediaMutation = useUploadMysteryCommentMedia(mysteryId);
    const likeCommentFn = useCallback(
        (commentId: string) => likeCommentMutation.mutateAsync(commentId).then(() => {}),
        [likeCommentMutation],
    );
    const unlikeCommentFn = useCallback(
        (commentId: string) => unlikeCommentMutation.mutateAsync(commentId).then(() => {}),
        [unlikeCommentMutation],
    );
    const deleteCommentFn = useCallback(
        (commentId: string) => deleteCommentMutation.mutateAsync(commentId).then(() => {}),
        [deleteCommentMutation],
    );
    const updateCommentFn = useCallback(
        (commentId: string, body: string) => updateCommentMutation.mutateAsync({ id: commentId, body }).then(() => {}),
        [updateCommentMutation],
    );
    const createCommentFn = useCallback(
        (_postId: string, body: string, parentId?: string) =>
            createCommentMutation.mutateAsync({ body, parentId }).then(c => ({ id: c.id })),
        [createCommentMutation],
    );
    const uploadCommentMediaFn = useCallback(
        (commentId: string, file: File) => uploadCommentMediaMutation.mutateAsync({ commentId, file }),
        [uploadCommentMediaMutation],
    );
    const hash = location.hash;
    const highlightedAttempt = hash.startsWith("#attempt-") ? hash.replace("#attempt-", "") : null;
    const [attemptBody, setAttemptBody] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [collapsedPlayers, setCollapsedPlayers] = useState<Set<string>>(new Set());
    const [unreadPlayers, setUnreadPlayers] = useState<Set<string>>(new Set());
    const [newClueBody, setNewClueBody] = useState("");
    const [addingClue, setAddingClue] = useState(false);
    const [uploadingAttachment, setUploadingAttachment] = useState(false);
    const [attachmentError, setAttachmentError] = useState("");
    const attachmentInputRef = useRef<HTMLInputElement>(null);

    function togglePlayerCollapse(authorId: string) {
        setCollapsedPlayers(prev => {
            const next = new Set(prev);
            if (next.has(authorId)) {
                next.delete(authorId);
            } else {
                next.add(authorId);
            }
            return next;
        });
    }

    function markPlayerRead(authorId: string) {
        setUnreadPlayers(prev => {
            if (!prev.has(authorId)) {
                return prev;
            }
            const next = new Set(prev);
            next.delete(authorId);
            return next;
        });
    }

    function jumpToPlayer(authorId: string) {
        markPlayerRead(authorId);
        setCollapsedPlayers(prev => {
            if (!prev.has(authorId)) {
                return prev;
            }
            const next = new Set(prev);
            next.delete(authorId);
            return next;
        });
        requestAnimationFrame(() => {
            const el = document.getElementById(`player-group-${authorId}`);
            if (el) {
                el.scrollIntoView({ behavior: "smooth", block: "start" });
            }
        });
    }

    const winningAttempt = useMemo(
        () => (mystery?.solved ? findWinningAttempt(mystery.attempts ?? []) : null),
        [mystery],
    );

    const groupedAttempts = useMemo(() => {
        if (!mystery) {
            return [];
        }
        const groups = new Map<string, { author: MysteryAttempt["author"]; attempts: MysteryAttempt[] }>();
        for (const a of mystery.attempts ?? []) {
            const existing = groups.get(a.author.id);
            if (existing) {
                existing.attempts.push(a);
            } else {
                groups.set(a.author.id, { author: a.author, attempts: [a] });
            }
        }
        const result = Array.from(groups.values());
        if (mystery.free_for_all && user) {
            result.sort((a, b) => {
                if (a.author.id === user.id) {
                    return -1;
                }
                if (b.author.id === user.id) {
                    return 1;
                }
                return 0;
            });
        }
        return result;
    }, [mystery, user]);

    const fetchMystery = useCallback(() => {
        void refresh();
    }, [refresh]);

    const throttledFetchMystery = useThrottled(fetchMystery, 200);

    useEffect(() => {
        if (!id) {
            return;
        }
        return addWSListener(msg => {
            if (msg.type === "mystery_solved") {
                const data = msg.data as { mystery_id?: string; attempt_id?: string };
                if (data.mystery_id !== id) {
                    return;
                }
                throttledFetchMystery();
                if (data.attempt_id) {
                    requestAnimationFrame(() => {
                        const el = document.getElementById(`attempt-${data.attempt_id}`);
                        if (el) {
                            el.scrollIntoView({ behavior: "smooth", block: "center" });
                        }
                    });
                }
                return;
            }
            if (msg.type === "mystery_attempt_created") {
                const data = msg.data as { mystery_id?: string; author_id?: string };
                if (data.mystery_id !== id) {
                    return;
                }
                if (data.author_id && data.author_id !== user?.id) {
                    setUnreadPlayers(prev => {
                        const next = new Set(prev);
                        next.add(data.author_id as string);
                        return next;
                    });
                }
                throttledFetchMystery();
            }
            if (
                msg.type === "mystery_clue_added" ||
                msg.type === "mystery_clue_updated" ||
                msg.type === "mystery_paused" ||
                msg.type === "mystery_gm_away"
            ) {
                const data = msg.data as { mystery_id?: string };
                if (data.mystery_id === id) {
                    throttledFetchMystery();
                }
            }
        });
    }, [id, addWSListener, throttledFetchMystery, user?.id]);

    const [unreadComputedForMystery, setUnreadComputedForMystery] = useState<string | null>(null);
    if (mystery && id && unreadComputedForMystery !== id) {
        setUnreadComputedForMystery(id);
        const isGM = user?.id === mystery.author.id || user?.role === "super_admin";
        if (isGM && !mystery.solved) {
            const cursorRaw = localStorage.getItem(`mystery-read-cursor-${id}`);
            if (!cursorRaw) {
                localStorage.setItem(`mystery-read-cursor-${id}`, new Date().toISOString());
            } else {
                const cursorDate = parseServerDate(cursorRaw);
                const cursor = cursorDate ? cursorDate.getTime() : 0;
                const unread = new Set<string>();
                for (const a of mystery.attempts ?? []) {
                    const createdDate = parseServerDate(a.created_at);
                    const created = createdDate ? createdDate.getTime() : 0;
                    if (created > cursor && a.author.id !== user?.id) {
                        unread.add(a.author.id);
                    }
                }
                if (unread.size > 0) {
                    setUnreadPlayers(unread);
                }
            }
        }
    }

    useEffect(() => {
        if (!id) {
            return;
        }
        return () => {
            localStorage.setItem(`mystery-read-cursor-${id}`, new Date().toISOString());
        };
    }, [id]);

    useScrollToHash(!loading && !!mystery, highlightedAttempt ? `attempt-${highlightedAttempt}` : null);

    if (loading) {
        return <div className="loading">Investigating the mystery...</div>;
    }

    if (!mystery) {
        return <div className="empty-state">Mystery not found.</div>;
    }

    const isAuthor = user?.id === mystery.author.id;
    const canEdit = can(user?.role, "edit_any_theory");
    const canDelete = isAuthor || can(user?.role, "delete_any_theory");
    const canSeeAsGameMaster = isAuthor || user?.role === "super_admin";

    async function handleSubmitAttempt() {
        if (!attemptBody.trim() || submitting || !id) {
            return;
        }
        setSubmitting(true);
        try {
            await createAttemptMutation.mutateAsync({ body: attemptBody.trim() });
            setAttemptBody("");
            fetchMystery();
        } catch {
            // ignore
        } finally {
            setSubmitting(false);
        }
    }

    async function handleAddClue() {
        if (!newClueBody.trim() || addingClue || !id) {
            return;
        }
        setAddingClue(true);
        try {
            await addClueMutation.mutateAsync({ body: newClueBody.trim(), truthType: "red" });
            setNewClueBody("");
            fetchMystery();
        } catch {
            // ignore
        } finally {
            setAddingClue(false);
        }
    }

    async function handleDelete() {
        if (!window.confirm("Delete this mystery? This cannot be undone.")) {
            return;
        }
        await deleteMysteryMutation.mutateAsync(mystery!.id);
        navigate("/mysteries");
    }

    async function handleAttachmentUpload(e: React.ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (!file || uploadingAttachment || !id) {
            return;
        }
        setUploadingAttachment(true);
        setAttachmentError("");
        try {
            await uploadAttachmentMutation.mutateAsync(file);
            fetchMystery();
        } catch (err) {
            setAttachmentError(err instanceof Error ? err.message : "Failed to upload attachment");
        } finally {
            setUploadingAttachment(false);
            if (attachmentInputRef.current) {
                attachmentInputRef.current.value = "";
            }
        }
    }

    async function handleDeleteAttachment(attachment: MysteryAttachment) {
        if (!window.confirm(`Delete attachment "${attachment.file_name}"?`)) {
            return;
        }
        try {
            await deleteAttachmentMutation.mutateAsync(attachment.id);
            fetchMystery();
        } catch {}
    }

    function formatFileSize(bytes: number): string {
        if (bytes < 1024) {
            return `${bytes} B`;
        }
        if (bytes < 1024 * 1024) {
            return `${(bytes / 1024).toFixed(1)} KB`;
        }
        return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    }

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate("/mysteries")}>
                &larr; All Mysteries
            </span>

            {mystery.solved && mystery.winner && (
                <div className={styles.solvedBanner}>Mystery solved! Winner: {mystery.winner.display_name}</div>
            )}

            <div className={styles.detail}>
                <div
                    style={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "flex-start",
                        flexWrap: "wrap",
                        gap: "0.5rem",
                    }}
                >
                    <div>
                        <h1 className={styles.detailTitle}>{mystery.title}</h1>
                        <div className={styles.detailMeta}>
                            <ProfileLink user={mystery.author} size="small" />
                            <span>{relativeTime(mystery.created_at)}</span>
                        </div>
                        <div className={styles.cardBadges}>
                            <span className={`${styles.badge} ${styles.badgeDifficulty}`}>{mystery.difficulty}</span>
                            <span
                                className={`${styles.badge} ${mystery.solved ? styles.badgeSolved : styles.badgeOpen}`}
                            >
                                {mystery.solved ? "Solved" : "Open"}
                            </span>
                            {mystery.paused && <span className={`${styles.badge} ${styles.badgePaused}`}>Paused</span>}
                            {mystery.gm_away && !mystery.paused && (
                                <span className={`${styles.badge} ${styles.badgeAway}`}>GM Away</span>
                            )}
                            {mystery.free_for_all && (
                                <span className={`${styles.badge} ${styles.badgeFreeForAll}`}>Free-for-all</span>
                            )}
                            <span className={`${styles.badge} ${styles.badgePieces}`}>
                                {mystery.player_count} piece{mystery.player_count !== 1 ? "s" : ""} attempting
                            </span>
                        </div>
                    </div>
                    <div style={{ display: "flex", gap: "0.5rem", alignItems: "center", flexWrap: "wrap" }}>
                        {canEdit && (
                            <Button
                                variant="secondary"
                                size="small"
                                onClick={() => navigate(`/mystery/${mystery.id}/edit`)}
                            >
                                Edit
                            </Button>
                        )}
                        {canDelete && (
                            <Button variant="danger" size="small" onClick={handleDelete}>
                                Delete
                            </Button>
                        )}
                        {(isAuthor || canEdit) && !mystery.solved && (
                            <Button
                                variant={mystery.paused ? "primary" : "ghost"}
                                size="small"
                                onClick={async () => {
                                    await setPausedMutation.mutateAsync(!mystery.paused);
                                    fetchMystery();
                                }}
                            >
                                {mystery.paused ? "Resume" : "Pause"}
                            </Button>
                        )}
                        {(isAuthor || canEdit) && !mystery.solved && !mystery.paused && (
                            <Button
                                variant={mystery.gm_away ? "primary" : "ghost"}
                                size="small"
                                onClick={async () => {
                                    await setGmAwayMutation.mutateAsync(!mystery.gm_away);
                                    fetchMystery();
                                }}
                            >
                                {mystery.gm_away ? "I'm back" : "Mark as away"}
                            </Button>
                        )}
                        <ShareButton contentId={mystery.id} contentType="mystery" contentTitle={mystery.title} />
                        {user && !isAuthor && <ReportButton targetType="mystery" targetId={mystery.id} />}
                    </div>
                </div>

                <div className={styles.detailBody}>{renderRich(mystery.body)}</div>

                {mystery.clues.filter(c => !c.player_id).length > 0 && (
                    <div className={styles.cluesSection}>
                        <h3 className={styles.cluesTitle}>Red Truths</h3>
                        {mystery.clues
                            .filter(c => !c.player_id)
                            .map(clue => (
                                <div
                                    key={clue.id}
                                    className={`${styles.clue}${clue.truth_type === "purple" ? ` ${styles.cluePurple}` : ""}`}
                                >
                                    {clue.body}
                                    <span className={styles.clueActions}>
                                        <ClueCopyBtn text={clue.body} />
                                    </span>
                                </div>
                            ))}
                    </div>
                )}

                {isAuthor && (
                    <div className={styles.composer}>
                        <textarea
                            className={styles.composerTextarea}
                            placeholder="Add a new red truth clue..."
                            value={newClueBody}
                            onChange={e => setNewClueBody(e.target.value)}
                            rows={2}
                        />
                        <div className={styles.composerActions}>
                            <Button
                                variant="danger"
                                size="small"
                                onClick={handleAddClue}
                                disabled={!newClueBody.trim() || addingClue}
                            >
                                {addingClue ? "..." : "Add global Red Truth"}
                            </Button>
                        </div>
                    </div>
                )}

                {((mystery.attachments && mystery.attachments.length > 0) || isAuthor || canEdit) && (
                    <div className={styles.attachments}>
                        <h3 className={styles.attachmentsTitle}>Attachments</h3>
                        {mystery.attachments?.map(att => (
                            <div key={att.id} className={styles.attachmentItem}>
                                <svg
                                    className={styles.attachmentIcon}
                                    width="16"
                                    height="16"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="2"
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                >
                                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                                    <polyline points="14 2 14 8 20 8" />
                                </svg>
                                <a
                                    href={att.file_url}
                                    className={styles.attachmentLink}
                                    download={att.file_name}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    {att.file_name}
                                </a>
                                <span className={styles.attachmentSize}>{formatFileSize(att.file_size)}</span>
                                {(isAuthor || canEdit) && (
                                    <button
                                        type="button"
                                        className={styles.attachmentDelete}
                                        onClick={() => handleDeleteAttachment(att)}
                                        title="Delete attachment"
                                    >
                                        &times;
                                    </button>
                                )}
                            </div>
                        ))}
                        {(isAuthor || canEdit) && (
                            <>
                                <input
                                    ref={attachmentInputRef}
                                    type="file"
                                    accept=".pdf,.txt,.docx"
                                    style={{ display: "none" }}
                                    onChange={handleAttachmentUpload}
                                />
                                <Button
                                    variant="secondary"
                                    size="small"
                                    onClick={() => attachmentInputRef.current?.click()}
                                    disabled={uploadingAttachment}
                                >
                                    {uploadingAttachment ? "Uploading..." : "Add Attachment"}
                                </Button>
                                {attachmentError && <ErrorBanner message={attachmentError} />}
                            </>
                        )}
                    </div>
                )}
            </div>

            <div className={styles.attemptsSection}>
                <h3 className={styles.attemptsTitle}>Blue Truth Attempts ({mystery.attempts.length})</h3>

                {(canSeeAsGameMaster || mystery.free_for_all) && !mystery.solved && groupedAttempts.length > 0 && (
                    <div className={styles.playerPills}>
                        {groupedAttempts.map(group => {
                            const isUnread = unreadPlayers.has(group.author.id);
                            return (
                                <button
                                    key={group.author.id}
                                    type="button"
                                    className={`${styles.playerPill}${isUnread ? ` ${styles.playerPillUnread}` : ""}`}
                                    onClick={() => jumpToPlayer(group.author.id)}
                                    title={`Jump to ${group.author.display_name}'s attempts`}
                                >
                                    {group.author.avatar_url ? (
                                        <img className={styles.playerPillAvatar} src={group.author.avatar_url} alt="" />
                                    ) : (
                                        <span className={styles.playerPillAvatarPlaceholder}>
                                            {group.author.display_name[0]}
                                        </span>
                                    )}
                                    <span className={styles.playerPillName}>{group.author.display_name}</span>
                                    {isUnread && <span className={styles.playerPillDot} aria-label="unread" />}
                                </button>
                            );
                        })}
                    </div>
                )}

                {winningAttempt && (
                    <div className={styles.pinnedWinner}>
                        <div className={styles.pinnedWinnerHeader}>
                            <span className={styles.pinnedWinnerLabel}>Winning Attempt</span>
                            <a
                                className={styles.pinnedWinnerJump}
                                href={`#attempt-${winningAttempt.id}`}
                                onClick={e => {
                                    e.preventDefault();
                                    const el = document.getElementById(`attempt-${winningAttempt.id}`);
                                    if (el) {
                                        el.scrollIntoView({ behavior: "smooth", block: "center" });
                                        window.history.replaceState(null, "", `#attempt-${winningAttempt.id}`);
                                    }
                                }}
                            >
                                Jump to original &rarr;
                            </a>
                        </div>
                        <div className={styles.pinnedWinnerMeta}>
                            <ProfileLink user={winningAttempt.author} size="small" />
                            <span>{relativeTime(winningAttempt.created_at)}</span>
                        </div>
                        <div className={styles.pinnedWinnerBody}>{winningAttempt.body}</div>
                    </div>
                )}

                {canSeeAsGameMaster || mystery.solved || mystery.free_for_all ? (
                    groupedAttempts.map(group => {
                        const collapsed = collapsedPlayers.has(group.author.id);
                        return (
                            <div
                                key={group.author.id}
                                id={`player-group-${group.author.id}`}
                                className={styles.playerGroup}
                            >
                                <button
                                    type="button"
                                    className={styles.playerGroupHeader}
                                    onClick={() => togglePlayerCollapse(group.author.id)}
                                    aria-expanded={!collapsed}
                                >
                                    <span className={styles.playerGroupChevron}>{collapsed ? "\u25B6" : "\u25BC"}</span>
                                    <ProfileLink user={group.author} size="small" />
                                    <span className={styles.playerGroupCount}>
                                        {group.attempts.length} attempt
                                        {group.attempts.length !== 1 ? "s" : ""}
                                    </span>
                                </button>
                                {!collapsed && (
                                    <>
                                        <PrivateCluesDisplay
                                            clues={mystery.clues}
                                            playerId={group.author.id}
                                            mysteryId={mystery.id}
                                            canEditClues={canEdit}
                                            onAdded={fetchMystery}
                                        />
                                        {group.attempts.map(a => (
                                            <AttemptItem
                                                key={a.id}
                                                attempt={a}
                                                mysteryId={mystery.id}
                                                isAuthor={isAuthor}
                                                onRefresh={fetchMystery}
                                                mysterySolved={mystery.solved}
                                                mysteryPaused={mystery.paused}
                                            />
                                        ))}
                                        {isAuthor && (
                                            <PrivateClueInput
                                                playerId={group.author.id}
                                                mysteryId={mystery.id}
                                                onAdded={fetchMystery}
                                            />
                                        )}
                                    </>
                                )}
                            </div>
                        );
                    })
                ) : (
                    <>
                        {user && (
                            <PrivateCluesDisplay
                                clues={mystery.clues}
                                playerId={user.id}
                                mysteryId={mystery.id}
                                canEditClues={false}
                                onAdded={fetchMystery}
                                title="Private Red Truths (to you)"
                            />
                        )}
                        {mystery.attempts.map(a => (
                            <AttemptItem
                                key={a.id}
                                attempt={a}
                                mysteryId={mystery.id}
                                isAuthor={isAuthor}
                                onRefresh={fetchMystery}
                                mysterySolved={mystery.solved}
                                mysteryPaused={mystery.paused}
                            />
                        ))}
                    </>
                )}

                {mystery.attempts.length === 0 && (
                    <div className="empty-state">
                        {!canSeeAsGameMaster && mystery.player_count > 0
                            ? `There ${mystery.player_count === 1 ? "is" : "are"} ${mystery.player_count} piece${mystery.player_count !== 1 ? "s" : ""} playing this mystery. Join the game board and declare your own blue truth!`
                            : canSeeAsGameMaster
                              ? "No attempts yet. Waiting for pieces to make their move."
                              : "No attempts yet. Be the first to declare your blue truth!"}
                    </div>
                )}

                {user &&
                    !isAuthor &&
                    !mystery.solved &&
                    (mystery.paused ? (
                        <div className={styles.pausedBanner}>
                            The Game Master has paused this mystery. New attempts are temporarily disabled.
                        </div>
                    ) : (
                        <>
                            {mystery.gm_away && (
                                <div className={styles.awayBanner}>
                                    The Game Master is currently away. You can still post theories, but responses may be
                                    delayed.
                                </div>
                            )}
                            <div className={styles.composer}>
                                <textarea
                                    className={styles.composerTextarea}
                                    placeholder="Declare your blue truth..."
                                    value={attemptBody}
                                    onChange={e => setAttemptBody(e.target.value)}
                                    rows={3}
                                />
                                <div className={styles.composerActions}>
                                    <Button
                                        variant="primary"
                                        onClick={handleSubmitAttempt}
                                        disabled={!attemptBody.trim() || submitting}
                                    >
                                        {submitting ? "..." : "Submit Blue Truth"}
                                    </Button>
                                </div>
                            </div>
                        </>
                    ))}

                {!user && (
                    <div className="empty-state">
                        <Button variant="primary" onClick={() => navigate("/login")}>
                            Sign in to attempt
                        </Button>
                    </div>
                )}
            </div>

            {mystery.solved && (
                <div className={styles.discussionSection}>
                    <h3 className={styles.attemptsTitle}>
                        Post-Game Discussion {mystery.comments.length > 0 && `(${mystery.comments.length})`}
                    </h3>
                    {mystery.comments.map(c => (
                        <CommentItem
                            key={c.id}
                            comment={c as unknown as PostComment}
                            postId={mystery.id}
                            onDelete={fetchMystery}
                            highlightedId={undefined}
                            linkPrefix="/mystery"
                            reportType="mystery_comment"
                            likeFn={likeCommentFn}
                            unlikeFn={unlikeCommentFn}
                            deleteFn={deleteCommentFn}
                            updateFn={updateCommentFn}
                            createCommentFn={createCommentFn}
                            uploadMediaFn={uploadCommentMediaFn}
                        />
                    ))}
                    {mystery.comments.length === 0 && (
                        <p className="empty-state">The mystery is solved. Share your thoughts on the game!</p>
                    )}
                    {user && id && (
                        <CommentComposer
                            postId={id}
                            onCreated={fetchMystery}
                            createCommentFn={createCommentFn}
                            uploadMediaFn={uploadCommentMediaFn}
                        />
                    )}
                </div>
            )}
        </div>
    );
}
