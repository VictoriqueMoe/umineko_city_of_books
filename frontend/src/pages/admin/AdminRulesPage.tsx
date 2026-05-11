import { useState } from "react";
import { marked } from "marked";
import DOMPurify from "dompurify";
import { useAdminSettings } from "../../api/queries/admin";
import { useUpdateAdminSettings } from "../../api/mutations/admin";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import styles from "./AdminAnnouncements.module.css";

function renderMarkdown(md: string): string {
    const raw = marked.parse(md, { async: false }) as string;
    return DOMPurify.sanitize(raw);
}

export function AdminRulesPage() {
    usePageTitle("Admin - Rules");
    const { settings, loading } = useAdminSettings();
    const updateMutation = useUpdateAdminSettings();
    const [draft, setDraft] = useState<string | null>(null);
    const [showPreview, setShowPreview] = useState(false);
    const [feedback, setFeedback] = useState("");

    const saved = (settings?.rules_page as string | undefined) ?? "";
    const body = draft ?? saved;

    async function handleSave() {
        if (!settings) {
            return;
        }
        setFeedback("");
        try {
            await updateMutation.mutateAsync({ ...settings, rules_page: body });
            setDraft(null);
            setFeedback("Saved");
        } catch (e) {
            setFeedback(e instanceof Error ? e.message : "Failed to save");
        }
    }

    if (loading) {
        return <div className="loading">Loading...</div>;
    }

    const saving = updateMutation.isPending;

    return (
        <div>
            <div className={styles.editor}>
                <h3 className={styles.editorTitle}>Rules Page</h3>
                <p style={{ color: "var(--text-muted)", fontSize: "0.85rem", marginBottom: "0.5rem" }}>
                    Site-wide rules shown at /rules. Leave empty to hide the page and sidebar link.
                </p>
                <div className={styles.tabBar}>
                    <button
                        className={`${styles.tabBtn}${!showPreview ? ` ${styles.tabBtnActive}` : ""}`}
                        onClick={() => setShowPreview(false)}
                    >
                        Write
                    </button>
                    <button
                        className={`${styles.tabBtn}${showPreview ? ` ${styles.tabBtnActive}` : ""}`}
                        onClick={() => setShowPreview(true)}
                    >
                        Preview
                    </button>
                </div>
                {showPreview ? (
                    <div className={styles.preview} dangerouslySetInnerHTML={{ __html: renderMarkdown(body) }} />
                ) : (
                    <textarea
                        className={styles.textarea}
                        placeholder="Write the rules in Markdown..."
                        value={body}
                        onChange={e => {
                            setDraft(e.target.value);
                            setFeedback("");
                        }}
                        rows={18}
                    />
                )}
                <div className={styles.editorActions}>
                    {feedback && (
                        <span style={{ color: "var(--text-muted)", fontSize: "0.85rem", alignSelf: "center" }}>
                            {feedback}
                        </span>
                    )}
                    <Button variant="primary" onClick={handleSave} disabled={saving}>
                        {saving ? "Saving..." : "Save"}
                    </Button>
                </div>
            </div>
        </div>
    );
}
