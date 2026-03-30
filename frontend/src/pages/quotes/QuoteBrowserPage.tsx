import { useCallback, useEffect, useState } from "react";
import type { QuoteBrowseResponse } from "../../types/api";
import { browseQuotes, getCharacters } from "../../api/endpoints";
import { TruthCard } from "../../components/truth/TruthCard/TruthCard";
import { Pagination } from "../../components/Pagination/Pagination";
import { Select } from "../../components/Select/Select";
import styles from "./QuoteBrowserPage.module.css";

const TRUTH_TYPES = ["red", "blue", "gold", "purple"] as const;

const TRUTH_COLOURS: Record<string, { base: string; active: string }> = {
    red: { base: styles.filterBtnRed, active: styles.filterBtnRedActive },
    blue: { base: styles.filterBtnBlue, active: styles.filterBtnBlueActive },
    gold: { base: styles.filterBtnGold, active: styles.filterBtnGoldActive },
    purple: { base: styles.filterBtnPurple, active: styles.filterBtnPurpleActive },
};

export function QuoteBrowserPage() {
    const [episode, setEpisode] = useState(0);
    const [character, setCharacter] = useState("");
    const [truth, setTruth] = useState("");
    const [characters, setCharacters] = useState<Record<string, string>>({});
    const [data, setData] = useState<QuoteBrowseResponse | null>(null);
    const [loading, setLoading] = useState(false);
    const [offset, setOffset] = useState(0);
    const limit = 30;

    useEffect(() => {
        getCharacters()
            .then(setCharacters)
            .catch(() => {});
    }, []);

    const fetchQuotes = useCallback(
        async (currentOffset: number) => {
            setLoading(true);
            try {
                const result = await browseQuotes({
                    episode: episode || undefined,
                    character: character || undefined,
                    truth: truth || undefined,
                    limit,
                    offset: currentOffset,
                });
                setData(result);
            } catch {
                setData(null);
            } finally {
                setLoading(false);
            }
        },
        [episode, character, truth],
    );

    useEffect(() => {
        setOffset(0);
        void fetchQuotes(0);
    }, [fetchQuotes]);

    function truthBtnClass(t: string): string {
        const colour = TRUTH_COLOURS[t];
        const isActive = truth === t;
        return [styles.filterBtn, colour.base, isActive ? `${styles.filterBtnActive} ${colour.active}` : ""]
            .filter(Boolean)
            .join(" ");
    }

    return (
        <div>
            <div className={styles.filterPanel}>
                <div className={styles.filterGroup}>
                    <button
                        className={`${styles.filterBtn}${truth === "" ? ` ${styles.filterBtnActive}` : ""}`}
                        onClick={() => setTruth("")}
                    >
                        All
                    </button>
                    {TRUTH_TYPES.map(t => (
                        <button
                            key={t}
                            className={truthBtnClass(t)}
                            onClick={() => setTruth(prev => (prev === t ? "" : t))}
                        >
                            {t.charAt(0).toUpperCase() + t.slice(1)} Truth
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

                <Select value={character} onChange={e => setCharacter((e.target as HTMLSelectElement).value)}>
                    <option value="">All Characters</option>
                    {Object.entries(characters).map(([id, name]) => (
                        <option key={id} value={id}>
                            {name}
                        </option>
                    ))}
                </Select>
            </div>

            {loading && <div className="loading">Consulting the game board...</div>}

            {!loading && data && data.quotes.length === 0 && <div className="empty-state">No quotes found.</div>}

            {!loading && data?.quotes.map((q, i) => <TruthCard key={q.audioId || i} quote={q} />)}

            {!loading && data && (
                <Pagination
                    offset={offset}
                    limit={limit}
                    total={data.total}
                    hasNext={offset + limit < data.total}
                    hasPrev={offset > 0}
                    onNext={() => {
                        const next = offset + limit;
                        setOffset(next);
                        void fetchQuotes(next);
                    }}
                    onPrev={() => {
                        const prev = Math.max(0, offset - limit);
                        setOffset(prev);
                        void fetchQuotes(prev);
                    }}
                />
            )}
        </div>
    );
}
