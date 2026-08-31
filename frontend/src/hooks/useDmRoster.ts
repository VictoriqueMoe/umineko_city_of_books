import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";
import { useLocation, useNavigate } from "react-router";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";
import { moveRoomToFront, patchRoom, prependRoom, removeRoom, rosterEventFor } from "../domain/chat/dmRoster";
import type { ChatMessage, ChatRoom, User } from "../types/api";
import { fetchResolveDMRoom, useDmRooms, useUserRoomsCache } from "./queries/chat";
import { fetchMutualFollowers, fetchSearchUsers } from "./queries/user";
import { useAuth } from "./useAuth";

const CHAT_PATH = "/chat";
const SEARCH_DEBOUNCE_MS = 200;
const OPEN_FAILED = "Failed to open conversation";

export interface DmRoster {
    rooms: ChatRoom[];
    loading: boolean;
    draftRecipient: User | null;
    setDraftRecipient: Dispatch<SetStateAction<User | null>>;
    showNewDm: boolean;
    setShowNewDm: Dispatch<SetStateAction<boolean>>;
    search: string;
    setSearch: Dispatch<SetStateAction<string>>;
    results: User[];
    mutuals: User[];
    error: string;
    creating: boolean;
    selectRoom: (roomId: string) => void;
    selectUser: (selected: User) => Promise<void>;
    adoptRoom: (room: ChatRoom) => void;
    dropRoom: (roomId: string) => void;
    noteSent: (message: ChatMessage) => void;
    backToList: () => void;
}

export function useDmRoster(activeRoomId: string | undefined): DmRoster {
    const location = useLocation();
    const navigate = useNavigate();
    const { user } = useAuth();

    const { rooms, allRooms, loading: roomsLoading } = useDmRooms(!!user);
    const { patchRooms, refreshRooms } = useUserRoomsCache();
    const loading = !user || roomsLoading;

    const [showNewDm, setShowNewDm] = useState(false);
    const [search, setSearch] = useState("");
    const [results, setResults] = useState<User[]>([]);
    const [mutuals, setMutuals] = useState<User[]>([]);
    const [error, setError] = useState("");
    const [creating, setCreating] = useState(false);
    const [draftRecipient, setDraftRecipient] = useState<User | null>(null);

    useEffect(() => {
        const state = location.state as { dmUserId?: string } | null;
        if (!state?.dmUserId) {
            return;
        }

        const targetId = state.dmUserId;
        navigate(location.pathname, { replace: true, state: null });

        fetchResolveDMRoom(targetId)
            .then(resolved => {
                const resolvedRoom = resolved.room;
                if (!resolvedRoom) {
                    setDraftRecipient(resolved.recipient);
                    return;
                }

                patchRooms(prev => prependRoom(prev, resolvedRoom));
                setDraftRecipient(null);
                navigate(`${CHAT_PATH}/${resolvedRoom.id}`);
            })
            .catch(() => {});
    }, [location.state, location.pathname, navigate, patchRooms]);

    const unlistedRoomsRef = useRef(new Set<string>());
    useEffect(() => {
        for (const room of allRooms) {
            unlistedRoomsRef.current.delete(room.id);
        }
    }, [allRooms]);

    useRealtimeEvent(REALTIME_EVENTS.CHAT_MESSAGE, event => {
        if (!user) {
            return;
        }

        const chatMsg = event.data;
        const rosterEvent = rosterEventFor(allRooms, chatMsg.room_id);

        if (rosterEvent === "unknown") {
            if (unlistedRoomsRef.current.has(chatMsg.room_id)) {
                return;
            }

            unlistedRoomsRef.current.add(chatMsg.room_id);
            refreshRooms();
            return;
        }

        if (rosterEvent !== "dm") {
            return;
        }

        patchRooms(prev =>
            moveRoomToFront(prev, chatMsg.room_id, {
                last_message_at: chatMsg.created_at,
                unread: chatMsg.room_id !== activeRoomId && chatMsg.sender.id !== user.id,
            }),
        );
    });

    useEffect(() => {
        if (showNewDm) {
            fetchMutualFollowers()
                .then(setMutuals)
                .catch(() => setMutuals([]));
        }
    }, [showNewDm]);

    const searchDebounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    useEffect(() => {
        clearTimeout(searchDebounceRef.current);
        if (!search.trim()) {
            searchDebounceRef.current = setTimeout(() => {
                setResults([]);
            }, 0);
            return () => clearTimeout(searchDebounceRef.current);
        }

        searchDebounceRef.current = setTimeout(() => {
            fetchSearchUsers(search)
                .then(setResults)
                .catch(() => setResults([]));
        }, SEARCH_DEBOUNCE_MS);
        return () => clearTimeout(searchDebounceRef.current);
    }, [search]);

    function selectRoom(roomId: string) {
        patchRooms(prev => patchRoom(prev, roomId, { unread: false }));
        navigate(`${CHAT_PATH}/${roomId}`);
    }

    function adoptRoom(room: ChatRoom) {
        patchRooms(prev => patchRoom(prependRoom(prev, room), room.id, { unread: false }));
        setDraftRecipient(null);
        navigate(`${CHAT_PATH}/${room.id}`);
    }

    function dropRoom(roomId: string) {
        patchRooms(prev => removeRoom(prev, roomId));
        navigate(CHAT_PATH);
    }

    function noteSent(message: ChatMessage) {
        patchRooms(prev =>
            moveRoomToFront(prev, message.room_id, { last_message_at: message.created_at, unread: false }),
        );
    }

    function backToList() {
        setDraftRecipient(null);
        navigate(CHAT_PATH, { replace: true });
    }

    async function selectUser(selected: User): Promise<void> {
        setCreating(true);
        setError("");

        try {
            const resolved = await fetchResolveDMRoom(selected.id);
            setShowNewDm(false);
            setSearch("");
            setResults([]);

            const resolvedRoom = resolved.room;
            if (!resolvedRoom) {
                setDraftRecipient(resolved.recipient);
                navigate(CHAT_PATH);
                return;
            }

            adoptRoom(resolvedRoom);
        } catch (err) {
            setError(err instanceof Error ? err.message : OPEN_FAILED);
        } finally {
            setCreating(false);
        }
    }

    return {
        rooms,
        loading,
        draftRecipient,
        setDraftRecipient,
        showNewDm,
        setShowNewDm,
        search,
        setSearch,
        results,
        mutuals,
        error,
        creating,
        selectRoom,
        selectUser,
        adoptRoom,
        dropRoom,
        noteSent,
        backToList,
    };
}
