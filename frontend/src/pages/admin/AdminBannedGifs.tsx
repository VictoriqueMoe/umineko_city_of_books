import { useState } from "react";
import type { BannedGiphyEntry } from "../../api/endpoints";
import { useBannedGifs } from "../../api/queries/admin";
import { useAddBannedGif, useRemoveBannedGif } from "../../api/mutations/admin";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { formatFullDateTime } from "../../utils/time";
import styles from "./AdminBannedGifs.module.css";

function formatDate(s: string): string {
    return formatFullDateTime(s, "en-GB");
}

export function AdminBannedGifs() {
    usePageTitle("Admin - Banned GIFs");
    const { entries, loading } = useBannedGifs();
    const addMutation = useAddBannedGif();
    const removeMutation = useRemoveBannedGif();
    const [error, setError] = useState("");
    const [input, setInput] = useState("");
    const [reason, setReason] = useState("");
    const [removing, setRemoving] = useState<string | null>(null);

    const saving = addMutation.isPending;

    async function handleAdd() {
        if (!input.trim() || saving) {
            return;
        }
        setError("");
        try {
            await addMutation.mutateAsync({ input: input.trim(), reason: reason.trim() });
            setInput("");
            setReason("");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to add");
        }
    }

    async function handleRemove(entry: BannedGiphyEntry) {
        if (!window.confirm(`Remove ${entry.kind} "${entry.value}" from the banlist?`)) {
            return;
        }
        const key = `${entry.kind}:${entry.value}`;
        setRemoving(key);
        try {
            await removeMutation.mutateAsync({ kind: entry.kind, value: entry.value });
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to remove");
        } finally {
            setRemoving(null);
        }
    }

    if (loading) {
        return <div className={styles.loading}>Loading banlist...</div>;
    }

    return (
        <div className={styles.page}>
            <div className={styles.header}>
                <h1 className={styles.title}>Banned Giphy GIFs &amp; Channels</h1>
            </div>

            <p className={styles.intro}>
                Paste a Giphy GIF URL (e.g. <code>https://giphy.com/gifs/&hellip;-ID</code>) to ban a single GIF, or a
                channel URL (e.g. <code>https://giphy.com/channel/Larperine</code>) to ban every GIF from that uploader.
                Banned entries are filtered out of the GIF picker and rejected when users paste them into posts, chat
                messages, comments, or any other content.
            </p>

            <div className={styles.addCard}>
                <label className={styles.fieldLabel}>
                    Giphy URL or ID
                    <Input
                        type="text"
                        value={input}
                        onChange={e => setInput(e.target.value)}
                        placeholder="https://giphy.com/gifs/... or https://giphy.com/channel/..."
                        fullWidth
                    />
                </label>
                <label className={styles.fieldLabel}>
                    Reason (optional)
                    <Input
                        type="text"
                        value={reason}
                        onChange={e => setReason(e.target.value)}
                        placeholder="Why is this being banned?"
                        fullWidth
                    />
                </label>
                <div className={styles.formActions}>
                    <Button variant="primary" onClick={handleAdd} disabled={saving || !input.trim()}>
                        {saving ? "Adding..." : "Add to banlist"}
                    </Button>
                </div>
            </div>

            {error && <div className={styles.error}>{error}</div>}

            {entries.length === 0 ? (
                <div className={styles.empty}>Nothing banned yet.</div>
            ) : (
                <table className={styles.table}>
                    <thead>
                        <tr>
                            <th>Kind</th>
                            <th>Value</th>
                            <th>Reason</th>
                            <th>Added</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        {entries.map(e => {
                            const key = `${e.kind}:${e.value}`;
                            return (
                                <tr key={key}>
                                    <td>
                                        <span className={e.kind === "user" ? styles.badgeUser : styles.badgeGif}>
                                            {e.kind === "user" ? "Channel" : "GIF"}
                                        </span>
                                    </td>
                                    <td className={styles.mono}>{e.value}</td>
                                    <td className={styles.reasonCell}>{e.reason || "\u2014"}</td>
                                    <td className={styles.date}>{formatDate(e.created_at)}</td>
                                    <td className={styles.actions}>
                                        <Button
                                            variant="danger"
                                            size="small"
                                            onClick={() => handleRemove(e)}
                                            disabled={removing === key}
                                        >
                                            {removing === key ? "..." : "Remove"}
                                        </Button>
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            )}
        </div>
    );
}
