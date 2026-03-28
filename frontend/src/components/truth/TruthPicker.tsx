import { useCallback, useEffect, useRef, useState } from "react";
import type { Quote } from "../../types/api";
import { browseQuotes, getCharacters, searchQuotes } from "../../api/endpoints";
import { Modal } from "../common/Modal";
import { TruthCard } from "./TruthCard";
import { Pagination } from "../common/Pagination";

interface TruthPickerProps {
    isOpen: boolean;
    onClose: () => void;
    onSelect: (quote: Quote) => void;
    selectedKeys: string[];
}

const TRUTH_TYPES = ["red", "blue", "gold", "purple"];
const LIMIT = 20;

function quoteKey(q: Quote): string {
    if (q.audioId) {
        return `audio:${q.audioId}`;
    }
    return `index:${q.index}`;
}

export function TruthPicker({ isOpen, onClose, onSelect, selectedKeys }: TruthPickerProps) {
    const [query, setQuery] = useState("");
    const [episode, setEpisode] = useState(0);
    const [character, setCharacter] = useState("");
    const [truth, setTruth] = useState("");
    const [quotes, setQuotes] = useState<Quote[]>([]);
    const [total, setTotal] = useState(0);
    const [offset, setOffset] = useState(0);
    const [characters, setCharacters] = useState<Record<string, string>>({});
    const [loading, setLoading] = useState(false);
    const initialLoadDone = useRef(false);

    useEffect(() => {
        getCharacters()
            .then(setCharacters)
            .catch(() => {});
    }, []);

    const doFetch = useCallback(async (q: string, ep: number, char: string, tr: string, off: number) => {
        setLoading(true);
        try {
            if (q.trim()) {
                const result = await searchQuotes({
                    query: q.trim(),
                    episode: ep || undefined,
                    character: char || undefined,
                    truth: tr || undefined,
                    limit: LIMIT,
                    offset: off,
                });
                setQuotes(result.results.map(r => r.quote));
                setTotal(result.total);
            } else {
                const result = await browseQuotes({
                    episode: ep || undefined,
                    character: char || undefined,
                    truth: tr || undefined,
                    limit: LIMIT,
                    offset: off,
                });
                setQuotes(result.quotes);
                setTotal(result.total);
            }
        } catch {
            setQuotes([]);
            setTotal(0);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        if (isOpen && !initialLoadDone.current) {
            initialLoadDone.current = true;
            void doFetch("", 0, "", "", 0);
        }
        if (!isOpen) {
            initialLoadDone.current = false;
            setQuery("");
            setEpisode(0);
            setCharacter("");
            setTruth("");
            setQuotes([]);
            setTotal(0);
            setOffset(0);
        }
    }, [isOpen, doFetch]);

    function handleSearch() {
        setOffset(0);
        void doFetch(query, episode, character, truth, 0);
    }

    function handlePageChange(newOffset: number) {
        setOffset(newOffset);
        void doFetch(query, episode, character, truth, newOffset);
    }

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Select Evidence">
            <form
                className="picker-search"
                action=""
                onSubmit={e => {
                    e.preventDefault();
                    handleSearch();
                }}
            >
                <input
                    type="text"
                    className="search-input"
                    placeholder="Search quotes..."
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                />
                <button className="search-btn" type="submit">
                    Search
                </button>
            </form>

            <div className="picker-filters">
                <select
                    className="filter-select"
                    value={episode}
                    onChange={e => {
                        const val = Number(e.target.value);
                        setEpisode(val);
                        setOffset(0);
                        void doFetch(query, val, character, truth, 0);
                    }}
                >
                    <option value={0}>All Episodes</option>
                    {[1, 2, 3, 4, 5, 6, 7, 8].map(ep => (
                        <option key={ep} value={ep}>
                            Episode {ep}
                        </option>
                    ))}
                </select>

                <select
                    className="filter-select"
                    value={character}
                    onChange={e => {
                        const val = e.target.value;
                        setCharacter(val);
                        setOffset(0);
                        void doFetch(query, episode, val, truth, 0);
                    }}
                >
                    <option value="">All Characters</option>
                    {Object.entries(characters).map(([id, name]) => (
                        <option key={id} value={id}>
                            {name}
                        </option>
                    ))}
                </select>

                <select
                    className="filter-select"
                    value={truth}
                    onChange={e => {
                        const val = e.target.value;
                        setTruth(val);
                        setOffset(0);
                        void doFetch(query, episode, character, val, 0);
                    }}
                >
                    <option value="">All Types</option>
                    {TRUTH_TYPES.map(t => (
                        <option key={t} value={t}>
                            {t.charAt(0).toUpperCase() + t.slice(1)} Truth
                        </option>
                    ))}
                </select>
            </div>

            <div className={`picker-results${loading ? " loading-overlay" : ""}`}>
                {quotes.map(q => (
                    <TruthCard
                        key={q.audioId || `idx-${q.index}`}
                        quote={q}
                        onClick={() => onSelect(q)}
                        selected={selectedKeys.includes(quoteKey(q))}
                    />
                ))}
                {!loading && quotes.length === 0 && <div className="empty-state">No quotes found.</div>}
            </div>

            {total > LIMIT && (
                <div className="picker-pagination">
                    <Pagination
                        offset={offset}
                        limit={LIMIT}
                        total={total}
                        hasNext={offset + LIMIT < total}
                        hasPrev={offset > 0}
                        onNext={() => handlePageChange(offset + LIMIT)}
                        onPrev={() => handlePageChange(Math.max(0, offset - LIMIT))}
                    />
                </div>
            )}
        </Modal>
    );
}
