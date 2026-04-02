import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useProfile } from "../../hooks/useProfile";
import { useTheoryFeed } from "../../hooks/useTheoryFeed";
import { useFollow } from "../../hooks/useFollow";
import {
    createGallery,
    getFollowers,
    getFollowing,
    getUserActivity,
    getUserArt,
    getUserGalleries,
    getUserPosts,
} from "../../api/endpoints";
import type { ActivityItem, Art, Gallery, Post, User } from "../../types/api";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { TheoryCard } from "../../components/theory/TheoryCard/TheoryCard";
import { PostCard } from "../../components/post/PostCard/PostCard";
import { ArtGrid } from "../../components/art/ArtGrid/ArtGrid";
import { Pagination } from "../../components/Pagination/Pagination";
import { RolePill } from "../../components/RolePill/RolePill";
import { RoleStyledName } from "../../components/RoleStyledName/RoleStyledName";
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

type TabType = "posts" | "theories" | "art" | "galleries" | "activity" | "followers" | "following";

export function ProfilePage() {
    const { username } = useParams<{ username: string }>();
    const navigate = useNavigate();
    const { user: currentUser } = useAuth();
    const { profile, loading } = useProfile(username ?? "");
    const [activeTab, setActiveTab] = useState<TabType>("posts");
    const follow = useFollow(profile?.id ?? "");

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

    const [userPosts, setUserPosts] = useState<Post[]>([]);
    const [postsTotal, setPostsTotal] = useState(0);
    const [postsOffset, setPostsOffset] = useState(0);
    const [postsLoading, setPostsLoading] = useState(false);
    const postsLimit = 20;

    const fetchPosts = useCallback(async (id: string, off: number) => {
        setPostsLoading(true);
        try {
            const result = await getUserPosts(id, postsLimit, off);
            setUserPosts(result.posts ?? []);
            setPostsTotal(result.total);
        } catch {
            setUserPosts([]);
            setPostsTotal(0);
        } finally {
            setPostsLoading(false);
        }
    }, []);

    useEffect(() => {
        if (activeTab === "posts" && profile?.id) {
            fetchPosts(profile.id, postsOffset);
        }
    }, [activeTab, profile?.id, postsOffset, fetchPosts]);

    const [userArt, setUserArt] = useState<Art[]>([]);
    const [artTotal, setArtTotal] = useState(0);
    const [artOffset, setArtOffset] = useState(0);
    const [artLoading, setArtLoading] = useState(false);
    const artLimit = 24;

    const fetchUserArt = useCallback(async (id: string, off: number) => {
        setArtLoading(true);
        try {
            const result = await getUserArt(id, artLimit, off);
            setUserArt(result.art ?? []);
            setArtTotal(result.total);
        } catch {
            setUserArt([]);
            setArtTotal(0);
        } finally {
            setArtLoading(false);
        }
    }, []);

    useEffect(() => {
        if (activeTab === "art" && profile?.id) {
            fetchUserArt(profile.id, artOffset);
        }
    }, [activeTab, profile?.id, artOffset, fetchUserArt]);

    const [userGalleries, setUserGalleries] = useState<Gallery[]>([]);
    const [galleriesLoading, setGalleriesLoading] = useState(false);

    useEffect(() => {
        if (activeTab === "galleries" && profile?.id) {
            setGalleriesLoading(true);
            getUserGalleries(profile.id)
                .then(g => setUserGalleries(g ?? []))
                .catch(() => setUserGalleries([]))
                .finally(() => setGalleriesLoading(false));
        }
    }, [activeTab, profile?.id]);

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

    const [followList, setFollowList] = useState<User[]>([]);
    const [followListLoading, setFollowListLoading] = useState(false);

    useEffect(() => {
        if ((activeTab === "followers" || activeTab === "following") && profile?.id) {
            setFollowListLoading(true);
            const fn = activeTab === "followers" ? getFollowers : getFollowing;
            fn(profile.id, 200, 0)
                .then(r => setFollowList(r.users ?? []))
                .catch(() => setFollowList([]))
                .finally(() => setFollowListLoading(false));
        }
    }, [activeTab, profile?.id]);

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
                        <RoleStyledName name={profile.display_name} role={profile.role} />
                        {profile.role && <RolePill role={profile.role} />}
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
                            {follow.stats.follows_you && <span className={styles.followsYou}>Follows you</span>}
                        </div>
                    )}
                    <div className={styles.metaRow}>
                        {showGender && <span className={styles.metaItem}>{profile.gender}</span>}
                        {profile.pronoun_subject && profile.pronoun_possessive && (
                            <span className={styles.metaItem}>
                                {profile.pronoun_subject}/{profile.pronoun_possessive}
                            </span>
                        )}
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
                        userPosts.map(p => (
                            <PostCard key={p.id} post={p} onDelete={() => fetchPosts(profile.id, postsOffset)} />
                        ))}
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
                        <CreateGalleryInline
                            onCreated={() => {
                                getUserGalleries(profile.id)
                                    .then(g => setUserGalleries(g ?? []))
                                    .catch(() => {});
                            }}
                        />
                    )}
                    {galleriesLoading && <div className="loading">Loading galleries...</div>}
                    {!galleriesLoading && userGalleries.length === 0 && (
                        <div className="empty-state">No galleries yet.</div>
                    )}
                    {!galleriesLoading && (
                        <div className={styles.galleryGrid}>
                            {userGalleries.map(g => (
                                <div
                                    key={g.id}
                                    className={styles.galleryCard}
                                    onClick={() => navigate(`/gallery/view/${g.id}`)}
                                >
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
                                </div>
                            ))}
                        </div>
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
    const [submitting, setSubmitting] = useState(false);

    async function handleCreate() {
        if (!name.trim() || submitting) {
            return;
        }
        setSubmitting(true);
        try {
            await createGallery(name.trim(), description.trim());
            setName("");
            setDescription("");
            setOpen(false);
            onCreated();
        } finally {
            setSubmitting(false);
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
