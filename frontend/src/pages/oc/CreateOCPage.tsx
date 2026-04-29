import React, { useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useOC } from "../../api/queries/oc";
import {
    useAddOCGalleryImage,
    useCreateOC,
    useDeleteOC,
    useDeleteOCGalleryImage,
    useUpdateOC,
    useUploadOCImageById,
} from "../../api/mutations/oc";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Select } from "../../components/Select/Select";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import { MentionTextArea } from "../../components/MentionTextArea/MentionTextArea";
import type { OCDetail, OCImage } from "../../types/api";
import shipStyles from "../ships/ShipPages.module.css";

interface Props {
    mode: "create" | "edit";
}

interface FormProps {
    editing: boolean;
    initial: OCDetail | null;
    id?: string;
}

interface PendingGalleryImage {
    file: File;
    caption: string;
    preview: string;
}

function OCForm({ editing, initial, id }: FormProps) {
    const navigate = useNavigate();
    const [name, setName] = useState(initial?.name ?? "");
    const [description, setDescription] = useState(initial?.description ?? "");
    const [series, setSeries] = useState(initial?.series ?? "umineko");
    const [customSeriesName, setCustomSeriesName] = useState(initial?.custom_series_name ?? "");
    const [imageFile, setImageFile] = useState<File | null>(null);
    const [imagePreview, setImagePreview] = useState<string>(initial?.image_url ?? "");
    const [imageReplaced, setImageReplaced] = useState(false);
    const [existingGallery, setExistingGallery] = useState<OCImage[]>(initial?.gallery ?? []);
    const [removedExistingIDs, setRemovedExistingIDs] = useState<Set<number>>(new Set());
    const [pendingGallery, setPendingGallery] = useState<PendingGalleryImage[]>([]);
    const [galleryFile, setGalleryFile] = useState<File | null>(null);
    const [galleryCaption, setGalleryCaption] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const fileInputRef = useRef<HTMLInputElement>(null);
    const galleryInputRef = useRef<HTMLInputElement>(null);

    const createMutation = useCreateOC();
    const updateMutation = useUpdateOC(id ?? "");
    const deleteMutation = useDeleteOC();
    const uploadImageMutation = useUploadOCImageById();
    const addGalleryMutation = useAddOCGalleryImage();
    const deleteGalleryMutation = useDeleteOCGalleryImage();

    function handleImageChange(e: React.ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (!file) {
            return;
        }
        setImageFile(file);
        setImagePreview(URL.createObjectURL(file));
        setImageReplaced(true);
    }

    function removeImage() {
        setImageFile(null);
        setImagePreview("");
        setImageReplaced(true);
        if (fileInputRef.current) {
            fileInputRef.current.value = "";
        }
    }

    function stageGalleryImage() {
        if (!galleryFile) {
            return;
        }
        setPendingGallery(prev => [
            ...prev,
            { file: galleryFile, caption: galleryCaption, preview: URL.createObjectURL(galleryFile) },
        ]);
        setGalleryFile(null);
        setGalleryCaption("");
        if (galleryInputRef.current) {
            galleryInputRef.current.value = "";
        }
    }

    function unstagePendingImage(index: number) {
        setPendingGallery(prev => prev.filter((_, i) => i !== index));
    }

    function markExistingForRemoval(imageId: number) {
        setRemovedExistingIDs(prev => {
            const next = new Set(prev);
            next.add(imageId);
            return next;
        });
        setExistingGallery(prev => prev.filter(img => img.id !== imageId));
    }

    async function handleDelete() {
        if (!id) {
            return;
        }
        if (!window.confirm("Delete this OC permanently? This cannot be undone.")) {
            return;
        }
        try {
            await deleteMutation.mutateAsync(id);
            navigate("/oc");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to delete oc");
        }
    }

    async function handleSubmit() {
        setError("");
        const trimmedName = name.trim();
        if (!trimmedName) {
            setError("Name is required");
            return;
        }
        if (series === "custom" && !customSeriesName.trim()) {
            setError("Custom series name is required when series is custom");
            return;
        }

        setSubmitting(true);
        try {
            const payload = {
                name: trimmedName,
                description: description.trim(),
                series,
                custom_series_name: series === "custom" ? customSeriesName.trim() : "",
            };
            let targetId = id ?? "";
            if (editing) {
                await updateMutation.mutateAsync(payload);
            } else {
                const result = await createMutation.mutateAsync(payload);
                targetId = result.id;
            }
            if (imageReplaced && imageFile && targetId) {
                try {
                    await uploadImageMutation.mutateAsync({ id: targetId, file: imageFile });
                } catch {
                    void 0;
                }
            }
            for (const removedID of removedExistingIDs) {
                try {
                    await deleteGalleryMutation.mutateAsync({ ocId: targetId, imageId: removedID });
                } catch {
                    void 0;
                }
            }
            for (const item of pendingGallery) {
                try {
                    await addGalleryMutation.mutateAsync({ id: targetId, file: item.file, caption: item.caption });
                } catch {
                    void 0;
                }
            }
            navigate(`/oc/${targetId}`);
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to save oc");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <div className={shipStyles.page}>
            <h1 className={shipStyles.heading}>{editing ? "Edit OC" : "New OC"}</h1>

            {error && <ErrorBanner message={error} />}

            <div style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
                <label>
                    Name
                    <Input type="text" value={name} onChange={e => setName(e.target.value)} fullWidth />
                </label>

                <label>
                    Series
                    <Select value={series} onChange={e => setSeries(e.target.value)}>
                        <option value="umineko">Umineko</option>
                        <option value="higurashi">Higurashi</option>
                        <option value="ciconia">Ciconia</option>
                        <option value="custom">Custom</option>
                    </Select>
                </label>

                {series === "custom" && (
                    <label>
                        Custom series name
                        <Input
                            type="text"
                            value={customSeriesName}
                            onChange={e => setCustomSeriesName(e.target.value)}
                            placeholder="e.g. Higanbana, Rose Guns Days, original universe..."
                            fullWidth
                        />
                    </label>
                )}

                <label>
                    Description
                    <MentionTextArea
                        value={description}
                        onChange={setDescription}
                        placeholder="Tell us about this OC. **bold**, *italic*, > quotes, [links](url)."
                    />
                </label>

                <div>
                    <label>Main image (optional)</label>
                    <input ref={fileInputRef} type="file" accept="image/*" onChange={handleImageChange} hidden />
                    <Button variant="ghost" size="small" onClick={() => fileInputRef.current?.click()}>
                        {imagePreview ? "Replace" : "+ Media"}
                    </Button>
                    {imagePreview && (
                        <div style={{ marginTop: "0.5rem" }}>
                            <img
                                src={imagePreview}
                                alt="preview"
                                style={{
                                    maxWidth: "100%",
                                    maxHeight: "200px",
                                    borderRadius: "6px",
                                    display: "block",
                                }}
                            />
                            <Button variant="ghost" size="small" onClick={removeImage}>
                                Remove
                            </Button>
                        </div>
                    )}
                </div>

                <div>
                    <label>Gallery</label>
                    {(existingGallery.length > 0 || pendingGallery.length > 0) && (
                        <div
                            style={{
                                display: "grid",
                                gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))",
                                gap: "0.75rem",
                                marginTop: "0.5rem",
                            }}
                        >
                            {existingGallery.map(img => (
                                <figure
                                    key={`existing-${img.id}`}
                                    style={{ margin: 0, display: "flex", flexDirection: "column", gap: "0.35rem" }}
                                >
                                    <img
                                        src={img.thumbnail_url || img.image_url}
                                        alt={img.caption ?? ""}
                                        style={{ width: "100%", borderRadius: "6px" }}
                                    />
                                    {img.caption && (
                                        <figcaption style={{ fontSize: "0.85rem" }}>{img.caption}</figcaption>
                                    )}
                                    <Button variant="ghost" size="small" onClick={() => markExistingForRemoval(img.id)}>
                                        Remove
                                    </Button>
                                </figure>
                            ))}
                            {pendingGallery.map((item, idx) => (
                                <figure
                                    key={`pending-${idx}`}
                                    style={{ margin: 0, display: "flex", flexDirection: "column", gap: "0.35rem" }}
                                >
                                    <img
                                        src={item.preview}
                                        alt={item.caption}
                                        style={{ width: "100%", borderRadius: "6px" }}
                                    />
                                    <figcaption style={{ fontSize: "0.85rem", fontStyle: "italic" }}>
                                        {item.caption || "(no caption)"} - pending upload
                                    </figcaption>
                                    <Button variant="ghost" size="small" onClick={() => unstagePendingImage(idx)}>
                                        Remove
                                    </Button>
                                </figure>
                            ))}
                        </div>
                    )}
                    <div
                        style={{
                            marginTop: "0.75rem",
                            display: "flex",
                            flexDirection: "column",
                            gap: "0.5rem",
                            maxWidth: "400px",
                        }}
                    >
                        <input
                            ref={galleryInputRef}
                            type="file"
                            accept="image/*"
                            onChange={e => setGalleryFile(e.target.files?.[0] ?? null)}
                            hidden
                        />
                        <Button variant="ghost" size="small" onClick={() => galleryInputRef.current?.click()}>
                            {galleryFile ? `Selected: ${galleryFile.name}` : "+ Media"}
                        </Button>
                        <Input
                            type="text"
                            placeholder="Optional caption"
                            value={galleryCaption}
                            onChange={e => setGalleryCaption(e.target.value)}
                        />
                        <Button variant="primary" size="small" onClick={stageGalleryImage} disabled={!galleryFile}>
                            Add to gallery
                        </Button>
                    </div>
                </div>

                <div style={{ display: "flex", gap: "0.5rem" }}>
                    <Button variant="primary" onClick={handleSubmit} disabled={submitting}>
                        {submitting ? "Saving..." : editing ? "Save changes" : "Create OC"}
                    </Button>
                    {editing && (
                        <Button variant="danger" onClick={handleDelete}>
                            Delete OC
                        </Button>
                    )}
                </div>
            </div>
        </div>
    );
}

export function CreateOCPage({ mode }: Props) {
    const { id } = useParams<{ id: string }>();
    const editing = mode === "edit" && !!id;
    usePageTitle(editing ? "Edit OC" : "New OC");

    const { oc, loading } = useOC(editing ? (id ?? "") : "");

    if (editing && loading) {
        return (
            <div className={shipStyles.page}>
                <div className="loading">Loading OC...</div>
            </div>
        );
    }

    if (editing && !oc) {
        return (
            <div className={shipStyles.page}>
                <div className="empty-state">OC not found.</div>
            </div>
        );
    }

    return <OCForm key={oc?.id ?? "new"} editing={editing} initial={oc ?? null} id={id} />;
}
