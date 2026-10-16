import type { ChatMessage, ChatRoomMember, PostMedia, User } from "../../types/api";

const NONCE_BYTES = 16;

export type DeliveryState = "sending" | "failed";

export interface PendingReply {
    id: string;
    senderName: string;
    bodyPreview: string;
}

export interface OutgoingAttachment {
    file: File;
    spoiler: boolean;
}

export interface OutgoingChatMessage {
    body: string;
    replyTo: PendingReply | null;
    attachments: OutgoingAttachment[];
}

export interface PendingAttachment extends OutgoingAttachment {
    previewUrl: string;
}

export type SendThread = { kind: "room"; roomId: string } | { kind: "draft"; recipientId: string };

export interface PendingSend {
    nonce: string;
    thread: SendThread;
    body: string;
    replyTo: PendingReply | null;
    attachments: PendingAttachment[];
    createdAt: string;
    delivery: DeliveryState;
    error: string | null;
}

export interface NewPendingSend {
    nonce: string;
    thread: SendThread;
    outgoing: OutgoingChatMessage;
    previewUrls: string[];
    createdAt: string;
}

export type ThreadEntry = ChatMessage | PendingSend;

export type SenderIdentity = Pick<ChatRoomMember, "nickname" | "member_avatar_url">;

export function createClientNonce(): string {
    const bytes = new Uint8Array(NONCE_BYTES);
    crypto.getRandomValues(bytes);

    return Array.from(bytes, byte => byte.toString(16).padStart(2, "0")).join("");
}

export function isPendingSend(entry: ThreadEntry): entry is PendingSend {
    return "delivery" in entry;
}

export function sameThread(a: SendThread, b: SendThread): boolean {
    if (a.kind === "room" && b.kind === "room") {
        return a.roomId === b.roomId;
    }
    if (a.kind === "draft" && b.kind === "draft") {
        return a.recipientId === b.recipientId;
    }

    return false;
}

export function createPendingSend(input: NewPendingSend): PendingSend {
    const { nonce, thread, outgoing, previewUrls, createdAt } = input;

    return {
        nonce,
        thread,
        body: outgoing.body,
        replyTo: outgoing.replyTo,
        attachments: outgoing.attachments.map((attachment, index) => ({
            file: attachment.file,
            spoiler: attachment.spoiler,
            previewUrl: previewUrls[index] ?? "",
        })),
        createdAt,
        delivery: "sending",
        error: null,
    };
}

export function addPendingSend(pending: PendingSend[], entry: PendingSend): PendingSend[] {
    return [...pending, entry];
}

export function findPendingSend(pending: PendingSend[], nonce: string): PendingSend | null {
    return pending.find(entry => entry.nonce === nonce) ?? null;
}

export function markPendingFailed(pending: PendingSend[], nonce: string, error: string): PendingSend[] {
    if (!pending.some(entry => entry.nonce === nonce)) {
        return pending;
    }

    return pending.map((entry): PendingSend =>
        entry.nonce === nonce ? { ...entry, delivery: "failed", error } : entry,
    );
}

export function markPendingSending(pending: PendingSend[], nonce: string): PendingSend[] {
    if (!pending.some(entry => entry.nonce === nonce)) {
        return pending;
    }

    return pending.map((entry): PendingSend =>
        entry.nonce === nonce ? { ...entry, delivery: "sending", error: null } : entry,
    );
}

export function removePendingSend(pending: PendingSend[], nonce: string): PendingSend[] {
    if (!pending.some(entry => entry.nonce === nonce)) {
        return pending;
    }

    return pending.filter(entry => entry.nonce !== nonce);
}

export function deliveredNonce(message: ChatMessage, viewerId: string | undefined): string | null {
    if (!message.client_nonce || message.sender.id !== viewerId) {
        return null;
    }

    return message.client_nonce;
}

export function pendingInThread(pending: PendingSend[], thread: SendThread | null): PendingSend[] {
    if (thread === null || pending.length === 0) {
        return [];
    }

    return pending.filter(entry => sameThread(entry.thread, thread));
}

export function failSendingInThread(pending: PendingSend[], thread: SendThread, error: string): PendingSend[] {
    if (!pending.some(entry => entry.delivery === "sending" && sameThread(entry.thread, thread))) {
        return pending;
    }

    return pending.map((entry): PendingSend =>
        entry.delivery === "sending" && sameThread(entry.thread, thread)
            ? { ...entry, delivery: "failed", error }
            : entry,
    );
}

export function moveThread(pending: PendingSend[], from: SendThread, to: SendThread): PendingSend[] {
    if (!pending.some(entry => sameThread(entry.thread, from))) {
        return pending;
    }

    return pending.map(entry => (sameThread(entry.thread, from) ? { ...entry, thread: to } : entry));
}

export function mergeThread(
    messages: ChatMessage[],
    pending: PendingSend[],
    viewerId: string | undefined,
): ThreadEntry[] {
    if (pending.length === 0) {
        return messages;
    }

    const delivered = new Set<string>();
    for (const message of messages) {
        const nonce = deliveredNonce(message, viewerId);
        if (nonce) {
            delivered.add(nonce);
        }
    }

    const waiting = pending.filter(entry => !delivered.has(entry.nonce));
    if (waiting.length === 0) {
        return messages;
    }

    return [...messages, ...waiting];
}

export function threadEntryKey(entry: ThreadEntry): string {
    if (isPendingSend(entry)) {
        return entry.nonce;
    }

    return entry.client_nonce || entry.id;
}

function mediaTypeOf(file: File): PostMedia["media_type"] {
    if (file.type.startsWith("video/")) {
        return "video";
    }
    if (file.type.startsWith("audio/")) {
        return "audio";
    }

    return "image";
}

export function pendingPreview(pending: PendingSend, viewer: User, identity: SenderIdentity | null): ChatMessage {
    return {
        id: pending.nonce,
        room_id: pending.thread.kind === "room" ? pending.thread.roomId : "",
        sender: viewer,
        sender_nickname: identity?.nickname || undefined,
        sender_member_avatar_url: identity?.member_avatar_url || undefined,
        body: pending.body,
        is_system: false,
        created_at: pending.createdAt,
        media: pending.attachments.map((attachment, index) => ({
            id: index,
            media_url: attachment.previewUrl,
            media_type: mediaTypeOf(attachment.file),
            filename: attachment.file.name,
            sort_order: index,
            is_spoiler: attachment.spoiler,
        })),
        reply_to: pending.replyTo
            ? {
                  id: pending.replyTo.id,
                  sender_id: "",
                  sender_name: pending.replyTo.senderName,
                  body_preview: pending.replyTo.bodyPreview,
              }
            : undefined,
        pinned: false,
        reactions: [],
        client_nonce: pending.nonce,
    };
}
