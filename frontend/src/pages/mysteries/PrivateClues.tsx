import { type MouseEvent, useState } from "react";
import { cluesForPlayer } from "../../domain/mystery";
import { useResetOnChange } from "../../hooks/useResetOnChange";
import type { MysteryClue } from "../../types/api";
import { renderRich } from "../../components/richText/richText";
import { Button } from "../../components/Button/Button";
import styles from "./MysteryPages.module.css";

const ADDED_NOTICE_MS = 2000;

export function ClueCopyButton({ text }: { text: string }) {
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
                "✓"
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

function readCollapsed(storageKey: string): boolean {
    try {
        return window.localStorage.getItem(storageKey) === "1";
    } catch {
        return false;
    }
}

export function PrivateClues({
    clues,
    playerId,
    mysteryId,
    canEditClues,
    onSave,
    onDelete,
    title,
}: {
    clues: MysteryClue[];
    playerId: string;
    mysteryId: string;
    canEditClues: boolean;
    onSave: (clueId: number, body: string) => Promise<void>;
    onDelete: (clueId: number) => Promise<void>;
    title?: string;
}) {
    const storageKey = `mystery:${mysteryId}:private-clues:${playerId}:collapsed`;
    const [editingClueId, setEditingClueId] = useState<number | null>(null);
    const [editClueBody, setEditClueBody] = useState("");
    const [collapsed, setCollapsed] = useState<boolean>(() => readCollapsed(storageKey));

    useResetOnChange(storageKey, () => {
        setEditingClueId(null);
        setEditClueBody("");
        setCollapsed(readCollapsed(storageKey));
    });

    const playerClues = cluesForPlayer(clues, playerId);

    function toggleCollapsed() {
        setCollapsed(prev => {
            const next = !prev;
            try {
                if (next) {
                    window.localStorage.setItem(storageKey, "1");
                } else {
                    window.localStorage.removeItem(storageKey);
                }
            } catch {}
            return next;
        });
    }

    async function handleSaveClue(clueId: number) {
        if (!editClueBody.trim()) {
            return;
        }
        await onSave(clueId, editClueBody.trim());
        setEditingClueId(null);
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
                <span>{collapsed ? "▶" : "▼"}</span>
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
                                        dir="auto"
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
                                    <span dir="auto">{renderRich(clue.body)}</span>
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
                                                    onClick={() => onDelete(clue.id)}
                                                >
                                                    delete
                                                </button>
                                            </>
                                        )}
                                        <ClueCopyButton text={clue.body} />
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

export function PrivateClueInput({
    playerId,
    onAdd,
}: {
    playerId: string;
    onAdd: (playerId: string, body: string) => Promise<void>;
}) {
    const [body, setBody] = useState("");
    const [adding, setAdding] = useState(false);
    const [added, setAdded] = useState(false);

    async function handleAdd() {
        if (!body.trim() || adding) {
            return;
        }
        setAdding(true);
        try {
            await onAdd(playerId, body.trim());
            setBody("");
            setAdded(true);
            setTimeout(() => setAdded(false), ADDED_NOTICE_MS);
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
                    dir="auto"
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
