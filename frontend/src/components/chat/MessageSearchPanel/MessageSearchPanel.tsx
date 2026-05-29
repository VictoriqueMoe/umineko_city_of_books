import { useEffect, useState } from "react";
import DOMPurify from "dompurify";
import type { SearchResult } from "../../../types/api";
import { useRoomMessageSearch } from "../../../api/queries/search";
import { Pagination } from "../../Pagination/Pagination";
import { parseServerDate } from "../../../utils/time";
import styles from "./MessageSearchPanel.module.css";

const PAGE_LIMIT = 30;

interface MessageSearchPanelProps {
    roomId: string;
    isOpen: boolean;
    onClose: () => void;
    onJump: (messageId: string, createdAt?: string) => void;
}

function formatDateTime(iso: string): string {
    const d = parseServerDate(iso);
    if (!d) {
        return "";
    }
    return d.toLocaleString([], {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
    });
}

function sanitiseSnippet(input: string): string {
    return DOMPurify.sanitize(input, { ALLOWED_TAGS: ["mark"], ALLOWED_ATTR: [] });
}

export function MessageSearchPanel({ roomId, isOpen, onClose, onJump }: MessageSearchPanelProps) {
    const [term, setTerm] = useState("");
    const [debounced, setDebounced] = useState("");
    const [page, setPage] = useState(0);
    const offset = page * PAGE_LIMIT;

    useEffect(() => {
        const t = setTimeout(() => setDebounced(term), 250);
        return () => clearTimeout(t);
    }, [term]);

    const { results, total, loading } = useRoomMessageSearch(roomId, debounced, PAGE_LIMIT, offset, isOpen);
    const trimmed = debounced.trim();

    function handleTermChange(value: string) {
        setTerm(value);
        setPage(0);
    }

    if (!isOpen) {
        return null;
    }

    return (
        <div className={styles.overlay} onClick={onClose}>
            <aside
                className={styles.drawer}
                onClick={e => e.stopPropagation()}
                role="dialog"
                aria-label="Search messages"
            >
                <header className={styles.header}>
                    <span className={styles.title}>Search messages</span>
                    <button type="button" className={styles.closeBtn} onClick={onClose} aria-label="Close">
                        {"✕"}
                    </button>
                </header>
                <div className={styles.searchBar}>
                    <input
                        className={styles.searchInput}
                        type="text"
                        placeholder="Search this conversation..."
                        value={term}
                        onChange={e => handleTermChange(e.target.value)}
                        autoFocus
                    />
                </div>
                <div className={styles.body}>
                    {trimmed.length < 2 && <div className={styles.empty}>Type at least 2 characters to search.</div>}
                    {trimmed.length >= 2 && loading && <div className={styles.empty}>Searching...</div>}
                    {trimmed.length >= 2 && !loading && results.length === 0 && (
                        <div className={styles.empty}>No matching messages.</div>
                    )}
                    {results.map((r: SearchResult) => {
                        const name = r.author.display_name || r.author.username;
                        return (
                            <button
                                key={r.id}
                                type="button"
                                className={styles.resultItem}
                                onClick={() => {
                                    onJump(r.id, r.created_at);
                                    onClose();
                                }}
                            >
                                <div className={styles.resultMeta}>
                                    {r.author.avatar_url ? (
                                        <img className={styles.resultAvatar} src={r.author.avatar_url} alt="" />
                                    ) : (
                                        <span className={styles.resultAvatarPlaceholder}>{name.charAt(0)}</span>
                                    )}
                                    <div className={styles.resultMetaText}>
                                        <span className={styles.resultSender}>{name}</span>
                                        <span className={styles.resultTime}>{formatDateTime(r.created_at)}</span>
                                    </div>
                                </div>
                                {r.snippet && (
                                    <div
                                        className={styles.resultSnippet}
                                        dangerouslySetInnerHTML={{ __html: sanitiseSnippet(r.snippet) }}
                                    />
                                )}
                            </button>
                        );
                    })}
                </div>
                {trimmed.length >= 2 && !loading && total > 0 && (
                    <div className={styles.footer}>
                        <Pagination
                            offset={offset}
                            limit={PAGE_LIMIT}
                            total={total}
                            hasNext={offset + PAGE_LIMIT < total}
                            hasPrev={page > 0}
                            onNext={() => setPage(page + 1)}
                            onPrev={() => setPage(Math.max(0, page - 1))}
                            onFirst={() => setPage(0)}
                            onLast={() => setPage(Math.max(0, Math.ceil(total / PAGE_LIMIT) - 1))}
                            size="small"
                            compact
                        />
                    </div>
                )}
            </aside>
        </div>
    );
}
