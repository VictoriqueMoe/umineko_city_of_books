import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useProfile } from "../../api/queries/profile";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useTheoryFeed } from "../../api/queries/theory";
import { useFollow } from "../../hooks/useFollow";
import { useBlock } from "../../hooks/useBlock";
import {
    useUserActivity,
    useUserArt,
    useUserFanficFavourites,
    useUserFanfics,
    useUserFollowedJournals,
    useUserGalleries,
    useUserJournals,
    useUserMysteries,
    useUserPosts,
    useUserShips,
} from "../../api/queries/user";
import { useFollowers, useFollowing } from "../../api/queries/misc";
import { useCreateGallery } from "../../api/mutations/art";
import { parseServerDate } from "../../utils/time";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { TheoryCard } from "../../components/theory/TheoryCard/TheoryCard";
import { PostCard } from "../../components/post/PostCard/PostCard";
import { JournalCard } from "../../components/journal/JournalCard/JournalCard";
import { ArtGrid } from "../../components/art/ArtGrid/ArtGrid";
import { Pagination } from "../../components/Pagination/Pagination";
import { RolePill } from "../../components/RolePill/RolePill";
import { RoleStyledName } from "../../components/RoleStyledName/RoleStyledName";
import { TrophyCase } from "./TrophyCase";
import { HuntsInProgress } from "../../features/easterEgg";
import styles from "./ProfilePage.module.css";

const SOCIAL_LABELS: Record<string, string> = {
    social_twitter: "Twitter / X",
    social_discord: "Discord",
    social_waifulist: "WaifuList",
    social_tumblr: "Tumblr",
    social_github: "GitHub",
};

function formatDate(iso: string): string {
    const d = parseServerDate(iso);
    if (!d) {
        return "";
    }
    return d.toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" });
}

