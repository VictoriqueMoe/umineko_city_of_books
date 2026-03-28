import { createContext } from "react";
import type { User } from "../types/api";

export interface AuthContextValue {
    user: User | null;
    loading: boolean;
    loginUser: (username: string, password: string) => Promise<void>;
    registerUser: (username: string, password: string, displayName: string) => Promise<void>;
    logoutUser: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextValue | null>(null);
