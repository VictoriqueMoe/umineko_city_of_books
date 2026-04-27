import { useState } from "react";
import { useCreateReport } from "../../api/mutations/misc";
import { useAuth } from "../../hooks/useAuth";
import { Button } from "../Button/Button";
import { Input } from "../Input/Input";
import { Modal } from "../Modal/Modal";
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
    const [submitted, setSubmitted] = useState(false);
    const [error, setError] = useState("");
    const reportMutation = useCreateReport();

    if (!user) {
        return null;
    }

    async function handleSubmit() {
        if (!reason.trim() || reportMutation.isPending) {
            return;
        }
        setError("");
        try {
            await reportMutation.mutateAsync({ targetType, targetId, reason: reason.trim(), contextId });
            setSubmitted(true);
            setReason("");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to submit report");
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
                            <Button
                                variant="danger"
                                onClick={handleSubmit}
                                disabled={reportMutation.isPending || !reason.trim()}
                            >
                                {reportMutation.isPending ? "Submitting..." : "Submit Report"}
                            </Button>
                        </div>
                    </div>
                )}
            </Modal>
        </>
    );
}
