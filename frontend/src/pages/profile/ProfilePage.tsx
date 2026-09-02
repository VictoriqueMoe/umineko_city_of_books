import { Link, useNavigate, useParams } from "react-router";
import { useProfilePage, type ProfileTab } from "../../hooks/useProfilePage";
import { formatDate, formatDOBWithAge, socialEntries, socialHref } from "../../domain/profile";
import { renderRich } from "../../components/richText/richText";
import { Button } from "../../components/Button/Button";
import { ListSection } from "../../components/ListSection/ListSection";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { TheoryCard } from "../../components/theory/TheoryCard/TheoryCard";
import { PostCard } from "../../components/post/PostCard/PostCard";
import { JournalCard } from "../../components/journal/JournalCard/JournalCard";
import { ArtGrid } from "../../components/art/ArtGrid/ArtGrid";
import { GalleryCard } from "../../components/art/GalleryCard/GalleryCard";
import { RolePill } from "../../components/RolePill/RolePill";
import { RoleStyledName } from "../../components/RoleStyledName/RoleStyledName";
import { BotUsageGuide } from "./BotUsageGuide";
import { CreateGalleryInline } from "./CreateGalleryInline";
import { TrophyCase } from "./TrophyCase";
import { HuntsInProgress } from "../../components/easterEgg";
import styles from "./ProfilePage.module.css";
import { ellipsise } from "../../utils/text";

const TABS: { tab: ProfileTab; label: string }[] = [
    { tab: "posts", label: "Posts" },
    { tab: "theories", label: "Theories" },
    { tab: "art", label: "Art" },
    { tab: "galleries", label: "Galleries" },
    { tab: "ships", label: "Ships" },
    { tab: "ocs", label: "OCs" },
    { tab: "mysteries", label: "Mysteries" },
    { tab: "fanfics", label: "Fanfictions" },
    { tab: "fanfic-favourites", label: "Saved Fics" },
    { tab: "journals", label: "Journals" },
    { tab: "journal-follows", label: "Following Journals" },
    { tab: "activity", label: "Activity" },
];

