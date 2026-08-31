export type MessageActionId = "react" | "reply" | "pin" | "edit" | "delete";

export interface MessageAction {
    id: MessageActionId;
    label: string;
    icon: string;
    destructive: boolean;
    opensPicker: boolean;
}

export interface MessageActionContext {
    pinned: boolean;
    isOwn: boolean;
    editing: boolean;
    senderIsStaff: boolean;
    canToggleReaction: boolean;
    canPin: boolean;
    canModerate: boolean;
    canEdit: boolean;
    hasReply: boolean;
    hasPinToggle: boolean;
    hasEdit: boolean;
    hasDelete: boolean;
}

const REACT_ICON = "\u{1F642} +";
const REPLY_ICON = "↩";
const PIN_ICON = "\u{1F4CC}";
const UNPIN_ICON = "\u{1F4CC}✕";
const EDIT_ICON = "✎";
const DELETE_ICON = "\u{1F5D1}";

export function messageActions(context: MessageActionContext): MessageAction[] {
    const actions: MessageAction[] = [];

    if (context.canToggleReaction) {
        actions.push({ id: "react", label: "React", icon: REACT_ICON, destructive: false, opensPicker: true });
    }

    if (context.hasReply) {
        actions.push({ id: "reply", label: "Reply", icon: REPLY_ICON, destructive: false, opensPicker: false });
    }

    if (context.canPin && context.hasPinToggle) {
        actions.push({
            id: "pin",
            label: context.pinned ? "Unpin message" : "Pin message",
            icon: context.pinned ? UNPIN_ICON : PIN_ICON,
            destructive: false,
            opensPicker: false,
        });
    }

    if (context.isOwn && context.canEdit && context.hasEdit && !context.editing) {
        actions.push({ id: "edit", label: "Edit message", icon: EDIT_ICON, destructive: false, opensPicker: false });
    }

    if ((context.isOwn || (context.canModerate && !context.senderIsStaff)) && context.hasDelete) {
        actions.push({
            id: "delete",
            label: "Delete message",
            icon: DELETE_ICON,
            destructive: true,
            opensPicker: false,
        });
    }

    return actions;
}
