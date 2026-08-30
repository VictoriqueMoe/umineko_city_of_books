import { useNavigate } from "react-router";
import { useArtGallery } from "../../hooks/useArtGallery";
import { hasNextPage } from "../../hooks/usePageOffset";
import { usePageTitle } from "../../hooks/usePageTitle";
import { ArtGrid } from "../../components/art/ArtGrid/ArtGrid";
import { ArtUploadForm } from "../../components/art/ArtUploadForm/ArtUploadForm";
import { GalleryCard } from "../../components/art/GalleryCard/GalleryCard";
import { Pagination } from "../../components/Pagination/Pagination";
import { Input } from "../../components/Input/Input";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import { PieceTrigger } from "../../components/easterEgg";
import styles from "./ArtGalleryPage.module.css";

const SORT_OPTIONS: { value: string; label: string }[] = [
    { value: "new", label: "New" },
    { value: "popular", label: "Popular" },
    { value: "views", label: "Most Viewed" },
];

const TYPE_OPTIONS: { value: string; label: string }[] = [
    { value: "", label: "All" },
    { value: "drawing", label: "Drawing" },
    { value: "cosplay", label: "Cosplay" },
    { value: "figure", label: "Figure" },
    { value: "other", label: "Other" },
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
    const navigate = useNavigate();
    const gallery = useArtGallery(corner);
    const { viewer, page, updateParams } = gallery;

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
                        onClick={() => (viewer ? navigate(`/user/${viewer.username}`) : navigate("/login"))}
                    >
                        profile
                    </span>{" "}
                    (Galleries tab), then upload art into it. You can also upload directly using the Upload Art button
                    above. Share your drawings, cosplay photos, figure collections, and more. Use the &quot;All
                    Art&quot; view to filter by type. <PieceTrigger pieceId="piece_09" />
                </p>
            </InfoPanel>

            <div className={styles.controls}>
                <div className={styles.viewToggle}>
                    <button
                        className={`${styles.toggleBtn}${gallery.viewMode === "galleries" ? ` ${styles.toggleBtnActive}` : ""}`}
                        onClick={() => updateParams({ view: "galleries", page: "1" })}
                    >
                        By Artist
                    </button>
                    <button
                        className={`${styles.toggleBtn}${gallery.viewMode === "all" ? ` ${styles.toggleBtnActive}` : ""}`}
                        onClick={() => updateParams({ view: "all", page: "1" })}
                    >
                        All Art
                    </button>
                </div>
                {gallery.viewMode === "all" && (
                    <Input
                        type="text"
                        placeholder="Search art..."
                        value={gallery.searchInput}
                        onChange={e => gallery.setSearchInput(e.target.value)}
                        className={styles.searchInput}
                    />
                )}
                {viewer && (
                    <Button variant="primary" size="small" onClick={() => gallery.setShowUpload(!gallery.showUpload)}>
                        {gallery.showUpload ? "Cancel" : "Upload Art"}
                    </Button>
                )}
            </div>

            {gallery.showUpload && viewer && (
                <div className={styles.uploadSection}>
                    {gallery.userGalleries.length === 0 ? (
                        <div className={styles.createGalleryPrompt}>
                            <p>You need a gallery first. Create one to start uploading art.</p>
                            <div className={styles.createGalleryRow}>
                                <input
                                    dir="auto"
                                    className={styles.createGalleryInput}
                                    type="text"
                                    placeholder="Gallery name"
                                    value={gallery.newGalleryName}
                                    onChange={e => gallery.setNewGalleryName(e.target.value)}
                                />
                                <Button
                                    variant="primary"
                                    size="small"
                                    onClick={gallery.createGallery}
                                    disabled={!gallery.newGalleryName.trim() || gallery.creatingGallery}
                                >
                                    {gallery.creatingGallery ? "Creating..." : "Create"}
                                </Button>
                            </div>
                            {gallery.galleryError && <ErrorBanner message={gallery.galleryError} />}
                        </div>
                    ) : gallery.selectedGallery ? (
                        <ArtUploadForm
                            galleryId={gallery.selectedGallery}
                            corner={corner}
                            inline
                            onCreated={() => gallery.setShowUpload(false)}
                            galleries={gallery.userGalleries}
                            selectedGallery={gallery.selectedGallery}
                            onGalleryChange={gallery.setSelectedGallery}
                        />
                    ) : null}
                </div>
            )}

            {gallery.viewMode === "galleries" && (
                <>
                    {gallery.galleriesLoading && <div className="loading">Loading galleries...</div>}
                    {!gallery.galleriesLoading && gallery.artists.length === 0 && (
                        <div className="empty-state">No galleries yet. Be the first to create one.</div>
                    )}
                    {!gallery.galleriesLoading && (
                        <div className={styles.artistList}>
                            {gallery.artists.map(({ user: artist, galleries: artistGalleries }) => (
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
                                            <GalleryCard key={g.id} gallery={g} />
                                        ))}
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </>
            )}

            {gallery.viewMode === "all" && (
                <>
                    <div className={styles.sortBar}>
                        {SORT_OPTIONS.map(opt => (
                            <button
                                key={opt.value}
                                className={`${styles.sortBtn}${gallery.sort === opt.value ? ` ${styles.sortBtnActive}` : ""}`}
                                onClick={() => updateParams({ sort: opt.value, page: "1" })}
                            >
                                {opt.label}
                            </button>
                        ))}
                    </div>
                    <div className={styles.typeBar}>
                        <span className={styles.typeLabel}>Type:</span>
                        {TYPE_OPTIONS.map(opt => (
                            <button
                                key={opt.value}
                                className={`${styles.sortBtn}${gallery.activeType === opt.value ? ` ${styles.sortBtnActive}` : ""}`}
                                onClick={() => updateParams({ type: opt.value || undefined, page: "1" })}
                            >
                                {opt.label}
                            </button>
                        ))}
                    </div>

                    {gallery.popularTags.length > 0 && (
                        <div className={styles.tagBar}>
                            {gallery.activeTag && (
                                <button
                                    className={`${styles.tagChip} ${styles.tagChipClear}`}
                                    onClick={() => updateParams({ tag: undefined, page: "1" })}
                                >
                                    Clear filter
                                </button>
                            )}
                            {gallery.popularTags.map(t => (
                                <button
                                    key={t.tag}
                                    className={`${styles.tagChip}${gallery.activeTag === t.tag ? ` ${styles.tagChipActive}` : ""}`}
                                    onClick={() =>
                                        updateParams({
                                            tag: gallery.activeTag === t.tag ? undefined : t.tag,
                                            page: "1",
                                        })
                                    }
                                >
                                    {t.tag} ({t.count})
                                </button>
                            ))}
                        </div>
                    )}

                    {gallery.feedLoading && <div className="loading">Loading gallery...</div>}

                    {!gallery.feedLoading && gallery.art.length === 0 && (
                        <div className="empty-state">
                            {gallery.search || gallery.activeTag
                                ? "No art matches your search."
                                : "No art yet. Be the first to upload."}
                        </div>
                    )}

                    {!gallery.feedLoading && gallery.art.length > 0 && <ArtGrid art={gallery.art} />}

                    {!gallery.feedLoading && (
                        <Pagination
                            offset={page.offset}
                            limit={page.limit}
                            total={gallery.total}
                            hasNext={hasNextPage(page, gallery.total)}
                            hasPrev={page.hasPrev}
                            onNext={page.goNext}
                            onPrev={page.goPrev}
                        />
                    )}
                </>
            )}
        </div>
    );
}
