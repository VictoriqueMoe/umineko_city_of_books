import { useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import type { Gallery } from "../../../types/api";
import { createArt, setArtGallery } from "../../../api/endpoints";
import { useSiteInfo } from "../../../hooks/useSiteInfo";
import { validateFileSize } from "../../../utils/fileValidation";
import { Button } from "../../Button/Button";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import { TagInput } from "../TagInput/TagInput";
import styles from "./ArtUploadForm.module.css";

interface ArtUploadFormProps {
    galleryId: string;
    corner?: string;
    onCreated: () => void;
    inline?: boolean;
    galleries?: Gallery[];
    selectedGallery?: string;
    onGalleryChange?: (id: string) => void;
}

export function ArtUploadForm({
    galleryId,
    corner = "general",
    onCreated,
    inline = false,
    galleries,
    selectedGallery,
    onGalleryChange,
}: ArtUploadFormProps) {
    const navigate = useNavigate();
    const siteInfo = useSiteInfo();
    const [title, setTitle] = useState("");
    const [artType, setArtType] = useState("drawing");
    const [description, setDescription] = useState("");
    const [tags, setTags] = useState<string[]>([]);
    const [file, setFile] = useState<File | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const [open, setOpen] = useState(false);
    const fileInputRef = useRef<HTMLInputElement>(null);

    async function handleSubmit() {
        if (submitting || !title.trim() || !file) {
            return;
        }
        setSubmitting(true);
        setError("");

        try {
            const { id } = await createArt(
                {
                    title: title.trim(),
                    description: description.trim(),
                    corner,
                    art_type: artType,
                    tags,
                    gallery_id: galleryId,
                },
                file,
            );
            await setArtGallery(id, galleryId).catch(() => {});
            onCreated();
            navigate(`/gallery/art/${id}`);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to upload art");
        } finally {
            setSubmitting(false);
        }
    }

    function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
        const selected = e.target.files?.[0];
        if (!selected) {
            return;
        }

        const err = validateFileSize(selected, siteInfo.max_image_size, siteInfo.max_video_size);
        if (err) {
            setError(err);
            return;
        }

        if (!selected.type.startsWith("image/")) {
            setError("Only image files are allowed");
            return;
        }

        setFile(selected);
        setError("");
        e.target.value = "";
    }

    const preview = useMemo(() => (file ? URL.createObjectURL(file) : null), [file]);

    if (!inline && !open) {
        return (
            <div style={{ marginBottom: "1rem" }}>
                <Button variant="primary" size="small" onClick={() => setOpen(true)}>
                    Upload Art
                </Button>
            </div>
        );
    }

    return (
        <div className={styles.form}>
            <h2 className={styles.heading}>Upload Art</h2>

            {error && <div className={styles.error}>{error}</div>}

            {galleries && galleries.length > 0 && onGalleryChange && (
                <div className={styles.field}>
                    <label className={styles.label}>Gallery</label>
                    <select
                        className={styles.input}
                        value={selectedGallery || galleryId}
                        onChange={e => onGalleryChange(e.target.value)}
                    >
                        {galleries.map(g => (
                            <option key={g.id} value={g.id}>
                                {g.name}
                            </option>
                        ))}
                    </select>
                </div>
            )}

            <div className={styles.field}>
                <label className={styles.label}>Type</label>
                <select className={styles.input} value={artType} onChange={e => setArtType(e.target.value)}>
                    <option value="drawing">Drawing</option>
                    <option value="cosplay">Cosplay</option>
                    <option value="figure">Figure</option>
                    <option value="other">Other</option>
                </select>
            </div>

            <div className={styles.field}>
                <label className={styles.label}>Title *</label>
                <input
                    className={styles.input}
                    type="text"
                    value={title}
                    onChange={e => setTitle(e.target.value)}
                    placeholder="Give your art a title"
                    maxLength={200}
                />
            </div>

            <div className={styles.field}>
                <label className={styles.label}>Description</label>
                <MentionTextArea
                    placeholder="Describe your art (optional)"
                    value={description}
                    onChange={setDescription}
                    rows={3}
                />
            </div>

            <div className={styles.field}>
                <label className={styles.label}>Tags</label>
                <TagInput tags={tags} onChange={setTags} />
                <span className={styles.hint}>Press Enter or comma to add. Max 10 tags.</span>
            </div>

            <div className={styles.field}>
                <label className={styles.label}>Image *</label>
                <input ref={fileInputRef} type="file" accept="image/*" onChange={handleFileChange} hidden />
                {preview ? (
                    <div className={styles.previewWrap}>
                        <img src={preview} alt="Preview" className={styles.preview} />
                        <button type="button" className={styles.removeBtn} onClick={() => setFile(null)}>
                            &times; Remove
                        </button>
                    </div>
                ) : (
                    <div className={styles.dropZone} onClick={() => fileInputRef.current?.click()}>
                        Click to select an image
                    </div>
                )}
            </div>

            <div style={{ display: "flex", gap: "0.5rem" }}>
                {!inline && (
                    <Button variant="secondary" onClick={() => setOpen(false)}>
                        Cancel
                    </Button>
                )}
                <Button variant="primary" onClick={handleSubmit} disabled={submitting || !title.trim() || !file}>
                    {submitting ? "Uploading..." : "Upload"}
                </Button>
            </div>
        </div>
    );
}
