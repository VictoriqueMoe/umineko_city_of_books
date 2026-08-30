import { type ChangeEvent, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router";
import type { ShipCharacter } from "../types/api";
import { contentPermissions } from "../domain/contentPermissions";
import {
    clearDraft,
    draftFromFields,
    draftToFields,
    resumableDraft,
    saveDraft,
    type DraftData,
} from "../domain/fanfic/draft";
import {
    fieldDefaults,
    mergeDefined,
    OTHER_VALUE,
    seedFromFanfic,
    seriesOptions,
    toPayload,
    validateDetails,
    withCharacter,
    withoutCharacter,
    withoutTag,
    withTag,
    type EditorFields,
} from "../domain/fanfic/form";
import { errorMessage } from "../utils/errorMessage";
import { useAuth } from "./useAuth";
import { useFanfic, useFanficChapter, useFanficLanguages, useFanficSeries } from "./queries/fanfic";
import {
    useCreateFanfic,
    useCreateFanficChapter,
    useDeleteFanficCover,
    useUpdateFanfic,
    useUpdateFanficChapter,
    useUploadFanficCover,
    useUploadFanficCoverFor,
} from "./mutations/fanfic";

const COVER_UPLOAD_FAILED = "The fanfic was saved but its cover image was not";
const COVER_REMOVE_FAILED = "The fanfic was saved but its cover image was not removed";

function coverProblem(headline: string, thrown: unknown): string {
    const detail = errorMessage(thrown, "");

    return detail ? `${headline}: ${detail}` : headline;
}

export interface FanficDetailsStep {
    fields: EditorFields;
    series: string[];
    languages: string[];
    chapterCount: number;
    oneshotLocked: boolean;
    tagInput: string;
    setField: <K extends keyof EditorFields>(key: K, value: EditorFields[K]) => void;
    chooseSeries: (value: string) => void;
    chooseLanguage: (value: string) => void;
    setTagInput: (value: string) => void;
    addTag: () => void;
    removeTag: (index: number) => void;
    addCharacter: (character: ShipCharacter) => void;
    removeCharacter: (index: number) => void;
    chooseCover: (e: ChangeEvent<HTMLInputElement>) => void;
    removeCover: () => void;
    next: () => void;
    save: () => void;
}

export interface FanficStoryStep {
    body: string;
    setBody: (value: string) => void;
    back: () => void;
    saveChapter: () => void;
    publish: () => void;
    saveAsDraft: () => void;
}

export interface FanficEditor {
    isEdit: boolean;
    editId: string;
    loading: boolean;
    notFound: boolean;
    hidden: boolean;
    step: number;
    error: string;
    submitting: boolean;
    draftPrompt: DraftData | null;
    resumeDraft: () => void;
    startFresh: () => void;
    back: () => void;
    cancel: () => void;
    details: FanficDetailsStep;
    story: FanficStoryStep;
}

export function useFanficEditor(): FanficEditor {
    const { id } = useParams<{ id: string }>();
    const editId = id ?? "";
    const isEdit = !!id;
    const navigate = useNavigate();
    const { user } = useAuth();

    const [draftPrompt, setDraftPrompt] = useState<DraftData | null>(() => (isEdit ? null : resumableDraft()));
    const [initialised, setInitialised] = useState(() => (isEdit ? false : !resumableDraft()));

    const { series: dynamicSeries } = useFanficSeries();
    const { languages } = useFanficLanguages();
    const { fanfic: editData, loading: editLoading } = useFanfic(isEdit ? editId : "");

    const updateMutation = useUpdateFanfic(editId);
    const createMutation = useCreateFanfic();
    const uploadCoverMutation = useUploadFanficCover(editId);
    const uploadCoverForMutation = useUploadFanficCoverFor();
    const deleteCoverMutation = useDeleteFanficCover(editId);
    const updateChapterMutation = useUpdateFanficChapter(editId);
    const createChapterMutation = useCreateFanficChapter(editId);

    const [step, setStep] = useState(1);
    const [draft, setDraft] = useState<Partial<EditorFields>>({});
    const [tagInput, setTagInput] = useState("");
    const [bodyDraft, setBodyDraft] = useState<string | null>(null);
    const [coverFile, setCoverFile] = useState<File | null>(null);
    const [coverRemoved, setCoverRemoved] = useState(false);
    const [createdId, setCreatedId] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    const seed = useMemo(() => seedFromFanfic(editData, languages), [editData, languages]);
    const fields = useMemo(() => mergeDefined(fieldDefaults(), seed, draft), [seed, draft]);
    const chapterCount = seed?.chapterCount ?? 0;
    const hasFirstChapter = (editData?.chapters?.length ?? 0) > 0;

    const { chapter } = useFanficChapter(editId, isEdit && fields.isOneshot && hasFirstChapter ? 1 : 0);
    const body = bodyDraft ?? chapter?.body ?? "";
    const editChapterId = chapter?.id ?? "";

    const canEdit = contentPermissions(user, { family: "fanfic", authorId: editData?.author.id }).canEdit;

    useEffect(() => {
        if (editData && !canEdit) {
            navigate(`/fanfiction/${editId}`);
        }
    }, [editData, canEdit, navigate, editId]);

    useEffect(() => {
        if (!initialised || isEdit) {
            return;
        }
        saveDraft(draftFromFields(fields, body, step));
    }, [fields, body, step, initialised, isEdit]);

    function setField<K extends keyof EditorFields>(key: K, value: EditorFields[K]) {
        setDraft(prev => ({ ...prev, [key]: value }));
    }

    function updateField<K extends keyof EditorFields>(key: K, updater: (prev: EditorFields[K]) => EditorFields[K]) {
        setDraft(prev => {
            const current = prev[key] ?? seed?.[key] ?? fieldDefaults()[key];
            return { ...prev, [key]: updater(current) };
        });
    }

    function chooseSeries(value: string) {
        if (value === OTHER_VALUE) {
            setField("series", OTHER_VALUE);
            return;
        }

        setDraft(prev => ({ ...prev, series: value, customSeries: "" }));
    }

    function chooseLanguage(value: string) {
        if (value === OTHER_VALUE) {
            setField("language", OTHER_VALUE);
            return;
        }

        setDraft(prev => ({ ...prev, language: value, customLanguage: "" }));
    }

    function addTag() {
        updateField("tags", prev => withTag(prev, tagInput));
        setTagInput("");
    }

    function removeTag(index: number) {
        updateField("tags", prev => withoutTag(prev, index));
    }

    function addCharacter(character: ShipCharacter) {
        updateField("characters", prev => withCharacter(prev, character));
    }

    function removeCharacter(index: number) {
        updateField("characters", prev => withoutCharacter(prev, index));
    }

    function chooseCover(e: ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (!file) {
            return;
        }

        setCoverFile(file);
        setField("coverPreview", URL.createObjectURL(file));
        setCoverRemoved(false);
    }

    function removeCover() {
        setCoverFile(null);
        setField("coverPreview", "");
        setCoverRemoved(true);
    }

    function resumeDraft() {
        if (!draftPrompt) {
            return;
        }

        setDraft(draftToFields(draftPrompt));
        setBodyDraft(draftPrompt.body);
        setStep(draftPrompt.step);
        setDraftPrompt(null);
        setInitialised(true);
    }

    function startFresh() {
        clearDraft();
        setDraftPrompt(null);
        setInitialised(true);
    }

    async function applyCover(newFanficId: string): Promise<boolean> {
        if (coverFile) {
            try {
                if (newFanficId) {
                    await uploadCoverForMutation.mutateAsync({ id: newFanficId, file: coverFile });
                } else {
                    await uploadCoverMutation.mutateAsync(coverFile);
                }
            } catch (e) {
                setError(coverProblem(COVER_UPLOAD_FAILED, e));
                return false;
            }

            return true;
        }

        if (!newFanficId && coverRemoved) {
            try {
                await deleteCoverMutation.mutateAsync();
            } catch (e) {
                setError(coverProblem(COVER_REMOVE_FAILED, e));
                return false;
            }
        }

        return true;
    }

    async function next() {
        setError("");

        const problem = validateDetails(fields);
        if (problem) {
            setError(problem);
            return;
        }

        if (!isEdit) {
            setStep(2);
            return;
        }

        setSubmitting(true);
        try {
            await updateMutation.mutateAsync(toPayload(fields));

            if (!(await applyCover(""))) {
                return;
            }

            setStep(2);
        } catch (e) {
            setError(errorMessage(e, "Failed to save"));
        } finally {
            setSubmitting(false);
        }
    }

    async function submit(asDraft: boolean) {
        setError("");
        setSubmitting(true);

        try {
            if (isEdit && editId) {
                await updateMutation.mutateAsync(toPayload(fields));

                if (!(await applyCover(""))) {
                    return;
                }

                navigate(`/fanfiction/${editId}`);
                return;
            }

            const newId =
                createdId ||
                (
                    await createMutation.mutateAsync({
                        ...toPayload(fields),
                        status: asDraft ? "draft" : fields.status,
                        body: body || undefined,
                    })
                ).id;
            setCreatedId(newId);
            clearDraft();

            if (!(await applyCover(newId))) {
                return;
            }

            navigate(`/fanfiction/${newId}`);
        } catch (e) {
            setError(errorMessage(e, "Failed to create fanfic"));
        } finally {
            setSubmitting(false);
        }
    }

    async function saveChapter() {
        if (!body.trim()) {
            return;
        }

        setSubmitting(true);
        try {
            if (editChapterId) {
                await updateChapterMutation.mutateAsync({ chapterId: editChapterId, title: "", body });
            } else {
                await createChapterMutation.mutateAsync({ title: "", body });
            }

            navigate(`/fanfiction/${editId}`);
        } catch (e) {
            setError(errorMessage(e, "Failed to save"));
        } finally {
            setSubmitting(false);
        }
    }

    function cancel() {
        if (fields.title.trim() || (bodyDraft ?? "").trim()) {
            if (!window.confirm("You have unsaved work. Discard your draft?")) {
                return;
            }
        }

        clearDraft();
        navigate("/fanfiction");
    }

    function back() {
        if (isEdit) {
            navigate(`/fanfiction/${editId}`);
            return;
        }

        cancel();
    }

    return {
        isEdit,
        editId,
        loading: isEdit && editLoading,
        notFound: isEdit && !editLoading && !editData,
        hidden: !isEdit && !initialised,
        step,
        error,
        submitting,
        draftPrompt,
        resumeDraft,
        startFresh,
        back,
        cancel,
        details: {
            fields,
            series: seriesOptions(dynamicSeries),
            languages,
            chapterCount,
            oneshotLocked: isEdit && chapterCount > 1,
            tagInput,
            setField,
            chooseSeries,
            chooseLanguage,
            setTagInput,
            addTag,
            removeTag,
            addCharacter,
            removeCharacter,
            chooseCover,
            removeCover,
            next,
            save: () => submit(false),
        },
        story: {
            body,
            setBody: setBodyDraft,
            back: () => setStep(1),
            saveChapter,
            publish: () => submit(false),
            saveAsDraft: () => submit(true),
        },
    };
}
