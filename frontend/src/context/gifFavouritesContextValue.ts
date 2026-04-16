import { createContext } from "react";
import type { GiphyFavourite } from "../api/endpoints";

export interface GifFavouritesContextValue {
    favourites: GiphyFavourite[];
    ids: Set<string>;
    isFavourite: (giphyID: string) => boolean;
    toggle: (fav: GiphyFavourite) => Promise<void>;
    refresh: () => Promise<void>;
}

export const GifFavouritesContext = createContext<GifFavouritesContextValue | null>(null);
