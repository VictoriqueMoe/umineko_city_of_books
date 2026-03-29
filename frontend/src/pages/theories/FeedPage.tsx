import {useState} from "react";
import type {TheorySort} from "../../types/app";
import {useTheoryFeed} from "../../hooks/useTheoryFeed";
import {TheoryCard} from "../../components/theory/TheoryCard/TheoryCard";
import {Pagination} from "../../components/Pagination/Pagination";
import {Select} from "../../components/Select/Select";
import styles from "./FeedPage.module.css";

export function FeedPage() {
    const [sort, setSort] = useState<TheorySort>("new");
    const [episode, setEpisode] = useState(0);
    const { theories, total, loading, offset, limit, goNext, goPrev, hasNext, hasPrev } = useTheoryFeed(sort, episode);

    return (
        <div>
            <div className={styles.controls}>
                <div className={styles.filterGroup}>
                    {(["new", "popular", "controversial"] as TheorySort[]).map(s => (
                        <button
                            key={s}
                            className={`${styles.filterBtn}${sort === s ? ` ${styles.filterBtnActive}` : ""}`}
                            onClick={() => setSort(s)}
                        >
                            {s.charAt(0).toUpperCase() + s.slice(1)}
                        </button>
                    ))}
                </div>

                <Select value={episode} onChange={e => setEpisode(Number((e.target as HTMLSelectElement).value))}>
                    <option value={0}>All Episodes</option>
                    {[1, 2, 3, 4, 5, 6, 7, 8].map(ep => (
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
