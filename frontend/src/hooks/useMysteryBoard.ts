import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";
import {
    type AttemptGroup,
    findWinningAttempt,
    globalClues,
    groupAttemptsByAuthor,
    type MysteryPermissions,
    mysteryPermissions,
    winningAuthorIds,
} from "../domain/mystery";
import { readCursorKey, unreadAuthorIds } from "../domain/mystery/unread";
import type { MysteryAttachment, MysteryAttempt, MysteryClue, MysteryDetail } from "../types/api";
import { errorMessage } from "../utils/errorMessage";
import { useAuth } from "./useAuth";
import { useResetOnChange } from "./useResetOnChange";
import { stageMedia, type StagedMedia } from "./useStagedMedia";
import { type CommentHandlers, useCommentHandlers } from "./useCommentHandlers";
import { useMystery } from "./queries/mystery";
import {
    useAddMysteryClue,
    useCloseMystery,
    useCreateMysteryAttempt,
    useDeleteMystery,
    useDeleteMysteryAttachment,
    useDeleteMysteryClue,
    useDeleteMysteryMedia,
    useSetMysteryGmAway,
    useSetMysteryPaused,
    useUpdateMysteryClue,
    useUploadMysteryAttachment,
    useUploadMysteryMedia,
} from "./mutations/mystery";
import { useThrottled } from "./useThrottled";

const REFRESH_THROTTLE_MS = 200;

const BOARD_EVENTS = [
    REALTIME_EVENTS.MYSTERY_SOLVED,
    REALTIME_EVENTS.MYSTERY_WINNER_ADDED,
    REALTIME_EVENTS.MYSTERY_ATTEMPT_CREATED,
    REALTIME_EVENTS.MYSTERY_CLUE_ADDED,
    REALTIME_EVENTS.MYSTERY_CLUE_UPDATED,
    REALTIME_EVENTS.MYSTERY_PAUSED,
    REALTIME_EVENTS.MYSTERY_GM_AWAY,
] as const;

const NO_PERMISSIONS: MysteryPermissions = {
    isAuthor: false,
    canEdit: false,
    canDelete: false,
    canSeeAsGameMaster: false,
};

const NO_ATTEMPTS: MysteryAttempt[] = [];

const NO_IDS: ReadonlySet<string> = new Set();

export type MysteryCommentHandlers = CommentHandlers<
    "create" | "update" | "delete" | "like" | "unlike" | "uploadMedia"
>;

export interface MysteryBoard {
    mystery: MysteryDetail | null;
    loading: boolean;
    permissions: MysteryPermissions;
    winningAttempt: MysteryAttempt | null;
    groupedAttempts: AttemptGroup[];
    winners: Set<string>;
    sharedClues: MysteryClue[];
    unreadPlayers: Set<string>;
    collapsedPlayers: Set<string>;
    togglePlayerCollapse: (authorId: string) => void;
    jumpToPlayer: (authorId: string) => void;
    attemptBody: string;
    setAttemptBody: (body: string) => void;
    submitting: boolean;
    submitAttempt: () => Promise<void>;
    newClueBody: string;
    setNewClueBody: (body: string) => void;
    addingClue: boolean;
    addGlobalClue: () => Promise<void>;
    addPrivateClue: (playerId: string, body: string) => Promise<void>;
    saveClue: (clueId: number, body: string) => Promise<void>;
    removeClue: (clueId: number) => Promise<void>;
    removeMystery: () => Promise<void>;
    togglePaused: () => Promise<void>;
    toggleGmAway: () => Promise<void>;
    closeMystery: () => Promise<void>;
    uploadingAttachment: boolean;
    attachmentError: string;
    uploadAttachment: (file: File | null | undefined) => Promise<void>;
    removeAttachment: (attachment: MysteryAttachment) => Promise<void>;
    pendingMedia: File[];
    pendingMediaSpoilers: boolean[];
    addPendingMedia: (files: File[]) => void;
    removePendingMedia: (index: number) => void;
    togglePendingMediaSpoiler: (index: number) => void;
    uploadingMedia: boolean;
    mediaError: string;
    setMediaError: (message: string) => void;
    uploadMedia: () => Promise<void>;
    removeMedia: (mediaId: number) => Promise<void>;
    comments: MysteryCommentHandlers;
}

