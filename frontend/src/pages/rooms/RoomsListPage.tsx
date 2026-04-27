import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import type { ChatRoom, WSMessage } from "../../types/api";
import { getUserRooms, listMyChatRooms, listPublicChatRooms } from "../../api/endpoints";
import { useJoinChatRoom } from "../../api/mutations/chat";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { CreateRoomModal } from "../../components/chat/CreateRoomModal/CreateRoomModal";
import { isSiteStaff } from "../../utils/permissions";
import { PieceTrigger } from "../../features/easterEgg";
import { formatActiveLabel, formatFullDateTime } from "../../utils/time";
import styles from "./RoomsPages.module.css";

const PAGE_SIZE = 20;
const HOT_THRESHOLD = 50;

interface FilterState {
    search: string;
    rpOnly: boolean;
    tagFilter: string;
    includeArchived: boolean;
}

function listKey(scope: string, filters: FilterState, pages: number): readonly unknown[] {
    return [
        "chat",
        "rooms-list",
        scope,
        filters.search,
        filters.rpOnly,
        filters.tagFilter,
        filters.includeArchived,
        pages,
    ];
}

export function RoomsListPage() {
    usePageTitle("Chat Rooms");
    const navigate = useNavigate();
    const qc = useQueryClient();
    const { user } = useAuth();
    const { addWSListener } = useNotifications();
    const [searchInput, setSearchInput] = useState("");
    const [search, setSearch] = useState("");
    const [rpOnly, setRpOnly] = useState(false);
    const [tagFilter, setTagFilter] = useState("");
    const [includeArchived, setIncludeArchived] = useState(false);
    const [pages, setPages] = useState<{ key: string; hosted: number; joined: number; discover: number }>({
        key: "",
        hosted: 1,
        joined: 1,
        discover: 1,
    });
    const [showCreate, setShowCreate] = useState(false);
    const [joining, setJoining] = useState<string | null>(null);
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    const joinRoomMutation = useJoinChatRoom();

    const filters: FilterState = useMemo(
        () => ({ search, rpOnly, tagFilter, includeArchived }),
        [search, rpOnly, tagFilter, includeArchived],
    );

    const filtersKey = `${filters.search}|${filters.rpOnly}|${filters.tagFilter}|${filters.includeArchived}`;
    const activePages = pages.key === filtersKey ? pages : { key: filtersKey, hosted: 1, joined: 1, discover: 1 };
    const hostedPages = activePages.hosted;
    const joinedPages = activePages.joined;
    const discoverPages = activePages.discover;

    function loadMoreHosted() {
        setPages(prev => {
            const base = prev.key === filtersKey ? prev : { key: filtersKey, hosted: 1, joined: 1, discover: 1 };
            return { ...base, hosted: base.hosted + 1 };
        });
    }
    function loadMoreJoined() {
        setPages(prev => {
            const base = prev.key === filtersKey ? prev : { key: filtersKey, hosted: 1, joined: 1, discover: 1 };
            return { ...base, joined: base.joined + 1 };
        });
    }
    function loadMoreDiscover() {
        setPages(prev => {
            const base = prev.key === filtersKey ? prev : { key: filtersKey, hosted: 1, joined: 1, discover: 1 };
            return { ...base, discover: base.discover + 1 };
        });
    }

    const hostedQuery = useQuery({
        queryKey: listKey("hosted", filters, hostedPages),
        queryFn: () =>
            listMyChatRooms({
                role: "host",
                search: filters.search,
                rp: filters.rpOnly,
                tag: filters.tagFilter || undefined,
                includeArchived: filters.includeArchived,
                limit: PAGE_SIZE * hostedPages,
                offset: 0,
            }),
        enabled: !!user,
    });
    const joinedQuery = useQuery({
        queryKey: listKey("joined", filters, joinedPages),
        queryFn: () =>
            listMyChatRooms({
                role: "member",
                search: filters.search,
                rp: filters.rpOnly,
                tag: filters.tagFilter || undefined,
                includeArchived: filters.includeArchived,
                limit: PAGE_SIZE * joinedPages,
                offset: 0,
            }),
        enabled: !!user,
    });
    const discoverQuery = useQuery({
        queryKey: listKey("discover", filters, discoverPages),
        queryFn: () =>
            listPublicChatRooms({
                search: filters.search,
                rp: filters.rpOnly,
                tag: filters.tagFilter || undefined,
                includeArchived: filters.includeArchived,
                limit: PAGE_SIZE * discoverPages,
                offset: 0,
            }),
    });

    const systemRoomsQuery = useQuery({
        queryKey: ["chat", "rooms", "user", "system"],
        queryFn: () => getUserRooms(),
        enabled: !!user,
    });

    const hosted = {
        items: hostedQuery.data?.rooms ?? [],
        total: hostedQuery.data?.total ?? 0,
        loading: hostedQuery.isFetching,
    };
    const joined = {
        items: joinedQuery.data?.rooms ?? [],
        total: joinedQuery.data?.total ?? 0,
        loading: joinedQuery.isFetching,
    };
    const discover = {
        items: discoverQuery.data?.rooms ?? [],
        total: discoverQuery.data?.total ?? 0,
        loading: discoverQuery.isFetching,
    };
    const systemRooms = (systemRoomsQuery.data?.rooms ?? []).filter(r => r.is_system);

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
                qc.invalidateQueries({ queryKey: ["chat", "rooms-list", "joined"] });
                qc.invalidateQueries({ queryKey: ["chat", "rooms", "user", "system"] });
                return;
            }
            if (msg.type === "chat_kicked" || msg.type === "chat_room_deleted") {
                qc.invalidateQueries({ queryKey: ["chat", "rooms-list"] });
                qc.invalidateQueries({ queryKey: ["chat", "rooms", "user", "system"] });
            }
        });
    }, [addWSListener, qc]);

    async function handleJoin(room: ChatRoom, ghost = false) {
        setJoining(room.id);
        try {
            const joinedRoom = await joinRoomMutation.mutateAsync({ roomId: room.id, ghost });
            qc.invalidateQueries({ queryKey: ["chat", "rooms-list", "joined"] });
            qc.invalidateQueries({ queryKey: ["chat", "rooms-list", "discover"] });
            navigate(`/rooms/${joinedRoom.id}`);
        } catch {
            void 0;
        } finally {
            setJoining(null);
        }
    }

    function renderMemberCard(room: ChatRoom) {
        const classes = [styles.card];
        const isHot = !room.archived_at && (room.hot_score ?? 0) >= HOT_THRESHOLD;
        if (room.is_system) {
            classes.push(styles.systemCard);
        }
        if (room.viewer_ghost) {
            classes.push(styles.ghostCard);
        }
        if (room.viewer_muted) {
            classes.push(styles.mutedCard);
        }
        if (room.archived_at) {
            classes.push(styles.archivedCard);
        }
        if (isHot) {
            classes.push(styles.hotCard);
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
                        {room.archived_at && (
                            <span className={styles.archivedBadge} title="No recent messages">
                                Archived
                            </span>
                        )}
                        {isHot && (
                            <span className={styles.hotBadge} title="Lots of activity in the last 24 hours">
                                Hot
                            </span>
                        )}
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
                    <span
                        className={styles.cardActivity}
                        title={room.last_message_at ? formatFullDateTime(room.last_message_at) : undefined}
                    >
                        {formatActiveLabel(room.last_message_at)}
                    </span>
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
        const classes = [styles.card];
        const isHot = !room.archived_at && (room.hot_score ?? 0) >= HOT_THRESHOLD;
        if (room.archived_at) {
            classes.push(styles.archivedCard);
        }
        if (isHot) {
            classes.push(styles.hotCard);
        }
        return (
            <div key={room.id} className={classes.join(" ")}>
                <div className={styles.cardHeader}>
                    <h3 className={styles.cardTitle}>{room.name}</h3>
                    <div className={styles.cardBadges}>
                        {room.is_rp && <span className={styles.rpBadge}>RP</span>}
                        <span className={styles.publicBadge}>Public</span>
                        {room.archived_at && (
                            <span className={styles.archivedBadge} title="No recent messages">
                                Archived
                            </span>
                        )}
                        {isHot && (
                            <span className={styles.hotBadge} title="Lots of activity in the last 24 hours">
                                Hot
                            </span>
                        )}
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
                    <span
                        className={styles.cardActivity}
                        title={room.last_message_at ? formatFullDateTime(room.last_message_at) : undefined}
                    >
                        {formatActiveLabel(room.last_message_at)}
                    </span>
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
                        + New Room <PieceTrigger pieceId="piece_03" />
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
                    <button
                        className={`${styles.filterChip}${includeArchived ? ` ${styles.filterChipActive}` : ""}`}
                        onClick={() => setIncludeArchived(prev => !prev)}
                    >
                        Include archived
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
                            <Button variant="secondary" size="small" onClick={loadMoreHosted} disabled={hosted.loading}>
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
                            <Button variant="secondary" size="small" onClick={loadMoreJoined} disabled={joined.loading}>
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
                        <Button variant="secondary" size="small" onClick={loadMoreDiscover} disabled={discover.loading}>
                            {discover.loading ? "Loading..." : "Load more"}
                        </Button>
                    </div>
                )}
            </section>

            <CreateRoomModal
                isOpen={showCreate}
                onClose={() => setShowCreate(false)}
                onCreated={room => {
                    qc.invalidateQueries({ queryKey: ["chat", "rooms-list", "hosted"] });
                    navigate(`/rooms/${room.id}`);
                }}
            />
        </div>
    );
}
