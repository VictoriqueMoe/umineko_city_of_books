import { useNavigate } from "react-router";
import type { Art } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import styles from "./ArtCard.module.css";

interface ArtCardProps {
    art: Art;
}

export function ArtCard({ art }: ArtCardProps) {
    const navigate = useNavigate();
    const imgSrc = art.thumbnail_url || art.image_url;

    return (
        <div className={styles.card} onClick={() => navigate(`/gallery/art/${art.id}`)}>
            <div className={styles.imageWrap}>
                <img
                    src={imgSrc}
                    alt={art.title}
                    className={styles.image}
                    loading="lazy"
                    onError={e => {
                        if (e.currentTarget.src !== art.image_url) {
                            e.currentTarget.src = art.image_url;
                        }
                    }}
                />
            </div>
            <div className={styles.info}>
                <span className={styles.title}>{art.title}</span>
                <div className={styles.meta}>
                    <ProfileLink user={art.author} size="small" />
                    <span className={styles.likes}>&#9829; {art.like_count}</span>
                </div>
            </div>
        </div>
    );
}
