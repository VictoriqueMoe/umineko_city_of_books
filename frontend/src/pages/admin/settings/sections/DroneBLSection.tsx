import { Button } from "../../../../components/Button/Button";
import { Input } from "../../../../components/Input/Input";
import { ToggleSwitch } from "../../../../components/ToggleSwitch/ToggleSwitch";
import { DRONEBL_CLASSES } from "../../../../domain/dronebl";
import { KNOWN_CRAWLER_FEEDS } from "../../../../domain/crawlerFeeds";
import { isEnabled } from "../../../../domain/siteSettings";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function DroneBLSection({ form }: SectionProps) {
    const { settings, updateField, toggleField, dronebl, feeds } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>DroneBL</h2>
            <div className={styles.fieldGroup}>
                <ToggleSwitch
                    label="Enable DroneBL"
                    description="Refuse requests from addresses listed on the DroneBL public abuse blocklist. Signed-in members are never blocked, and a blocked visitor can still reach the login page."
                    enabled={isEnabled(settings.dronebl_enabled)}
                    onChange={v => toggleField("dronebl_enabled", v)}
                />
                {isEnabled(settings.dronebl_enabled) && (
                    <>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Listings To Ignore</span>
                            <span className={styles.fieldHint}>
                                Tick a reason to stop it blocking. Anything left unticked still blocks. DroneBL only
                                lists addresses it has evidence against, so a well behaved VPN is usually not listed at
                                all.
                            </span>
                            <div className={styles.classGrid}>
                                {DRONEBL_CLASSES.map(cls => {
                                    const ignored = dronebl.ignoredClasses.has(cls.id);
                                    return (
                                        <label key={cls.id} className={styles.classRow}>
                                            <input
                                                type="checkbox"
                                                checked={ignored}
                                                onChange={e => dronebl.toggleClass(cls.id, e.target.checked)}
                                            />
                                            <span className={styles.className}>
                                                {cls.label} <code className={styles.classId}>{cls.id}</code>
                                                {cls.note && <em className={styles.classNote}>{cls.note}</em>}
                                            </span>
                                        </label>
                                    );
                                })}
                            </div>
                        </div>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Allowlist</span>
                            <Input
                                value={settings.dronebl_allowlist ?? ""}
                                onChange={e => updateField("dronebl_allowlist", e.target.value)}
                                fullWidth
                                placeholder="203.0.113.4, 2600:387:15:4015::/64"
                            />
                            <span className={styles.fieldHint}>
                                Comma separated addresses or CIDR ranges that are never checked. Put your own address
                                here so a false positive cannot lock you out of this page.
                            </span>
                        </div>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Search Engine Crawlers</span>
                            <span className={styles.fieldHint}>
                                These publish the addresses their crawlers use, and are never blocked. Untick one and it
                                is treated like any other visitor, which can cost you search indexing.
                            </span>
                            <div className={styles.classGrid}>
                                {KNOWN_CRAWLER_FEEDS.map(known => (
                                    <label key={known.name} className={styles.classRow}>
                                        <input
                                            type="checkbox"
                                            checked={feeds.isKnownEnabled(known)}
                                            onChange={e => feeds.toggleKnown(known, e.target.checked)}
                                        />
                                        <span className={styles.className}>{known.label}</span>
                                    </label>
                                ))}
                            </div>
                        </div>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Other Crawler Feeds</span>
                            <span className={styles.fieldHint}>
                                Any endpoint publishing the same format Google, Bing and Apple use: a prefixes array of
                                ipv4Prefix or ipv6Prefix entries. Every URL is fetched when you save, and one that fails
                                will block the save.
                            </span>
                            <div className={styles.feedList}>
                                {feeds.custom.map((entry, i) => (
                                    <div key={i} className={styles.feedRow}>
                                        <Input
                                            value={entry.name}
                                            onChange={e => feeds.update(i, { name: e.target.value })}
                                            placeholder="name"
                                            aria-label={`Feed ${i + 1} name`}
                                        />
                                        <Input
                                            value={entry.url}
                                            onChange={e => feeds.update(i, { url: e.target.value })}
                                            placeholder="https://example.com/ranges.json"
                                            aria-label={`Feed ${i + 1} url`}
                                            fullWidth
                                        />
                                        <button
                                            type="button"
                                            className={styles.feedRemove}
                                            onClick={() => feeds.remove(i)}
                                            aria-label={`Remove feed ${i + 1}`}
                                        >
                                            &times;
                                        </button>
                                    </div>
                                ))}
                            </div>
                            <div>
                                <Button variant="ghost" size="small" onClick={feeds.add}>
                                    + Add Feed
                                </Button>
                            </div>
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}
