import { useMemo } from "react";
import { useBlockedUsers } from "../api/queries/user";
import { useAuth } from "./useAuth";

export function useBlockedUserIds(): Set<string> {
    const { user } = useAuth();
    const { blocked } = useBlockedUsers(user?.id ?? "");

    return useMemo(() => {
        const ids = new Set<string>();
        for (const item of blocked) {
            ids.add(item.id);
        }
        return ids;
    }, [blocked]);
}
