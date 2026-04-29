import { type KeyboardEvent, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useQuickSearch } from "../../../api/queries/search";
import { useClickOutside } from "../../../hooks/useClickOutside";
import { SearchResultRow } from "./SearchResultRow";
import { SEARCH_GROUP_LABEL, SEARCH_GROUP_ORDER, SEARCH_TYPE_META, type SearchTypeGroup } from "./searchTypeMeta";
import styles from "./GlobalSearch.module.css";

export function GlobalSearch() {
    const navigate = useNavigate();
    const [value, setValue] = useState("");
    const [debounced, setDebounced] = useState("");
    const [open, setOpen] = useState(false);
    const [activeIndex, setActiveIndex] = useState(-1);
    const containerRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

    useClickOutside(containerRef, () => setOpen(false));

    useEffect(() => {
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => setDebounced(value.trim()), 200);
        return () => clearTimeout(debounceRef.current);
    }, [value]);

    const { results, loading } = useQuickSearch(debounced, open);

    const grouped = useMemo(() => {
        const map = new Map<SearchTypeGroup, typeof results>();
        for (const r of results) {
            const group = SEARCH_TYPE_META[r.type].group;
            const list = map.get(group) ?? [];
            list.push(r);
            map.set(group, list);
        }
        return SEARCH_GROUP_ORDER.map(g => ({ group: g, items: map.get(g) ?? [] })).filter(g => g.items.length > 0);
    }, [results]);

    const showDropdown = open && debounced.length >= 2;
    const hasResults = results.length > 0;

    function submit() {
        const trimmed = value.trim();
        setOpen(false);
        if (trimmed) {
            navigate(`/search?q=${encodeURIComponent(trimmed)}`);
        } else {
            navigate("/search");
        }
    }

    function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
        if (event.key === "Enter") {
            event.preventDefault();
            if (activeIndex >= 0 && activeIndex < results.length) {
                const r = results[activeIndex];
                if (r.url) {
                    setOpen(false);
                    navigate(r.url);
                    return;
                }
            }
            submit();
            return;
        }
        if (event.key === "ArrowDown") {
            event.preventDefault();
            setActiveIndex(idx => Math.min(idx + 1, results.length - 1));
            return;
        }
        if (event.key === "ArrowUp") {
            event.preventDefault();
            setActiveIndex(idx => Math.max(idx - 1, -1));
            return;
        }
        if (event.key === "Escape") {
            setOpen(false);
            inputRef.current?.blur();
        }
    }

    return (
        <div className={styles.container} ref={containerRef}>
            <div className={styles.inputWrap}>
                <svg className={styles.icon} width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                    <circle cx="7" cy="7" r="5" stroke="currentColor" strokeWidth="1.4" />
                    <path d="M11 11l3 3" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
                </svg>
                <input
                    ref={inputRef}
                    type="search"
                    className={styles.input}
                    placeholder="Search the site..."
                    value={value}
                    onChange={e => {
                        setValue(e.target.value);
                        setActiveIndex(-1);
                        setOpen(true);
                    }}
                    onFocus={() => setOpen(true)}
                    onKeyDown={handleKeyDown}
                    aria-label="Search the site"
                />
                <button type="button" className={styles.submitButton} onClick={submit} aria-label="Open search page">
                    Search
                </button>
            </div>
            {showDropdown && (
                <div className={styles.dropdown} role="listbox">
                    {loading && results.length === 0 && <div className={styles.loadingRow}>Searching...</div>}
                    {!loading && !hasResults && <div className={styles.emptyRow}>No results for "{debounced}".</div>}
                    {hasResults &&
                        grouped.map(({ group, items }) => (
                            <div key={group} className={styles.group}>
                                <div className={styles.groupHeader}>{SEARCH_GROUP_LABEL[group]}</div>
                                {items.map(item => {
                                    const flatIndex = results.indexOf(item);
                                    return (
                                        <SearchResultRow
                                            key={`${item.type}-${item.id}`}
                                            result={item}
                                            active={flatIndex === activeIndex}
                                            onSelect={() => setOpen(false)}
                                        />
                                    );
                                })}
                            </div>
                        ))}
                    {hasResults && (
                        <button type="button" className={styles.seeAll} onClick={submit}>
                            See all results for "{debounced}"
                        </button>
                    )}
                </div>
            )}
        </div>
    );
}
