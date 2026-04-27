import { Link } from "react-router";
import { useHomeActivity } from "../../api/queries/sidebar";
import type { HomeActivityEntry, HomeMember, HomePublicRoom } from "../../types/api";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { relativeTime } from "../../utils/time";
import styles from "./LiveActivity.module.css";

const kindLabel: Record<HomeActivityEntry["kind"], string> = {
    theory: "Theory",
    post: "Post",
    journal: "Journal",
    art: "Gallery",
};

function displayTitle(entry: HomeActivityEntry): string {
    if (entry.title) {
        return entry.title;
    }
    const excerpt = entry.excerpt.trim();
    if (!excerpt) {
        return `${kindLabel[entry.kind]} entry`;
    }
    return excerpt.length > 80 ? `${excerpt.slice(0, 80)}…` : excerpt;
}

interface ActivityRowProps {
    entry: HomeActivityEntry;
}

function ActivityRow({ entry }: ActivityRowProps) {
    return (
        <li className={styles.activityItem}>
            <Link to={entry.url} className={styles.activityLink}>
                <span className={styles.activityKind}>{kindLabel[entry.kind]}</span>
                <span className={styles.activityTitle}>{displayTitle(entry)}</span>
            </Link>
            <div className={styles.activityMeta}>
                <ProfileLink
                    user={{
                        id: entry.author.id,
                        username: entry.author.username,
                        display_name: entry.author.display_name,
                        avatar_url: entry.author.avatar_url,
                    }}
                    size="small"
                />
                <span className={styles.activityTime}>{relativeTime(entry.created_at)}</span>
            </div>
        </li>
    );
}

interface MemberChipProps {
    member: HomeMember;
}

function MemberChip({ member }: MemberChipProps) {
    return (
        <ProfileLink
            user={{
                id: member.id,
                username: member.username,
                display_name: member.display_name,
                avatar_url: member.avatar_url,
            }}
            size="medium"
        />
    );
}

interface RoomCardProps {
    room: HomePublicRoom;
}

function RoomCard({ room }: RoomCardProps) {
    return (
        <Link to={`/rooms/${room.id}`} className={styles.roomCard}>
            <span className={styles.roomName}>{room.name || "Untitled room"}</span>
            {room.description && <span className={styles.roomDescription}>{room.description}</span>}
            <span className={styles.roomMembers}>
                {room.member_count} {room.member_count === 1 ? "witch" : "witches"}
                {room.last_message_at && <> &middot; {relativeTime(room.last_message_at)}</>}
            </span>
        </Link>
    );
}

export function LiveActivity() {
    const { data } = useHomeActivity();

    if (!data) {
        return (
            <section id="live" className={styles.live} aria-busy="true">
                <div className={styles.heading}>
                    <h2 className={styles.title}>Live on the Board</h2>
                    <span className={styles.loading}>Listening...</span>
                </div>
            </section>
        );
    }

    const hasActivity = data.recent_activity.length > 0;
    const hasMembers = data.recent_members.length > 0;
    const hasRooms = data.public_rooms.length > 0;

    return (
        <section id="live" className={styles.live}>
            <div className={styles.heading}>
                <h2 className={styles.title}>Live on the Board</h2>
                <span className={styles.onlineBadge}>
                    <span className={styles.onlineDot} aria-hidden="true" />
                    {data.online_count} online now
                </span>
            </div>

            <div className={styles.grid}>
                <div className={styles.column}>
                    <h3 className={styles.columnTitle}>Recent activity</h3>
                    {hasActivity ? (
                        <ul className={styles.activityList}>
                            {data.recent_activity.map(entry => (
                                <ActivityRow key={`${entry.kind}-${entry.id}`} entry={entry} />
                            ))}
                        </ul>
                    ) : (
                        <p className={styles.empty}>The board is quiet. Be the first to post.</p>
                    )}
                </div>

                <div className={styles.sideColumn}>
                    <div>
                        <h3 className={styles.columnTitle}>Public rooms</h3>
                        {hasRooms ? (
                            <div className={styles.roomList}>
                                {data.public_rooms.map(room => (
                                    <RoomCard key={room.id} room={room} />
                                ))}
                            </div>
                        ) : (
                            <p className={styles.empty}>
                                No public rooms yet. <Link to="/rooms">Open one</Link> to gather witnesses.
                            </p>
                        )}
                    </div>

                    <div>
                        <h3 className={styles.columnTitle}>New witnesses</h3>
                        {hasMembers ? (
                            <div className={styles.memberList}>
                                {data.recent_members.map(member => (
                                    <MemberChip key={member.id} member={member} />
                                ))}
                            </div>
                        ) : (
                            <p className={styles.empty}>No new sign-ups yet.</p>
                        )}
                    </div>
                </div>
            </div>
        </section>
    );
}
