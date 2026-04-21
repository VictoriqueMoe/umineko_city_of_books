import { useEffect, useState } from "react";
import { getAdminSettings, updateAdminSettings } from "../../api/endpoints";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { TextArea } from "../../components/TextArea/TextArea";
import type { SiteSettings } from "../../types/api";
import styles from "./AdminSettings.module.css";

const pages = [
    { key: "rules_landing", label: "Welcome (Landing)" },
    { key: "rules_theories", label: "Theories (Umineko)" },
    { key: "rules_theories_higurashi", label: "Theories (Higurashi)" },
    { key: "rules_mysteries", label: "Mysteries" },
    { key: "rules_ships", label: "Ships" },
    { key: "rules_fanfiction", label: "Fanfiction" },
    { key: "rules_journals", label: "Reading Journals" },
    { key: "rules_game_board", label: "Game Board (General)" },
    { key: "rules_game_board_umineko", label: "Game Board (Umineko)" },
    { key: "rules_game_board_higurashi", label: "Game Board (Higurashi)" },
    { key: "rules_game_board_ciconia", label: "Game Board (Ciconia)" },
    { key: "rules_game_board_higanbana", label: "Game Board (Higanbana)" },
    { key: "rules_game_board_roseguns", label: "Game Board (Rose Guns Days)" },
    { key: "rules_gallery", label: "Gallery (General)" },
    { key: "rules_gallery_umineko", label: "Gallery (Umineko)" },
    { key: "rules_gallery_higurashi", label: "Gallery (Higurashi)" },
    { key: "rules_gallery_ciconia", label: "Gallery (Ciconia)" },
    { key: "rules_suggestions", label: "Site Improvements" },
    { key: "rules_chat_rooms", label: "Chat Rooms" },
    { key: "rules_games", label: "Games" },
];

export function AdminContentRules() {
    usePageTitle("Admin - Content Rules");
    const [settings, setSettings] = useState<SiteSettings>({});
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");

    useEffect(() => {
        getAdminSettings()
            .then(setSettings)
            .catch(e => setError(e.message))
            .finally(() => setLoading(false));
    }, []);

    async function handleSave() {
        setSaving(true);
        setError("");
        setSuccess("");
        try {
            await updateAdminSettings(settings);
            setSuccess("Rules saved successfully");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to save rules");
        } finally {
            setSaving(false);
        }
    }

    if (loading) {
        return <div className={styles.loading}>Loading rules...</div>;
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.title}>Content Rules</h1>

            {pages.map(page => (
                <div key={page.key} className={styles.card}>
                    <h2 className={styles.sectionTitle}>{page.label}</h2>
                    <div className={styles.fieldGroup}>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>
                                Rules displayed at the top of the page. Leave empty to hide.
                            </span>
                            <TextArea
                                value={settings[page.key] ?? ""}
                                onChange={e => {
                                    setSettings(prev => ({ ...prev, [page.key]: e.target.value }));
                                    setSuccess("");
                                }}
                                rows={5}
                                placeholder="Enter rules for this section..."
                            />
                        </div>
                    </div>
                </div>
            ))}

            <div className={styles.saveRow}>
                <Button variant="primary" onClick={handleSave} disabled={saving}>
                    {saving ? "Saving..." : "Save Rules"}
                </Button>
                {error && <span className={styles.saveError}>{error}</span>}
                {success && <span className={styles.success}>{success}</span>}
            </div>
        </div>
    );
}
