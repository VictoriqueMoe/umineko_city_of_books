import styles from "./YouTubeEmbed.module.css";

interface YouTubeEmbedProps {
    videoIds: string[];
}

export function YouTubeEmbed({ videoIds }: YouTubeEmbedProps) {
    if (videoIds.length === 0) {
        return null;
    }
    return (
        <div className={styles.list}>
            {videoIds.map(id => (
                <div key={id} className={styles.frame}>
                    <iframe
                        src={`https://www.youtube-nocookie.com/embed/${id}`}
                        title="YouTube video"
                        allow="accelerometer; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                        allowFullScreen
                        loading="lazy"
                        className={styles.iframe}
                    />
                </div>
            ))}
        </div>
    );
}
