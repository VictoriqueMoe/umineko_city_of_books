import {useCallback, useEffect, useState} from "react";
import type {QuoteBrowseResponse} from "../types/api";
import {browseQuotes, getCharacters} from "../api/endpoints";
import {TruthCard} from "../components/truth/TruthCard";
import {Pagination} from "../components/common/Pagination";

const TRUTH_TYPES = ["red", "blue", "gold", "purple"];

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

    return (
        <div>
            <div className="filter-panel">
                <div className="filter-group">
                    <button className={`filter-btn${truth === "" ? " active" : ""}`} onClick={() => setTruth("")}>
                        All
                    </button>
                    {TRUTH_TYPES.map(t => (
                        <button
                            key={t}
                            className={`filter-btn ${t}${truth === t ? " active" : ""}`}
                            onClick={() => setTruth(prev => (prev === t ? "" : t))}
                        >
                            {t.charAt(0).toUpperCase() + t.slice(1)} Truth
                        </button>
                    ))}
                </div>

                <select className="filter-select" value={episode} onChange={e => setEpisode(Number(e.target.value))}>
                    <option value={0}>All Episodes</option>
                    {[1, 2, 3, 4, 5, 6, 7, 8].map(ep => (
                        <option key={ep} value={ep}>
                            Episode {ep}
                        </option>
                    ))}
                </select>

                <select className="filter-select" value={character} onChange={e => setCharacter(e.target.value)}>
                    <option value="">All Characters</option>
                    {Object.entries(characters).map(([id, name]) => (
                        <option key={id} value={id}>
                            {name}
                        </option>
                    ))}
                </select>
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
