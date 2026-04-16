import { useContext } from "react";
import { GifFavouritesContext } from "../context/gifFavouritesContextValue";

export function useGifFavourites() {
    const ctx = useContext(GifFavouritesContext);
    if (!ctx) {
        throw new Error("useGifFavourites must be used within a GifFavouritesProvider");
    }
    return ctx;
}
