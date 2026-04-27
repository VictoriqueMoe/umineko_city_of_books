import { useState } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useDeleteAccount } from "../../api/mutations/auth";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import styles from "./SettingsPage.module.css";

export function DangerZoneSection() {
    const navigate = useNavigate();
    const { setUser } = useAuth();
    const [showModal, setShowModal] = useState(false);
    const [password, setPassword] = useState("");
    const [error, setError] = useState("");
    const deleteMutation = useDeleteAccount();
    const deleting = deleteMutation.isPending;

    async function handleDelete() {
        setError("");
        try {
            await deleteMutation.mutateAsync({ password });
            setUser(null);
            navigate("/");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to delete account.");
        }
    }

    return (
        <>
            <div className={styles.dangerZone}>
                <h3 className={styles.dangerTitle}>Danger Zone</h3>
                <p className={styles.dangerText}>
                    Deleting your account is permanent. All your theories, responses, and votes will be removed.
                </p>
                <Button variant="danger" onClick={() => setShowModal(true)} style={{ width: "100%" }}>
                    Delete Account
                </Button>
            </div>

            <Modal isOpen={showModal} onClose={() => setShowModal(false)} title="Delete Account">
                <div className={styles.modalBody}>
                    <p className={styles.dangerText}>
                        This action cannot be undone. Please enter your password to confirm.
                    </p>
                    {error && <div className={styles.error}>{error}</div>}
                    <label className={styles.label}>
                        Password
                        <Input type="password" fullWidth value={password} onChange={e => setPassword(e.target.value)} />
                    </label>
                    <div className={styles.modalActions}>
                        <Button variant="secondary" onClick={() => setShowModal(false)}>
                            Cancel
                        </Button>
                        <Button variant="danger" disabled={deleting || !password} onClick={handleDelete}>
                            {deleting ? "Deleting..." : "Delete My Account"}
                        </Button>
                    </div>
                </div>
            </Modal>
        </>
    );
}
