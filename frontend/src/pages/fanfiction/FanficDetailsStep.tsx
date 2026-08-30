import { useRef } from "react";
import { Button } from "../../components/Button/Button";
import { CharacterPicker } from "../../components/CharacterPicker/CharacterPicker";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import { Input } from "../../components/Input/Input";
import { MentionTextArea } from "../../components/MentionTextArea/MentionTextArea";
import { Select } from "../../components/Select/Select";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import { GENRES, MAX_TAG_LENGTH, OTHER_VALUE } from "../../domain/fanfic/form";
import type { FanficDetailsStep as FanficDetailsStepState } from "../../hooks/useFanficEditor";
import styles from "./FanficPages.module.css";

interface FanficDetailsStepProps {
    isEdit: boolean;
    error: string;
    submitting: boolean;
    details: FanficDetailsStepState;
    onBack: () => void;
    onCancel: () => void;
}

export function FanficDetailsStep({ isEdit, error, submitting, details, onBack, onCancel }: FanficDetailsStepProps) {
    const { fields, setField } = details;
    const coverInputRef = useRef<HTMLInputElement>(null);
    const showCustomSeries = fields.series === OTHER_VALUE;
    const showCustomLanguage = fields.language === OTHER_VALUE;

    function openCoverPicker() {
        coverInputRef.current?.click();
    }

    function removeCover() {
        if (coverInputRef.current) {
            coverInputRef.current.value = "";
        }
        details.removeCover();
    }

    return (
        <div className={styles.formPage}>
            <span className={styles.back} onClick={onBack}>
                &larr; {isEdit ? "Back to Fanfic" : "All Fanfiction"}
            </span>
            <h1 className={styles.formHeading}>{isEdit ? "Edit Fanfic" : "New Fanfic"}</h1>

            <div className={styles.formRow}>
                <label className={styles.formLabel}>Title</label>
                <Input
                    type="text"
                    value={fields.title}
                    onChange={e => setField("title", e.target.value)}
                    placeholder="Your fanfic title..."
                    fullWidth
                />
            </div>

            <div className={styles.formRow}>
                <label className={styles.formLabel}>Summary</label>
                <MentionTextArea
                    value={fields.summary}
                    onChange={value => setField("summary", value)}
                    placeholder="Brief summary of your story..."
                    rows={3}
                />
            </div>

            <div className={styles.formRowDouble}>
                <div>
                    <label className={styles.formLabel}>Series</label>
                    <Select
                        value={showCustomSeries ? OTHER_VALUE : fields.series}
                        onChange={e => details.chooseSeries(e.target.value)}
                    >
                        {details.series.map(s => (
                            <option key={s} value={s}>
                                {s}
                            </option>
                        ))}
                        <option value={OTHER_VALUE}>Other...</option>
                    </Select>
                    {showCustomSeries && (
                        <Input
                            type="text"
                            value={fields.customSeries}
                            onChange={e => setField("customSeries", e.target.value)}
                            placeholder="Enter series name..."
                            fullWidth
                            style={{ marginTop: "0.5rem" }}
                        />
                    )}
                </div>
                <div>
                    <label className={styles.formLabel}>Rating</label>
                    <Select value={fields.rating} onChange={e => setField("rating", e.target.value)}>
                        <option value="K">K - All ages</option>
                        <option value="K+">K+ - 9 and older</option>
                        <option value="T">T - Teens</option>
                        <option value="M">M - Mature</option>
                    </Select>
                </div>
            </div>

            <div className={styles.formRowDouble}>
                <div>
                    <label className={styles.formLabel}>Language</label>
                    <Select
                        value={showCustomLanguage ? OTHER_VALUE : fields.language}
                        onChange={e => details.chooseLanguage(e.target.value)}
                    >
                        {details.languages.map(l => (
                            <option key={l} value={l}>
                                {l}
                            </option>
                        ))}
                        {!details.languages.includes("English") && <option value="English">English</option>}
                        <option value={OTHER_VALUE}>Other...</option>
                    </Select>
                    {showCustomLanguage && (
                        <Input
                            type="text"
                            value={fields.customLanguage}
                            onChange={e => setField("customLanguage", e.target.value)}
                            placeholder="Enter language..."
                            fullWidth
                            style={{ marginTop: "0.5rem" }}
                        />
                    )}
                </div>
                <div>
                    <label className={styles.formLabel}>Genre A</label>
                    <Select value={fields.genreA} onChange={e => setField("genreA", e.target.value)}>
                        <option value="">-- select genre --</option>
                        {GENRES.map(g => (
                            <option key={g} value={g}>
                                {g}
                            </option>
                        ))}
                    </Select>
                </div>
            </div>

            <div className={styles.formRowDouble}>
                <div>
                    <label className={styles.formLabel}>Genre B (optional)</label>
                    <Select value={fields.genreB} onChange={e => setField("genreB", e.target.value)}>
                        <option value="">-- select genre --</option>
                        {GENRES.map(g => (
                            <option key={g} value={g}>
                                {g}
                            </option>
                        ))}
                    </Select>
                </div>
                <div />
            </div>

            <div className={styles.formRow}>
                <label className={styles.formLabel}>Characters</label>
                <CharacterPicker onAdd={details.addCharacter} existing={fields.characters} />
                {fields.characters.length > 0 && (
                    <div className={styles.charList}>
                        {fields.characters.map((c, i) => (
                            <span
                                key={`${c.series}-${c.character_id ?? c.character_name}-${i}`}
                                className={styles.charPill}
                            >
                                <span dir="auto">{c.character_name}</span>
                                <button
                                    type="button"
                                    className={styles.charPillRemove}
                                    onClick={() => details.removeCharacter(i)}
                                    aria-label="Remove character"
                                >
                                    &times;
                                </button>
                            </span>
                        ))}
                    </div>
                )}
            </div>

            <ToggleSwitch
                enabled={fields.isPairing}
                onChange={value => setField("isPairing", value)}
                label="Pairing / ship fic"
                description="These characters are in a relationship"
            />

            <ToggleSwitch
                enabled={fields.isOneshot}
                onChange={details.oneshotLocked ? () => {} : value => setField("isOneshot", value)}
                label="One-shot"
                description={
                    details.oneshotLocked
                        ? `Cannot switch to one-shot with ${details.chapterCount} chapters. Delete extra chapters first.`
                        : "Single chapter story. Turn off to add chapters after creation."
                }
            />

            <ToggleSwitch
                enabled={fields.containsLemons}
                onChange={value => setField("containsLemons", value)}
                label="Contains lemons"
                description="This story contains explicit content. It will be hidden by default."
            />

            <div className={styles.formRow}>
                <label className={styles.formLabel}>Status</label>
                <Select value={fields.status} onChange={e => setField("status", e.target.value)}>
                    {isEdit && <option value="draft">Draft</option>}
                    <option value="in_progress">In Progress</option>
                    <option value="complete">Complete</option>
                </Select>
            </div>

            <div className={styles.formRow}>
                <label className={styles.formLabel}>Tags (up to 10)</label>
                <p style={{ color: "var(--text-muted)", fontSize: "0.85rem", marginBottom: "0.5rem" }}>
                    Content warnings, themes, or anything you want readers to see at a glance.
                </p>
                <div style={{ display: "flex", gap: "0.5rem" }}>
                    <Input
                        type="text"
                        value={details.tagInput}
                        onChange={e => details.setTagInput(e.target.value)}
                        onKeyDown={e => {
                            if (e.key === "Enter" || e.key === ",") {
                                e.preventDefault();
                                details.addTag();
                            }
                        }}
                        placeholder="Type a tag and press Enter..."
                        fullWidth
                        maxLength={MAX_TAG_LENGTH}
                    />
                </div>
                {fields.tags.length > 0 && (
                    <div className={styles.charList} style={{ marginTop: "0.5rem" }}>
                        {fields.tags.map((t, i) => (
                            <span key={`${t}-${i}`} className={styles.tagPill}>
                                <span dir="auto">{t}</span>
                                <button
                                    type="button"
                                    className={styles.charPillRemove}
                                    onClick={() => details.removeTag(i)}
                                    aria-label="Remove tag"
                                >
                                    &times;
                                </button>
                            </span>
                        ))}
                    </div>
                )}
            </div>

            <div className={styles.formRow}>
                <label className={styles.formLabel}>Cover image (optional)</label>
                <input ref={coverInputRef} type="file" accept="image/*" onChange={details.chooseCover} hidden />
                <Button variant="ghost" size="small" onClick={openCoverPicker}>
                    + Cover Image
                </Button>
                {fields.coverPreview && (
                    <div style={{ marginTop: "0.5rem" }}>
                        <img
                            src={fields.coverPreview}
                            alt="preview"
                            style={{ maxWidth: "100%", maxHeight: "200px", borderRadius: "6px", display: "block" }}
                        />
                        <Button variant="ghost" size="small" onClick={removeCover}>
                            Remove
                        </Button>
                    </div>
                )}
            </div>

            {error && <ErrorBanner message={error} />}

            <div className={styles.formActions}>
                <Button variant="ghost" onClick={onCancel}>
                    Cancel
                </Button>
                {isEdit && !fields.isOneshot ? (
                    <Button variant="primary" onClick={details.save} disabled={submitting || !fields.title.trim()}>
                        {submitting ? "Saving..." : "Save Changes"}
                    </Button>
                ) : (
                    <Button variant="primary" onClick={details.next}>
                        Next: {fields.isOneshot ? "Edit Story" : "Write Story"}
                    </Button>
                )}
            </div>
        </div>
    );
}
