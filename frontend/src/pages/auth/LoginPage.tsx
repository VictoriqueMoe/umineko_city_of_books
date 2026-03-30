import React, {useEffect, useState} from "react";
import {useNavigate} from "react-router";
import {useAuth} from "../../hooks/useAuth";
import {getSiteInfo} from "../../api/endpoints";
import {Button} from "../../components/Button/Button";
import {Input} from "../../components/Input/Input";
import styles from "./LoginPage.module.css";

export function LoginPage() {
    const navigate = useNavigate();
    const { loginUser, registerUser } = useAuth();
    const [isRegister, setIsRegister] = useState(false);
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [displayName, setDisplayName] = useState("");
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);
    const [registrationEnabled, setRegistrationEnabled] = useState(true);

    useEffect(() => {
        getSiteInfo()
            .then(info => setRegistrationEnabled(info.registration_enabled))
            .catch(() => {});
    }, []);

    async function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault();
        setError("");
        setLoading(true);

        try {
            if (isRegister) {
                await registerUser(username, password, displayName || username);
            } else {
                await loginUser(username, password);
            }
            navigate("/");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Something went wrong.");
        } finally {
            setLoading(false);
        }
    }

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
                        <Input
                            type="text"
                            fullWidth
                            placeholder="Display Name (optional)"
                            value={displayName}
                            onChange={e => setDisplayName(e.target.value)}
                        />
                    )}

                    <Button
                        variant="primary"
                        type="submit"
                        disabled={!username || !password || loading}
                        style={{ width: "100%", marginTop: "0.5rem" }}
                    >
                        {loading ? "..." : isRegister ? "Register" : "Sign In"}
                    </Button>
                </form>

                {registrationEnabled ? (
                    <Button
                        variant="ghost"
                        onClick={() => setIsRegister(!isRegister)}
                        style={{ width: "100%", marginTop: "1rem" }}
                    >
                        {isRegister ? "Already have an account? Sign in" : "Need an account? Register"}
                    </Button>
                ) : (
                    !isRegister && (
                        <p className={styles.disabledNotice}>Registration is currently closed.</p>
                    )
                )}
            </div>
        </div>
    );
}