function formatDOBWithAge(value: string): string {
    const parts = value.split("-");
    if (parts.length !== 3) {
        return value;
    }

    const year = Number(parts[0]);
    const month = Number(parts[1]);
    const day = Number(parts[2]);

    if (!Number.isInteger(year) || !Number.isInteger(month) || !Number.isInteger(day)) {
        return value;
    }

    const parsed = new Date(Date.UTC(year, month - 1, day));
    if (Number.isNaN(parsed.getTime())) {
        return value;
    }

    const now = new Date();
    let age = now.getUTCFullYear() - year;
    if (now.getUTCMonth() + 1 < month || (now.getUTCMonth() + 1 === month && now.getUTCDate() < day)) {
        age -= 1;
    }

    const ageLabel = age === 1 ? "year old" : "years old";
    if (age < 0) {
        return parsed.toLocaleDateString(undefined, {
            year: "numeric",
            month: "long",
            day: "2-digit",
            timeZone: "UTC",
        });
    }

    const formatted = parsed.toLocaleDateString(undefined, {
        year: "numeric",
        month: "long",
        day: "2-digit",
        timeZone: "UTC",
    });

    return `${formatted} (${age} ${ageLabel})`;
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

type TabType =
    | "posts"
    | "theories"
    | "art"
    | "galleries"
    | "ships"
    | "mysteries"
    | "fanfics"
    | "fanfic-favourites"
    | "journals"
    | "journal-follows"
    | "activity"
    | "followers"
    | "following";

export function ProfilePage() {
    const { username } = useParams<{ username: string }>();
    const navigate = useNavigate();
    const { user: currentUser } = useAuth();
    const { profile, loading } = useProfile(username ?? "");
    usePageTitle(profile?.display_name ?? "Profile");
    const [activeTab, setActiveTab] = useState<TabType>("posts");
    const follow = useFollow(profile?.id ?? "");
    const blockHook = useBlock(profile?.id ?? "");

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

    const profileID = profile?.id ?? "";
    const [postsOffset, setPostsOffset] = useState(0);
    const postsLimit = 20;
    const postsQuery = useUserPosts(activeTab === "posts" ? profileID : "", postsLimit, postsOffset);
    const userPosts = postsQuery.posts;
    const postsTotal = postsQuery.total;
    const postsLoading = postsQuery.loading;

    const [artOffset, setArtOffset] = useState(0);
    const artLimit = 24;
    const artQuery = useUserArt(activeTab === "art" ? profileID : "", artLimit, artOffset);
    const userArt = artQuery.art;
    const artTotal = artQuery.total;
    const artLoading = artQuery.loading;

    const galleriesQuery = useUserGalleries(activeTab === "galleries" ? profileID : "");
    const userGalleries = galleriesQuery.galleries;
    const galleriesLoading = galleriesQuery.loading;

    const [shipsOffset, setShipsOffset] = useState(0);
    const shipsLimit = 20;
    const shipsQuery = useUserShips(activeTab === "ships" ? profileID : "", shipsLimit, shipsOffset);
    const userShips = shipsQuery.ships;
    const shipsTotal = shipsQuery.total;
    const shipsLoading = shipsQuery.loading;

    const [mysteriesOffset, setMysteriesOffset] = useState(0);
    const mysteriesLimit = 20;
    const mysteriesQuery = useUserMysteries(
        activeTab === "mysteries" ? profileID : "",
        mysteriesLimit,
        mysteriesOffset,
    );
    const userMysteries = mysteriesQuery.mysteries;
    const mysteriesTotal = mysteriesQuery.total;
    const mysteriesLoading = mysteriesQuery.loading;

    const [fanficsOffset, setFanficsOffset] = useState(0);
    const fanficsLimit = 20;
    const fanficsQuery = useUserFanfics(activeTab === "fanfics" ? profileID : "", fanficsLimit, fanficsOffset);
    const userFanfics = fanficsQuery.fanfics;
    const fanficsTotal = fanficsQuery.total;
    const fanficsLoading = fanficsQuery.loading;

    const [favouritesOffset, setFavouritesOffset] = useState(0);
    const favouritesLimit = 20;
    const favouritesQuery = useUserFanficFavourites(
        activeTab === "fanfic-favourites" ? profileID : "",
        favouritesLimit,
        favouritesOffset,
    );
    const userFavourites = favouritesQuery.fanfics;
    const favouritesTotal = favouritesQuery.total;
    const favouritesLoading = favouritesQuery.loading;

    const [journalsOffset, setJournalsOffset] = useState(0);
    const journalsLimit = 20;
    const journalsQuery = useUserJournals(activeTab === "journals" ? profileID : "", journalsLimit, journalsOffset);
    const userJournals = journalsQuery.journals;
    const journalsTotal = journalsQuery.total;
    const journalsLoading = journalsQuery.loading;

    const [followedJournalsOffset, setFollowedJournalsOffset] = useState(0);
    const followedJournalsLimit = 20;
    const followedJournalsQuery = useUserFollowedJournals(
        activeTab === "journal-follows" ? profileID : "",
        followedJournalsLimit,
        followedJournalsOffset,
    );
    const followedJournals = followedJournalsQuery.journals;
    const followedJournalsTotal = followedJournalsQuery.total;
    const followedJournalsLoading = followedJournalsQuery.loading;

    const [activityOffset, setActivityOffset] = useState(0);
    const activityLimit = 20;
    const activityQuery = useUserActivity(activeTab === "activity" ? (username ?? "") : "");
    const activityItems = activityQuery.activity;
    const activityTotal = activityItems.length;
    const activityLoading = activityQuery.loading;

    const followersQuery = useFollowers(activeTab === "followers" && profile?.id ? profile.id : "");
    const followingQuery = useFollowing(activeTab === "following" && profile?.id ? profile.id : "");
    const followList = activeTab === "followers" ? followersQuery.users : followingQuery.users;
    const followListLoading =
        activeTab === "followers" ? followersQuery.loading : activeTab === "following" ? followingQuery.loading : false;

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

    const isBanned = profile.banned === true;

    return (
        <div className={`${styles.page} ${isBanned ? styles.bannedProfile : ""}`}>
            {isBanned && (
                <div className={styles.banBanner}>
                    <span className={styles.banBannerTitle}>This user has been banned</span>
                    {profile.ban_reason && <span className={styles.banBannerReason}>Reason: {profile.ban_reason}</span>}
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
                    <span className={styles.username}>@{profile.username}</span>
                    {currentUser && currentUser.id !== profile.id && follow.stats && (
                        <div className={styles.followRow}>
                            <Button
                                variant={follow.stats.is_following ? "secondary" : "primary"}
                                size="small"
                                onClick={follow.toggleFollow}
                            >
                                {follow.stats.is_following ? "Unfollow" : "Follow"}
                            </Button>
                            {profile.dms_enabled && !blockHook.status?.blocking && !blockHook.status?.blocked_by && (
                                <Button
                                    variant="secondary"
                                    size="small"
                                    onClick={() => navigate("/chat", { state: { dmUserId: profile.id } })}
                                >
                                    Message
                                </Button>
                            )}
                            {follow.stats.follows_you && <span className={styles.followsYou}>Follows you</span>}
                            {blockHook.status && !profile.role && (
                                <Button variant="ghost" size="small" onClick={blockHook.toggleBlock}>
                                    {blockHook.status.blocking ? "Unblock" : "Block"}
                                </Button>
                            )}
                        </div>
                    )}
                    {blockHook.status?.blocked_by && (
                        <div className={styles.blockedBanner}>This user has blocked you.</div>
                    )}
                    <div className={styles.metaRow}>
                        {showGender && <span className={styles.metaItem}>{profile.gender}</span>}
                        {profile.pronoun_subject && profile.pronoun_possessive && (
                            <span className={styles.metaItem}>
                                {profile.pronoun_subject}/{profile.pronoun_possessive}
                            </span>
                        )}
                        {profile.dob && <span className={styles.metaItem}>Born {formatDOBWithAge(profile.dob)}</span>}
                        <span className={styles.metaItem}>Joined {formatDate(profile.created_at)}</span>
                        {profile.email && (
                            <a href={`mailto:${profile.email}`} className={styles.metaItem}>
                                {profile.email}
                            </a>
                        )}
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
                <button
                    className={`${styles.tab} ${activeTab === "posts" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("posts")}
                >
                    Posts
                </button>
                <button
                    className={`${styles.tab} ${activeTab === "theories" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("theories")}
                >
                    Theories
                </button>
                <button
                    className={`${styles.tab} ${activeTab === "art" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("art")}
                >
                    Art
                </button>
                <button
                    className={`${styles.tab} ${activeTab === "galleries" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("galleries")}
                >
                    Galleries
                </button>
                <button
                    className={`${styles.tab} ${activeTab === "ships" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("ships")}
                >
                    Ships
                </button>
                <button
                    className={`${styles.tab} ${activeTab === "mysteries" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("mysteries")}
                >
                    Mysteries
                </button>
                <button
                    className={`${styles.tab} ${activeTab === "fanfics" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("fanfics")}
                >
                    Fanfictions
                </button>
                <button
                    className={`${styles.tab} ${activeTab === "fanfic-favourites" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("fanfic-favourites")}
                >
                    Saved Fics
                </button>
                <button
                    className={`${styles.tab} ${activeTab === "journals" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("journals")}
                >
                    Journals
                </button>
                <button
                    className={`${styles.tab} ${activeTab === "journal-follows" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("journal-follows")}
                >
                    Following Journals
                </button>
                <button
                    className={`${styles.tab} ${activeTab === "activity" ? styles.tabActive : ""}`}
                    onClick={() => setActiveTab("activity")}
                >
                    Activity
                </button>
            </div>

            {activeTab === "posts" && (
                <div className={styles.tabContent}>
                    {postsLoading && <div className="loading">Loading posts...</div>}
                    {!postsLoading && userPosts.length === 0 && <div className="empty-state">No posts yet.</div>}
                    {!postsLoading &&
                        userPosts.map(p => <PostCard key={p.id} post={p} onDelete={() => postsQuery.refresh()} />)}
                    {!postsLoading && postsTotal > postsLimit && (
                        <Pagination
                            offset={postsOffset}
                            limit={postsLimit}
                            total={postsTotal}
                            hasNext={postsOffset + postsLimit < postsTotal}
                            hasPrev={postsOffset > 0}
                            onNext={() => setPostsOffset(postsOffset + postsLimit)}
                            onPrev={() => setPostsOffset(Math.max(0, postsOffset - postsLimit))}
                        />
                    )}
                </div>
            )}

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

            {activeTab === "art" && (
                <div className={styles.tabContent}>
                    {artLoading && <div className="loading">Loading art...</div>}
                    {!artLoading && userArt.length === 0 && <div className="empty-state">No art yet.</div>}
                    {!artLoading && userArt.length > 0 && <ArtGrid art={userArt} />}
                    {!artLoading && artTotal > artLimit && (
                        <Pagination
                            offset={artOffset}
                            limit={artLimit}
                            total={artTotal}
                            hasNext={artOffset + artLimit < artTotal}
                            hasPrev={artOffset > 0}
                            onNext={() => setArtOffset(artOffset + artLimit)}
                            onPrev={() => setArtOffset(Math.max(0, artOffset - artLimit))}
                        />
                    )}
                </div>
            )}

            {activeTab === "galleries" && (
                <div className={styles.tabContent}>
                    {currentUser && profile && currentUser.id === profile.id && (
                        <CreateGalleryInline onCreated={() => galleriesQuery.refresh()} />
                    )}
                    {galleriesLoading && <div className="loading">Loading galleries...</div>}
                    {!galleriesLoading && userGalleries.length === 0 && (
                        <div className="empty-state">No galleries yet.</div>
                    )}
                    {!galleriesLoading && (
                        <div className={styles.galleryGrid}>
                            {userGalleries.map(g => (
                                <Link key={g.id} to={`/gallery/view/${g.id}`} className={styles.galleryCard}>
                                    <div className={styles.galleryCover}>
                                        {g.cover_thumbnail_url || g.cover_image_url ? (
                                            <img
                                                src={g.cover_thumbnail_url || g.cover_image_url}
                                                alt=""
                                                className={styles.galleryCoverImage}
                                            />
                                        ) : g.preview_images && g.preview_images.length > 0 ? (
                                            <ProfileGalleryPreview images={g.preview_images} />
                                        ) : (
                                            <div className={styles.galleryCoverPlaceholder}>Empty</div>
                                        )}
                                    </div>
                                    <div className={styles.galleryInfo}>
                                        <span className={styles.galleryName}>{g.name}</span>
                                        <span className={styles.galleryCount}>{g.art_count} pieces</span>
                                    </div>
                                </Link>
                            ))}
                        </div>
                    )}
                </div>
            )}

            {activeTab === "ships" && (
                <div className={styles.tabContent}>
                    {shipsLoading && <div className="loading">Loading ships...</div>}
                    {!shipsLoading && userShips.length === 0 && (
                        <div className="empty-state">No ships declared yet.</div>
                    )}
                    {!shipsLoading && userShips.length > 0 && (
                        <div className={styles.shipList}>
                            {userShips.map(s => (
                                <Link key={s.id} to={`/ships/${s.id}`} className={styles.shipCard}>
                                    {(s.thumbnail_url || s.image_url) && (
                                        <img
                                            className={styles.shipThumb}
                                            src={s.thumbnail_url || s.image_url}
                                            alt={s.title}
                                        />
                                    )}
                                    <div className={styles.shipInfo}>
                                        <span className={styles.shipTitle}>{s.title}</span>
                                        <span className={styles.shipMeta}>
                                            {s.characters.map(c => c.character_name).join(" × ")}
                                        </span>
                                        <span className={styles.shipMeta}>
                                            {s.vote_score > 0 ? "+" : ""}
                                            {s.vote_score} &middot; {s.comment_count} comment
                                            {s.comment_count !== 1 ? "s" : ""}
                                            {s.is_crackship && " \u00B7 Crackship"}
                                        </span>
                                    </div>
                                </Link>
                            ))}
                        </div>
                    )}
                    {!shipsLoading && shipsTotal > shipsLimit && (
                        <Pagination
                            offset={shipsOffset}
                            limit={shipsLimit}
                            total={shipsTotal}
                            hasNext={shipsOffset + shipsLimit < shipsTotal}
                            hasPrev={shipsOffset > 0}
                            onNext={() => setShipsOffset(shipsOffset + shipsLimit)}
                            onPrev={() => setShipsOffset(Math.max(0, shipsOffset - shipsLimit))}
                        />
                    )}
                </div>
            )}

            {activeTab === "mysteries" && (
                <div className={styles.tabContent}>
                    {mysteriesLoading && <div className="loading">Loading mysteries...</div>}
                    {!mysteriesLoading && userMysteries.length === 0 && (
                        <div className="empty-state">No mysteries declared yet.</div>
                    )}
                    {!mysteriesLoading && userMysteries.length > 0 && (
                        <div className={styles.mysteryList}>
                            {userMysteries.map(m => (
                                <Link key={m.id} to={`/mystery/${m.id}`} className={styles.mysteryCard}>
                                    <div className={styles.mysteryHeader}>
                                        <span className={styles.mysteryTitle}>{m.title}</span>
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
                                        {m.winner && ` \u00B7 Winner: ${m.winner.display_name}`}
                                    </span>
                                </Link>
                            ))}
                        </div>
                    )}
                    {!mysteriesLoading && mysteriesTotal > mysteriesLimit && (
                        <Pagination
                            offset={mysteriesOffset}
                            limit={mysteriesLimit}
                            total={mysteriesTotal}
                            hasNext={mysteriesOffset + mysteriesLimit < mysteriesTotal}
                            hasPrev={mysteriesOffset > 0}
                            onNext={() => setMysteriesOffset(mysteriesOffset + mysteriesLimit)}
                            onPrev={() => setMysteriesOffset(Math.max(0, mysteriesOffset - mysteriesLimit))}
                        />
                    )}
                </div>
            )}

            {activeTab === "fanfics" && (
                <div className={styles.tabContent}>
                    {fanficsLoading && <div className="loading">Loading fanfics...</div>}
                    {!fanficsLoading && userFanfics.length === 0 && (
                        <div className="empty-state">No fanfics written yet.</div>
                    )}
                    {!fanficsLoading && userFanfics.length > 0 && (
                        <div className={styles.shipList}>
                            {userFanfics.map(f => (
                                <Link key={f.id} to={`/fanfiction/${f.id}`} className={styles.shipCard}>
                                    <div className={styles.shipInfo}>
                                        <span className={styles.shipTitle}>{f.title}</span>
                                        <span className={styles.shipMeta}>
                                            {f.series} &middot; {f.word_count.toLocaleString()} words &middot;{" "}
                                            {f.chapter_count} {f.chapter_count === 1 ? "chapter" : "chapters"}
                                        </span>
                                    </div>
                                </Link>
                            ))}
                        </div>
                    )}
                    {!fanficsLoading && fanficsTotal > fanficsLimit && (
                        <Pagination
                            offset={fanficsOffset}
                            limit={fanficsLimit}
                            total={fanficsTotal}
                            hasNext={fanficsOffset + fanficsLimit < fanficsTotal}
                            hasPrev={fanficsOffset > 0}
                            onNext={() => setFanficsOffset(fanficsOffset + fanficsLimit)}
                            onPrev={() => setFanficsOffset(Math.max(0, fanficsOffset - fanficsLimit))}
                        />
                    )}
                </div>
            )}

            {activeTab === "fanfic-favourites" && (
                <div className={styles.tabContent}>
                    {favouritesLoading && <div className="loading">Loading favourites...</div>}
                    {!favouritesLoading && userFavourites.length === 0 && (
                        <div className="empty-state">No favourites saved yet.</div>
                    )}
                    {!favouritesLoading && userFavourites.length > 0 && (
                        <div className={styles.shipList}>
                            {userFavourites.map(f => (
                                <Link key={f.id} to={`/fanfiction/${f.id}`} className={styles.shipCard}>
                                    <div className={styles.shipInfo}>
                                        <span className={styles.shipTitle}>{f.title}</span>
                                        <span className={styles.shipMeta}>
                                            {f.series} &middot; {f.word_count.toLocaleString()} words &middot;{" "}
                                            {f.chapter_count} {f.chapter_count === 1 ? "chapter" : "chapters"}
                                        </span>
                                    </div>
                                </Link>
                            ))}
                        </div>
                    )}
                    {!favouritesLoading && favouritesTotal > favouritesLimit && (
                        <Pagination
                            offset={favouritesOffset}
                            limit={favouritesLimit}
                            total={favouritesTotal}
                            hasNext={favouritesOffset + favouritesLimit < favouritesTotal}
                            hasPrev={favouritesOffset > 0}
                            onNext={() => setFavouritesOffset(favouritesOffset + favouritesLimit)}
                            onPrev={() => setFavouritesOffset(Math.max(0, favouritesOffset - favouritesLimit))}
                        />
                    )}
                </div>
            )}

            {activeTab === "journals" && (
                <div className={styles.tabContent}>
                    {journalsLoading && <div className="loading">Loading journals...</div>}
                    {!journalsLoading && userJournals.length === 0 && (
                        <div className="empty-state">No reading journals yet.</div>
                    )}
                    {!journalsLoading && userJournals.map(j => <JournalCard key={j.id} journal={j} />)}
                    {!journalsLoading && journalsTotal > journalsLimit && (
                        <Pagination
                            offset={journalsOffset}
                            limit={journalsLimit}
                            total={journalsTotal}
                            hasNext={journalsOffset + journalsLimit < journalsTotal}
                            hasPrev={journalsOffset > 0}
                            onNext={() => setJournalsOffset(journalsOffset + journalsLimit)}
                            onPrev={() => setJournalsOffset(Math.max(0, journalsOffset - journalsLimit))}
                        />
                    )}
                </div>
            )}

            {activeTab === "journal-follows" && (
                <div className={styles.tabContent}>
                    {followedJournalsLoading && <div className="loading">Loading followed journals...</div>}
                    {!followedJournalsLoading && followedJournals.length === 0 && (
                        <div className="empty-state">Not following any journals yet.</div>
                    )}
                    {!followedJournalsLoading && followedJournals.map(j => <JournalCard key={j.id} journal={j} />)}
                    {!followedJournalsLoading && followedJournalsTotal > followedJournalsLimit && (
                        <Pagination
                            offset={followedJournalsOffset}
                            limit={followedJournalsLimit}
                            total={followedJournalsTotal}
                            hasNext={followedJournalsOffset + followedJournalsLimit < followedJournalsTotal}
                            hasPrev={followedJournalsOffset > 0}
                            onNext={() => setFollowedJournalsOffset(followedJournalsOffset + followedJournalsLimit)}
                            onPrev={() =>
                                setFollowedJournalsOffset(Math.max(0, followedJournalsOffset - followedJournalsLimit))
                            }
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
                                <div className={styles.activityTitle}>{item.theory_title}</div>
                                <div className={styles.activityBody}>
                                    {item.body.length > 200 ? `${item.body.substring(0, 200)}...` : item.body}
                                </div>
                            </Link>
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

            {(activeTab === "followers" || activeTab === "following") && (
                <div className={styles.tabContent}>
                    {followListLoading && <div className="loading">Loading...</div>}
                    {!followListLoading && followList.length === 0 && (
                        <div className="empty-state">
                            {activeTab === "followers" ? "No followers yet." : "Not following anyone yet."}
                        </div>
                    )}
                    {!followListLoading && (
                        <div className={styles.followList}>
                            {followList.map(u => (
                                <ProfileLink key={u.id} user={u} size="medium" />
                            ))}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}

function CreateGalleryInline({ onCreated }: { onCreated: () => void }) {
    const [open, setOpen] = useState(false);
    const [name, setName] = useState("");
    const [description, setDescription] = useState("");
    const createGalleryMutation = useCreateGallery();
    const submitting = createGalleryMutation.isPending;

    async function handleCreate() {
        if (!name.trim() || submitting) {
            return;
        }
        try {
            await createGalleryMutation.mutateAsync({ name: name.trim(), description: description.trim() });
            setName("");
            setDescription("");
            setOpen(false);
            onCreated();
        } catch {
            return;
        }
    }

    if (!open) {
        return (
            <div style={{ marginBottom: "1rem" }}>
                <Button variant="primary" size="small" onClick={() => setOpen(true)}>
                    Create Gallery
                </Button>
            </div>
        );
    }

    return (
        <div className={styles.galleryCard} style={{ padding: "1rem", cursor: "default", marginBottom: "1rem" }}>
            <input
                type="text"
                placeholder="Gallery name"
                value={name}
                onChange={e => setName(e.target.value)}
                style={{
                    width: "100%",
                    padding: "0.5rem 0.75rem",
                    border: "1px solid rgba(var(--gold-rgb), 0.2)",
                    borderRadius: "6px",
                    background: "var(--bg-card)",
                    color: "var(--text)",
                    fontSize: "0.9rem",
                    marginBottom: "0.5rem",
                    boxSizing: "border-box",
                }}
            />
            <textarea
                placeholder="Description (optional)"
                value={description}
                onChange={e => setDescription(e.target.value)}
                rows={2}
                style={{
                    width: "100%",
                    padding: "0.5rem 0.75rem",
                    border: "1px solid rgba(var(--gold-rgb), 0.2)",
                    borderRadius: "6px",
                    background: "var(--bg-card)",
                    color: "var(--text)",
                    fontSize: "0.9rem",
                    marginBottom: "0.5rem",
                    resize: "vertical",
                    boxSizing: "border-box",
                }}
            />
            <div style={{ display: "flex", gap: "0.5rem" }}>
                <Button variant="secondary" size="small" onClick={() => setOpen(false)}>
                    Cancel
                </Button>
                <Button variant="primary" size="small" onClick={handleCreate} disabled={!name.trim() || submitting}>
                    {submitting ? "Creating..." : "Create"}
                </Button>
            </div>
        </div>
    );
}

function ProfilePreviewImg({ img, className }: { img: { thumbnail: string; full: string }; className: string }) {
    return (
        <img
            src={img.thumbnail || img.full}
            alt=""
            className={className}
            onError={e => {
                if (e.currentTarget.src !== img.full) {
                    e.currentTarget.src = img.full;
                }
            }}
        />
    );
}

function ProfileGalleryPreview({ images }: { images: { thumbnail: string; full: string }[] }) {
    if (images.length === 1) {
        return <ProfilePreviewImg img={images[0]} className={styles.galleryCoverImage} />;
    }

    if (images.length === 2) {
        return (
            <div className={styles.galleryPreview2}>
                <ProfilePreviewImg img={images[0]} className={styles.galleryPreviewImg} />
                <ProfilePreviewImg img={images[1]} className={styles.galleryPreviewImg} />
            </div>
        );
    }

    return (
        <div className={styles.galleryPreview3}>
            <ProfilePreviewImg img={images[0]} className={styles.galleryPreviewMain} />
            <div className={styles.galleryPreviewSide}>
                <ProfilePreviewImg img={images[1]} className={styles.galleryPreviewImg} />
                {images[2] && <ProfilePreviewImg img={images[2]} className={styles.galleryPreviewImg} />}
            </div>
        </div>
    );
}
