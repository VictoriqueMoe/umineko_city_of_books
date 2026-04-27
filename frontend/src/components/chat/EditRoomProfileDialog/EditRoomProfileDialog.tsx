import { useRef, useState } from "react";
import { Button } from "../../Button/Button";
import { Modal } from "../../Modal/Modal";
import type { ChatRoomMember } from "../../../types/api";
import {
    useClearChatRoomAvatar,
    useUpdateChatRoomNickname,
    useUploadChatRoomAvatar,
} from "../../../api/mutations/chat";
import styles from "./EditRoomProfileDialog.module.css";

interface EditRoomProfileDialogProps {
    isOpen: boolean;
    roomId: string;
    currentMember: ChatRoomMember | null;
    onClose: () => void;
    onSaved: (member: ChatRoomMember) => void;
}

const NICKNAME_MAX = 32;

export function EditRoomProfileDialog({ isOpen, roomId, currentMember, onClose, onSaved }: EditRoomProfileDialogProps) {
    const [nickname, setNickname] = useState(currentMember?.nickname ?? "");
    const [avatarPreview, setAvatarPreview] = useState<string>(
        currentMember?.member_avatar_url || currentMember?.user.avatar_url || "",
    );
    const [pendingFile, setPendingFile] = useState<File | null>(null);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string>("");
    const fileInputRef = useRef<HTMLInputElement | null>(null);

    const updateNicknameMutation = useUpdateChatRoomNickname(roomId);
    const uploadAvatarMutation = useUploadChatRoomAvatar(roomId);
    const clearAvatarMutation = useClearChatRoomAvatar(roomId);

    function handleFileChange(event: React.ChangeEvent<HTMLInputElement>) {
        const file = event.target.files?.[0];
        if (!file) {
            return;
        }
        setPendingFile(file);
        const url = URL.createObjectURL(file);
        setAvatarPreview(url);
    }

    async function handleSave() {
        if (!currentMember) {
            return;
        }
        setSaving(true);
        setError("");
        try {
            let member = currentMember;
            const trimmed = nickname.trim();
            if (trimmed !== (currentMember.nickname ?? "")) {
                member = await updateNicknameMutation.mutateAsync(trimmed);
            }
            if (pendingFile) {
                member = await uploadAvatarMutation.mutateAsync(pendingFile);
            }
            onSaved(member);
            onClose();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to save profile");
        } finally {
            setSaving(false);
        }
    }

    async function handleClear() {
        if (!currentMember) {
            return;
        }
        setSaving(true);
        setError("");
        try {
            let member = await updateNicknameMutation.mutateAsync("");
            try {
                member = await clearAvatarMutation.mutateAsync();
            } catch {
                // avatar may already be absent
            }
            onSaved(member);
            onClose();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to clear profile");
        } finally {
            setSaving(false);
        }
    }

    if (!isOpen || !currentMember) {
        return null;
    }

    const remaining = NICKNAME_MAX - nickname.length;
    const locked = currentMember.nickname_locked;

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Edit profile in this room">
            <div className={styles.body}>
                {locked && (
                    <div className={styles.lockedNote}>
                        Your profile in this room has been locked by a moderator. Contact a moderator to unlock.
                    </div>
                )}
                <div className={styles.avatarRow}>
                    {avatarPreview ? (
                        <img src={avatarPreview} alt="" className={styles.avatarPreview} />
                    ) : (
                        <span className={styles.avatarPlaceholder}>
                            {(currentMember.nickname || currentMember.user.display_name)[0] ?? "?"}
                        </span>
                    )}
                    <div className={styles.avatarControls}>
                        <input
                            ref={fileInputRef}
                            type="file"
                            accept="image/*"
                            onChange={handleFileChange}
                            className={styles.hiddenInput}
                            disabled={locked}
                        />
                        <Button
                            variant="secondary"
                            size="small"
                            type="button"
                            onClick={() => fileInputRef.current?.click()}
                            disabled={locked}
                        >
                            Choose avatar
                        </Button>
                        {pendingFile && <span className={styles.hint}>Saved when you click Save.</span>}
                    </div>
                </div>

                <label className={styles.label} htmlFor="room-nickname">
                    Nickname
                </label>
                <input
                    id="room-nickname"
                    type="text"
                    className={styles.input}
                    maxLength={NICKNAME_MAX}
                    value={nickname}
                    onChange={e => setNickname(e.target.value)}
                    placeholder={currentMember.user.display_name}
                    disabled={locked}
                />
                <span className={styles.counter}>{remaining} characters remaining</span>

                {error && <div className={styles.error}>{error}</div>}

                <div className={styles.actions}>
                    <Button variant="ghost" size="small" onClick={onClose} disabled={saving}>
                        Cancel
                    </Button>
                    <Button variant="secondary" size="small" onClick={handleClear} disabled={saving || locked}>
                        Clear
                    </Button>
                    <Button variant="primary" size="small" onClick={handleSave} disabled={saving || locked}>
                        {saving ? "Saving..." : "Save"}
                    </Button>
                </div>
            </div>
        </Modal>
    );
}
