import { makeChatThread } from "./useChatThread.fixture";
import type { DmController } from "./useDmController";

function noop(): void {}

function never(): Promise<void> {
    return Promise.resolve();
}

export function makeDmController(overrides: Partial<DmController> = {}): DmController {
    const { thread, session, capabilities, toast } = makeChatThread();

    return {
        user: session.viewer,
        loading: false,
        mobileView: "list",
        rooms: [],
        activeRoomId: null,
        activeRoom: undefined,
        capabilities,
        draftRecipient: null,
        setDraftRecipient: noop,
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
        typingNames: [],
        voice: thread.voice,
        voiceEnabled: thread.voice.enabled,
        replyingTo: session.replyingTo,
        setReplyingTo: session.setReplyingTo,
        editingMessageId: session.editingMessageId,
        startEditing: session.startEditing,
        cancelEditing: session.cancelEditing,
        lightboxSrc: null,
        setLightboxSrc: noop,
        showNewDm: false,
        setShowNewDm: noop,
        dmSearch: "",
        setDmSearch: noop,
        dmResults: [],
        dmMutuals: [],
        dmError: "",
        dmCreating: false,
        toast: toast.message,
        showToast: toast.show,
        handleRoomSelect: noop,
        handleMobileBack: noop,
        handleSentMessage: session.onSent,
        handleSelectUser: never,
        handleDeleteMessage: session.deleteMessage,
        handleEditMessage: session.editMessage,
        handleReactionToggle: session.toggleReaction,
        handleEditLast: session.editLast,
        handleDeleteChat: never,
        handleToggleMute: thread.toggleMute,
        mutePending: thread.mutePending,
        notifyTyping: session.notifyTyping,
        ...overrides,
    };
}
