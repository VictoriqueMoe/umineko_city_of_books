import { useRef, useState } from "react";
import { useNavigate } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { ShipCharacter } from "../../types/api";
import { useCreateShip, useUploadShipImageById } from "../../api/mutations/ship";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { CharacterPicker } from "../../components/CharacterPicker/CharacterPicker";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import { MentionTextArea } from "../../components/MentionTextArea/MentionTextArea";
import styles from "./ShipPages.module.css";

export function CreateShipPage() {
    usePageTitle("New Ship");
    const navigate = useNavigate();
    const [title, setTitle] = useState("");
    const [description, setDescription] = useState("");
    const [characters, setCharacters] = useState<ShipCharacter[]>([]);
    const [imageFile, setImageFile] = useState<File | null>(null);
    const [imagePreview, setImagePreview] = useState<string>("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const fileInputRef = useRef<HTMLInputElement>(null);
    const createShipMutation = useCreateShip();
    const uploadImageMutation = useUploadShipImageById();

    function addCharacter(character: ShipCharacter) {
        setCharacters(prev => [...prev, { ...character, sort_order: prev.length }]);
    }

    function removeCharacter(index: number) {
        setCharacters(prev => prev.filter((_, i) => i !== index).map((c, i) => ({ ...c, sort_order: i })));
    }

    function handleImageChange(e: React.ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (!file) {
            return;
        }
        setImageFile(file);
        setImagePreview(URL.createObjectURL(file));
    }

    function removeImage() {
        setImageFile(null);
        setImagePreview("");
        if (fileInputRef.current) {
            fileInputRef.current.value = "";
        }
    }

    async function handleSubmit() {
        setError("");
        if (!title.trim()) {
            setError("Title is required");
            return;
        }
        if (characters.length < 2) {
            setError("A ship needs at least 2 characters");
            return;
        }

        setSubmitting(true);
        try {
            const result = await createShipMutation.mutateAsync({
                title: title.trim(),
                description: description.trim(),
                characters,
            });
            if (imageFile) {
                try {
                    await uploadImageMutation.mutateAsync({ id: result.id, file: imageFile });
                } catch {
                    void 0;
                }
            }
            navigate(`/ships/${result.id}`);
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to create ship");
        } finally {
            setSubmitting(false);
        }
    }

    function characterPillClass(series: string): string {
        if (series === "umineko") {
            return `${styles.selectedCharacter} ${styles.characterPillUmineko}`;
        }
        if (series === "higurashi") {
            return `${styles.selectedCharacter} ${styles.characterPillHigurashi}`;
        }
        return `${styles.selectedCharacter} ${styles.characterPillOc}`;
    }

    return (
        <div className={styles.formPage}>
            <span className={styles.back} onClick={() => navigate("/ships")}>
                &larr; All Ships
            </span>
            <h1 className={styles.formHeading}>Declare a Ship</h1>

            <div className={styles.formSection}>
                <label className={styles.formLabel}>Ship Title</label>
                <Input
                    type="text"
                    value={title}
                    onChange={e => setTitle(e.target.value)}
                    placeholder="e.g. Battler × Beatrice"
                    fullWidth
                />
            </div>

            <div className={styles.formSection}>
                <label className={styles.formLabel}>Characters (at least 2)</label>
                <CharacterPicker onAdd={addCharacter} existing={characters} />
                {characters.length > 0 && (
                    <div className={styles.selectedCharacters}>
                        {characters.map((c, i) => (
                            <span
                                key={`${c.series}-${c.character_id ?? c.character_name}-${i}`}
                                className={characterPillClass(c.series)}
                            >
                                {c.character_name}
                                <button
                                    type="button"
                                    className={styles.removeCharBtn}
                                    onClick={() => removeCharacter(i)}
                                    aria-label="Remove character"
                                >
                                    ×
                                </button>
                            </span>
                        ))}
                    </div>
                )}
            </div>

            <div className={styles.formSection}>
                <label className={styles.formLabel}>Why do you ship it? (optional)</label>
                <MentionTextArea
                    value={description}
                    onChange={setDescription}
                    placeholder="Tell us why this pairing works..."
                    rows={5}
                    showColours
                />
            </div>

            <div className={styles.formSection}>
                <label className={styles.formLabel}>Ship Image (optional)</label>
                <input ref={fileInputRef} type="file" accept="image/*" onChange={handleImageChange} hidden />
                <Button variant="ghost" size="small" onClick={() => fileInputRef.current?.click()}>
                    + Media
                </Button>
                {imagePreview && (
                    <div style={{ marginTop: "0.5rem" }}>
                        <img
                            src={imagePreview}
                            alt="preview"
                            style={{ maxWidth: "100%", maxHeight: "200px", borderRadius: "6px", display: "block" }}
                        />
                        <Button variant="ghost" size="small" onClick={removeImage}>
                            Remove
                        </Button>
                    </div>
                )}
            </div>

            {error && <ErrorBanner message={error} />}

            <div className={styles.formActions}>
                <Button variant="ghost" onClick={() => navigate("/ships")}>
                    Cancel
                </Button>
                <Button
                    variant="primary"
                    onClick={handleSubmit}
                    disabled={submitting || !title.trim() || characters.length < 2}
                >
                    {submitting ? "Creating..." : "Declare Ship"}
                </Button>
            </div>
        </div>
    );
}
