import { createContext } from "react";
import type { User } from "../types/api";

export interface AuthContextValue {
    user: User | null;
    loading: boolean;
    setUser: (user: User | null) => void;
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
