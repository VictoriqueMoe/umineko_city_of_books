import { usePageTitle } from "../../hooks/usePageTitle";
import { useAuth } from "../../hooks/useAuth";
import { useRoomsDirectory } from "../../hooks/chat/useRoomsDirectory";
import type { ChatRoom } from "../../types/api";
import { Button } from "../../components/Button/Button";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import { Input } from "../../components/Input/Input";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { CreateRoomModal } from "../../components/chat/CreateRoomModal/CreateRoomModal";
import { RoomCard } from "../../components/chat/RoomCard/RoomCard";
import { isSiteStaff } from "../../domain/permissions";
import { PieceTrigger } from "../../components/easterEgg";
import styles from "./RoomsPages.module.css";

const GHOST_JOIN_TITLE = "Join silently, no system message, hidden from member list except to staff";

export function RoomsListPage() {
    usePageTitle("Chat Rooms");
    const { user } = useAuth();
    const { hosted, joined, discover, systemRooms, filters, join, create } = useRoomsDirectory();

    function renderMemberCard(room: ChatRoom) {
        return <RoomCard key={room.id} variant="member" room={room} onTagClick={filters.setTagFilter} />;
    }

    const hostedFiltered = hosted.items.filter(r => !r.is_system);
    const joinedFiltered = joined.items.filter(r => !r.is_system);

    function renderGroupedGrid(
        rooms: ChatRoom[],
        renderCard: (room: ChatRoom) => React.ReactNode,
        forceLabels = false,
    ) {
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
        const showLabels = forceLabels || (hasRP && hasChat);
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
            <RoomCard
                key={room.id}
                variant="discover"
                room={room}
                onTagClick={filters.setTagFilter}
                actions={
                    user ? (
                        <>
                            <Button
                                variant="primary"
                                size="small"
                                onClick={() => join.join(room)}
                                disabled={join.pending === room.id}
                            >
                                {join.pending === room.id ? "Joining..." : "Join Room"}
                            </Button>
                            {isSiteStaff(user.role) && (
                                <Button
                                    variant="ghost"
                                    size="small"
                                    onClick={() => join.join(room, true)}
                                    disabled={join.pending === room.id}
                                    title={GHOST_JOIN_TITLE}
                                >
                                    👻 Ghost
                                </Button>
                            )}
                        </>
                    ) : null
                }
            />
        );
    }

    return (
        <div className={styles.page}>
            <div className={styles.pageHeader}>
                <h1 className={styles.pageTitle}>Chat Rooms</h1>
                {user && (
                    <Button variant="primary" size="small" onClick={create.open}>
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
                    value={filters.searchInput}
                    onChange={e => filters.setSearchInput(e.target.value)}
                    className={styles.searchInput}
                />
                <div className={styles.filterRow}>
                    <button
                        className={`${styles.filterChip}${filters.rpOnly ? ` ${styles.filterChipActive}` : ""}`}
                        onClick={filters.toggleRpOnly}
                    >
                        RP only
                    </button>
                    <button
                        className={`${styles.filterChip}${filters.includeArchived ? ` ${styles.filterChipActive}` : ""}`}
                        onClick={filters.toggleIncludeArchived}
                    >
                        Include archived
                    </button>
                    {filters.tagFilter && (
                        <button
                            className={`${styles.filterChip} ${styles.filterChipActive}`}
                            onClick={() => filters.setTagFilter("")}
                        >
                            #{filters.tagFilter} x
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
                            {filters.active
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
                                onClick={hosted.loadMore}
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
                            {filters.active
                                ? "No joined rooms match the filters."
                                : "You haven't joined any rooms yet. Browse below or create one."}
                        </div>
                    )}
                    {joinedFiltered.length > 0 &&
                        renderGroupedGrid(joinedFiltered, renderMemberCard, systemRooms.length > 0)}
                    {joined.items.length < joined.total && (
                        <div className={styles.loadMoreRow}>
                            <Button
                                variant="secondary"
                                size="small"
                                onClick={joined.loadMore}
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
                {join.error && <ErrorBanner message={join.error} />}
                {discover.loading && discover.items.length === 0 && (
                    <div className="loading">Loading public rooms...</div>
                )}
                {!discover.loading && discover.items.length === 0 && (
                    <div className="empty-state">
                        {filters.active
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
                            onClick={discover.loadMore}
                            disabled={discover.loading}
                        >
                            {discover.loading ? "Loading..." : "Load more"}
                        </Button>
                    </div>
                )}
            </section>

            <CreateRoomModal isOpen={create.isOpen} onClose={create.close} onCreated={create.onCreated} />
        </div>
    );
}
