import { Fragment, useState } from "react";
import { Link } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { ShipCharacter } from "../../types/api";
import { useShipList } from "../../api/queries/ship";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Pagination } from "../../components/Pagination/Pagination";
import { Select } from "../../components/Select/Select";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import { relativeTime } from "../../utils/notifications";
import { PieceTrigger } from "../../features/easterEgg";
import styles from "./ShipPages.module.css";

function characterPillClass(series: string): string {
    if (series === "umineko") {
        return `${styles.characterPill} ${styles.characterPillUmineko}`;
    }
    if (series === "higurashi") {
        return `${styles.characterPill} ${styles.characterPillHigurashi}`;
    }
    return `${styles.characterPill} ${styles.characterPillOc}`;
}

export function CharacterPills({ characters }: { characters: ShipCharacter[] }) {
    const sorted = [...characters].sort((a, b) => a.sort_order - b.sort_order);
    return (
        <div className={styles.characterPills}>
            {sorted.map((c, idx) => (
                <Fragment key={`${c.series}-${c.character_id ?? c.character_name}-${idx}`}>
                    {idx > 0 && <span className={styles.xDivider}>×</span>}
                    <span className={characterPillClass(c.series)}>{c.character_name}</span>
                </Fragment>
            ))}
        </div>
    );
}

export function ShipsListPage() {
    usePageTitle("Ships");
    const [offset, setOffset] = useState(0);
    const [sort, setSort] = useState("new");
    const [series, setSeries] = useState("");
    const [crackshipsOnly, setCrackshipsOnly] = useState(false);
    const limit = 20;
    const { ships, total, loading } = useShipList({
        sort,
        series: series || undefined,
        crackships: crackshipsOnly,
        limit,
        offset,
    });

    return (
        <div className={styles.page}>
            <h1 className={styles.heading}>Ships</h1>

            <InfoPanel title="Declare Your OTP">
                <p>
                    This is the place to declare your favourite pairings from Umineko, Higurashi, or even your original
                    characters. Vote other ships up or down. If a ship drops below the crackship threshold, it gets
                    branded as a certified crackship, mii~ <PieceTrigger pieceId="piece_06" />
                </p>
            </InfoPanel>

            <RulesBox page="ships" />

            <div className={styles.controls}>
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
                    <option value="crackship">Crackship (lowest score)</option>
                    <option value="controversial">Most Controversial</option>
                    <option value="comments">Most Commented</option>
                </Select>
                <Select
                    value={series}
                    onChange={e => {
                        setSeries(e.target.value);
                        setOffset(0);
                    }}
                >
                    <option value="">All series</option>
                    <option value="umineko">Umineko</option>
                    <option value="higurashi">Higurashi</option>
                    <option value="oc">OC</option>
                </Select>
                <div className={styles.toggleWrap}>
                    <ToggleSwitch
                        enabled={crackshipsOnly}
                        onChange={next => {
                            setCrackshipsOnly(next);
                            setOffset(0);
                        }}
                        label="Crackships only"
                    />
                </div>
            </div>

            {loading && <div className="loading">Loading ships...</div>}

            {!loading && ships.length === 0 && (
                <div className="empty-state">No ships found. Be the first to declare a pairing!</div>
            )}

            {!loading && (
                <div className={styles.list}>
                    {ships.map(s => (
                        <Link
                            key={s.id}
                            to={`/ships/${s.id}`}
                            className={`${styles.card}${s.is_crackship ? ` ${styles.cardCrack}` : ""}`}
                        >
                            {s.thumbnail_url || s.image_url ? (
                                <img className={styles.cardImage} src={s.thumbnail_url || s.image_url} alt={s.title} />
                            ) : (
                                <div className={styles.cardImagePlaceholder}>♥</div>
                            )}
                            <div className={styles.cardBody}>
                                <h3 className={styles.cardTitle}>{s.title}</h3>
                                <CharacterPills characters={s.characters} />
                                {s.description && <p className={styles.cardDescription}>{s.description}</p>}
                                <div className={styles.cardMeta}>
                                    <ProfileLink user={s.author} size="small" clickable={false} />
                                    <span>{relativeTime(s.created_at)}</span>
                                </div>
                                <div className={styles.cardStats}>
                                    <span
                                        className={`${styles.voteChip}${s.is_crackship ? ` ${styles.voteChipCrack}` : ""}`}
                                    >
                                        {s.vote_score > 0 ? "+" : ""}
                                        {s.vote_score}
                                    </span>
                                    <span>
                                        {s.comment_count} comment{s.comment_count !== 1 ? "s" : ""}
                                    </span>
                                    {s.is_crackship && <span className={styles.crackshipBadge}>Crackship</span>}
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
