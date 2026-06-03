import React, { useRef, useState } from "react";
import { useNavigate } from "react-router";
import { Turnstile, type TurnstileInstance } from "@marsidev/react-turnstile";
import { useForgotPassword } from "../../api/mutations/auth";
import { useStaff } from "../../api/queries/auth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { ROLE_GROUPS } from "../../utils/permissions";
import styles from "./LoginPage.module.css";

export function ForgotPasswordPage() {
    usePageTitle("Forgot Password");
    const navigate = useNavigate();
    const siteInfo = useSiteInfo();
    const { staff } = useStaff();
    const forgotPasswordMutation = useForgotPassword();
    const [username, setUsername] = useState("");
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");
    const [loading, setLoading] = useState(false);
    const [turnstileToken, setTurnstileToken] = useState("");
    const turnstileRef = useRef<TurnstileInstance>(null);

    const turnstileEnabled = siteInfo.turnstile_enabled;
    const turnstileSiteKey = siteInfo.turnstile_site_key;

    async function handleSubmit(e: React.FormEvent) {
        e.preventDefault();
        setError("");
        setSuccess("");

        if (turnstileEnabled && !turnstileToken) {
            setError("Please complete the verification.");
            return;
        }

        setLoading(true);

        try {
            await forgotPasswordMutation.mutateAsync({
                username,
                turnstileToken: turnstileEnabled ? turnstileToken : undefined,
            });
            setSuccess("A password reset link has been sent to the email on your account.");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Something went wrong.");
            setTurnstileToken("");
            turnstileRef.current?.reset();
        } finally {
            setLoading(false);
        }
    }

    return (
        <div className={styles.page}>
            <div className={styles.card}>
                <h2 className={styles.title}>Reset your password</h2>

                {error && <div className={styles.error}>{error}</div>}
                {success && <div className={styles.success}>{success}</div>}

                {!success && (
                    <>
                        <p className={styles.hint}>
                            Enter your username and we will email a reset link to the address on your account.
                        </p>
                        <form onSubmit={handleSubmit}>
                            <Input
                                type="text"
                                fullWidth
                                placeholder="Username"
                                value={username}
                                onChange={e => setUsername(e.target.value)}
                                autoComplete="username"
                            />

                            {turnstileEnabled && turnstileSiteKey && (
                                <div className={styles.turnstile}>
                                    <Turnstile
                                        ref={turnstileRef}
                                        siteKey={turnstileSiteKey}
                                        onSuccess={setTurnstileToken}
                                        onExpire={() => setTurnstileToken("")}
                                        options={{
                                            refreshExpired: "auto",
                                            theme: "dark",
                                        }}
                                    />
                                </div>
                            )}

                            <Button
                                variant="primary"
                                type="submit"
                                disabled={!username || loading || (turnstileEnabled && !turnstileToken)}
                                style={{ width: "100%", marginTop: "0.5rem" }}
                            >
                                {loading ? "..." : "Send reset link"}
                            </Button>
                        </form>
                    </>
                )}

                {staff.length > 0 && (
                    <div className={styles.staffContact}>
                        <p className={styles.hint}>
                            No email on your account? You will not be able to reset your password yourself. Please
                            contact one of our admins:
                        </p>
                        {ROLE_GROUPS.map(group => {
                            const members = staff.filter(member => member.role === group.role);
                            if (members.length === 0) {
                                return null;
                            }
                            return (
                                <div key={group.role} className={styles.staffGroup}>
                                    <span className={styles.staffGroupLabel}>{group.label}</span>
                                    <ul className={styles.staffList}>
                                        {members.map(member => (
                                            <li key={member.id}>
                                                <ProfileLink user={member} size="small" showRoles={false} />
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            );
                        })}
                    </div>
                )}

                <Button variant="ghost" onClick={() => navigate("/login")} style={{ width: "100%", marginTop: "1rem" }}>
                    Back to sign in
                </Button>
            </div>
        </div>
    );
}
