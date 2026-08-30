import { type Dispatch, type SetStateAction, useState } from "react";
import { applyLocalMemberChangeToMembers, applyLocalMemberChangeToMessages } from "../../domain/chat/messagePatches";
import type { ChatMessage, ChatRoomMember } from "../../types/api";
import { parseServerDate } from "../../utils/time";
import {
    useBanChatRoomMember,
    useClearChatRoomMemberTimeout,
    useKickChatRoomMember,
    useSetChatRoomMemberNickname,
    useSetChatRoomMemberTimeout,
    useUnlockChatRoomMemberNickname,
} from "../mutations/chat";

const KICK_CONFIRM = "Kick this member from the room?";
const BAN_PROMPT = "Ban this member from the room? They will not be able to rejoin or see the room. Optional reason:";
const BAN_DONE = "Member banned from the room.";
const NICKNAME_FAILED = "Failed to set nickname";
const UNLOCK_FAILED = "Failed to unlock nickname";
const TIMEOUT_AMOUNT_INVALID = "Enter a whole number greater than zero";
const TIMEOUT_FAILED = "Failed to set timeout";
const CLEAR_TIMEOUT_FAILED = "Failed to clear timeout";
const KICK_FAILED = "Failed to kick";
const BAN_FAILED = "Failed to ban";

function failureText(err: unknown, fallback: string): string {
    return err instanceof Error ? err.message : fallback;
}

export function formatTimeoutUntil(value?: string): string {
    if (!value) {
        return "";
    }

    const parsed = parseServerDate(value);
    if (!parsed) {
        return value;
    }

    return parsed.toLocaleString();
}

export interface RoomModerationOptions {
    roomId: string | undefined;
    setMembers: Dispatch<SetStateAction<ChatRoomMember[]>>;
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>;
    notify: (message: string) => void;
}

