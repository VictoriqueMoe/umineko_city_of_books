import { useQuery } from "@tanstack/react-query";
import { getCharacterGroups, getCharacters, searchOCCharacters, type CharacterGroups, type Series } from "../endpoints";

const EMPTY: { umineko: Record<string, string>; higurashi: Record<string, string>; ciconia: CharacterGroups } = {
    umineko: {},
    higurashi: {},
    ciconia: { main: {}, additional: {} },
};

export function useAllCharacters() {
    const query = useQuery({
        queryKey: ["characters", "all"],
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
        queryKey: ["characters", "flat", series],
        queryFn: () => getCharacters(series),
        staleTime: Infinity,
    });
    return { characters: q.data ?? {}, loading: q.isPending };
}

export function useOCCharacters(query = "") {
    const q = useQuery({
        queryKey: ["characters", "oc", query],
        queryFn: () => searchOCCharacters(query),
        staleTime: Infinity,
    });
    return { characters: q.data ?? [], loading: q.isPending };
}

const EMPTY_GROUPS: CharacterGroups = { main: {}, additional: {} };

export function useCharacterGroups(series: Series) {
    const q = useQuery({
        queryKey: ["character-groups", series],
        queryFn: () => getCharacterGroups(series),
        staleTime: Infinity,
    });
    return { groups: q.data ?? EMPTY_GROUPS, loading: q.isPending };
}
