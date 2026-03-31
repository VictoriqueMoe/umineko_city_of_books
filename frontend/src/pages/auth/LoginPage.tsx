import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { Turnstile } from "@marsidev/react-turnstile";
import { useAuth } from "../../hooks/useAuth";
import { getSiteInfo } from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import styles from "./LoginPage.module.css";

export function LoginPage() {
    const navigate = useNavigate();
    const { loginUser, registerUser } = useAuth();
    const [isRegister, setIsRegister] = useState(false);
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [displayName, setDisplayName] = useState("");
    const [inviteCode, setInviteCode] = useState("");
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);
    const [regType, setRegType] = useState<"open" | "invite" | "closed">("open");
    const [turnstileEnabled, setTurnstileEnabled] = useState(false);
    const [turnstileSiteKey, setTurnstileSiteKey] = useState("");
    const [turnstileToken, setTurnstileToken] = useState("");

    useEffect(() => {
        getSiteInfo()
            .then(info => {
                setRegType(info.registration_type as "open" | "invite" | "closed");
                setTurnstileEnabled(info.turnstile_enabled);
                setTurnstileSiteKey(info.turnstile_site_key);
            })
            .catch(() => {});
    }, []);

    async function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault();
        setError("");

        if (turnstileEnabled && !turnstileToken) {
            setError("Please complete the verification.");
            return;
        }

        setLoading(true);

        try {
            if (isRegister) {
                await registerUser(
                    username,
                    password,
                    displayName || username,
                    inviteCode || undefined,
                    turnstileEnabled ? turnstileToken : undefined,
                );
            } else {
                await loginUser(username, password, turnstileEnabled ? turnstileToken : undefined);
            }
            navigate("/");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Something went wrong.");
            setTurnstileToken("");
        } finally {
            setLoading(false);
        }
    }

    const canRegister = regType !== "closed";

    return (
        <div className={styles.page}>
            <div className={styles.card}>
                <h2 className={styles.title}>{isRegister ? "Join the Game Board" : "Enter the Game Board"}</h2>

                {error && <div className={styles.error}>{error}</div>}

                <form onSubmit={handleSubmit}>
                    <Input
                        type="text"
                        fullWidth
                        placeholder="Username"
                        value={username}
                        onChange={e => setUsername(e.target.value)}
                        autoComplete="username"
                    />
                    <Input
                        type="password"
                        fullWidth
                        placeholder="Password"
                        value={password}
                        onChange={e => setPassword(e.target.value)}
                        autoComplete={isRegister ? "new-password" : "current-password"}
                    />
                    {isRegister && (
                        <>
                            <Input
                                type="text"
                                fullWidth
                                placeholder="Display Name (optional)"
                                value={displayName}
                                onChange={e => setDisplayName(e.target.value)}
                            />
                            {regType === "invite" && (
                                <Input
                                    type="text"
                                    fullWidth
                                    placeholder="Invite Code"
                                    value={inviteCode}
                                    onChange={e => setInviteCode(e.target.value)}
                                />
                            )}
                        </>
                    )}

                    {turnstileEnabled && turnstileSiteKey && (
                        <div className={styles.turnstile}>
                            <Turnstile
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
                        disabled={
                            !username ||
                            !password ||
                            loading ||
                            (isRegister && regType === "invite" && !inviteCode) ||
                            (turnstileEnabled && !turnstileToken)
                        }
                        style={{ width: "100%", marginTop: "0.5rem" }}
                    >
                        {loading ? "..." : isRegister ? "Register" : "Sign In"}
                    </Button>
                </form>

                {canRegister ? (
                    <Button
                        variant="ghost"
                        onClick={() => setIsRegister(!isRegister)}
                        style={{ width: "100%", marginTop: "1rem" }}
                    >
                        {isRegister ? "Already have an account? Sign in" : "Need an account? Register"}
                    </Button>
                ) : (
                    !isRegister && <p className={styles.disabledNotice}>Registration is currently closed.</p>
                )}
            </div>
        </div>
    );
}
