import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router";
import type { ShipCharacter } from "../../types/api";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { fanficQueryFns, useFanfic, useFanficLanguages, useFanficSeries } from "../../api/queries/fanfic";
import {
    useCreateFanfic,
    useCreateFanficChapter,
    useDeleteFanficCover,
    useUpdateFanfic,
    useUpdateFanficChapter,
    useUploadFanficCover,
    useUploadFanficCoverFor,
} from "../../api/mutations/fanfic";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Select } from "../../components/Select/Select";
import { MentionTextArea } from "../../components/MentionTextArea/MentionTextArea";
import { CharacterPicker } from "../../components/CharacterPicker/CharacterPicker";
import { RichTextEditor } from "../../components/RichTextEditor/RichTextEditor";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import { can } from "../../utils/permissions";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import styles from "./FanficPages.module.css";

const GENRES = [
    "Adventure",
    "Angst",
    "Crime",
    "Drama",
    "Family",
    "Fantasy",
    "Friendship",
    "General",
    "Horror",
    "Humour",
    "Hurt/Comfort",
    "Mystery",
    "Parody",
    "Poetry",
    "Romance",
    "Sci-Fi",
    "Spiritual",
    "Supernatural",
    "Suspense",
    "Tragedy",
    "Western",
];

const PINNED_SERIES = ["Umineko", "Higurashi", "Ciconia"];
const OTHER_VALUE = "__other__";
const DRAFT_KEY = "fanfic-draft";

interface DraftData {
    title: string;
    summary: string;
    series: string;
    customSeries: string;
    rating: string;
    language: string;
    customLanguage: string;
    genreA: string;
    genreB: string;
    tags: string[];
    status: string;
    characters: ShipCharacter[];
    isPairing: boolean;
    isOneshot: boolean;
    containsLemons: boolean;
    body: string;
    step: number;
}

function loadDraft(): DraftData | null {
    try {
        const raw = localStorage.getItem(DRAFT_KEY);
        if (!raw) {
            return null;
        }
        return JSON.parse(raw) as DraftData;
    } catch {
        return null;
    }
}

function clearDraft() {
    localStorage.removeItem(DRAFT_KEY);
}

