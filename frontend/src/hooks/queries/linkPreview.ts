import { useQuery } from "@tanstack/react-query";
import { getLinkPreview } from "../../api/endpoints/linkPreview";
import { queryKeys } from "../../api/queryKeys";

const ONE_HOUR = 60 * 60 * 1000;

export function useLinkPreview(url: string) {
    const q = useQuery({
        queryKey: queryKeys.linkPreview(url),
        queryFn: () => getLinkPreview(url),
        enabled: url.length > 0,
        staleTime: ONE_HOUR,
        gcTime: ONE_HOUR,
        retry: false,
    });

    return {
        preview: q.data,
        loading: q.isLoading,
    };
}
