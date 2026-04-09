import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { Art, Gallery } from "../../types/api";
import {
    deleteGallery as apiDeleteGallery,
    getGallery,
    setArtGallery,
    setGalleryCover,
    updateGallery,
} from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { ArtUploadForm } from "../../components/art/ArtUploadForm/ArtUploadForm";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Button } from "../../components/Button/Button";
import { Modal } from "../../components/Modal/Modal";
import { Pagination } from "../../components/Pagination/Pagination";
import styles from "./GalleryDetailPage.module.css";

export function GalleryDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user } = useAuth();
    const [gallery, setGallery] = useState<Gallery | null>(null);
    usePageTitle(gallery?.name ?? "Gallery");
    const [art, setArt] = useState<Art[]>([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [offset, setOffset] = useState(0);
    const [refreshKey, setRefreshKey] = useState(0);
    const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
    const [editing, setEditing] = useState(false);
    const [editName, setEditName] = useState("");
    const [editDesc, setEditDesc] = useState("");
    const [managing, setManaging] = useState(false);
    const limit = 24;

    useEffect(() => {
        if (!id) {
            return;
        }
        let cancelled = false;
        getGallery(id, limit, offset)
            .then(data => {
                if (!cancelled) {
                    setGallery(data.gallery);
                    setArt(data.art ?? []);
                    setTotal(data.total);
                }
            })
            .catch(() => {
                if (!cancelled) {
                    setGallery(null);
                }
            })
            .finally(() => {
                if (!cancelled) {
                    setLoading(false);
                }
            });
        return () => {
            cancelled = true;
        };
    }, [id, offset, refreshKey]);

    function refresh() {
        setRefreshKey(k => k + 1);
    }

    async function handleDelete() {
        if (!id) {
            return;
        }
        await apiDeleteGallery(id);
        navigate(-1);
    }

    function startEdit() {
        if (!gallery) {
            return;
        }
        setEditName(gallery.name);
        setEditDesc(gallery.description);
        setEditing(true);
    }

    async function saveEdit() {
        if (!id || !editName.trim()) {
            return;
        }
        await updateGallery(id, editName.trim(), editDesc.trim());
        setEditing(false);
        refresh();
    }

    async function handleSetCover(artId: string) {
        if (!id) {
            return;
        }
        await setGalleryCover(id, artId);
        refresh();
    }

    async function handleRemoveArt(artId: string) {
        await setArtGallery(artId, null);
        refresh();
    }

    if (loading) {
        return <div className="loading">Loading gallery...</div>;
    }

    if (!gallery) {
        return <div className="empty-state">Gallery not found.</div>;
    }

    const isOwner = user && user.id === gallery.author.id;

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate(-1)}>
                &larr; Back
            </span>

            <div className={styles.header}>
                {gallery.cover_image_url && (
                    <div className={styles.coverWrap}>
                        <img
                            src={gallery.cover_thumbnail_url || gallery.cover_image_url}
                            alt=""
                            className={styles.coverImage}
                        />
                    </div>
                )}
                <div className={styles.headerInfo}>
                    {editing ? (
                        <div className={styles.editSection}>
                            <input
                                className={styles.editInput}
                                value={editName}
                                onChange={e => setEditName(e.target.value)}
                                placeholder="Gallery name"
                            />
                            <textarea
                                className={styles.editTextarea}
                                value={editDesc}
                                onChange={e => setEditDesc(e.target.value)}
                                placeholder="Description (optional)"
                                rows={2}
                            />
                            <div className={styles.editActions}>
                                <Button variant="secondary" size="small" onClick={() => setEditing(false)}>
                                    Cancel
                                </Button>
                                <Button variant="primary" size="small" onClick={saveEdit} disabled={!editName.trim()}>
                                    Save
                                </Button>
                            </div>
                        </div>
                    ) : (
                        <>
                            <h1 className={styles.name}>{gallery.name}</h1>
                            {gallery.description && <p className={styles.description}>{gallery.description}</p>}
                        </>
                    )}
                    <div className={styles.metaRow}>
                        <ProfileLink user={gallery.author} size="medium" />
                        <span className={styles.artCount}>{gallery.art_count} pieces</span>
                    </div>
                    {isOwner && !editing && (
                        <div className={styles.ownerActions}>
                            <Button variant="secondary" size="small" onClick={startEdit}>
                                Edit
                            </Button>
                            <Button variant="secondary" size="small" onClick={() => setManaging(prev => !prev)}>
                                {managing ? "Done" : "Manage Art"}
                            </Button>
                            <Button variant="danger" size="small" onClick={() => setDeleteConfirmOpen(true)}>
                                Delete
                            </Button>
                        </div>
                    )}
                </div>
            </div>

            {isOwner && id && <ArtUploadForm galleryId={id} onCreated={refresh} />}

            {art.length === 0 && !managing && <div className="empty-state">This gallery is empty.</div>}

            {art.length > 0 && !managing && (
                <div className={styles.grid}>
                    {art.map(a => (
                        <Link key={a.id} to={`/gallery/art/${a.id}`} className={styles.artCard}>
                            <img
                                src={a.thumbnail_url || a.image_url}
                                alt={a.title}
                                className={styles.artImage}
                                loading="lazy"
                                onError={e => {
                                    if (e.currentTarget.src !== a.image_url) {
                                        e.currentTarget.src = a.image_url;
                                    }
                                }}
                            />
                            <div className={styles.artInfo}>
                                <span className={styles.artTitle}>{a.title}</span>
                                <span className={styles.artLikes}>&#9829; {a.like_count}</span>
                            </div>
                        </Link>
                    ))}
                </div>
            )}

            {art.length > 0 && managing && (
                <div className={styles.manageGrid}>
                    {art.map(a => (
                        <div key={a.id} className={styles.manageCard}>
                            <img
                                src={a.thumbnail_url || a.image_url}
                                alt={a.title}
                                className={styles.artImage}
                                loading="lazy"
                                onError={e => {
                                    if (e.currentTarget.src !== a.image_url) {
                                        e.currentTarget.src = a.image_url;
                                    }
                                }}
                            />
                            <div className={styles.manageInfo}>
                                <span className={styles.artTitle}>{a.title}</span>
                                <div className={styles.manageActions}>
                                    <Button variant="secondary" size="small" onClick={() => handleSetCover(a.id)}>
                                        Set as Cover
                                    </Button>
                                    <Button
                                        variant="danger"
                                        size="small"
                                        onClick={() => {
                                            if (window.confirm("Remove this art from the gallery?")) {
                                                handleRemoveArt(a.id);
                                            }
                                        }}
                                    >
                                        Remove
                                    </Button>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {total > limit && (
                <Pagination
                    offset={offset}
                    limit={limit}
                    total={total}
                    hasNext={offset + limit < total}
                    hasPrev={offset > 0}
                    onNext={() => setOffset(offset + limit)}
                    onPrev={() => setOffset(Math.max(0, offset - limit))}
                />
            )}

            <Modal isOpen={deleteConfirmOpen} onClose={() => setDeleteConfirmOpen(false)} title="Delete Gallery">
                <div style={{ padding: "1.25rem" }}>
                    <p style={{ marginBottom: "1rem" }}>
                        Are you sure you want to delete this gallery? All art in it will be permanently deleted.
                    </p>
                    <div className={styles.confirmActions}>
                        <Button variant="secondary" onClick={() => setDeleteConfirmOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="danger" onClick={handleDelete}>
                            Delete Gallery
                        </Button>
                    </div>
                </div>
            </Modal>
        </div>
    );
}
