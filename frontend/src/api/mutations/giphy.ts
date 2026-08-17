import { useMutation, useQueryClient } from "@tanstack/react-query";
import { addGiphyFavourite, removeGiphyFavourite, type GiphyFavourite } from "../endpoints";
import { queryKeys } from "../queryKeys";

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
