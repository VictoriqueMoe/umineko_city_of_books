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

    const content = (
        <>
            <span className={styles.avatarWrapper} style={{ width: px, height: px }}>
                {user.avatar_url ? (
                    <img className={styles.avatar} src={user.avatar_url} alt="" style={{ width: px, height: px }} />
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
                    <RoleStyledName name={user.display_name} role={user.role} />
                    <RolePill role={user.role ?? ""} userId={user.id} />
                </span>
            )}
        </>
    );

    if (!clickable) {
        return <span className={styles.link}>{content}</span>;
    }

    return (
        <Link to={`/user/${user.username}`} className={styles.link}>
            {content}
        </Link>
    );
}
