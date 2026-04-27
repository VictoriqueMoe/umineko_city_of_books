import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import type { JournalWork } from "../../types/api";
import { useJournalFeed, type JournalSort } from "../../api/queries/journal";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAuth } from "../../hooks/useAuth";
import { JournalCard } from "../../components/journal/JournalCard/JournalCard";
import { Pagination } from "../../components/Pagination/Pagination";
import { Input } from "../../components/Input/Input";
import { Button } from "../../components/Button/Button";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { JOURNAL_WORKS } from "../../utils/journalWorks";
import { PieceTrigger } from "../../features/easterEgg";
import styles from "./FeedPage.module.css";

const SORTS: { id: JournalSort; label: string }[] = [
    { id: "new", label: "Newest" },
    { id: "recently_active", label: "Recently Active" },
    { id: "most_followed", label: "Most Followed" },
    { id: "old", label: "Oldest" },
];

export function JournalsFeedPage() {
    usePageTitle("Reading Journals");
    const { user } = useAuth();
    const [sort, setSort] = useState<JournalSort>("recently_active");
    const [work, setWork] = useState<JournalWork | "">("");
    const [includeArchived, setIncludeArchived] = useState(false);
    const [searchInput, setSearchInput] = useState("");
    const [search, setSearch] = useState("");
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

    useEffect(() => {
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => {
            setSearch(searchInput);
        }, 300);
        return () => clearTimeout(debounceRef.current);
    }, [searchInput]);

    const { journals, total, loading, offset, limit, goNext, goPrev, hasNext, hasPrev } = useJournalFeed(
        sort,
        work,
        search,
        includeArchived,
    );

    return (
        <div className={styles.page}>
            <div className={styles.pageHeader}>
                <h1 className={styles.pageTitle}>Reading Journals</h1>
                {user && (
                    <Link to="/journals/new">
                        <Button variant="primary" size="small">
                            + New Journal <PieceTrigger pieceId="piece_08" />
                        </Button>
                    </Link>
                )}
            </div>

            <InfoPanel title="What are Reading Journals?">
                <p>
                    A <strong>Reading Journal</strong> is your own dedicated thread for live-blogging a read-through (or
                    re-read) of one of Ryukishi's works. Think of it like a personal forum thread where you post
                    reactions, theories, and predictions as you go. It's perfect for first-time readers documenting
                    their journey, or veterans pointing out things they missed the first time around.
                </p>
                <p>
                    <strong>How it works:</strong> create a journal with a title, pick the work you're reading, and
                    write a short intro. Then post your updates as comments on your own journal. Anyone who has followed
                    it will get a notification each time you do. Other readers can comment and reply to discuss, but
                    only your own posts ping followers.
                </p>
                <p>
                    Journals <strong>auto-archive after 7 days of author inactivity</strong>. Archived journals stay
                    readable but new comments are disabled, so finish (or pause) yours before you ghost.
                </p>
            </InfoPanel>

            <RulesBox page="journals" />

            <div className={styles.controls}>
                <Input
                    type="text"
                    placeholder="Search journals..."
                    value={searchInput}
                    onChange={e => setSearchInput(e.target.value)}
                    className={styles.searchInput}
                />
                <div className={styles.filterGroup}>
                    {SORTS.map(s => (
                        <button
                            key={s.id}
                            className={`${styles.filterBtn}${sort === s.id ? ` ${styles.filterBtnActive}` : ""}`}
                            onClick={() => setSort(s.id)}
                        >
                            {s.label}
                        </button>
                    ))}
                </div>
            </div>

            <div className={styles.workFilter}>
                <button
                    className={`${styles.workChip}${work === "" ? ` ${styles.workChipActive}` : ""}`}
                    onClick={() => setWork("")}
                >
                    All works
                </button>
                {JOURNAL_WORKS.map(w => (
                    <button
                        key={w.id}
                        className={`${styles.workChip}${work === w.id ? ` ${styles.workChipActive}` : ""}`}
                        onClick={() => setWork(w.id)}
                    >
                        {w.label}
                    </button>
                ))}
            </div>

            <div className={styles.archivedToggle}>
                <ToggleSwitch
                    enabled={includeArchived}
                    onChange={setIncludeArchived}
                    label="Include archived"
                    description="Show journals archived after 7 days of inactivity"
                />
            </div>

            {loading && <div className="loading">Turning the pages...</div>}

            {!loading && journals.length === 0 && (
                <div className="empty-state">No journals yet. Be the first to start your read-through.</div>
            )}

            {!loading && journals.map(j => <JournalCard key={j.id} journal={j} />)}

            {!loading && (
                <Pagination
                    offset={offset}
                    limit={limit}
                    total={total}
                    hasNext={hasNext}
                    hasPrev={hasPrev}
                    onNext={goNext}
                    onPrev={goPrev}
                />
            )}
        </div>
    );
}
