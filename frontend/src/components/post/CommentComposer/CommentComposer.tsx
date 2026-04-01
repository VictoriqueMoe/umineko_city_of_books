import {useState} from "react";
import {createComment} from "../../../api/endpoints";
import {Button} from "../../Button/Button";
import {TextArea} from "../../TextArea/TextArea";
import styles from "./CommentComposer.module.css";

interface CommentComposerProps {
    postId: string;
    onCreated: () => void;
}

export function CommentComposer({ postId, onCreated }: CommentComposerProps) {
    const [body, setBody] = useState("");
    const [submitting, setSubmitting] = useState(false);

    async function handleSubmit() {
        if (!body.trim() || submitting) {
            return;
        }
        setSubmitting(true);
        try {
            await createComment(postId, body.trim());
            setBody("");
            onCreated();
        } catch {
            void 0;
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <div className={styles.composer}>
            <TextArea placeholder="Write a comment..." value={body} onChange={e => setBody(e.target.value)} rows={2} />
            <div className={styles.bar}>
                <Button variant="primary" size="small" onClick={handleSubmit} disabled={submitting || !body.trim()}>
                    {submitting ? "..." : "Comment"}
                </Button>
            </div>
        </div>
    );
}
