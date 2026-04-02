import {useCallback, useEffect, useRef, useState} from "react";
import {useSearchParams} from "react-router";
import {useAuth} from "../../hooks/useAuth";
import {useArtFeed} from "../../hooks/useArtFeed";
import {createGallery, getPopularTags, getUserGalleries} from "../../api/endpoints";
import type {Gallery, TagCount} from "../../types/api";
import {ArtGrid} from "../../components/art/ArtGrid/ArtGrid";
import {ArtUploadForm} from "../../components/art/ArtUploadForm/ArtUploadForm";
import {Pagination} from "../../components/Pagination/Pagination";
import {Input} from "../../components/Input/Input";
import {Button} from "../../components/Button/Button";
import {RulesBox} from "../../components/RulesBox/RulesBox";
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
    const { user } = useAuth();
    const [searchParams, setSearchParams] = useSearchParams();

    const sort = (searchParams.get("sort") as ArtSort) || "new";
    const search = searchParams.get("search") || "";
    const activeTag = searchParams.get("tag") || "";
    const page = parseInt(searchParams.get("page") || "1", 10);

    const [searchInput, setSearchInput] = useState(search);
    const [popularTags, setPopularTags] = useState<TagCount[]>([]);
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    const [refreshKey, setRefreshKey] = useState(0);

    const [galleries, setGalleries] = useState<Gallery[]>([]);
    const [selectedGallery, setSelectedGallery] = useState("");
    const [showUpload, setShowUpload] = useState(false);
    const [creatingGallery, setCreatingGallery] = useState(false);
    const [newGalleryName, setNewGalleryName] = useState("");

    const feed = useArtFeed(corner, search || undefined, activeTag || undefined, sort, page, refreshKey);

    function refresh() {
        setRefreshKey(k => k + 1);
    }

    useEffect(() => {
        getPopularTags(corner).then(setPopularTags).catch(() => setPopularTags([]));
    }, [corner]);

    useEffect(() => {
        if (user?.id) {
            getUserGalleries(user.id).then(g => {
                setGalleries(g ?? []);
                if (g && g.length > 0) {
                    setSelectedGallery(prev => prev || g[0].id);
                }
            }).catch(() => {});
        }
    }, [user?.id]);

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
                setGalleries(updated ?? []);
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
                            !(key === "page" && value === "1")
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

    return (
        <div className={styles.page}>
            {CORNER_TITLES[corner] && <h1 className={styles.cornerTitle}>{CORNER_TITLES[corner]}</h1>}
            {!CORNER_TITLES[corner] && <h1 className={styles.cornerTitle}>Gallery</h1>}
            <RulesBox page={CORNER_RULES[corner] || "gallery"} />

            <div className={styles.controls}>
                <Input
                    type="text"
                    placeholder="Search art..."
                    value={searchInput}
                    onChange={e => setSearchInput(e.target.value)}
                    className={styles.searchInput}
                />
                {user && (
                    <Button
                        variant="primary"
                        size="small"
                        onClick={() => setShowUpload(prev => !prev)}
                    >
                        {showUpload ? "Cancel" : "Upload Art"}
                    </Button>
                )}
            </div>

            {showUpload && user && (
                <div className={styles.uploadSection}>
                    {galleries.length === 0 ? (
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
                            galleries={galleries}
                            selectedGallery={selectedGallery}
                            onGalleryChange={setSelectedGallery}
                        />
                    ) : null}
                </div>
            )}

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
        </div>
    );
}
