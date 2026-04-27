import { useState } from "react";
import { marked } from "marked";
import { usePageTitle } from "../../hooks/usePageTitle";
import DOMPurify from "dompurify";
import type { Announcement } from "../../types/api";
import { useAdminAnnouncements } from "../../api/queries/admin";
import {
    useCreateAnnouncement,
    useDeleteAnnouncement,
    usePinAnnouncement,
    useUpdateAnnouncement,
} from "../../api/mutations/admin";
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
    const { announcements, loading } = useAdminAnnouncements();
    const createMutation = useCreateAnnouncement();
    const updateMutation = useUpdateAnnouncement();
    const deleteMutation = useDeleteAnnouncement();
    const pinMutation = usePinAnnouncement();
    const [editingId, setEditingId] = useState<string | null>(null);
    const [title, setTitle] = useState("");
    const [body, setBody] = useState("");
    const [showPreview, setShowPreview] = useState(false);

    const saving = createMutation.isPending || updateMutation.isPending;

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
        try {
            if (editingId === "new") {
                await createMutation.mutateAsync({ title: title.trim(), body: body.trim() });
            } else if (editingId) {
                await updateMutation.mutateAsync({ id: editingId, title: title.trim(), body: body.trim() });
            }
            cancelEdit();
        } catch {
            // ignore
        }
    }

    async function handleDelete(id: string) {
        if (!window.confirm("Delete this announcement?")) {
            return;
        }
        await deleteMutation.mutateAsync(id);
    }

    async function handlePin(id: string, pinned: boolean) {
        await pinMutation.mutateAsync({ id, pinned: !pinned });
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
