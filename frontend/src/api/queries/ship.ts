import { useQuery } from "@tanstack/react-query";
import { getShip, listShips } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useShipList(params: {
    sort?: string;
    series?: string;
    character?: string;
    crackships?: boolean;
    limit?: number;
    offset?: number;
}) {
    const q = useQuery({
        queryKey: queryKeys.ship.feed(params),
        queryFn: () => listShips(params),
    });
    return { ships: q.data?.ships ?? [], total: q.data?.total ?? 0, loading: q.isPending };
}

export function useShip(id: string) {
    const q = useQuery({
        queryKey: queryKeys.ship.detail(id),
        queryFn: () => getShip(id),
        enabled: !!id,
    });
    return { ship: q.data ?? null, loading: q.isPending, refresh: q.refetch };
}
