import { useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import type { Fanfic } from "../../types/api";
import {
    getCharacters,
    getFanficLanguages,
    getFanficSeries,
    listFanfics,
    searchOCCharacters,
} from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Pagination } from "../../components/Pagination/Pagination";
import { Select } from "../../components/Select/Select";
import { Input } from "../../components/Input/Input";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import { relativeTime } from "../../utils/notifications";
import styles from "./FanficPages.module.css";

const GENRES = [
    "Adventure",
    "Angst",
    "Crime",
    "Drama",
    "Family",
    "Fantasy",
    "Friendship",
    "General",
    "Horror",
    "Humour",
    "Hurt/Comfort",
    "Mystery",
    "Parody",
    "Poetry",
    "Romance",
    "Sci-Fi",
    "Spiritual",
    "Supernatural",
    "Suspense",
    "Tragedy",
    "Western",
];

function ratingBadgeClass(rating: string): string {
    switch (rating) {
        case "K":
            return styles.badgeRatingK;
        case "K+":
            return styles.badgeRatingKPlus;
        case "T":
            return styles.badgeRatingT;
        case "M":
            return styles.badgeRatingM;
        default:
            return "";
    }
}

function formatWordCount(n: number): string {
    if (n >= 1000) {
        return `${(n / 1000).toFixed(1)}k`;
    }
    return String(n);
}

