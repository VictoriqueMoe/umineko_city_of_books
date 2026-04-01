import type {PostEmbed} from "../../../types/api";
import styles from "./PostEmbeds.module.css";

interface PostEmbedsProps {
    embeds: PostEmbed[];
}

function YouTubeEmbed({ embed }: { embed: PostEmbed }) {
    return (
        <div className={styles.youtube}>
            <iframe
                src={`https://www.youtube-nocookie.com/embed/${embed.video_id}`}
                title={embed.title || "YouTube video"}
                allow="accelerometer; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                allowFullScreen
                className={styles.youtubeIframe}
            />
        </div>
    );
}

function LinkEmbed({ embed }: { embed: PostEmbed }) {
    if (!embed.title && !embed.description && !embed.image) {
        return null;
    }

    return (
        <a href={embed.url} target="_blank" rel="noopener noreferrer" className={styles.linkCard}>
            {embed.image && (
                <div className={styles.linkImageWrap}>
                    <img src={embed.image} alt="" className={styles.linkImage} loading="lazy" />
                </div>
            )}
            <div className={styles.linkBody}>
                {embed.site_name && <span className={styles.linkSite}>{embed.site_name}</span>}
                {embed.title && <span className={styles.linkTitle}>{embed.title}</span>}
                {embed.description && <span className={styles.linkDesc}>{embed.description}</span>}
            </div>
        </a>
    );
}

export function PostEmbeds({ embeds }: PostEmbedsProps) {
    if (!embeds || embeds.length === 0) {
        return null;
    }

    return (
        <div className={styles.embeds}>
            {embeds.map((embed, i) => (
                <div key={`${embed.url}-${i}`}>
                    {embed.type === "youtube" ? <YouTubeEmbed embed={embed} /> : <LinkEmbed embed={embed} />}
                </div>
            ))}
        </div>
    );
}
