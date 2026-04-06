import { useCallback, useEffect, useRef, useState } from "react";
import type { Quote } from "../../../types/api";
import { browseQuotes, getCharacters, searchQuotes, type Series } from "../../../api/endpoints";
import { getSeriesConfig } from "../../../utils/seriesConfig";
import { Button } from "../../Button/Button";
import { Input } from "../../Input/Input";
import { Modal } from "../../Modal/Modal";
import { TruthCard } from "../TruthCard/TruthCard";
import { Pagination } from "../../Pagination/Pagination";
import { Select } from "../../Select/Select";
import styles from "./TruthPicker.module.css";

interface TruthPickerProps {
    isOpen: boolean;
    onClose: () => void;
    onSelect: (quote: Quote, lang: string) => void;
    selectedKeys: string[];
    series?: Series;
}

const TRUTH_TYPES = ["red", "blue", "gold", "purple"];
const LIMIT = 20;

function quoteKey(q: Quote): string {
    if (q.audioId) {
        return `audio:${q.audioId}`;
    }
    return `index:${q.index}`;
}

export function TruthPicker({ isOpen, onClose, onSelect, selectedKeys, series = "umineko" }: TruthPickerProps) {
    const cfg = getSeriesConfig(series);
    const [query, setQuery] = useState("");
    const [episode, setEpisode] = useState(0);
    const [character, setCharacter] = useState("");
    const [truth, setTruth] = useState("");
    const [lang, setLang] = useState("");
    const [quotes, setQuotes] = useState<Quote[]>([]);
    const [total, setTotal] = useState(0);
    const [offset, setOffset] = useState(0);
    const [characters, setCharacters] = useState<Record<string, string>>({});
    const [loading, setLoading] = useState(false);
    const initialLoadDone = useRef(false);

    useEffect(() => {
        getCharacters(series)
            .then(setCharacters)
            .catch(() => {});
    }, [series]);

    const doFetch = useCallback(
        async (q: string, ep: number, char: string, tr: string, ln: string, off: number) => {
            setLoading(true);
            try {
                if (q.trim()) {
                    const result = await searchQuotes({
                        query: q.trim(),
                        episode: ep || undefined,
                        character: char || undefined,
                        truth: tr || undefined,
                        lang: ln || undefined,
                        limit: LIMIT,
                        offset: off,
                        series,
                    });
                    setQuotes(result.results.map(r => r.quote));
                    setTotal(result.total);
                } else {
                    const result = await browseQuotes({
                        episode: ep || undefined,
                        character: char || undefined,
                        truth: tr || undefined,
                        lang: ln || undefined,
                        limit: LIMIT,
                        offset: off,
                        series,
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
        },
        [series],
    );

    useEffect(() => {
        if (isOpen && !initialLoadDone.current) {
            initialLoadDone.current = true;
            void doFetch("", 0, "", "", lang, 0);
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
    }, [isOpen, doFetch, lang]);

    function handleSearch() {
        setOffset(0);
        void doFetch(query, episode, character, truth, lang, 0);
    }

    function handlePageChange(newOffset: number) {
        setOffset(newOffset);
        void doFetch(query, episode, character, truth, lang, newOffset);
    }

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Select Evidence">
            <form
                className={styles.search}
                action=""
                onSubmit={e => {
                    e.preventDefault();
                    handleSearch();
                }}
            >
                <Input
                    type="text"
                    fullWidth
                    placeholder="Search quotes..."
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                />
                <Button variant="primary" type="submit">
                    Search
                </Button>
            </form>

            <div className={styles.filters}>
                <Select
                    value={episode}
                    onChange={e => {
                        const val = Number((e.target as HTMLSelectElement).value);
                        setEpisode(val);
                        setOffset(0);
                        void doFetch(query, val, character, truth, lang, 0);
                    }}
                >
                    <option value={0}>All Episodes</option>
                    {Array.from({ length: cfg.episodeCount }, (_, i) => i + 1).map(ep => (
                        <option key={ep} value={ep}>
                            Episode {ep}
                        </option>
                    ))}
                </Select>

                <Select
                    value={character}
                    onChange={e => {
                        const val = (e.target as HTMLSelectElement).value;
                        setCharacter(val);
                        setOffset(0);
                        void doFetch(query, episode, val, truth, lang, 0);
                    }}
                >
                    <option value="">All Characters</option>
                    {Object.entries(characters).map(([id, name]) => (
                        <option key={id} value={id}>
                            {name}
                        </option>
                    ))}
                </Select>

                <Select
                    value={truth}
                    onChange={e => {
                        const val = (e.target as HTMLSelectElement).value;
                        setTruth(val);
                        setOffset(0);
                        void doFetch(query, episode, character, val, lang, 0);
                    }}
                >
                    <option value="">All Types</option>
                    {TRUTH_TYPES.map(t => (
                        <option key={t} value={t}>
                            {t.charAt(0).toUpperCase() + t.slice(1)} Truth
                        </option>
                    ))}
                </Select>

                <Select
                    value={lang}
                    onChange={e => {
                        const val = (e.target as HTMLSelectElement).value;
                        setLang(val);
                        setOffset(0);
                        void doFetch(query, episode, character, truth, val, 0);
                    }}
                >
                    <option value="">Default Language</option>
                    {cfg.languages.map(l => (
                        <option key={l.value} value={l.value}>
                            {l.label}
                        </option>
                    ))}
                </Select>
            </div>

            <div className={`${styles.results}${loading ? ` ${styles.loadingOverlay}` : ""}`}>
                {quotes.map(q => (
                    <TruthCard
                        key={q.audioId || `idx-${q.index}`}
                        quote={q}
                        onClick={() => onSelect(q, lang || "en")}
                        selected={selectedKeys.includes(quoteKey(q))}
                        lang={lang || undefined}
                    />
                ))}
                {!loading && quotes.length === 0 && <div className="empty-state">No quotes found.</div>}
            </div>

            {total > LIMIT && (
                <div className={styles.pagination}>
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
