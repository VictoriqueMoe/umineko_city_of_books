import {useState} from "react";
import {createReport} from "../../api/endpoints";
import {useAuth} from "../../hooks/useAuth";
import {Button} from "../Button/Button";
import {Input} from "../Input/Input";
import {Modal} from "../Modal/Modal";
import styles from "./ReportButton.module.css";

interface ReportButtonProps {
    targetType: string;
    targetId: string;
    contextId?: string;
}

export function ReportButton({ targetType, targetId, contextId }: ReportButtonProps) {
    const { user } = useAuth();
    const [open, setOpen] = useState(false);
    const [reason, setReason] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [submitted, setSubmitted] = useState(false);
    const [error, setError] = useState("");

    if (!user) {
        return null;
    }

    async function handleSubmit() {
        if (!reason.trim() || submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            await createReport(targetType, targetId, reason.trim(), contextId);
            setSubmitted(true);
            setReason("");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to submit report");
        } finally {
            setSubmitting(false);
        }
    }

    function handleClose() {
        setOpen(false);
        setSubmitted(false);
        setError("");
        setReason("");
    }

    return (
        <>
            <Button variant="ghost" size="small" onClick={() => setOpen(true)}>
                Report
            </Button>
            <Modal isOpen={open} onClose={handleClose} title="Report Content">
                {submitted ? (
                    <div className={styles.body}>
                        <p className={styles.success}>Report submitted. A moderator will review it.</p>
                        <div className={styles.actions}>
                            <Button variant="primary" onClick={handleClose}>
                                Close
                            </Button>
                        </div>
                    </div>
                ) : (
                    <div className={styles.body}>
                        <Input
                            fullWidth
                            type="text"
                            placeholder="Why are you reporting this?"
                            value={reason}
                            onChange={e => setReason(e.target.value)}
                        />
                        {error && <div className={styles.error}>{error}</div>}
                        <div className={styles.actions}>
                            <Button variant="secondary" onClick={handleClose}>
                                Cancel
                            </Button>
                            <Button variant="danger" onClick={handleSubmit} disabled={submitting || !reason.trim()}>
                                {submitting ? "Submitting..." : "Submit Report"}
                            </Button>
                        </div>
                    </div>
                )}
            </Modal>
        </>
    );
}
