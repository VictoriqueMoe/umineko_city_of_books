import React, {useEffect, useState} from "react";
import {useNavigate, useParams} from "react-router";
import {useAuth} from "../hooks/useAuth";
import {useEvidence} from "../hooks/useEvidence";
import {getTheory, updateTheory} from "../api/endpoints";
import {TruthPicker} from "../components/truth/TruthPicker";
import {TruthChip} from "../components/truth/TruthChip";

export function EditTheoryPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const theoryId = parseInt(id ?? "0", 10);
    const [title, setTitle] = useState("");
    const [body, setBody] = useState("");
    const [episode, setEpisode] = useState(0);
    const [submitting, setSubmitting] = useState(false);
    const [loading, setLoading] = useState(true);
    const ev = useEvidence();

    useEffect(() => {
        if (!theoryId) {
            return;
        }
        getTheory(theoryId)
            .then(theory => {
                setTitle(theory.title);
                setBody(theory.body);
                setEpisode(theory.episode);
                setLoading(false);
            })
            .catch(() => {
                navigate("/");
            });
    }, [theoryId, navigate]);

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    if (authLoading || !user) {
        return null;
    }

    if (loading) {
        return <div className="loading">Loading theory...</div>;
    }

    async function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault();
        if (!title.trim() || !body.trim() || submitting) {
            return;
        }

        setSubmitting(true);
        try {
            await updateTheory(theoryId, {
                title: title.trim(),
                body: body.trim(),
                episode,
                evidence: ev.toInput(),
            });
            navigate(`/theory/${theoryId}`);
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <div className="create-theory-page">
            <h2 className="logic-title">Edit Theory</h2>

            <form onSubmit={handleSubmit}>
                <input
                    type="text"
                    className="theory-title-input"
                    placeholder="Theory title..."
                    value={title}
                    onChange={e => setTitle(e.target.value)}
                    maxLength={200}
                />

                <textarea
                    className="theory-textarea"
                    placeholder="State your theory..."
                    value={body}
                    onChange={e => setBody(e.target.value)}
                />

                <select className="filter-select" value={episode} onChange={e => setEpisode(Number(e.target.value))}>
                    <option value={0}>General (no specific episode)</option>
                    {[1, 2, 3, 4, 5, 6, 7, 8].map(ep => (
                        <option key={ep} value={ep}>
                            Episode {ep}
                        </option>
                    ))}
                </select>

                {ev.evidence.length > 0 && (
                    <div className="editor-evidence">
                        {ev.evidence.map((item, i) => (
                            <div key={item.quote.audioId} className="editor-evidence-item">
                                <TruthChip quote={item.quote} onRemove={() => ev.removeAt(i)} />
                                <input
                                    type="text"
                                    className="evidence-note-input"
                                    placeholder="Why is this relevant?"
                                    value={item.note}
                                    onChange={e => ev.updateNote(i, e.target.value)}
                                />
                            </div>
                        ))}
                    </div>
                )}

                <div className="editor-actions">
                    <button type="button" className="evidence-add-btn" onClick={ev.openPicker}>
                        + Attach Evidence
                    </button>
                    <button
                        className="theory-submit"
                        type="submit"
                        disabled={!title.trim() || !body.trim() || submitting}
                    >
                        {submitting ? "Saving..." : "Save Changes"}
                    </button>
                </div>
            </form>

            <TruthPicker
                isOpen={ev.pickerOpen}
                onClose={ev.closePicker}
                onSelect={ev.addQuote}
                selectedKeys={ev.selectedKeys}
            />
        </div>
    );
}