export function FanficEditorPage() {
    const { id: editId } = useParams<{ id: string }>();
    const isEdit = !!editId;
    usePageTitle(isEdit ? "Edit Fanfic" : "New Fanfic");
    const navigate = useNavigate();
    const { user } = useAuth();
    const qc = useQueryClient();
    const fileInputRef = useRef<HTMLInputElement>(null);
    const [draftPrompt, setDraftPrompt] = useState<DraftData | null>(() => {
        if (isEdit) {
            return null;
        }
        const existing = loadDraft();
        if (existing && existing.title) {
            return existing;
        }
        return null;
    });
    const [initialised, setInitialised] = useState(() => {
        if (isEdit) {
            return false;
        }
        const existing = loadDraft();
        if (existing && existing.title) {
            return false;
        }
        return true;
    });

    const { series: dynamicSeries } = useFanficSeries();
    const { languages: availableLanguages } = useFanficLanguages();
    const { fanfic: editData, loading: editLoading } = useFanfic(isEdit ? (editId ?? "") : "");

    const updateMutation = useUpdateFanfic(editId ?? "");
    const createMutation = useCreateFanfic();
    const uploadCoverMutation = useUploadFanficCover(editId ?? "");
    const uploadCoverForMutation = useUploadFanficCoverFor();
    const deleteCoverMutation = useDeleteFanficCover(editId ?? "");
    const updateChapterMutation = useUpdateFanficChapter(editId ?? "");
    const createChapterMutation = useCreateFanficChapter(editId ?? "");

    const seedFromEdit = useMemo(() => {
        if (!editData) {
            return null;
        }
        let seedSeries = editData.series;
        let seedCustomSeries = "";
        if (!PINNED_SERIES.includes(editData.series)) {
            seedSeries = OTHER_VALUE;
            seedCustomSeries = editData.series;
        }
        let seedLanguage = editData.language;
        let seedCustomLanguage = "";
        if (availableLanguages.length > 0 && !availableLanguages.includes(editData.language)) {
            seedLanguage = OTHER_VALUE;
            seedCustomLanguage = editData.language;
        }
        return {
            title: editData.title,
            summary: editData.summary,
            rating: editData.rating,
            isOneshot: editData.is_oneshot,
            containsLemons: editData.contains_lemons,
            isPairing: editData.is_pairing,
            status: editData.status,
            characters:
                editData.characters?.map((c, i) => ({
                    series: c.series,
                    character_id: c.character_id,
                    character_name: c.character_name,
                    sort_order: i,
                })) ?? [],
            genreA: editData.genres?.[0] ?? "",
            genreB: editData.genres?.[1] ?? "",
            tags: editData.tags ?? [],
            series: seedSeries,
            customSeries: seedCustomSeries,
            language: seedLanguage,
            customLanguage: seedCustomLanguage,
            coverPreview: editData.cover_image_url ?? "",
            chapterCount: editData.chapter_count ?? 0,
        };
    }, [editData, availableLanguages]);

    useEffect(() => {
        if (editData && editData.author.id !== user?.id && !can(user?.role, "edit_any_theory")) {
            navigate(`/fanfiction/${editId}`);
        }
    }, [editData, user?.id, user?.role, navigate, editId]);

    const [step, setStep] = useState(1);
    const [titleDraft, setTitleDraft] = useState<string | null>(null);
    const [summaryDraft, setSummaryDraft] = useState<string | null>(null);
    const [ratingDraft, setRatingDraft] = useState<string | null>(null);
    const [isOneshotDraft, setIsOneshotDraft] = useState<boolean | null>(null);
    const [containsLemonsDraft, setContainsLemonsDraft] = useState<boolean | null>(null);
    const [isPairingDraft, setIsPairingDraft] = useState<boolean | null>(null);
    const [statusDraft, setStatusDraft] = useState<string | null>(null);
    const [charactersDraft, setCharactersDraft] = useState<ShipCharacter[] | null>(null);
    const [genreADraft, setGenreADraft] = useState<string | null>(null);
    const [genreBDraft, setGenreBDraft] = useState<string | null>(null);
    const [tagsDraft, setTagsDraft] = useState<string[] | null>(null);
    const [seriesDraft, setSeriesDraft] = useState<string | null>(null);
    const [customSeriesDraft, setCustomSeriesDraft] = useState<string | null>(null);
    const [languageDraft, setLanguageDraft] = useState<string | null>(null);
    const [customLanguageDraft, setCustomLanguageDraft] = useState<string | null>(null);
    const [coverPreviewDraft, setCoverPreviewDraft] = useState<string | null>(null);
    const [tagInput, setTagInput] = useState("");
    const [body, setBody] = useState("");
    const [coverFile, setCoverFile] = useState<File | null>(null);
    const [coverRemoved, setCoverRemoved] = useState(false);
    const [editChapterId, setEditChapterId] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    const title = titleDraft ?? seedFromEdit?.title ?? "";
    const summary = summaryDraft ?? seedFromEdit?.summary ?? "";
    const rating = ratingDraft ?? seedFromEdit?.rating ?? "K";
    const isOneshot = isOneshotDraft ?? seedFromEdit?.isOneshot ?? true;
    const containsLemons = containsLemonsDraft ?? seedFromEdit?.containsLemons ?? false;
    const isPairing = isPairingDraft ?? seedFromEdit?.isPairing ?? false;
    const status = statusDraft ?? seedFromEdit?.status ?? "in_progress";
    const characters = useMemo(
        () => charactersDraft ?? seedFromEdit?.characters ?? [],
        [charactersDraft, seedFromEdit?.characters],
    );
    const genreA = genreADraft ?? seedFromEdit?.genreA ?? "";
    const genreB = genreBDraft ?? seedFromEdit?.genreB ?? "";
    const tags = useMemo(() => tagsDraft ?? seedFromEdit?.tags ?? [], [tagsDraft, seedFromEdit?.tags]);
    const series = seriesDraft ?? seedFromEdit?.series ?? "Umineko";
    const customSeries = customSeriesDraft ?? seedFromEdit?.customSeries ?? "";
    const language = languageDraft ?? seedFromEdit?.language ?? "English";
    const customLanguage = customLanguageDraft ?? seedFromEdit?.customLanguage ?? "";
    const coverPreview = coverPreviewDraft ?? seedFromEdit?.coverPreview ?? "";
    const editChapterCount = seedFromEdit?.chapterCount ?? 0;

    function setTitle(v: string) {
        setTitleDraft(v);
    }
    function setSummary(v: string) {
        setSummaryDraft(v);
    }
    function setRating(v: string) {
        setRatingDraft(v);
    }
    function setIsOneshot(v: boolean) {
        setIsOneshotDraft(v);
    }
    function setContainsLemons(v: boolean) {
        setContainsLemonsDraft(v);
    }
    function setIsPairing(v: boolean) {
        setIsPairingDraft(v);
    }
    function setStatus(v: string) {
        setStatusDraft(v);
    }
    function setCharacters(updater: ShipCharacter[] | ((prev: ShipCharacter[]) => ShipCharacter[])) {
        if (typeof updater === "function") {
            setCharactersDraft(prev => updater(prev ?? seedFromEdit?.characters ?? []));
        } else {
            setCharactersDraft(updater);
        }
    }
    function setGenreA(v: string) {
        setGenreADraft(v);
    }
    function setGenreB(v: string) {
        setGenreBDraft(v);
    }
    function setTags(updater: string[] | ((prev: string[]) => string[])) {
        if (typeof updater === "function") {
            setTagsDraft(prev => updater(prev ?? seedFromEdit?.tags ?? []));
        } else {
            setTagsDraft(updater);
        }
    }
    function setSeries(v: string) {
        setSeriesDraft(v);
    }
    function setCustomSeries(v: string) {
        setCustomSeriesDraft(v);
    }
    function setLanguage(v: string) {
        setLanguageDraft(v);
    }
    function setCustomLanguage(v: string) {
        setCustomLanguageDraft(v);
    }
    function setCoverPreview(v: string) {
        setCoverPreviewDraft(v);
    }

    const showCustomSeries = series === OTHER_VALUE;
    const showCustomLanguage = language === OTHER_VALUE;

    function restoreDraft(draft: DraftData) {
        setTitle(draft.title);
        setSummary(draft.summary);
        setSeries(draft.series);
        setCustomSeries(draft.customSeries);
        setRating(draft.rating);
        setLanguage(draft.language);
        setCustomLanguage(draft.customLanguage);
        setGenreA(draft.genreA);
        setGenreB(draft.genreB);
        setTags(draft.tags ?? []);
        setStatus(draft.status ?? "in_progress");
        setCharacters(draft.characters);
        setIsPairing(draft.isPairing);
        setIsOneshot(draft.isOneshot);
        setContainsLemons(draft.containsLemons);
        setBody(draft.body);
        setStep(draft.step);
        setDraftPrompt(null);
        setInitialised(true);
    }

    function startFresh() {
        clearDraft();
        setDraftPrompt(null);
        setInitialised(true);
    }

    const saveDraft = useCallback(() => {
        const data: DraftData = {
            title,
            summary,
            series,
            customSeries,
            rating,
            language,
            customLanguage,
            genreA,
            genreB,
            tags,
            status,
            characters,
            isPairing,
            isOneshot,
            containsLemons,
            body,
            step,
        };
        localStorage.setItem(DRAFT_KEY, JSON.stringify(data));
    }, [
        title,
        summary,
        series,
        customSeries,
        rating,
        language,
        customLanguage,
        genreA,
        genreB,
        tags,
        status,
        characters,
        isPairing,
        isOneshot,
        containsLemons,
        body,
        step,
    ]);

    useEffect(() => {
        if (!initialised || isEdit) {
            return;
        }
        saveDraft();
    }, [saveDraft, initialised, isEdit]);

    const allSeries = [...PINNED_SERIES, ...dynamicSeries.filter(s => !PINNED_SERIES.includes(s))];

    function addCharacter(character: ShipCharacter) {
        setCharacters(prev => [...prev, { ...character, sort_order: prev.length }]);
    }

    function removeCharacter(index: number) {
        setCharacters(prev => prev.filter((_, i) => i !== index).map((c, i) => ({ ...c, sort_order: i })));
    }

    function handleCoverChange(e: React.ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (!file) {
            return;
        }
        setCoverFile(file);
        setCoverPreview(URL.createObjectURL(file));
        setCoverRemoved(false);
    }

    function removeCover() {
        setCoverFile(null);
        setCoverPreview("");
        setCoverRemoved(true);
        if (fileInputRef.current) {
            fileInputRef.current.value = "";
        }
    }

    function handleNextStep() {
        setError("");
        if (!title.trim()) {
            setError("Title is required");
            return;
        }
        const resolvedSeries = showCustomSeries ? customSeries.trim() : series;
        if (!resolvedSeries) {
            setError("Series is required");
            return;
        }
        const resolvedLanguage = showCustomLanguage ? customLanguage.trim() : language;
        if (!resolvedLanguage) {
            setError("Language is required");
            return;
        }
        setStep(2);
    }

    async function handleNextStepEdit() {
        setError("");
        if (!title.trim()) {
            setError("Title is required");
            return;
        }
        const resolvedSeries = showCustomSeries ? customSeries.trim() : series;
        const resolvedLanguage = showCustomLanguage ? customLanguage.trim() : language;
        if (!resolvedSeries || !resolvedLanguage) {
            setError("Series and language are required");
            return;
        }

        const genres: string[] = [];
        if (genreA) {
            genres.push(genreA);
        }
        if (genreB && genreB !== genreA) {
            genres.push(genreB);
        }

        setSubmitting(true);
        try {
            await updateMutation.mutateAsync({
                title: title.trim(),
                summary: summary.trim(),
                series: resolvedSeries,
                rating,
                language: resolvedLanguage,
                status,
                is_oneshot: isOneshot,
                contains_lemons: containsLemons,
                genres,
                tags,
                characters,
                is_pairing: isPairing,
            });

            if (coverFile && editId) {
                try {
                    await uploadCoverMutation.mutateAsync(coverFile);
                } catch {}
            } else if (coverRemoved && editId) {
                try {
                    await deleteCoverMutation.mutateAsync();
                } catch {}
            }

            if (isOneshot && editId) {
                const fanficData = await qc.fetchQuery(fanficQueryFns.fanfic(editId));
                if (fanficData.chapters?.length > 0) {
                    const ch = await qc.fetchQuery(fanficQueryFns.chapter(editId, 1));
                    setBody(ch.body);
                    setEditChapterId(ch.id);
                }
            }
            setStep(2);
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to save");
        } finally {
            setSubmitting(false);
        }
    }

    async function handleSubmit(asDraft: boolean) {
        setError("");

        const resolvedSeries = showCustomSeries ? customSeries.trim() : series;
        const resolvedLanguage = showCustomLanguage ? customLanguage.trim() : language;

        const genres: string[] = [];
        if (genreA) {
            genres.push(genreA);
        }
        if (genreB && genreB !== genreA) {
            genres.push(genreB);
        }

        setSubmitting(true);
        try {
            if (isEdit && editId) {
                await updateMutation.mutateAsync({
                    title: title.trim(),
                    summary: summary.trim(),
                    series: resolvedSeries,
                    rating,
                    language: resolvedLanguage,
                    status,
                    is_oneshot: isOneshot,
                    contains_lemons: containsLemons,
                    genres,
                    tags,
                    characters,
                    is_pairing: isPairing,
                });
                if (coverFile) {
                    try {
                        await uploadCoverMutation.mutateAsync(coverFile);
                    } catch {}
                } else if (coverRemoved) {
                    try {
                        await deleteCoverMutation.mutateAsync();
                    } catch {}
                }
                navigate(`/fanfiction/${editId}`);
                return;
            }

            const result = await createMutation.mutateAsync({
                title: title.trim(),
                summary: summary.trim(),
                series: resolvedSeries,
                rating,
                language: resolvedLanguage,
                status: asDraft ? "draft" : status,
                is_oneshot: isOneshot,
                contains_lemons: containsLemons,
                genres,
                tags,
                characters,
                is_pairing: isPairing,
                body: body || undefined,
            });

            if (coverFile) {
                try {
                    await uploadCoverForMutation.mutateAsync({ id: result.id, file: coverFile });
                } catch {}
            }

            clearDraft();
            navigate(`/fanfiction/${result.id}`);
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to create fanfic");
        } finally {
            setSubmitting(false);
        }
    }

    function handleCancel() {
        if (title.trim() || body.trim()) {
            if (!window.confirm("You have unsaved work. Discard your draft?")) {
                return;
            }
        }
        clearDraft();
        navigate("/fanfiction");
    }

    if (editLoading) {
        return <div className="loading">Loading...</div>;
    }

    if (isEdit && !title) {
        return <div className="empty-state">Fanfic not found.</div>;
    }

    if (draftPrompt) {
        return (
            <div className={styles.formPage}>
                <h1 className={styles.formHeading}>Unfinished Draft</h1>
                <p style={{ color: "var(--text)", marginBottom: "1rem" }}>
                    You have an unfinished draft: <strong>{draftPrompt.title}</strong>
                </p>
                <div className={styles.formActions}>
                    <Button variant="ghost" onClick={startFresh}>
                        Start Fresh
                    </Button>
                    <Button variant="primary" onClick={() => restoreDraft(draftPrompt)}>
                        Continue Draft
                    </Button>
                </div>
            </div>
        );
    }

    if (!initialised) {
        return null;
    }

    if (step === 1) {
        return (
            <div className={styles.formPage}>
                <span className={styles.back} onClick={isEdit ? () => navigate(`/fanfiction/${editId}`) : handleCancel}>
                    &larr; {isEdit ? "Back to Fanfic" : "All Fanfiction"}
                </span>
                <h1 className={styles.formHeading}>{isEdit ? "Edit Fanfic" : "New Fanfic"}</h1>

                <div className={styles.formRow}>
                    <label className={styles.formLabel}>Title</label>
                    <Input
                        type="text"
                        value={title}
                        onChange={e => setTitle(e.target.value)}
                        placeholder="Your fanfic title..."
                        fullWidth
                    />
                </div>

                <div className={styles.formRow}>
                    <label className={styles.formLabel}>Summary</label>
                    <MentionTextArea
                        value={summary}
                        onChange={setSummary}
                        placeholder="Brief summary of your story..."
                        rows={3}
                    />
                </div>

                <div className={styles.formRowDouble}>
                    <div>
                        <label className={styles.formLabel}>Series</label>
                        <Select
                            value={showCustomSeries ? OTHER_VALUE : series}
                            onChange={e => {
                                const v = e.target.value;
                                if (v === OTHER_VALUE) {
                                    setSeries(OTHER_VALUE);
                                } else {
                                    setSeries(v);
                                    setCustomSeries("");
                                }
                            }}
                        >
                            {allSeries.map(s => (
                                <option key={s} value={s}>
                                    {s}
                                </option>
                            ))}
                            <option value={OTHER_VALUE}>Other...</option>
                        </Select>
                        {showCustomSeries && (
                            <Input
                                type="text"
                                value={customSeries}
                                onChange={e => setCustomSeries(e.target.value)}
                                placeholder="Enter series name..."
                                fullWidth
                                style={{ marginTop: "0.5rem" }}
                            />
                        )}
                    </div>
                    <div>
                        <label className={styles.formLabel}>Rating</label>
                        <Select value={rating} onChange={e => setRating(e.target.value)}>
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
                            value={showCustomLanguage ? OTHER_VALUE : language}
                            onChange={e => {
                                const v = e.target.value;
                                if (v === OTHER_VALUE) {
                                    setLanguage(OTHER_VALUE);
                                } else {
                                    setLanguage(v);
                                    setCustomLanguage("");
                                }
                            }}
                        >
                            {availableLanguages.map(l => (
                                <option key={l} value={l}>
                                    {l}
                                </option>
                            ))}
                            {!availableLanguages.includes("English") && <option value="English">English</option>}
                            <option value={OTHER_VALUE}>Other...</option>
                        </Select>
                        {showCustomLanguage && (
                            <Input
                                type="text"
                                value={customLanguage}
                                onChange={e => setCustomLanguage(e.target.value)}
                                placeholder="Enter language..."
                                fullWidth
                                style={{ marginTop: "0.5rem" }}
                            />
                        )}
                    </div>
                    <div>
                        <label className={styles.formLabel}>Genre A</label>
                        <Select value={genreA} onChange={e => setGenreA(e.target.value)}>
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
                        <Select value={genreB} onChange={e => setGenreB(e.target.value)}>
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
                    <CharacterPicker onAdd={addCharacter} existing={characters} />
                    {characters.length > 0 && (
                        <div className={styles.charList}>
                            {characters.map((c, i) => (
                                <span
                                    key={`${c.series}-${c.character_id ?? c.character_name}-${i}`}
                                    className={styles.charPill}
                                >
                                    {c.character_name}
                                    <button
                                        type="button"
                                        className={styles.charPillRemove}
                                        onClick={() => removeCharacter(i)}
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
                    enabled={isPairing}
                    onChange={setIsPairing}
                    label="Pairing / ship fic"
                    description="These characters are in a relationship"
                />

                <ToggleSwitch
                    enabled={isOneshot}
                    onChange={isEdit && editChapterCount > 1 ? () => {} : setIsOneshot}
                    label="One-shot"
                    description={
                        isEdit && editChapterCount > 1
                            ? `Cannot switch to one-shot with ${editChapterCount} chapters. Delete extra chapters first.`
                            : "Single chapter story. Turn off to add chapters after creation."
                    }
                />

                <ToggleSwitch
                    enabled={containsLemons}
                    onChange={setContainsLemons}
                    label="Contains lemons"
                    description="This story contains explicit content. It will be hidden by default."
                />

                <div className={styles.formRow}>
                    <label className={styles.formLabel}>Status</label>
                    <Select value={status} onChange={e => setStatus(e.target.value)}>
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
                            value={tagInput}
                            onChange={e => setTagInput(e.target.value)}
                            onKeyDown={e => {
                                if (e.key === "Enter" || e.key === ",") {
                                    e.preventDefault();
                                    const val = tagInput.trim().slice(0, 30);
                                    if (
                                        val &&
                                        tags.length < 10 &&
                                        !tags.some(t => t.toLowerCase() === val.toLowerCase())
                                    ) {
                                        setTags(prev => [...prev, val]);
                                    }
                                    setTagInput("");
                                }
                            }}
                            placeholder="Type a tag and press Enter..."
                            fullWidth
                            maxLength={30}
                        />
                    </div>
                    {tags.length > 0 && (
                        <div className={styles.charList} style={{ marginTop: "0.5rem" }}>
                            {tags.map((t, i) => (
                                <span key={`${t}-${i}`} className={styles.tagPill}>
                                    {t}
                                    <button
                                        type="button"
                                        className={styles.charPillRemove}
                                        onClick={() => setTags(prev => prev.filter((_, j) => j !== i))}
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
                    <input ref={fileInputRef} type="file" accept="image/*" onChange={handleCoverChange} hidden />
                    <Button variant="ghost" size="small" onClick={() => fileInputRef.current?.click()}>
                        + Cover Image
                    </Button>
                    {coverPreview && (
                        <div style={{ marginTop: "0.5rem" }}>
                            <img
                                src={coverPreview}
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
                    <Button variant="ghost" onClick={handleCancel}>
                        Cancel
                    </Button>
                    {isEdit && !isOneshot ? (
                        <Button
                            variant="primary"
                            onClick={() => handleSubmit(false)}
                            disabled={submitting || !title.trim()}
                        >
                            {submitting ? "Saving..." : "Save Changes"}
                        </Button>
                    ) : (
                        <Button variant="primary" onClick={isEdit ? handleNextStepEdit : handleNextStep}>
                            Next: {isOneshot ? "Edit Story" : "Write Story"}
                        </Button>
                    )}
                </div>
            </div>
        );
    }

    return (
        <div className={styles.formPage}>
            <span className={styles.back} onClick={() => setStep(1)}>
                &larr; Back to Details
            </span>
            <h1 className={styles.formHeading}>
                {isEdit ? "Edit Story" : isOneshot ? "Write Your Story" : "Write First Chapter"}
            </h1>

            <RichTextEditor
                content={body}
                onChange={setBody}
                placeholder={isOneshot ? "Write your story here..." : "Write your first chapter here..."}
            />

            {error && <ErrorBanner message={error} />}

            <div className={styles.formActions} style={{ marginTop: "1rem" }}>
                <Button variant="ghost" onClick={() => setStep(1)}>
                    Back
                </Button>
                {isEdit ? (
                    <Button
                        variant="primary"
                        onClick={async () => {
                            if (!body.trim()) {
                                return;
                            }
                            setSubmitting(true);
                            try {
                                if (editChapterId) {
                                    await updateChapterMutation.mutateAsync({
                                        chapterId: editChapterId,
                                        title: "",
                                        body,
                                    });
                                } else {
                                    await createChapterMutation.mutateAsync({ title: "", body });
                                }
                                navigate(`/fanfiction/${editId}`);
                            } catch (e) {
                                setError(e instanceof Error ? e.message : "Failed to save");
                            } finally {
                                setSubmitting(false);
                            }
                        }}
                        disabled={submitting || !body.trim()}
                    >
                        {submitting ? "Saving..." : "Save Changes"}
                    </Button>
                ) : (
                    <>
                        <Button variant="secondary" onClick={() => handleSubmit(true)} disabled={submitting}>
                            {submitting ? "Saving..." : "Save as Draft"}
                        </Button>
                        <Button variant="primary" onClick={() => handleSubmit(false)} disabled={submitting}>
                            {submitting ? "Publishing..." : "Publish"}
                        </Button>
                    </>
                )}
            </div>
        </div>
    );
}
