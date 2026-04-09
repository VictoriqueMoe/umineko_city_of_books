import React, { useRef, useState } from "react";
import { useNavigate } from "react-router";
import { Turnstile, type TurnstileInstance } from "@marsidev/react-turnstile";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import styles from "./LoginPage.module.css";

export function LoginPage() {
    usePageTitle("Sign In");
    const navigate = useNavigate();
    const { loginUser, registerUser } = useAuth();
    const siteInfo = useSiteInfo();
    const [isRegister, setIsRegister] = useState(false);
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [displayName, setDisplayName] = useState("");
    const [inviteCode, setInviteCode] = useState("");
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);
    const [turnstileToken, setTurnstileToken] = useState("");
    const turnstileRef = useRef<TurnstileInstance>(null);

    const regType = siteInfo.registration_type as "open" | "invite" | "closed";
    const turnstileEnabled = siteInfo.turnstile_enabled;
    const turnstileSiteKey = siteInfo.turnstile_site_key;

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
            turnstileRef.current?.reset();
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
