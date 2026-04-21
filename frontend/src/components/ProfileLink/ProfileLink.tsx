import { Link } from "react-router";
import type { User } from "../../types/api";
import { RolePill } from "../RolePill/RolePill";
import { RoleStyledName } from "../RoleStyledName/RoleStyledName";
import styles from "./ProfileLink.module.css";

interface ProfileLinkProps {
    user: User;
    size?: "small" | "medium" | "large";
    showName?: boolean;
    prefix?: string;
    online?: boolean;
    clickable?: boolean;
}

const sizes = {
    small: 20,
    medium: 28,
    large: 40,
};

export function ProfileLink({
    user,
    size = "medium",
    showName = true,
    prefix,
    online,
    clickable = true,
}: ProfileLinkProps) {
    const px = sizes[size];
    const banned = user.banned === true;

    const content = (
        <>
            <span className={styles.avatarWrapper} style={{ width: px, height: px }}>
                {user.avatar_url ? (
                    <img
                        className={styles.avatar}
                        src={user.avatar_url}
                        alt=""
                        width={px}
                        height={px}
                        decoding="async"
                        loading="lazy"
                    />
                ) : (
                    <span className={styles.avatarPlaceholder} style={{ width: px, height: px, fontSize: px * 0.4 }}>
                        {user.display_name[0]}
                    </span>
                )}
                {online && <span className={styles.onlineDot} />}
            </span>
            {showName && (
                <span className={styles.name}>
                    {prefix && `${prefix} `}
                    <span className={banned ? styles.bannedName : undefined}>
                        <RoleStyledName name={user.display_name} role={user.role} />
                    </span>
                    <RolePill role={user.role ?? ""} userId={user.id} />
                    {banned && <span className={styles.bannedPill}>banned</span>}
                </span>
            )}
        </>
    );

    const rootClass = banned ? `${styles.link} ${styles.banned}` : styles.link;

    if (!clickable) {
        return <span className={rootClass}>{content}</span>;
    }

    return (
        <Link to={`/user/${user.username}`} className={rootClass}>
            {content}
        </Link>
    );
}