function withoutMember(members: Set<string>, member: string): Set<string> {
    if (!members.has(member)) {
        return members;
    }

    const next = new Set(members);
    next.delete(member);

    return next;
}

function scrollTo(elementId: string): void {
    requestAnimationFrame(() => {
        const element = document.getElementById(elementId);
        if (element) {
            element.scrollIntoView({ behavior: "smooth", block: "center" });
        }
    });
}

export function useMysteryBoard(mysteryId: string): MysteryBoard {
    const navigate = useNavigate();
    const { user } = useAuth();
    const { mystery, loading, refresh } = useMystery(mysteryId);

    const createAttemptMutation = useCreateMysteryAttempt(mysteryId);
    const addClueMutation = useAddMysteryClue(mysteryId);
    const updateClueMutation = useUpdateMysteryClue(mysteryId);
    const deleteClueMutation = useDeleteMysteryClue(mysteryId);
    const deleteMysteryMutation = useDeleteMystery();
    const setPausedMutation = useSetMysteryPaused(mysteryId);
    const setGmAwayMutation = useSetMysteryGmAway(mysteryId);
    const closeMysteryMutation = useCloseMystery(mysteryId);
    const uploadAttachmentMutation = useUploadMysteryAttachment(mysteryId);
    const deleteAttachmentMutation = useDeleteMysteryAttachment(mysteryId);
    const uploadMediaMutation = useUploadMysteryMedia(mysteryId);
    const deleteMediaMutation = useDeleteMysteryMedia(mysteryId);

    const comments = useCommentHandlers("mystery", mysteryId, {
        enabled: ["create", "update", "delete", "like", "unlike", "uploadMedia"],
    });

    const [attemptBody, setAttemptBody] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [newClueBody, setNewClueBody] = useState("");
    const [addingClue, setAddingClue] = useState(false);
    const [collapsedPlayers, setCollapsedPlayers] = useState<Set<string>>(new Set());
    const [arrivedPlayers, setArrivedPlayers] = useState<Set<string>>(new Set());
    const [readPlayers, setReadPlayers] = useState<Set<string>>(new Set());
    const [uploadingAttachment, setUploadingAttachment] = useState(false);
    const [attachmentError, setAttachmentError] = useState("");
    const [pendingStaged, setPendingStaged] = useState<StagedMedia[]>([]);
    const pendingMedia = useMemo(() => pendingStaged.map(s => s.file), [pendingStaged]);
    const pendingMediaSpoilers = useMemo(() => pendingStaged.map(s => s.isSpoiler), [pendingStaged]);
    const [uploadingMedia, setUploadingMedia] = useState(false);
    const [mediaError, setMediaError] = useState("");

    useResetOnChange(mysteryId, () => {
        setAttemptBody("");
        setSubmitting(false);
        setNewClueBody("");
        setAddingClue(false);
        setCollapsedPlayers(new Set());
        setArrivedPlayers(new Set());
        setReadPlayers(new Set());
        setUploadingAttachment(false);
        setAttachmentError("");
        setPendingStaged([]);
        setUploadingMedia(false);
        setMediaError("");
    });

    const attempts = mystery?.attempts ?? NO_ATTEMPTS;

    const permissions = useMemo(() => (mystery ? mysteryPermissions(user, mystery) : NO_PERMISSIONS), [mystery, user]);

    const winningAttempt = useMemo(
        () => (mystery?.solved ? findWinningAttempt(attempts) : null),
        [attempts, mystery?.solved],
    );

    const groupedAttempts = useMemo(
        () => groupAttemptsByAuthor(attempts, { freeForAll: mystery?.free_for_all ?? false, viewerId: user?.id }),
        [attempts, mystery?.free_for_all, user?.id],
    );

    const winners = useMemo(() => winningAuthorIds(attempts), [attempts]);

    const sharedClues = useMemo(() => globalClues(mystery?.clues ?? []), [mystery?.clues]);

    const refreshBoard = useCallback(() => {
        refresh();
    }, [refresh]);

    const throttledRefresh = useThrottled(refreshBoard, REFRESH_THROTTLE_MS);

    useRealtimeEvent(BOARD_EVENTS, event => {
        if (event.data.mystery_id !== mysteryId) {
            return;
        }

        if (event.type === REALTIME_EVENTS.MYSTERY_SOLVED) {
            throttledRefresh();
            if (event.data.attempt_id) {
                scrollTo(`attempt-${event.data.attempt_id}`);
            }
            return;
        }

        if (event.type === REALTIME_EVENTS.MYSTERY_ATTEMPT_CREATED) {
            const authorId = event.data.author_id;
            if (authorId && authorId !== user?.id) {
                setArrivedPlayers(previous => new Set(previous).add(authorId));
                setReadPlayers(previous => withoutMember(previous, authorId));
            }
        }

        throttledRefresh();
    });

    const readCursor = useMemo(() => (mysteryId ? localStorage.getItem(readCursorKey(mysteryId)) : null), [mysteryId]);

    const carriedOverPlayers = useMemo(() => {
        if (!mystery || !permissions.canSeeAsGameMaster || mystery.solved) {
            return NO_IDS;
        }

        return unreadAuthorIds({ attempts, cursor: readCursor, viewerId: user?.id });
    }, [attempts, mystery, permissions.canSeeAsGameMaster, readCursor, user?.id]);

    const unreadPlayers = useMemo(() => {
        const unread = new Set(carriedOverPlayers);

        for (const authorId of arrivedPlayers) {
            unread.add(authorId);
        }
        for (const authorId of readPlayers) {
            unread.delete(authorId);
        }

        return unread;
    }, [arrivedPlayers, carriedOverPlayers, readPlayers]);

    const cursorSeededFor = useRef<string | null>(null);

    useEffect(() => {
        if (!mystery || !mysteryId || cursorSeededFor.current === mysteryId) {
            return;
        }
        cursorSeededFor.current = mysteryId;

        if (!permissions.canSeeAsGameMaster || mystery.solved || readCursor) {
            return;
        }

        localStorage.setItem(readCursorKey(mysteryId), new Date().toISOString());
    }, [mystery, mysteryId, permissions.canSeeAsGameMaster, readCursor]);

    useEffect(() => {
        if (!mysteryId) {
            return;
        }

        return () => {
            localStorage.setItem(readCursorKey(mysteryId), new Date().toISOString());
        };
    }, [mysteryId]);

    const togglePlayerCollapse = useCallback((authorId: string) => {
        setCollapsedPlayers(previous => {
            const next = new Set(previous);
            if (next.has(authorId)) {
                next.delete(authorId);
            } else {
                next.add(authorId);
            }
            return next;
        });
    }, []);

    const jumpToPlayer = useCallback((authorId: string) => {
        setArrivedPlayers(previous => withoutMember(previous, authorId));
        setReadPlayers(previous => new Set(previous).add(authorId));
        setCollapsedPlayers(previous => withoutMember(previous, authorId));

        requestAnimationFrame(() => {
            const element = document.getElementById(`player-group-${authorId}`);
            if (element) {
                element.scrollIntoView({ behavior: "smooth", block: "start" });
            }
        });
    }, []);

    const submitAttempt = useCallback(async () => {
        if (!attemptBody.trim() || submitting || !mysteryId) {
            return;
        }

        setSubmitting(true);
        try {
            await createAttemptMutation.mutateAsync({ body: attemptBody.trim() });
            setAttemptBody("");
        } catch {
        } finally {
            setSubmitting(false);
        }
    }, [attemptBody, createAttemptMutation, mysteryId, submitting]);

    const addGlobalClue = useCallback(async () => {
        if (!newClueBody.trim() || addingClue || !mysteryId) {
            return;
        }

        setAddingClue(true);
        try {
            await addClueMutation.mutateAsync({ body: newClueBody.trim(), truthType: "red" });
            setNewClueBody("");
        } catch {
        } finally {
            setAddingClue(false);
        }
    }, [addClueMutation, addingClue, mysteryId, newClueBody]);

    const addPrivateClue = useCallback(
        async (playerId: string, body: string) => {
            await addClueMutation.mutateAsync({ body, truthType: "red", playerId });
        },
        [addClueMutation],
    );

    const saveClue = useCallback(
        async (clueId: number, body: string) => {
            await updateClueMutation.mutateAsync({ clueId, body });
        },
        [updateClueMutation],
    );

    const removeClue = useCallback(
        async (clueId: number) => {
            if (!window.confirm("Delete this red truth? This cannot be undone.")) {
                return;
            }
            await deleteClueMutation.mutateAsync(clueId);
        },
        [deleteClueMutation],
    );

    const removeMystery = useCallback(async () => {
        if (!mystery || !window.confirm("Delete this mystery? This cannot be undone.")) {
            return;
        }

        await deleteMysteryMutation.mutateAsync(mystery.id);
        navigate("/mysteries");
    }, [deleteMysteryMutation, mystery, navigate]);

    const togglePaused = useCallback(async () => {
        if (!mystery) {
            return;
        }
        await setPausedMutation.mutateAsync(!mystery.paused);
    }, [mystery, setPausedMutation]);

    const toggleGmAway = useCallback(async () => {
        if (!mystery) {
            return;
        }
        await setGmAwayMutation.mutateAsync(!mystery.gm_away);
    }, [mystery, setGmAwayMutation]);

    const closeMystery = useCallback(async () => {
        if (
            !window.confirm(
                "Mark this mystery as permanently solved? All attempts will be revealed and the post-mystery discussion will open. This cannot be undone.",
            )
        ) {
            return;
        }

        await closeMysteryMutation.mutateAsync();
    }, [closeMysteryMutation]);

    const uploadAttachment = useCallback(
        async (file: File | null | undefined) => {
            if (!file || uploadingAttachment || !mysteryId) {
                return;
            }

            setUploadingAttachment(true);
            setAttachmentError("");
            try {
                await uploadAttachmentMutation.mutateAsync(file);
            } catch (e) {
                setAttachmentError(errorMessage(e, "Failed to upload attachment"));
            } finally {
                setUploadingAttachment(false);
            }
        },
        [mysteryId, uploadAttachmentMutation, uploadingAttachment],
    );

    const removeAttachment = useCallback(
        async (attachment: MysteryAttachment) => {
            if (!window.confirm(`Delete attachment "${attachment.file_name}"?`)) {
                return;
            }

            try {
                await deleteAttachmentMutation.mutateAsync(attachment.id);
            } catch {}
        },
        [deleteAttachmentMutation],
    );

    const addPendingMedia = useCallback((files: File[]) => {
        setPendingStaged(previous => [...previous, ...stageMedia(files)]);
    }, []);

    const togglePendingMediaSpoiler = useCallback((index: number) => {
        setPendingStaged(previous =>
            previous.map((item, position) => (position === index ? { ...item, isSpoiler: !item.isSpoiler } : item)),
        );
    }, []);

    const removePendingMedia = useCallback((index: number) => {
        setPendingStaged(previous => previous.filter((_, position) => position !== index));
    }, []);

    const uploadMedia = useCallback(async () => {
        if (pendingStaged.length === 0 || uploadingMedia) {
            return;
        }

        setUploadingMedia(true);
        setMediaError("");
        try {
            for (const item of pendingStaged) {
                await uploadMediaMutation.mutateAsync({ file: item.file, isSpoiler: item.isSpoiler });
            }
            setPendingStaged([]);
        } catch (e) {
            setMediaError(errorMessage(e, "Failed to upload image"));
        } finally {
            setUploadingMedia(false);
        }
    }, [pendingStaged, uploadMediaMutation, uploadingMedia]);

    const removeMedia = useCallback(
        async (mediaId: number) => {
            if (!window.confirm("Remove this image?")) {
                return;
            }

            try {
                await deleteMediaMutation.mutateAsync(mediaId);
            } catch {}
        },
        [deleteMediaMutation],
    );

    return {
        mystery,
        loading,
        permissions,
        winningAttempt,
        groupedAttempts,
        winners,
        sharedClues,
        unreadPlayers,
        collapsedPlayers,
        togglePlayerCollapse,
        jumpToPlayer,
        attemptBody,
        setAttemptBody,
        submitting,
        submitAttempt,
        newClueBody,
        setNewClueBody,
        addingClue,
        addGlobalClue,
        addPrivateClue,
        saveClue,
        removeClue,
        removeMystery,
        togglePaused,
        toggleGmAway,
        closeMystery,
        uploadingAttachment,
        attachmentError,
        uploadAttachment,
        removeAttachment,
        pendingMedia,
        addPendingMedia,
        removePendingMedia,
        pendingMediaSpoilers,
        togglePendingMediaSpoiler,
        uploadingMedia,
        mediaError,
        setMediaError,
        uploadMedia,
        removeMedia,
        comments,
    };
}