export function ProfilePage() {
    const { username } = useParams<{ username: string }>();
    const navigate = useNavigate();
    const page = useProfilePage(username ?? "");
    const { profile, viewer, follow, block, activeTab, setActiveTab } = page;

    if (page.loading) {
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

    const socials = socialEntries(profile);
    const showGender = profile.gender && profile.gender !== "Prefer not to say";
    const isBanned = profile.banned === true;
    const isBot = profile.is_bot === true;

    return (
        <div className={`${styles.page} ${isBanned ? styles.bannedProfile : ""}`}>
            {isBanned && (
                <div className={styles.banBanner}>
                    <span className={styles.banBannerTitle}>This user has been banned</span>
                    {profile.ban_reason && (
                        <span className={styles.banBannerReason}>
                            Reason: <bdi>{profile.ban_reason}</bdi>
                        </span>
                    )}
                </div>
            )}
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
                        <RoleStyledName name={profile.display_name} role={profile.role} />
                        <RolePill role={profile.role ?? ""} userId={profile.id} />
                        <HuntsInProgress profileUserId={profile.id} />
                    </h1>
                    <span dir="auto" className={styles.username}>
                        @{profile.username}
                    </span>
                    {viewer && viewer.id !== profile.id && follow.stats && (
                        <div className={styles.followRow}>
                            <Button
                                variant={follow.stats.is_following ? "secondary" : "primary"}
                                size="small"
                                onClick={follow.toggleFollow}
                            >
                                {follow.stats.is_following ? "Unfollow" : "Follow"}
                            </Button>
                            {profile.dms_enabled && !block.status?.blocking && !block.status?.blocked_by && (
                                <Button
                                    variant="secondary"
                                    size="small"
                                    onClick={() => navigate("/chat", { state: { dmUserId: profile.id } })}
                                >
                                    Message
                                </Button>
                            )}
                            {follow.stats.follows_you && <span className={styles.followsYou}>Follows you</span>}
                            {block.status && !profile.role && (
                                <Button variant="ghost" size="small" onClick={block.toggleBlock}>
                                    {block.status.blocking ? "Unblock" : "Block"}
                                </Button>
                            )}
                        </div>
                    )}
                    {page.canManageUser && (
                        <div className={styles.followRow}>
                            <Button variant="ghost" size="small" onClick={() => navigate(`/admin/users/${profile.id}`)}>
                                Manage account
                            </Button>
                        </div>
                    )}
                    {block.status?.blocked_by && <div className={styles.blockedBanner}>This user has blocked you.</div>}
                    <div className={styles.metaRow}>
                        {showGender && (
                            <span dir="auto" className={styles.metaItem}>
                                {profile.gender}
                            </span>
                        )}
                        {profile.pronoun_subject && profile.pronoun_possessive && (
                            <span dir="auto" className={styles.metaItem}>
                                {profile.pronoun_subject}/{profile.pronoun_possessive}
                            </span>
                        )}
                        {profile.dob && (
                            <span className={styles.metaItem}>Born {formatDOBWithAge(profile.dob, new Date())}</span>
                        )}
                        <span className={styles.metaItem}>Joined {formatDate(profile.created_at)}</span>
                    </div>
                </div>
            </div>

            {(profile.bio || !isBot) && (
                <div dir="auto" className={styles.bio}>
                    {profile.bio ? renderRich(profile.bio) : "This player has not written a bio yet."}
                </div>
            )}

            {isBot && <BotUsageGuide />}

            {socials.length > 0 && (
                <div className={styles.socialRow}>
                    {socials.map(entry => (
                        <span key={entry.key} className={styles.socialChip}>
                            <span className={styles.socialChipLabel}>{entry.label}</span>
                            {entry.key === "social_discord" ? (
                                <span dir="auto" className={styles.socialChipValue}>
                                    {entry.value}
                                </span>
                            ) : (
                                <a
                                    dir="auto"
                                    className={styles.socialChipValue}
                                    href={socialHref(entry)}
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
                    <span dir="auto" className={styles.favouriteValue}>
                        {profile.favourite_character}
                    </span>
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
                <div className={`${styles.statBox} ${styles.statBoxClickable}`} onClick={() => setActiveTab("ships")}>
                    <span className={styles.statNumber}>{profile.stats.ship_count}</span>
                    <span className={styles.statLabel}>Ships</span>
                </div>
                <div
                    className={`${styles.statBox} ${styles.statBoxClickable}`}
                    onClick={() => setActiveTab("mysteries")}
                >
                    <span className={styles.statNumber}>{profile.stats.mystery_count}</span>
                    <span className={styles.statLabel}>Mysteries</span>
                </div>
                <div className={`${styles.statBox} ${styles.statBoxClickable}`} onClick={() => setActiveTab("fanfics")}>
                    <span className={styles.statNumber}>{profile.stats.fanfic_count}</span>
                    <span className={styles.statLabel}>Fanfictions</span>
                </div>
                {follow.stats && (
                    <>
                        <div
                            className={`${styles.statBox} ${styles.statBoxClickable}`}
                            onClick={() => setActiveTab("followers")}
                        >
                            <span className={styles.statNumber}>{follow.stats.follower_count}</span>
                            <span className={styles.statLabel}>Followers</span>
                        </div>
                        <div
                            className={`${styles.statBox} ${styles.statBoxClickable}`}
                            onClick={() => setActiveTab("following")}
                        >
                            <span className={styles.statNumber}>{follow.stats.following_count}</span>
                            <span className={styles.statLabel}>Following</span>
                        </div>
                    </>
                )}
            </div>

            <TrophyCase profileUserId={profile.id} profileSecrets={profile.secrets} />

            <div className={styles.tabs}>
                {TABS.map(({ tab, label }) => (
                    <button
                        key={tab}
                        className={`${styles.tab} ${activeTab === tab ? styles.tabActive : ""}`}
                        onClick={() => setActiveTab(tab)}
                    >
                        {label}
                    </button>
                ))}
            </div>

            {activeTab === "posts" && (
                <div className={styles.tabContent}>
                    <ListSection {...page.posts} loadingText="Loading posts..." emptyText="No posts yet.">
                        {items => items.map(p => <PostCard key={p.id} post={p} onDelete={() => page.refreshPosts()} />)}
                    </ListSection>
                </div>
            )}

            {activeTab === "theories" && (
                <div className={styles.tabContent}>
                    <ListSection
                        {...page.theories}
                        loadingText="Loading theories..."
                        emptyText="This player has not declared any theories yet."
                    >
                        {items => items.map(theory => <TheoryCard key={theory.id} theory={theory} />)}
                    </ListSection>
                </div>
            )}

            {activeTab === "art" && (
                <div className={styles.tabContent}>
                    <ListSection {...page.art} loadingText="Loading art..." emptyText="No art yet.">
                        {items => <ArtGrid art={items} />}
                    </ListSection>
                </div>
            )}

            {activeTab === "galleries" && (
                <div className={styles.tabContent}>
                    {viewer && viewer.id === profile.id && (
                        <CreateGalleryInline onCreated={() => page.refreshGalleries()} />
                    )}
                    <ListSection {...page.galleries} loadingText="Loading galleries..." emptyText="No galleries yet.">
                        {items => (
                            <div className={styles.galleryGrid}>
                                {items.map(g => (
                                    <GalleryCard key={g.id} gallery={g} />
                                ))}
                            </div>
                        )}
                    </ListSection>
                </div>
            )}

            {activeTab === "ocs" && (
                <div className={styles.tabContent}>
                    <ListSection {...page.ocs} loadingText="Loading OCs..." emptyText="No OCs created yet.">
                        {items => (
                            <div className={styles.shipList}>
                                {items.map(oc => (
                                    <Link key={oc.id} to={`/oc/${oc.id}`} className={styles.shipCard}>
                                        {(oc.thumbnail_url || oc.image_url) && (
                                            <img
                                                className={styles.shipThumb}
                                                src={oc.thumbnail_url || oc.image_url}
                                                alt={oc.name}
                                            />
                                        )}
                                        <div className={styles.shipInfo}>
                                            <span dir="auto" className={styles.shipTitle}>
                                                {oc.name}
                                            </span>
                                            <span dir="auto" className={styles.shipMeta}>
                                                {oc.series === "custom"
                                                    ? (oc.custom_series_name ?? "Custom")
                                                    : oc.series}
                                            </span>
                                            <span className={styles.shipMeta}>
                                                {oc.vote_score > 0 ? "+" : ""}
                                                {oc.vote_score} &middot; ♥ {oc.favourite_count} &middot;{" "}
                                                {oc.comment_count} comment
                                                {oc.comment_count !== 1 ? "s" : ""}
                                            </span>
                                        </div>
                                    </Link>
                                ))}
                            </div>
                        )}
                    </ListSection>
                </div>
            )}

            {activeTab === "ships" && (
                <div className={styles.tabContent}>
                    <ListSection {...page.ships} loadingText="Loading ships..." emptyText="No ships declared yet.">
                        {items => (
                            <div className={styles.shipList}>
                                {items.map(s => (
                                    <Link key={s.id} to={`/ships/${s.id}`} className={styles.shipCard}>
                                        {(s.thumbnail_url || s.image_url) && (
                                            <img
                                                className={styles.shipThumb}
                                                src={s.thumbnail_url || s.image_url}
                                                alt={s.title}
                                            />
                                        )}
                                        <div className={styles.shipInfo}>
                                            <span dir="auto" className={styles.shipTitle}>
                                                {s.title}
                                            </span>
                                            <span dir="auto" className={styles.shipMeta}>
                                                {s.characters.map(c => c.character_name).join(" × ")}
                                            </span>
                                            <span className={styles.shipMeta}>
                                                {s.vote_score > 0 ? "+" : ""}
                                                {s.vote_score} &middot; {s.comment_count} comment
                                                {s.comment_count !== 1 ? "s" : ""}
                                                {s.is_crackship && " · Crackship"}
                                            </span>
                                        </div>
                                    </Link>
                                ))}
                            </div>
                        )}
                    </ListSection>
                </div>
            )}

            {activeTab === "mysteries" && (
                <div className={styles.tabContent}>
                    <ListSection
                        {...page.mysteries}
                        loadingText="Loading mysteries..."
                        emptyText="No mysteries declared yet."
                    >
                        {items => (
                            <div className={styles.mysteryList}>
                                {items.map(m => (
                                    <Link key={m.id} to={`/mystery/${m.id}`} className={styles.mysteryCard}>
                                        <div className={styles.mysteryHeader}>
                                            <span dir="auto" className={styles.mysteryTitle}>
                                                {m.title}
                                            </span>
                                            <span
                                                className={`${styles.mysteryBadge} ${m.solved ? styles.mysteryBadgeSolved : styles.mysteryBadgeOpen}`}
                                            >
                                                {m.solved ? "Solved" : "Open"}
                                            </span>
                                        </div>
                                        <span className={styles.mysteryMeta}>
                                            Difficulty: {m.difficulty} &middot; {m.attempt_count} attempt
                                            {m.attempt_count !== 1 ? "s" : ""} &middot; {m.clue_count} clue
                                            {m.clue_count !== 1 ? "s" : ""}
                                            {m.winner && (
                                                <>
                                                    {" · Winner: "}
                                                    <bdi>{m.winner.display_name}</bdi>
                                                </>
                                            )}
                                        </span>
                                    </Link>
                                ))}
                            </div>
                        )}
                    </ListSection>
                </div>
            )}

            {activeTab === "fanfics" && (
                <div className={styles.tabContent}>
                    <ListSection {...page.fanfics} loadingText="Loading fanfics..." emptyText="No fanfics written yet.">
                        {items => (
                            <div className={styles.shipList}>
                                {items.map(f => (
                                    <Link key={f.id} to={`/fanfiction/${f.id}`} className={styles.shipCard}>
                                        <div className={styles.shipInfo}>
                                            <span dir="auto" className={styles.shipTitle}>
                                                {f.title}
                                            </span>
                                            <span className={styles.shipMeta}>
                                                <bdi>{f.series}</bdi> &middot; {f.word_count.toLocaleString()} words
                                                &middot; {f.chapter_count}{" "}
                                                {f.chapter_count === 1 ? "chapter" : "chapters"}
                                            </span>
                                        </div>
                                    </Link>
                                ))}
                            </div>
                        )}
                    </ListSection>
                </div>
            )}

            {activeTab === "fanfic-favourites" && (
                <div className={styles.tabContent}>
                    <ListSection
                        {...page.favourites}
                        loadingText="Loading favourites..."
                        emptyText="No favourites saved yet."
                    >
                        {items => (
                            <div className={styles.shipList}>
                                {items.map(f => (
                                    <Link key={f.id} to={`/fanfiction/${f.id}`} className={styles.shipCard}>
                                        <div className={styles.shipInfo}>
                                            <span dir="auto" className={styles.shipTitle}>
                                                {f.title}
                                            </span>
                                            <span className={styles.shipMeta}>
                                                <bdi>{f.series}</bdi> &middot; {f.word_count.toLocaleString()} words
                                                &middot; {f.chapter_count}{" "}
                                                {f.chapter_count === 1 ? "chapter" : "chapters"}
                                            </span>
                                        </div>
                                    </Link>
                                ))}
                            </div>
                        )}
                    </ListSection>
                </div>
            )}

            {activeTab === "journals" && (
                <div className={styles.tabContent}>
                    <ListSection
                        {...page.journals}
                        loadingText="Loading journals..."
                        emptyText="No reading journals yet."
                    >
                        {items => items.map(j => <JournalCard key={j.id} journal={j} />)}
                    </ListSection>
                </div>
            )}

            {activeTab === "journal-follows" && (
                <div className={styles.tabContent}>
                    <ListSection
                        {...page.followedJournals}
                        loadingText="Loading followed journals..."
                        emptyText="Not following any journals yet."
                    >
                        {items => items.map(j => <JournalCard key={j.id} journal={j} />)}
                    </ListSection>
                </div>
            )}

            {activeTab === "activity" && (
                <div className={styles.tabContent}>
                    <ListSection {...page.activity} loadingText="Loading activity..." emptyText="No activity yet.">
                        {items =>
                            items.map((item, i) => (
                                <Link
                                    key={`${item.type}-${item.theory_id}-${item.created_at}-${i}`}
                                    to={`/theory/${item.theory_id}`}
                                    className={styles.activityItem}
                                >
                                    <div className={styles.activityHeader}>
                                        <span className={styles.activityType}>
                                            {item.type === "theory"
                                                ? "Created theory"
                                                : `Responded ${item.side === "with_love" ? "with love" : "without love"}`}
                                        </span>
                                        <span className={styles.activityDate}>{formatDate(item.created_at)}</span>
                                    </div>
                                    <div dir="auto" className={styles.activityTitle}>
                                        {item.theory_title}
                                    </div>
                                    <div dir="auto" className={styles.activityBody}>
                                        {ellipsise(item.body, 200)}
                                    </div>
                                </Link>
                            ))
                        }
                    </ListSection>
                </div>
            )}

            {(activeTab === "followers" || activeTab === "following") && (
                <div className={styles.tabContent}>
                    <ListSection
                        {...page.followList}
                        loadingText="Loading..."
                        emptyText={activeTab === "followers" ? "No followers yet." : "Not following anyone yet."}
                    >
                        {items => (
                            <div className={styles.followList}>
                                {items.map(u => (
                                    <ProfileLink key={u.id} user={u} size="medium" />
                                ))}
                            </div>
                        )}
                    </ListSection>
                </div>
            )}
        </div>
    );
}
