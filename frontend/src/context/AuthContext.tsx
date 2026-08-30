import { type PropsWithChildren, useCallback, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { SessionUser } from "../types/api";
import { AuthContext } from "./authContextValue";
import { useMe } from "../hooks/queries/auth";
import { useLogin, useLogout, useRegister } from "../hooks/mutations/auth";
import { queryKeys } from "../api/queryKeys";

export function AuthProvider({ children }: PropsWithChildren) {
    const qc = useQueryClient();
    const { me, loading: meLoading, refresh } = useMe();
    const user = me;

    const setUser = useCallback(
        (next: SessionUser | null) => {
            qc.setQueryData<SessionUser | null>(queryKeys.auth.me(), next);
        },
        [qc],
    );

    const { mutateAsync: login } = useLogin();
    const { mutateAsync: register } = useRegister();
    const { mutateAsync: logout } = useLogout();

    const loginUser = useCallback(
        async (username: string, password: string, turnstileToken?: string) => {
            await login({ username, password, turnstileToken });
            await refresh();
        },
        [login, refresh],
    );

    const registerUser = useCallback(
        async (
            username: string,
            email: string,
            password: string,
            displayName: string,
            inviteCode?: string,
            turnstileToken?: string,
        ) => {
            await register({ username, email, password, displayName, inviteCode, turnstileToken });
            await refresh();
        },
        [register, refresh],
    );

    const logoutUser = useCallback(async () => {
        await logout();
        setUser(null);
    }, [logout, setUser]);

    const value = useMemo(
        () => ({ user, loading: meLoading, setUser, loginUser, registerUser, logoutUser }),
        [user, meLoading, setUser, loginUser, registerUser, logoutUser],
    );

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
