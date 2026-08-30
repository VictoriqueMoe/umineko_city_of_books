import { useState } from "react";
import { useCreateGallery } from "../../hooks/mutations/art";
import { Button } from "../../components/Button/Button";
import styles from "./ProfilePage.module.css";

export function CreateGalleryInline({ onCreated }: { onCreated: () => void }) {
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
        <div className={styles.createGalleryCard} style={{ padding: "1rem", cursor: "default", marginBottom: "1rem" }}>
            <input
                type="text"
                dir="auto"
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
                dir="auto"
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
