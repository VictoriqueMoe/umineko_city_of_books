import { useQuery } from "@tanstack/react-query";
import { getUserProfile } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useProfile(username: string) {
    const query = useQuery({
        queryKey: queryKeys.profile.byUsername(username),
        queryFn: () => getUserProfile(username),
        enabled: !!username,
    });
    return {
        profile: query.data ?? null,
        loading: query.isPending,
    };
}
