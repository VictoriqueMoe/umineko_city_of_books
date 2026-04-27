import { useMutation, useQueryClient } from "@tanstack/react-query";
import { addGiphyFavourite, removeGiphyFavourite, type GiphyFavourite } from "../endpoints";

export function useAddGiphyFavourite() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (fav: GiphyFavourite) => addGiphyFavourite(fav),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["giphy", "favourites"] }),
    });
}

export function useRemoveGiphyFavourite() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (giphyId: string) => removeGiphyFavourite(giphyId),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["giphy", "favourites"] }),
    });
}
