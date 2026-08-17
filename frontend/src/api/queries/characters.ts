import { useQuery } from "@tanstack/react-query";
import { getCharacterGroups, getCharacters, searchOCCharacters, type CharacterGroups, type Series } from "../endpoints";
import { queryKeys } from "../queryKeys";

const EMPTY: { umineko: Record<string, string>; higurashi: Record<string, string>; ciconia: CharacterGroups } = {
    umineko: {},
    higurashi: {},
    ciconia: { main: {}, additional: {} },
};

export function useAllCharacters() {
    const query = useQuery({
        queryKey: queryKeys.characters.all(),
        queryFn: async () => {
            const [umineko, higurashi, ciconia] = await Promise.all([
                getCharacters("umineko"),
                getCharacters("higurashi"),
                getCharacterGroups("ciconia"),
            ]);
            return { umineko, higurashi, ciconia };
        },
        staleTime: Infinity,
    });
    return query.data ?? EMPTY;
}

export function useCharactersFlat(series: Series) {
    const q = useQuery({
        queryKey: queryKeys.characters.flat(series),
        queryFn: () => getCharacters(series),
        staleTime: Infinity,
    });
    return { characters: q.data ?? {}, loading: q.isLoading };
}

export function useOCCharacters(query = "") {
    const q = useQuery({
        queryKey: queryKeys.characters.oc(query),
        queryFn: () => searchOCCharacters(query),
        staleTime: Infinity,
    });
    return { characters: q.data ?? [], loading: q.isLoading };
}

const EMPTY_GROUPS: CharacterGroups = { main: {}, additional: {} };

export function useCharacterGroups(series: Series) {
    const q = useQuery({
        queryKey: queryKeys.characters.groups(series),
        queryFn: () => getCharacterGroups(series),
        staleTime: Infinity,
    });
    return { groups: q.data ?? EMPTY_GROUPS, loading: q.isLoading };
}
