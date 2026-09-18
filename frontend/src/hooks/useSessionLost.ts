import { useSyncExternalStore } from "react";
import { isSessionLost, subscribeSessionLost } from "../api/sessionLost";
import { useAuth } from "./useAuth";

export function useSessionLost(): boolean {
    const { user } = useAuth();
    const lost = useSyncExternalStore(subscribeSessionLost, isSessionLost);

    return user !== null && lost;
}
