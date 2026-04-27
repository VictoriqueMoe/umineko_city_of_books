import { useState } from "react";
import { useNavigate } from "react-router";
import { useCreatePost } from "../../../api/mutations/post";
import { Modal } from "../../Modal/Modal";
import { Select } from "../../Select/Select";
import { MentionTextArea } from "../../MentionTextArea/MentionTextArea";
import { Button } from "../../Button/Button";
import styles from "./ShareDialog.module.css";

interface ShareDialogProps {
    isOpen: boolean;
    onClose: () => void;
    contentId: string;
    contentType: string;
    contentTitle?: string;
    onShared?: () => void;
}

export function ShareDialog({ isOpen, onClose, contentId, contentType, contentTitle, onShared }: ShareDialogProps) {
    const navigate = useNavigate();
    const [corner, setCorner] = useState("general");
    const [message, setMessage] = useState("");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const createMutation = useCreatePost();

    async function handleSubmit() {
        if (loading) {
            return;
        }
        setLoading(true);
        setError("");
        try {
            const result = await createMutation.mutateAsync({
                body: message,
                corner,
                sharedContentId: contentId,
                sharedContentType: contentType,
            });
            if (onShared) {
                onShared();
            }
            onClose();
            navigate(`/game-board/${result.id}`);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to share");
        } finally {
            setLoading(false);
        }
    }

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Share to Game Board">
            <div className={styles.content}>
                <label className={styles.label}>
                    Corner
                    <Select value={corner} onChange={e => setCorner(e.target.value)}>
                        <option value="general">General</option>
                        <option value="umineko">Umineko</option>
                        <option value="higurashi">Higurashi</option>
                        <option value="ciconia">Ciconia</option>
                    </Select>
                </label>

                <MentionTextArea
                    value={message}
                    onChange={setMessage}
                    placeholder="Add a comment (optional)"
                    rows={3}
                    showColours
                />

                <p className={styles.preview}>Sharing: {contentTitle || contentType}</p>

                {error && <p className={styles.error}>{error}</p>}

                <div className={styles.actions}>
                    <Button variant="ghost" onClick={onClose} disabled={loading}>
                        Cancel
                    </Button>
                    <Button variant="primary" onClick={handleSubmit} disabled={loading}>
                        {loading ? "Sharing..." : "Share"}
                    </Button>
                </div>
            </div>
        </Modal>
    );
}
