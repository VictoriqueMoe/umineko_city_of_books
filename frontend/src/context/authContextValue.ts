import { createContext } from "react";
import type { UserProfile } from "../types/api";

export interface AuthContextValue {
    user: UserProfile | null;
    loading: boolean;
    setUser: (user: UserProfile | null) => void;
    loginUser: (username: string, password: string, turnstileToken?: string) => Promise<void>;
    registerUser: (
        username: string,
        password: string,
        displayName: string,
        inviteCode?: string,
        turnstileToken?: string,
    ) => Promise<void>;
    logoutUser: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextValue | null>(null);
