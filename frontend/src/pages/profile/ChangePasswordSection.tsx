import { useState } from "react";
import { useChangePassword } from "../../api/mutations/auth";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import styles from "./SettingsPage.module.css";

export function ChangePasswordSection() {
    const [oldPassword, setOldPassword] = useState("");
    const [newPassword, setNewPassword] = useState("");
    const [confirmPassword, setConfirmPassword] = useState("");
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");
    const changeMutation = useChangePassword();
    const changing = changeMutation.isPending;

    async function handleSubmit() {
        setError("");
        setSuccess("");

        if (newPassword.length < 8) {
            setError("New password must be at least 8 characters.");
            return;
        }
        if (newPassword !== confirmPassword) {
            setError("Passwords do not match.");
            return;
        }

        try {
            await changeMutation.mutateAsync({ old_password: oldPassword, new_password: newPassword });
            setSuccess("Password changed successfully.");
            setOldPassword("");
            setNewPassword("");
            setConfirmPassword("");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to change password.");
        }
    }

    return (
        <div className={styles.section}>
            <h3 className={styles.sectionTitle}>Change Password</h3>
            {error && <div className={styles.error}>{error}</div>}
            {success && <div className={styles.success}>{success}</div>}
            <label className={styles.label}>
                Current Password
                <Input type="password" fullWidth value={oldPassword} onChange={e => setOldPassword(e.target.value)} />
            </label>
            <label className={styles.label}>
                New Password
                <Input type="password" fullWidth value={newPassword} onChange={e => setNewPassword(e.target.value)} />
            </label>
            <label className={styles.label}>
                Confirm New Password
                <Input
                    type="password"
                    fullWidth
                    value={confirmPassword}
                    onChange={e => setConfirmPassword(e.target.value)}
                />
            </label>
            <Button
                variant="primary"
                type="button"
                disabled={changing}
                onClick={handleSubmit}
                style={{ width: "100%" }}
            >
                {changing ? "Changing..." : "Change Password"}
            </Button>
        </div>
    );
}
