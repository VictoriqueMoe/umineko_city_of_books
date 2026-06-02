import { useState } from "react";
import { Link } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useResendVerification } from "../../api/mutations/auth";
import styles from "./VerifyEmailBanner.module.css";

const MS_PER_DAY = 1000 * 60 * 60 * 24;

function daysLeft(until: string): number {
    const target = new Date(until).getTime();
    if (isNaN(target)) {
        return 0;
    }
    return Math.ceil((target - Date.now()) / MS_PER_DAY);
}

export function VerifyEmailBanner() {
    const { user } = useAuth();
    const resend = useResendVerification();
    const [sent, setSent] = useState(false);
    const [error, setError] = useState("");

    if (!user) {
        return null;
    }
    if (user.email && user.email_verified !== false) {
        return null;
    }

    const noEmail = !user.email;
    const remaining = user.verify_grace_until ? daysLeft(user.verify_grace_until) : 0;
    const action = noEmail ? "Add an email address to your account" : `Verify your email (${user.email})`;
    const message =
        remaining > 0
            ? `${action} within ${remaining} day${remaining === 1 ? "" : "s"} to keep posting.`
            : `${action} to post again. Your account is read-only until you do.`;

    async function handleResend() {
        setError("");
        try {
            await resend.mutateAsync();
            setSent(true);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to resend");
        }
    }

    return (
        <div className={styles.banner}>
            <span className={styles.text}>{message}</span>
            {noEmail ? (
                <Link to="/set-email" className={styles.button}>
                    Add email
                </Link>
            ) : sent ? (
                <span className={styles.sent}>Sent. Check your inbox.</span>
            ) : (
                <button className={styles.button} onClick={handleResend} disabled={resend.isPending}>
                    {resend.isPending ? "Sending..." : "Resend email"}
                </button>
            )}
            {error && <span className={styles.sent}>{error}</span>}
        </div>
    );
}
