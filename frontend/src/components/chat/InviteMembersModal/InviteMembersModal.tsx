import { useRef, useState } from "react";
import type { User } from "../../../types/api";
import { useMutualFollowers, useSearchUsers } from "../../../api/queries/misc";
import { useInviteChatRoomMembers } from "../../../api/mutations/chat";
import { Modal } from "../../Modal/Modal";
import { Input } from "../../Input/Input";
import { Button } from "../../Button/Button";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import styles from "./InviteMembersModal.module.css";

interface InviteMembersModalProps {
    isOpen: boolean;
    roomId: string;
    existingMemberIds: Set<string>;
    onClose: () => void;
    onInvited?: (result: { invited_count: number; skipped_count: number }) => void;
}

export function InviteMembersModal({ isOpen, roomId, existingMemberIds, onClose, onInvited }: InviteMembersModalProps) {
    const [search, setSearch] = useState("");
    const [debouncedSearch, setDebouncedSearch] = useState("");
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    const [selected, setSelected] = useState<User[]>([]);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    const [openInstance, setOpenInstance] = useState(0);
    const [prevIsOpen, setPrevIsOpen] = useState(isOpen);
    if (isOpen !== prevIsOpen) {
        setPrevIsOpen(isOpen);
        if (isOpen) {
            setOpenInstance(n => n + 1);
        }
    }
    const [resetForOpenInstance, setResetForOpenInstance] = useState(0);
    if (resetForOpenInstance !== openInstance && !isOpen) {
        setResetForOpenInstance(openInstance);
        setSearch("");
        setDebouncedSearch("");
        setSelected([]);
        setError("");
    }

    function handleSearchChange(value: string) {
        setSearch(value);
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => setDebouncedSearch(value.trim()), 200);
    }

    const mutualsQuery = useMutualFollowers(isOpen);
    const mutuals = mutualsQuery.mutuals;

    const searchQuery = useSearchUsers(debouncedSearch, isOpen && !!debouncedSearch);
    const results = searchQuery.users;
    const inviteMutation = useInviteChatRoomMembers(roomId);

    function toggleSelected(u: User) {
        setSelected(prev => {
            if (prev.some(p => p.id === u.id)) {
                return prev.filter(p => p.id !== u.id);
            }
            return [...prev, u];
        });
    }

    async function handleSubmit() {
        if (selected.length === 0 || submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            const result = await inviteMutation.mutateAsync(selected.map(u => u.id));
            onInvited?.(result);
            onClose();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to invite members");
        } finally {
            setSubmitting(false);
        }
    }

    const rawCandidates = search.trim() ? results : mutuals;
    const candidates = rawCandidates.filter(u => !existingMemberIds.has(u.id));

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Invite Members">
            <div className={styles.body}>
                {error && <div className={styles.error}>{error}</div>}

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
                    <label className={styles.label}>Find users</label>
                    <Input
                        fullWidth
                        type="text"
                        placeholder="Search users..."
                        value={search}
                        onChange={e => handleSearchChange(e.target.value)}
                    />
                </div>

                <div className={styles.userList}>
                    {!search.trim() && candidates.length > 0 && (
                        <div className={styles.mutualsLabel}>Mutual followers</div>
                    )}
                    {candidates.length === 0 && (
                        <div className={styles.empty}>
                            {search.trim() ? "No users found" : "No mutual followers to invite"}
                        </div>
                    )}
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
                    <Button
                        variant="primary"
                        size="small"
                        onClick={handleSubmit}
                        disabled={submitting || selected.length === 0}
                    >
                        {submitting ? "Inviting..." : `Invite ${selected.length || ""}`.trim()}
                    </Button>
                </div>
            </div>
        </Modal>
    );
}
