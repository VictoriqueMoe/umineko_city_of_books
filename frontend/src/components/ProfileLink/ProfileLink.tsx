import {useNavigate} from "react-router";
import type {User} from "../../types/api";
import {RolePill} from "../RolePill/RolePill";
import styles from "./ProfileLink.module.css";

interface ProfileLinkProps {
    user: User;
    size?: "small" | "medium" | "large";
    showName?: boolean;
    prefix?: string;
    online?: boolean;
}

const sizes = {
    small: 20,
    medium: 28,
    large: 40,
};

export function ProfileLink({ user, size = "medium", showName = true, prefix, online }: ProfileLinkProps) {
    const navigate = useNavigate();
    const px = sizes[size];

    return (
        <span className={styles.link} onClick={() => navigate(`/user/${user.username}`)}>
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
                    {user.display_name}
                    {user.role && <RolePill role={user.role} />}
                </span>
            )}
        </span>
    );
}
