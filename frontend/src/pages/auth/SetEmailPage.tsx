import React, { useState } from "react";
import { Navigate, useNavigate } from "react-router";
import { useSetEmail } from "../../api/mutations/auth";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import styles from "./LoginPage.module.css";

export function SetEmailPage() {
    usePageTitle("Add your email");
    const navigate = useNavigate();
    const { user, loading } = useAuth();
    const setEmailMutation = useSetEmail();
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);

    if (loading) {
        return <div className={styles.page}>Loading...</div>;
    }
    if (!user) {
        return <Navigate to="/login" replace />;
    }
    if (user.email) {
        return <Navigate to="/" replace />;
    }

    async function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault();
        setError("");
        setSubmitting(true);

        try {
            await setEmailMutation.mutateAsync({ email, password });
            navigate("/");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Something went wrong.");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <div className={styles.page}>
            <div className={styles.card}>
                <h2 className={styles.title}>Add your email</h2>
                <p className={styles.hint}>
                    We now require an email address so you can recover your account and keep posting. Enter one along
                    with your current password to continue; we will send a confirmation link to verify it.
                </p>

                {error && <div className={styles.error}>{error}</div>}

                <form onSubmit={handleSubmit}>
                    <Input
                        type="email"
                        fullWidth
                        placeholder="Email"
                        value={email}
                        onChange={e => setEmail(e.target.value)}
                        autoComplete="email"
                    />
                    <Input
                        type="password"
                        fullWidth
                        placeholder="Current password"
                        value={password}
                        onChange={e => setPassword(e.target.value)}
                        autoComplete="current-password"
                    />
                    <Button
                        variant="primary"
                        type="submit"
                        disabled={!email || !password || submitting}
                        style={{ width: "100%", marginTop: "0.5rem" }}
                    >
                        {submitting ? "..." : "Save and continue"}
                    </Button>
                </form>
            </div>
        </div>
    );
}
