import React, { useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { useResetPassword } from "../../api/mutations/auth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import styles from "./LoginPage.module.css";

export function ResetPasswordPage() {
    usePageTitle("Reset Password");
    const navigate = useNavigate();
    const [params] = useSearchParams();
    const token = params.get("token") ?? "";
    const resetPasswordMutation = useResetPassword();
    const [password, setPassword] = useState("");
    const [confirm, setConfirm] = useState("");
    const [error, setError] = useState("");
    const [success, setSuccess] = useState(false);
    const [loading, setLoading] = useState(false);

    async function handleSubmit(e: React.FormEvent) {
        e.preventDefault();
        setError("");

        if (password !== confirm) {
            setError("Passwords do not match.");
            return;
        }

        setLoading(true);

        try {
            await resetPasswordMutation.mutateAsync({ token, newPassword: password });
            setSuccess(true);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Something went wrong.");
        } finally {
            setLoading(false);
        }
    }

    if (!token) {
        return (
            <div className={styles.page}>
                <div className={styles.card}>
                    <h2 className={styles.title}>Reset your password</h2>
                    <div className={styles.error}>This reset link is invalid or incomplete.</div>
                    <Button
                        variant="ghost"
                        onClick={() => navigate("/forgot-password")}
                        style={{ width: "100%", marginTop: "1rem" }}
                    >
                        Request a new link
                    </Button>
                </div>
            </div>
        );
    }

    return (
        <div className={styles.page}>
            <div className={styles.card}>
                <h2 className={styles.title}>Choose a new password</h2>

                {error && <div className={styles.error}>{error}</div>}

                {success ? (
                    <>
                        <div className={styles.success}>
                            Your password has been reset. You can now sign in with your new password.
                        </div>
                        <Button
                            variant="primary"
                            onClick={() => navigate("/login")}
                            style={{ width: "100%", marginTop: "0.5rem" }}
                        >
                            Go to sign in
                        </Button>
                    </>
                ) : (
                    <form onSubmit={handleSubmit}>
                        <Input
                            type="password"
                            fullWidth
                            placeholder="New password"
                            value={password}
                            onChange={e => setPassword(e.target.value)}
                            autoComplete="new-password"
                        />
                        <Input
                            type="password"
                            fullWidth
                            placeholder="Confirm new password"
                            value={confirm}
                            onChange={e => setConfirm(e.target.value)}
                            autoComplete="new-password"
                        />

                        <Button
                            variant="primary"
                            type="submit"
                            disabled={!password || !confirm || loading}
                            style={{ width: "100%", marginTop: "0.5rem" }}
                        >
                            {loading ? "..." : "Reset password"}
                        </Button>
                    </form>
                )}
            </div>
        </div>
    );
}
