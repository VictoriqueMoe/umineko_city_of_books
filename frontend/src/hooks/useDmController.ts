import { useState } from "react";
import { typingNames as resolveTypingNames } from "../domain/chat/memberRoster";
import type { ChatMessage, ChatRoom } from "../types/api";
import { useDeleteChatRoom } from "./mutations/chat";
import { useChatThread } from "./useChatThread";
import { useDmRoster } from "./useDmRoster";

const MAX_DM_MESSAGES = 300;
const DELETE_CONFIRMATION = "Remove this conversation from your chat list?";
const DELETE_FAILED = "Failed to delete conversation";

export function useDmController() {
    const { thread, session, capabilities, toast } = useChatThread({ maxMessages: MAX_DM_MESSAGES });
    const roster = useDmRoster(thread.id);

    const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);

    const activeRoomId = thread.id ?? null;
    const activeRoom = thread.data ?? undefined;
    const mobileView: "list" | "room" = thread.id || roster.draftRecipient ? "room" : "list";

    const deleteChatRoomMutation = useDeleteChatRoom();

    function handleSentMessage(message: ChatMessage, room?: ChatRoom) {
        if (room) {
            session.history.seedMessages(room.id, [message]);
            roster.adoptRoom(room);
            session.toBottom({ force: true });
            return;
        }

        session.onSent(message);
        roster.noteSent(message);
    }

    async function handleDeleteChat() {
        if (!activeRoomId) {
            return;
        }

        if (!window.confirm(DELETE_CONFIRMATION)) {
            return;
        }

        try {
            await deleteChatRoomMutation.mutateAsync(activeRoomId);
            session.history.setMessages([]);
            roster.dropRoom(activeRoomId);
        } catch (err) {
            toast.show(err instanceof Error ? err.message : DELETE_FAILED);
        }
    }

    const typingNames = resolveTypingNames(session.typingUserIds, activeRoom?.members, session.viewer?.id);

    return {
        user: session.viewer,
        loading: roster.loading,
        mobileView,
        rooms: roster.rooms,
        activeRoomId,
        activeRoom,
        capabilities,
        draftRecipient: roster.draftRecipient,
        setDraftRecipient: roster.setDraftRecipient,
        messages: session.messages,
        hasMore: session.hasMore,
        loadingMore: session.loadingMore,
        loadUntilMessage: session.history.loadUntilMessage,
        messagesContainerRef: session.containerRef,
        messagesContentRef: session.contentRef,
        messagesEndRef: session.endRef,
        handleDmScroll: session.onScroll,
        scrollToBottom: session.toBottom,
        readReceipts: session.readReceipts,
        matchesViewerMention: session.matchesViewerMention,
        typingNames,
        voice: thread.voice,
        voiceEnabled: thread.voice.enabled,
        replyingTo: session.replyingTo,
        setReplyingTo: session.setReplyingTo,
        editingMessageId: session.editingMessageId,
        startEditing: session.startEditing,
        cancelEditing: session.cancelEditing,
        lightboxSrc,
        setLightboxSrc,
        showNewDm: roster.showNewDm,
        setShowNewDm: roster.setShowNewDm,
        dmSearch: roster.search,
        setDmSearch: roster.setSearch,
        dmResults: roster.results,
        dmMutuals: roster.mutuals,
        dmError: roster.error,
        dmCreating: roster.creating,
        toast: toast.message,
        showToast: toast.show,
        handleRoomSelect: roster.selectRoom,
        handleMobileBack: roster.backToList,
        handleSentMessage,
        handleSelectUser: roster.selectUser,
        handleDeleteMessage: session.deleteMessage,
        handleEditMessage: session.editMessage,
        handleReactionToggle: session.toggleReaction,
        handleEditLast: session.editLast,
        handleDeleteChat,
        handleToggleMute: thread.toggleMute,
        mutePending: thread.mutePending,
        notifyTyping: session.notifyTyping,
    };
}

export type DmController = ReturnType<typeof useDmController>;
