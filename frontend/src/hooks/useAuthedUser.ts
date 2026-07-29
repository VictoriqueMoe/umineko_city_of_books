import type { UserProfile } from "../types/api";
import { useAuth } from "./useAuth";

export function useAuthedUser(): UserProfile {
    const { user } = useAuth();
    if (!user) {
        throw new Error("useAuthedUser must be used beneath a ProtectedRoute");
    }
    return user;
}
