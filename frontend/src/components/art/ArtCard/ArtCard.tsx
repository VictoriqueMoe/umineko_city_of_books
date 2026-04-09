import { Link } from "react-router";
import type { Art } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { SpoilerImage } from "../../SpoilerImage/SpoilerImage";
import styles from "./ArtCard.module.css";

interface ArtCardProps {
    art: Art;
}

export function ArtCard({ art }: ArtCardProps) {
    const imgSrc = art.thumbnail_url || art.image_url;

    return (
        <Link to={`/gallery/art/${art.id}`} className={styles.card}>
            <SpoilerImage
                src={imgSrc}
                alt={art.title}
                isSpoiler={art.is_spoiler}
                className={styles.imageWrap}
                imageClassName={styles.image}
                loading="lazy"
                onError={e => {
                    if (e.currentTarget.src !== art.image_url) {
                        e.currentTarget.src = art.image_url;
                    }
                }}
            />
            <div className={styles.info}>
                <span className={styles.title}>{art.title}</span>
                <div className={styles.meta}>
                    <ProfileLink user={art.author} size="small" clickable={false} />
                    <span className={styles.likes}>&#9829; {art.like_count}</span>
                </div>
            </div>
        </Link>
    );
}
