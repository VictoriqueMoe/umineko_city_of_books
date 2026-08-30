import { useQuery } from "@tanstack/react-query";
import { getSession } from "../../api/endpoints/auth";
import { getUserProfile } from "../../api/endpoints/user";
import type { SessionUser } from "../../types/api";
import { queryKeys } from "../../api/queryKeys";

async function loadMe(): Promise<SessionUser | null> {
    const session = await getSession();
    if (!session.authenticated || !session.username) {
        return null;
    }

    const profile = await getUserProfile(session.username);

    return { ...profile, permissions: session.permissions ?? [] };
}

export function useMe() {
    const query = useQuery({
        queryKey: queryKeys.auth.me(),
        queryFn: () => loadMe(),
    });
    return { me: query.data ?? null, loading: query.isLoading, refresh: query.refetch };
}
