import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { createMystery, getMystery, updateMystery, uploadMysteryAttachment } from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { TextArea } from "../../components/TextArea/TextArea";
import { Select } from "../../components/Select/Select";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import styles from "./MysteryPages.module.css";

interface ClueInput {
    body: string;
    truth_type: string;
}

export function CreateMysteryPage() {
    const { id: editId } = useParams<{ id: string }>();
    const isEdit = !!editId;
    usePageTitle(isEdit ? "Edit Mystery" : "New Mystery");
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const [title, setTitle] = useState("");
    const [body, setBody] = useState("");
    const [difficulty, setDifficulty] = useState("medium");
    const [freeForAll, setFreeForAll] = useState(false);
    const [clues, setClues] = useState<ClueInput[]>([{ body: "", truth_type: "red" }]);
    const [attachments, setAttachments] = useState<File[]>([]);
    const attachmentInputRef = useRef<HTMLInputElement>(null);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const [editLoading, setEditLoading] = useState(isEdit);

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    useEffect(() => {
        if (!isEdit || !editId) {
            return;
        }
        getMystery(editId)
            .then(data => {
                setTitle(data.title);
                setBody(data.body);
                setDifficulty(data.difficulty);
                setFreeForAll(data.free_for_all);
                const gmClues = (data.clues ?? []).filter(c => !c.player_id);
                if (gmClues.length > 0) {
                    setClues(gmClues.map(c => ({ body: c.body, truth_type: c.truth_type })));
                }
                setEditLoading(false);
            })
            .catch(() => {
                setEditLoading(false);
            });
    }, [isEdit, editId]);

    function addClue() {
        setClues(prev => [...prev, { body: "", truth_type: "red" }]);
    }

    function removeClue(index: number) {
        setClues(prev => prev.filter((_, i) => i !== index));
    }

    function updateClue(index: number, field: keyof ClueInput, value: string) {
        setClues(prev => {
            const updated = [...prev];
            updated[index] = { ...updated[index], [field]: value };
            return updated;
        });
    }

    async function handleSubmit(e: React.FormEvent) {
        e.preventDefault();
        if (!title.trim() || !body.trim() || submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            const validClues = clues.filter(c => c.body.trim());
            if (isEdit && editId) {
                await updateMystery(editId, {
                    title: title.trim(),
                    body: body.trim(),
                    difficulty,
                    free_for_all: freeForAll,
                    clues: validClues,
                });
                for (const file of attachments) {
                    try {
                        await uploadMysteryAttachment(editId, file);
                    } catch {}
                }
                navigate(`/mystery/${editId}`);
            } else {
                const result = await createMystery({
                    title: title.trim(),
                    body: body.trim(),
                    difficulty,
                    free_for_all: freeForAll,
                    clues: validClues,
                });
                for (const file of attachments) {
                    try {
                        await uploadMysteryAttachment(result.id, file);
                    } catch {}
                }
                navigate(`/mystery/${result.id}`);
            }
        } catch (err) {
            setError(
                err instanceof Error ? err.message : isEdit ? "Failed to update mystery" : "Failed to create mystery",
            );
        } finally {
            setSubmitting(false);
        }
    }

    if (authLoading || !user) {
        return null;
    }

    if (editLoading) {
        return <div className="loading">Loading mystery...</div>;
    }

    return (
        <div className={styles.formPage}>
            <span className={styles.back} onClick={() => navigate(isEdit ? `/mystery/${editId}` : "/mysteries")}>
                &larr; {isEdit ? "Back to Mystery" : "All Mysteries"}
            </span>
            <h2 className={styles.formHeading}>{isEdit ? "Edit Mystery" : "Create a Mystery"}</h2>

            {!isEdit && (
                <InfoPanel title="Game Master's Guide">
                    <p>
                        As the Game Master, you control the board. Write a mystery scenario that is solvable from the
                        information you provide, the pieces should have everything they need to reach the truth.
                    </p>
                    <p>
                        Set your <strong>red truths</strong>, these are absolute facts that cannot be denied. Use them
                        to define the boundaries of your mystery. You can also use <strong>purple truths</strong> for
                        statements from characters that may or may not be reliable.
                    </p>
                    <p>
                        Once pieces begin submitting their blue truths, you can reply to dismantle incorrect theories,
                        add additional red truths if needed, and ultimately select a winner when someone solves it.
                    </p>
                </InfoPanel>
            )}

            {error && <ErrorBanner message={error} />}

            <form onSubmit={handleSubmit}>
                <Input
                    type="text"
                    fullWidth
                    placeholder="Mystery title..."
                    value={title}
                    onChange={e => setTitle(e.target.value)}
                    maxLength={200}
                />

                <TextArea
                    placeholder="Write your mystery scenario... Set the scene, introduce the characters, present the puzzle."
                    value={body}
                    onChange={e => setBody(e.target.value)}
                />

                <Select value={difficulty} onChange={e => setDifficulty(e.target.value)}>
                    <option value="easy">Easy</option>
                    <option value="medium">Medium</option>
                    <option value="hard">Hard</option>
                    <option value="nightmare">Nightmare</option>
                </Select>

                <div style={{ marginTop: "1rem" }}>
                    <ToggleSwitch
                        enabled={freeForAll}
                        onChange={setFreeForAll}
                        label="Free-for-all mode"
                        description="All players can see each other's attempts. By default, each player only sees their own thread with the Game Master."
                    />
                </div>

                <h3 className={styles.cluesTitle} style={{ marginTop: "1.5rem" }}>
                    Red Truths (Clues)
                </h3>
                <p style={{ color: "var(--text-muted)", fontSize: "0.85rem", marginBottom: "0.75rem" }}>
                    These are the absolute truths of your mystery. They cannot be denied.
                </p>

                <div className={styles.clueInputs}>
                    {clues.map((clue, i) => (
                        <div key={i} className={styles.clueRow}>
                            <Input
                                type="text"
                                fullWidth
                                placeholder={`Red truth #${i + 1}...`}
                                value={clue.body}
                                onChange={e => updateClue(i, "body", e.target.value)}
                                className={styles.clueInput}
                            />
                            <Select
                                value={clue.truth_type}
                                onChange={e => updateClue(i, "truth_type", e.target.value)}
                                style={{ width: "auto" }}
                            >
                                <option value="red">Red</option>
                                <option value="purple">Purple</option>
                            </Select>
                            {clues.length > 1 && (
                                <Button variant="ghost" size="small" onClick={() => removeClue(i)}>
                                    {"\u2715"}
                                </Button>
                            )}
                        </div>
                    ))}
                </div>
                <Button variant="ghost" type="button" onClick={addClue}>
                    + Add Clue
                </Button>

                <div className={styles.attachments} style={{ marginTop: "1.5rem" }}>
                    <h3 className={styles.attachmentsTitle}>Attachments (optional)</h3>
                    <p style={{ color: "var(--text-muted)", fontSize: "0.85rem", marginBottom: "0.75rem" }}>
                        Upload PDF, TXT, or DOCX files as evidence or supplementary material.
                    </p>
                    {attachments.map((file, i) => (
                        <div key={i} className={styles.attachmentItem}>
                            <span className={styles.attachmentLink}>{file.name}</span>
                            <span className={styles.attachmentSize}>
                                {file.size < 1024
                                    ? `${file.size} B`
                                    : file.size < 1024 * 1024
                                      ? `${(file.size / 1024).toFixed(1)} KB`
                                      : `${(file.size / (1024 * 1024)).toFixed(1)} MB`}
                            </span>
                            <button
                                type="button"
                                className={styles.attachmentDelete}
                                onClick={() => setAttachments(prev => prev.filter((_, j) => j !== i))}
                            >
                                &times;
                            </button>
                        </div>
                    ))}
                    <input
                        ref={attachmentInputRef}
                        type="file"
                        accept=".pdf,.txt,.docx"
                        style={{ display: "none" }}
                        onChange={e => {
                            const file = e.target.files?.[0];
                            if (file) {
                                setAttachments(prev => [...prev, file]);
                            }
                            if (attachmentInputRef.current) {
                                attachmentInputRef.current.value = "";
                            }
                        }}
                    />
                    <Button variant="ghost" type="button" onClick={() => attachmentInputRef.current?.click()}>
                        + Add File
                    </Button>
                </div>

                <div className={styles.formActions}>
                    <Button
                        variant="ghost"
                        type="button"
                        onClick={() => navigate(isEdit ? `/mystery/${editId}` : "/mysteries")}
                    >
                        Cancel
                    </Button>
                    <Button variant="primary" type="submit" disabled={!title.trim() || !body.trim() || submitting}>
                        {submitting
                            ? isEdit
                                ? "Saving..."
                                : "Creating..."
                            : isEdit
                              ? "Save Changes"
                              : "Present Mystery"}
                    </Button>
                </div>
            </form>
        </div>
    );
}
