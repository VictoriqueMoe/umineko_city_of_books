import {useCallback, useEffect, useState} from "react";
import {useNavigate, useParams} from "react-router";
import {useProfile} from "../../hooks/useProfile";
import {useTheoryFeed} from "../../hooks/useTheoryFeed";
import {getUserActivity} from "../../api/endpoints";
import type {ActivityItem} from "../../types/api";
import {Button} from "../../components/Button/Button";
import {TheoryCard} from "../../components/theory/TheoryCard/TheoryCard";
import {Pagination} from "../../components/Pagination/Pagination";
import {RolePill} from "../../components/RolePill/RolePill";
import styles from "./ProfilePage.module.css";

const SOCIAL_LABELS: Record<string, string> = {
    social_twitter: "Twitter / X",
    social_discord: "Discord",
    social_waifulist: "WaifuList",
    social_tumblr: "Tumblr",
    social_github: "GitHub",
};

function formatDate(iso: string): string {
    const d = new Date(iso);
    return d.toLocaleDateString("en-GB", { year: "numeric", month: "long", day: "numeric" });
}

function socialUrl(key: string, value: string): string {
    if (value.startsWith("http://") || value.startsWith("https://")) {
        return value;
    }
    switch (key) {
        case "social_twitter":
            return `https://x.com/${value}`;
        case "social_github":
            return `https://github.com/${value}`;
        case "social_tumblr":
            return `https://${value}.tumblr.com`;
        case "social_waifulist":
            return value.includes("/") ? `https://${value}` : value;
        default:
            return value;
    }
}

type TabType = "theories" | "activity";