export function FanfictionListPage() {
    const { user } = useAuth();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();

    const p = (key: string, fallback = "") => searchParams.get(key) || fallback;
    const sort = p("sort", "updated");
    const series = p("series");
    const rating = p("rating");
    const status = p("status");
    const language = p("language");
    const genreA = p("genre_a");
    const genreB = p("genre_b");
    const charA = p("char_a");
    const charB = p("char_b");
    const charC = p("char_c");
    const charD = p("char_d");
    const pairing = searchParams.get("pairing") === "true";
    const lemons = searchParams.get("lemons") === "true";
    const search = p("search");
    const offset = parseInt(p("offset", "0"), 10);
    const limit = 25;

    const [fanfics, setFanfics] = useState<Fanfic[]>([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);

    const [seriesOptions, setSeriesOptions] = useState<string[]>([]);
    const [languageOptions, setLanguageOptions] = useState<string[]>([]);
    const [uminekoChars, setUminekoChars] = useState<string[]>([]);
    const [higuChars, setHiguChars] = useState<string[]>([]);
    const [ocChars, setOcChars] = useState<string[]>([]);

    const [searchInput, setSearchInput] = useState(search);
    const [filtersOpen, setFiltersOpen] = useState(false);

    const activeFilterCount =
        [series, rating, status, language, genreA, genreB, charA, charB, charC, charD].filter(Boolean).length +
        (pairing ? 1 : 0) +
        (lemons ? 1 : 0);

    function setParam(key: string, value: string) {
        setLoading(true);
        setSearchParams(prev => {
            const next = new URLSearchParams(prev);
            if (value) {
                next.set(key, value);
            } else {
                next.delete(key);
            }
            next.delete("offset");
            return next;
        });
    }

    function setOffsetParam(value: number) {
        setLoading(true);
        setSearchParams(prev => {
            const next = new URLSearchParams(prev);
            if (value > 0) {
                next.set("offset", String(value));
            } else {
                next.delete("offset");
            }
            return next;
        });
    }

    useEffect(() => {
        Promise.all([getFanficSeries(), getFanficLanguages()])
            .then(([s, l]) => {
                setSeriesOptions(s);
                setLanguageOptions(l);
            })
            .catch(() => {});
    }, []);

    useEffect(() => {
        Promise.all([getCharacters("umineko"), getCharacters("higurashi"), searchOCCharacters("")])
            .then(([umi, higu, ocs]) => {
                setUminekoChars(Object.values(umi).sort((a, b) => a.localeCompare(b)));
                setHiguChars(Object.values(higu).sort((a, b) => a.localeCompare(b)));
                setOcChars(ocs);
            })
            .catch(() => {});
    }, []);

    useEffect(() => {
        let cancelled = false;
        listFanfics({
            sort,
            series: series || undefined,
            rating: rating || undefined,
            status: status || undefined,
            language: language || undefined,
            genre_a: genreA || undefined,
            genre_b: genreB || undefined,
            char_a: charA || undefined,
            char_b: charB || undefined,
            char_c: charC || undefined,
            char_d: charD || undefined,
            pairing: pairing || undefined,
            lemons: lemons || undefined,
            search: search || undefined,
            limit,
            offset,
        })
            .then(data => {
                if (!cancelled) {
                    setFanfics(data.fanfics ?? []);
                    setTotal(data.total);
                    setLoading(false);
                }
            })
            .catch(() => {
                if (!cancelled) {
                    setFanfics([]);
                    setTotal(0);
                    setLoading(false);
                }
            });
        return () => {
            cancelled = true;
        };
    }, [
        sort,
        series,
        rating,
        status,
        language,
        genreA,
        genreB,
        charA,
        charB,
        charC,
        charD,
        pairing,
        lemons,
        search,
        offset,
    ]);

    function renderCharacterSelect(label: string, paramKey: string, value: string) {
        return (
            <Select value={value} onChange={e => setParam(paramKey, e.target.value)} aria-label={label}>
                <option value="">All Characters</option>
                <optgroup label="Umineko">
                    {uminekoChars.map(c => (
                        <option key={c} value={c}>
                            {c}
                        </option>
                    ))}
                </optgroup>
                <optgroup label="Higurashi">
                    {higuChars.map(c => (
                        <option key={c} value={c}>
                            {c}
                        </option>
                    ))}
                </optgroup>
                {ocChars.length > 0 && (
                    <optgroup label="OC">
                        {ocChars.map(c => (
                            <option key={c} value={c}>
                                {c} (OC)
                            </option>
                        ))}
                    </optgroup>
                )}
            </Select>
        );
    }

    return (
        <div className={styles.page}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <h1 className={styles.heading}>Fanfiction</h1>
                {user && (
                    <Button variant="primary" size="small" onClick={() => navigate("/fanfiction/new")}>
                        + New Fanfic
                    </Button>
                )}
            </div>

            <InfoPanel title="Welcome to the Archive">
                <p>
                    Browse fanfiction from across the When They Cry universe. Filter by series, genre, characters, and
                    more to find your next read.
                </p>
            </InfoPanel>

            <RulesBox page="fanfiction" />

            <div className={styles.topBar}>
                <form
                    style={{ flex: 1 }}
                    onSubmit={e => {
                        e.preventDefault();
                        setParam("search", searchInput);
                    }}
                >
                    <Input
                        type="text"
                        placeholder="Search by title or summary..."
                        value={searchInput}
                        onChange={e => setSearchInput(e.target.value)}
                        fullWidth
                    />
                </form>

                <Select value={sort} onChange={e => setParam("sort", e.target.value)} aria-label="Sort">
                    <option value="updated">Recently Updated</option>
                    <option value="published">Recently Published</option>
                    <option value="favourites">Most Favourited</option>
                </Select>

                <button
                    type="button"
                    className={`${styles.filterToggleBtn}${filtersOpen ? ` ${styles.filterToggleBtnActive}` : ""}`}
                    onClick={() => setFiltersOpen(prev => !prev)}
                >
                    Filters
                    {activeFilterCount > 0 && <span className={styles.filterActiveCount}>{activeFilterCount}</span>}
                </button>
            </div>

            {filtersOpen && (
                <div className={styles.filterPanel}>
                    <Select value={series} onChange={e => setParam("series", e.target.value)} aria-label="Series">
                        <option value="">All Series</option>
                        {seriesOptions.map(s => (
                            <option key={s} value={s}>
                                {s}
                            </option>
                        ))}
                    </Select>

                    <Select value={rating} onChange={e => setParam("rating", e.target.value)} aria-label="Rating">
                        <option value="">All Ratings</option>
                        <option value="K">K</option>
                        <option value="K+">K+</option>
                        <option value="T">T</option>
                        <option value="M">M</option>
                    </Select>

                    <Select value={status} onChange={e => setParam("status", e.target.value)} aria-label="Status">
                        <option value="">All Statuses</option>
                        <option value="in_progress">In Progress</option>
                        <option value="complete">Complete</option>
                    </Select>

                    <Select value={language} onChange={e => setParam("language", e.target.value)} aria-label="Language">
                        <option value="">All Languages</option>
                        {languageOptions.map(l => (
                            <option key={l} value={l}>
                                {l}
                            </option>
                        ))}
                    </Select>

                    <Select value={genreA} onChange={e => setParam("genre_a", e.target.value)} aria-label="Genre A">
                        <option value="">Genre A (All)</option>
                        {GENRES.map(g => (
                            <option key={g} value={g}>
                                {g}
                            </option>
                        ))}
                    </Select>

                    <Select value={genreB} onChange={e => setParam("genre_b", e.target.value)} aria-label="Genre B">
                        <option value="">Genre B (All)</option>
                        {GENRES.map(g => (
                            <option key={g} value={g}>
                                {g}
                            </option>
                        ))}
                    </Select>

                    {renderCharacterSelect("Character A", "char_a", charA)}
                    {renderCharacterSelect("Character B", "char_b", charB)}
                    {renderCharacterSelect("Character C", "char_c", charC)}
                    {renderCharacterSelect("Character D", "char_d", charD)}

                    <div className={styles.filterPanelFull}>
                        <ToggleSwitch
                            enabled={pairing}
                            onChange={v => setParam("pairing", v ? "true" : "")}
                            label="Pairing"
                            description="Filter for character pairings/ships"
                        />
                    </div>
                    <div className={styles.filterPanelFull}>
                        <ToggleSwitch
                            enabled={lemons}
                            onChange={v => setParam("lemons", v ? "true" : "")}
                            label="Show lemons"
                            description="Include stories with explicit content"
                        />
                    </div>
                </div>
            )}

            {loading && <div className="loading">Loading fanfiction...</div>}

            {!loading && fanfics.length === 0 && (
                <div className="empty-state">No fanfics found matching your filters.</div>
            )}

            {!loading && (
                <div className={styles.list}>
                    {fanfics.map(f => (
                        <Link key={f.id} to={`/fanfiction/${f.id}`} className={styles.card}>
                            {(f.cover_thumbnail_url || f.cover_image_url) && (
                                <img
                                    className={styles.cardCover}
                                    src={f.cover_thumbnail_url || f.cover_image_url}
                                    alt=""
                                />
                            )}
                            <div className={styles.cardContent}>
                                <div className={styles.cardTitleRow}>
                                    <h3 className={styles.cardTitle}>{f.title}</h3>
                                    <span className={`${styles.badge} ${ratingBadgeClass(f.rating)}`}>{f.rating}</span>
                                    {f.status === "complete" && (
                                        <span className={`${styles.badge} ${styles.badgeComplete}`}>Complete</span>
                                    )}
                                    {f.status === "draft" && (
                                        <span className={`${styles.badge} ${styles.badgeStatus}`}>Draft</span>
                                    )}
                                </div>

                                <div className={styles.cardByline}>
                                    <ProfileLink user={f.author} size="small" clickable={false} />
                                    <span>{f.series}</span>
                                    <span>{f.language}</span>
                                    {f.updated_at ? (
                                        <span>Updated {relativeTime(f.updated_at)}</span>
                                    ) : (
                                        <span>{relativeTime(f.published_at)}</span>
                                    )}
                                </div>

                                {f.summary && <p className={styles.cardSummary}>{f.summary}</p>}

                                {(f.genres?.length > 0 || f.characters?.length > 0) && (
                                    <div className={styles.cardBadges}>
                                        {(f.genres ?? []).map(g => (
                                            <span key={g} className={`${styles.badge} ${styles.badgeGenre}`}>
                                                {g}
                                            </span>
                                        ))}
                                        {(f.characters ?? []).map((c, i) => (
                                            <span key={`${c.character_name}-${i}`} className={styles.charPill}>
                                                {c.character_name}
                                            </span>
                                        ))}
                                    </div>
                                )}

                                <div className={styles.cardFooter}>
                                    <div className={styles.cardStats}>
                                        <span>{formatWordCount(f.word_count)} words</span>
                                        <span>
                                            {f.chapter_count} {f.chapter_count === 1 ? "chapter" : "chapters"}
                                        </span>
                                        <span>
                                            {f.favourite_count} {f.favourite_count === 1 ? "fav" : "favs"}
                                        </span>
                                    </div>
                                </div>
                            </div>
                        </Link>
                    ))}
                </div>
            )}

            <Pagination
                offset={offset}
                limit={limit}
                total={total}
                hasNext={offset + limit < total}
                hasPrev={offset > 0}
                onNext={() => setOffsetParam(offset + limit)}
                onPrev={() => setOffsetParam(Math.max(0, offset - limit))}
            />
        </div>
    );
}
