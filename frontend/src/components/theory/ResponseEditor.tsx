import React, { useState } from "react";
import { createResponse } from "../../api/endpoints";
import { useEvidence } from "../../hooks/useEvidence";
import { TruthPicker } from "../truth/TruthPicker";
import { TruthChip } from "../truth/TruthChip";

interface ResponseEditorProps {
    theoryId: number;
    parentId?: number;
    inheritedSide?: "with_love" | "without_love";
    onCreated: () => void;
}

export function ResponseEditor({ theoryId, parentId, inheritedSide, onCreated }: ResponseEditorProps) {
    const [side, setSide] = useState<"with_love" | "without_love" | null>(inheritedSide ?? null);
    const [body, setBody] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const ev = useEvidence();
    const isReply = parentId !== undefined;

    async function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault();
        if (!side || !body.trim() || submitting) {
            return;
        }

        setSubmitting(true);
        try {
            await createResponse(theoryId, { parent_id: parentId, side, body: body.trim(), evidence: ev.toInput() });
            setBody("");
            if (!isReply) {
                setSide(null);
            }
            ev.clear();
            onCreated();
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <form className="response-editor" onSubmit={handleSubmit}>
            <h4 className="section-title">{isReply ? "Reply" : "Add your response"}</h4>

            {!isReply && (
                <div className="side-selector">
                    <button
                        type="button"
                        className={`side-btn with-love${side === "with_love" ? " active" : ""}`}
                        onClick={() => setSide("with_love")}
                    >
                        <span className="side-btn-title">With love, it can be seen</span>
                        <span className="side-btn-subtitle">I support this theory</span>
                    </button>
                    <button
                        type="button"
                        className={`side-btn without-love${side === "without_love" ? " active" : ""}`}
                        onClick={() => setSide("without_love")}
                    >
                        <span className="side-btn-title">Without love, it cannot be seen</span>
                        <span className="side-btn-subtitle">I deny this theory</span>
                    </button>
                </div>
            )}

            <textarea
                className="response-textarea"
                placeholder={isReply ? "Write your reply..." : "State your argument..."}
                value={body}
                onChange={e => setBody(e.target.value)}
            />

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
                <button className="theory-submit" type="submit" disabled={!side || !body.trim() || submitting}>
                    {submitting ? "Submitting..." : isReply ? "Reply" : "Submit Response"}
                </button>
            </div>

            <TruthPicker
                isOpen={ev.pickerOpen}
                onClose={ev.closePicker}
                onSelect={ev.addQuote}
                selectedKeys={ev.selectedKeys}
            />
        </form>
    );
}
