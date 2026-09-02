import styles from "../AdminSettings.module.css";

const EMBED_PREVIEW_DESCRIPTION =
    "Welcome to the game board. Declare blue truths, solve mysteries, debate pairings, read and write fanfiction, and chronicle your journey through When They Cry.";

export function EmbedPreviews({ image, siteName, baseURL }: { image: string; siteName: string; baseURL: string }) {
    const domain = baseURL.replace(/^https?:\/\//, "").replace(/\/$/, "") || "whentheycry.social";

    return (
        <div className={styles.embedPreviews}>
            <div className={styles.embedPreviewColumn}>
                <span className={styles.embedPreviewLabel}>Discord</span>
                <div className={styles.discordPreview}>
                    <div className={styles.discordBar} />
                    <div className={styles.discordBody}>
                        <span className={styles.discordSite}>{siteName}</span>
                        <span className={styles.discordTitle}>{siteName}</span>
                        <span className={styles.discordDesc}>{EMBED_PREVIEW_DESCRIPTION}</span>
                        <img src={image} alt="Embed preview" className={styles.discordImage} />
                    </div>
                </div>
            </div>
            <div className={styles.embedPreviewColumn}>
                <span className={styles.embedPreviewLabel}>X / Twitter</span>
                <div className={styles.twitterPreview}>
                    <img src={image} alt="Embed preview" className={styles.twitterImage} />
                    <div className={styles.twitterBody}>
                        <span className={styles.twitterDomain}>{domain}</span>
                        <span className={styles.twitterTitle}>{siteName}</span>
                        <span className={styles.twitterDesc}>{EMBED_PREVIEW_DESCRIPTION}</span>
                    </div>
                </div>
            </div>
        </div>
    );
}
