import { useEffect, useRef, useState } from "react";
import type { ChatRoom, User } from "../../../types/api";
import { createGroupRoom, getMutualFollowers, searchUsers } from "../../../api/endpoints";
import { Modal } from "../../Modal/Modal";
import { Input } from "../../Input/Input";
import { Button } from "../../Button/Button";
import { ToggleSwitch } from "../../ToggleSwitch/ToggleSwitch";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import styles from "./CreateRoomModal.module.css";

interface CreateRoomModalProps {
    isOpen: boolean;
    onClose: () => void;
    onCreated: (room: ChatRoom) => void;
}

export function CreateRoomModal({ isOpen, onClose, onCreated }: CreateRoomModalProps) {
    const [name, setName] = useState("");
    const [description, setDescription] = useState("");
    const [isPublic, setIsPublic] = useState(true);
    const [isRP, setIsRP] = useState(false);
    const [tagInput, setTagInput] = useState("");
    const [tags, setTags] = useState<string[]>([]);
    const [search, setSearch] = useState("");
    const [results, setResults] = useState<User[]>([]);
    const [mutuals, setMutuals] = useState<User[]>([]);
    const [selected, setSelected] = useState<User[]>([]);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

    useEffect(() => {
        if (!isOpen) {
            return;
        }
        getMutualFollowers()
            .then(setMutuals)
            .catch(() => setMutuals([]));
    }, [isOpen]);

    useEffect(() => {
        if (!isOpen) {
            setName("");
            setDescription("");
            setIsPublic(true);
            setIsRP(false);
            setTagInput("");
            setTags([]);
            setSearch("");
            setResults([]);
            setSelected([]);
            setError("");
        }
    }, [isOpen]);

    function normalizeTag(raw: string): string {
        return raw
            .toLowerCase()
            .trim()
            .replace(/\s+/g, "-")
            .replace(/[^a-z0-9-]+/g, "")
            .replace(/^-+|-+$/g, "")
            .slice(0, 30);
    }

    function commitTagInput() {
        if (!tagInput) {
            return;
        }
        const parts = tagInput.split(",").map(normalizeTag).filter(Boolean);
        if (parts.length === 0) {
            setTagInput("");
            return;
        }
        setTags(prev => {
            const next = [...prev];
            for (const t of parts) {
                if (!next.includes(t) && next.length < 10) {
                    next.push(t);
                }
            }
            return next;
        });
        setTagInput("");
    }

    function removeTag(t: string) {
        setTags(prev => prev.filter(x => x !== t));
    }

    function handleTagKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
        if (e.key === "Enter" || e.key === ",") {
            e.preventDefault();
            commitTagInput();
            return;
        }
        if (e.key === "Backspace" && tagInput === "" && tags.length > 0) {
            e.preventDefault();
            setTags(prev => prev.slice(0, -1));
        }
    }

    useEffect(() => {
        clearTimeout(debounceRef.current);
        if (!search.trim()) {
            setResults([]);
            return;
        }
        debounceRef.current = setTimeout(() => {
            searchUsers(search)
                .then(setResults)
                .catch(() => setResults([]));
        }, 200);
        return () => clearTimeout(debounceRef.current);
    }, [search]);

    function toggleSelected(u: User) {
        setSelected(prev => {
            if (prev.some(p => p.id === u.id)) {
                return prev.filter(p => p.id !== u.id);
            }
            return [...prev, u];
        });
    }

    async function handleSubmit() {
        if (!name.trim() || submitting) {
            return;
        }
        const finalTags = [...tags];
        const trailing = normalizeTag(tagInput);
        if (trailing && !finalTags.includes(trailing) && finalTags.length < 10) {
            finalTags.push(trailing);
        }
        setSubmitting(true);
        setError("");
        try {
            const room = await createGroupRoom({
                name: name.trim(),
                description: description.trim(),
                is_public: isPublic,
                is_rp: isRP,
                tags: finalTags,
                member_ids: selected.map(u => u.id),
            });
            onCreated(room);
            onClose();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to create room");
        } finally {
            setSubmitting(false);
        }
    }

    const candidates = search.trim() ? results : mutuals;

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Create Chat Room">
            <div className={styles.body}>
                {error && <div className={styles.error}>{error}</div>}
                <div className={styles.field}>
                    <label className={styles.label}>Name</label>
                    <Input
                        fullWidth
                        type="text"
                        value={name}
                        onChange={e => setName(e.target.value)}
                        placeholder="e.g. Higurashi book club"
                        maxLength={80}
                    />
                </div>
                <div className={styles.field}>
                    <label className={styles.label}>Description (optional)</label>
                    <Input
                        fullWidth
                        type="text"
                        value={description}
                        onChange={e => setDescription(e.target.value)}
                        placeholder="What's the room about?"
                        maxLength={500}
                    />
                </div>
                <ToggleSwitch
                    enabled={isPublic}
                    onChange={setIsPublic}
                    label="Public"
                    description="Public rooms appear in Browse and anyone can join. Private rooms are invite-only."
                />
                <ToggleSwitch
                    enabled={isRP}
                    onChange={setIsRP}
                    label="Roleplay (RP)"
                    description="Mark this as a roleplay room. Future RP-specific features will be enabled for these."
                />

                <div className={styles.field}>
                    <label className={styles.label}>Tags (optional)</label>
                    {tags.length > 0 && (
                        <div className={styles.tagBar}>
                            {tags.map(t => (
                                <button key={t} className={styles.tagChip} onClick={() => removeTag(t)}>
                                    #{t} ✕
                                </button>
                            ))}
                        </div>
                    )}
                    <Input
                        fullWidth
                        type="text"
                        placeholder="Type a tag and press Enter or comma (max 10)"
                        value={tagInput}
                        onChange={e => setTagInput(e.target.value)}
                        onKeyDown={handleTagKeyDown}
                        onBlur={commitTagInput}
                        disabled={tags.length >= 10}
                    />
                </div>

                {selected.length > 0 && (
                    <div className={styles.selectedBar}>
                        <span className={styles.selectedLabel}>Inviting:</span>
                        {selected.map(u => (
                            <button key={u.id} className={styles.selectedChip} onClick={() => toggleSelected(u)}>
                                {u.display_name} ✕
                            </button>
                        ))}
                    </div>
                )}

                <div className={styles.field}>
                    <label className={styles.label}>Invite users (optional)</label>
                    <Input
                        fullWidth
                        type="text"
                        placeholder="Search users..."
                        value={search}
                        onChange={e => setSearch(e.target.value)}
                    />
                </div>

                <div className={styles.userList}>
                    {!search.trim() && mutuals.length > 0 && (
                        <div className={styles.mutualsLabel}>Mutual followers</div>
                    )}
                    {candidates.length === 0 && search.trim() && <div className={styles.empty}>No users found</div>}
                    {candidates.map(u => {
                        const isSelected = selected.some(s => s.id === u.id);
                        return (
                            <button
                                key={u.id}
                                className={`${styles.userOption}${isSelected ? ` ${styles.userOptionSelected}` : ""}`}
                                onClick={() => toggleSelected(u)}
                            >
                                <ProfileLink user={u} size="small" clickable={false} />
                                <span className={styles.checkmark}>{isSelected ? "✓" : ""}</span>
                            </button>
                        );
                    })}
                </div>

                <div className={styles.actions}>
                    <Button variant="ghost" size="small" onClick={onClose}>
                        Cancel
                    </Button>
                    <Button variant="primary" size="small" onClick={handleSubmit} disabled={submitting || !name.trim()}>
                        {submitting ? "Creating..." : "Create Room"}
                    </Button>
                </div>
            </div>
        </Modal>
    );
}
