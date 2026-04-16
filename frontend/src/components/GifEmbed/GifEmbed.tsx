import { useAuth } from "../../hooks/useAuth";
import { useGifFavourites } from "../../hooks/useGifFavourites";
import { extractGiphyId } from "../../utils/gif";
import styles from "./GifEmbed.module.css";

interface GifEmbedProps {
    src: string;
    alt?: string;
    imgClassName?: string;
    onClick?: () => void;
}

export function GifEmbed({ src, alt = "GIF", imgClassName, onClick }: GifEmbedProps) {
    const { user } = useAuth();
    const { isFavourite, toggle } = useGifFavourites();
    const giphyID = extractGiphyId(src);
    const starred = giphyID ? isFavourite(giphyID) : false;

    async function handleStar(e: React.MouseEvent) {
        e.stopPropagation();
        if (!giphyID) {
            return;
        }
        await toggle({
            giphy_id: giphyID,
            url: src,
            title: alt === "GIF" ? "" : alt,
            preview_url: src,
            width: 0,
            height: 0,
        });
    }

    return (
        <div className={styles.wrapper}>
            <img className={imgClassName} src={src} alt={alt} loading="lazy" onClick={onClick} />
            {user && giphyID && (
                <button
                    type="button"
                    className={`${styles.star} ${starred ? styles.starFilled : ""}`}
                    onClick={handleStar}
                    aria-label={starred ? "Remove from favourites" : "Add to favourites"}
                    title={starred ? "Remove from favourites" : "Add to favourites"}
                >
                    {starred ? "\u2605" : "\u2606"}
                </button>
            )}
        </div>
    );
}
