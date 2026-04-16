import { type PropsWithChildren, useCallback, useEffect, useMemo, useState } from "react";
import { addGiphyFavourite, type GiphyFavourite, listGiphyFavourites, removeGiphyFavourite } from "../api/endpoints";
import { useAuth } from "../hooks/useAuth";
import { GifFavouritesContext } from "./gifFavouritesContextValue";

export function GifFavouritesProvider({ children }: PropsWithChildren) {
    const { user } = useAuth();
    const [favourites, setFavourites] = useState<GiphyFavourite[]>([]);

    const refresh = useCallback(async () => {
        if (!user) {
            setFavourites([]);
            return;
        }
        try {
            const r = await listGiphyFavourites(0, 500);
            setFavourites(r.data ?? []);
        } catch {
            setFavourites([]);
        }
    }, [user]);

    useEffect(() => {
        let cancelled = false;
        const fetcher = user
            ? listGiphyFavourites(0, 500).then(r => r.data ?? [])
            : Promise.resolve<GiphyFavourite[]>([]);
        fetcher
            .then(rows => {
                if (!cancelled) {
                    setFavourites(rows);
                }
            })
            .catch(() => {
                if (!cancelled) {
                    setFavourites([]);
                }
            });
        return () => {
            cancelled = true;
        };
    }, [user]);

    const ids = useMemo(() => new Set(favourites.map(f => f.giphy_id)), [favourites]);

    const isFavourite = useCallback((giphyID: string) => ids.has(giphyID), [ids]);

    const toggle = useCallback(
        async (fav: GiphyFavourite) => {
            if (!user || !fav.giphy_id) {
                return;
            }
            if (ids.has(fav.giphy_id)) {
                try {
                    await removeGiphyFavourite(fav.giphy_id);
                } catch {
                    return;
                }
                setFavourites(prev => prev.filter(f => f.giphy_id !== fav.giphy_id));
                return;
            }
            try {
                await addGiphyFavourite(fav);
            } catch {
                return;
            }
            setFavourites(prev => [fav, ...prev.filter(f => f.giphy_id !== fav.giphy_id)]);
        },
        [ids, user],
    );

    const value = useMemo(
        () => ({ favourites, ids, isFavourite, toggle, refresh }),
        [favourites, ids, isFavourite, toggle, refresh],
    );

    return <GifFavouritesContext.Provider value={value}>{children}</GifFavouritesContext.Provider>;
}
