import type { ReactNode } from "react";
import { Link } from "react-router";
import type { ChatRoom } from "../../../types/api";
import { RelativeTimestamp } from "../../RelativeTimestamp/RelativeTimestamp";
import styles from "./RoomCard.module.css";

const HOT_THRESHOLD = 50;

export type RoomCardVariant = "member" | "discover";

export interface RoomCardProps {
    variant: RoomCardVariant;
    room: ChatRoom;
    onTagClick: (tag: string) => void;
    actions?: ReactNode;
}

function visibilityBadge(variant: RoomCardVariant, room: ChatRoom): ReactNode {
    if (variant === "discover") {
        return <span className={styles.publicBadge}>Public</span>;
    }

    if (room.is_system) {
        return null;
    }

    if (room.is_public) {
        return <span className={styles.publicBadge}>Public</span>;
    }

    return <span className={styles.privateBadge}>Private</span>;
}

export function RoomCard({ variant, room, onTagClick, actions }: RoomCardProps) {
    const isHot = !room.archived_at && (room.hot_score ?? 0) >= HOT_THRESHOLD;

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
    if (room.archived_at) {
        classes.push(styles.archivedCard);
    }
    if (isHot) {
        classes.push(styles.hotCard);
    }

    const body = (
        <>
            <div className={styles.cardHeader}>
                <h3 dir="auto" className={styles.cardTitle}>
                    {room.name}
                </h3>
                <div className={styles.cardBadges}>
                    {room.is_system && <span className={styles.systemBadge}>System</span>}
                    {(room.voice_count ?? 0) > 0 && (
                        <span className={styles.voiceBadge} title="Voice chat active">
                            {"\u{1F50A}"} {room.voice_count}
                        </span>
                    )}
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
                    {visibilityBadge(variant, room)}
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
            {room.description && (
                <p dir="auto" className={styles.cardDesc}>
                    {room.description}
                </p>
            )}
            {room.tags && room.tags.length > 0 && (
                <div className={styles.cardTags}>
                    {room.tags.map(tag => (
                        <button
                            key={tag}
                            type="button"
                            className={styles.cardTag}
                            onClick={e => {
                                e.preventDefault();
                                e.stopPropagation();
                                onTagClick(tag);
                            }}
                        >
                            #{tag}
                        </button>
                    ))}
                </div>
            )}
            <div className={styles.cardMeta}>
                <span>
                    {"★"} {room.member_count ?? room.members.length} members
                </span>
                <RelativeTimestamp value={room.last_message_at} variant="active" className={styles.cardActivity} />
            </div>
            {actions && <div className={styles.cardActions}>{actions}</div>}
        </>
    );

    if (variant === "member") {
        return (
            <Link to={`/rooms/${room.id}`} className={classes.join(" ")}>
                {body}
            </Link>
        );
    }

    return <div className={classes.join(" ")}>{body}</div>;
}
