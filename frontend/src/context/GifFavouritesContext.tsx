import { type PropsWithChildren, useCallback, useMemo } from "react";
import type { GiphyFavourite } from "../api/endpoints";
import { useGiphyFavourites } from "../api/queries/giphy";
import { useAddGiphyFavourite, useRemoveGiphyFavourite } from "../api/mutations/giphy";
import { useAuth } from "../hooks/useAuth";
import { GifFavouritesContext } from "./gifFavouritesContextValue";

export function GifFavouritesProvider({ children }: PropsWithChildren) {
    const { user } = useAuth();
    const { favourites: rawFavourites, refresh: refreshQuery } = useGiphyFavourites(0, 500);
    const favourites = useMemo<GiphyFavourite[]>(() => (user ? rawFavourites : []), [user, rawFavourites]);
    const addMutation = useAddGiphyFavourite();
    const removeMutation = useRemoveGiphyFavourite();

    const refresh = useCallback(async () => {
        await refreshQuery();
    }, [refreshQuery]);

    const ids = useMemo(() => new Set(favourites.map(f => f.giphy_id)), [favourites]);

    const isFavourite = useCallback((giphyID: string) => ids.has(giphyID), [ids]);

    const toggle = useCallback(
        async (fav: GiphyFavourite) => {
            if (!user || !fav.giphy_id) {
                return;
            }
            if (ids.has(fav.giphy_id)) {
                try {
                    await removeMutation.mutateAsync(fav.giphy_id);
                } catch {
                    return;
                }
                return;
            }
            try {
                await addMutation.mutateAsync(fav);
            } catch {
                return;
            }
        },
        [ids, user, removeMutation, addMutation],
    );

    const value = useMemo(
        () => ({ favourites, ids, isFavourite, toggle, refresh }),
        [favourites, ids, isFavourite, toggle, refresh],
    );

    return <GifFavouritesContext.Provider value={value}>{children}</GifFavouritesContext.Provider>;
}
