import { startTransition, useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
    addPendingSend,
    createClientNonce,
    createPendingSend,
    failSendingInThread,
    findPendingSend,
    markPendingFailed,
    markPendingSending,
    moveThread,
    pendingInThread,
    removePendingSend,
    type OutgoingChatMessage,
    type PendingSend,
    type SendThread,
} from "../../domain/chat/optimisticSend";
import type { ChatMessage, ChatRoom } from "../../types/api";
import { describeChatSendFailure, useSendChatMessage, useSendFirstDMMessage } from "../mutations/chat";

export interface ChatSender {
    roomId: string;
    adoptedFrom?: string;
    send: (outgoing: OutgoingChatMessage) => void;
}

export interface DraftSender {
    recipientId: string;
    send: (outgoing: OutgoingChatMessage) => void;
}

export interface OptimisticSendsOptions {
    roomId: string | undefined;
    draftRecipientId?: string;
    onSending: () => void;
    onDelivered: (message: ChatMessage) => void;
    onAdopted?: (message: ChatMessage, room: ChatRoom) => void;
}

export interface OptimisticSends {
    pending: PendingSend[];
    sender: ChatSender | null;
    draftSender: DraftSender | null;
    retry: (nonce: string) => void;
    discard: (nonce: string) => void;
    reconcile: (nonce: string) => void;
}

const NO_ADOPTIONS: ReadonlyMap<string, string> = new Map();

function revokePreviews(entry: PendingSend): void {
    for (const attachment of entry.attachments) {
        URL.revokeObjectURL(attachment.previewUrl);
    }
}

export function useOptimisticSends(options: OptimisticSendsOptions): OptimisticSends {
    const { roomId, draftRecipientId, onSending, onDelivered, onAdopted } = options;

    const [all, setAll] = useState<PendingSend[]>([]);
    const [adoptions, setAdoptions] = useState<ReadonlyMap<string, string>>(NO_ADOPTIONS);
    const allRef = useRef<PendingSend[]>([]);
    const openingRef = useRef(new Set<string>());
    const onSendingRef = useRef(onSending);
    const onDeliveredRef = useRef(onDelivered);
    const onAdoptedRef = useRef(onAdopted);
    const sendMessage = useSendChatMessage().mutateAsync;
    const sendFirstMessage = useSendFirstDMMessage().mutateAsync;

    useEffect(() => {
        onSendingRef.current = onSending;
    }, [onSending]);

    useEffect(() => {
        onDeliveredRef.current = onDelivered;
    }, [onDelivered]);

    useEffect(() => {
        onAdoptedRef.current = onAdopted;
    }, [onAdopted]);

    useEffect(() => {
        return () => {
            for (const entry of allRef.current) {
                revokePreviews(entry);
            }
        };
    }, []);

    const update = useCallback((change: (current: PendingSend[]) => PendingSend[]) => {
        const next = change(allRef.current);
        if (next === allRef.current) {
            return;
        }

        allRef.current = next;
        setAll(next);
    }, []);

    const release = useCallback(
        (nonce: string) => {
            const entry = findPendingSend(allRef.current, nonce);
            if (!entry) {
                return;
            }

            revokePreviews(entry);
            update(current => removePendingSend(current, nonce));
        },
        [update],
    );

    const deliverToRoom = useCallback(
        (entry: PendingSend, targetRoomId: string) => {
            sendMessage({
                roomId: targetRoomId,
                body: entry.body,
                reply_to_id: entry.replyTo?.id,
                files: entry.attachments.map(attachment => attachment.file),
                spoilers: entry.attachments.map(attachment => attachment.spoiler),
                client_nonce: entry.nonce,
            }).then(
                message => {
                    release(entry.nonce);
                    onDeliveredRef.current(message);
                },
                (err: unknown) => {
                    update(current => markPendingFailed(current, entry.nonce, describeChatSendFailure(err)));
                },
            );
        },
        [sendMessage, release, update],
    );

    const deliverFirstMessage = useCallback(
        (entry: PendingSend, recipientId: string) => {
            const draft: SendThread = { kind: "draft", recipientId };
            openingRef.current.add(recipientId);

            sendFirstMessage({
                recipientId,
                body: entry.body,
                files: entry.attachments.map(attachment => attachment.file),
                spoilers: entry.attachments.map(attachment => attachment.spoiler),
                client_nonce: entry.nonce,
            }).then(
                created => {
                    openingRef.current.delete(recipientId);

                    const room: SendThread = { kind: "room", roomId: created.room.id };
                    const held = pendingInThread(allRef.current, draft).filter(
                        waiting => waiting.nonce !== entry.nonce && waiting.delivery === "sending",
                    );

                    startTransition(() => {
                        release(entry.nonce);
                        update(current => moveThread(current, draft, room));
                        setAdoptions(current => new Map(current).set(created.room.id, recipientId));
                        onAdoptedRef.current?.(created.message, created.room);
                    });

                    for (const waiting of held) {
                        deliverToRoom(waiting, created.room.id);
                    }
                },
                (err: unknown) => {
                    openingRef.current.delete(recipientId);
                    update(current => failSendingInThread(current, draft, describeChatSendFailure(err)));
                },
            );
        },
        [sendFirstMessage, release, update, deliverToRoom],
    );

    const dispatch = useCallback(
        (entry: PendingSend) => {
            if (entry.thread.kind === "room") {
                deliverToRoom(entry, entry.thread.roomId);
                return;
            }

            if (openingRef.current.has(entry.thread.recipientId)) {
                return;
            }

            deliverFirstMessage(entry, entry.thread.recipientId);
        },
        [deliverToRoom, deliverFirstMessage],
    );

    const send = useCallback(
        (thread: SendThread, outgoing: OutgoingChatMessage) => {
            const entry = createPendingSend({
                nonce: createClientNonce(),
                thread,
                outgoing,
                previewUrls: outgoing.attachments.map(attachment => URL.createObjectURL(attachment.file)),
                createdAt: new Date().toISOString(),
            });

            update(current => addPendingSend(current, entry));
            onSendingRef.current();
            dispatch(entry);
        },
        [update, dispatch],
    );

    const retry = useCallback(
        (nonce: string) => {
            const entry = findPendingSend(allRef.current, nonce);
            if (!entry || entry.delivery !== "failed") {
                return;
            }

            update(current => markPendingSending(current, nonce));
            dispatch(entry);
        },
        [update, dispatch],
    );

    const openThread = useMemo((): SendThread | null => {
        if (roomId) {
            return { kind: "room", roomId };
        }
        if (draftRecipientId) {
            return { kind: "draft", recipientId: draftRecipientId };
        }

        return null;
    }, [roomId, draftRecipientId]);

    const pending = useMemo(() => pendingInThread(all, openThread), [all, openThread]);

    const sender = useMemo((): ChatSender | null => {
        if (openThread?.kind !== "room") {
            return null;
        }

        const thread = openThread;

        return {
            roomId: thread.roomId,
            adoptedFrom: adoptions.get(thread.roomId),
            send: outgoing => send(thread, outgoing),
        };
    }, [openThread, adoptions, send]);

    const draftSender = useMemo((): DraftSender | null => {
        if (openThread?.kind !== "draft") {
            return null;
        }

        const thread = openThread;

        return {
            recipientId: thread.recipientId,
            send: outgoing => send(thread, outgoing),
        };
    }, [openThread, send]);

    return { pending, sender, draftSender, retry, discard: release, reconcile: release };
}
