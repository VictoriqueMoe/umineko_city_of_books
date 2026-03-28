import React, {useEffect, useState} from "react";
import {useNavigate} from "react-router";
import {useAuth} from "../hooks/useAuth";
import {useEvidence} from "../hooks/useEvidence";
import {createTheory} from "../api/endpoints";
import {TruthPicker} from "../components/truth/TruthPicker";
import {TruthChip} from "../components/truth/TruthChip";

export function CreateTheoryPage() {
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const [title, setTitle] = useState("");
    const [body, setBody] = useState("");
    const [episode, setEpisode] = useState(0);
    const [submitting, setSubmitting] = useState(false);
    const ev = useEvidence();

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    if (authLoading || !user) {
        return null;
    }

    async function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault();
        if (!title.trim() || !body.trim() || submitting) {
            return;
        }

        setSubmitting(true);
        try {
            const result = await createTheory({
                title: title.trim(),
                body: body.trim(),
                episode,
                evidence: ev.toInput(),
            });
            navigate(`/theory/${result.id}`);
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <div className="create-theory-page">
            <h2 className="logic-title">Declare Your Blue Truth</h2>

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
                        {submitting ? "Declaring..." : "Declare Blue Truth"}
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
