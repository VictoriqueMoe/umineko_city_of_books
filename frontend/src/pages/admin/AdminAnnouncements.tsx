import { useCallback, useEffect, useState } from "react";
import { marked } from "marked";
import { usePageTitle } from "../../hooks/usePageTitle";
import DOMPurify from "dompurify";
import type { Announcement } from "../../types/api";
import {
    createAnnouncement,
    deleteAnnouncement,
    listAnnouncements,
    pinAnnouncement,
    updateAnnouncement,
} from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { relativeTime } from "../../utils/notifications";
import styles from "./AdminAnnouncements.module.css";

function renderMarkdown(md: string): string {
    const raw = marked.parse(md, { async: false }) as string;
    return DOMPurify.sanitize(raw);
}

export function AdminAnnouncements() {
    usePageTitle("Admin - Announcements");
    const [announcements, setAnnouncements] = useState<Announcement[]>([]);
    const [loading, setLoading] = useState(true);
    const [editingId, setEditingId] = useState<string | null>(null);
    const [title, setTitle] = useState("");
    const [body, setBody] = useState("");
    const [saving, setSaving] = useState(false);
    const [showPreview, setShowPreview] = useState(false);

    const fetchAll = useCallback(async () => {
        try {
            const data = await listAnnouncements(100, 0);
            setAnnouncements(data.announcements);
        } catch {
            setAnnouncements([]);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchAll();
    }, [fetchAll]);

    function startCreate() {
        setEditingId("new");
        setTitle("");
        setBody("");
        setShowPreview(false);
    }

    function startEdit(a: Announcement) {
        setEditingId(a.id);
        setTitle(a.title);
        setBody(a.body);
        setShowPreview(false);
    }

    function cancelEdit() {
        setEditingId(null);
        setTitle("");
        setBody("");
    }

    async function handleSave() {
        if (!title.trim() || !body.trim() || saving) {
            return;
        }
        setSaving(true);
        try {
            if (editingId === "new") {
                await createAnnouncement(title.trim(), body.trim());
            } else if (editingId) {
                await updateAnnouncement(editingId, title.trim(), body.trim());
            }
            cancelEdit();
            await fetchAll();
        } catch {
            // ignore
        } finally {
            setSaving(false);
        }
    }

    async function handleDelete(id: string) {
        if (!window.confirm("Delete this announcement?")) {
            return;
        }
        await deleteAnnouncement(id);
        await fetchAll();
    }

    async function handlePin(id: string, pinned: boolean) {
        await pinAnnouncement(id, !pinned);
        await fetchAll();
    }

    if (loading) {
        return <div className="loading">Loading...</div>;
    }

    return (
        <div>
            {editingId ? (
                <div className={styles.editor}>
                    <h3 className={styles.editorTitle}>
                        {editingId === "new" ? "Create Announcement" : "Edit Announcement"}
                    </h3>
                    <Input
                        type="text"
                        fullWidth
                        placeholder="Announcement title..."
                        value={title}
                        onChange={e => setTitle(e.target.value)}
                    />
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
                            placeholder="Write your announcement in Markdown..."
                            value={body}
                            onChange={e => setBody(e.target.value)}
                            rows={12}
                        />
                    )}
                    <div className={styles.editorActions}>
                        <Button variant="ghost" onClick={cancelEdit}>
                            Cancel
                        </Button>
                        <Button
                            variant="primary"
                            onClick={handleSave}
                            disabled={!title.trim() || !body.trim() || saving}
                        >
                            {saving ? "Saving..." : editingId === "new" ? "Publish" : "Save Changes"}
                        </Button>
                    </div>
                </div>
            ) : (
                <Button variant="primary" onClick={startCreate}>
                    Create Announcement
                </Button>
            )}

            <div className={styles.list}>
                {announcements.length === 0 && <div className="empty-state">No announcements yet.</div>}
                {announcements.map(a => (
                    <div key={a.id} className={styles.item}>
                        <div className={styles.itemHeader}>
                            <span className={styles.itemTitle}>{a.title}</span>
                            {a.pinned && <span className={styles.pinnedBadge}>Pinned</span>}
                        </div>
                        <div className={styles.itemMeta}>
                            <ProfileLink user={a.author} size="small" />
                            <span>{relativeTime(a.created_at)}</span>
                        </div>
                        <div className={styles.itemActions}>
                            <Button variant="ghost" size="small" onClick={() => startEdit(a)}>
                                Edit
                            </Button>
                            <Button variant="ghost" size="small" onClick={() => handlePin(a.id, a.pinned)}>
                                {a.pinned ? "Unpin" : "Pin"}
                            </Button>
                            <Button variant="danger" size="small" onClick={() => handleDelete(a.id)}>
                                Delete
                            </Button>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}
