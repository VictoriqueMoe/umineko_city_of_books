import { useState } from "react";
import type { Quote } from "../../../types/api";
import type { Series } from "../../../api/endpoints";
import { useBrowseQuotes, useSearchQuotes } from "../../../api/queries/quote";
import { useCharacterGroups } from "../../../api/queries/characters";
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

function sortedEntries(map: Record<string, string>): [string, string][] {
    return Object.entries(map).sort((a, b) => a[1].localeCompare(b[1]));
}

export function TruthPicker(props: TruthPickerProps) {
    if (!props.isOpen) {
        return null;
    }
    return <TruthPickerInner {...props} />;
}

function TruthPickerInner({ isOpen, onClose, onSelect, selectedKeys, series = "umineko" }: TruthPickerProps) {
    const cfg = getSeriesConfig(series);
    const segmentNoun = cfg.chapters ? "Chapter" : cfg.arcs ? "Arc" : "Episode";
    const [query, setQuery] = useState("");
    const [submittedQuery, setSubmittedQuery] = useState("");
    const [episode, setEpisode] = useState(0);
    const [arc, setArc] = useState("");
    const [chapter, setChapter] = useState("");
    const [character, setCharacter] = useState("");
    const [truth, setTruth] = useState("");
    const [lang, setLang] = useState("");
    const [offset, setOffset] = useState(0);

    const { groups: characters } = useCharacterGroups(series);

    const common = {
        character: character || undefined,
        episode: episode || undefined,
        arc: arc || undefined,
        chapter: chapter || undefined,
        truth: truth || undefined,
        lang: lang || undefined,
        limit: LIMIT,
        offset,
        series,
    };
    const trimmedQuery = submittedQuery.trim();
    const isSearch = trimmedQuery.length > 0;
    const searchQuery = useSearchQuotes({ query: trimmedQuery, ...common }, isOpen && isSearch);
    const browseQuery = useBrowseQuotes(common, isOpen && !isSearch);
    const loading = isSearch ? searchQuery.loading : browseQuery.loading;
    const quotes: Quote[] = isSearch
        ? (searchQuery.data?.results.map(r => r.quote) ?? [])
        : (browseQuery.data?.quotes ?? []);
    const total = isSearch ? (searchQuery.data?.total ?? 0) : (browseQuery.data?.total ?? 0);

    function handleSearch() {
        setOffset(0);
        setSubmittedQuery(query);
    }

    function handlePageChange(newOffset: number) {
        setOffset(newOffset);
    }

    const mainEntries = sortedEntries(characters.main);
    const additionalEntries = sortedEntries(characters.additional);

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
                {cfg.chapters ? (
                    <Select
                        value={chapter}
                        onChange={e => {
                            setChapter((e.target as HTMLSelectElement).value);
                            setEpisode(0);
                            setArc("");
                            setOffset(0);
                        }}
                    >
                        <option value="">All Chapters</option>
                        {cfg.chapters.map(c => (
                            <option key={c.value} value={c.value}>
                                {c.label}
                            </option>
                        ))}
                    </Select>
                ) : cfg.arcs ? (
                    <Select
                        value={arc}
                        onChange={e => {
                            setArc((e.target as HTMLSelectElement).value);
                            setEpisode(0);
                            setChapter("");
                            setOffset(0);
                        }}
                    >
                        <option value="">All Arcs</option>
                        {cfg.arcs.map(a => (
                            <option key={a.value} value={a.value}>
                                {a.label}
                            </option>
                        ))}
                    </Select>
                ) : (
                    <Select
                        value={episode}
                        onChange={e => {
                            setEpisode(Number((e.target as HTMLSelectElement).value));
                            setArc("");
                            setChapter("");
                            setOffset(0);
                        }}
                    >
                        <option value={0}>All Episodes</option>
                        {Array.from({ length: cfg.episodeCount }, (_, i) => i + 1).map(ep => (
                            <option key={ep} value={ep}>
                                Episode {ep}
                            </option>
                        ))}
                    </Select>
                )}

                <Select
                    value={character}
                    onChange={e => {
                        setCharacter((e.target as HTMLSelectElement).value);
                        setOffset(0);
                    }}
                    aria-label={`Filter by ${segmentNoun.toLowerCase()} character`}
                >
                    <option value="">All Characters</option>
                    {additionalEntries.length === 0 ? (
                        mainEntries.map(([id, name]) => (
                            <option key={id} value={id}>
                                {name}
                            </option>
                        ))
                    ) : (
                        <>
                            <optgroup label="Main cast">
                                {mainEntries.map(([id, name]) => (
                                    <option key={id} value={id}>
                                        {name}
                                    </option>
                                ))}
                            </optgroup>
                            <optgroup label="Additional">
                                {additionalEntries.map(([id, name]) => (
                                    <option key={id} value={id}>
                                        {name}
                                    </option>
                                ))}
                            </optgroup>
                        </>
                    )}
                </Select>

                <Select
                    value={truth}
                    onChange={e => {
                        setTruth((e.target as HTMLSelectElement).value);
                        setOffset(0);
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
                        setLang((e.target as HTMLSelectElement).value);
                        setOffset(0);
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
