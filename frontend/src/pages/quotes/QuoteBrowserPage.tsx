import { useCallback, useEffect, useState } from "react";
import type { QuoteBrowseResponse } from "../../types/api";
import type { Series } from "../../api/endpoints";
import { browseQuotes, getCharacters } from "../../api/endpoints";
import { usePageTitle } from "../../hooks/usePageTitle";
import { getSeriesConfig } from "../../utils/seriesConfig";
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
    usePageTitle("Quotes");
    const [series, setSeries] = useState<Series>("umineko");
    const [episode, setEpisode] = useState(0);
    const [arc, setArc] = useState("");
    const [character, setCharacter] = useState("");
    const [truth, setTruth] = useState("");
    const [lang, setLang] = useState("");
    const [characters, setCharacters] = useState<Record<string, string>>({});
    const [data, setData] = useState<QuoteBrowseResponse | null>(null);
    const [loading, setLoading] = useState(false);
    const [offset, setOffset] = useState(0);
    const limit = 30;
    const cfg = getSeriesConfig(series);

    useEffect(() => {
        getCharacters(series)
            .then(setCharacters)
            .catch(() => setCharacters({}));
    }, [series]);

    const fetchQuotes = useCallback(
        async (currentOffset: number) => {
            setLoading(true);
            try {
                const result = await browseQuotes({
                    episode: episode || undefined,
                    character: character || undefined,
                    truth: truth || undefined,
                    arc: arc || undefined,
                    lang: lang || undefined,
                    limit,
                    offset: currentOffset,
                    series,
                });
                setData(result);
            } catch {
                setData(null);
            } finally {
                setLoading(false);
            }
        },
        [episode, character, truth, arc, lang, series],
    );

    function changeSeries(next: Series) {
        setSeries(next);
        setCharacter("");
        setTruth("");
        setEpisode(0);
        setArc("");
        setLang("");
    }

    useEffect(() => {
        setOffset(0);
        fetchQuotes(0);
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
            <div className={styles.seriesTabs}>
                <button
                    className={`${styles.seriesTab}${series === "umineko" ? ` ${styles.seriesTabActive}` : ""}`}
                    onClick={() => changeSeries("umineko")}
                >
                    Umineko
                </button>
                <button
                    className={`${styles.seriesTab}${series === "higurashi" ? ` ${styles.seriesTabActive}` : ""}`}
                    onClick={() => changeSeries("higurashi")}
                >
                    Higurashi
                </button>
            </div>

            <div className={styles.filterPanel}>
                {series === "umineko" && (
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
                )}

                {series === "umineko" && (
                    <Select value={episode} onChange={e => setEpisode(Number((e.target as HTMLSelectElement).value))}>
                        <option value={0}>All Episodes</option>
                        {[1, 2, 3, 4, 5, 6, 7, 8].map(ep => (
                            <option key={ep} value={ep}>
                                Episode {ep}
                            </option>
                        ))}
                    </Select>
                )}

                {series === "higurashi" && (
                    <Select value={arc} onChange={e => setArc((e.target as HTMLSelectElement).value)}>
                        <option value="">All Arcs</option>
                        {(cfg.arcs ?? []).map(a => (
                            <option key={a.value} value={a.value}>
                                {a.label}
                            </option>
                        ))}
                    </Select>
                )}

                <Select value={character} onChange={e => setCharacter((e.target as HTMLSelectElement).value)}>
                    <option value="">All Characters</option>
                    {Object.entries(characters)
                        .sort((a, b) => a[1].localeCompare(b[1]))
                        .map(([id, name]) => (
                            <option key={id} value={id}>
                                {name}
                            </option>
                        ))}
                </Select>

                <Select value={lang} onChange={e => setLang((e.target as HTMLSelectElement).value)}>
                    <option value="">Default Language</option>
                    {cfg.languages.map(l => (
                        <option key={l.value} value={l.value}>
                            {l.label}
                        </option>
                    ))}
                </Select>
            </div>

            {loading && <div className="loading">Consulting the game board...</div>}

            {!loading && data && data.quotes.length === 0 && <div className="empty-state">No quotes found.</div>}

            {!loading &&
                data?.quotes.map((q, i) => <TruthCard key={q.audioId || i} quote={q} lang={lang || undefined} />)}

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
                        fetchQuotes(next);
                    }}
                    onPrev={() => {
                        const prev = Math.max(0, offset - limit);
                        setOffset(prev);
                        fetchQuotes(prev);
                    }}
                />
            )}
        </div>
    );
}
