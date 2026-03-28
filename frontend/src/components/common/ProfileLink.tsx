import { useNavigate } from "react-router";
import type { User } from "../../types/api";

interface ProfileLinkProps {
    user: User;
    size?: "small" | "medium" | "large";
    showName?: boolean;
    prefix?: string;
}

const sizes = {
    small: 20,
    medium: 28,
    large: 40,
};

export function ProfileLink({ user, size = "medium", showName = true, prefix }: ProfileLinkProps) {
    const navigate = useNavigate();
    const px = sizes[size];

    return (
        <span className="profile-link" onClick={() => navigate(`/user/${user.username}`)}>
            {user.avatar_url ? (
                <img className="profile-link-avatar" src={user.avatar_url} alt="" style={{ width: px, height: px }} />
            ) : (
                <span className="profile-link-avatar-placeholder" style={{ width: px, height: px, fontSize: px * 0.4 }}>
                    {user.display_name[0]}
                </span>
            )}
            {showName && (
                <span className="profile-link-name">
                    {prefix && `${prefix} `}
                    {user.display_name}
                </span>
            )}
        </span>
    );
}
