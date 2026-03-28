import React, {useState} from "react";
import {useNavigate} from "react-router";
import {useAuth} from "../hooks/useAuth";

export function LoginPage() {
    const navigate = useNavigate();
    const { loginUser, registerUser } = useAuth();
    const [isRegister, setIsRegister] = useState(false);
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [displayName, setDisplayName] = useState("");
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);

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
        <div className="login-page">
            <div className="login-card">
                <h2 className="login-title">{isRegister ? "Join the Game Board" : "Enter the Game Board"}</h2>

                {error && <div className="login-error">{error}</div>}

                <form onSubmit={handleSubmit}>
                    <input
                        type="text"
                        className="login-input"
                        placeholder="Username"
                        value={username}
                        onChange={e => setUsername(e.target.value)}
                        autoComplete="username"
                    />
                    <input
                        type="password"
                        className="login-input"
                        placeholder="Password"
                        value={password}
                        onChange={e => setPassword(e.target.value)}
                        autoComplete={isRegister ? "new-password" : "current-password"}
                    />
                    {isRegister && (
                        <input
                            type="text"
                            className="login-input"
                            placeholder="Display Name (optional)"
                            value={displayName}
                            onChange={e => setDisplayName(e.target.value)}
                        />
                    )}

                    <button className="login-submit" type="submit" disabled={!username || !password || loading}>
                        {loading ? "..." : isRegister ? "Register" : "Sign In"}
                    </button>
                </form>

                <button className="login-toggle" onClick={() => setIsRegister(!isRegister)}>
                    {isRegister ? "Already have an account? Sign in" : "Need an account? Register"}
                </button>
            </div>
        </div>
    );
}
