import DOMPurify from "dompurify";
import { Link } from "react-router";
import type { SearchResult } from "../../../types/api";
import { SEARCH_TYPE_META } from "./searchTypeMeta";
import styles from "./GlobalSearch.module.css";

interface SearchResultRowProps {
    result: SearchResult;
    onSelect?: () => void;
    variant?: "dropdown" | "page";
    active?: boolean;
}

export function SearchResultRow({ result, onSelect, variant = "dropdown", active = false }: SearchResultRowProps) {
    const meta = SEARCH_TYPE_META[result.type];
    const author = result.author;
    const isUser = result.type === "user";
    const avatarClass = isUser ? styles.resultAvatarLarge : styles.resultAvatar;
    const initial = (author.display_name || author.username || "?").charAt(0).toUpperCase();

    return (
        <Link
            to={result.url || "#"}
            className={[
                styles.resultRow,
                variant === "page" ? styles.resultRowPage : styles.resultRowDropdown,
                active ? styles.resultRowActive : "",
            ]
                .filter(Boolean)
                .join(" ")}
            onClick={onSelect}
        >
            <span className={styles.typeBadge} style={{ background: meta.color }}>
                {meta.short}
            </span>
            {author.avatar_url ? (
                <img src={author.avatar_url} alt="" className={avatarClass} loading="lazy" decoding="async" />
            ) : (
                <span className={`${avatarClass} ${styles.resultAvatarPlaceholder}`}>{initial}</span>
            )}
            <span className={styles.resultMain}>
                <span className={styles.resultTitle}>{result.title || "(untitled)"}</span>
                {result.snippet && (
                    <span
                        className={styles.resultSnippet}
                        dangerouslySetInnerHTML={{ __html: sanitiseSnippet(result.snippet) }}
                    />
                )}
                {!isUser && (
                    <span className={styles.resultMeta}>
                        by <span className={styles.resultAuthor}>{author.display_name || author.username}</span>
                    </span>
                )}
            </span>
        </Link>
    );
}

function sanitiseSnippet(input: string): string {
    return DOMPurify.sanitize(input, { ALLOWED_TAGS: ["mark"], ALLOWED_ATTR: [] });
}
