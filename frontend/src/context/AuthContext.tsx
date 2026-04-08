import { type PropsWithChildren, useCallback, useEffect, useState } from "react";
import type { UserProfile } from "../types/api";
import { AuthContext } from "./authContextValue";
import * as api from "../api/endpoints";

export function AuthProvider({ children }: PropsWithChildren) {
    const [user, setUser] = useState<UserProfile | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        api.getMe()
            .then(setUser)
            .catch(() => setUser(null))
            .finally(() => setLoading(false));
    }, []);

    const loginUser = useCallback(async (username: string, password: string, turnstileToken?: string) => {
        await api.login(username, password, turnstileToken);
        const me = await api.getMe();
        setUser(me);
    }, []);

    const registerUser = useCallback(
        async (
            username: string,
            password: string,
            displayName: string,
            inviteCode?: string,
            turnstileToken?: string,
        ) => {
            await api.register(username, password, displayName, inviteCode, turnstileToken);
            const me = await api.getMe();
            setUser(me);
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
