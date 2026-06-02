import { useEffect, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { useVerifyEmail } from "../../api/mutations/auth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import styles from "./LoginPage.module.css";

type Status = "verifying" | "ok" | "error";

export function VerifyEmailPage() {
    usePageTitle("Verify Email");
    const navigate = useNavigate();
    const [params] = useSearchParams();
    const token = params.get("token") ?? "";
    const verifyMutation = useVerifyEmail();
    const [status, setStatus] = useState<Status>(token ? "verifying" : "error");
    const started = useRef(false);

    useEffect(() => {
        if (started.current || !token) {
            return;
        }
        started.current = true;
        verifyMutation
            .mutateAsync(token)
            .then(() => setStatus("ok"))
            .catch(() => setStatus("error"));
    }, [token, verifyMutation]);

    return (
        <div className={styles.page}>
            <div className={styles.card}>
                <h2 className={styles.title}>Verify your email</h2>

                {status === "verifying" && <p className={styles.hint}>Verifying your email...</p>}

                {status === "ok" && (
                    <>
                        <div className={styles.success}>Your email is verified. Thank you!</div>
                        <Button
                            variant="primary"
                            onClick={() => navigate("/")}
                            style={{ width: "100%", marginTop: "0.5rem" }}
                        >
                            Continue
                        </Button>
                    </>
                )}

                {status === "error" && (
                    <>
                        <div className={styles.error}>This verification link is invalid or has expired.</div>
                        <Button
                            variant="ghost"
                            onClick={() => navigate("/")}
                            style={{ width: "100%", marginTop: "1rem" }}
                        >
                            Go home
                        </Button>
                    </>
                )}
            </div>
        </div>
    );
}
