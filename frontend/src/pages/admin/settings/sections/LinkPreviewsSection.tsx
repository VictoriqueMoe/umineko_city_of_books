import { Button } from "../../../../components/Button/Button";
import { EmbedPreviews } from "./EmbedPreviews";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function LinkPreviewsSection({ form }: SectionProps) {
    const { settings, defaultSiteName, ogImageInputRef, ogImage } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Link Previews</h2>
            <div className={styles.fieldGroup}>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>
                        Default embed image shown when a link to the site is shared on Discord, X, and other platforms,
                        and the page has no image of its own. JPG only.
                    </span>
                    <div className={styles.embedActions}>
                        <Button variant="secondary" onClick={ogImage.choose} disabled={ogImage.uploading}>
                            {ogImage.uploading ? "Uploading..." : "Upload image"}
                        </Button>
                        {ogImage.hasCustom && (
                            <Button variant="secondary" onClick={ogImage.clear}>
                                Reset to built-in
                            </Button>
                        )}
                        {ogImage.error && <span className={styles.saveError}>{ogImage.error}</span>}
                    </div>
                    <input
                        ref={ogImageInputRef}
                        type="file"
                        accept="image/jpeg,.jpg"
                        className={styles.hiddenInput}
                        onChange={ogImage.onSelected}
                    />
                </div>
                <EmbedPreviews
                    image={settings.og_default_image || "/Featherine.jpg"}
                    siteName={settings.site_name ?? defaultSiteName}
                    baseURL={settings.base_url ?? ""}
                />
            </div>
        </div>
    );
}
