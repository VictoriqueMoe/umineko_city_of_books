import { type PropsWithChildren, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { UserProfile } from "../types/api";
import { AuthContext } from "./authContextValue";
import { useMe } from "../api/queries/auth";
import { useLogin, useLogout, useRegister } from "../api/mutations/auth";

export function AuthProvider({ children }: PropsWithChildren) {
    const qc = useQueryClient();
    const { me, loading: meLoading, refresh } = useMe();
    const user = me;

    const setUser = useCallback(
        (next: UserProfile | null) => {
            qc.setQueryData<UserProfile | null>(["auth", "me"], next);
        },
        [qc],
    );

    const loginMutation = useLogin();
    const registerMutation = useRegister();
    const logoutMutation = useLogout();

    const loginUser = useCallback(
        async (username: string, password: string, turnstileToken?: string) => {
            await loginMutation.mutateAsync({ username, password, turnstileToken });
            await refresh();
        },
        [loginMutation, refresh],
    );

    const registerUser = useCallback(
        async (
            username: string,
            password: string,
            displayName: string,
            inviteCode?: string,
            turnstileToken?: string,
        ) => {
            await registerMutation.mutateAsync({ username, password, displayName, inviteCode, turnstileToken });
            await refresh();
        },
        [registerMutation, refresh],
    );

    const logoutUser = useCallback(async () => {
        await logoutMutation.mutateAsync();
        setUser(null);
    }, [logoutMutation, setUser]);

    return (
        <AuthContext.Provider value={{ user, loading: meLoading, setUser, loginUser, registerUser, logoutUser }}>
            {children}
        </AuthContext.Provider>
    );
}
