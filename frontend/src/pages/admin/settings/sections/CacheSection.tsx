import { Input } from "../../../../components/Input/Input";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function CacheSection({ form }: SectionProps) {
    const { settings, updateField, getNumber } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Cache</h2>
            <div className={styles.fieldGroup}>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Valkey URL</span>
                    <Input
                        value={settings.valkey_url ?? ""}
                        onChange={e => updateField("valkey_url", e.target.value)}
                        fullWidth
                        placeholder="redis://valkey-cache:6379/0"
                    />
                    <span className={styles.fieldHint}>
                        Connection URL for the app cache, separate from the LiveKit coordination Valkey. Changes take
                        effect immediately, with no restart needed.
                    </span>
                    <span className={styles.fieldHint}>
                        Caching cannot be switched off. Leave this empty and the built-in in-memory cache is used as the
                        primary store. Set a URL and Valkey takes over, with the in-memory cache demoted to a fallback:
                        if Valkey stops responding, reads and writes are served from memory automatically and Valkey is
                        retried about every 30 seconds. Invalidations are sent to both, so a recovered Valkey never
                        serves a stale value.
                    </span>
                    <span className={styles.fieldHint}>
                        The in-memory cache is per-process, and anything cached without its own expiry is capped at 60
                        seconds. It is a safety net, not a substitute for Valkey.
                    </span>
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>In-Memory Cache Size (MB)</span>
                    <Input
                        type="number"
                        value={getNumber("cache_in_memory_max_mb")}
                        onChange={e => updateField("cache_in_memory_max_mb", e.target.value)}
                    />
                    <span className={styles.fieldHint}>
                        How much memory the built-in cache may use, counting the stored value, its key and the per-entry
                        overhead. When it is full the least recently used entries are dropped to make room. Applies
                        whether the in-memory cache is the primary store or only the fallback, so it also bounds how
                        much is held while Valkey is unreachable. Accepted range is 1 to 4096 MB, and anything outside
                        it falls back to 128. Lowering it takes effect immediately and evicts straight away rather than
                        waiting for the next write.
                    </span>
                </div>
            </div>
        </div>
    );
}
