import { useMutation, useQueryClient } from "@tanstack/react-query";
import { addGiphyFavourite, removeGiphyFavourite } from "../../api/endpoints/giphy";
import type { GiphyFavourite } from "../../types/api";
import { queryKeys } from "../../api/queryKeys";

export function useAddGiphyFavourite() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (fav: GiphyFavourite) => addGiphyFavourite(fav),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.giphy.favourites() }),
    });
}

export function useRemoveGiphyFavourite() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (giphyId: string) => removeGiphyFavourite(giphyId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.giphy.favourites() }),
    });
}
