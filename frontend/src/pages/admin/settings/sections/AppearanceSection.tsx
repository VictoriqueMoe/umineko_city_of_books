import { Select } from "../../../../components/Select/Select";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function AppearanceSection({ form }: SectionProps) {
    const { settings, updateField } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Appearance</h2>
            <div className={styles.fieldGroup}>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Default Theme</span>
                    <Select
                        value={settings.default_theme ?? "featherine"}
                        onChange={e => updateField("default_theme", e.target.value)}
                    >
                        <option value="featherine">Featherine</option>
                        <option value="beatrice">Beatrice</option>
                        <option value="bernkastel">Bernkastel</option>
                        <option value="lambdadelta">Lambdadelta</option>
                        <option value="erika">Erika Furudo</option>
                        <option value="battler">Battler Ushiromiya</option>
                        <option value="virgilia">Virgilia</option>
                        <option value="rika">Rika Furude</option>
                        <option value="mion">Mion Sonozaki</option>
                        <option value="satoko">Satoko Houjou</option>
                        <option value="miyao">Miyao</option>
                        <option value="lingji">Lingji</option>
                        <option value="stanislaw">Stanis&#322;aw</option>
                    </Select>
                </div>
            </div>
        </div>
    );
}
