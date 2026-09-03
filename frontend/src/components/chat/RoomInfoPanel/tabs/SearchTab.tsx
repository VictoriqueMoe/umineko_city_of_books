import { useEffect, useState } from "react";
import DOMPurify from "dompurify";
import type { SearchResult } from "../../../../types/api";
import { useRoomMessageSearch } from "../../../../hooks/queries/search";
import { Pagination } from "../../../Pagination/Pagination";
import { RelativeTimestamp } from "../../../RelativeTimestamp/RelativeTimestamp";
import { clampChars } from "../../../../utils/text";
import styles from "../../MessageSearchPanel/MessageSearchPanel.module.css";

const PAGE_LIMIT = 30;

export interface SearchTabProps {
    roomId: string;
    isActive: boolean;
    onJump: (messageId: string, createdAt?: string) => void;
    onJumped: () => void;
}

function sanitiseSnippet(input: string): string {
    return DOMPurify.sanitize(input, { ALLOWED_TAGS: ["mark"], ALLOWED_ATTR: [] });
}

export function SearchTab({ roomId, isActive, onJump, onJumped }: SearchTabProps) {
    const [term, setTerm] = useState("");
    const [debounced, setDebounced] = useState("");
    const [page, setPage] = useState(0);
    const offset = page * PAGE_LIMIT;

    useEffect(() => {
        const t = setTimeout(() => setDebounced(term), 250);
        return () => clearTimeout(t);
    }, [term]);

    const { results, total, loading } = useRoomMessageSearch(roomId, debounced, PAGE_LIMIT, offset, isActive);
    const trimmed = debounced.trim();

    function handleTermChange(value: string) {
        setTerm(value);
        setPage(0);
    }

    return (
        <>
            <div className={styles.searchBar}>
                <input
                    dir="auto"
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
                                onJumped();
                            }}
                        >
                            <div className={styles.resultMeta}>
                                {r.author.avatar_url ? (
                                    <img className={styles.resultAvatar} src={r.author.avatar_url} alt="" />
                                ) : (
                                    <span className={styles.resultAvatarPlaceholder}>{clampChars(name, 1)}</span>
                                )}
                                <div className={styles.resultMetaText}>
                                    <span className={styles.resultSender}>{name}</span>
                                    <RelativeTimestamp
                                        value={r.created_at}
                                        variant="dateTime"
                                        className={styles.resultTime}
                                    />
                                </div>
                            </div>
                            {r.snippet && (
                                <div
                                    dir="auto"
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
        </>
    );
}
