import { useQuery } from "@tanstack/react-query";
import { getSecret, listSecrets } from "../../api/endpoints/secret";
import { queryKeys } from "../../api/queryKeys";

export function useSecretList() {
    const q = useQuery({ queryKey: queryKeys.secrets.list(), queryFn: () => listSecrets() });
    return { data: q.data ?? null, loading: q.isLoading, refresh: q.refetch };
}

export function useSecret(id: string) {
    const q = useQuery({
        queryKey: queryKeys.secrets.detail(id),
        queryFn: () => getSecret(id),
        enabled: !!id,
    });
    return { data: q.data ?? null, loading: q.isLoading, refresh: q.refetch };
}
