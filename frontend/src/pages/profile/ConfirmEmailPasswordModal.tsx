import { useState } from "react";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import styles from "./SettingsPage.module.css";

interface ConfirmEmailPasswordModalProps {
    isOpen: boolean;
    newEmail: string;
    onConfirm: (password: string) => void;
    onCancel: () => void;
}

export function ConfirmEmailPasswordModal({ isOpen, newEmail, onConfirm, onCancel }: ConfirmEmailPasswordModalProps) {
    const [password, setPassword] = useState("");

    function handleCancel() {
        setPassword("");
        onCancel();
    }

    function handleConfirm() {
        const entered = password;
        setPassword("");
        onConfirm(entered);
    }

    return (
        <Modal isOpen={isOpen} onClose={handleCancel} title="Confirm your password">
            <p>
                You are changing your email address to <strong>{newEmail}</strong>. Enter your current password to
                confirm. We will email your previous address to let you know it changed.
            </p>
            <label className={styles.label}>
                Current password
                <Input
                    type="password"
                    fullWidth
                    value={password}
                    autoFocus
                    onChange={e => setPassword(e.target.value)}
                    onKeyDown={e => {
                        if (e.key === "Enter" && password.length > 0) {
                            e.preventDefault();
                            handleConfirm();
                        }
                    }}
                />
            </label>
            <div className={styles.modalActions}>
                <Button type="button" variant="secondary" onClick={handleCancel}>
                    Cancel
                </Button>
                <Button type="button" variant="primary" onClick={handleConfirm} disabled={password.length === 0}>
                    Confirm
                </Button>
            </div>
        </Modal>
    );
}
