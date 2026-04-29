import { useState } from "react";
import { Link } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useOCList } from "../../api/queries/oc";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Pagination } from "../../components/Pagination/Pagination";
import { Select } from "../../components/Select/Select";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { Input } from "../../components/Input/Input";
import { Button } from "../../components/Button/Button";
import { relativeTime } from "../../utils/time.ts";
import shipStyles from "../ships/ShipPages.module.css";

function seriesPillClass(series: string): string {
    if (series === "umineko") {
        return `${shipStyles.characterPill} ${shipStyles.characterPillUmineko}`;
    }
    if (series === "higurashi") {
        return `${shipStyles.characterPill} ${shipStyles.characterPillHigurashi}`;
    }
    return `${shipStyles.characterPill} ${shipStyles.characterPillOc}`;
}

function seriesLabel(series: string, customSeriesName?: string): string {
    if (series === "umineko") {
        return "Umineko";
    }
    if (series === "higurashi") {
        return "Higurashi";
    }
    if (series === "ciconia") {
        return "Ciconia";
    }
    return customSeriesName ? customSeriesName : "Custom";
}

export function OCListPage() {
    usePageTitle("Original Characters");
    const [offset, setOffset] = useState(0);
    const [sort, setSort] = useState("new");
    const [series, setSeries] = useState("");
    const [customSeries, setCustomSeries] = useState("");
    const limit = 20;
    const { ocs, total, loading } = useOCList({
        sort,
        series: series || undefined,
        custom: series === "custom" && customSeries ? customSeries : undefined,
        limit,
        offset,
    });

    return (
        <div className={shipStyles.page}>
            <h1 className={shipStyles.heading}>Original Characters</h1>

            <InfoPanel title="Bring your OCs to the city">
                <p>
                    Define your original characters once and reuse them anywhere on the site. Tag each OC as Umineko,
                    Higurashi, Ciconia, or a custom universe of your choosing. Your saved OCs become available in
                    fanfiction character pickers and your profile favourite character field.
                </p>
            </InfoPanel>

            <div className={shipStyles.controls}>
                <Select
                    value={sort}
                    onChange={e => {
                        setSort(e.target.value);
                        setOffset(0);
                    }}
                >
                    <option value="new">Newest</option>
                    <option value="old">Oldest</option>
                    <option value="top">Most Upvoted</option>
                    <option value="favourites">Most Favourited</option>
                    <option value="comments">Most Commented</option>
                    <option value="name">Name (A-Z)</option>
                </Select>
                <Select
                    value={series}
                    onChange={e => {
                        setSeries(e.target.value);
                        setCustomSeries("");
                        setOffset(0);
                    }}
                >
                    <option value="">All series</option>
                    <option value="umineko">Umineko</option>
                    <option value="higurashi">Higurashi</option>
                    <option value="ciconia">Ciconia</option>
                    <option value="custom">Custom</option>
                </Select>
                {series === "custom" && (
                    <Input
                        type="text"
                        placeholder="Custom series name (optional)"
                        value={customSeries}
                        onChange={e => {
                            setCustomSeries(e.target.value);
                            setOffset(0);
                        }}
                    />
                )}
                <Link to="/oc/new">
                    <Button variant="primary" size="small">
                        + New OC
                    </Button>
                </Link>
            </div>

            {loading && <div className="loading">Loading OCs...</div>}

            {!loading && ocs.length === 0 && <div className="empty-state">No OCs found. Be the first to add one!</div>}

            {!loading && (
                <div className={shipStyles.list}>
                    {ocs.map(oc => (
                        <Link key={oc.id} to={`/oc/${oc.id}`} className={shipStyles.card}>
                            {oc.thumbnail_url || oc.image_url ? (
                                <img
                                    className={shipStyles.cardImage}
                                    src={oc.thumbnail_url || oc.image_url}
                                    alt={oc.name}
                                />
                            ) : (
                                <div className={shipStyles.cardImagePlaceholder}>★</div>
                            )}
                            <div className={shipStyles.cardBody}>
                                <h3 className={shipStyles.cardTitle}>{oc.name}</h3>
                                <div className={shipStyles.characterPills}>
                                    <span className={seriesPillClass(oc.series)}>
                                        {seriesLabel(oc.series, oc.custom_series_name)}
                                    </span>
                                </div>
                                {oc.description && <p className={shipStyles.cardDescription}>{oc.description}</p>}
                                <div className={shipStyles.cardMeta}>
                                    <ProfileLink user={oc.author} size="small" clickable={false} />
                                    <span>{relativeTime(oc.created_at)}</span>
                                </div>
                                <div className={shipStyles.cardStats}>
                                    <span className={shipStyles.voteChip}>
                                        {oc.vote_score > 0 ? "+" : ""}
                                        {oc.vote_score}
                                    </span>
                                    <span>♥ {oc.favourite_count}</span>
                                    <span>
                                        {oc.comment_count} comment{oc.comment_count !== 1 ? "s" : ""}
                                    </span>
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
                onNext={() => setOffset(offset + limit)}
                onPrev={() => setOffset(Math.max(0, offset - limit))}
            />
        </div>
    );
}
