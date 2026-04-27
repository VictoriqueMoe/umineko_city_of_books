import { useQuery } from "@tanstack/react-query";
import { getSecret, listSecrets } from "../endpoints";

export function useSecretList() {
    const q = useQuery({ queryKey: ["secrets", "list"], queryFn: () => listSecrets() });
    return { data: q.data ?? null, loading: q.isPending, refresh: q.refetch };
}

export function useSecret(id: string) {
    const q = useQuery({
        queryKey: ["secrets", "detail", id],
        queryFn: () => getSecret(id),
        enabled: !!id,
    });
    return { data: q.data ?? null, loading: q.isPending, refresh: q.refetch };
}
