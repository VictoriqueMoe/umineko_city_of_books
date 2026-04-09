import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useArtFeed } from "../../hooks/useArtFeed";
import { usePageTitle } from "../../hooks/usePageTitle";
import { createGallery, getPopularTags, getUserGalleries, listAllGalleries } from "../../api/endpoints";
import type { Gallery, TagCount } from "../../types/api";
import { ArtGrid } from "../../components/art/ArtGrid/ArtGrid";
import { ArtUploadForm } from "../../components/art/ArtUploadForm/ArtUploadForm";
import { Pagination } from "../../components/Pagination/Pagination";
import { Input } from "../../components/Input/Input";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import styles from "./ArtGalleryPage.module.css";

type ArtSort = "new" | "popular" | "views";

const SORT_OPTIONS: { value: ArtSort; label: string }[] = [
    { value: "new", label: "New" },
    { value: "popular", label: "Popular" },
    { value: "views", label: "Most Viewed" },
];

const CORNER_RULES: Record<string, string> = {
    general: "gallery",
    umineko: "gallery_umineko",
    higurashi: "gallery_higurashi",
    ciconia: "gallery_ciconia",
};

const CORNER_TITLES: Record<string, string> = {
    umineko: "Umineko Gallery",
    higurashi: "Higurashi Gallery",
    ciconia: "Ciconia Gallery",
};

interface ArtGalleryPageProps {
    corner?: string;
}