export function ProfilePage() {
    const { username } = useParams<{ username: string }>();
    const navigate = useNavigate();
    const { profile, loading } = useProfile(username ?? "");
    const [activeTab, setActiveTab] = useState<TabType>("theories");

    const {
        theories,
        total,
        loading: theoriesLoading,
        offset,
        limit,
        goNext,
        goPrev,
        hasNext,
        hasPrev,
    } = useTheoryFeed("new", 0, profile?.id);

    const [activityItems, setActivityItems] = useState<ActivityItem[]>([]);
    const [activityTotal, setActivityTotal] = useState(0);
    const [activityOffset, setActivityOffset] = useState(0);
    const [activityLoading, setActivityLoading] = useState(false);
    const activityLimit = 20;

    const fetchActivity = useCallback(async (name: string, off: number) => {
        setActivityLoading(true);
        try {
            const result = await getUserActivity(name, activityLimit, off);
            setActivityItems(result.items ?? []);
            setActivityTotal(result.total);
        } catch {
            setActivityItems([]);
            setActivityTotal(0);
        } finally {
            setActivityLoading(false);
        }
    }, []);

    useEffect(() => {
        if (activeTab === "activity" && username) {
            fetchActivity(username, activityOffset);
        }
    }, [activeTab, username, activityOffset, fetchActivity]);

    if (loading) {
        return <div className="loading">Consulting the game board...</div>;
    }

    if (!profile) {
        return (
            <div className="empty-state">
                Player not found on the game board.
                <br />
                <Button variant="secondary" onClick={() => navigate("/")}>
                    Return to Feed
                </Button>
            </div>
        );
    }

    const socialEntries = Object.entries(SOCIAL_LABELS)
        .map(([key, label]) => ({
            key,
            label,
            value: profile[key as keyof typeof profile] as string,
        }))
        .filter(entry => entry.value);

    if (profile.website) {
        socialEntries.push({
            key: "website",
            label: "Website",
            value: profile.website,
        });
    }

    const showGender = profile.gender && profile.gender !== "Prefer not to say";

    return (
        <div className={styles.page}>
            <div className={styles.banner}>
                {profile.banner_url ? (
                    <img
                        src={profile.banner_url}
                        alt=""
                        className={styles.bannerImage}
                        style={{ objectPosition: `center ${profile.banner_position ?? 50}%` }}
                    />
                ) : (
                    <div className={styles.bannerGradient} />
                )}
            </div>

            <div className={styles.headerSection}>
                <div className={styles.avatarContainer}>
                    {profile.avatar_url ? (
                        <img src={profile.avatar_url} alt={profile.display_name} className={styles.avatar} />
                    ) : (
                        <div className={styles.avatarPlaceholder}>{profile.display_name.charAt(0).toUpperCase()}</div>
                    )}
                    {profile.online && <span className={styles.onlineDot} />}
                </div>
                <div className={styles.info}>
                    <h1 className={styles.displayName}>
                        {profile.display_name}
                        {profile.role && <RolePill role={profile.role} />}
                    </h1>
                    <span className={styles.username}>@{profile.username}</span>
                    <div className={styles.metaRow}>
                        {showGender && <span className={styles.metaItem}>{profile.gender}</span>}
                        {profile.pronoun_subject && profile.pronoun_possessive && (
                            <span className={styles.metaItem}>
                                {profile.pronoun_subject}/{profile.pronoun_possessive}
                            </span>
                        )}
                        <span className={styles.metaItem}>Joined {formatDate(profile.created_at)}</span>
                    </div>
                </div>
            </div>

            <div className={styles.bio}>{profile.bio || "This player has not written a bio yet."}</div>

            {socialEntries.length > 0 && (
                <div className={styles.socialRow}>
                    {socialEntries.map(entry => (
                        <span key={entry.key} className={styles.socialChip}>
                            <span className={styles.socialChipLabel}>{entry.label}</span>
                            {entry.key === "social_discord" ? (
                                <span className={styles.socialChipValue}>{entry.value}</span>
                            ) : (
                                <a
                                    className={styles.socialChipValue}
                                    href={
                                        entry.key === "website"
                                            ? entry.value.startsWith("http")
                                                ? entry.value
                                                : `https://${entry.value}`
                                            : socialUrl(entry.key, entry.value)
                                    }
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    {entry.value}
                                </a>
                            )}
                        </span>
                    ))}
                </div>
            )}

            {profile.favourite_character && (
                <div className={styles.favourite}>
                    <span className={styles.favouriteLabel}>Favourite Character</span>
                    <span className={styles.favouriteValue}>{profile.favourite_character}</span>
                </div>
            )}

            <div className={styles.stats}>
                <div className={styles.statBox}>
                    <span className={styles.statNumber}>{profile.stats.theory_count}</span>
                    <span className={styles.statLabel}>Theories</span>
                </div>
                <div className={styles.statBox}>
                    <span className={styles.statNumber}>{profile.stats.response_count}</span>
                    <span className={styles.statLabel}>Responses</span>
                </div>
                <div className={styles.statBox}>
                    <span className={styles.statNumber}>{profile.stats.votes_received}</span>
                    <span className={styles.statLabel}>Votes Received</span>
                </div>
            </div>

            <div className={styles.tabs}>
                <button
                    className={`${styles.tab} ${activeTab === "theories" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("theories")}
                >
                    Theories
                </button>
                <button
                    className={`${styles.tab} ${activeTab === "activity" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("activity")}
                >
                    Activity
                </button>
            </div>

            {activeTab === "theories" && (
                <div className={styles.tabContent}>
                    {theoriesLoading && <div className="loading">Loading theories...</div>}

                    {!theoriesLoading && theories.length === 0 && (
                        <div className="empty-state">This player has not declared any theories yet.</div>
                    )}

                    {!theoriesLoading && theories.map(theory => <TheoryCard key={theory.id} theory={theory} />)}

                    {!theoriesLoading && total > 0 && (
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
            )}

            {activeTab === "activity" && (
                <div className={styles.tabContent}>
                    {activityLoading && <div className="loading">Loading activity...</div>}

                    {!activityLoading && activityItems.length === 0 && (
                        <div className="empty-state">No activity yet.</div>
                    )}

                    {!activityLoading &&
                        activityItems.map((item, i) => (
                            <div
                                key={`${item.type}-${item.theory_id}-${item.created_at}-${i}`}
                                className={styles.activityItem}
                                onClick={() => navigate(`/theory/${item.theory_id}`)}
                            >
                                <div className={styles.activityHeader}>
                                    <span className={styles.activityType}>
                                        {item.type === "theory"
                                            ? "Created theory"
                                            : `Responded ${item.side === "with_love" ? "with love" : "without love"}`}
                                    </span>
                                    <span className={styles.activityDate}>{formatDate(item.created_at)}</span>
                                </div>
                                <div className={styles.activityTitle}>{item.theory_title}</div>
                                <div className={styles.activityBody}>
                                    {item.body.length > 200 ? `${item.body.substring(0, 200)}...` : item.body}
                                </div>
                            </div>
                        ))}

                    {!activityLoading && activityTotal > activityLimit && (
                        <Pagination
                            offset={activityOffset}
                            limit={activityLimit}
                            total={activityTotal}
                            hasNext={activityOffset + activityLimit < activityTotal}
                            hasPrev={activityOffset > 0}
                            onNext={() => setActivityOffset(prev => prev + activityLimit)}
                            onPrev={() => setActivityOffset(prev => Math.max(0, prev - activityLimit))}
                        />
                    )}
                </div>
            )}
        </div>
    );
}
