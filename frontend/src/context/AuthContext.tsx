import { type PropsWithChildren, useCallback, useEffect, useState } from "react";
import type { User } from "../types/api";
import { AuthContext } from "./authContextValue";
import * as api from "../api/endpoints";

export function AuthProvider({ children }: PropsWithChildren) {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        api.getMe()
            .then(setUser)
            .catch(() => setUser(null))
            .finally(() => setLoading(false));
    }, []);

    const loginUser = useCallback(async (username: string, password: string, turnstileToken?: string) => {
        const u = await api.login(username, password, turnstileToken);
        setUser(u);
    }, []);

    const registerUser = useCallback(
        async (
            username: string,
            password: string,
            displayName: string,
            inviteCode?: string,
            turnstileToken?: string,
        ) => {
            const u = await api.register(username, password, displayName, inviteCode, turnstileToken);
            setUser(u);
        },
        [],
    );

    const logoutUser = useCallback(async () => {
        await api.logout();
        setUser(null);
    }, []);

    return (
        <AuthContext.Provider value={{ user, loading, setUser, loginUser, registerUser, logoutUser }}>
            {children}
        </AuthContext.Provider>
    );
}
