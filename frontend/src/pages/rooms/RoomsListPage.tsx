import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import type { ChatRoom, WSMessage } from "../../types/api";
import { getUserRooms, joinChatRoom, listMyChatRooms, listPublicChatRooms } from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { CreateRoomModal } from "../../components/chat/CreateRoomModal/CreateRoomModal";
import { isSiteStaff } from "../../utils/permissions";
import styles from "./RoomsPages.module.css";

const PAGE_SIZE = 20;

function relativeTime(dateStr?: string): string {
    if (!dateStr) {
        return "no activity yet";
    }
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) {
        return "just now";
    }
    if (mins < 60) {
        return `${mins}m ago`;
    }
    const hours = Math.floor(mins / 60);
    if (hours < 24) {
        return `${hours}h ago`;
    }
    const days = Math.floor(hours / 24);
    if (days < 30) {
        return `${days}d ago`;
    }
    return new Date(dateStr).toLocaleDateString();
}

type Page<T> = {
    items: T[];
    total: number;
    loading: boolean;
};

const emptyPage: Page<ChatRoom> = { items: [], total: 0, loading: true };

export function RoomsListPage() {
    usePageTitle("Chat Rooms");
    const navigate = useNavigate();
    const { user } = useAuth();
    const { addWSListener } = useNotifications();
    const [hosted, setHosted] = useState<Page<ChatRoom>>(emptyPage);
    const [joined, setJoined] = useState<Page<ChatRoom>>(emptyPage);
    const [discover, setDiscover] = useState<Page<ChatRoom>>(emptyPage);
    const [systemRooms, setSystemRooms] = useState<ChatRoom[]>([]);
    const [searchInput, setSearchInput] = useState("");
    const [search, setSearch] = useState("");
    const [rpOnly, setRpOnly] = useState(false);
    const [tagFilter, setTagFilter] = useState("");
    const [showCreate, setShowCreate] = useState(false);
    const [joining, setJoining] = useState<string | null>(null);
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

    const fetchHosted = useCallback(
        async (q: string, rp: boolean, tag: string, offset: number, append: boolean) => {
            if (!user) {
                setHosted({ items: [], total: 0, loading: false });
                return;
            }
            setHosted(prev => ({ ...prev, loading: true }));
            try {
                const res = await listMyChatRooms({
                    role: "host",
                    search: q,
                    rp,
                    tag: tag || undefined,
                    limit: PAGE_SIZE,
                    offset,
                });
                setHosted(prev => ({
                    items: append ? [...prev.items, ...(res.rooms ?? [])] : (res.rooms ?? []),
                    total: res.total ?? 0,
                    loading: false,
                }));
            } catch {
                setHosted({ items: [], total: 0, loading: false });
            }
        },
        [user],
    );

    const fetchJoined = useCallback(
        async (q: string, rp: boolean, tag: string, offset: number, append: boolean) => {
            if (!user) {
                setJoined({ items: [], total: 0, loading: false });
                return;
            }
            setJoined(prev => ({ ...prev, loading: true }));
            try {
                const res = await listMyChatRooms({
                    role: "member",
                    search: q,
                    rp,
                    tag: tag || undefined,
                    limit: PAGE_SIZE,
                    offset,
                });
                setJoined(prev => ({
                    items: append ? [...prev.items, ...(res.rooms ?? [])] : (res.rooms ?? []),
                    total: res.total ?? 0,
                    loading: false,
                }));
            } catch {
                setJoined({ items: [], total: 0, loading: false });
            }
        },
        [user],
    );

    const fetchDiscover = useCallback(async (q: string, rp: boolean, tag: string, offset: number, append: boolean) => {
        setDiscover(prev => ({ ...prev, loading: true }));
        try {
            const res = await listPublicChatRooms({
                search: q,
                rp,
                tag: tag || undefined,
                limit: PAGE_SIZE,
                offset,
            });
            setDiscover(prev => ({
                items: append ? [...prev.items, ...(res.rooms ?? [])] : (res.rooms ?? []),
                total: res.total ?? 0,
                loading: false,
            }));
        } catch {
            setDiscover({ items: [], total: 0, loading: false });
        }
    }, []);

    useEffect(() => {
        fetchHosted(search, rpOnly, tagFilter, 0, false);
        fetchJoined(search, rpOnly, tagFilter, 0, false);
        fetchDiscover(search, rpOnly, tagFilter, 0, false);
    }, [fetchHosted, fetchJoined, fetchDiscover, search, rpOnly, tagFilter]);

    const fetchSystemRooms = useCallback(() => {
        if (!user) {
            setSystemRooms([]);
            return;
        }
        getUserRooms()
            .then(res => setSystemRooms((res.rooms ?? []).filter(r => r.is_system)))
            .catch(() => setSystemRooms([]));
    }, [user]);

    useEffect(() => {
        fetchSystemRooms();
    }, [fetchSystemRooms]);

    useEffect(() => {
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => {
            setSearch(searchInput);
        }, 250);
        return () => clearTimeout(debounceRef.current);
    }, [searchInput]);

    useEffect(() => {
        return addWSListener((msg: WSMessage) => {
            if (msg.type === "chat_room_invited") {
                fetchJoined(search, rpOnly, tagFilter, 0, false);
                fetchSystemRooms();
                return;
            }
            if (msg.type === "chat_kicked" || msg.type === "chat_room_deleted") {
                const data = msg.data as { room_id: string };
                setHosted(prev => ({
                    ...prev,
                    items: prev.items.filter(r => r.id !== data.room_id),
                    total: Math.max(0, prev.total - 1),
                }));
                setJoined(prev => ({
                    ...prev,
                    items: prev.items.filter(r => r.id !== data.room_id),
                    total: Math.max(0, prev.total - 1),
                }));
                setSystemRooms(prev => prev.filter(r => r.id !== data.room_id));
            }
        });
    }, [addWSListener, fetchJoined, fetchSystemRooms, search, rpOnly, tagFilter]);

    async function handleJoin(room: ChatRoom, ghost = false) {
        setJoining(room.id);
        try {
            const joinedRoom = await joinChatRoom(room.id, { ghost });
            setJoined(prev => {
                if (prev.items.some(r => r.id === joinedRoom.id)) {
                    return prev;
                }
                return {
                    ...prev,
                    items: [joinedRoom, ...prev.items],
                    total: prev.total + 1,
                };
            });
            setDiscover(prev => ({
                ...prev,
                items: prev.items.filter(r => r.id !== joinedRoom.id),
                total: Math.max(0, prev.total - 1),
            }));
            navigate(`/rooms/${joinedRoom.id}`);
        } catch {
            // handled server-side
        } finally {
            setJoining(null);
        }
    }

    function renderMemberCard(room: ChatRoom) {
        const classes = [styles.card];
        if (room.is_system) {
            classes.push(styles.systemCard);
        }
        if (room.viewer_ghost) {
            classes.push(styles.ghostCard);
        }
        if (room.viewer_muted) {
            classes.push(styles.mutedCard);
        }
        return (
            <Link key={room.id} to={`/rooms/${room.id}`} className={classes.join(" ")}>
                <div className={styles.cardHeader}>
                    <h3 className={styles.cardTitle}>{room.name}</h3>
                    <div className={styles.cardBadges}>
                        {room.is_system && <span className={styles.systemBadge}>Pinned</span>}
                        {room.viewer_role === "host" && <span className={styles.hostBadge}>Host</span>}
                        {room.viewer_ghost && (
                            <span className={styles.ghostBadge} title="You joined silently as a ghost">
                                👻 Ghost
                            </span>
                        )}
                        {room.viewer_muted && (
                            <span className={styles.mutedBadge} title="Notifications muted">
                                🔕 Muted
                            </span>
                        )}
                        {room.is_rp && <span className={styles.rpBadge}>RP</span>}
                        {!room.is_system &&
                            (room.is_public ? (
                                <span className={styles.publicBadge}>Public</span>
                            ) : (
                                <span className={styles.privateBadge}>Private</span>
                            ))}
                    </div>
                </div>
                {room.description && <p className={styles.cardDesc}>{room.description}</p>}
                {room.tags && room.tags.length > 0 && (
                    <div className={styles.cardTags}>
                        {room.tags.map(t => (
                            <button
                                key={t}
                                className={styles.cardTag}
                                onClick={e => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    setTagFilter(t);
                                }}
                            >
                                #{t}
                            </button>
                        ))}
                    </div>
                )}
                <div className={styles.cardMeta}>
                    <span>
                        {"\u2605"} {room.member_count ?? room.members.length} members
                    </span>
                    <span className={styles.cardActivity}>{relativeTime(room.last_message_at)}</span>
                </div>
            </Link>
        );
    }

    const filterActive = search !== "" || rpOnly || tagFilter !== "";
    const hostedFiltered = hosted.items.filter(r => !r.is_system);
    const joinedFiltered = joined.items.filter(r => !r.is_system);

    function renderGroupedGrid(rooms: ChatRoom[], renderCard: (room: ChatRoom) => React.ReactNode) {
        const rpRooms: ChatRoom[] = [];
        const chatRooms: ChatRoom[] = [];
        for (const room of rooms) {
            if (room.is_rp) {
                rpRooms.push(room);
            } else {
                chatRooms.push(room);
            }
        }
        const hasRP = rpRooms.length > 0;
        const hasChat = chatRooms.length > 0;
        const showLabels = hasRP && hasChat;
        return (
            <>
                {hasRP && (
                    <>
                        {showLabels && <div className={styles.subGroupLabel}>Roleplay</div>}
                        <div className={styles.cardGrid}>{rpRooms.map(renderCard)}</div>
                    </>
                )}
                {hasChat && (
                    <>
                        {showLabels && <div className={styles.subGroupLabel}>Chat</div>}
                        <div className={styles.cardGrid}>{chatRooms.map(renderCard)}</div>
                    </>
                )}
            </>
        );
    }

    function renderDiscoverCard(room: ChatRoom) {
        return (
            <div key={room.id} className={styles.card}>
                <div className={styles.cardHeader}>
                    <h3 className={styles.cardTitle}>{room.name}</h3>
                    <div className={styles.cardBadges}>
                        {room.is_rp && <span className={styles.rpBadge}>RP</span>}
                        <span className={styles.publicBadge}>Public</span>
                    </div>
                </div>
                {room.description && <p className={styles.cardDesc}>{room.description}</p>}
                {room.tags && room.tags.length > 0 && (
                    <div className={styles.cardTags}>
                        {room.tags.map(t => (
                            <button key={t} className={styles.cardTag} onClick={() => setTagFilter(t)}>
                                #{t}
                            </button>
                        ))}
                    </div>
                )}
                <div className={styles.cardMeta}>
                    <span>
                        {"\u2605"} {room.member_count ?? room.members.length} members
                    </span>
                    <span className={styles.cardActivity}>{relativeTime(room.last_message_at)}</span>
                </div>
                {user && (
                    <div className={styles.cardActions}>
                        <Button
                            variant="primary"
                            size="small"
                            onClick={() => handleJoin(room)}
                            disabled={joining === room.id}
                        >
                            {joining === room.id ? "Joining..." : "Join Room"}
                        </Button>
                        {isSiteStaff(user.role) && (
                            <Button
                                variant="ghost"
                                size="small"
                                onClick={() => handleJoin(room, true)}
                                disabled={joining === room.id}
                                title="Join silently — no system message, hidden from member list except to staff"
                            >
                                👻 Ghost
                            </Button>
                        )}
                    </div>
                )}
            </div>
        );
    }

    return (
        <div className={styles.page}>
            <div className={styles.pageHeader}>
                <h1 className={styles.pageTitle}>Chat Rooms</h1>
                {user && (
                    <Button variant="primary" size="small" onClick={() => setShowCreate(true)}>
                        + New Room
                    </Button>
                )}
            </div>

            <InfoPanel title="What are Chat Rooms?">
                <p>
                    Chat Rooms are <strong>live group chats</strong> for whatever you want: roleplay scenarios, episode
                    reaction crews, book clubs, or just hanging out. They're separate from Direct Messages.
                </p>
                <p>
                    Anyone can create a room and pick whether it's <strong>public</strong> (anyone can browse and join)
                    or <strong>private</strong> (invite-only). The room creator is the host and can kick members or
                    delete the room. Group rooms only ping you with a notification when someone{" "}
                    <strong>@mentions</strong>
                    you, so you won't get spammed by busy chats.
                </p>
            </InfoPanel>

            <RulesBox page="chat_rooms" />

            <div className={styles.filterBar}>
                <Input
                    type="text"
                    placeholder="Search rooms..."
                    value={searchInput}
                    onChange={e => setSearchInput(e.target.value)}
                    className={styles.searchInput}
                />
                <div className={styles.filterRow}>
                    <button
                        className={`${styles.filterChip}${rpOnly ? ` ${styles.filterChipActive}` : ""}`}
                        onClick={() => setRpOnly(prev => !prev)}
                    >
                        RP only
                    </button>
                    {tagFilter && (
                        <button
                            className={`${styles.filterChip} ${styles.filterChipActive}`}
                            onClick={() => setTagFilter("")}
                        >
                            #{tagFilter} x
                        </button>
                    )}
                </div>
            </div>

            {user && (
                <section className={styles.section}>
                    <h2 className={styles.sectionTitle}>
                        My Rooms{hostedFiltered.length > 0 ? ` (${hostedFiltered.length})` : ""}
                    </h2>
                    {hosted.loading && hostedFiltered.length === 0 && (
                        <div className="loading">Loading your rooms...</div>
                    )}
                    {!hosted.loading && hostedFiltered.length === 0 && (
                        <div className="empty-state">
                            {filterActive
                                ? "No rooms you host match the filters."
                                : "You haven't created any rooms yet."}
                        </div>
                    )}
                    {hostedFiltered.length > 0 && renderGroupedGrid(hostedFiltered, renderMemberCard)}
                    {hosted.items.length < hosted.total && (
                        <div className={styles.loadMoreRow}>
                            <Button
                                variant="secondary"
                                size="small"
                                onClick={() => fetchHosted(search, rpOnly, tagFilter, hosted.items.length, true)}
                                disabled={hosted.loading}
                            >
                                {hosted.loading ? "Loading..." : "Load more"}
                            </Button>
                        </div>
                    )}
                </section>
            )}

            {user && (
                <section className={styles.section}>
                    <h2 className={styles.sectionTitle}>
                        Joined Rooms
                        {joinedFiltered.length > 0 || systemRooms.length > 0
                            ? ` (${joinedFiltered.length + systemRooms.length})`
                            : ""}
                    </h2>
                    {systemRooms.length > 0 && (
                        <>
                            <div className={styles.pinnedGroupLabel}>Pinned</div>
                            <div className={styles.cardGrid}>{systemRooms.map(renderMemberCard)}</div>
                        </>
                    )}
                    {joined.loading && joinedFiltered.length === 0 && systemRooms.length === 0 && (
                        <div className="loading">Loading your rooms...</div>
                    )}
                    {!joined.loading && joinedFiltered.length === 0 && systemRooms.length === 0 && (
                        <div className="empty-state">
                            {filterActive
                                ? "No joined rooms match the filters."
                                : "You haven't joined any rooms yet. Browse below or create one."}
                        </div>
                    )}
                    {joinedFiltered.length > 0 && renderGroupedGrid(joinedFiltered, renderMemberCard)}
                    {joined.items.length < joined.total && (
                        <div className={styles.loadMoreRow}>
                            <Button
                                variant="secondary"
                                size="small"
                                onClick={() => fetchJoined(search, rpOnly, tagFilter, joined.items.length, true)}
                                disabled={joined.loading}
                            >
                                {joined.loading ? "Loading..." : "Load more"}
                            </Button>
                        </div>
                    )}
                </section>
            )}

            <section className={styles.section}>
                <h2 className={styles.sectionTitle}>
                    Discover Public Rooms{discover.total > 0 ? ` (${discover.total})` : ""}
                </h2>
                {discover.loading && discover.items.length === 0 && (
                    <div className="loading">Loading public rooms...</div>
                )}
                {!discover.loading && discover.items.length === 0 && (
                    <div className="empty-state">
                        {filterActive
                            ? "No public rooms match your search."
                            : "No public rooms yet. Create the first one!"}
                    </div>
                )}
                {discover.items.length > 0 && renderGroupedGrid(discover.items, renderDiscoverCard)}
                {discover.items.length < discover.total && (
                    <div className={styles.loadMoreRow}>
                        <Button
                            variant="secondary"
                            size="small"
                            onClick={() => fetchDiscover(search, rpOnly, tagFilter, discover.items.length, true)}
                            disabled={discover.loading}
                        >
                            {discover.loading ? "Loading..." : "Load more"}
                        </Button>
                    </div>
                )}
            </section>

            <CreateRoomModal
                isOpen={showCreate}
                onClose={() => setShowCreate(false)}
                onCreated={room => {
                    setHosted(prev => ({
                        ...prev,
                        items: [room, ...prev.items],
                        total: prev.total + 1,
                    }));
                    navigate(`/rooms/${room.id}`);
                }}
            />
        </div>
    );
}
