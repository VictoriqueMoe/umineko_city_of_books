import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { REALTIME_EVENTS } from "../../api/realtime/events";
import { useRealtimeEvent } from "../../api/realtime/useRealtime";
import { queryKeys, type RoomsListParams, type RoomsListScope } from "../../api/queryKeys";
import type { ChatRoom } from "../../types/api";
import { useAuth } from "../useAuth";
import { useJoinChatRoom } from "../mutations/chat";
import { useHostedRooms, useJoinedRooms, usePublicRooms, useUserRooms } from "../queries/chat";

const SEARCH_DEBOUNCE_MS = 250;

const ROOM_MEMBERSHIP_EVENTS = [
    REALTIME_EVENTS.CHAT_ROOM_UPDATED,
    REALTIME_EVENTS.CHAT_KICKED,
    REALTIME_EVENTS.CHAT_ROOM_DELETED,
] as const;

interface PageCounts {
    key: string;
    hosted: number;
    joined: number;
    discover: number;
}

export interface RoomsDirectorySection {
    items: ChatRoom[];
    total: number;
    loading: boolean;
    loadMore: () => void;
}

export interface RoomsDirectoryFilters {
    searchInput: string;
    setSearchInput: (value: string) => void;
    rpOnly: boolean;
    toggleRpOnly: () => void;
    includeArchived: boolean;
    toggleIncludeArchived: () => void;
    tagFilter: string;
    setTagFilter: (tag: string) => void;
    active: boolean;
}

export interface RoomsDirectoryJoin {
    pending: string | null;
    error: string;
    join: (room: ChatRoom, ghost?: boolean) => Promise<void>;
}

export interface RoomsDirectoryCreate {
    isOpen: boolean;
    open: () => void;
    close: () => void;
    onCreated: (room: ChatRoom) => void;
}

export interface UseRoomsDirectoryResult {
    hosted: RoomsDirectorySection;
    joined: RoomsDirectorySection;
    discover: RoomsDirectorySection;
    systemRooms: ChatRoom[];
    filters: RoomsDirectoryFilters;
    join: RoomsDirectoryJoin;
    create: RoomsDirectoryCreate;
}

function firstPage(key: string): PageCounts {
    return { key, hosted: 1, joined: 1, discover: 1 };
}

function bumpPage(previous: PageCounts, key: string, scope: RoomsListScope): PageCounts {
    const base = previous.key === key ? previous : firstPage(key);

    if (scope === "hosted") {
        return { ...base, hosted: base.hosted + 1 };
    }
    if (scope === "joined") {
        return { ...base, joined: base.joined + 1 };
    }

    return { ...base, discover: base.discover + 1 };
}

export function useRoomsDirectory(): UseRoomsDirectoryResult {
    const navigate = useNavigate();
    const queryClient = useQueryClient();
    const { user } = useAuth();

    const [searchInput, setSearchInput] = useState("");
    const [search, setSearch] = useState("");
    const [rpOnly, setRpOnly] = useState(false);
    const [tagFilter, setTagFilter] = useState("");
    const [includeArchived, setIncludeArchived] = useState(false);
    const [pages, setPages] = useState<PageCounts>(() => firstPage(""));
    const [showCreate, setShowCreate] = useState(false);
    const [joining, setJoining] = useState<string | null>(null);
    const [joinError, setJoinError] = useState("");
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    const joinRoomMutation = useJoinChatRoom();

    const filtersKey = `${search}|${rpOnly}|${tagFilter}|${includeArchived}`;
    const activePages = pages.key === filtersKey ? pages : firstPage(filtersKey);

    function paramsFor(scope: RoomsListScope): RoomsListParams {
        return { search, rpOnly, tagFilter, includeArchived, pages: activePages[scope] };
    }

    const hostedRooms = useHostedRooms(paramsFor("hosted"), !!user);
    const joinedRooms = useJoinedRooms(paramsFor("joined"), !!user);
    const discoverRooms = usePublicRooms(paramsFor("discover"));
    const userRoomsQuery = useUserRooms(!!user);

    useEffect(() => {
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => {
            setSearch(searchInput);
        }, SEARCH_DEBOUNCE_MS);

        return () => clearTimeout(debounceRef.current);
    }, [searchInput]);

    useRealtimeEvent(REALTIME_EVENTS.CHAT_ROOM_INVITED, () => {
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.roomsListJoined() });
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.userRooms() });
    });

    useRealtimeEvent(ROOM_MEMBERSHIP_EVENTS, () => {
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.roomsList() });
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.userRooms() });
    });

    useRealtimeEvent(REALTIME_EVENTS.VOICE_PRESENCE, () => {
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.roomsList() });
    });

    function loadMore(scope: RoomsListScope): void {
        setPages(previous => bumpPage(previous, filtersKey, scope));
    }

    async function join(room: ChatRoom, ghost = false): Promise<void> {
        setJoining(room.id);
        setJoinError("");

        try {
            const joinedRoom = await joinRoomMutation.mutateAsync({ roomId: room.id, ghost });

            queryClient.invalidateQueries({ queryKey: queryKeys.chat.roomsListJoined() });
            queryClient.invalidateQueries({ queryKey: queryKeys.chat.roomsListDiscover() });
            navigate(`/rooms/${joinedRoom.id}`);
        } catch (err) {
            setJoinError(err instanceof Error ? err.message : "Could not join that room.");
        } finally {
            setJoining(null);
        }
    }

    function onCreated(room: ChatRoom): void {
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.roomsListHosted() });
        navigate(`/rooms/${room.id}`);
    }

    return {
        hosted: {
            items: hostedRooms.rooms,
            total: hostedRooms.total,
            loading: hostedRooms.loading,
            loadMore: () => loadMore("hosted"),
        },
        joined: {
            items: joinedRooms.rooms,
            total: joinedRooms.total,
            loading: joinedRooms.loading,
            loadMore: () => loadMore("joined"),
        },
        discover: {
            items: discoverRooms.rooms,
            total: discoverRooms.total,
            loading: discoverRooms.loading,
            loadMore: () => loadMore("discover"),
        },
        systemRooms: userRoomsQuery.rooms.filter(room => room.is_system),
        filters: {
            searchInput,
            setSearchInput,
            rpOnly,
            toggleRpOnly: () => setRpOnly(previous => !previous),
            includeArchived,
            toggleIncludeArchived: () => setIncludeArchived(previous => !previous),
            tagFilter,
            setTagFilter,
            active: search !== "" || rpOnly || tagFilter !== "",
        },
        join: { pending: joining, error: joinError, join },
        create: {
            isOpen: showCreate,
            open: () => setShowCreate(true),
            close: () => setShowCreate(false),
            onCreated,
        },
    };
}
