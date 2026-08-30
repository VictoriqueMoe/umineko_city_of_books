import { useQuery } from "@tanstack/react-query";
import { getReports } from "../../api/endpoints/report";
import { queryKeys } from "../../api/queryKeys";

export function useReports(status: string) {
    const query = useQuery({
        queryKey: queryKeys.admin.reports({ status }),
        queryFn: () => getReports(status),
    });
    return { reports: query.data?.reports ?? [], loading: query.isLoading, refresh: query.refetch };
}