export function useRoomModeration(options: RoomModerationOptions) {
    const { roomId, setMembers, setMessages, notify } = options;

    const [busy, setBusy] = useState<string | null>(null);
    const [openMemberMenu, setOpenMemberMenu] = useState<string | null>(null);
    const [nicknameDialogTarget, setNicknameDialogTarget] = useState<ChatRoomMember | null>(null);
    const [nicknameDialogValue, setNicknameDialogValue] = useState("");
    const [nicknameDialogError, setNicknameDialogError] = useState("");
    const [nicknameDialogSaving, setNicknameDialogSaving] = useState(false);
    const [timeoutDialogTarget, setTimeoutDialogTarget] = useState<ChatRoomMember | null>(null);
    const [timeoutDialogAmount, setTimeoutDialogAmount] = useState("1");
    const [timeoutDialogUnit, setTimeoutDialogUnit] = useState("hours");
    const [timeoutDialogError, setTimeoutDialogError] = useState("");
    const [timeoutDialogSaving, setTimeoutDialogSaving] = useState(false);

    const kickMutation = useKickChatRoomMember(roomId ?? "");
    const banMutation = useBanChatRoomMember(roomId ?? "");
    const setNicknameMutation = useSetChatRoomMemberNickname(roomId ?? "");
    const unlockNicknameMutation = useUnlockChatRoomMemberNickname(roomId ?? "");
    const setTimeoutMutation = useSetChatRoomMemberTimeout(roomId ?? "");
    const clearTimeoutMutation = useClearChatRoomMemberTimeout(roomId ?? "");

    function openNicknameDialog(member: ChatRoomMember) {
        setNicknameDialogTarget(member);
        setNicknameDialogValue(member.nickname ?? "");
        setNicknameDialogError("");
        setOpenMemberMenu(null);
    }

    function openTimeoutDialog(member: ChatRoomMember) {
        setTimeoutDialogTarget(member);
        setTimeoutDialogAmount("1");
        setTimeoutDialogUnit("hours");
        setTimeoutDialogError("");
        setOpenMemberMenu(null);
    }

    async function handleModSetNickname() {
        if (!roomId || !nicknameDialogTarget) {
            return;
        }

        setNicknameDialogSaving(true);
        setNicknameDialogError("");
        try {
            const updated = await setNicknameMutation.mutateAsync({
                userId: nicknameDialogTarget.user.id,
                nickname: nicknameDialogValue.trim(),
            });
            setMembers(prev => applyLocalMemberChangeToMembers(prev, updated));
            setMessages(prev => applyLocalMemberChangeToMessages(prev, updated));
            setNicknameDialogTarget(null);
        } catch (err) {
            setNicknameDialogError(failureText(err, NICKNAME_FAILED));
        } finally {
            setNicknameDialogSaving(false);
        }
    }

    async function handleModUnlockNickname(targetId: string) {
        if (!roomId) {
            return;
        }

        setBusy(targetId);
        setOpenMemberMenu(null);
        try {
            const updated = await unlockNicknameMutation.mutateAsync(targetId);
            setMembers(prev => applyLocalMemberChangeToMembers(prev, updated));
            setMessages(prev => applyLocalMemberChangeToMessages(prev, updated));
        } catch (err) {
            notify(failureText(err, UNLOCK_FAILED));
        } finally {
            setBusy(null);
        }
    }

    async function handleSetTimeout() {
        if (!roomId || !timeoutDialogTarget) {
            return;
        }

        const amount = Number(timeoutDialogAmount);
        if (!Number.isInteger(amount) || amount <= 0) {
            setTimeoutDialogError(TIMEOUT_AMOUNT_INVALID);
            return;
        }

        setTimeoutDialogSaving(true);
        setTimeoutDialogError("");
        try {
            const updated = await setTimeoutMutation.mutateAsync({
                userId: timeoutDialogTarget.user.id,
                amount,
                unit: timeoutDialogUnit,
            });
            setMembers(prev => prev.map(m => (m.user.id === updated.user.id ? updated : m)));
            setTimeoutDialogTarget(null);
        } catch (err) {
            setTimeoutDialogError(failureText(err, TIMEOUT_FAILED));
        } finally {
            setTimeoutDialogSaving(false);
        }
    }

    async function handleClearTimeout(targetId: string) {
        if (!roomId) {
            return;
        }

        setBusy(`timeout:${targetId}`);
        setOpenMemberMenu(null);
        try {
            const updated = await clearTimeoutMutation.mutateAsync(targetId);
            setMembers(prev => prev.map(m => (m.user.id === updated.user.id ? updated : m)));
        } catch (err) {
            notify(failureText(err, CLEAR_TIMEOUT_FAILED));
        } finally {
            setBusy(null);
        }
    }

    async function handleKick(targetId: string) {
        if (!roomId || !window.confirm(KICK_CONFIRM)) {
            return;
        }

        setBusy(targetId);
        try {
            await kickMutation.mutateAsync(targetId);
            setMembers(prev => prev.filter(m => m.user.id !== targetId));
        } catch (err) {
            notify(failureText(err, KICK_FAILED));
        } finally {
            setBusy(null);
        }
    }

    async function handleBan(targetId: string) {
        if (!roomId) {
            return;
        }

        const banReason = window.prompt(BAN_PROMPT, "");
        if (banReason === null) {
            return;
        }

        setBusy(targetId);
        try {
            await banMutation.mutateAsync({ userId: targetId, reason: banReason });
            setMembers(prev => prev.filter(m => m.user.id !== targetId));
            notify(BAN_DONE);
        } catch (err) {
            notify(failureText(err, BAN_FAILED));
        } finally {
            setBusy(null);
        }
    }

    return {
        busy,
        setBusy,
        openMemberMenu,
        setOpenMemberMenu,
        nicknameDialogTarget,
        setNicknameDialogTarget,
        nicknameDialogValue,
        setNicknameDialogValue,
        nicknameDialogError,
        nicknameDialogSaving,
        timeoutDialogTarget,
        setTimeoutDialogTarget,
        timeoutDialogAmount,
        setTimeoutDialogAmount,
        timeoutDialogUnit,
        setTimeoutDialogUnit,
        timeoutDialogError,
        timeoutDialogSaving,
        formatTimeoutUntil,
        openNicknameDialog,
        openTimeoutDialog,
        handleModSetNickname,
        handleModUnlockNickname,
        handleSetTimeout,
        handleClearTimeout,
        handleKick,
        handleBan,
    };
}

export type RoomModeration = ReturnType<typeof useRoomModeration>;