export function ArtGalleryPage({ corner = "general" }: ArtGalleryPageProps) {
    usePageTitle("Gallery");
    const { user } = useAuth();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();

    const viewMode = searchParams.get("view") || "galleries";
    const sort = (searchParams.get("sort") as ArtSort) || "new";
    const search = searchParams.get("search") || "";
    const activeTag = searchParams.get("tag") || "";
    const activeType = searchParams.get("type") || "";
    const page = parseInt(searchParams.get("page") || "1", 10);

    const [searchInput, setSearchInput] = useState(search);
    const [popularTags, setPopularTags] = useState<TagCount[]>([]);
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    const [refreshKey, setRefreshKey] = useState(0);

    const [userGalleries, setUserGalleries] = useState<Gallery[]>([]);
    const [selectedGallery, setSelectedGallery] = useState("");
    const [showUpload, setShowUpload] = useState(false);
    const [creatingGallery, setCreatingGallery] = useState(false);
    const [newGalleryName, setNewGalleryName] = useState("");

    const [allGalleries, setAllGalleries] = useState<Gallery[]>([]);
    const [galleriesLoading, setGalleriesLoading] = useState(false);

    const feed = useArtFeed(
        corner,
        activeType || undefined,
        search || undefined,
        activeTag || undefined,
        sort,
        page,
        refreshKey,
    );

    function refresh() {
        setRefreshKey(k => k + 1);
    }

    useEffect(() => {
        getPopularTags(corner)
            .then(setPopularTags)
            .catch(() => setPopularTags([]));
    }, [corner]);

    useEffect(() => {
        if (user?.id) {
            getUserGalleries(user.id)
                .then(g => {
                    setUserGalleries(g ?? []);
                    if (g && g.length > 0) {
                        setSelectedGallery(prev => prev || g[0].id);
                    }
                })
                .catch(() => {});
        }
    }, [user?.id]);

    useEffect(() => {
        if (viewMode === "galleries") {
            setGalleriesLoading(true);
            listAllGalleries(corner)
                .then(g => setAllGalleries(g ?? []))
                .catch(() => setAllGalleries([]))
                .finally(() => setGalleriesLoading(false));
        }
    }, [viewMode, corner]);

    async function handleCreateGallery() {
        if (!newGalleryName.trim()) {
            return;
        }
        setCreatingGallery(true);
        try {
            const { id } = await createGallery(newGalleryName.trim());
            setNewGalleryName("");
            if (user?.id) {
                const updated = await getUserGalleries(user.id);
                setUserGalleries(updated ?? []);
                setSelectedGallery(id);
            }
        } finally {
            setCreatingGallery(false);
        }
    }

    const updateParams = useCallback(
        (updates: Record<string, string | undefined>) => {
            setSearchParams(
                prev => {
                    const next = new URLSearchParams(prev);
                    for (const [key, value] of Object.entries(updates)) {
                        if (
                            value &&
                            value !== "" &&
                            !(key === "sort" && value === "new") &&
                            !(key === "page" && value === "1") &&
                            !(key === "view" && value === "galleries")
                        ) {
                            next.set(key, value);
                        } else {
                            next.delete(key);
                        }
                    }
                    return next;
                },
                { replace: true },
            );
        },
        [setSearchParams],
    );

    useEffect(() => {
        if (searchInput === search) {
            return;
        }
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => {
            updateParams({ search: searchInput || undefined, page: "1" });
        }, 300);
        return () => clearTimeout(debounceRef.current);
    }, [searchInput, search, updateParams]);

    const grouped = new Map<string, { user: Gallery["author"]; galleries: Gallery[] }>();
    for (const g of allGalleries) {
        const key = g.author.id;
        if (!grouped.has(key)) {
            grouped.set(key, { user: g.author, galleries: [] });
        }
        grouped.get(key)!.galleries.push(g);
    }
    const sortedGroups = [...grouped.values()].sort((a, b) => a.user.display_name.localeCompare(b.user.display_name));

    return (
        <div className={styles.page}>
            {CORNER_TITLES[corner] && <h1 className={styles.cornerTitle}>{CORNER_TITLES[corner]}</h1>}
            {!CORNER_TITLES[corner] && <h1 className={styles.cornerTitle}>Gallery</h1>}
            <RulesBox page={CORNER_RULES[corner] || "gallery"} />

            <InfoPanel title="How It Works">
                <p>
                    Create a gallery from your{" "}
                    <span
                        style={{ color: "var(--gold)", cursor: "pointer" }}
                        onClick={() => (user ? navigate(`/user/${user.username}`) : navigate("/login"))}
                    >
                        profile
                    </span>{" "}
                    (Galleries tab), then upload art into it. You can also upload directly using the Upload Art button
                    above. Share your drawings, cosplay photos, figure collections, and more. Use the &quot;All
                    Art&quot; view to filter by type.
                </p>
            </InfoPanel>

            <div className={styles.controls}>
                <div className={styles.viewToggle}>
                    <button
                        className={`${styles.toggleBtn}${viewMode === "galleries" ? ` ${styles.toggleBtnActive}` : ""}`}
                        onClick={() => updateParams({ view: "galleries", page: "1" })}
                    >
                        By Artist
                    </button>
                    <button
                        className={`${styles.toggleBtn}${viewMode === "all" ? ` ${styles.toggleBtnActive}` : ""}`}
                        onClick={() => updateParams({ view: "all", page: "1" })}
                    >
                        All Art
                    </button>
                </div>
                {viewMode === "all" && (
                    <Input
                        type="text"
                        placeholder="Search art..."
                        value={searchInput}
                        onChange={e => setSearchInput(e.target.value)}
                        className={styles.searchInput}
                    />
                )}
                {user && (
                    <Button variant="primary" size="small" onClick={() => setShowUpload(prev => !prev)}>
                        {showUpload ? "Cancel" : "Upload Art"}
                    </Button>
                )}
            </div>

            {showUpload && user && (
                <div className={styles.uploadSection}>
                    {userGalleries.length === 0 ? (
                        <div className={styles.createGalleryPrompt}>
                            <p>You need a gallery first. Create one to start uploading art.</p>
                            <div className={styles.createGalleryRow}>
                                <input
                                    className={styles.createGalleryInput}
                                    type="text"
                                    placeholder="Gallery name"
                                    value={newGalleryName}
                                    onChange={e => setNewGalleryName(e.target.value)}
                                />
                                <Button
                                    variant="primary"
                                    size="small"
                                    onClick={handleCreateGallery}
                                    disabled={!newGalleryName.trim() || creatingGallery}
                                >
                                    {creatingGallery ? "Creating..." : "Create"}
                                </Button>
                            </div>
                        </div>
                    ) : selectedGallery ? (
                        <ArtUploadForm
                            galleryId={selectedGallery}
                            corner={corner}
                            inline
                            onCreated={() => {
                                setShowUpload(false);
                                refresh();
                            }}
                            galleries={userGalleries}
                            selectedGallery={selectedGallery}
                            onGalleryChange={setSelectedGallery}
                        />
                    ) : null}
                </div>
            )}

            {viewMode === "galleries" && (
                <>
                    {galleriesLoading && <div className="loading">Loading galleries...</div>}
                    {!galleriesLoading && grouped.size === 0 && (
                        <div className="empty-state">No galleries yet. Be the first to create one.</div>
                    )}
                    {!galleriesLoading && (
                        <div className={styles.artistList}>
                            {sortedGroups.map(({ user: artist, galleries: artistGalleries }) => (
                                <div key={artist.id} className={styles.artistSection}>
                                    <div className={styles.artistHeader}>
                                        <ProfileLink user={artist} size="medium" />
                                        <span className={styles.artistGalleryCount}>
                                            {artistGalleries.length}{" "}
                                            {artistGalleries.length === 1 ? "gallery" : "galleries"}
                                        </span>
                                    </div>
                                    <div className={styles.artistGalleries}>
                                        {artistGalleries.map(g => (
                                            <Link
                                                key={g.id}
                                                to={`/gallery/view/${g.id}`}
                                                className={styles.galleryCard}
                                            >
                                                <div className={styles.galleryCover}>
                                                    {g.cover_thumbnail_url || g.cover_image_url ? (
                                                        <img
                                                            src={g.cover_thumbnail_url || g.cover_image_url}
                                                            alt=""
                                                            className={styles.galleryCoverImage}
                                                            onError={e => {
                                                                if (
                                                                    g.cover_image_url &&
                                                                    e.currentTarget.src !== g.cover_image_url
                                                                ) {
                                                                    e.currentTarget.src = g.cover_image_url;
                                                                }
                                                            }}
                                                        />
                                                    ) : g.preview_images && g.preview_images.length > 0 ? (
                                                        <GalleryPreview images={g.preview_images} />
                                                    ) : (
                                                        <div className={styles.galleryCoverPlaceholder}>Empty</div>
                                                    )}
                                                </div>
                                                <div className={styles.galleryCardInfo}>
                                                    <span className={styles.galleryCardName}>{g.name}</span>
                                                    <span className={styles.galleryCardCount}>
                                                        {g.art_count} pieces
                                                    </span>
                                                </div>
                                            </Link>
                                        ))}
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </>
            )}

            {viewMode === "all" && (
                <>
                    <div className={styles.sortBar}>
                        {SORT_OPTIONS.map(opt => (
                            <button
                                key={opt.value}
                                className={`${styles.sortBtn}${sort === opt.value ? ` ${styles.sortBtnActive}` : ""}`}
                                onClick={() => updateParams({ sort: opt.value, page: "1" })}
                            >
                                {opt.label}
                            </button>
                        ))}
                    </div>
                    <div className={styles.typeBar}>
                        <span className={styles.typeLabel}>Type:</span>
                        {[
                            { value: "", label: "All" },
                            { value: "drawing", label: "Drawing" },
                            { value: "cosplay", label: "Cosplay" },
                            { value: "figure", label: "Figure" },
                            { value: "other", label: "Other" },
                        ].map(opt => (
                            <button
                                key={opt.value}
                                className={`${styles.sortBtn}${activeType === opt.value ? ` ${styles.sortBtnActive}` : ""}`}
                                onClick={() => updateParams({ type: opt.value || undefined, page: "1" })}
                            >
                                {opt.label}
                            </button>
                        ))}
                    </div>

                    {popularTags.length > 0 && (
                        <div className={styles.tagBar}>
                            {activeTag && (
                                <button
                                    className={`${styles.tagChip} ${styles.tagChipClear}`}
                                    onClick={() => updateParams({ tag: undefined, page: "1" })}
                                >
                                    Clear filter
                                </button>
                            )}
                            {popularTags.map(t => (
                                <button
                                    key={t.tag}
                                    className={`${styles.tagChip}${activeTag === t.tag ? ` ${styles.tagChipActive}` : ""}`}
                                    onClick={() =>
                                        updateParams({
                                            tag: activeTag === t.tag ? undefined : t.tag,
                                            page: "1",
                                        })
                                    }
                                >
                                    {t.tag} ({t.count})
                                </button>
                            ))}
                        </div>
                    )}

                    {feed.loading && <div className="loading">Loading gallery...</div>}

                    {!feed.loading && feed.art.length === 0 && (
                        <div className="empty-state">
                            {search || activeTag
                                ? "No art matches your search."
                                : "No art yet. Be the first to upload."}
                        </div>
                    )}

                    {!feed.loading && feed.art.length > 0 && <ArtGrid art={feed.art} />}

                    {!feed.loading && (
                        <Pagination
                            offset={feed.offset}
                            limit={feed.limit}
                            total={feed.total}
                            hasNext={feed.hasNext}
                            hasPrev={feed.hasPrev}
                            onNext={() => updateParams({ page: String(page + 1) })}
                            onPrev={() => updateParams({ page: String(Math.max(1, page - 1)) })}
                        />
                    )}
                </>
            )}
        </div>
    );
}

function PreviewImg({ img, className }: { img: { thumbnail: string; full: string }; className: string }) {
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

function GalleryPreview({ images }: { images: { thumbnail: string; full: string }[] }) {
    if (images.length === 1) {
        return <PreviewImg img={images[0]} className={styles.galleryCoverImage} />;
    }

    if (images.length === 2) {
        return (
            <div className={styles.preview2}>
                <PreviewImg img={images[0]} className={styles.previewImg} />
                <PreviewImg img={images[1]} className={styles.previewImg} />
            </div>
        );
    }

    return (
        <div className={styles.preview3}>
            <PreviewImg img={images[0]} className={styles.previewMain} />
            <div className={styles.previewSide}>
                <PreviewImg img={images[1]} className={styles.previewImg} />
                {images[2] && <PreviewImg img={images[2]} className={styles.previewImg} />}
            </div>
        </div>
    );
}
