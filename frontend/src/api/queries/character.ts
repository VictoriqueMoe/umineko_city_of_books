import { useQuery } from "@tanstack/react-query";
import { listCharacters } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useCharacterList(series: string, enabled = true) {
    const q = useQuery({
        queryKey: queryKeys.characters.series(series),
        queryFn: () => listCharacters(series),
        enabled: enabled && !!series && series !== "oc",
        staleTime: Infinity,
    });
    return { characters: q.data?.characters ?? [], loading: q.isLoading };
}
