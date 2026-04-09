import { useEffect, useRef, useState } from "react";
import type { TheorySort } from "../../types/app";
import { useTheoryFeed } from "../../hooks/useTheoryFeed";
import { usePageTitle } from "../../hooks/usePageTitle";
import { TheoryCard } from "../../components/theory/TheoryCard/TheoryCard";
import { Pagination } from "../../components/Pagination/Pagination";
import { Input } from "../../components/Input/Input";
import { Select } from "../../components/Select/Select";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import type { Series } from "../../api/endpoints";
import { getSeriesConfig } from "../../utils/seriesConfig";
import styles from "./FeedPage.module.css";

type SortCategory = "new" | "popular" | "controversial" | "credibility";

const sortPairs: Record<SortCategory, { desc: TheorySort; asc: TheorySort }> = {
    new: { desc: "new", asc: "old" },
    popular: { desc: "popular", asc: "popular_asc" },
    controversial: { desc: "controversial", asc: "controversial_asc" },
    credibility: { desc: "credibility", asc: "credibility_asc" },
};

function getCategory(sort: TheorySort): SortCategory {
    if (sort === "old") {
        return "new";
    }
    return sort.replace("_asc", "") as SortCategory;
}

function isAsc(sort: TheorySort): boolean {
    return sort === "old" || sort.endsWith("_asc");
}

export function FeedPage({ series = "umineko" }: { series?: Series }) {
    usePageTitle("Theories");
    const cfg = getSeriesConfig(series);
    const [sort, setSort] = useState<TheorySort>("new");
    const [episode, setEpisode] = useState(0);
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

    const { theories, total, loading, offset, limit, goNext, goPrev, hasNext, hasPrev } = useTheoryFeed(
        sort,
        episode,
        undefined,
        search,
        series,
    );

    function handleSortClick(category: SortCategory) {
        const current = getCategory(sort);
        if (current === category) {
            const pair = sortPairs[category];
            setSort(isAsc(sort) ? pair.desc : pair.asc);
        } else {
            setSort(sortPairs[category].desc);
        }
    }

    const activeCategory = getCategory(sort);
    const ascending = isAsc(sort);

    return (
        <div>
            <h1 className={styles.pageTitle}>{cfg.label} Theories</h1>
            <RulesBox page={series === "higurashi" ? "theories_higurashi" : "theories"} />
            <div className={styles.controls}>
                <Input
                    type="text"
                    placeholder="Search theories..."
                    value={searchInput}
                    onChange={e => setSearchInput(e.target.value)}
                    className={styles.searchInput}
                />
                <div className={styles.filterGroup}>
                    {(["new", "popular", "controversial", "credibility"] as SortCategory[]).map(s => (
                        <button
                            key={s}
                            className={`${styles.filterBtn}${activeCategory === s ? ` ${styles.filterBtnActive}` : ""}`}
                            onClick={() => handleSortClick(s)}
                        >
                            {s.charAt(0).toUpperCase() + s.slice(1)}
                            {activeCategory === s && (
                                <span className={styles.sortArrow}>{ascending ? " \u25B2" : " \u25BC"}</span>
                            )}
                        </button>
                    ))}
                </div>

                <Select value={episode} onChange={e => setEpisode(Number((e.target as HTMLSelectElement).value))}>
                    <option value={0}>All Episodes</option>
                    {Array.from({ length: cfg.episodeCount }, (_, i) => i + 1).map(ep => (
                        <option key={ep} value={ep}>
                            Episode {ep}
                        </option>
                    ))}
                </Select>
            </div>

            {loading && <div className="loading">Consulting the game board...</div>}

            {!loading && theories.length === 0 && (
                <div className="empty-state">No theories yet. Be the first to declare your blue truth.</div>
            )}

            {!loading && theories.map(theory => <TheoryCard key={theory.id} theory={theory} />)}

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
